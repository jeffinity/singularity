package kratosx

import "testing"

func TestServiceNameHelpers(t *testing.T) {
	if got := GrpcServiceName("order"); got != "order.grpc" {
		t.Fatalf("unexpected grpc service name: %s", got)
	}
	if got := HTTPServiceName("order"); got != "order.http" {
		t.Fatalf("unexpected http service name: %s", got)
	}
}
