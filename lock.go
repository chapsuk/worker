package worker

import (
	"context"
	"time"

	"github.com/bsm/redis-lock"
	"github.com/go-redis/redis"
	"go.uber.org/zap"
)

// LockFunc is job wrapper for controll exclusive execution
type LockFunc func(context.Context, Job) Job

// Locker interface
type Locker interface {
	// Lock acquire lock for job, returns error when the job should not be started
	Lock() error
	// Unlock release acquired lock
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

// RedisClient interface define all used functions from github.com/go-redis/redis
type RedisClient interface {
	SetNX(key string, value interface{}, expiration time.Duration) *redis.BoolCmd
	Eval(script string, keys []string, args ...interface{}) *redis.Cmd
	EvalSha(sha1 string, keys []string, args ...interface{}) *redis.Cmd
	ScriptExists(scripts ...string) *redis.BoolSliceCmd
	ScriptLoad(script string) *redis.StringCmd
	Del(keys ...string) *redis.IntCmd
}

// RedisLockLogger all needed logger funcs
type RedisLockLogger interface {
	Errorw(msg string, keysAndValues ...interface{})
	Warnw(msg string, keysAndValues ...interface{})
}

// RedisLockOptions describe redis lock settings
type RedisLockOptions struct {
	LockKey  string
	LockTTL  time.Duration
	RedisCLI RedisClient
	Logger   RedisLockLogger
}

// GetLogger return logger from options or zap.Noop logger
func (opts *RedisLockOptions) GetLogger() RedisLockLogger {
	if opts.Logger == nil {
		return zap.S()
	}
	return opts.Logger
}

// NewWith returns new redis lock options with given key name and ttl from source options
func (opts RedisLockOptions) NewWith(lockkey string, lockttl time.Duration) RedisLockOptions {
	opts.LockKey = lockkey
	opts.LockTTL = lockttl
	return opts
}

// WithRedisLock returns job wrapper func with redis lock
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

// BsmRedisLockOptions lock options for bsm/redis-lock locker
type BsmRedisLockOptions struct {
	RedisLockOptions
	RetryCount int
	RetryDelay time.Duration
}

// NewWith returns new bsm redis lock options from target options
func (opts BsmRedisLockOptions) NewWith(
	lockkey string,
	lockttl time.Duration,
	retries int,
	retryDelay time.Duration,
) BsmRedisLockOptions {
	opts.LockKey = lockkey
	opts.LockTTL = lockttl
	opts.RetryCount = retries
	opts.RetryDelay = retryDelay
	return opts
}

// WithBsmRedisLock returns job wrapper func with redis lock by bsm/redis-lock pkg
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
				if err != nil {
					opts.GetLogger().Errorw("worker get bsm/lock error", "lock_key", opts.LockKey, "error", err)
				} else {
					opts.GetLogger().Warnw("worker bsm/lock not obtained", "lock_key", opts.LockKey)
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
