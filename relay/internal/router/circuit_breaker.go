package router

import (
	"sync"
	"time"
)

// State represents the circuit breaker state.
type State int

const (
	// StateClosed allows requests.
	StateClosed State = iota
	// StateOpen blocks requests.
	StateOpen
	// StateHalfOpen allows a single test request.
	StateHalfOpen
)

type bucket struct {
	success int
	failure int
}

// CircuitBreaker is a sliding window circuit breaker.
type CircuitBreaker struct {
	mu               sync.Mutex
	clock            Clock
	window           time.Duration
	bucketSize       time.Duration
	buckets          map[int64]*bucket
	threshold        float64
	minRequests      int
	cooldown         time.Duration
	state            State
	lastOpenTime     time.Time
	halfOpenInFlight bool
}

// NewCircuitBreaker creates a circuit breaker.
func NewCircuitBreaker(clock Clock) *CircuitBreaker {
	return &CircuitBreaker{
		clock:       clock,
		window:      10 * time.Second,
		bucketSize:  1 * time.Second,
		buckets:     make(map[int64]*bucket),
		threshold:   0.5,
		minRequests: 5,
		cooldown:    5 * time.Second,
		state:       StateClosed,
	}
}

func (cb *CircuitBreaker) cleanup(now time.Time) {
	windowStart := now.Add(-cb.window).UnixNano() / cb.bucketSize.Nanoseconds()
	for k := range cb.buckets {
		if k <= windowStart {
			delete(cb.buckets, k)
		}
	}
}

// Allow checks if a request is allowed.
func (cb *CircuitBreaker) Allow() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	now := cb.clock.Now()

	switch cb.state {
	case StateClosed:
		return true
	case StateOpen:
		if now.Sub(cb.lastOpenTime) >= cb.cooldown {
			cb.state = StateHalfOpen
			cb.halfOpenInFlight = true
			return true
		}
		return false
	case StateHalfOpen:
		if !cb.halfOpenInFlight {
			cb.halfOpenInFlight = true
			return true
		}
		return false
	}
	return false
}

// Record records a request outcome.
func (cb *CircuitBreaker) Record(success bool) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	now := cb.clock.Now()

	if cb.state == StateHalfOpen {
		cb.halfOpenInFlight = false
		if success {
			cb.state = StateClosed
			cb.buckets = make(map[int64]*bucket) // Reset window
		} else {
			cb.state = StateOpen
			cb.lastOpenTime = now
		}
		return
	}

	if cb.state == StateOpen {
		return // Ignore records when open
	}

	cb.cleanup(now)
	idx := now.UnixNano() / cb.bucketSize.Nanoseconds()
	b, ok := cb.buckets[idx]
	if !ok {
		b = &bucket{}
		cb.buckets[idx] = b
	}

	if success {
		b.success++
	} else {
		b.failure++
	}

	// Check if we need to trip the breaker
	var totalSuccess, totalFailure int
	for _, bk := range cb.buckets {
		totalSuccess += bk.success
		totalFailure += bk.failure
	}

	total := totalSuccess + totalFailure
	if total >= cb.minRequests {
		rate := float64(totalFailure) / float64(total)
		if rate >= cb.threshold {
			cb.state = StateOpen
			cb.lastOpenTime = now
		}
	}
}
