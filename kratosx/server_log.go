package kratosx

import (
	"context"
	stdErr "errors"
	"fmt"
	"time"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/transport"
	pkgErr "github.com/pkg/errors"
	"google.golang.org/grpc/peer"
)

const MaxShowBodyLen = 1024

//nolint:funlen
func ServerLogger(logger log.Logger) middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (reply interface{}, err error) {
			// INFO[22945]                                               args="task_id:\"66bf4df37d164f82a6501f0f623c9eae\"" caller="validate.go:23" code=0 component=grpc
			// kind=server latency=0.000945698 operation=/tophant.parser.api.ParserTask/TaskStatus reason= stack= ts="2022-08-18 22:36:18"

			var (
				code      int32
				reason    string
				kind      string
				operation string
				clientIP  string
			)

			startTime := time.Now()
			if info, ok := transport.FromServerContext(ctx); ok {
				kind = info.Kind().String()
				operation = info.Operation()
			}

			// 获取来源 IP
			if p, ok := peer.FromContext(ctx); ok && p.Addr != nil {
				clientIP = p.Addr.String()
			}

			_ = log.WithContext(ctx, logger).Log(log.LevelInfo,
				"kind", "server",
				"component", kind,
				"operation", operation,
				"type", "New Request",
				"client_ip", clientIP,
				"msg", TruncateBytes(extractArgs(req), MaxShowBodyLen),
			)

			reply, err = handler(ctx, req)
			if se := errors.FromError(err); se != nil {
				code = se.Code
				reason = se.Reason
			}

			level, stack := extractError(err)
			if code >= 500 {
				_ = log.WithContext(ctx, logger).Log(level,
					"kind", "server",
					"component", kind,
					"operation", operation,
					"code", code,
					"reason", reason,
					"type", "Error Stack",
					"msg", fmt.Sprintf("%+v", getErrCauseStack(err)),
				)
			}

			_ = log.WithContext(ctx, logger).Log(level,
				"kind", "server",
				"component", kind,
				"operation", operation,
				"code", code,
				"reason", reason,
				"stack", stack,
				"latency", time.Since(startTime).Seconds(),
				"type", "Request Done",
				"msg", TruncateBytes(extractArgs(reply), MaxShowBodyLen),
			)
			return
		}
	}
}

// extractArgs returns the string of the req
func extractArgs(req interface{}) string {
	var argStr string
	var bs, err = Codec.Marshal(req)
	if err == nil {
		argStr = string(bs)
	} else {
		if stringer, ok := req.(fmt.Stringer); ok {
			argStr = stringer.String()
		} else {
			argStr = fmt.Sprintf("%+v", req)
		}
	}
	return argStr
}

// extractError returns the string of the error
func extractError(err error) (log.Level, string) {
	if err != nil {
		return log.LevelError, fmt.Sprintf("%+v", err)
	}
	return log.LevelInfo, ""
}

func TruncateBytes(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + fmt.Sprintf("...(len:%d)", len(s))
}

type LoggerFunc func(string, ...interface{})

type stackTracer interface {
	StackTrace() pkgErr.StackTrace
}

func getErrCauseStack(err error) pkgErr.StackTrace {

	if se := errors.FromError(err); se != nil && se.Unwrap() != nil {
		err = se.Unwrap()
	}

	for e := err; e != nil; e = stdErr.Unwrap(e) {
		if st, ok := e.(stackTracer); ok {
			return st.StackTrace()
		}
	}
	return nil
}
