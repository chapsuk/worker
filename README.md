# Jobs package

is collection of wrappers `func(contex.Context)`, allows to avoid boilerplate code for running background jobs.

## Example

```go

import (
    "context"
    "time"
    "log"

    "github.com/chapsuk/job"
)

func main() {
	tickerJob := job.ByTicker(time.Second, func(ctx context.Context) {
		time.Sleep(2 * time.Second)
		log.Print("ticker")
	})

	timerJob := job.ByTimer(time.Second, func(ctx context.Context) {
		time.Sleep(2 * time.Second)
		log.Print("timer")
	})

	go tickerJob(context.TODO())
	go timerJob(context.TODO())
	select {}
}

// Output: 
//
// ≻ go run example/example.go
// 2018/04/22 13:01:51 timer
// 2018/04/22 13:01:51 ticker
// 2018/04/22 13:01:53 ticker
// 2018/04/22 13:01:54 timer
// 2018/04/22 13:01:55 ticker
// 2018/04/22 13:01:57 ticker
// 2018/04/22 13:01:57 timer
// 2018/04/22 13:01:59 ticker
// 2018/04/22 13:02:00 timer
// 2018/04/22 13:02:01 ticker
// 2018/04/22 13:02:03 ticker
// 2018/04/22 13:02:03 timer
// 2018/04/22 13:02:05 ticker
```
