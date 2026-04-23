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
		shouldError   bool
		errorContains string
	}{
		{
			name:        "no features",
			features:    nil,
			shouldError: false,
		},
		{
			name: "valid action-mode dev",
			features: map[string]any{
				"action-mode": "dev",
			},
			shouldError: false,
		},
		{
			name: "valid action-mode release",
			features: map[string]any{
				"action-mode": "release",
			},
			shouldError: false,
		},
		{
			name: "invalid action-mode",
			features: map[string]any{
				"action-mode": "invalid-mode",
			},
			shouldError:   true,
			errorContains: "invalid action-mode feature flag",
		},
		{
			name: "empty action-mode is ignored",
			features: map[string]any{
				"action-mode": "",
			},
			shouldError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := testutil.TempDir(t, "feature-test")
			markdownPath := filepath.Join(tmpDir, "test.md")

			compiler := NewCompiler()
			workflowData := &WorkflowData{
				Name:            "Test",
				MarkdownContent: "# Test",
				AI:              "copilot",
				Features:        tt.features,
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
