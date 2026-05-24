//go:build !integration

package workflow

import (
	"path/filepath"
	"testing"

	"github.com/github/gh-aw/pkg/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestValidateExpressions tests expression safety and runtime-import validation.
func TestValidateExpressions(t *testing.T) {
	tests := []struct {
		name          string
		markdown      string
		shouldError   bool
		errorContains string
	}{
		{
			name:        "no expressions",
			markdown:    "# Hello\n\nNo expressions here.",
			shouldError: false,
		},
		{
			name:        "safe expression",
			markdown:    "# Hello\n\n${{ github.event.issue.number }}",
			shouldError: false,
		},
		{
			name:          "unsafe expression in markdown",
			markdown:      "# Hello\n\n${{ github.event.issue.body }}",
			shouldError:   true,
			errorContains: "unauthorized expressions found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := testutil.TempDir(t, "expr-test")
			markdownPath := filepath.Join(tmpDir, "test.md")

			compiler := NewCompiler()
			workflowData := &WorkflowData{
				Name:            "Test",
				MarkdownContent: tt.markdown,
				AI:              "copilot",
			}

			err := compiler.validateExpressions(workflowData, markdownPath)
			if tt.shouldError {
				require.Error(t, err, "Expected validateExpressions to return an error")
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains, "Error should contain expected message")
				}
			} else {
				assert.NoError(t, err, "validateExpressions should not return an error")
			}
		})
	}
}

// TestValidateFeatureConfig tests feature flag and action-mode validation.
func TestValidateFeatureConfig(t *testing.T) {
	tests := []struct {
		name          string
		features      map[string]any
		inlineDisable bool
		shouldError   bool
		errorContains string
	}{
		{
			name:          "no features",
			features:      nil,
			inlineDisable: false,
			shouldError:   false,
		},
		{
			name: "valid action-mode dev",
			features: map[string]any{
				"action-mode": "dev",
			},
			inlineDisable: false,
			shouldError:   false,
		},
		{
			name: "valid action-mode release",
			features: map[string]any{
				"action-mode": "release",
			},
			inlineDisable: false,
			shouldError:   false,
		},
		{
			name: "invalid action-mode",
			features: map[string]any{
				"action-mode": "invalid-mode",
			},
			inlineDisable: false,
			shouldError:   true,
			errorContains: "invalid action-mode feature flag",
		},
		{
			name: "empty action-mode is ignored",
			features: map[string]any{
				"action-mode": "",
			},
			inlineDisable: false,
			shouldError:   false,
		},
		{
			name:          "inline-sub-agents false is rejected",
			features:      nil,
			inlineDisable: true,
			shouldError:   true,
			errorContains: "inline-sub-agents: false is not supported",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := testutil.TempDir(t, "feature-test")
			markdownPath := filepath.Join(tmpDir, "test.md")

			compiler := NewCompiler()
			workflowData := &WorkflowData{
				Name:                    "Test",
				MarkdownContent:         "# Test",
				AI:                      "copilot",
				Features:                tt.features,
				InlineSubAgentsDisabled: tt.inlineDisable,
			}

			err := compiler.validateFeatureConfig(workflowData, markdownPath)
			if tt.shouldError {
				require.Error(t, err, "Expected validateFeatureConfig to return an error")
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains, "Error should contain expected message")
				}
			} else {
				assert.NoError(t, err, "validateFeatureConfig should not return an error")
			}
		})
	}
}

