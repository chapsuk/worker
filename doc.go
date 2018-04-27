/*
Package worker adding the abstraction layer around background jobs,
allows make a job periodically, observe execution time and to control concurrent execution.
Group of workers allows to control jobs start time and
wait until all runned workers finished when we need stop all jobs.

Usage

Create group and workers with empty job:

	wg := worker.NewGroup()
	w1 := worker.New(func(context.Context) {})
	w2 := worker.New(func(context.Context) {})
	w3 := worker.New(func(context.Context) {})

Add workers to group and run all jobs:

	wg.Add(w1, w2, w3)
	wg.Run()

Stop all workers:

	wg.Stop()

Periodic jobs

Set job execution period to worker (only the last will be applied)

	w := worker.New(func(context.Context) {})
	w.ByTicker(time.Second)
	w.ByTimer(time.Second)
	w.ByCronSpec("@every 1s")

or set custom schedule function

	// run 3 times
	w.BySchedule(func(ctx context.Context, j worker.Job) worker.Job {
		return func(ctx context.Context) {
			for i := 0; i < 3; i++ {
				j(ctx)
			}
		}
	})

Exclusive jobs

Control concurrent execution around single or multiple instances by redis locks

	worker.
		New(func(context.Context) {}).
		WithRedisLock(&worker.RedisLockOptions{}).
		Run(context.Background())

or set custom locker

	w.WithLock(worker.Locker)

Observe execution time

Collect job execution time metrics

	w.SetObserver(func(d float64) {
		fmt.Printf("time elapsed %.3fs", d)
	})
*/
package worker
