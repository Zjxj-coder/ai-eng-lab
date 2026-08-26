package gateway

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"runtime"
	"testing"
	"time"
)

func TestGateway_ContextCancellation(t *testing.T) {
	// Start an upstream server that streams slowly
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		flusher, _ := w.(http.Flusher)
		for i := 0; i < 100; i++ {
			select {
			case <-r.Context().Done():
				return
			default:
				fmt.Fprintf(w, "data: chunk %d\n\n", i)
				flusher.Flush()
				time.Sleep(50 * time.Millisecond)
			}
		}
	}))
	defer upstream.Close()

	gw := NewGateway(upstream.URL)

	initialGoroutines := runtime.NumGoroutine()

	req, _ := http.NewRequest("GET", "/", nil)
	ctx, cancel := context.WithCancel(context.Background())
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()

	done := make(chan struct{})
	go func() {
		gw.ProxySSE(rr, req)
		close(done)
	}()

	// Wait a bit to let stream start
	time.Sleep(100 * time.Millisecond)
	
	// Cancel the request
	cancel()

	// Wait for the handler to exit
	select {
	case <-done:
	case <-time.After(1 * time.Second):
		t.Fatal("handler did not exit after context cancellation")
	}

	time.Sleep(100 * time.Millisecond) // Let goroutines settle
	
	finalGoroutines := runtime.NumGoroutine()
	// allow a small diff due to internal go test or httptest internals, but not leaked proxy handlers
	if finalGoroutines > initialGoroutines+2 { 
		t.Fatalf("goroutine leak: initial %d, final %d", initialGoroutines, finalGoroutines)
	}
}

func TestGateway_Backpressure(t *testing.T) {
	// Not fully implemented in Gateway struct, we simulate checking max concurrent requests if added.
	// For now just basic test.
}

func TestGateway_UpstreamDisconnect(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		flusher, _ := w.(http.Flusher)
		fmt.Fprintf(w, "data: chunk 1\n\n")
		flusher.Flush()
		
		// Immediately close connection
		hj, _ := w.(http.Hijacker)
		conn, _, _ := hj.Hijack()
		conn.Close()
	}))
	defer upstream.Close()

	gw := NewGateway(upstream.URL)
	req, _ := http.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()
	
	_, _, err := gw.ProxySSE(rr, req)
	if err == nil {
		t.Error("expected error due to upstream disconnect")
	}
}

func TestGateway_GracefulDegradationError(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer upstream.Close()

	gw := NewGateway(upstream.URL)
	req, _ := http.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()
	
	gw.ProxySSE(rr, req)
	
	if rr.Code != http.StatusBadGateway {
		t.Errorf("expected 502 Bad Gateway for upstream 500, got %d", rr.Code)
	}
}

func TestGateway_SSEChunkBoundary(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		// send split chunk
		w.Write([]byte("data: half"))
		if f, ok := w.(http.Flusher); ok { f.Flush() }
		time.Sleep(10*time.Millisecond)
		w.Write([]byte("chunk\n\n"))
		if f, ok := w.(http.Flusher); ok { f.Flush() }
	}))
	defer upstream.Close()

	gw := NewGateway(upstream.URL)
	req, _ := http.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()
	
	gw.ProxySSE(rr, req)
	
	if rr.Body.String() != "data: halfchunk\n\n" {
		t.Errorf("unexpected body: %q", rr.Body.String())
	}
}

type noFlushWriter struct {
	http.ResponseWriter
}

func TestGateway_FlusherUnsupported(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	gw := NewGateway(upstream.URL)
	req, _ := http.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()
	
	// Wrap rr so it doesn't implement Flusher
	nfw := noFlushWriter{rr}
	
	_, _, err := gw.ProxySSE(nfw, req)
	if err == nil || err.Error() != "streaming unsupported" {
		t.Errorf("expected 'streaming unsupported' error, got %v", err)
	}
}
