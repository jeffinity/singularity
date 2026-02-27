package pgx

import (
	"context"
	"errors"
	"testing"
	"time"

	klog "github.com/go-kratos/kratos/v2/log"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type capturedEntry struct {
	level   klog.Level
	keyvals []interface{}
}

type capturedLogger struct {
	entries []capturedEntry
}

func (l *capturedLogger) Log(level klog.Level, keyvals ...interface{}) error {
	cp := make([]interface{}, len(keyvals))
	copy(cp, keyvals)
	l.entries = append(l.entries, capturedEntry{level: level, keyvals: cp})
	return nil
}

func (l *capturedLogger) hasLevel(level klog.Level) bool {
	for _, e := range l.entries {
		if e.level == level {
			return true
		}
	}
	return false
}

func TestNewGormLoggerAndLogMode(t *testing.T) {
	base := &capturedLogger{}
	helper := klog.NewHelper(base)

	gi := NewGormLogger(helper, logger.Config{LogLevel: logger.Warn})
	if gi == nil {
		t.Fatal("expected non-nil logger")
	}

	g2 := gi.LogMode(logger.Error)
	typed, ok := g2.(*gormLogger)
	if !ok {
		t.Fatalf("expected *gormLogger, got %T", g2)
	}
	if typed.LogLevel != logger.Error {
		t.Fatalf("unexpected log level: %v", typed.LogLevel)
	}
}

func TestGormLoggerInfoWarnError(t *testing.T) {
	base := &capturedLogger{}
	l := NewGormLogger(
		klog.NewHelper(base),
		logger.Config{LogLevel: logger.Info},
	).(*gormLogger)

	l.Info(context.Background(), "hello %s", "world")
	l.Warn(context.Background(), "warn %s", "x")
	l.Error(context.Background(), "err %s", "x")

	if !base.hasLevel(klog.LevelInfo) {
		t.Fatal("expected info log")
	}
	if !base.hasLevel(klog.LevelWarn) {
		t.Fatal("expected warn log")
	}
	if !base.hasLevel(klog.LevelError) {
		t.Fatal("expected error log")
	}
}

func TestGormLoggerTrace(t *testing.T) {
	fc := func() (string, int64) {
		return "select 1", 1
	}

	t.Run("silent should not log", func(t *testing.T) {
		base := &capturedLogger{}
		l := NewGormLogger(klog.NewHelper(base), logger.Config{
			LogLevel: logger.Silent,
		}).(*gormLogger)

		l.Trace(context.Background(), time.Now(), fc, nil)
		if len(base.entries) != 0 {
			t.Fatalf("expected no logs, got %d", len(base.entries))
		}
	})

	t.Run("error branch", func(t *testing.T) {
		base := &capturedLogger{}
		l := NewGormLogger(klog.NewHelper(base), logger.Config{
			LogLevel: logger.Error,
		}).(*gormLogger)

		l.Trace(context.Background(), time.Now(), fc, errors.New("boom"))
		if !base.hasLevel(klog.LevelError) {
			t.Fatal("expected error trace log")
		}
	})

	t.Run("record not found ignored", func(t *testing.T) {
		base := &capturedLogger{}
		l := NewGormLogger(klog.NewHelper(base), logger.Config{
			LogLevel:                  logger.Error,
			IgnoreRecordNotFoundError: true,
		}).(*gormLogger)

		l.Trace(context.Background(), time.Now(), fc, gorm.ErrRecordNotFound)
		if len(base.entries) != 0 {
			t.Fatalf("expected no logs, got %d", len(base.entries))
		}
	})

	t.Run("slow sql warn branch", func(t *testing.T) {
		base := &capturedLogger{}
		l := NewGormLogger(klog.NewHelper(base), logger.Config{
			LogLevel:      logger.Warn,
			SlowThreshold: time.Millisecond,
		}).(*gormLogger)

		l.Trace(context.Background(), time.Now().Add(-10*time.Millisecond), fc, nil)
		if !base.hasLevel(klog.LevelWarn) {
			t.Fatal("expected warn trace log")
		}
	})

	t.Run("info branch disabled by context flag", func(t *testing.T) {
		base := &capturedLogger{}
		l := NewGormLogger(klog.NewHelper(base), logger.Config{
			LogLevel: logger.Info,
		}).(*gormLogger)

		ctx := context.WithValue(context.Background(), FlagDisableSQLLog, true)
		l.Trace(ctx, time.Now(), fc, nil)
		if len(base.entries) != 0 {
			t.Fatalf("expected no logs, got %d", len(base.entries))
		}
	})

	t.Run("info branch", func(t *testing.T) {
		base := &capturedLogger{}
		l := NewGormLogger(klog.NewHelper(base), logger.Config{
			LogLevel: logger.Info,
		}).(*gormLogger)

		l.Trace(context.Background(), time.Now(), fc, nil)
		if !base.hasLevel(klog.LevelInfo) {
			t.Fatal("expected info trace log")
		}
	})
}
