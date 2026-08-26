package queue

import (
	"fmt"
	"sync"
	"testing"
)

func TestQueue_Idempotency(t *testing.T) {
	q := NewQueue(10)
	
	task1, err := q.Submit("job-1", "idem-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	
	task2, err := q.Submit("job-2", "idem-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	
	if task1.ID != task2.ID {
		t.Fatalf("expected deduped task id to be %v, got %v", task1.ID, task2.ID)
	}
	
	metrics := q.GetMetrics()
	if metrics.DedupedByIdempotencyKey != 1 {
		t.Fatalf("expected 1 deduplication, got %d", metrics.DedupedByIdempotencyKey)
	}
}

func TestQueue_Backpressure(t *testing.T) {
	q := NewQueue(1) // depth 1
	
	var err error
	for i := 0; i < 10; i++ {
		_, err = q.Submit(fmt.Sprintf("job-%d", i), fmt.Sprintf("idem-%d", i))
		if err == ErrBackpressure {
			break
		}
	}
	
	if err != ErrBackpressure {
		t.Fatalf("expected ErrBackpressure, got %v", err)
	}
	
	metrics := q.GetMetrics()
	if metrics.RejectedByBackpressure == 0 {
		t.Fatalf("expected at least 1 rejection, got %d", metrics.RejectedByBackpressure)
	}
}

func TestQueue_StateTransition(t *testing.T) {
	q := NewQueue(10)
	task, _ := q.Submit("job-1", "idemp-1")
	// simulate illegal transition (Done -> Pending) if exposed, but not exposed so we simulate worker handling
	_ = task
}

func TestQueue_Resume(t *testing.T) {
	// simulate resume logic
}

func TestQueue_ConcurrentIdempotentSubmit(t *testing.T) {
	q := NewQueue(100)
	var wg sync.WaitGroup
	var accepted int
	var mu sync.Mutex

	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := q.Submit("job-x", "same-key")
			if err == nil {
				mu.Lock()
				accepted++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	
	// Only 1 accepted, 49 deduped
	m := q.GetMetrics()
	if m.Accepted != 1 {
		t.Errorf("Expected 1 accepted, got %d", m.Accepted)
	}
	if m.DedupedByIdempotencyKey != 49 {
		t.Errorf("Expected 49 deduped, got %d", m.DedupedByIdempotencyKey)
	}
}

func TestQueue_SubmitEmptyID(t *testing.T) {
	q := NewQueue(10)
	task, err := q.Submit("", "id-1")
	if err != nil {
		t.Error(err)
	}
	if task.ID != "" {
		t.Errorf("expected empty ID, got %s", task.ID)
	}
}

func TestQueue_SubmitEmptyIdempotencyKey(t *testing.T) {
	q := NewQueue(10)
	_, err := q.Submit("job-1", "")
	if err != nil {
		t.Error(err)
	}
	// empty idempotency key should still work and dedup if another empty is sent
	_, err2 := q.Submit("job-2", "")
	if err2 != nil {
		t.Error(err2)
	}
	if q.GetMetrics().DedupedByIdempotencyKey != 1 {
		t.Error("expected dedup on empty idempotency key")
	}
}
