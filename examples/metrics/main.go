package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"time"

	"github.com/chapsuk/grace"
	"github.com/chapsuk/worker"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	metric = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Namespace:  "test",
			Subsystem:  "jobs",
			Name:       "duration_seconds",
			Help:       "test duration",
			Objectives: map[float64]float64{0.5: 0.5, 1: 1},
		},
		[]string{"name"},
	)

	promaddr = flag.String("promaddr", ":8090", "prometheus bind address")
)

func main() {
	prometheus.Register(metric)
	flag.Parse()

	metric.WithLabelValues("test_job").Observe(1.0)
	go func() {
		log.Printf("starting prometheus http server %s", *promaddr)
		s := http.Server{
			Addr:    *promaddr,
			Handler: promhttp.Handler(),
		}
		if err := s.ListenAndServe(); err != nil {
			panic(err)
		}
	}()

	job1 := createJobWithTimeout("job1", time.Second)
	job2 := createJobWithTimeout("job2", time.Second)
	job3 := createJobWithTimeout("job3", time.Second)
	job4 := createJobWithTimeout("job4", time.Second)

	wk := worker.NewGroup()
	wk.Add(
		worker.New(job1).
			ByTicker(time.Second).
			WithMetrics(metric.WithLabelValues("job1").Observe),

		worker.New(job2).
			ByTicker(time.Second).
			WithMetrics(metric.WithLabelValues("job2").Observe),

		worker.New(job3).
			ByTicker(time.Second).
			WithMetrics(metric.WithLabelValues("job3").Observe),

		worker.New(job4).
			ByTicker(time.Second).
			WithMetrics(metric.WithLabelValues("job4").Observe),
	)

	log.Print("starting workers...")
	wk.Run()
	<-grace.ShutdownContext(context.Background()).Done()
	wk.Stop()
	log.Print("stopped")
}

func createJobWithTimeout(name string, timeout time.Duration) func(context.Context) {
	return func(ctx context.Context) {
		log.Printf("%s start", name)
		time.Sleep(timeout)
	}
}
