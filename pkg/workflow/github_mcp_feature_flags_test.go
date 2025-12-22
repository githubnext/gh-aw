package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/githubnext/gh-aw/pkg/testutil"
)

// TestGitHubMCPFeatureFlags tests that feature flags can be passed via args to GitHub MCP server
func TestGitHubMCPFeatureFlags(t *testing.T) {
	tests := []struct {
		name        string
		markdown    string
		expectError bool
		checkFunc   func(t *testing.T, yaml string)
	}{
		{
			name: "feature flag via args in local mode",
			markdown: `---
on: workflow_dispatch
engine: copilot
permissions:
  contents: read
tools:
  github:
    mode: local
    toolsets: [default, actions]
    args:
      - "--features=consolidated-actions"
---
Test workflow with feature flags
`,
			expectError: false,
			checkFunc: func(t *testing.T, yaml string) {
				// Verify feature flag is passed to docker args
				if !strings.Contains(yaml, `"--features=consolidated-actions"`) {
					t.Errorf("Expected feature flag in docker args, but not found.\nYAML:\n%s", yaml)
				}
				// Verify it's after the docker image
				if !strings.Contains(yaml, `"ghcr.io/github/github-mcp-server:`) {
					t.Errorf("Expected GitHub MCP server docker image, but not found")
				}
				// Verify actions toolset is enabled
				if !strings.Contains(yaml, `GITHUB_TOOLSETS=`) {
					t.Errorf("Expected GITHUB_TOOLSETS environment variable, but not found")
				}
			},
		},
		{
			name: "multiple feature flags via args",
			markdown: `---
on: workflow_dispatch
engine: copilot
permissions:
  contents: read
tools:
  github:
    mode: local
    toolsets: [default]
    args:
      - "--features=feature1,feature2"
      - "--verbose"
---
Test workflow with multiple feature flags
`,
			expectError: false,
			checkFunc: func(t *testing.T, yaml string) {
				// Verify both feature flags are present
				if !strings.Contains(yaml, `"--features=feature1,feature2"`) {
					t.Errorf("Expected feature flags in docker args, but not found.\nYAML:\n%s", yaml)
				}
				// Verify additional arg is also present
				if !strings.Contains(yaml, `"--verbose"`) {
					t.Errorf("Expected --verbose arg in docker args, but not found")
				}
			},
		},
		{
			name: "remote mode without feature flags",
			markdown: `---
on: workflow_dispatch
engine: copilot
permissions:
  contents: read
tools:
  github:
    mode: remote
    toolsets: [default, actions]
---
Test workflow with remote mode (feature flags not needed)
`,
			expectError: false,
			checkFunc: func(t *testing.T, yaml string) {
				// Verify remote mode is used (http type)
				if !strings.Contains(yaml, `"type": "http"`) {
					t.Errorf("Expected remote mode (http type), but not found")
				}
				// Verify URL points to hosted service
				if !strings.Contains(yaml, `api.githubcopilot.com`) {
					t.Errorf("Expected hosted MCP server URL, but not found")
				}
				// Verify toolsets are in headers
				if !strings.Contains(yaml, `X-MCP-Toolsets`) {
					t.Errorf("Expected X-MCP-Toolsets header, but not found")
				}
			},
		},
		{
			name: "toolsets work with feature flags",
			markdown: `---
on: workflow_dispatch
engine: copilot
permissions:
  contents: read
  actions: read
  issues: read
tools:
  github:
    mode: local
    toolsets: [default, actions, issues]
    args:
      - "--features=consolidated-actions"
---
Test workflow demonstrating toolsets with feature flags
`,
			expectError: false,
			checkFunc: func(t *testing.T, yaml string) {
				// Verify feature flag is present
				if !strings.Contains(yaml, `"--features=consolidated-actions"`) {
					t.Errorf("Expected feature flag in docker args")
				}
				// Verify toolsets are configured
				if !strings.Contains(yaml, `GITHUB_TOOLSETS=`) {
					t.Errorf("Expected GITHUB_TOOLSETS environment variable")
				}
				// Verify toolsets include actions and issues
				if !strings.Contains(yaml, `actions`) {
					t.Errorf("Expected actions toolset in configuration")
				}
				if !strings.Contains(yaml, `issues`) {
					t.Errorf("Expected issues toolset in configuration")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory and workflow file
			tmpDir := testutil.TempDir(t, "github-mcp-feature-flags-test")
			testFile := filepath.Join(tmpDir, "test-workflow.md")
			
			if err := os.WriteFile(testFile, []byte(tt.markdown), 0644); err != nil {
				t.Fatalf("Failed to write workflow file: %v", err)
			}

			// Compile workflow
			compiler := NewCompiler(false, "", "test-version")
			err := compiler.CompileWorkflow(testFile)
			
			if err != nil && !tt.expectError {
				t.Fatalf("Failed to compile workflow: %v", err)
			}
			if err == nil && tt.expectError {
				t.Fatalf("Expected compilation error, but got none")
			}

			// Read generated lock file
			if !tt.expectError && tt.checkFunc != nil {
				lockFile := strings.Replace(testFile, ".md", ".lock.yml", 1)
				lockContent, err := os.ReadFile(lockFile)
				if err != nil {
					t.Fatalf("Failed to read generated lock file: %v", err)
				}
				
				yaml := string(lockContent)
				tt.checkFunc(t, yaml)
			}
		})
	}
}

// TestGitHubMCPOcticonIconsCompatibility tests that Octicon icons work with v0.26.0+
func TestGitHubMCPOcticonIconsCompatibility(t *testing.T) {
	// Octicon icons are server-side features in MCP server v0.26.0+
	// They don't require any special workflow configuration
	// This test verifies that workflows compile correctly with current version
	markdown := `---
on: workflow_dispatch
engine: copilot
permissions:
  contents: read
  issues: read
  pull-requests: read
  actions: read
tools:
  github:
    mode: remote
    toolsets: [default, issues, pull_requests, actions]
---
Test workflow to verify Octicon icon compatibility (v0.26.0+)
`

	// Create temporary directory and workflow file
	tmpDir := testutil.TempDir(t, "github-mcp-octicon-icons-test")
	testFile := filepath.Join(tmpDir, "test-workflow.md")
	
	if err := os.WriteFile(testFile, []byte(markdown), 0644); err != nil {
		t.Fatalf("Failed to write workflow file: %v", err)
	}

	// Compile workflow
	compiler := NewCompiler(false, "", "test-version")
	if err := compiler.CompileWorkflow(testFile); err != nil {
		t.Fatalf("Failed to compile workflow: %v", err)
	}

	// Read generated lock file
	lockFile := strings.Replace(testFile, ".md", ".lock.yml", 1)
	lockContent, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatalf("Failed to read generated lock file: %v", err)
	}
	
	yaml := string(lockContent)

	// Verify that workflow compiles successfully with multiple toolsets
	// Octicon icons are transparent to compilation - they're added by the MCP server
	if !strings.Contains(yaml, "X-MCP-Toolsets") {
		t.Errorf("Expected toolsets configuration in remote mode")
	}

	// Verify all requested toolsets are present
	requiredToolsets := []string{"default", "issues", "pull_requests", "actions"}
	for _, toolset := range requiredToolsets {
		if !strings.Contains(yaml, toolset) {
			t.Errorf("Expected toolset %s in configuration", toolset)
		}
	}
}

