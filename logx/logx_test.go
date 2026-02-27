package logx

import (
	"testing"

	"github.com/go-kratos/kratos/v2/log"
)

func TestLogConsole(t *testing.T) {

	logger, closeFn, err := New(Options{
		BaseFilename: "", // 为空 => 仅控制台 pretty
		Level:        log.LevelDebug,
	})
	if err != nil {
		panic(err)
	}

	mlog := log.NewHelper(log.With(logger, "module", "klog"))
	mlog.Info("console only")

	mlog.Error(
		"[stack]\ngitlab.gainetics.io/backend-omp/dnms/probe-executor/pkg/probe_center/data_layer.(*nodeControlRepo).ListNodes\n\t/Users/jeff/tao/workspace/gainetics/probe-executor/pkg/probe_center/data_layer/node_control.go:75\ngitlab.gainetics.io/backend-omp/dnms/probe-executor/app/probe_center/internal/biz.(*NodeControlUseCase).ListNodes\n\t/Users/jeff/tao/workspace/gainetics/probe-executor/app/probe_center/internal/biz/node_control.go:92\ngitlab.gainetics.io/backend-omp/dnms/probe-executor/app/probe_center/internal/service.(*NodeManagerService).NodeList\n\t/Users/jeff/tao/workspace/gainetics/probe-executor/app/probe_center/internal/service/node_manager.go:33\ngitlab.gainetics.io/backend-cdn/go-protos/probe-executor/control-plane/v1._NodeManagerService_NodeList_Handler.func1\n\t/Users/jeff/tao/workspace/gainetics/proto-hub/probe-executor/control-plane/v1/node_manager_grpc.pb.go:161\ngithub.com/go-kratos/kratos/v2/transport/grpc.(*Server).unaryServerInterceptor.func1.1\n\t/Users/jeff/.asdf/installs/golang/1.24.3/packages/pkg/mod/github.com/go-kratos/kratos/v2@v2.8.4/transport/grpc/interceptor.go:37\ngithub.com/go-kratos/kratos/contrib/middleware/validate/v2.ProtoValidate.func1.1\n\t/Users/jeff/.asdf/installs/golang/1.24.3/packages/pkg/mod/github.com/go-kratos/kratos/contrib/middleware/validate/v2@v2.0.0-20250527152916-d6f5f00cf562/validate.go:34\ngithub.com/go-kratos/kratos/v2/middleware/recovery.Recovery.func2.1\n\t/Users/jeff/.asdf/installs/golang/1.24.3/packages/pkg/mod/github.com/go-kratos/kratos/v2@v2.8.4/middleware/recovery/recovery.go:59\ngithub.com/go-kratos/kratos/v2/transport/grpc.(*Server).unaryServerInterceptor.func1\n\t/Users/jeff/.asdf/installs/golang/1.24.3/packages/pkg/mod/github.com/go-kratos/kratos/v2@v2.8.4/transport/grpc/interceptor.go:42\ngitlab.gainetics.io/backend-cdn/go-protos/probe-executor/control-plane/v1._NodeManagerService_NodeList_Handler\n\t/Users/jeff/tao/workspace/gainetics/proto-hub/probe-executor/control-plane/v1/node_manager_grpc.pb.go:163\ngoogle.golang.org/grpc.(*Server).processUnaryRPC\n\t/Users/jeff/.asdf/installs/golang/1.24.3/packages/pkg/mod/google.golang.org/grpc@v1.73.0/server.go:1405\ngoogle.golang.org/grpc.(*Server).handleStream\n\t/Users/jeff/.asdf/installs/golang/1.24.3/packages/pkg/mod/google.golang.org/grpc@v1.73.0/server.go:1815\ngoogle.golang.org/grpc.(*Server).serveStreams.func2.1\n\t/Users/jeff/.asdf/installs/golang/1.24.3/packages/pkg/mod/google.golang.org/grpc@v1.73.0/server.go:1035\nruntime.goexit\n\t/Users/jeff/.asdf/installs/golang/1.24.3/go/src/runtime/asm_arm64.s:122",
	)

	msg := `{"progress":[{"timing":{"dispatch_ts":"1753447085326"}, "error_message":"地方节点离线"}]}`
	mlog.Log(
		log.LevelDebug,
		"msg", msg,
		"kind", "server",
		"component", "grpc",
		"operation", "/gainetics.probe_executor.control_plane.v1.NodeManagerService/TaskProgress",
		"code", 0,
		"reason", "",
		"stack", "",
		"latency", 0.000985845,
		"type", "Request Done",
	)

	closeFn()
}

