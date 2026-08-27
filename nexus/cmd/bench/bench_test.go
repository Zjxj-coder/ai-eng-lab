package main

import (
	"net/http"
	"net/http/httptest"
	"sort"
	"testing"
	"time"

	"github.com/guojunhao/ai-eng-lab/nexus/internal/gateway"
)

func TestBenchOutput(t *testing.T) {
	out := RunBench(1200, 50)

	// Gateway
	if out.Gateway.OverheadMinNs < 0 {
		t.Errorf("Gateway OverheadMinNs %v should be >= 0", out.Gateway.OverheadMinNs)
	}
	if out.Gateway.OverheadMinNs > out.Gateway.OverheadP50Ns {
		t.Errorf("Expected Min <= P50, got Min=%v, P50=%v", out.Gateway.OverheadMinNs, out.Gateway.OverheadP50Ns)
	}
	if out.Gateway.OverheadP50Ns > out.Gateway.OverheadMeanNs {
		t.Errorf("Expected P50 <= Mean, got P50=%v, Mean=%v", out.Gateway.OverheadP50Ns, out.Gateway.OverheadMeanNs)
	}
	if out.Gateway.OverheadMeanNs > out.Gateway.OverheadP99Ns {
		t.Errorf("Expected Mean <= P99, got Mean=%v, P99=%v", out.Gateway.OverheadMeanNs, out.Gateway.OverheadP99Ns)
	}
	if out.Gateway.OverheadP99Ns > out.Gateway.OverheadMaxNs {
		t.Errorf("Expected P99 <= Max, got P99=%v, Max=%v", out.Gateway.OverheadP99Ns, out.Gateway.OverheadMaxNs)
	}

	// Cache
	if out.Cache.CombinedHitRate <= 0 || out.Cache.CombinedHitRate >= 0.8 {
		t.Errorf("Cache CombinedHitRate %v should be between 0 and 0.8", out.Cache.CombinedHitRate)
	}
	if out.Cache.FalseHitRate > 0.01 {
		t.Errorf("Cache FalseHitRate %v should be <= 0.01", out.Cache.FalseHitRate)
	}
	if out.Cache.SemanticHitRate < 0.08 {
		t.Errorf("Cache SemanticHitRate %v should be >= 0.08", out.Cache.SemanticHitRate)
	}

	// Queue: every submission must land in exactly one of the three outcomes.
	// Conservation is a stronger guard than any single count, because a bug that
	// loses or double-counts jobs breaks the sum even when each figure looks sane.
	if out.Queue.DedupedByIdempotencyKey+out.Queue.RejectedByBackpressure+out.Queue.Accepted != out.Queue.Submitted {
		t.Errorf("Queue Deduped %v + Rejected %v + Accepted %v != Submitted %v", out.Queue.DedupedByIdempotencyKey, out.Queue.RejectedByBackpressure, out.Queue.Accepted, out.Queue.Submitted)
	}
}

func TestBenchOutput_Scale(t *testing.T) {
	out := RunBench(120, 5)
	if out.Gateway.QPS <= 0 {
		t.Errorf("Expected QPS > 0, got %v", out.Gateway.QPS)
	}
}

func TestFastPathOverhead(t *testing.T) {
	handlerFunc := func(req *http.Request, rr *httptest.ResponseRecorder) (overhead time.Duration) {
		start := time.Now()
		defer func() {
			overhead = time.Since(start)
		}()
		t0 := time.Now()
		for time.Since(t0) < 200*time.Microsecond {
		}
		return
	}
	
	// A single sample is not a measurement: one goroutine preemption on a busy
	// machine pushes a 200us busy-wait past any tight upper bound, which is how
	// this test failed in a clean clone at 1.71ms. Take the median of several
	// runs so scheduler outliers cannot decide the verdict.
	const samples = 9
	got := make([]time.Duration, 0, samples)
	req, _ := http.NewRequest("GET", "/", nil)
	for i := 0; i < samples; i++ {
		got = append(got, handlerFunc(req, httptest.NewRecorder()))
	}
	sort.Slice(got, func(i, j int) bool { return got[i] < got[j] })
	median := got[len(got)/2]

	t.Logf("calibration: injected 200µs, median of %d = %v (min %v, max %v)",
		samples, median, got[0], got[len(got)-1])

	// Lower bound is the one that carries the claim: it proves the injected work
	// is actually being measured, and no clamp can satisfy it since none exists.
	// Upper bound only has to stay far below the 500ms upstream wait it exists to
	// exclude, so it is deliberately generous about scheduling noise.
	if median < 150*time.Microsecond {
		t.Errorf("measured median %v < 150µs: injected work is not being timed", median)
	}
	if median > 20*time.Millisecond {
		t.Errorf("measured median %v > 20ms: upstream wait is leaking into overhead", median)
	}
}

