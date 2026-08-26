package eval

import (
	"testing"
	"github.com/guojunhao/ai-eng-lab/relay/internal/router"
)

type mockJudge struct {
	results map[string]bool
}

func (m *mockJudge) Judge(caseData string) (bool, error) {
	return m.results[caseData], nil
}

func TestAgreement(t *testing.T) {
	judge := &mockJudge{
		results: map[string]bool{
			"case1": true,
			"case2": false,
			"case3": true,
		},
	}
	
	human := &HumanSpotCheck{
		Records: map[string]bool{
			"case1": true,  // agree
			"case2": true,  // disagree
			"case3": true,  // agree
		},
	}
	
	rate := Agreement(judge, human)
	expected := 2.0 / 3.0
	if rate != expected {
		t.Errorf("expected %v, got %v", expected, rate)
	}
}

func TestAgreement_Empty(t *testing.T) {
	judge := &mockJudge{}
	human := &HumanSpotCheck{Records: map[string]bool{}}
	if rate := Agreement(judge, human); rate != 0.0 {
		t.Errorf("expected 0.0 for empty samples, got %v", rate)
	}
}

func TestAgreement_AllAgree(t *testing.T) {
	judge := &mockJudge{results: map[string]bool{"1": true, "2": false}}
	human := &HumanSpotCheck{Records: map[string]bool{"1": true, "2": false}}
	if rate := Agreement(judge, human); rate != 1.0 {
		t.Errorf("expected 1.0 for all agree, got %v", rate)
	}
}

func TestAgreement_AllDisagree(t *testing.T) {
	judge := &mockJudge{results: map[string]bool{"1": false, "2": true}}
	human := &HumanSpotCheck{Records: map[string]bool{"1": true, "2": false}}
	if rate := Agreement(judge, human); rate != 0.0 {
		t.Errorf("expected 0.0 for all disagree, got %v", rate)
	}
}

func TestUpdateRouterWeights(t *testing.T) {
	report := &Report{
		ModelMetrics: map[string]Metrics{
			"modelA": {Cost: 1.0, Latency: 100, PassRate: 0.95},
			"modelB": {Cost: 2.0, Latency: 200, PassRate: 0.60},
		},
	}
	
	candidates := []router.Candidate{
		{Name: "modelA", Weight: 0.5},
		{Name: "modelB", Weight: 0.5},
		{Name: "modelC", Weight: 0.5}, // No report data
	}
	
	UpdateRouterWeights(report, candidates)
	
	if candidates[0].Weight != 0.95 {
		t.Errorf("modelA weight should be updated to 0.95, got %v", candidates[0].Weight)
	}
	if candidates[1].Weight != 0.60 {
		t.Errorf("modelB weight should be updated to 0.60, got %v", candidates[1].Weight)
	}
	if candidates[2].Weight != 0.5 { // Unchanged
		t.Errorf("modelC weight should remain 0.5, got %v", candidates[2].Weight)
	}
}
