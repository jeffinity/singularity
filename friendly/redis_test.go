package friendly

import (
	"context"
	"testing"

	"github.com/go-kratos/kratos/v2/log"
)

func TestNewRedisCluster(t *testing.T) {
	t.Run("empty seeds should return error", func(t *testing.T) {
		c, cleanup, err := NewRedisCluster(context.Background(), log.DefaultLogger, nil, "", false)
		if err == nil {
			t.Fatal("expected error for empty seeds")
		}
		if c != nil {
			t.Fatal("expected nil client")
		}
		if cleanup != nil {
			t.Fatal("expected nil cleanup")
		}
	})
}

func TestNewRedis(t *testing.T) {
	t.Run("invalid dsn should return error", func(t *testing.T) {
		c, cleanup, err := NewRedis(context.Background(), log.DefaultLogger, "://bad")
		if err == nil {
			t.Fatal("expected parse error for invalid dsn")
		}
		if c != nil {
			t.Fatal("expected nil client")
		}
		if cleanup != nil {
			t.Fatal("expected nil cleanup")
		}
	})
}
