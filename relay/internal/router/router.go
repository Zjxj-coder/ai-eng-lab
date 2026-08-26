package router

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"
)

// Clock defines a time source for testability.
type Clock interface {
	Now() time.Time
}

type realClock struct{}

func (realClock) Now() time.Time { return time.Now() }

// Candidate represents a model routing candidate.
type Candidate struct {
	Name      string
	Provider  string
	CostPer1K float64
	P50Millis int
	Weight    float64
}

// TaskKind represents the kind of task to route.
type TaskKind string

// Router manages routing and circuit breakers for candidates.
type Router struct {
	mu       sync.RWMutex
	clock    Clock
	breakers map[string]*CircuitBreaker
}

// NewRouter creates a new router.
func NewRouter(clock Clock) *Router {
	if clock == nil {
		clock = realClock{}
	}
	return &Router{
		clock:    clock,
		breakers: make(map[string]*CircuitBreaker),
	}
}

func (r *Router) getBreaker(name string) *CircuitBreaker {
	r.mu.Lock()
	defer r.mu.Unlock()
	if b, ok := r.breakers[name]; ok {
		return b
	}
	b := NewCircuitBreaker(r.clock)
	r.breakers[name] = b
	return b
}

// Score calculates a score for a candidate. Lower is better.
func Score(kind TaskKind, c Candidate) float64 {
	weight := c.Weight
	if weight <= 0 {
		weight = 0.01
	}
	return (c.CostPer1K*10.0 + float64(c.P50Millis)/100.0) / weight
}

// Route selects the best candidate for the task.
func (r *Router) Route(ctx context.Context, kind TaskKind, candidates []Candidate) (Candidate, error) {
	if len(candidates) == 0 {
		return Candidate{}, fmt.Errorf("no candidates provided")
	}

	// 免费额度可能是账户级共享（官方文档未写明，按 fail-safe 假设共享），所以同厂兜底不算冗余。
	// 跨供应商降级：同一 Provider 内的候选全部熔断时，必须跳到另一个 Provider。
	// 这里通过为每个候选人单独维护熔断器，当某一 Provider 的所有候选人都熔断时，自然会跳到下一个 Provider。

	type scoredCandidate struct {
		cand  Candidate
		score float64
	}

	var available []scoredCandidate
	for _, c := range candidates {
		cb := r.getBreaker(c.Name)
		if cb.Allow() {
			available = append(available, scoredCandidate{
				cand:  c,
				score: Score(kind, c),
			})
		}
	}

	if len(available) == 0 {
		return Candidate{}, fmt.Errorf("all candidates are melted down")
	}

	sort.Slice(available, func(i, j int) bool {
		if available[i].score == available[j].score {
			return available[i].cand.Name < available[j].cand.Name
		}
		return available[i].score < available[j].score
	})

	return available[0].cand, nil
}

// ReportSuccess reports a success for a candidate.
func (r *Router) ReportSuccess(name string) {
	r.getBreaker(name).Record(true)
}

// ReportFailure reports a failure for a candidate.
func (r *Router) ReportFailure(name string) {
	r.getBreaker(name).Record(false)
}
