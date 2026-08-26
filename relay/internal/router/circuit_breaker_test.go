package router

import (
	"testing"
	"time"
)

func TestCircuitBreaker_StateTransitions(t *testing.T) {
	clk := &mockClock{now: time.Now()}
	cb := NewCircuitBreaker(clk)

	// Initial state
	if !cb.Allow() {
		t.Fatal("expected allow")
	}

	// Trip breaker
	for i := 0; i < 5; i++ {
		cb.Record(false)
	}
	if cb.Allow() {
		t.Fatal("expected reject")
	}

	// Wait cooldown
	clk.now = clk.now.Add(6 * time.Second)

	// Half-open
	if !cb.Allow() {
		t.Fatal("expected allow half open")
	}
	// Second request should be rejected while half open
	if cb.Allow() {
		t.Fatal("expected reject while half open in flight")
	}

	// Success => closed
	cb.Record(true)
	if !cb.Allow() {
		t.Fatal("expected allow closed")
	}
}

func TestCircuitBreaker_SlidingWindow(t *testing.T) {
	clk := &mockClock{now: time.Now()}
	cb := NewCircuitBreaker(clk)

	// 4 failures, shouldn't trip
	for i := 0; i < 4; i++ {
		cb.Record(false)
	}
	if !cb.Allow() {
		t.Fatal("expected allow")
	}

	// advance time past window
	clk.now = clk.now.Add(11 * time.Second)

	// 4 more failures
	for i := 0; i < 4; i++ {
		cb.Record(false)
	}
	if !cb.Allow() {
		t.Fatal("expected allow, old failures should have been cleaned up")
	}
}

func TestCircuitBreaker_WindowBoundary(t *testing.T) {
	clk := &mockClock{now: time.Now()}
	cb := NewCircuitBreaker(clk)

	// Record 4 failures at start of window
	for i := 0; i < 4; i++ {
		cb.Record(false)
	}

	// Advance time by exactly window (10s)
	clk.now = clk.now.Add(10 * time.Second)

	// One more failure. Since window just rolled over, the first 4 should be cleaned up.
	cb.Record(false)
	
	// Total recorded in new window is 1, less than minRequests(5)
	if !cb.Allow() {
		t.Fatal("expected allow, old failures should be outside window boundary")
	}
}

func TestCircuitBreaker_HalfOpenConcurrency(t *testing.T) {
	clk := &mockClock{now: time.Now()}
	cb := NewCircuitBreaker(clk)

	for i := 0; i < 5; i++ {
		cb.Record(false)
	}
	if cb.Allow() {
		t.Fatal("expected reject")
	}

	clk.now = clk.now.Add(6 * time.Second)

	// Half-open allowed once
	if !cb.Allow() {
		t.Fatal("expected allow half open")
	}
	
	// Concurrent requests should all fail
	for i := 0; i < 10; i++ {
		if cb.Allow() {
			t.Fatal("expected reject while half open in flight")
		}
	}
}
