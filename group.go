package worker

import (
	"context"
	"sync"

	"github.com/chapsuk/wait"
)

type Worker func(context.Context)

type Group struct {
	workers []Worker
	runned  bool
	mu      *sync.Mutex
	wg      *wait.Group
	ctx     context.Context
	stop    context.CancelFunc
}

func NewGroup() *Group {
	return &Group{
		mu: new(sync.Mutex),
		wg: new(wait.Group),
	}
}

func (g *Group) Add(w ...Worker) {
	g.mu.Lock()
	for _, w := range w {
		g.workers = append(g.workers, w)
		if g.runned {
			g.wg.AddWithContext(g.ctx, w)
		}
	}
	g.mu.Unlock()
}

func (g *Group) Run() {
	g.mu.Lock()
	g.runned = true
	g.ctx, g.stop = context.WithCancel(context.Background())
	for _, w := range g.workers {
		g.wg.AddWithContext(g.ctx, w)
	}
	g.mu.Unlock()
}

func (g *Group) Stop() {
	g.mu.Lock()
	g.stop()
	g.wg.Wait()
	g.mu.Unlock()
}
