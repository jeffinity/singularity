package friendly

import (
	"context"
	"fmt"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/pkg/errors"
	"github.com/redis/go-redis/v9"
)

func NewRedisCluster(
	rootCtx context.Context,
	mLogger log.Logger,
	seeds []string,
	passwd string,
	readOnly bool,
) (*redis.ClusterClient, func(), error) {
	hl := log.NewHelper(log.With(mLogger, "module", "redis"))

	if len(seeds) == 0 {
		return nil, nil, fmt.Errorf("redis.seeds is empty")
	}

	co := &redis.ClusterOptions{
		Addrs:                 seeds,
		Password:              passwd,
		ReadOnly:              readOnly,
		PoolSize:              50,
		MinIdleConns:          5,
		ConnMaxLifetime:       30 * time.Minute,
		ConnMaxIdleTime:       5 * time.Minute,
		DialTimeout:           10 * time.Second,
		ReadTimeout:           10 * time.Second,
		WriteTimeout:          10 * time.Second,
		PoolTimeout:           10 * time.Second,
		ContextTimeoutEnabled: true,
	}
	client := redis.NewClusterClient(co)
	cleanup := func() {
		CloseQuietly(client)
	}

	// 非致命探活：只打日志，不影响启动
	go func() {
		ctx, cancel := context.WithTimeout(rootCtx, 2*time.Second)
		defer cancel()
		if err := client.Ping(ctx).Err(); err != nil {
			hl.Warnf("Redis ping failed at startup (will retry on use): %v", err)
			return
		}
		hl.Infof("Redis ping OK at startup")
	}()

	hl.Infof("Redis cluster client created seeds=%v readonly=%v (lazy connect)", co.Addrs, co.ReadOnly)
	return client, cleanup, nil
}

func NewRedis(rootCtx context.Context, mLogger log.Logger, dsn string) (*redis.Client, func(), error) {
	hl := log.NewHelper(log.With(mLogger, "module", "redis"))
	opts, err := redis.ParseURL(dsn)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid redis DSN: %w", err)
	}

	opts.PoolSize = 50                      // 最大连接数
	opts.MinIdleConns = 5                   // 最小空闲连接数
	opts.ConnMaxLifetime = 30 * time.Minute // 连接最大存活时间
	opts.ConnMaxIdleTime = 5 * time.Minute  // 空闲连接超时释放
	opts.DialTimeout = 10 * time.Second     // 建立连接超时
	opts.ReadTimeout = 10 * time.Second     // 读超时
	opts.WriteTimeout = 10 * time.Second    // 写超时
	opts.PoolTimeout = 20 * time.Second     // 获取连接最大等待时间

	client := redis.NewClient(opts)
	cleanup := func() {
		CloseQuietly(client)
	}

	ctx, cancel := context.WithTimeout(rootCtx, 12*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, cleanup, errors.WithMessage(err, "redis ping failed:")
	}

	hl.Infof("Connected to redis @ %s db=%d", opts.Addr, opts.DB)
	return client, cleanup, nil
}
