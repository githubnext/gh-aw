//go:build !integration

package workflow

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/testutil"
)

func TestSetIssueFieldHandlerConfigIncludesAllowedFields(t *testing.T) {
	tmpDir := testutil.TempDir(t, "set-issue-field-handler-config-test")

	testContent := `---
name: Test Set Issue Field Handler Config
on: workflow_dispatch
permissions:
  contents: read
engine: copilot
safe-outputs:
  set-issue-field:
    max: 2
    allowed-fields: [Priority, Iteration]
---

Set issue field values.
`

	testFile := filepath.Join(tmpDir, "test-set-issue-field-handler-config.md")
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatal(err)
	}

	compiler := NewCompiler()
	if err := compiler.CompileWorkflow(testFile); err != nil {
		t.Fatalf("failed to compile workflow: %v", err)
	}

	outputFile := filepath.Join(tmpDir, "test-set-issue-field-handler-config.lock.yml")
	compiledContent, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("failed to read compiled output: %v", err)
	}

	lines := strings.Split(string(compiledContent), "\n")
	var configJSON string
	for _, line := range lines {
		if strings.Contains(line, "GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG:") {
			parts := strings.SplitN(line, "GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG:", 2)
			if len(parts) == 2 {
				configJSON = strings.TrimSpace(parts[1])
				configJSON = strings.Trim(configJSON, "\"")
				configJSON = strings.ReplaceAll(configJSON, "\\\"", "\"")
				break
			}
		}
	}

	if configJSON == "" {
		t.Fatal("could not extract handler config JSON")
	}

	var config map[string]any
	if err := json.Unmarshal([]byte(configJSON), &config); err != nil {
		t.Fatalf("failed to parse handler config JSON: %v\njson: %s", err, configJSON)
	}

	setIssueFieldConfig, ok := config["set_issue_field"].(map[string]any)
	if !ok {
		t.Fatal("expected set_issue_field in handler config")
	}

	allowedFields, ok := setIssueFieldConfig["allowed_fields"].([]any)
	if !ok {
		t.Fatal("expected allowed_fields array in set_issue_field config")
	}
	if len(allowedFields) != 2 || allowedFields[0] != "Priority" || allowedFields[1] != "Iteration" {
		t.Fatalf("expected allowed_fields=[Priority, Iteration], got: %v", allowedFields)
	}
}
