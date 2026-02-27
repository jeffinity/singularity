package logx

import (
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jeffinity/singularity/friendly"
)

type rotatorConfig struct {
	base      string
	loc       *time.Location
	maxSize   int64
	maxBackup int
	compress  bool
	forceDay  bool
}

type DailySizeRotator struct {
	cfg rotatorConfig

	// 状态
	mu         sync.Mutex
	curFile    *os.File
	curDate    string // YYYYMMDD
	curClock   string
	curSize    int64
	closed     bool
	midCancel  context.CancelFunc // 午夜滚动 goroutine 取消
	swapTarget *AtomicWriter

	// 预编译
	reSuffix *regexp.Regexp
}

func NewDailySizeRotator(aw *AtomicWriter, cfg rotatorConfig) (*DailySizeRotator, error) {
	r := &DailySizeRotator{
		cfg:        cfg,
		swapTarget: aw,
		reSuffix:   regexp.MustCompile(`\.(\d{8})(\.(\d+))?(\.gz)?$`),
	}
	if err := r.openForTodayOrResume(); err != nil {
		return nil, err
	}
	if r.cfg.forceDay {
		r.startMidnightRollover()
	}
	r.startSizeMonitor()
	r.startCompression()

	if r.cfg.maxBackup > 0 {
		go func() {
			dir := filepath.Dir(r.cfg.base)
			cleanupBackups(dir, r.cfg.base, r.cfg.maxBackup)
		}()
	}
	return r, nil
}

func (r *DailySizeRotator) startCompression() {
	go func() {
		ticker := time.NewTicker(time.Minute * 10)
		defer ticker.Stop()
		for range ticker.C {
			dir := filepath.Dir(r.cfg.base)
			prefix := filepath.Base(r.cfg.base) + "."
			files, _ := filepath.Glob(filepath.Join(dir, prefix+"*"))

			r.mu.Lock()
			cur := ""
			if r.curFile != nil {
				cur = r.curFile.Name()
			}
			r.mu.Unlock()

			for _, f := range files {
				if strings.HasSuffix(f, ".gz") {
					continue
				}
				if f == cur {
					continue
				}
				go gzipAndRemove(f)
			}
			if r.cfg.maxBackup > 0 {
				cleanupBackups(dir, r.cfg.base, r.cfg.maxBackup)
			}
		}
	}()
}

func (r *DailySizeRotator) startMidnightRollover() {
	ctx, cancel := context.WithCancel(context.Background())
	r.midCancel = cancel
	go func() {
		for {
			next := r.nextMidnight()
			timer := time.NewTimer(time.Until(next))
			select {
			case <-ctx.Done():
				timer.Stop()
				return
			case <-timer.C:
				today := time.Now().In(r.cfg.loc).Format("20060102")
				r.rotateTo(today, "0000")
			}
		}
	}()
}

func (r *DailySizeRotator) startSizeMonitor() {
	go func() {
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			r.mu.Lock()
			if r.closed {
				r.mu.Unlock()
				return
			}
			fi, err := r.curFile.Stat()
			if err != nil {
				r.mu.Unlock()
				continue
			}
			size := fi.Size()
			maxSize := r.cfg.maxSize
			r.mu.Unlock()

			if maxSize > 0 && size > maxSize {
				r.rotateTo(r.curDate, time.Now().In(r.cfg.loc).Format("1504"))
			}
		}
	}()
}

func (r *DailySizeRotator) nextMidnight() time.Time {
	now := time.Now().In(r.cfg.loc)
	return time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, r.cfg.loc)
}

