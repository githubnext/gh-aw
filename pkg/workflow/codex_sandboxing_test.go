package workflow

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCodexNetworkAccessEvaluation(t *testing.T) {
	engine := NewCodexEngine()

	tests := []struct {
		name               string
		networkPermissions *NetworkPermissions
		expectedAccess     bool
		expectError        bool
		expectedErrorMsg   string
	}{
		{
			name:               "nil permissions should default to false",
			networkPermissions: nil,
			expectedAccess:     false,
			expectError:        false,
		},
		{
			name: "defaults mode should return false",
			networkPermissions: &NetworkPermissions{
				Mode: "defaults",
			},
			expectedAccess: false,
			expectError:    false,
		},
		{
			name: "empty allowed list should return false",
			networkPermissions: &NetworkPermissions{
				Allowed: []string{},
			},
			expectedAccess: false,
			expectError:    false,
		},
		{
			name: "wildcard should return true",
			networkPermissions: &NetworkPermissions{
				Allowed: []string{"*"},
			},
			expectedAccess: true,
			expectError:    false,
		},
		{
			name: "wildcard with other domains should return true",
			networkPermissions: &NetworkPermissions{
				Allowed: []string{"example.com", "*", "api.github.com"},
			},
			expectedAccess: true,
			expectError:    false,
		},
		{
			name: "specific domains should cause error",
			networkPermissions: &NetworkPermissions{
				Allowed: []string{"example.com", "api.github.com"},
			},
			expectedAccess:   false,
			expectError:      true,
			expectedErrorMsg: "Codex sandboxing does not support specific domain allowlists",
		},
		{
			name: "single specific domain should cause error",
			networkPermissions: &NetworkPermissions{
				Allowed: []string{"api.github.com"},
			},
			expectedAccess:   false,
			expectError:      true,
			expectedErrorMsg: "Codex sandboxing does not support specific domain allowlists",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			access, err := engine.evaluateNetworkAccessForCodex(tt.networkPermissions)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if tt.expectedErrorMsg != "" && !strings.Contains(err.Error(), tt.expectedErrorMsg) {
					t.Errorf("Expected error to contain '%s', got: %s", tt.expectedErrorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}

			if access != tt.expectedAccess {
				t.Errorf("Expected network access %t, got %t", tt.expectedAccess, access)
			}
		})
	}
}

func TestCodexSandboxingConfigGeneration(t *testing.T) {
	// Create temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "codex-sandbox-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	compiler := NewCompiler(false, "", "test")

	tests := []struct {
		name                  string
		frontmatter           string
		expectSandboxMode     bool
		expectNetworkAccess   bool
		expectError           bool
		expectedErrorInConfig string
	}{
		{
			name: "codex with defaults network should disable network_access",
			frontmatter: `---
engine: codex
tools:
  github:
    allowed: [get_issue]
network: defaults
---`,
			expectSandboxMode:   true,
			expectNetworkAccess: false,
			expectError:         false,
		},
		{
			name: "codex with wildcard network should enable network_access",
			frontmatter: `---
engine: codex
tools:
  github:
    allowed: [get_issue]
network:
  allowed: ["*"]
---`,
			expectSandboxMode:   true,
			expectNetworkAccess: true,
			expectError:         false,
		},
		{
			name: "codex with no network config should default to disabled network_access",
			frontmatter: `---
engine: codex
tools:
  github:
    allowed: [get_issue]
---`,
			expectSandboxMode:   true,
			expectNetworkAccess: false,
			expectError:         false,
		},
		{
			name: "codex with specific domains should generate error in config",
			frontmatter: `---
engine: codex
tools:
  github:
    allowed: [get_issue]
network:
  allowed: ["api.github.com", "example.com"]
---`,
			expectSandboxMode:     true,
			expectNetworkAccess:   false,
			expectError:           false, // Error is in config, not compilation
			expectedErrorInConfig: "ERROR: Codex sandboxing does not support specific domain allowlists",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testContent := tt.frontmatter + `

# Test Codex Sandboxing

This is a test workflow for Codex sandboxing configuration.
`

			testFile := filepath.Join(tmpDir, tt.name+"-workflow.md")
			if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
				t.Fatal(err)
			}

			// Compile the workflow
			err := compiler.CompileWorkflow(testFile)
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected compilation error but got none")
				}
				return
			}
			if err != nil {
				t.Fatalf("Unexpected compilation error: %v", err)
			}

			// Read the generated lock file
			lockFile := strings.TrimSuffix(testFile, ".md") + ".lock.yml"
			content, err := os.ReadFile(lockFile)
			if err != nil {
				t.Fatalf("Failed to read lock file: %v", err)
			}

			lockContent := string(content)

			// Check for sandbox_mode
			if tt.expectSandboxMode {
				if !strings.Contains(lockContent, `sandbox_mode = "workspace-write"`) {
					t.Errorf("Expected sandbox_mode = \"workspace-write\" but didn't find it in:\n%s", lockContent)
				}

				// Check for [sandbox_workspace_write] section
				if !strings.Contains(lockContent, "[sandbox_workspace_write]") {
					t.Errorf("Expected [sandbox_workspace_write] section but didn't find it in:\n%s", lockContent)
				}

				// Check network_access setting
				expectedNetworkAccess := fmt.Sprintf("network_access = %t", tt.expectNetworkAccess)
				if !strings.Contains(lockContent, expectedNetworkAccess) {
					t.Errorf("Expected %s but didn't find it in:\n%s", expectedNetworkAccess, lockContent)
				}

				// Check for expected error in config
				if tt.expectedErrorInConfig != "" {
					if !strings.Contains(lockContent, tt.expectedErrorInConfig) {
						t.Errorf("Expected error in config '%s' but didn't find it in:\n%s", tt.expectedErrorInConfig, lockContent)
					}
				}
			}
		})
	}
}