// TestValidatePermissions tests permission parsing and MCP tool constraint validation.
func TestValidatePermissions(t *testing.T) {
	tests := []struct {
		name            string
		workflowData    *WorkflowData
		strictMode      bool
		shouldError     bool
		errorContains   string
		wantPermissions bool // whether the returned *Permissions should be non-nil
	}{
		{
			name: "no permissions returns empty Permissions",
			workflowData: &WorkflowData{
				Name:            "Test",
				MarkdownContent: "# Test",
				AI:              "copilot",
				Permissions:     "",
			},
			shouldError:     false,
			wantPermissions: true,
		},
		{
			name: "valid permissions parses successfully",
			workflowData: &WorkflowData{
				Name:            "Test",
				MarkdownContent: "# Test",
				AI:              "copilot",
				Permissions:     "permissions:\n  contents: read\n",
			},
			shouldError:     false,
			wantPermissions: true,
		},
		{
			name: "engine auth github-oidc requires id-token write",
			workflowData: &WorkflowData{
				Name:            "Test",
				MarkdownContent: "# Test",
				AI:              "copilot",
				Permissions:     "permissions:\n  contents: read\n",
				EngineConfig: &EngineConfig{
					ID: "copilot",
					Auth: &EngineAuthConfig{
						Type: "github-oidc",
					},
				},
			},
			shouldError:     true,
			errorContains:   "engine.auth.type: github-oidc requires permissions.id-token: write",
			wantPermissions: false,
		},
		{
			name: "engine auth github-oidc with id-token write succeeds",
			workflowData: &WorkflowData{
				Name:            "Test",
				MarkdownContent: "# Test",
				AI:              "copilot",
				Permissions:     "permissions:\n  contents: read\n  id-token: write\n",
				EngineConfig: &EngineConfig{
					ID: "copilot",
					Auth: &EngineAuthConfig{
						Type: "github-oidc",
					},
				},
			},
			shouldError:     false,
			wantPermissions: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := testutil.TempDir(t, "perms-test")
			markdownPath := filepath.Join(tmpDir, "test.md")

			compiler := NewCompiler()
			compiler.strictMode = tt.strictMode

			perms, err := compiler.validatePermissions(tt.workflowData, markdownPath)
			if tt.shouldError {
				require.Error(t, err, "Expected validatePermissions to return an error")
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains, "Error should contain expected message")
				}
			} else {
				require.NoError(t, err, "validatePermissions should not return an error")
				if tt.wantPermissions {
					assert.NotNil(t, perms, "validatePermissions should return a non-nil *Permissions")
				}
			}
		})
	}
}

// TestValidateToolConfiguration tests safe-outputs, GitHub tools, and dispatch validation.
func TestValidateToolConfiguration(t *testing.T) {
	tests := []struct {
		name          string
		workflowData  *WorkflowData
		permissions   string // raw permissions YAML to parse
		shouldError   bool
		errorContains string
	}{
		{
			name: "minimal workflow passes",
			workflowData: &WorkflowData{
				Name:            "Test",
				MarkdownContent: "# Test",
				AI:              "copilot",
			},
			permissions: "",
			shouldError: false,
		},
		{
			name: "agentic-workflows tool requires actions read permission",
			workflowData: &WorkflowData{
				Name:            "Test",
				MarkdownContent: "# Test",
				AI:              "copilot",
				Tools: map[string]any{
					"agentic-workflows": map[string]any{},
				},
				Permissions: "",
			},
			permissions:   "",
			shouldError:   true,
			errorContains: "Missing required permission for agentic-workflows tool",
		},
		{
			name: "agentic-workflows tool with actions read succeeds",
			workflowData: &WorkflowData{
				Name:            "Test",
				MarkdownContent: "# Test",
				AI:              "copilot",
				Tools: map[string]any{
					"agentic-workflows": map[string]any{},
				},
				Permissions: "permissions:\n  actions: read\n",
			},
			permissions: "permissions:\n  actions: read\n",
			shouldError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := testutil.TempDir(t, "tool-test")
			markdownPath := filepath.Join(tmpDir, "test.md")

			compiler := NewCompiler()
			parsedPermissions := NewPermissionsParser(tt.permissions).ToPermissions()

			err := compiler.validateToolConfiguration(tt.workflowData, markdownPath, parsedPermissions)
			if tt.shouldError {
				require.Error(t, err, "Expected validateToolConfiguration to return an error")
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains, "Error should contain expected message")
				}
			} else {
				assert.NoError(t, err, "validateToolConfiguration should not return an error")
			}
		})
	}
}

