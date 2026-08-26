package router

import (
	"context"
	"testing"
	"time"
)

type mockClock struct {
	now time.Time
}

func (m *mockClock) Now() time.Time {
	return m.now
}

func TestScore(t *testing.T) {
	tests := []struct {
		name      string
		candidate Candidate
		expected  float64
	}{
		{"normal", Candidate{CostPer1K: 0.01, P50Millis: 500, Weight: 1.0}, 5.1},
		{"zero_weight", Candidate{CostPer1K: 0.01, P50Millis: 500, Weight: 0}, 510}, // divided by 0.01
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := Score("", tt.candidate)
			if s < tt.expected-1e-6 || s > tt.expected+1e-6 {
				t.Errorf("Score = %v, want %v", s, tt.expected)
			}
		})
	}
}

func TestRouter_Route(t *testing.T) {
	clk := &mockClock{now: time.Now()}
	r := NewRouter(clk)

	candidates := []Candidate{
		{Name: "c1", Provider: "p1", CostPer1K: 0.02, P50Millis: 200, Weight: 1.0},
		{Name: "c2", Provider: "p2", CostPer1K: 0.01, P50Millis: 500, Weight: 1.0},
	}

	ctx := context.Background()

	// Both should be available. c1 score: 0.2 + 2 = 2.2. c2 score: 0.1 + 5 = 5.1
	c, err := r.Route(ctx, "test", candidates)
	if err != nil || c.Name != "c1" {
		t.Fatalf("expected c1, got %v, err %v", c.Name, err)
	}

	// Fail c1 enough times to trip breaker
	for i := 0; i < 5; i++ {
		r.ReportFailure("c1")
	}

	// Now it should route to c2, demonstrating fallback across providers
	c, err = r.Route(ctx, "test", candidates)
	if err != nil || c.Name != "c2" {
		t.Fatalf("expected c2 after c1 melted down, got %v, err %v", c.Name, err)
	}

	// Fail c2 as well
	for i := 0; i < 5; i++ {
		r.ReportFailure("c2")
	}

	_, err = r.Route(ctx, "test", candidates)
	if err == nil {
		t.Fatalf("expected error when all melted down")
	}

	// Fast forward time to cooldown
	clk.now = clk.now.Add(6 * time.Second)

	// Half open, should allow one request for c1 (best score)
	c, err = r.Route(ctx, "test", candidates)
	if err != nil || c.Name != "c1" {
		t.Fatalf("expected c1 in half open, got %v", c.Name)
	}

	// Success recovers c1
	r.ReportSuccess("c1")
	c, err = r.Route(ctx, "test", candidates)
	if err != nil || c.Name != "c1" {
		t.Fatalf("expected c1 after recovery, got %v", c.Name)
	}
}
