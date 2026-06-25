//go:build integration

package workflow

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/testutil"
)

func TestBYOKProviderEnvPassthroughIntegration(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		engineBlock     string
		toolsBlock      string
		expectModelFlag string
	}{
		{
			name: "copilot",
			engineBlock: `engine:
  id: copilot
  env:
    COPILOT_PROVIDER_BASE_URL: ${{ secrets.PROVIDER_BASE_URL }}
    COPILOT_PROVIDER_API_KEY: ${{ secrets.PROVIDER_API_KEY }}`,
		},
		{
			name: "pi",
			engineBlock: `engine:
  id: pi
  model: copilot/claude-sonnet-4
  env:
    COPILOT_PROVIDER_BASE_URL: ${{ secrets.PROVIDER_BASE_URL }}
    COPILOT_PROVIDER_API_KEY: ${{ secrets.PROVIDER_API_KEY }}`,
			toolsBlock: `tools:
  github:
    mode: gh-proxy
  cli-proxy: true`,
			expectModelFlag: "aw-gateway/claude-sonnet-4",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := testutil.TempDir(t, "test-*")
			workflowsDir := filepath.Join(tmpDir, ".github", "workflows")
			if err := os.MkdirAll(workflowsDir, 0o755); err != nil {
				t.Fatalf("Failed to create workflows directory: %v", err)
			}

			workflowContent := fmt.Sprintf(`---
on: workflow_dispatch
permissions:
  contents: read
network:
  allowed:
    - defaults
    - api.openai.com
%s
%s
---

# Test BYOK passthrough

Verify BYOK provider env vars remain available inside AWF.
`, tt.engineBlock, tt.toolsBlock)

			workflowPath := filepath.Join(workflowsDir, "test-byok-"+tt.name+".md")
			if err := os.WriteFile(workflowPath, []byte(workflowContent), 0o644); err != nil {
				t.Fatalf("Failed to write workflow file: %v", err)
			}

			compiler := NewCompiler(WithVersion("test-byok-" + tt.name))
			if err := compiler.CompileWorkflow(workflowPath); err != nil {
				t.Fatalf("Failed to compile workflow: %v", err)
			}

			lockPath := filepath.Join(workflowsDir, "test-byok-"+tt.name+".lock.yml")
			lockContent, err := os.ReadFile(lockPath)
			if err != nil {
				t.Fatalf("Failed to read compiled workflow: %v", err)
			}

			lockYAML := string(lockContent)

			if !strings.Contains(lockYAML, constants.CopilotProviderBaseURL+": ${{ secrets.PROVIDER_BASE_URL }}") {
				t.Fatalf("Compiled workflow should keep %s in step env, got:\n%s", constants.CopilotProviderBaseURL, lockYAML)
			}
			if !strings.Contains(lockYAML, constants.CopilotProviderAPIKey+": ${{ secrets.PROVIDER_API_KEY }}") {
				t.Fatalf("Compiled workflow should keep %s in step env, got:\n%s", constants.CopilotProviderAPIKey, lockYAML)
			}
			if strings.Contains(lockYAML, "--exclude-env "+constants.CopilotProviderBaseURL) {
				t.Fatalf("Compiled workflow should not exclude %s in BYOK mode, got:\n%s", constants.CopilotProviderBaseURL, lockYAML)
			}
			if strings.Contains(lockYAML, "--exclude-env "+constants.CopilotProviderAPIKey) {
				t.Fatalf("Compiled workflow should not exclude %s in BYOK mode, got:\n%s", constants.CopilotProviderAPIKey, lockYAML)
			}
			if tt.expectModelFlag != "" && !strings.Contains(lockYAML, tt.expectModelFlag) {
				t.Fatalf("Compiled workflow should route Pi through %q, got:\n%s", tt.expectModelFlag, lockYAML)
			}
		})
	}
}
