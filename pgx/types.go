package pgx

import (
	"database/sql/driver"
	"fmt"

	"github.com/jeffinity/singularity/kratosx"
)

// JSONB wraps any struct & handles pg jsonb ↔ Go struct transparently.
type JSONB[T any] struct {
	Val *T // 可为 nil
}

// noinspection GoMixedReceiverTypes
func (j JSONB[T]) Value() (driver.Value, error) {
	if j.Val == nil {
		return nil, nil // DB 设置为 NULL
	}
	b, err := kratosx.Codec.Marshal(j.Val)
	if err != nil {
		return nil, err
	}
	return b, nil // PG driver 会自动加 '::jsonb'
}

// noinspection GoMixedReceiverTypes
func (j *JSONB[T]) Scan(src any) error {
	if src == nil {
		j.Val = nil
		return nil
	}
	var data []byte
	switch v := src.(type) {
	case []byte:
		data = v
	case string:
		data = []byte(v)
	default:
		return fmt.Errorf("unsupported type: %T", src)
	}
	var t T
	if err := kratosx.Codec.Unmarshal(data, &t); err != nil {
		return err
	}
	j.Val = &t
	return nil
}
