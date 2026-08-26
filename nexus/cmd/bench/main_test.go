package main

import (
	"math"
	"testing"
)

func TestCalcTradeoffCurve(t *testing.T) {
	// 构造一个已知的小切片（手工给定 bestSim / correct）
	// corpusSize = 10
	// 阈值测试：
	// Threshold = 0.8:
	// - >= 0.8 的有：rec1 (0.9, correct), rec2 (0.85, correct), rec3 (0.81, incorrect), rec4 (0.8, correct) -> 4 个 semantic hits, 1 错误
	//   semantic_hit_rate = 4/10 = 0.4
	//   false_hit_rate = 1/4 = 0.25 (假设没有 prefixHit)
	// Threshold = 0.85:
	// - >= 0.85 的有：rec1 (0.9, correct), rec2 (0.85, correct) -> 2 个 semantic hits, 0 错误
	//   semantic_hit_rate = 2/10 = 0.2
	//   false_hit_rate = 0/2 = 0
	records := []QueryRecord{
		{BestSim: 0.90, Correct: true, PrefixHit: false},
		{BestSim: 0.85, Correct: true, PrefixHit: false},
		{BestSim: 0.81, Correct: false, PrefixHit: false},
		{BestSim: 0.80, Correct: true, PrefixHit: false},
		{BestSim: 0.70, Correct: false, PrefixHit: false},
		{BestSim: 0.50, Correct: true, PrefixHit: false},
		{BestSim: 0.40, Correct: false, PrefixHit: false},
		{BestSim: 0.30, Correct: true, PrefixHit: false},
		{BestSim: 0.20, Correct: true, PrefixHit: false},
		{BestSim: 0.10, Correct: false, PrefixHit: false},
	}

	thresholds := []float64{0.75, 0.8, 0.85, 0.9}
	corpusSize := 10

	curve := CalcTradeoffCurve(records, thresholds, corpusSize)

	// 验证在阈值 0.8 下的值
	var p08 TradeoffPoint
	var p085 TradeoffPoint
	for _, pt := range curve {
		if pt.Threshold == 0.8 {
			p08 = pt
		} else if pt.Threshold == 0.85 {
			p085 = pt
		}
	}

	if math.Abs(p08.SemanticHitRate-0.4) > 1e-6 {
		t.Errorf("Expected semantic_hit_rate 0.4 at threshold 0.8, got %v", p08.SemanticHitRate)
	}
	if math.Abs(p08.FalseHitRate-0.25) > 1e-6 {
		t.Errorf("Expected false_hit_rate 0.25 at threshold 0.8, got %v", p08.FalseHitRate)
	}

	if math.Abs(p085.SemanticHitRate-0.2) > 1e-6 {
		t.Errorf("Expected semantic_hit_rate 0.2 at threshold 0.85, got %v", p085.SemanticHitRate)
	}
	if math.Abs(p085.FalseHitRate-0.0) > 1e-6 {
		t.Errorf("Expected false_hit_rate 0 at threshold 0.85, got %v", p085.FalseHitRate)
	}

	// 断言曲线上任意两点：阈值更高 ⇒ false_hit_rate 不增（单调性）
	for i := 0; i < len(curve)-1; i++ {
		if curve[i+1].Threshold > curve[i].Threshold {
			if curve[i+1].FalseHitRate > curve[i].FalseHitRate {
				t.Errorf("Monotonicity broken: threshold %v has FHR %v, higher threshold %v has FHR %v",
					curve[i].Threshold, curve[i].FalseHitRate, curve[i+1].Threshold, curve[i+1].FalseHitRate)
			}
		}
	}
}
