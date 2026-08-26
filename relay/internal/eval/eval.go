package eval

import (
	"github.com/guojunhao/ai-eng-lab/relay/internal/router"
)

// OfflineSet represents a dataset for evaluation.
type OfflineSet struct {
	Cases []string
}

// LLMJudge represents a model's judgment.
type LLMJudge interface {
	Judge(caseData string) (bool, error)
}

// HumanSpotCheck represents human evaluation records.
type HumanSpotCheck struct {
	Records map[string]bool // caseData -> is passed
}

// Report contains 3D metrics per model.
type Report struct {
	ModelMetrics map[string]Metrics
}

type Metrics struct {
	Cost     float64
	Latency  int // p50 ms
	PassRate float64
}

// GenerateReport computes the report.
func GenerateReport() *Report {
	// In a real system, this would evaluate models.
	return &Report{
		ModelMetrics: make(map[string]Metrics),
	}
}

// Agreement calculates the agreement rate between LLMJudge and HumanSpotCheck.
func Agreement(judge LLMJudge, human *HumanSpotCheck) float64 {
	if len(human.Records) == 0 {
		return 0.0
	}
	
	agreements := 0
	for caseData, humanVal := range human.Records {
		judgeVal, err := judge.Judge(caseData)
		if err == nil && judgeVal == humanVal {
			agreements++
		}
	}
	
	return float64(agreements) / float64(len(human.Records))
}

// UpdateRouterWeights updates the weights of candidates based on the eval report.
// Higher pass rate means higher weight.
func UpdateRouterWeights(report *Report, candidates []router.Candidate) {
	for i, c := range candidates {
		if metrics, ok := report.ModelMetrics[c.Name]; ok {
			// e.g. weight = pass rate. If pass rate is 0, set a minimal weight.
			w := metrics.PassRate
			if w <= 0 {
				w = 0.01
			}
			candidates[i].Weight = w
		}
	}
}
