package locker

import (
	"errors"
	"fmt"
	"time"

	"github.com/mediocregopher/radix/v3"
)

// RedisOption allows redefine default redis locker settings
type RedisOption func(*Redis)

// RedisLockTTL sets ttl for lock key
func RedisLockTTL(ttl time.Duration) RedisOption {
	return func(r *Redis) {
		r.ttlms = fmt.Sprintf("%d", ttl.Milliseconds())
	}
}

// Redis locker
type Redis struct {
	key   string
	ttlms string
	me    string
	cli   radix.Client
}

// NewRedis returns redis locker
func NewRedis(cli radix.Client, lockKey string, opts ...RedisOption) *Redis {
	r := &Redis{
		key:   lockKey,
		ttlms: "60000",
		cli:   cli,
		me:    randomString(32),
	}

	for _, opt := range opts {
		opt(r)
	}

	return r
}

// Lock acquires lock
func (r *Redis) Lock() error {
	var res radix.MaybeNil
	err := r.cli.Do(radix.Cmd(&res, "SET", r.key, r.me, "NX", "PX", r.ttlms))
	if err != nil {
		return err
	} else if res.Nil {
		return errors.New("locked")
	}
	return nil
}

// Unlock release acquired lock
func (r *Redis) Unlock() {
	s := radix.NewEvalScript(1, `
	if redis.call("get",KEYS[1]) == ARGV[1]
then
    return redis.call("del",KEYS[1])
else
    return 0
end
`)

	r.cli.Do(s.Cmd(nil, r.key, r.me))
}
