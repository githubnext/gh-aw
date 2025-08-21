package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateWorkflowSchema(t *testing.T) {
	compiler := NewCompiler(false, "", "test")
	compiler.SetSkipValidation(false) // Enable validation for testing

	tests := []struct {
		name    string
		yaml    string
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid minimal workflow",
			yaml: `name: "Test Workflow"
on: push
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout
        uses: actions/checkout@v3`,
			wantErr: false,
		},
		{
			name: "invalid workflow - missing jobs",
			yaml: `name: "Test Workflow"
on: push`,
			wantErr: true,
			errMsg:  "missing property 'jobs'",
		},
		{
			name: "invalid workflow - invalid YAML",
			yaml: `name: "Test Workflow"
on: push
jobs:
  test: [invalid yaml structure`,
			wantErr: true,
			errMsg:  "failed to parse generated YAML",
		},
		{
			name: "invalid workflow - invalid job structure",
			yaml: `name: "Test Workflow"
on: push
jobs:
  test:
    invalid-property: value`,
			wantErr: true,
			errMsg:  "validation failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := compiler.validateWorkflowSchema(tt.yaml)

			if tt.wantErr {
				if err == nil {
					t.Errorf("validateWorkflowSchema() expected error but got none")
					return
				}
				if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("validateWorkflowSchema() error = %v, expected to contain %v", err, tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("validateWorkflowSchema() unexpected error = %v", err)
				}
			}
		})
	}
}

func TestValidationCanBeSkipped(t *testing.T) {
	compiler := NewCompiler(false, "", "test")

	// Test via CompileWorkflow - should succeed because validation is skipped by default
	tmpDir, err := os.MkdirTemp("", "validation-skip-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	testContent := `---
name: Test Workflow
on: push
---
# Test workflow`

	testFile := filepath.Join(tmpDir, "test.md")
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatal(err)
	}

	compiler.customOutput = tmpDir

	// This should succeed because validation is skipped by default
	err = compiler.CompileWorkflow(testFile)
	if err != nil {
		t.Errorf("CompileWorkflow() should succeed when validation is skipped, but got error: %v", err)
	}
}
