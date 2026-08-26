package queue

import (
	"errors"
	"sync"
	"time"
)

type TaskState int

const (
	Pending TaskState = iota
	Running
	Done
	Failed
)

var ErrBackpressure = errors.New("queue is full, backpressure applied")

type Task struct {
	ID             string
	IdempotencyKey string
	State          TaskState
	Result         string
	Error          error
}

type Queue struct {
	mu           sync.Mutex
	tasks        map[string]*Task
	idempotency  map[string]string // IdempotencyKey -> TaskID
	pendingQueue chan *Task
	maxDepth     int
	Metrics      QueueMetrics
}

type QueueMetrics struct {
	TotalAttempts            int
	Accepted                 int
	DedupedByIdempotencyKey  int
	RejectedByBackpressure   int
}

func NewQueue(maxDepth int) *Queue {
	q := &Queue{
		tasks:        make(map[string]*Task),
		idempotency:  make(map[string]string),
		pendingQueue: make(chan *Task, maxDepth),
		maxDepth:     maxDepth,
	}
	go q.worker() // Single serial consumer
	return q
}

func (q *Queue) Submit(id string, idempotencyKey string) (*Task, error) {
	q.mu.Lock()
	defer q.mu.Unlock()

	q.Metrics.TotalAttempts++

	if existingID, ok := q.idempotency[idempotencyKey]; ok {
		q.Metrics.DedupedByIdempotencyKey++
		return q.tasks[existingID], nil
	}

	if len(q.pendingQueue) >= q.maxDepth {
		q.Metrics.RejectedByBackpressure++
		return nil, ErrBackpressure
	}

	t := &Task{
		ID:             id,
		IdempotencyKey: idempotencyKey,
		State:          Pending,
	}

	q.tasks[id] = t
	q.idempotency[idempotencyKey] = id
	q.pendingQueue <- t
	q.Metrics.Accepted++
	
	return t, nil
}

func (q *Queue) Get(id string) *Task {
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.tasks[id]
}

func (q *Queue) GetMetrics() QueueMetrics {
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.Metrics
}

func (q *Queue) worker() {
	for t := range q.pendingQueue {
		q.mu.Lock()
		t.State = Running
		q.mu.Unlock()

		// Simulate image generation latency
		time.Sleep(5 * time.Millisecond)

		q.mu.Lock()
		t.State = Done
		t.Result = "image_url_placeholder"
		q.mu.Unlock()
	}
}
