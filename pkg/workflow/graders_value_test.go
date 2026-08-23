package workflow

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestPrepareValueGrader(t *testing.T) {
	repoRoot := t.TempDir()
	if err := os.Mkdir(filepath.Join(repoRoot, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	workflowPath := filepath.Join(repoRoot, ".github", "workflows", "example.md")
	functionPath := filepath.Join(repoRoot, ".github", "graders", "value.sh")
	if err := os.MkdirAll(filepath.Dir(workflowPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(functionPath), 0o755); err != nil {
		t.Fatal(err)
	}
	content := "#!/usr/bin/env bash\nset -euo pipefail\n"
	if err := os.WriteFile(functionPath, []byte(content), 0o755); err != nil {
		t.Fatal(err)
	}
	data := valueGraderWorkflowData(".github/graders/value.sh")

	if err := (&Compiler{}).prepareValueGrader(data, workflowPath); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	grader := data.Graders.Graders["value"]
	if grader.functionContent != content {
		t.Fatal("expected value function content to be frozen")
	}
	if len(grader.FunctionDigest()) != 64 {
		t.Fatalf("expected SHA-256 digest, got %q", grader.FunctionDigest())
	}
}

func TestPrepareValueGraderRejectsInvalidFiles(t *testing.T) {
	tests := []struct {
		name    string
		content string
		errText string
	}{
		{name: "missing", errText: "cannot read"},
		{name: "not bash", content: "echo value\n", errText: "Bash shebang"},
		{name: "oversized", content: "#!/usr/bin/env bash\n" + strings.Repeat("x", maxValueFunctionSize), errText: "exceeds"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repoRoot := t.TempDir()
			if err := os.Mkdir(filepath.Join(repoRoot, ".git"), 0o755); err != nil {
				t.Fatal(err)
			}
			workflowPath := filepath.Join(repoRoot, ".github", "workflows", "example.md")
			functionPath := filepath.Join(repoRoot, ".github", "graders", "value.sh")
			if err := os.MkdirAll(filepath.Dir(workflowPath), 0o755); err != nil {
				t.Fatal(err)
			}
			if test.content != "" {
				if err := os.MkdirAll(filepath.Dir(functionPath), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(functionPath, []byte(test.content), 0o755); err != nil {
					t.Fatal(err)
				}
			}

			err := (&Compiler{}).prepareValueGrader(valueGraderWorkflowData(".github/graders/value.sh"), workflowPath)
			if err == nil || !strings.Contains(err.Error(), test.errText) {
				t.Fatalf("expected error containing %q, got %v", test.errText, err)
			}
		})
	}
}

func TestPrepareValueGraderRejectsSymlinkEscape(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation requires additional privileges on Windows")
	}
	repoRoot := t.TempDir()
	outside := filepath.Join(t.TempDir(), "value.sh")
	if err := os.WriteFile(outside, []byte("#!/usr/bin/env bash\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(repoRoot, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	workflowPath := filepath.Join(repoRoot, ".github", "workflows", "example.md")
	functionPath := filepath.Join(repoRoot, ".github", "graders", "value.sh")
	if err := os.MkdirAll(filepath.Dir(workflowPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(functionPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, functionPath); err != nil {
		t.Fatal(err)
	}

	err := (&Compiler{}).prepareValueGrader(valueGraderWorkflowData(".github/graders/value.sh"), workflowPath)
	if err == nil || !strings.Contains(err.Error(), "escapes") {
		t.Fatalf("expected symlink escape error, got %v", err)
	}
}

func valueGraderWorkflowData(functionPath string) *WorkflowData {
	return &WorkflowData{
		Graders: &GradersConfig{
			Graders: map[string]*GraderDefinition{
				"value": {ID: "value", Function: functionPath},
			},
		},
	}
}