func (r *DailySizeRotator) openForTodayOrResume() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed {
		return fmt.Errorf("rotator closed")
	}

	today := time.Now().In(r.cfg.loc).Format("20060102")

	dir := filepath.Dir(r.cfg.base)
	prefix := filepath.Base(r.cfg.base) + "." + today
	pattern := filepath.Join(dir, prefix+"*") // 包括 .gz
	matches, _ := filepath.Glob(pattern)

	hhmm := "0000"
	if len(matches) > 0 {
		sort.Strings(matches)
		last := matches[len(matches)-1]
		if parts := r.reSuffix.FindStringSubmatch(filepath.Base(last)); len(parts) >= 4 {
			hhmm = parts[3]
		}
	}
	path := r.filenameFor(today, hhmm)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		hhmm := time.Now().In(r.cfg.loc).Format("1504")
		path = r.filenameFor(today, hhmm) // 生成 <base>.YYYYMMDD.HHMM
		f, err = os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
		if err != nil {
			return err
		}
	}

	info, _ := f.Stat()
	r.curFile = f
	r.curDate = today
	r.curClock = hhmm
	r.curSize = 0
	if info != nil {
		r.curSize = info.Size()
	}

	_ = r.updateLink(path)
	return nil
}

func (r *DailySizeRotator) maxSeqFromMatches(matches []string, ymd string) int {
	maxSeq := 1
	for _, m := range matches {
		base := filepath.Base(m)
		parts := r.reSuffix.FindStringSubmatch(base)
		if len(parts) >= 4 && parts[1] == ymd {
			if parts[3] == "" {
				if maxSeq < 1 {
					maxSeq = 1
				}
			} else if n, err := strconv.Atoi(parts[3]); err == nil {
				if n > maxSeq {
					maxSeq = n
				}
			}
		}
	}
	return maxSeq
}

func (r *DailySizeRotator) filenameFor(ymd, hhmm string) string {
	if hhmm == "" {
		return fmt.Sprintf("%s.%s", r.cfg.base, ymd)
	}
	return fmt.Sprintf("%s.%s.%s", r.cfg.base, ymd, hhmm)
}

func (r *DailySizeRotator) updateLink(target string) error {
	link := r.cfg.base
	_ = os.Remove(link)
	return os.Symlink(filepath.Base(target), link)
}

func (r *DailySizeRotator) CurrentFile() *os.File {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.curFile
}

func (r *DailySizeRotator) rotateTo(newDate string, hhmm string) {
	newPath := r.filenameFor(newDate, hhmm)
	if err := os.MkdirAll(filepath.Dir(newPath), 0o755); err != nil {
		return
	}
	newF, err := os.OpenFile(newPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return // 打不开就放弃
	}

	info, _ := newF.Stat()
	newSize := int64(0)
	if info != nil {
		newSize = info.Size()
	}

	r.mu.Lock()
	if r.closed {
		r.mu.Unlock()
		friendly.CloseQuietly(newF)
		return
	}

	oldF := r.curFile

	r.curFile = newF
	r.curDate = newDate
	r.curClock = hhmm
	r.curSize = newSize
	_ = r.updateLink(newPath)
	r.mu.Unlock()

	r.swapTarget.Swap(newF)
	_ = oldF.Sync()
	_ = oldF.Close()
}

func (r *DailySizeRotator) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed {
		return nil
	}
	r.closed = true
	if r.midCancel != nil {
		r.midCancel()
	}
	if r.curFile != nil {
		_ = r.curFile.Sync()
		_ = r.curFile.Close()
	}
	return nil
}

func gzipAndRemove(src string) {
	in, err := os.Open(src)
	if err != nil {
		return
	}
	defer friendly.CloseQuietly(in)

	out, err := os.Create(src + ".gz")
	if err != nil {
		return
	}
	gw := gzip.NewWriter(out)
	gw.Name = filepath.Base(src)
	gw.ModTime = time.Now()
	_, _ = io.Copy(gw, in)
	_ = gw.Close()
	_ = out.Close()

	_ = os.Remove(src)
}

func cleanupBackups(dir, base string, keep int) {
	glob := filepath.Join(dir, filepath.Base(base)+".*.gz")
	files, _ := filepath.Glob(glob)
	if len(files) <= keep {
		return
	}

	sort.Slice(files, func(i, j int) bool {
		if files[i] != files[j] {
			return files[i] < files[j]
		}
		fi, _ := os.Stat(files[i])
		fj, _ := os.Stat(files[j])
		if fi == nil || fj == nil {
			return files[i] < files[j]
		}
		return fi.ModTime().Before(fj.ModTime())
	})
	for i := 0; i < len(files)-keep; i++ {
		_ = os.Remove(files[i])
	}
}
