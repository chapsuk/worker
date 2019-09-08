# Worker

[![GoDoc](http://godoc.org/github.com/chapsuk/worker?status.png)](http://godoc.org/github.com/chapsuk/worker)
[![Build Status](https://travis-ci.org/chapsuk/worker.svg?branch=master)](https://travis-ci.org/chapsuk/worker)
[![codecov](https://codecov.io/gh/chapsuk/worker/branch/master/graph/badge.svg)](https://codecov.io/gh/chapsuk/worker)
[![Go Report Card](https://goreportcard.com/badge/github.com/chapsuk/worker?)](https://goreportcard.com/report/github.com/chapsuk/worker)
[![codebeat badge](https://codebeat.co/badges/3ddfb9a1-9fb9-49b2-ac72-b259822576aa)](https://codebeat.co/projects/github-com-chapsuk-worker-master)

Package worker adding the abstraction layer around background jobs,
allows make a job periodically, observe execution time and to control concurrent execution.

Group of workers allows to control jobs start time and
wait until all runned workers finished when we need stop all jobs.

## Features

* Scheduling, use one from existing `worker.By*` schedule functions. Supporting cron schedule spec format by [robfig/cron](https://github.com/robfig/cron) parser.
* Control concurrent execution around multiple instances by `worker.WithLock`. See existing lockers
* Observe a job execution time duration with `worker.SetObserever`. Friendly for [prometheus/client_golang](https://github.com/prometheus/client_golang/) package.
* Graceful stop, wait until all running jobs was completed.

## Example

```go
wg := worker.NewGroup()
wg.Add(
    worker.
        New(func(context.Context) {}).
        ByTicker(time.Second),

    worker.
        New(func(context.Context) {}).
        ByTimer(time.Second),

    worker.
        New(func(context.Context) {}).
        ByCronSpec("@every 1s"),
)
wg.Run()
```

## Lockers

You can use redis locks for controll exclusive job execution:

```go
l := locker.NewRedis(radix.Client, "job_lock_name", locker.RedisLockTTL(time.Minute))

w := worker.
        New(func(context.Context) {}).
        WithLock(l)

// Job will be executed only if `job_lock_name` redis key not exists.
w.Run(context.Background())
```
