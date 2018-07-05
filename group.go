package worker

import (
	"context"
	"sync"

	"github.com/chapsuk/wait"
)

// Group of workers controlling background jobs execution
// allows graceful stop all running background jobs
type Group struct {
	workers []*Worker
	runned  bool
	mu      sync.Mutex
	wg      wait.Group
	ctx     context.Context
	stop    context.CancelFunc
}

// NewGroup yield new workers group
func NewGroup() *Group {
	return &Group{}
}

// Add workers to group, if group runned then start worker immediately
func (g *Group) Add(workers ...*Worker) {
	g.mu.Lock()
	for _, w := range workers {
		g.workers = append(g.workers, w)
		if g.runned {
			g.wg.AddWithContext(g.ctx, w.Run)
		}
	}
	g.mu.Unlock()
}

// Run starting each worker in separate goroutine with wait.Group control
func (g *Group) Run() {
	g.mu.Lock()
	g.runned = true
	g.ctx, g.stop = context.WithCancel(context.Background())
	for _, w := range g.workers {
		g.wg.AddWithContext(g.ctx, w.Run)
	}
	g.mu.Unlock()
}

// Stop cancel workers context and wait until all runned workers was completed.
// Be careful! It can be deadlock if some worker hanging
func (g *Group) Stop() {
	g.mu.Lock()
	g.stop()
	g.wg.Wait()
	g.mu.Unlock()
}
