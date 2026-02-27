package friendly

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/redis/go-redis/v9"
)

// RedisLeader 通过 Redis 实现单点执行的领导者选举：
// - 以 SET NX PX 获取锁（TTL=ttl）
// - 周期性校验并续期（仅当 value 匹配本实例 id 时）
// - 丢失领导权后自动退出工作回调并重试抢占
// - Stop 时安全释放（compare-and-del）
// 兼容：*redis.Client / *redis.ClusterClient（通过 redis.Cmdable）
type RedisLeader struct {
	rdb        redis.Cmdable
	key        string
	id         string
	ttl        time.Duration
	renewEvery time.Duration
	logger     *log.Helper
	onStarted  func(ctx context.Context)
	onStopped  func(context.Context)

	running atomic.Bool
	cancel  context.CancelFunc
	wg      sync.WaitGroup
}

// NewRedisLeader：单机版构造器
func NewRedisLeader(
	rdb *redis.Client,
	key string,
	ttl, renewEvery time.Duration,
	logger log.Logger,
	onStarted func(context.Context),
	onStopped func(context.Context),
) *RedisLeader {
	return newRedisLeader(rdb, key, ttl, renewEvery, logger, onStarted, onStopped)
}

// NewRedisClusterLeader：集群版构造器
func NewRedisClusterLeader(
	rdb *redis.ClusterClient,
	key string,
	ttl, renewEvery time.Duration,
	logger log.Logger,
	onStarted func(context.Context),
	onStopped func(context.Context),
) *RedisLeader {
	return newRedisLeader(rdb, key, ttl, renewEvery, logger, onStarted, onStopped)
}

func newRedisLeader(
	rdb redis.Cmdable,
	key string,
	ttl, renewEvery time.Duration,
	logger log.Logger,
	onStarted func(context.Context),
	onStopped func(context.Context),
) *RedisLeader {
	if renewEvery <= 0 || (ttl > 0 && renewEvery >= ttl) {
		renewEvery = ttl / 3
	}
	return &RedisLeader{
		rdb:        rdb,
		key:        key,
		id:         randID(),
		ttl:        ttl,
		renewEvery: renewEvery,
		logger:     log.NewHelper(log.With(logger, "module", "leader/redis")),
		onStarted:  onStarted,
		onStopped:  onStopped,
	}
}

// Start 启动领导者循环：抢占 → 续期 → 丢失后重试
func (l *RedisLeader) Start(ctx context.Context) error {
	if !l.running.CompareAndSwap(false, true) {
		return nil
	}
	runCtx, cancel := context.WithCancel(ctx)
	l.cancel = cancel

	l.wg.Add(1)
	go l.loop(runCtx)

	l.logger.Infof("leader loop started, key=%s ttl=%s renew=%s id=%s",
		l.key, l.ttl, l.renewEvery, l.id)
	return nil
}

// Stop 停止并释放锁
func (l *RedisLeader) Stop(ctx context.Context) error {
	if !l.running.CompareAndSwap(true, false) {
		return nil
	}
	if l.cancel != nil {
		l.cancel()
	}
	done := make(chan struct{})
	go func() {
		defer close(done)
		l.wg.Wait()
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (l *RedisLeader) loop(ctx context.Context) {
	defer l.wg.Done()

	var (
		isLeader   bool
		workCancel context.CancelFunc
	)

	renewTicker := time.NewTicker(l.renewEvery)
	defer renewTicker.Stop()

	backoff := 300 * time.Millisecond

	for {
		if ctx.Err() != nil {
			l.stopIfLeader(&isLeader, &workCancel, true) // ctx 退出：需要闭环 onStopped
			return
		}

		if !isLeader {
			ok, err := l.acquire(ctx)
			if err != nil {
				l.logger.Warnf("acquire error: %+v", err)
				sleepJitter(ctx, backoff)
				continue
			}
			if !ok {
				sleepJitter(ctx, backoff)
				continue
			}

			isLeader = true
			workCancel = l.startWork(ctx)
			continue
		}

		select {
		case <-ctx.Done():
			l.stopIfLeader(&isLeader, &workCancel, true)
			return

		case <-renewTicker.C:
			ok, err := l.renew(ctx)
			if err != nil {
				// 出错不等于丢失领导权，下一轮再尝试
				l.logger.Warnf("renew error: %+v", err)
				continue
			}
			if !ok {
				// 领导权被抢/丢失：停止工作并闭环 onStopped，但不 release（因为已不是我们的锁）
				l.stopIfLeader(&isLeader, &workCancel, false)
			}
		}
	}
}

func (l *RedisLeader) startWork(parent context.Context) context.CancelFunc {
	workCtx, cancel := context.WithCancel(parent)
	if l.onStarted != nil {
		go l.onStarted(workCtx)
	}
	return cancel
}

// stopIfLeader：保证 onStarted/onStopped 闭环；releaseOnlyWhenOwned=true 时会尝试释放锁
func (l *RedisLeader) stopIfLeader(isLeader *bool, workCancel *context.CancelFunc, releaseOnlyWhenOwned bool) {
	if isLeader == nil || !*isLeader {
		return
	}
	*isLeader = false

	// 先 cancel，让工作尽快停，再释放锁，降低“新 leader 已开始但旧 worker 仍在跑”的窗口
	if workCancel != nil && *workCancel != nil {
		(*workCancel)()
		*workCancel = nil
	}

	if l.onStopped != nil {
		go l.onStopped(context.Background())
	}

	if releaseOnlyWhenOwned {
		l.release(context.Background())
	}
}

func (l *RedisLeader) acquire(ctx context.Context) (bool, error) {
	return l.rdb.SetNX(ctx, l.key, l.id, l.ttl).Result()
}

func (l *RedisLeader) renew(ctx context.Context) (bool, error) {
	// 仅当 value 匹配本实例 id 时续期
	const lua = `
if redis.call('GET', KEYS[1]) == ARGV[1] then
  return redis.call('PEXPIRE', KEYS[1], ARGV[2])
end
return 0
`
	res, err := l.rdb.Eval(ctx, lua, []string{l.key}, l.id, int(l.ttl/time.Millisecond)).Result()
	if err != nil {
		return false, err
	}
	switch v := res.(type) {
	case int64:
		return v == 1, nil
	case int:
		return v == 1, nil
	default:
		return false, nil
	}
}

func (l *RedisLeader) release(ctx context.Context) {
	// compare-and-del，避免误删他人锁
	const lua = `
if redis.call('GET', KEYS[1]) == ARGV[1] then
  return redis.call('DEL', KEYS[1])
end
return 0
`
	_, _ = l.rdb.Eval(ctx, lua, []string{l.key}, l.id).Result()
}

func randID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

func sleepJitter(ctx context.Context, d time.Duration) {
	t := time.NewTimer(d + time.Duration(randByte()%250)*time.Millisecond)
	defer t.Stop()
	select {
	case <-ctx.Done():
	case <-t.C:
	}
}

func randByte() byte {
	var b [1]byte
	_, _ = rand.Read(b[:])
	return b[0]
}
