package worker

import (
	"context"
	"time"

	"github.com/bsm/redis-lock"
	"github.com/go-redis/redis"
	"go.uber.org/zap"
)

// ------ Simple locker -------

// Locker interface
type Locker interface {
	Lock() error
	Unlock()
}

// WithLock returns func with call Worker in lock
func WithLock(w Worker, l Locker) Worker {
	return func(ctx context.Context) {
		if err := l.Lock(); err != nil {
			return
		}
		defer l.Unlock()

		w(ctx)
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

func WithRedisLock(w Worker, opts RedisLockOptions) Worker {
	return func(ctx context.Context) {
		ok, err := opts.RedisCLI.SetNX(opts.LockKey, 1, opts.LockTTL).Result()
		if !ok {
			if err != nil {
				opts.GetLogger().Errorw(
					"worker redis locker return error",
					"lock_key", opts.LockKey, "error", err)
			}
			return
		}

		defer func() {
			if err := opts.RedisCLI.Del(opts.LockKey); err != nil {
				opts.GetLogger().Errorw("worker release lock error", "error", err)
			}
		}()

		w(ctx)
	}
}

type BsmRedisLockOptions struct {
	RedisLockOptions
	RetryCount int
	RetryDelay time.Duration
}

func WithBsmRedisLock(w Worker, opts BsmRedisLockOptions) Worker {
	return func(ctx context.Context) {
		l, err := lock.Obtain(opts.RedisCLI, opts.LockKey, &lock.Options{
			LockTimeout: opts.LockTTL,
			RetryCount:  opts.RetryCount,
			RetryDelay:  opts.RetryDelay,
		})

		if err != nil {
			if err == lock.ErrLockNotObtained {
				opts.GetLogger().Warnw("worker lock not obtained", "lock_key", opts.LockKey)
			} else {
				opts.GetLogger().Errorw("worker get lock error", "lock_key", opts.LockKey, "error", err)
			}

			return
		}

		defer func() {
			if err := l.Unlock(); err != nil {
				opts.GetLogger().Warnw("worker release lock error",
					"lock_key", opts.LockKey, "error", err)
			}
		}()

		w(ctx)
	}
}