// TestWarnPromptTmpPaths tests the /tmp path heuristic used by the compiler.
func TestWarnPromptTmpPaths(t *testing.T) {
	tests := []struct {
		name          string
		content       string
		expectWarning bool
	}{
		{
			name:          "no tmp reference",
			content:       "# Workflow\n\nDo some work.",
			expectWarning: false,
		},
		{
			name:          "correct path /tmp/gh-aw/agent/ only",
			content:       "Store output in /tmp/gh-aw/agent/result.txt",
			expectWarning: false,
		},
		{
			name:          "correct path /tmp/gh-aw/agent/ subdirectory",
			content:       "Write to /tmp/gh-aw/agent/logs/output.log for later inspection.",
			expectWarning: false,
		},
		{
			name:          "bare /tmp/ reference",
			content:       "Save the file to /tmp/myfile.txt",
			expectWarning: true,
		},
		{
			name:          "/tmp/gh-aw/ subpath is safe",
			content:       "Write your output to /tmp/gh-aw/result.json",
			expectWarning: false,
		},
		{
			name:          "/tmp/ root only",
			content:       "Use /tmp/ for temporary storage.",
			expectWarning: true,
		},
		{
			name:          "mix of correct and raw /tmp/ reference",
			content:       "Prefer /tmp/gh-aw/agent/ but avoid plain /tmp/foo paths.",
			expectWarning: true,
		},
		{
			name:          "cache-memory path is safe",
			content:       "Read state from /tmp/gh-aw/cache-memory/my-workflow/state.json",
			expectWarning: false,
		},
		{
			name:          "named cache-memory path is safe",
			content:       "Use /tmp/gh-aw/cache-memory-focus-areas/ for storage.",
			expectWarning: false,
		},
		{
			name:          "repo-memory path is safe",
			content:       "Access shared memory at /tmp/gh-aw/repo-memory/default/metrics.json",
			expectWarning: false,
		},
		{
			name:          "comment-memory path is safe",
			content:       "Append haiku to /tmp/gh-aw/comment-memory/notes.md",
			expectWarning: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := warnPromptTmpPaths(tt.content)
			if tt.expectWarning {
				assert.NotEmpty(t, msg, "expected a warning message")
				assert.Contains(t, msg, "/tmp/gh-aw/agent/", "warning should suggest /tmp/gh-aw/agent/")
			} else {
				assert.Empty(t, msg, "expected no warning message")
			}
		})
	}
}

// TestValidatePromptTmpPaths tests that validatePromptTmpPaths increments the
// warning counter and emits a message when the markdown body has problematic /tmp paths.
func TestValidatePromptTmpPaths(t *testing.T) {
	tests := []struct {
		name       string
		markdown   string
		expectWarn bool
	}{
		{
			name:       "no tmp reference — no warning",
			markdown:   "# Hello\n\nDo some work.",
			expectWarn: false,
		},
		{
			name:       "correct /tmp/gh-aw/agent/ — no warning",
			markdown:   "Store results in /tmp/gh-aw/agent/output.txt",
			expectWarn: false,
		},
		{
			name:       "bare /tmp/ reference — warning",
			markdown:   "Save artifacts to /tmp/data.json",
			expectWarn: true,
		},
		{
			name:       "/tmp/gh-aw/ subpath — no warning",
			markdown:   "Write summary to /tmp/gh-aw/summary.txt",
			expectWarn: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := testutil.TempDir(t, "tmp-path-test")
			markdownPath := filepath.Join(tmpDir, "workflow.md")

			compiler := NewCompiler()
			workflowData := &WorkflowData{
				Name:            "Test",
				MarkdownContent: tt.markdown,
				AI:              "copilot",
			}

			before := compiler.GetWarningCount()
			compiler.validatePromptTmpPaths(workflowData, markdownPath)
			after := compiler.GetWarningCount()

			if tt.expectWarn {
				assert.Greater(t, after, before, "warning count should have increased")
			} else {
				assert.Equal(t, before, after, "warning count should not have changed")
			}
		})
	}
}
