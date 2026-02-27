package logx

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	klog "github.com/go-kratos/kratos/v2/log"
	"github.com/rs/zerolog"
)

const maxMsgRunes = 4096

func truncateMsg(s string, n int) string {
	if n <= 0 {
		return fmt.Sprintf("...(len:%d)", len([]rune(s)))
	}
	rs := []rune(s)
	if len(rs) <= n {
		return s
	}

	suffix := fmt.Sprintf("...(len:%d)", len(rs))
	keep := n - len([]rune(suffix))
	if keep < 0 {
		keep = 0
	}
	return string(rs[:keep]) + suffix
}

func init() {
	zerolog.MessageFieldName = "msg"
	zerolog.CallerMarshalFunc = func(_ uintptr, file string, line int) string {
		parent := filepath.Base(filepath.Dir(file))
		name := filepath.Base(file)
		return fmt.Sprintf("%s/%s:%d", parent, name, line)
	}
}

// Options 配置
type Options struct {

	// 仅打印指定 Level 及以上级别的日志
	Level klog.Level

	// 若为空：仅输出到控制台（pretty），不写文件
	// 若非空：作为基础名，例如 /var/log/region.log
	BaseFilename string

	// 单文件最大大小（字节），默认 100MB
	MaxSizeBytes int64

	// 仅针对 .gz 的保留数量，默认 7
	MaxBackups int

	// 滚动后是否 gzip 压缩，默认 true
	Compress bool

	// 每天至少滚动一次（在本地午夜），默认 true
	ForceDailyRollover bool

	// 时区；默认 time.Local
	Location *time.Location

	// 控制台是否 pretty，默认 true
	ConsolePretty bool

	// 控制台输出到 stderr（默认 false -> stdout）
	ConsoleToStderr bool

	// 时间字段格式（同时用于文件 JSON 与控制台 pretty）
	// 默认 "2006-01-02 15:04:05"
	TimeFieldFormat string
}

// normalizeOptions 为 opts 填充默认值
func normalizeOptions(opts *Options) {
	if opts.Level == 0 {
		opts.Level = klog.LevelInfo
	}
	if opts.MaxSizeBytes <= 0 {
		opts.MaxSizeBytes = 100 * 1024 * 1024 // 100MB
	}
	if opts.MaxBackups == 0 {
		opts.MaxBackups = 7
	}
	if opts.Location == nil {
		opts.Location = time.Local
	}
	if !opts.ConsolePretty {
		opts.ConsolePretty = true
	}
	if !opts.ForceDailyRollover {
		opts.ForceDailyRollover = true
	}
	if !opts.Compress {
		opts.Compress = true
	}
	if opts.TimeFieldFormat == "" {
		opts.TimeFieldFormat = "2006-01-02 15:04:05"
	}
}

// New 构建 Kratos Logger（文件 JSON + 控制台 pretty 或仅控制台）并返回关闭函数
func New(opts Options) (klog.Logger, func(), error) {
	normalizeOptions(&opts)

	writers := make([]io.Writer, 0, 2) // 文件 + 控制台
	var rot *DailySizeRotator
	var aw *AtomicWriter

	if opts.BaseFilename != "" {
		var err error
		rot, aw, err = buildRotator(opts)
		if err != nil {
			return nil, nil, err
		}
		writers = append(writers, aw)
	}

	writers = append(writers, buildConsoleWriter(opts))

	multi := zerolog.MultiLevelWriter(writers...)
	zerolog.TimeFieldFormat = opts.TimeFieldFormat
	zl := zerolog.
		New(multi).
		With().
		Timestamp().
		CallerWithSkipFrameCount(5).
		Logger()

	kzl := &kratosZeroLogger{zl: &zl, level: opts.Level}
	closeFn := func() {
		if rot != nil {
			_ = rot.Close()
		}
	}
	return kzl, closeFn, nil
}

func buildRotator(opts Options) (*DailySizeRotator, *AtomicWriter, error) {
	if err := os.MkdirAll(filepath.Dir(opts.BaseFilename), 0o755); err != nil {
		return nil, nil, fmt.Errorf("mkdir: %w", err)
	}

	aw := &AtomicWriter{}
	rot, err := NewDailySizeRotator(aw, rotatorConfig{
		base:      opts.BaseFilename,
		loc:       opts.Location,
		maxSize:   opts.MaxSizeBytes,
		maxBackup: opts.MaxBackups,
		compress:  opts.Compress,
		forceDay:  opts.ForceDailyRollover,
	})
	if err != nil {
		return nil, nil, err
	}

	aw.Swap(rot.CurrentFile())
	return rot, aw, nil
}

func buildConsoleWriter(opts Options) io.Writer {
	if !opts.ConsolePretty {
		if opts.ConsoleToStderr {
			return os.Stderr
		}
		return os.Stdout
	}

	cw := zerolog.ConsoleWriter{
		Out:        chooseWriter(opts.ConsoleToStderr),
		TimeFormat: opts.TimeFieldFormat,
	}
	cw.FormatMessage = func(i interface{}) string {
		if i == nil {
			return ""
		}
		return fmt.Sprint(i)
	}
	cw.PartsOrder = []string{"time", "level", "caller", "msg"}
	return cw
}

func chooseWriter(toErr bool) io.Writer {
	if toErr {
		return os.Stderr
	}
	return os.Stdout
}

type kratosZeroLogger struct {
	zl *zerolog.Logger

	level klog.Level
}

func (l *kratosZeroLogger) Log(level klog.Level, keyvals ...interface{}) error {
	if level < l.level {
		return nil
	}

	if len(keyvals)%2 != 0 {
		keyvals = append(keyvals, "(MISSING)")
	}
	var evt *zerolog.Event
	switch level {
	case klog.LevelDebug:
		evt = l.zl.Debug()
	case klog.LevelInfo:
		evt = l.zl.Info()
	case klog.LevelWarn:
		evt = l.zl.Warn()
	case klog.LevelError:
		evt = l.zl.Error()
	case klog.LevelFatal:
		evt = l.zl.Fatal()
	default:
		evt = l.zl.Info()
	}

	var msgVal interface{}
	for i := 0; i < len(keyvals); i += 2 {
		k := fmt.Sprint(keyvals[i])
		v := keyvals[i+1]
		if strings.EqualFold(k, "msg") || strings.EqualFold(k, "message") {
			msgVal = v
			continue
		}
		evt = evt.Interface(k, v)
	}

	if msgVal != nil {
		msg := fmt.Sprint(msgVal)
		msg = truncateMsg(msg, maxMsgRunes)
		evt.Msg(msg)
	} else {
		evt.Send()
	}
	return nil
}
