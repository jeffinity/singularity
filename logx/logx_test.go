package logx

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	klog "github.com/go-kratos/kratos/v2/log"
	"github.com/rs/zerolog"
)

func TestTruncateMsg(t *testing.T) {
	t.Run("no truncate", func(t *testing.T) {
		got := truncateMsg("hello", 10)
		if got != "hello" {
			t.Fatalf("unexpected value: %s", got)
		}
	})

	t.Run("truncate with suffix", func(t *testing.T) {
		got := truncateMsg("abcdef", 3)
		if got != "...(len:6)" {
			t.Fatalf("unexpected value: %s", got)
		}
	})

	t.Run("non-positive limit", func(t *testing.T) {
		got := truncateMsg("abcdef", 0)
		if got != "...(len:6)" {
			t.Fatalf("unexpected value: %s", got)
		}
	})
}

func TestNormalizeOptions(t *testing.T) {
	opts := Options{}
	normalizeOptions(&opts)

	if opts.Level != klog.LevelInfo {
		t.Fatalf("unexpected level: %v", opts.Level)
	}
	if opts.MaxSizeBytes != 100*1024*1024 {
		t.Fatalf("unexpected max size: %d", opts.MaxSizeBytes)
	}
	if opts.MaxBackups != 7 {
		t.Fatalf("unexpected max backups: %d", opts.MaxBackups)
	}
	if opts.Location == nil {
		t.Fatal("expected non-nil location")
	}
	if !opts.ConsolePretty {
		t.Fatal("expected ConsolePretty to be true")
	}
	if !opts.ForceDailyRollover {
		t.Fatal("expected ForceDailyRollover to be true")
	}
	if !opts.Compress {
		t.Fatal("expected Compress to be true")
	}
	if opts.TimeFieldFormat != "2006-01-02 15:04:05" {
		t.Fatalf("unexpected time format: %s", opts.TimeFieldFormat)
	}
}

func TestBuildConsoleWriterRaw(t *testing.T) {
	w := buildConsoleWriter(Options{ConsolePretty: false, ConsoleToStderr: false})
	if w != os.Stdout {
		t.Fatal("expected stdout writer")
	}
	w = buildConsoleWriter(Options{ConsolePretty: false, ConsoleToStderr: true})
	if w != os.Stderr {
		t.Fatal("expected stderr writer")
	}
}

func TestKratosZeroLogger(t *testing.T) {
	t.Run("level filter", func(t *testing.T) {
		var buf bytes.Buffer
		zl := zerolog.New(&buf)
		l := &kratosZeroLogger{zl: &zl, level: klog.LevelInfo}
		if err := l.Log(klog.LevelDebug, "msg", "hidden"); err != nil {
			t.Fatalf("Log failed: %v", err)
		}
		if buf.Len() != 0 {
			t.Fatalf("expected no log output, got: %s", buf.String())
		}
	})

	t.Run("msg and fields", func(t *testing.T) {
		var buf bytes.Buffer
		zl := zerolog.New(&buf)
		l := &kratosZeroLogger{zl: &zl, level: klog.LevelDebug}
		if err := l.Log(klog.LevelInfo, "msg", "hello", "k", 1); err != nil {
			t.Fatalf("Log failed: %v", err)
		}
		out := buf.String()
		if !strings.Contains(out, `"msg":"hello"`) {
			t.Fatalf("missing msg field: %s", out)
		}
		if !strings.Contains(out, `"k":1`) {
			t.Fatalf("missing custom field: %s", out)
		}
	})

	t.Run("message alias", func(t *testing.T) {
		var buf bytes.Buffer
		zl := zerolog.New(&buf)
		l := &kratosZeroLogger{zl: &zl, level: klog.LevelDebug}
		if err := l.Log(klog.LevelInfo, "message", "hello"); err != nil {
			t.Fatalf("Log failed: %v", err)
		}
		if !strings.Contains(buf.String(), `"msg":"hello"`) {
			t.Fatalf("message alias not mapped: %s", buf.String())
		}
	})

	t.Run("odd keyvals", func(t *testing.T) {
		var buf bytes.Buffer
		zl := zerolog.New(&buf)
		l := &kratosZeroLogger{zl: &zl, level: klog.LevelDebug}
		if err := l.Log(klog.LevelInfo, "k"); err != nil {
			t.Fatalf("Log failed: %v", err)
		}
		if !strings.Contains(buf.String(), `"k":"(MISSING)"`) {
			t.Fatalf("odd keyvals not handled: %s", buf.String())
		}
	})

	t.Run("truncate long message", func(t *testing.T) {
		var buf bytes.Buffer
		zl := zerolog.New(&buf)
		l := &kratosZeroLogger{zl: &zl, level: klog.LevelDebug}
		long := strings.Repeat("a", maxMsgRunes+10)
		if err := l.Log(klog.LevelInfo, "msg", long); err != nil {
			t.Fatalf("Log failed: %v", err)
		}
		out := buf.String()
		if !strings.Contains(out, `...(len:4106)`) {
			t.Fatalf("expected truncated suffix, got: %s", out)
		}
	})
}

func TestNew(t *testing.T) {
	t.Run("console only", func(t *testing.T) {
		logger, closeFn, err := New(Options{BaseFilename: ""})
		if err != nil {
			t.Fatalf("New failed: %v", err)
		}
		if logger == nil || closeFn == nil {
			t.Fatal("expected logger and closeFn")
		}
		closeFn()
	})

	t.Run("file and console", func(t *testing.T) {
		dir := t.TempDir()
		base := filepath.Join(dir, "region.log")

		logger, closeFn, err := New(Options{
			Level:        klog.LevelDebug,
			BaseFilename: base,
		})
		if err != nil {
			t.Fatalf("New failed: %v", err)
		}

		if err := logger.Log(klog.LevelInfo, "msg", "hello"); err != nil {
			t.Fatalf("log write failed: %v", err)
		}
		closeFn()

		files, err := filepath.Glob(base + ".*")
		if err != nil {
			t.Fatalf("glob failed: %v", err)
		}
		if len(files) == 0 {
			t.Fatalf("expected rotated base files under %s", dir)
		}
	})
}
