package kratosx

import "fmt"

type ServiceName = string
type ServiceID string

const (
	ServiceNameAppLayout ServiceName = "app_layout" // 项目模板

	ServiceNameProbeApi ServiceName = "probe-api" // 可用性探测服务

	ServiceNameProbeCenter ServiceName = "probe_center" // 控制面服务

	ServiceNameProbeRegin ServiceName = "probe_region" // 控制面服务
)

func GrpcServiceName(name ServiceName) string {
	return fmt.Sprintf("%s.%s", name, "grpc")
}

func HTTPServiceName(name ServiceName) string {
	return fmt.Sprintf("%s.%s", name, "http")
}
