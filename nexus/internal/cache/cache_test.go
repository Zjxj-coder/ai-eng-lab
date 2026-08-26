package cache

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestCache_PrefixBoundary(t *testing.T) {
	c := NewCache(1 * time.Hour)
	now := time.Now()

	prompt1 := "Translate the following english text to french: Hello world, how are you today?"

	c.SetPrefix(prompt1, "Response 1", now)

	// prompt1 and prompt2 share the first 50 chars: "Translate the following english text to french: " (which is 48 chars).
	// Let's make sure they share exactly 50 chars.
	// "Translate the following english text to french: AB" is 50 chars.
	
	p1 := "Translate the following english text to french: AB Hello world"
	p2 := "Translate the following english text to french: AB Good morning"
	
	c.SetPrefix(p1, "Resp1", now)
	
	resp, ok := c.GetPrefix(p2, now)
	if !ok {
		t.Fatalf("Expected prefix cache hit, since they share first 50 chars")
	}
	if resp != "Resp1" {
		t.Fatalf("Expected Resp1, got %v", resp)
	}
	
	p3 := "Translate the following english text to french: XY something"
	_, ok2 := c.GetPrefix(p3, now)
	if ok2 {
		t.Fatalf("Expected miss because prefix is different")
	}
}

func TestCache_SemanticExpiration(t *testing.T) {
	window := 24 * time.Hour
	c := NewCache(window)

	prompt := "What is the capital of France?"
	response := "Paris"

	yesterday := time.Now().Add(-25 * time.Hour)
	c.SetSemantic(prompt, response, yesterday)

	today := time.Now()
	// Ask the same question today
	_, hit := c.GetSemantic(prompt, 0.9, today)
	if hit {
		t.Fatalf("yesterday's answer should not hit today's question")
	}
	
	// Test hit within window
	c.SetSemantic(prompt, response, today)
	resp, hit := c.GetSemantic("What is the capital of France?", 0.9, today)
	if !hit || resp != "Paris" {
		t.Fatalf("should hit within window")
	}
}

func TestCache_PrefixExpiration(t *testing.T) {
	window := 24 * time.Hour
	c := NewCache(window)

	prompt := "Translate: apple"
	response := "pomme"

	yesterday := time.Now().Add(-25 * time.Hour)
	c.SetPrefix(prompt, response, yesterday)

	today := time.Now()
	// Ask the same question today
	_, hit := c.GetPrefix(prompt, today)
	if hit {
		t.Fatalf("yesterday's answer should not hit today's question for prefix cache")
	}
}

func TestCache_ConcurrentReadWrite(t *testing.T) {
	c := NewCache(1 * time.Hour)
	now := time.Now()
	var wg sync.WaitGroup

	// Writes
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			c.SetSemantic(fmt.Sprintf("Concurrent prompt %d", idx), "resp", now)
		}(i)
	}

	// Reads
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			c.GetSemantic(fmt.Sprintf("Concurrent prompt %d", idx), 0.9, now)
		}(i)
	}
	wg.Wait()
}

func TestCache_SemanticThresholdBoundary(t *testing.T) {
	c := NewCache(1 * time.Hour)
	now := time.Now()
	c.SetSemantic("This is a test prompt about something specific", "response", now)

	// Threshold 0.99 should fail for a slightly different prompt
	_, ok1 := c.GetSemantic("This is a test prompt about something entirely different", 0.99, now)
	if ok1 {
		t.Error("expected false for strict threshold")
	}

	// Threshold 0.1 should pass
	_, ok2 := c.GetSemantic("This is a test prompt about something entirely different", 0.10, now)
	if !ok2 {
		t.Error("expected true for loose threshold")
	}
}

func TestCache_SemanticNegativeCase(t *testing.T) {
	c := NewCache(1 * time.Hour)
	now := time.Now()
	// Two semantically similar requests with different answers
	req1 := "Can you please tell me the capital of France?"
	req2 := "Can you please tell me the capital of Germany?"

	c.SetSemantic(req1, "Paris", now)

	// The threshold should be tight enough to NOT hit req1 when req2 is queried.
	resp, ok := c.GetSemantic(req2, 0.99, now) // Note: using 0.99 threshold here!
	if ok {
		t.Errorf("Negative case failed! Request 2 hit Request 1's cache and got: %s", resp)
	}
}

