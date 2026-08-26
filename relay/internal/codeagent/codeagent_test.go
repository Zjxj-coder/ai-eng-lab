package codeagent

import (
	"errors"
	"testing"
)

type mockRepo struct {
	failTests bool
}

func (m *mockRepo) ApplyPatch(patch string) error {
	return nil
}

func (m *mockRepo) RunTests() error {
	if m.failTests {
		return errors.New("tests failed")
	}
	return nil
}

func TestApplyAndGate(t *testing.T) {
	tests := []struct {
		name      string
		patch     string
		repo      Repo
		want      Result
	}{
		{
			name: "valid patch",
			patch: `--- a/main.go
+++ b/main.go
@@ -1,2 +1,3 @@
 package main
+import "fmt"`,
			repo: &mockRepo{failTests: false},
			want: PassedRegression,
		},
		{
			name: "patch touches test file (_test.go)",
			patch: `--- a/main_test.go
+++ b/main_test.go
@@ -1,3 +1,2 @@
 func TestFoo(t *testing.T) {
-    assert.True(t, false)
 }`,
			repo: &mockRepo{failTests: false},
			want: Rejected_TouchedTests,
		},
		{
			name: "patch touches tests directory",
			patch: `--- a/tests/integration.go
+++ b/tests/integration.go
@@ -1,2 +1,3 @@
`,
			repo: &mockRepo{failTests: false},
			want: Rejected_TouchedTests,
		},
		{
			name: "tests fail after patch",
			patch: `--- a/main.go
+++ b/main.go
@@ -1,2 +1,3 @@
`,
			repo: &mockRepo{failTests: true},
			want: Rejected_TestsFailed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ApplyAndGate(tt.repo, tt.patch)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

// 必须有一个单测专门喂一个"删断言"的补丁并断言它被拒
func TestApplyAndGate_RejectDeleteAssertion(t *testing.T) {
	patch := `--- a/internal/eval/eval_test.go
+++ b/internal/eval/eval_test.go
@@ -10,7 +10,6 @@
 func TestAgreement(t *testing.T) {
 	res := Agreement(judge, human)
-	assert.Equal(t, 1.0, res) // Delete assertion to cheat
 }
`
	repo := &mockRepo{failTests: false}
	
	res, err := ApplyAndGate(repo, patch)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res != Rejected_TouchedTests {
		t.Errorf("expected Rejected_TouchedTests, got %v", res)
	}
}
