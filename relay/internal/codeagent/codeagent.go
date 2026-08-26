package codeagent

import (
	"strings"
)

// Result is the result of applying a patch.
type Result string

const (
	PassedRegression      Result = "PassedRegression"
	Rejected_TouchedTests Result = "Rejected_TouchedTests"
	Rejected_TestsFailed  Result = "Rejected_TestsFailed"
)

// Repo interface abstracts repository interactions.
type Repo interface {
	ApplyPatch(patch string) error
	RunTests() error
}

// ApplyAndGate applies a patch and gates it based on rules.
func ApplyAndGate(repo Repo, patch string) (Result, error) {
	// Rule 1: Check if patch touches test files
	lines := strings.Split(patch, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "+++ ") || strings.HasPrefix(line, "--- ") {
			path := strings.TrimPrefix(line[4:], "b/")
			path = strings.TrimPrefix(path, "a/")
			path = strings.TrimSpace(path)
			lowerPath := strings.ToLower(path)
			if strings.HasSuffix(lowerPath, "_test.go") || strings.Contains(lowerPath, "/test/") || strings.Contains(lowerPath, "/tests/") || strings.HasPrefix(lowerPath, "test/") || strings.HasPrefix(lowerPath, "tests/") {
				return Rejected_TouchedTests, nil
			}
		}
	}

	// Apply patch
	if err := repo.ApplyPatch(patch); err != nil {
		return Rejected_TestsFailed, err // treating patch apply failure as test failed
	}

	// Rule 2: Run tests
	if err := repo.RunTests(); err != nil {
		return Rejected_TestsFailed, nil
	}

	return PassedRegression, nil
}
