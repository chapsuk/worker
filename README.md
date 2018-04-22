# Worker


## Example

```go
func main() {
	// Create controll group
	g := worker.NewGroup()

	// Init workers with wrappers for implement schedule or exclusive run
	w1 := worker.ByTicker(createWorker("worker #1"), time.Second)
	w2 := worker.ByTimer(createWorker("worker #2"), time.Second)
	w3 := worker.WithLock(createWorker("worker #3"), &locker{})
	w4 := worker.ByTicker(w3, time.Second)

	// Add workers to controll group
	g.Add(w1, w2, w3, w4)

	// Start each worker in separate goroutine
	g.Run()

	// Wait stop signal: SIGTERM, SIGINT
	<-grace.ShutdownContext(context.Background()).Done()

	// Stop workers and wait until all running jobs completed
	g.Stop()
}
```
