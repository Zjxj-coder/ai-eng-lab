package loadgen

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sort"
	"sync"
	"time"
)

type LoadResult struct {
	OverheadMin  time.Duration
	OverheadP50  time.Duration
	OverheadP99  time.Duration
	OverheadMax  time.Duration
	OverheadMean time.Duration
	QPS          float64
	Duration     time.Duration
}

type RequestFunc func(req *http.Request, rr *httptest.ResponseRecorder) time.Duration

func RunLoadTest(fn RequestFunc, concurrency int, duration time.Duration) LoadResult {
	var wg sync.WaitGroup
	
	ctx, cancel := context.WithTimeout(context.Background(), duration)
	defer cancel()

	var mu sync.Mutex
	var overheads []time.Duration
	var count int
	
	start := time.Now()

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				default:
					req, _ := http.NewRequestWithContext(ctx, "GET", "/", nil)
					rr := httptest.NewRecorder()
					
					overhead := fn(req, rr)
					
					mu.Lock()
					overheads = append(overheads, overhead)
					count++
					mu.Unlock()
				}
			}
		}()
	}
	
	wg.Wait()
	actualDuration := time.Since(start)

	if count == 0 {
		return LoadResult{}
	}

	sort.Slice(overheads, func(i, j int) bool {
		return overheads[i] < overheads[j]
	})

	min := overheads[0]
	p50 := overheads[int(float64(len(overheads))*0.50)]
	p99 := overheads[int(float64(len(overheads))*0.99)]
	max := overheads[len(overheads)-1]
	
	var sum int64
	for _, v := range overheads {
		sum += int64(v)
	}
	mean := time.Duration(sum / int64(len(overheads)))
	qps := float64(count) / actualDuration.Seconds()

	return LoadResult{
		OverheadMin:  min,
		OverheadP50:  p50,
		OverheadP99:  p99,
		OverheadMax:  max,
		OverheadMean: mean,
		QPS:          qps,
		Duration:     actualDuration,
	}
}
