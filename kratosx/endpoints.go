package kratosx

import (
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

type ServerConf interface {
	GetAddr() string // 监听地址，如 ":11002" / "0.0.0.0:11002" / "10.0.0.1:10002"

	GetPublicEndpoint() string // 手动公开地址，如 "grpc://172.28.1.1:7411" 或 "172.28.1.1:7411"
}

// ParseEndpoints 生成用于注册中心的 endpoints（URL 形式），顺序为 http、grpc（如提供）。
// 规则（按优先级选 host；端口一律取自 GetAddr()）：
// 1) ADVERTISE_HOST 环境变量
// 2) GetPublicEndpoint（若包含 scheme 则使用该 scheme，否则用默认 scheme）
// 3) GetAddr 中的主机（排除 0.0.0.0/::/回环）
// 4) 本机非回环 IPv4
// 5) 127.0.0.1 兜底
func ParseEndpoints(httpConf, grpcConf ServerConf) ([]*url.URL, error) {
	var endpoints []*url.URL

	if httpConf != nil {
		ep, err := buildEndpointURL(httpConf, "http")
		if err != nil {
			return nil, err
		}
		if ep != nil {
			endpoints = append(endpoints, ep)
		}
	}
	if grpcConf != nil {
		ep, err := buildEndpointURL(grpcConf, "grpc")
		if err != nil {
			return nil, err
		}
		if ep != nil {
			endpoints = append(endpoints, ep)
		}
	}
	return endpoints, nil
}

// ParseStrEndpoints 生成用于注册中心的 endpoints（string 形式），顺序为 http、grpc（如提供）。
// 规则（按优先级选 host；端口一律取自 GetAddr()）：
// 1) ADVERTISE_HOST 环境变量
// 2) GetPublicEndpoint（若包含 scheme 则使用该 scheme，否则用默认 scheme）
// 3) GetAddr 中的主机（排除 0.0.0.0/::/回环）
// 4) 本机非回环 IPv4
// 5) 127.0.0.1 兜底
func ParseStrEndpoints(httpConf, grpcConf ServerConf) ([]string, error) {
	ups, err := ParseEndpoints(httpConf, grpcConf)
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(ups))
	for _, u := range ups {
		out = append(out, u.String())
	}
	return out, nil
}

func buildEndpointURL(sc ServerConf, defaultScheme string) (*url.URL, error) {
	addr := strings.TrimSpace(sc.GetAddr())
	if addr == "" {
		return nil, errors.New("addr is empty")
	}

	// 端口始终来自 GetAddr
	hostFromAddr, port, err := splitHostPortFlexible(addr)
	if err != nil {
		return nil, errors.WithMessage(err, "parse addr")
	}
	if port == "" {
		return nil, errors.Errorf("addr missing port: %q", addr)
	}

	// 1) ENV 优先（仅替换 host；端口仍用 addr）
	if envHost := strings.TrimSpace(os.Getenv("ADVERTISE_HOST")); envHost != "" {
		return makeURL(defaultScheme, envHost, port), nil
	}

	// 2) PublicEndpoint 次之：可携带 scheme；若不携带则用默认 scheme。
	//    只取 host，端口仍以 addr 为准。
	if pub := strings.TrimSpace(sc.GetPublicEndpoint()); pub != "" {
		pubHost, scheme, perr := parsePublicHostAndScheme(pub, defaultScheme)
		if perr != nil {
			return nil, errors.WithMessage(perr, "parse public endpoint")
		}
		if pubHost != "" {
			return makeURL(scheme, pubHost, port), nil
		}
	}

	// 3) addr 内的 host（排除 0.0.0.0/::/回环）
	if isUsableHost(hostFromAddr) {
		return makeURL(defaultScheme, hostFromAddr, port), nil
	}

	// 4) 本机非回环 IPv4
	if h, e := firstNonLoopbackIPv4(); e == nil && h != "" {
		return makeURL(defaultScheme, h, port), nil
	}

	// 5) 兜底
	return makeURL(defaultScheme, "127.0.0.1", port), nil
}

func makeURL(scheme, host, port string) *url.URL {
	return &url.URL{
		Scheme: scheme,
		Host:   joinHostPort(host, port), // JoinHostPort 会正确处理 IPv6 Brackets
	}
}

func splitHostPortFlexible(s string) (host, port string, err error) {
	s = strings.TrimSpace(s)
	// 典型形式（含冒号或 IPv6 Brackets）优先用标准拆分
	if strings.HasPrefix(s, "[") || strings.Count(s, ":") >= 1 {
		h, p, e := net.SplitHostPort(s)
		if e == nil {
			return stripBrackets(h), p, nil
		}
		// 兼容形如 ":11002"
		if strings.HasPrefix(s, ":") {
			return "", strings.TrimPrefix(s, ":"), nil
		}
	}

	// 兼容纯端口（"11002"）
	if _, e := strconv.Atoi(s); e == nil {
		return "", s, nil
	}
	return "", "", errors.Errorf("invalid addr %q; expect host:port or :port", s)
}

func parsePublicHostAndScheme(s, defaultScheme string) (host, scheme string, err error) {
	if strings.Contains(s, "://") {
		u, e := url.Parse(s)
		if e != nil {
			return "", "", e
		}
		scheme = u.Scheme
		if scheme == "" {
			scheme = defaultScheme
		}
		h := strings.TrimSpace(u.Host)
		if h == "" {
			return "", scheme, errors.Errorf("public endpoint missing host: %q", s)
		}
		return stripBrackets(hostOnly(h)), scheme, nil
	}

	// 不含 scheme 的 host:port
	h, _, e := splitHostPortFlexible(s)
	if e != nil {
		return "", "", e
	}
	return stripBrackets(h), defaultScheme, nil
}

func hostOnly(hostport string) string {
	h, _, err := net.SplitHostPort(hostport)
	if err == nil {
		return h
	}
	// 没端口时直接返回原串（如已包含 brackets 的纯 IPv6 主机）
	return hostport
}

func isUsableHost(h string) bool {
	h = stripBrackets(strings.TrimSpace(h))
	if h == "" || h == "0.0.0.0" || h == "::" {
		return false
	}
	if ip := net.ParseIP(h); ip != nil && ip.IsLoopback() {
		return false
	}
	return true
}

func firstNonLoopbackIPv4() (string, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return "", errors.WithStack(err)
	}
	for _, iface := range ifaces {
		if (iface.Flags&net.FlagUp) == 0 || (iface.Flags&net.FlagLoopback) != 0 {
			continue
		}
		addrs, _ := iface.Addrs()
		for _, a := range addrs {
			if in, ok := a.(*net.IPNet); ok && !in.IP.IsLoopback() {
				if v4 := in.IP.To4(); v4 != nil {
					return v4.String(), nil
				}
			}
		}
	}
	return "", errors.New("no non-loopback IPv4 found")
}

func stripBrackets(h string) string {
	return strings.Trim(h, "[]")
}

func joinHostPort(host, port string) string {
	// 传入裸 host，交给 JoinHostPort 正确加上 brackets（IPv6）
	return net.JoinHostPort(stripBrackets(host), port)
}
