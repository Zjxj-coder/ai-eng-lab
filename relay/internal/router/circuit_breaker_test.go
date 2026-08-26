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
