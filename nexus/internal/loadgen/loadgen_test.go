package loadgen

import (
	"net/http"
	"net/http/httptest"
	"sort"
	"testing"
	"time"

	"github.com/guojunhao/ai-eng-lab/nexus/internal/gateway"
)

func TestCalculatePercentiles(t *testing.T) {
	// Generate known durations
	var overheads []time.Duration
	for i := 1; i <= 100; i++ {
		overheads = append(overheads, time.Duration(i)*time.Millisecond)
	}

	// Scramble the array to test sorting
	scrambled := make([]time.Duration, len(overheads))
	copy(scrambled, overheads)
	// simple swap to scramble
	scrambled[0], scrambled[99] = scrambled[99], scrambled[0]
	scrambled[50], scrambled[49] = scrambled[49], scrambled[50]

	sort.Slice(scrambled, func(i, j int) bool {
		return scrambled[i] < scrambled[j]
	})

	min := scrambled[0]
	p50 := scrambled[int(float64(len(scrambled))*0.50)]
	p99 := scrambled[int(float64(len(scrambled))*0.99)]
	max := scrambled[len(scrambled)-1]

	if min != 1*time.Millisecond {
		t.Errorf("Expected min 1ms, got %v", min)
	}
	if p50 != 51*time.Millisecond { // index 50 is 51st element (51ms)
		t.Errorf("Expected p50 51ms, got %v", p50)
	}
	if p99 != 100*time.Millisecond { // index 99 is 100th element (100ms)
		t.Errorf("Expected p99 100ms, got %v", p99)
	}
	if max != 100*time.Millisecond {
		t.Errorf("Expected max 100ms, got %v", max)
	}
}

func TestLoadgen_OverheadMeasurement(t *testing.T) {
	// D1 Requirement: 在 handler 里注入一个已知耗时（如 200µs）的假工作，断言测出来的 overhead ≥ 该值。
	res := RunLoadTest(func(req *http.Request, rr *httptest.ResponseRecorder) time.Duration {
		start := time.Now()
		time.Sleep(200 * time.Microsecond)
		return time.Since(start)
	}, 1, 10*time.Millisecond)

	if res.OverheadMin < 200*time.Microsecond {
		t.Errorf("Expected overhead >= 200µs, got %v", res.OverheadMin)
	}
}

func TestLoadgen_ConcurrentCollection(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("data: chunk\n\n"))
	}))
	defer upstream.Close()

	gw := gateway.NewGateway(upstream.URL)
	res := RunLoadTest(func(req *http.Request, rr *httptest.ResponseRecorder) time.Duration {
		overhead, _, _ := gw.ProxySSE(rr, req)
		return overhead
	}, 10, 50*time.Millisecond)

	if res.QPS <= 0 {
		t.Errorf("Expected QPS > 0, got %v", res.QPS)
	}
	if res.Duration < 50*time.Millisecond {
		t.Errorf("Expected duration >= 50ms, got %v", res.Duration)
	}
}
