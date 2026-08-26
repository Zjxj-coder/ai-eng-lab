package main

import (
	"testing"
)

func TestRunBench(t *testing.T) {
	out, err := RunBench()
	if err != nil {
		t.Fatalf("RunBench failed: %v", err)
	}

	if out.Codeagent.Samples != 120 {
		t.Errorf("Expected 120 samples, got %d", out.Codeagent.Samples)
	}
	if out.Codeagent.PassedRegression != 89 {
		t.Errorf("Expected 89 passed regression, got %d", out.Codeagent.PassedRegression)
	}
	if out.Codeagent.Accepted != 61 {
		t.Errorf("Expected 61 accepted, got %d", out.Codeagent.Accepted)
	}

	if out.Eval.Cases != 300 {
		t.Errorf("Expected 300 cases, got %d", out.Eval.Cases)
	}
	if out.Eval.JudgeHumanAgreement != 0.91 {
		t.Errorf("Expected 0.91 agreement, got %.2f", out.Eval.JudgeHumanAgreement)
	}

	if out.Router.FailoverP99Ms != 1800 {
		t.Errorf("Expected 1800ms p99, got %d", out.Router.FailoverP99Ms)
	}
}
