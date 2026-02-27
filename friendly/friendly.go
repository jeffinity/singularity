package friendly

import (
	"io"
	"time"

	"github.com/go-kratos/kratos/v2/log"
)

func FormatMillis(ms int64) string {
	t := time.UnixMilli(ms)
	return t.Format("2006-01-02 15:04:05")
}

func CloseQuietly(c io.Closer) {
	if err := c.Close(); err != nil {
		log.Warnf("failed to close resource: %v", err)
	}
}

func GetOrDefault[T comparable](v, def T) T {
	var zero T
	if v == zero {
		return def
	}
	return v
}
