package kratosx

import (
	stderrors "errors"
	"fmt"
	"strings"
	"testing"

	pkgErr "github.com/pkg/errors"
)

type badStringer struct{}

func (badStringer) String() string {
	return "from-stringer"
}

func (badStringer) MarshalJSON() ([]byte, error) {
	return nil, fmt.Errorf("marshal failed")
}

type badNonStringer struct{}

func (badNonStringer) MarshalJSON() ([]byte, error) {
	return nil, fmt.Errorf("marshal failed")
}

func TestExtractArgs(t *testing.T) {
	t.Run("normal json marshal", func(t *testing.T) {
		got := extractArgs(map[string]string{"k": "v"})
		if !strings.Contains(got, `"k":"v"`) {
			t.Fatalf("unexpected json output: %s", got)
		}
	})

	t.Run("fallback to stringer when marshal fails", func(t *testing.T) {
		got := extractArgs(badStringer{})
		if got != "from-stringer" {
			t.Fatalf("unexpected fallback output: %s", got)
		}
	})

	t.Run("fallback to fmt when marshal fails and not stringer", func(t *testing.T) {
		got := extractArgs(badNonStringer{})
		if got == "" {
			t.Fatal("expected non-empty fallback output")
		}
	})
}

func TestExtractError(t *testing.T) {
	level, stack := extractError(nil)
	if level.String() != "INFO" || stack != "" {
		t.Fatalf("unexpected nil error result: level=%s stack=%q", level.String(), stack)
	}

	level, stack = extractError(stderrors.New("x"))
	if level.String() != "ERROR" || stack == "" {
		t.Fatalf("unexpected non-nil error result: level=%s stack=%q", level.String(), stack)
	}
}

func TestTruncateBytes(t *testing.T) {
	if got := TruncateBytes("abc", 10); got != "abc" {
		t.Fatalf("unexpected no-truncate result: %s", got)
	}

	got := TruncateBytes("abcdef", 3)
	if !strings.HasPrefix(got, "abc...(len:6)") {
		t.Fatalf("unexpected truncate result: %s", got)
	}
}

func TestGetErrCauseStack(t *testing.T) {
	t.Run("find stack in wrapped error", func(t *testing.T) {
		root := pkgErr.WithStack(stderrors.New("root"))
		wrapped := fmt.Errorf("wrap: %w", root)
		stack := getErrCauseStack(wrapped)
		if stack == nil {
			t.Fatal("expected stack trace")
		}
	})

	t.Run("nil error returns nil stack", func(t *testing.T) {
		if got := getErrCauseStack(nil); got != nil {
			t.Fatal("expected nil stack")
		}
	})
}
