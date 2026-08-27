package main

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestSSEHeaders(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	req := httptest.NewRequest("GET", "/ask", nil).WithContext(ctx)
	w := httptest.NewRecorder()
	go HandleSSE(w, req)
	cancel()
	time.Sleep(10 * time.Millisecond) // Give it a moment to run

	if w.Header().Get("Content-Type") != "text/event-stream" {
		t.Errorf("wrong content type")
	}
	if w.Header().Get("Cache-Control") != "no-cache" {
		t.Errorf("wrong cache control")
	}
}

func TestSSEEventsSent(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	req := httptest.NewRequest("GET", "/ask", nil).WithContext(ctx)
	w := httptest.NewRecorder()
	go HandleSSE(w, req)
	time.Sleep(50 * time.Millisecond) // Let events generate
	cancel()
	time.Sleep(10 * time.Millisecond) // Let handler return

	body := w.Body.String()
	if !strings.Contains(body, "event: conclusion") {
		t.Errorf("missing conclusion event")
	}
	if !strings.Contains(body, "event: chart") {
		t.Errorf("missing chart event")
	}
	if !strings.Contains(body, "DAU: Daily Active Users") {
		t.Errorf("missing trace metric def in output")
	}
}

func TestSSEDisconnect(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	req := httptest.NewRequest("GET", "/ask", nil).WithContext(ctx)
	w := httptest.NewRecorder()

	done := make(chan bool)
	go func() {
		HandleSSE(w, req)
		done <- true
	}()

	cancel()
	
	select {
	case <-done:
		// test passed, handler returned early
	case <-time.After(1 * time.Second):
		t.Errorf("handler did not exit after cancel")
	}
}

// Every figure the UI shows has to name the SQL and the metric definition that
// produced it -- a number a planner cannot trace back is a number they cannot
// act on. One table covers the value types the gateway actually carries.
func TestDataPointCarriesItsProvenance(t *testing.T) {
	for _, tc := range []struct {
		name  string
		value interface{}
	}{
		{"int", 10},
		{"string", "val"},
		{"bool", true},
		{"float", 12.5},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dp := NewDataPoint(tc.value, "SELECT 1", "dau=count(distinct uid)")
			if dp.Value != tc.value {
				t.Errorf("value: got %v (%T), want %v (%T)",
					dp.Value, dp.Value, tc.value, tc.value)
			}
			if dp.Trace.SQL != "SELECT 1" {
				t.Errorf("trace.SQL lost: got %q", dp.Trace.SQL)
			}
			if dp.Trace.MetricDef != "dau=count(distinct uid)" {
				t.Errorf("trace.MetricDef lost: got %q", dp.Trace.MetricDef)
			}
		})
	}
}
