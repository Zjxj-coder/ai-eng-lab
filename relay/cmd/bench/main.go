package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/guojunhao/ai-eng-lab/relay/internal/router"
)

type codeagentSample struct {
	ID               int  `json:"id"`
	PassedRegression bool `json:"passed_regression"`
	Accepted         bool `json:"accepted"`
}

type evalCase struct {
	ID    int  `json:"id"`
	Judge bool `json:"judge"`
	Human bool `json:"human"`
}

type Output struct {
	Codeagent struct {
		Samples          int `json:"samples"`
		PassedRegression int `json:"passed_regression"`
		Accepted         int `json:"accepted"`
	} `json:"codeagent"`
	Eval struct {
		Cases               int     `json:"cases"`
		JudgeHumanAgreement float64 `json:"judge_human_agreement"`
	} `json:"eval"`
	Router struct {
		Trials        int `json:"trials"`
		FailoverP50Ms int `json:"failover_p50_ms"`
		FailoverP99Ms int `json:"failover_p99_ms"`
	} `json:"router"`
}

type mockClock struct {
	now time.Time
}

func (m *mockClock) Now() time.Time {
	return m.now
}

func (m *mockClock) advance(d time.Duration) {
	m.now = m.now.Add(d)
}

func main() {
	jsonOut := flag.Bool("json", false, "output json")
	flag.Parse()

	out, err := RunBench()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if *jsonOut {
		b, _ := json.Marshal(out)
		fmt.Println(string(b))
	} else {
		fmt.Printf("codeagent: samples %d, passed regression %d, accepted %d\n", out.Codeagent.Samples, out.Codeagent.PassedRegression, out.Codeagent.Accepted)
		fmt.Printf("eval: cases %d, judge human agreement %.2f\n", out.Eval.Cases, out.Eval.JudgeHumanAgreement)
		fmt.Printf("router: failover p50 ms %d, failover p99 ms %d (from %d trials)\n", out.Router.FailoverP50Ms, out.Router.FailoverP99Ms, out.Router.Trials)
	}
}

func RunBench() (Output, error) {
	var out Output

	// 1. Codeagent
	caData, err := os.ReadFile(filepath.Join("testdata", "codeagent", "samples.json"))
	if err == nil {
		var samples []codeagentSample
		json.Unmarshal(caData, &samples)
		out.Codeagent.Samples = len(samples)
		for _, s := range samples {
			if s.PassedRegression {
				out.Codeagent.PassedRegression++
			}
			if s.Accepted {
				out.Codeagent.Accepted++
			}
		}
	} else {
		// Fallback for tests if running from wrong directory
		caData, err = os.ReadFile(filepath.Join("..", "..", "testdata", "codeagent", "samples.json"))
		if err == nil {
			var samples []codeagentSample
			json.Unmarshal(caData, &samples)
			out.Codeagent.Samples = len(samples)
			for _, s := range samples {
				if s.PassedRegression {
					out.Codeagent.PassedRegression++
				}
				if s.Accepted {
					out.Codeagent.Accepted++
				}
			}
		} else {
            return out, fmt.Errorf("failed to read codeagent samples: %v", err)
        }
	}

	// 2. Eval
	evalData, err := os.ReadFile(filepath.Join("testdata", "eval", "cases.json"))
	if err == nil {
		var cases []evalCase
		json.Unmarshal(evalData, &cases)
		out.Eval.Cases = len(cases)
		var agree int
		for _, c := range cases {
			if c.Judge == c.Human {
				agree++
			}
		}
		if out.Eval.Cases > 0 {
			out.Eval.JudgeHumanAgreement = float64(agree) / float64(out.Eval.Cases)
		}
	} else {
		evalData, err = os.ReadFile(filepath.Join("..", "..", "testdata", "eval", "cases.json"))
		if err == nil {
			var cases []evalCase
			json.Unmarshal(evalData, &cases)
			out.Eval.Cases = len(cases)
			var agree int
			for _, c := range cases {
				if c.Judge == c.Human {
					agree++
				}
			}
			if out.Eval.Cases > 0 {
				out.Eval.JudgeHumanAgreement = float64(agree) / float64(out.Eval.Cases)
			}
		} else {
            return out, fmt.Errorf("failed to read eval cases: %v", err)
        }
	}

	// 3. Router Simulation
	out.Router.Trials, out.Router.FailoverP50Ms, out.Router.FailoverP99Ms = simRouter()

	return out, nil
}

func simRouter() (int, int, int) {
	var recoveryTimes []int
	ctx := context.Background()
	
	rnd := rand.New(rand.NewSource(42))
	trials := 200

	for i := 0; i < trials; i++ {
		clk := &mockClock{now: time.Now()}
		r := router.NewRouter(clk)
		candidates := []router.Candidate{
			{Name: "c1", Provider: "p1", Weight: 1},
			{Name: "c2", Provider: "p2", Weight: 1},
		}

		candidates[0].CostPer1K = 0.01
		candidates[1].CostPer1K = 0.02

		start := clk.Now()
		
		for {
			c, err := r.Route(ctx, "test", candidates)
			if err != nil {
				panic(fmt.Sprintf("route error: %v", err))
			}
			
			if c.Name == "c1" {
				// c1 is broken. Simulate timeout.
				reqTime := time.Duration(200 + rnd.Intn(300)) * time.Millisecond
				clk.advance(reqTime)
				r.ReportFailure(c.Name)
				
				// Add a small backoff delay before retry
				backoff := time.Duration(50 + rnd.Intn(50)) * time.Millisecond
				clk.advance(backoff)
			} else if c.Name == "c2" {
				// c2 is healthy, request succeeds.
				reqTime := time.Duration(100 + rnd.Intn(200)) * time.Millisecond
				clk.advance(reqTime)
				r.ReportSuccess(c.Name)
				break // Recovered!
			} else {
				panic("unexpected candidate")
			}
		}

		recoveryTimes = append(recoveryTimes, int(clk.Now().Sub(start).Milliseconds()))
	}

	sort.Ints(recoveryTimes)
	
	p50Idx := int(float64(len(recoveryTimes)) * 0.50)
	if p50Idx >= len(recoveryTimes) { p50Idx = len(recoveryTimes) - 1 }
	p99Idx := int(float64(len(recoveryTimes)) * 0.99)
	if p99Idx >= len(recoveryTimes) { p99Idx = len(recoveryTimes) - 1 }
	
	return trials, recoveryTimes[p50Idx], recoveryTimes[p99Idx]
}
