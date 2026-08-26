package verdict

import (
	"os"
	"strings"
	"testing"
	"time"
)

type mockFileInfo struct {
	size int64
}
func (m mockFileInfo) Name() string       { return "file" }
func (m mockFileInfo) Size() int64        { return m.size }
func (m mockFileInfo) Mode() os.FileMode  { return 0644 }
func (m mockFileInfo) ModTime() time.Time { return time.Now() }
func (m mockFileInfo) IsDir() bool        { return false }
func (m mockFileInfo) Sys() interface{}   { return nil }

func TestJudge(t *testing.T) {
	origStat := statFunc
	defer func() { statFunc = origStat }()

	statFunc = func(name string) (os.FileInfo, error) {
		if name == "missing.txt" {
			return nil, os.ErrNotExist
		}
		if name == "empty.txt" {
			return mockFileInfo{size: 0}, nil
		}
		if name == "ok.txt" {
			return mockFileInfo{size: 100}, nil
		}
		return nil, os.ErrNotExist
	}

	tests := []struct {
		name       string
		evidence   Evidence
		wantVerd   Verdict
		wantReason string
	}{
		{
			name: "ok",
			evidence: Evidence{
				ExitCode: 0,
				Stdout:   "all good",
				Usage:    &Usage{Completed: true},
				Artifact: "ok.txt",
			},
			wantVerd: Accepted,
		},
		{
			name: "exit 0 but failed",
			evidence: Evidence{
				ExitCode: 0,
				Usage:    &Usage{Failed: true},
			},
			wantVerd:   Rejected,
			wantReason: "ExitCode is 0 but Usage.Failed is true",
		},
		{
			name: "stdout contains error",
			evidence: Evidence{
				ExitCode: 0,
				Stdout:   "some Error occurred",
				Usage:    &Usage{Completed: true},
				Artifact: "ok.txt",
			},
			wantVerd:   Rejected,
			wantReason: "Stdout contains error information",
		},
		{
			name: "completed but artifact missing",
			evidence: Evidence{
				ExitCode: 0,
				Usage:    &Usage{Completed: true},
				Artifact: "missing.txt",
			},
			wantVerd:   Rejected,
			wantReason: "Artifact missing.txt does not exist or is empty",
		},
		{
			name: "completed but artifact empty",
			evidence: Evidence{
				ExitCode: 0,
				Usage:    &Usage{Completed: true},
				Artifact: "empty.txt",
			},
			wantVerd:   Rejected,
			wantReason: "Artifact empty.txt does not exist or is empty",
		},
		{
			name: "completed but artifact mtime before dispatch",
			evidence: Evidence{
				ExitCode:     0,
				Usage:        &Usage{Completed: true},
				Artifact:     "ok.txt",
				DispatchTime: time.Now().Add(1 * time.Hour), // Future dispatch time means artifact is "old"
			},
			wantVerd:   Rejected,
			wantReason: "Artifact ok.txt mtime is earlier than dispatch time",
		},
		{
			name: "non-zero exit code",
			evidence: Evidence{
				ExitCode: 1,
			},
			wantVerd:   Rejected,
			wantReason: "Non-zero exit code: 1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v, reasons := Judge(tt.evidence)
			if v != tt.wantVerd {
				t.Errorf("Judge() got = %v, want %v", v, tt.wantVerd)
			}
			if tt.wantVerd == Rejected {
				found := false
				for _, r := range reasons {
					if strings.Contains(r.Message, tt.wantReason) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected reason containing '%s', got %v", tt.wantReason, reasons)
				}
			}
		})
	}
}
