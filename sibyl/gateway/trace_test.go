package main

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestNewDataPoint(t *testing.T) {
	dp := NewDataPoint(100, "SELECT 1", "def")
	if dp.Value != 100 {
		t.Errorf("expected 100, got %v", dp.Value)
	}
	if dp.Trace.SQL != "SELECT 1" {
		t.Errorf("expected SELECT 1, got %v", dp.Trace.SQL)
	}
}

func TestTraceJSON(t *testing.T) {
	dp := NewDataPoint(100, "SELECT 1", "def")
	b, err := json.Marshal(&dp)
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	if !contains(s, `"value":100`) {
		t.Errorf("missing value in json: %s", s)
	}
	if !contains(s, `"trace":`) {
		t.Errorf("missing trace in json: %s", s)
	}
	if !contains(s, `"sql":"SELECT 1"`) {
		t.Errorf("missing sql in json: %s", s)
	}
}

func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}