// TestGitHubMCPVersionDefault tests that the default MCP server version is v0.26.3
func TestGitHubMCPVersionDefault(t *testing.T) {
	markdown := `---
on: workflow_dispatch
engine: copilot
permissions:
  contents: read
tools:
  github:
    mode: local
    toolsets: [default]
---
Test default GitHub MCP server version
`

	// Create temporary directory and workflow file
	tmpDir := testutil.TempDir(t, "github-mcp-version-test")
	testFile := filepath.Join(tmpDir, "test-workflow.md")
	
	if err := os.WriteFile(testFile, []byte(markdown), 0644); err != nil {
		t.Fatalf("Failed to write workflow file: %v", err)
	}

	// Compile workflow
	compiler := NewCompiler(false, "", "test-version")
	if err := compiler.CompileWorkflow(testFile); err != nil {
		t.Fatalf("Failed to compile workflow: %v", err)
	}

	// Read generated lock file
	lockFile := strings.Replace(testFile, ".md", ".lock.yml", 1)
	lockContent, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatalf("Failed to read generated lock file: %v", err)
	}
	
	yaml := string(lockContent)

	// Verify default version is v0.26.3 (includes Octicon icons and feature flags support)
	if !strings.Contains(yaml, "ghcr.io/github/github-mcp-server:v0.26.3") {
		t.Errorf("Expected default GitHub MCP server version v0.26.3, but not found.\nYAML:\n%s", yaml)
	}
}
