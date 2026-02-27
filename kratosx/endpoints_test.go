package kratosx

import (
	"os"
	"testing"
)

type fakeServerConf struct {
	addr string
	pub  string
}

func (f fakeServerConf) GetAddr() string           { return f.addr }
func (f fakeServerConf) GetPublicEndpoint() string { return f.pub }

func TestParseStrEndpoints_PriorityAndPorts(t *testing.T) {
	t.Run("Env overrides everything (http)", func(t *testing.T) {
		t.Setenv("ADVERTISE_HOST", "1.2.3.4")
		http := fakeServerConf{addr: "0.0.0.0:8080"}
		eps, err := ParseStrEndpoints(http, nil)
		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		if len(eps) != 1 {
			t.Fatalf("want 1 endpoint, got %d", len(eps))
		}
		if eps[0] != "http://1.2.3.4:8080" {
			t.Fatalf("unexpected endpoint: %q", eps[0])
		}
	})

	t.Run("Public endpoint with scheme (grpc) + port from addr", func(t *testing.T) {
		// 清空 ENV，优先级落到 PublicEndpoint
		_ = os.Unsetenv("ADVERTISE_HOST")
		grpc := fakeServerConf{
			addr: ":7411",                  // 端口以 addr 为准
			pub:  "grpc://172.28.1.1:9999", // host 来自 pub，scheme=grpc 保留
		}
		eps, err := ParseStrEndpoints(nil, grpc)
		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		if len(eps) != 1 {
			t.Fatalf("want 1 endpoint, got %d", len(eps))
		}
		if eps[0] != "grpc://172.28.1.1:7411" {
			t.Fatalf("unexpected endpoint: %q", eps[0])
		}
	})

	t.Run("Public endpoint without scheme -> default scheme(http)", func(t *testing.T) {
		_ = os.Unsetenv("ADVERTISE_HOST")
		http := fakeServerConf{
			addr: ":8081",
			pub:  "198.51.100.9:5555", // 无 scheme，取默认 http，但端口仍来自 addr
		}
		eps, err := ParseStrEndpoints(http, nil)
		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		if eps[0] != "http://198.51.100.9:8081" {
			t.Fatalf("unexpected endpoint: %q", eps[0])
		}
	})

	t.Run("Use host in addr when usable", func(t *testing.T) {
		_ = os.Unsetenv("ADVERTISE_HOST")
		http := fakeServerConf{addr: "10.0.0.1:9000"}
		eps, err := ParseStrEndpoints(http, nil)
		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		if eps[0] != "http://10.0.0.1:9000" {
			t.Fatalf("unexpected endpoint: %q", eps[0])
		}
	})

	t.Run("Fallback to non-loopback IPv4 or 127.0.0.1", func(t *testing.T) {
		_ = os.Unsetenv("ADVERTISE_HOST")
		http := fakeServerConf{addr: ":11002"} // 无 host，触发 4/5 级回退
		eps, err := ParseStrEndpoints(http, nil)
		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		got := eps[0]
		wantHost := "127.0.0.1"
		if h, e := firstNonLoopbackIPv4(); e == nil && h != "" {
			wantHost = h
		}
		want := "http://" + wantHost + ":11002"
		if got != want {
			t.Fatalf("unexpected endpoint:\n  got: %q\n  want: %q", got, want)
		}
	})

	t.Run("Both http & grpc returned; order and ports correct", func(t *testing.T) {
		_ = os.Unsetenv("ADVERTISE_HOST")
		http := fakeServerConf{addr: "0.0.0.0:8080", pub: "203.0.113.8:4000"}
		grpc := fakeServerConf{addr: "0.0.0.0:9090", pub: "grpc://203.0.113.9:5000"}
		eps, err := ParseStrEndpoints(http, grpc)
		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		if len(eps) != 2 {
			t.Fatalf("want 2 endpoints, got %d", len(eps))
		}
		if eps[0] != "http://203.0.113.8:8080" {
			t.Fatalf("unexpected http endpoint: %q", eps[0])
		}
		if eps[1] != "grpc://203.0.113.9:9090" {
			t.Fatalf("unexpected grpc endpoint: %q", eps[1])
		}
	})

	t.Run("Invalid addr -> error", func(t *testing.T) {
		_ = os.Unsetenv("ADVERTISE_HOST")
		http := fakeServerConf{addr: "0.0.0.0"} // 缺端口
		_, err := ParseStrEndpoints(http, nil)
		if err == nil {
			t.Fatalf("expected error for invalid addr, got nil")
		}
	})
}
