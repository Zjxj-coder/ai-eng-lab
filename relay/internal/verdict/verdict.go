package verdict

import (
	"fmt"
	"os"
	"strings"
)

// Verdict is the result of a judgment.
type Verdict string

const (
	Accepted Verdict = "Accepted"
	Rejected Verdict = "Rejected"
)

// Reason provides explanation for rejection.
type Reason struct {
	Message string
}

// Usage represents API usage information.
type Usage struct {
	InputTokens  int
	OutputTokens int
	APICalls     int
	Completed    bool
	Failed       bool
}

// Evidence represents the output of a task.
type Evidence struct {
	ExitCode int
	Stdout   string
	Stderr   string
	Usage    *Usage
	Artifact string
}

// OSStat defines an interface for os.Stat for testability.
type OSStat func(name string) (os.FileInfo, error)

var statFunc OSStat = os.Stat

// Judge evaluates the evidence and returns a verdict and reasons if rejected.
func Judge(ev Evidence) (Verdict, []Reason) {
	var reasons []Reason

	// 1. ExitCode==0 but Usage.Failed==true (e.g. rate limited but process exit 0)
	if ev.ExitCode == 0 && ev.Usage != nil && ev.Usage.Failed {
		reasons = append(reasons, Reason{Message: "ExitCode is 0 but Usage.Failed is true"})
	}

	// 2. Stdout has error info
	lowerOut := strings.ToLower(ev.Stdout)
	if strings.Contains(lowerOut, "error") || strings.Contains(lowerOut, "exception") || strings.Contains(lowerOut, "panic") {
		reasons = append(reasons, Reason{Message: "Stdout contains error information"})
	}

	// 3. Usage.Completed==true but Artifact doesn't exist or size 0
	if ev.Usage != nil && ev.Usage.Completed {
		if ev.Artifact == "" {
			reasons = append(reasons, Reason{Message: "Usage completed but no artifact specified"})
		} else {
			info, err := statFunc(ev.Artifact)
			if err != nil || info.Size() == 0 {
				reasons = append(reasons, Reason{Message: fmt.Sprintf("Artifact %s does not exist or is empty", ev.Artifact)})
			}
		}
	}

	if ev.ExitCode != 0 && (ev.Usage == nil || !ev.Usage.Failed) { // default non-zero exit code
		reasons = append(reasons, Reason{Message: fmt.Sprintf("Non-zero exit code: %d", ev.ExitCode)})
	}

	if len(reasons) > 0 {
		return Rejected, reasons
	}

	return Accepted, nil
}
