package worker

import (
	"context"
	"time"

	"github.com/bsm/redis-lock"
	"github.com/go-redis/redis"
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
type redisLockLogger interface {
	Errorw(msg string, keysAndValues ...interface{})
	Warnw(msg string, keysAndValues ...interface{})
}

type RedisLockOptions struct {
	LockKey  string
	LockTTL  time.Duration
	RedisCLI *redis.Client
	Logger   redisLockLogger
}

func WithRedisLock(w Worker, opts RedisLockOptions) Worker {
	return func(ctx context.Context) {
		ok, err := opts.RedisCLI.SetNX(opts.LockKey, 1, opts.LockTTL).Result()
		if !ok {
			opts.Logger.Errorw("worker redis locker return error", "lock_key", opts.LockKey, "error", err.Error())
			return
		}

		defer func() {
			if err := opts.RedisCLI.Del(opts.LockKey); err != nil {
				opts.Logger.Errorw("worker release ")
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
				opts.Logger.Warnw("worker lock not obtained", "lock_key", opts.LockKey)
			}
			opts.Logger.Errorw("worker get lock error",
				"lock_key", opts.LockKey, "error", err)
			return
		}

		defer func() {
			if err := l.Unlock(); err != nil {
				opts.Logger.Warnw("worker release lock error",
					"lock_key", opts.LockKey, "error", err)
			}
		}()

		w(ctx)
	}
}
