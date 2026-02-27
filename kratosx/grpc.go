package kratosx

import (
	"context"
	"net"
	"time"

	"github.com/pkg/errors"
	"golang.org/x/net/proxy"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials/insecure"
)

func NewGrpcConn(target string, opts ...grpc.DialOption) (*grpc.ClientConn, error) {
	cc, err := grpc.NewClient(
		target,
		append([]grpc.DialOption{
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		}, opts...)...,
	)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return cc, nil
}

func NewGrpcConnAndTest(ctx context.Context, target string, timeout time.Duration, opts ...grpc.DialOption) (*grpc.ClientConn, error) {

	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cc, err := NewGrpcConn(target, opts...)
	if err != nil {
		return nil, err
	}
	cc.Connect()

	for {
		s := cc.GetState()
		if s == connectivity.Ready {
			return cc, nil // 成功
		}
		if !cc.WaitForStateChange(timeoutCtx, s) {
			_ = cc.Close()
			return nil, errors.WithStack(err) // 超时 / 取消
		}
	}
}

func NewGrpcConnWithSS(proxyAddr, target string, opts ...grpc.DialOption) (*grpc.ClientConn, error) {

	if proxyAddr != "" {
		dialer, err := proxy.SOCKS5("tcp", proxyAddr, nil, proxy.Direct)
		if err != nil {
			return nil, errors.Wrap(err, "failed to create socks5 dialer")
		}

		ctxDialer := func(ctx context.Context, addr string) (net.Conn, error) {
			return dialer.Dial("tcp", addr)
		}
		opts = append(opts, grpc.WithContextDialer(ctxDialer))
	}
	return NewGrpcConn(target, opts...)
}

type NewClientFunc[C any] func(grpc.ClientConnInterface) C

func GetGrpcClient[C any](ctx context.Context, factory *ConnFactory, service ServiceName, fn NewClientFunc[C]) (C, error) {
	conn, err := factory.Conn(ctx, service)
	if err != nil {
		var zero C
		return zero, errors.WithStack(err)
	}
	return fn(conn), nil
}