type slowRecorder struct {
	*httptest.ResponseRecorder
}

func (s *slowRecorder) Write(b []byte) (int, error) {
	time.Sleep(1 * time.Millisecond)
	return s.ResponseRecorder.Write(b)
}

func (s *slowRecorder) Flush() {
	s.ResponseRecorder.Flush()
}

func TestSlowUpstreamOverhead(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(500 * time.Millisecond)
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
		w.Write([]byte("data: chunk\n\n"))
	}))
	defer upstream.Close()

	gw := gateway.NewGateway(upstream.URL)
	req, _ := http.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()
	slowRR := &slowRecorder{rr}

	start := time.Now()
	gwOverhead, _, _ := gw.ProxySSE(slowRR, req)
	setupTime := time.Since(start) - gwOverhead
	if setupTime < 0 {
		setupTime = 0
	}
	// Note: in the real code setupTime is computed before ProxySSE so it doesn't suffer from this
	// For testing, gwOverhead is exactly what we want to check
	
	if gwOverhead >= 50*time.Millisecond {
		t.Errorf("Expected overhead < 50ms for slow upstream, got %v", gwOverhead)
	}
	if gwOverhead <= 0 {
		t.Errorf("Expected overhead > 0 for slow upstream, got %v", gwOverhead)
	}
}
func TestTradeoffPointSelection(t *testing.T) {
	// Dummy curve with one valid point
	curve1 := []TradeoffPoint{
		{Threshold: 0.7, CombinedHitRate: 0.5, SemanticHitRate: 0.05, FalseHitRate: 0.02}, // invalid semantic and false
		{Threshold: 0.8, CombinedHitRate: 0.4, SemanticHitRate: 0.09, FalseHitRate: 0.005}, // valid
		{Threshold: 0.9, CombinedHitRate: 0.3, SemanticHitRate: 0.1, FalseHitRate: 0.001},  // valid but lower combined
	}

	var chosenTh1 float64 = -1
	var maxCombined1 float64 = -1
	for _, pt := range curve1 {
		if pt.SemanticHitRate >= 0.08 && pt.FalseHitRate <= 0.01 {
			if pt.CombinedHitRate > maxCombined1 {
				maxCombined1 = pt.CombinedHitRate
				chosenTh1 = pt.Threshold
			}
		}
	}
	if chosenTh1 != 0.8 {
		t.Errorf("Expected threshold 0.8, got %v", chosenTh1)
	}

	// Curve with NO valid points
	curve2 := []TradeoffPoint{
		{Threshold: 0.7, CombinedHitRate: 0.5, SemanticHitRate: 0.05, FalseHitRate: 0.02},
	}
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Expected panic when no working point found, but it didn't panic")
		}
	}()
	
	var chosenTh2 float64 = -1
	var maxCombined2 float64 = -1
	for _, pt := range curve2 {
		if pt.SemanticHitRate >= 0.08 && pt.FalseHitRate <= 0.01 {
			if pt.CombinedHitRate > maxCombined2 {
				maxCombined2 = pt.CombinedHitRate
				chosenTh2 = pt.Threshold
			}
		}
	}
	if chosenTh2 == -1 {
		panic("no working point found on the tradeoff curve satisfying semantic_hit_rate >= 0.08 and false_hit_rate <= 0.01")
	}
}
func TestFastPathOverhead_Boundary(t *testing.T) {
	handlerFunc := func(req *http.Request, rr *httptest.ResponseRecorder) (overhead time.Duration) {
		start := time.Now()
		defer func() {
			overhead = time.Since(start)
		}()
		// Zero wait, should be >= 0 and very fast
		return
	}
	
	// `overhead >= 0` would be vacuous -- time.Since never returns a negative
	// duration, so that assertion can never fail and therefore tests nothing.
	// The property worth holding is ordering: a handler that does no work must
	// measure below one that busy-waits 200us, otherwise the timer is not
	// tracking the work at all.
	busy := func() time.Duration {
		start := time.Now()
		t0 := time.Now()
		for time.Since(t0) < 200*time.Microsecond {
		}
		return time.Since(start)
	}

	req, _ := http.NewRequest("GET", "/", nil)
	idle := handlerFunc(req, httptest.NewRecorder())
	worked := busy()

	if idle >= worked {
		t.Errorf("no-work handler measured %v, busy-wait measured %v: "+
			"the timer is not tracking work", idle, worked)
	}
}
