package locker_test

import (
	"testing"
	"time"

	"github.com/chapsuk/worker/locker"
	"github.com/mediocregopher/radix/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRedisLock(t *testing.T) {
	cli, err := radix.NewPool("tcp", "127.0.0.1:6379", 1)
	require.NoError(t, err)

	r := locker.NewRedis(cli, t.Name())
	assert.NotNil(t, r)

	assert.NoError(t, r.Lock())
	assert.Error(t, r.Lock())
	assert.NotPanics(t, r.Unlock)
	assert.NoError(t, r.Lock())
	assert.NotPanics(t, r.Unlock)
}

func TestRedisLockExpiredUnlock(t *testing.T) {
	cli, err := radix.NewPool("tcp", "127.0.0.1:6379", 1)
	require.NoError(t, err)

	l1 := locker.NewRedis(cli, t.Name(), locker.RedisLockTTL(10*time.Millisecond))
	l2 := locker.NewRedis(cli, t.Name(), locker.RedisLockTTL(60*time.Second))

	assert.NoError(t, l1.Lock())
	<-time.After(20 * time.Millisecond)
	assert.NoError(t, l2.Lock())

	assert.NotPanics(t, l1.Unlock)

	var rcv string
	err = cli.Do(radix.Cmd(&rcv, "GET", t.Name()))
	assert.NoError(t, err)
	assert.NotEmpty(t, rcv)
	assert.NotPanics(t, l2.Unlock)
}

func TestRedisLockError(t *testing.T) {
	cli, err := radix.NewPool("tcp", "127.0.0.1:6379", 1)
	require.NoError(t, err)

	r := locker.NewRedis(cli, t.Name())
	assert.NotNil(t, r)

	assert.NoError(t, cli.Close())
	assert.Error(t, r.Lock())
}
