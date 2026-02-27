package kratosx

import (
	"context"
	"sync"
	"time"

	"github.com/go-kratos/kratos/v2/registry"
	"github.com/go-kratos/kratos/v2/selector"
	"github.com/go-kratos/kratos/v2/selector/wrr"
	kgrpc "github.com/go-kratos/kratos/v2/transport/grpc"
	"github.com/go-kratos/kratos/v2/transport/grpc/resolver/discovery"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/keepalive"

	"github.com/jeffinity/singularity/nacosx"
)

const (
	idleTimeout   = 5 * time.Minute           // gRPC Channel 空闲 5min 即进入 IDLE
	cleanupPeriod = idleTimeout + time.Minute // 超过 IdleTimeout 再加 1min 彻底回收
)

type entry struct {
	conn    *grpc.ClientConn
	lastUse time.Time
}

type ConnFactory struct {
	discovery registry.Discovery

	mu    sync.RWMutex
	cache map[string]*entry
}

// NewConnFactory 使用默认的 WRR 负载均衡
func NewConnFactory(
	ctx context.Context,
	d *nacosx.Registry,
) (*ConnFactory, func(), error) {

	if selector.GlobalSelector() == nil {
		selector.SetGlobalSelector(wrr.NewBuilder())
	}

	f := &ConnFactory{
		discovery: d,
		cache:     make(map[string]*entry),
	}

	// janitor 协程：空闲回收 & ctx 退出
	go f.janitor(ctx)

	cleanup := func() { f.Close() }

	return f, cleanup, nil
}

// Conn 返回（或新建）到 service 的 *grpc.ClientConn
func (f *ConnFactory) Conn(
	ctx context.Context,
	service string,
) (*grpc.ClientConn, error) {

	f.mu.RLock()
	e, ok := f.cache[service]
	f.mu.RUnlock()

	if ok && e != nil && e.conn != nil {
		st := e.conn.GetState()
		if st != connectivity.Shutdown {
			e.lastUse = time.Now()
			return e.conn, nil
		}

		// 旧连接已经 Shutdown，丢弃，让后面走重拨逻辑
		f.mu.Lock()
		delete(f.cache, service)
		f.mu.Unlock()
	}

	return f.dialAndCache(ctx, service)
}

func (f *ConnFactory) dialAndCache(
	ctx context.Context,
	service string,
) (*grpc.ClientConn, error) {

	conn, err := kgrpc.DialInsecure(
		ctx,
		kgrpc.WithEndpoint("discovery:///"+service),
		kgrpc.WithTimeout(30*time.Second),
		kgrpc.WithOptions(
			grpc.WithIdleTimeout(idleTimeout),
			grpc.WithKeepaliveParams(keepalive.ClientParameters{
				Time:                30 * time.Second,
				Timeout:             20 * time.Second,
				PermitWithoutStream: true,
			}),
			grpc.WithResolvers(
				discovery.NewBuilder(
					f.discovery,
					discovery.PrintDebugLog(false),
					discovery.WithInsecure(true),
				),
			),
		),
	)
	if err != nil {
		return nil, err
	}

	f.mu.Lock()
	f.cache[service] = &entry{
		conn:    conn,
		lastUse: time.Now(),
	}
	f.mu.Unlock()
	return conn, nil
}

// Close 在进程退出时调用
func (f *ConnFactory) Close() {
	f.mu.Lock()
	defer f.mu.Unlock()
	for svc, e := range f.cache {
		_ = e.conn.Close()
		delete(f.cache, svc)
	}
}

// janitor：后台定期回收超时未用连接
func (f *ConnFactory) janitor(ctx context.Context) {
	tk := time.NewTicker(cleanupPeriod / 2)
	defer tk.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-tk.C:
			cutoff := time.Now().Add(-cleanupPeriod)

			f.mu.Lock()
			for svc, e := range f.cache {
				if e.lastUse.Before(cutoff) {
					_ = e.conn.Close()
					delete(f.cache, svc)
				}
			}
			f.mu.Unlock()
		}
	}
}
