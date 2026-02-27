package logx

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestAtomicWriter(t *testing.T) {
	t.Run("discard before swap", func(t *testing.T) {
		var aw AtomicWriter
		n, err := aw.Write([]byte("abc"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if n != 3 {
			t.Fatalf("unexpected write size: %d", n)
		}
	})

	t.Run("write to swapped writer", func(t *testing.T) {
		var aw AtomicWriter
		var buf bytes.Buffer
		aw.Swap(&buf)
		_, err := aw.Write([]byte("abc"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if buf.String() != "abc" {
			t.Fatalf("unexpected buffer content: %s", buf.String())
		}
	})

	t.Run("non-writer in atomic value", func(t *testing.T) {
		var aw AtomicWriter
		aw.val.Store("bad")
		_, err := aw.Write([]byte("abc"))
		if err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestCleanupBackups(t *testing.T) {
	dir := t.TempDir()
	base := filepath.Join(dir, "region.log")

	files := []string{
		base + ".20260101.gz",
		base + ".20260102.gz",
		base + ".20260103.gz",
		base + ".20260104.gz",
	}
	for _, f := range files {
		if err := os.WriteFile(f, []byte("x"), 0o644); err != nil {
			t.Fatalf("write file failed: %v", err)
		}
	}

	cleanupBackups(dir, base, 2)

	remain, err := filepath.Glob(base + ".*.gz")
	if err != nil {
		t.Fatalf("glob failed: %v", err)
	}
	if len(remain) != 2 {
		t.Fatalf("expected 2 backups, got %d", len(remain))
	}

	if filepath.Base(remain[0]) != "region.log.20260103.gz" || filepath.Base(remain[1]) != "region.log.20260104.gz" {
		t.Fatalf("unexpected retained files: %#v", remain)
	}
}
