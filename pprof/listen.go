package pprof

import (
	"net"
	"net/http"
	"net/http/pprof"

	"github.com/go-kratos/kratos/v2/log"
)

func ListenInBackground(logger *log.Helper) {
	go func() {
		// 端口用 0 ， 会自动获取一个随机端口做监听
		l, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			logger.Error(err)
			return
		}
		r := http.NewServeMux()
		r.HandleFunc("/debug/pprof/", pprof.Index)
		r.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
		r.HandleFunc("/debug/pprof/profile", pprof.Profile)
		r.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
		r.HandleFunc("/debug/pprof/trace", pprof.Trace)
		v, _ := l.Addr().(*net.TCPAddr)
		logger.Infof("listening pprof at [%d], pprof index: /debug/pprof/", v.Port)
		err = http.Serve(l, r)
		if err != nil {
			logger.Errorf("listening pprof failed,err: %s", err.Error())
			return
		}
	}()
}