func TestLogFileAndConsole(t *testing.T) {
	logger, closeFn, err := New(Options{
		Level:              log.LevelDebug,
		BaseFilename:       "/tmp/region.log", // 非空 => 写文件(JSON) + 控制台(pretty)
		MaxSizeBytes:       0,                 // 使用默认 100MB
		MaxBackups:         0,                 // 使用默认 7
		Compress:           true,              // 默认 true，可省略
		ForceDailyRollover: true,              // 默认 true，可省略
		// Location:         nil,  // 默认本地时区
		// ConsolePretty:    true, // 默认
		// TimeFieldFormat:  "2006-01-02 15:04:05", // 默认
	})
	if err != nil {
		panic(err)
	}

	mlog := log.NewHelper(log.With(logger, "module", "klog"))
	mlog.Info("hello")

	mlog.Error(
		"[stack]\ngitlab.gainetics.io/backend-omp/dnms/probe-executor/pkg/probe_center/data_layer.(*nodeControlRepo).ListNodes\n\t/Users/jeff/tao/workspace/gainetics/probe-executor/pkg/probe_center/data_layer/node_control.go:75\ngitlab.gainetics.io/backend-omp/dnms/probe-executor/app/probe_center/internal/biz.(*NodeControlUseCase).ListNodes\n\t/Users/jeff/tao/workspace/gainetics/probe-executor/app/probe_center/internal/biz/node_control.go:92\ngitlab.gainetics.io/backend-omp/dnms/probe-executor/app/probe_center/internal/service.(*NodeManagerService).NodeList\n\t/Users/jeff/tao/workspace/gainetics/probe-executor/app/probe_center/internal/service/node_manager.go:33\ngitlab.gainetics.io/backend-cdn/go-protos/probe-executor/control-plane/v1._NodeManagerService_NodeList_Handler.func1\n\t/Users/jeff/tao/workspace/gainetics/proto-hub/probe-executor/control-plane/v1/node_manager_grpc.pb.go:161\ngithub.com/go-kratos/kratos/v2/transport/grpc.(*Server).unaryServerInterceptor.func1.1\n\t/Users/jeff/.asdf/installs/golang/1.24.3/packages/pkg/mod/github.com/go-kratos/kratos/v2@v2.8.4/transport/grpc/interceptor.go:37\ngithub.com/go-kratos/kratos/contrib/middleware/validate/v2.ProtoValidate.func1.1\n\t/Users/jeff/.asdf/installs/golang/1.24.3/packages/pkg/mod/github.com/go-kratos/kratos/contrib/middleware/validate/v2@v2.0.0-20250527152916-d6f5f00cf562/validate.go:34\ngithub.com/go-kratos/kratos/v2/middleware/recovery.Recovery.func2.1\n\t/Users/jeff/.asdf/installs/golang/1.24.3/packages/pkg/mod/github.com/go-kratos/kratos/v2@v2.8.4/middleware/recovery/recovery.go:59\ngithub.com/go-kratos/kratos/v2/transport/grpc.(*Server).unaryServerInterceptor.func1\n\t/Users/jeff/.asdf/installs/golang/1.24.3/packages/pkg/mod/github.com/go-kratos/kratos/v2@v2.8.4/transport/grpc/interceptor.go:42\ngitlab.gainetics.io/backend-cdn/go-protos/probe-executor/control-plane/v1._NodeManagerService_NodeList_Handler\n\t/Users/jeff/tao/workspace/gainetics/proto-hub/probe-executor/control-plane/v1/node_manager_grpc.pb.go:163\ngoogle.golang.org/grpc.(*Server).processUnaryRPC\n\t/Users/jeff/.asdf/installs/golang/1.24.3/packages/pkg/mod/google.golang.org/grpc@v1.73.0/server.go:1405\ngoogle.golang.org/grpc.(*Server).handleStream\n\t/Users/jeff/.asdf/installs/golang/1.24.3/packages/pkg/mod/google.golang.org/grpc@v1.73.0/server.go:1815\ngoogle.golang.org/grpc.(*Server).serveStreams.func2.1\n\t/Users/jeff/.asdf/installs/golang/1.24.3/packages/pkg/mod/google.golang.org/grpc@v1.73.0/server.go:1035\nruntime.goexit\n\t/Users/jeff/.asdf/installs/golang/1.24.3/go/src/runtime/asm_arm64.s:122",
	)

	msg := `{"progress":[{"timing":{"dispatch_ts":"1753447085326"}, "error_message":"地方节点离线"}]}`
	mlog.Log(
		log.LevelDebug,
		"msg", msg,
		"kind", "server",
		"component", "grpc",
		"operation", "/gainetics.probe_executor.control_plane.v1.NodeManagerService/TaskProgress",
		"code", 0,
		"reason", "",
		"stack", "",
		"latency", 0.000985845,
		"type", "Request Done",
	)

	closeFn()
}
