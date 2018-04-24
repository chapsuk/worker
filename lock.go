package worker

import (
	"context"
	"time"

	"github.com/bsm/redis-lock"
	"github.com/go-redis/redis"
	"go.uber.org/zap"
)

type LockFunc func(context.Context, Job) Job

// Locker interface
type Locker interface {
	Lock() error
	Unlock()
}

// WithLock returns func with call Worker in lock
func WithLock(l Locker) LockFunc {
	return func(ctx context.Context, j Job) Job {
		return func(ctx context.Context) {
			if err := l.Lock(); err != nil {
				return
			}
			defer l.Unlock()
			j(ctx)
		}
	}
}

// ------ Redis family locks -------

type RedisClient interface {
	SetNX(key string, value interface{}, expiration time.Duration) *redis.BoolCmd
	Eval(script string, keys []string, args ...interface{}) *redis.Cmd
	EvalSha(sha1 string, keys []string, args ...interface{}) *redis.Cmd
	ScriptExists(scripts ...string) *redis.BoolSliceCmd
	ScriptLoad(script string) *redis.StringCmd
	Del(keys ...string) *redis.IntCmd
}

type redisLockLogger interface {
	Errorw(msg string, keysAndValues ...interface{})
	Warnw(msg string, keysAndValues ...interface{})
}

type RedisLockOptions struct {
	LockKey  string
	LockTTL  time.Duration
	RedisCLI RedisClient
	Logger   redisLockLogger
}

func (opts *RedisLockOptions) GetLogger() redisLockLogger {
	if opts.Logger == nil {
		return zap.S()
	}
	return opts.Logger
}

func WithRedisLock(opts RedisLockOptions) LockFunc {
	return func(ctx context.Context, j Job) Job {
		return func(ctx context.Context) {
			ok, err := opts.RedisCLI.SetNX(opts.LockKey, 1, opts.LockTTL).Result()
			if !ok {
				if err != nil {
					opts.GetLogger().Errorw(
						"worker get lock error",
						"lock_key", opts.LockKey, "error", err)
				} else {
					opts.GetLogger().Warnw("worker lock not obtained",
						"lock_key", opts.LockKey)
				}
				return
			}

			defer func() {
				if err := opts.RedisCLI.Del(opts.LockKey).Err(); err != nil {
					opts.GetLogger().Errorw("worker release lock error", "error", err)
				}
			}()

			j(ctx)
		}
	}
}

type BsmRedisLockOptions struct {
	RedisLockOptions
	RetryCount int
	RetryDelay time.Duration
}

func WithBsmRedisLock(opts BsmRedisLockOptions) LockFunc {
	return func(ctx context.Context, j Job) Job {
		return func(ctx context.Context) {
			l := lock.New(opts.RedisCLI, opts.LockKey, &lock.Options{
				LockTimeout: opts.LockTTL,
				RetryCount:  opts.RetryCount,
				RetryDelay:  opts.RetryDelay,
			})

			ok, err := l.LockWithContext(ctx)
			if !ok {
				if err == lock.ErrLockNotObtained {
					opts.GetLogger().Warnw("worker bsm/lock not obtained", "lock_key", opts.LockKey)
				} else if err != nil {
					opts.GetLogger().Errorw("worker get bsm/lock error", "lock_key", opts.LockKey, "error", err)
				}

				return
			}

			defer func() {
				if err := l.Unlock(); err != nil {
					opts.GetLogger().Warnw("worker release bsm/lock error",
						"lock_key", opts.LockKey, "error", err)
				}
			}()

			j(ctx)
		}
	}
}
