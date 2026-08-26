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

	if out.Router.Trials < 200 {
		t.Errorf("Expected at least 200 trials, got %d", out.Router.Trials)
	}
	if out.Router.FailoverP50Ms <= 0 || out.Router.FailoverP99Ms <= 0 {
		t.Errorf("Expected positive metrics, got p50=%d, p99=%d", out.Router.FailoverP50Ms, out.Router.FailoverP99Ms)
	}
	if out.Router.FailoverP99Ms < out.Router.FailoverP50Ms {
		t.Errorf("Expected p99 >= p50, got p50=%d, p99=%d", out.Router.FailoverP50Ms, out.Router.FailoverP99Ms)
	}
	
	// E.g. expected around ~2200-2400 ms based on math
	if out.Router.FailoverP99Ms < 1000 || out.Router.FailoverP99Ms > 4000 {
		t.Errorf("Expected p99 in reasonable range (1000-4000), got %d", out.Router.FailoverP99Ms)
	}
}
