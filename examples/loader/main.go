package main

import (
	"context"
	"log"
	"time"

	"github.com/chapsuk/grace"
	"github.com/chapsuk/worker"
)

func main() {
	ctx := grace.ShutdownContext(context.Background())

	// first load, we need check error before move on
	err := load(ctx)
	if err != nil {
		panic(err)
	}

	// run in background
	wk := worker.NewGroup()
	wk.Add(worker.New(loadBackground).ByTicker(time.Second))
	wk.Run()

	// wait and check loaded data
	for i := 0; i < 5; i++ {
		time.Sleep(900 * time.Millisecond)
		if _, ok := getUser(50 + int32(i)*50); !ok {
			panic("background load failed")
		}
		log.Printf("#%d check passed!", i+1)
	}

	// wait stop signal and stop workers
	<-ctx.Done()
	wk.Stop()
	log.Printf("all workers was stopped")
}
