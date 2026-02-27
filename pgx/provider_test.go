package pgx

import (
	"strings"
	"testing"

	"gorm.io/gorm/logger"
)

func TestParsePostgresLogLevel(t *testing.T) {
	cases := []struct {
		in   string
		want logger.LogLevel
	}{
		{in: "INFO", want: logger.Warn},
		{in: "info", want: logger.Warn},
		{in: "WARNING", want: logger.Warn},
		{in: "WARN", want: logger.Warn},
		{in: "ERROR", want: logger.Error},
		{in: "unknown", want: logger.Info},
	}

	for _, c := range cases {
		if got := ParsePostgresLogLevel(c.in); got != c.want {
			t.Fatalf("ParsePostgresLogLevel(%q)=%v want=%v", c.in, got, c.want)
		}
	}
}

func TestNewPostgresInvalidDSN(t *testing.T) {
	db, err := NewPostgres("INFO", "://bad", nil)
	if err == nil {
		t.Fatal("expected error for invalid dsn")
	}
	if db != nil {
		t.Fatal("expected nil db")
	}
	if !strings.Contains(err.Error(), "invalid dsn") {
		t.Fatalf("unexpected error: %v", err)
	}
}

