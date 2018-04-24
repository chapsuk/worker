package main

import (
	"context"
	"log"
	"time"

	"github.com/chapsuk/grace"
	"github.com/chapsuk/worker"
)

var i int

func main() {
	ctx := grace.ShutdownContext(context.Background())
	stp := make(chan struct{})

	wk := worker.NewGroup()
	wk.Add(worker.New(criticalJob(stp)).ByTicker(time.Second))

	wk.Run()
	select {
	case <-ctx.Done():
	case <-stp:
	}

	time.Sleep(2 * time.Second)
	wk.Stop()
}

func criticalJob(stopChan chan struct{}) func(ctx context.Context) {
	return func(ctx context.Context) {
		log.Printf("i: %d", i)
		if i++; i == 5 {
			log.Printf("send stop signal from critical job")
			stopChan <- struct{}{}

			log.Printf("to avoid repeat run by schedule")
			<-ctx.Done()
		}
	}
}
