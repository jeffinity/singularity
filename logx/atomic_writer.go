package logx

import (
	"io"
	"sync/atomic"

	perr "github.com/pkg/errors"
)

type AtomicWriter struct {
	val atomic.Value
}

func (aw *AtomicWriter) Write(p []byte) (int, error) {
	v := aw.val.Load()
	if v == nil {
		return io.Discard.Write(p) // 首次 Swap 前的兜底
	}
	w, ok := v.(io.Writer)
	if !ok {
		return 0, perr.Errorf("AtomicWriter holds non-writer: %T", v)
	}
	return w.Write(p)
}

func (aw *AtomicWriter) Swap(w io.Writer) {
	aw.val.Store(w)
}
