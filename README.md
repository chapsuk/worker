# Worker


## Example

```go
func main() {
	// Create controll group
	g := worker.NewGroup()

	// Init workers with wrappers for implement schedule or exclusive run
	w1 := worker.ByTicker(time.Second, createWorker("worker #1"))
	w2 := worker.ByTimer(time.Second, createWorker("worker #2"))
	w3 := worker.WithLock(&locker{}, createWorker("worker #3"))
	w4 := worker.Many(10, w3)
	w5 := worker.ByTicker(time.Second, w4)

	// Add workers to controll group
	g.Add(w1, w2, w3, w4, w5)

	// Start each worker in separate goroutine
	g.Run()

	// Wait stop signal: SIGTERM, SIGINT
	<-grace.ShutdownContext(context.Background()).Done()

	// Stop workers and wait until all running jobs completed
	g.Stop()
}
```
