package friendly

import (
	"errors"
	"testing"
	"time"
)

type closerStub struct {
	err    error
	called bool
}

func (c *closerStub) Close() error {
	c.called = true
	return c.err
}

func TestFormatMillis(t *testing.T) {
	layout := "2006-01-02 15:04:05"

	t.Run("zero timestamp", func(t *testing.T) {
		got := FormatMillis(0)
		want := time.UnixMilli(0).Format(layout)
		if got != want {
			t.Fatalf("unexpected formatted time: %s", got)
		}
	})

	t.Run("positive timestamp", func(t *testing.T) {
		got := FormatMillis(1700000000000)
		want := time.UnixMilli(1700000000000).Format(layout)
		if got != want {
			t.Fatalf("unexpected formatted time: %s", got)
		}
	})
}

func TestCloseQuietly(t *testing.T) {
	t.Run("close success", func(t *testing.T) {
		c := &closerStub{}
		CloseQuietly(c)
		if !c.called {
			t.Fatal("expected Close to be called")
		}
	})

	t.Run("close failed should not panic", func(t *testing.T) {
		c := &closerStub{err: errors.New("close failed")}
		CloseQuietly(c)
		if !c.called {
			t.Fatal("expected Close to be called")
		}
	})
}

func TestGetOrDefault(t *testing.T) {
	t.Run("string zero value", func(t *testing.T) {
		got := GetOrDefault("", "fallback")
		if got != "fallback" {
			t.Fatalf("expected fallback, got: %s", got)
		}
	})

	t.Run("string non-zero value", func(t *testing.T) {
		got := GetOrDefault("value", "fallback")
		if got != "value" {
			t.Fatalf("expected value, got: %s", got)
		}
	})

	t.Run("int zero value", func(t *testing.T) {
		got := GetOrDefault(0, 42)
		if got != 42 {
			t.Fatalf("expected 42, got: %d", got)
		}
	})

	t.Run("bool false treated as zero", func(t *testing.T) {
		got := GetOrDefault(false, true)
		if !got {
			t.Fatal("expected true")
		}
	})
}
