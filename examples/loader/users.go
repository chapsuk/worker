package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
)

type user struct {
	name string
}

var (
	users map[int32]user
	mu    sync.RWMutex
	seq   int32
)

func getUser(id int32) (user, bool) {
	mu.RLock()
	defer mu.RUnlock()
	u, ok := users[id]
	return u, ok
}

func loadBackground(ctx context.Context) {
	if err := load(ctx); err != nil {
		log.Printf("load error: %s", err)
	}
}

func load(ctx context.Context) error {
	mu.Lock()
	users = generate(50)
	mu.Unlock()
	log.Printf("users list updated, current sequance %d", atomic.LoadInt32(&seq))
	if atomic.LoadInt32(&seq)%100 == 0 {
		return errors.New("ooops")
	}
	return nil
}

func generate(count int) map[int32]user {
	result := make(map[int32]user, count)
	for i := 1; i <= count; i++ {
		result[atomic.AddInt32(&seq, 1)] = user{
			name: fmt.Sprintf("user#%d", i),
		}
	}
	return result
}
