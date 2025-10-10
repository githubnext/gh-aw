package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGitHubToolsetIntegration(t *testing.T) {
	tests := []struct {
		name           string
		workflowMD     string
		expectedInYAML []string
		notInYAML      []string
	}{
		{
			name: "Claude engine with toolsets",
			workflowMD: `---
on: push
engine: claude
tools:
  github:
    toolset: [repos, issues, pull_requests]
---

# Test Workflow

This workflow tests GitHub toolsets.
`,
			expectedInYAML: []string{
				`GITHUB_TOOLSETS`,
				`repos,issues,pull_requests`,
			},
			notInYAML: []string{},
		},
		{
			name: "Copilot engine with array toolsets",
			workflowMD: `---
on: push
engine: copilot
tools:
  github:
    toolset: [repos, issues, actions]
---

# Test Workflow

This workflow tests GitHub toolsets as array.
`,
			expectedInYAML: []string{
				`GITHUB_TOOLSETS`,
				`repos,issues,actions`,
			},
			notInYAML: []string{},
		},
		{
			name: "Codex engine with all toolset",
			workflowMD: `---
on: push
engine: codex
tools:
  github:
    toolset: [all]
---

# Test Workflow

This workflow enables all GitHub toolsets.
`,
			expectedInYAML: []string{
				`GITHUB_TOOLSETS`,
				`all`,
			},
			notInYAML: []string{},
		},
		{
			name: "Workflow without toolsets",
			workflowMD: `---
on: push
engine: claude
tools:
  github:
---

# Test Workflow

This workflow has no toolsets configured.
`,
			expectedInYAML: []string{
				`GITHUB_PERSONAL_ACCESS_TOKEN`,
				`GITHUB_TOOLSETS`,
			},
		},
		{
			name: "Toolsets with read-only mode",
			workflowMD: `---
on: push
engine: claude
tools:
  github:
    toolset: [repos, issues]
    read-only: true
---

# Test Workflow

This workflow combines toolsets with read-only mode.
`,
			expectedInYAML: []string{
				`GITHUB_TOOLSETS`,
				`repos,issues`,
				`GITHUB_READ_ONLY`,
			},
			notInYAML: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory for test
			tempDir := t.TempDir()
			mdPath := filepath.Join(tempDir, "test-workflow.md")

			// Write workflow file
			err := os.WriteFile(mdPath, []byte(tt.workflowMD), 0644)
			if err != nil {
				t.Fatalf("Failed to write test workflow: %v", err)
			}

			// Compile the workflow
			compiler := NewCompiler(false, "", "test")
			compileErr := compiler.CompileWorkflow(mdPath)
			if compileErr != nil {
				t.Fatalf("Failed to compile workflow: %v", compileErr)
			}

			// Read the generated YAML (same directory, .lock.yml extension)
			yamlPath := strings.TrimSuffix(mdPath, ".md") + ".lock.yml"
			yamlContent, err := os.ReadFile(yamlPath)
			if err != nil {
				t.Fatalf("Failed to read generated YAML: %v", err)
			}

			yamlStr := string(yamlContent)

			// Check expected strings
			for _, expected := range tt.expectedInYAML {
				if !strings.Contains(yamlStr, expected) {
					t.Errorf("Expected YAML to contain %q, but it didn't.\nGenerated YAML:\n%s", expected, yamlStr)
				}
			}

			// Check strings that should not be present
			for _, notExpected := range tt.notInYAML {
				if strings.Contains(yamlStr, notExpected) {
					t.Errorf("Expected YAML to NOT contain %q, but it did.\nGenerated YAML:\n%s", notExpected, yamlStr)
				}
			}
		})
	}
}

func TestGitHubToolsetRemoteMode(t *testing.T) {
	workflowMD := `---
on: push
engine: claude
tools:
  github:
    mode: remote
    toolset: [repos, issues]
---

# Test Workflow

This workflow tests remote mode with array toolsets.
`

	// Create temporary directory for test
	tempDir := t.TempDir()
	mdPath := filepath.Join(tempDir, "test-workflow.md")

	// Write workflow file
	err := os.WriteFile(mdPath, []byte(workflowMD), 0644)
	if err != nil {
		t.Fatalf("Failed to write test workflow: %v", err)
	}

	// Compile the workflow
	compiler := NewCompiler(false, "", "test")
	compileErr := compiler.CompileWorkflow(mdPath)
	if compileErr != nil {
		t.Fatalf("Failed to compile workflow: %v", compileErr)
	}

	// Read the generated YAML (same directory, .lock.yml extension)
	yamlPath := strings.TrimSuffix(mdPath, ".md") + ".lock.yml"
	yamlContent, readErr := os.ReadFile(yamlPath)
	if readErr != nil {
		t.Fatalf("Failed to read generated YAML: %v", readErr)
	}

	yamlStr := string(yamlContent)

	// In remote mode, toolsets are not currently supported via environment variables
	// The remote server might have different configuration mechanism
	// For now, we verify the workflow compiles successfully
	if !strings.Contains(yamlStr, "https://api.githubcopilot.com/mcp/") {
		t.Errorf("Expected remote mode URL in YAML")
	}
}
