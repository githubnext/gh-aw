package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestSafeOutputsMCPConfigFormat verifies that the GITHUB_AW_SAFE_OUTPUTS_CONFIG
// environment variable is properly formatted in the MCP configuration.
// This test prevents regression of the toJSON double-encoding bug.
func TestSafeOutputsMCPConfigFormat(t *testing.T) {
	tests := []struct {
		name           string
		engine         string
		expectedFormat string
	}{
		{
			name:           "Claude engine uses quoted env variable",
			engine:         "claude",
			expectedFormat: `"GITHUB_AW_SAFE_OUTPUTS_CONFIG": "${{ env.GITHUB_AW_SAFE_OUTPUTS_CONFIG }}"`,
		},
		{
			name:           "Custom engine uses quoted env variable",
			engine:         "custom",
			expectedFormat: `"GITHUB_AW_SAFE_OUTPUTS_CONFIG": "${{ env.GITHUB_AW_SAFE_OUTPUTS_CONFIG }}"`,
		},
		{
			name:           "Copilot engine uses quoted env variable",
			engine:         "copilot",
			expectedFormat: `"GITHUB_AW_SAFE_OUTPUTS_CONFIG": "${{ env.GITHUB_AW_SAFE_OUTPUTS_CONFIG }}"`,
		},
		{
			name:           "Codex engine uses quoted env variable",
			engine:         "codex",
			expectedFormat: `"GITHUB_AW_SAFE_OUTPUTS_CONFIG" = "${{ env.GITHUB_AW_SAFE_OUTPUTS_CONFIG }}"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a minimal workflow with safe-outputs
			workflowContent := `---
on: push
engine: ` + tt.engine + `
safe-outputs:
  create-issue:
    max: 1
---
# Test workflow
Test workflow content.
`

			// Create a temporary workflow file
			tempDir := t.TempDir()
			workflowPath := filepath.Join(tempDir, "test-workflow.md")
			if err := os.WriteFile(workflowPath, []byte(workflowContent), 0644); err != nil {
				t.Fatalf("Failed to create workflow file: %v", err)
			}

			// Compile the workflow
			compiler := NewCompiler(false, "", "test")
			err := compiler.CompileWorkflow(workflowPath)
			if err != nil {
				t.Fatalf("Failed to compile workflow: %v", err)
			}

			// Read the generated lock file
			lockFilePath := strings.TrimSuffix(workflowPath, ".md") + ".lock.yml"
			compiledYAMLBytes, err := os.ReadFile(lockFilePath)
			if err != nil {
				t.Fatalf("Failed to read compiled workflow: %v", err)
			}
			compiledYAML := string(compiledYAMLBytes)

			// Verify the expected format is present
			if !strings.Contains(compiledYAML, tt.expectedFormat) {
				t.Errorf("Expected MCP config format not found.\nExpected substring: %s\nGenerated YAML does not contain the expected format.\nSearching for incorrect format with toJSON...", tt.expectedFormat)

				// Check if the incorrect format is present
				incorrectFormat := `GITHUB_AW_SAFE_OUTPUTS_CONFIG": ${{ toJSON(env.GITHUB_AW_SAFE_OUTPUTS_CONFIG) }}`
				if strings.Contains(compiledYAML, incorrectFormat) {
					t.Errorf("REGRESSION: Found incorrect toJSON format in generated YAML!")
				}

				// Find and print the actual line
				lines := strings.Split(compiledYAML, "\n")
				for i, line := range lines {
					if strings.Contains(line, "GITHUB_AW_SAFE_OUTPUTS_CONFIG") {
						t.Errorf("Line %d: %s", i+1, line)
					}
				}
			}

			// Also verify that toJSON is NOT used (regression check)
			incorrectPattern := `${{ toJSON(env.GITHUB_AW_SAFE_OUTPUTS_CONFIG) }}`
			if strings.Contains(compiledYAML, incorrectPattern) {
				t.Errorf("REGRESSION: Found toJSON in GITHUB_AW_SAFE_OUTPUTS_CONFIG, which causes double-encoding bug")
			}
		})
	}
}
