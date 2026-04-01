//go:build !integration

package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/stringutil"
	"github.com/github/gh-aw/pkg/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestValidateUpdateCheck tests the validateUpdateCheck function
func TestValidateUpdateCheck(t *testing.T) {
	tests := []struct {
		name        string
		frontmatter map[string]any
		strictMode  bool
		wantErr     bool
		errContains string
		wantWarning bool
	}{
		{
			name:        "check-for-updates not set defaults to enabled (no error, no warning)",
			frontmatter: map[string]any{"engine": "copilot"},
			strictMode:  false,
			wantErr:     false,
			wantWarning: false,
		},
		{
			name:        "check-for-updates: true is allowed in any mode",
			frontmatter: map[string]any{"check-for-updates": true},
			strictMode:  true,
			wantErr:     false,
			wantWarning: false,
		},
		{
			name:        "check-for-updates: false in non-strict mode produces warning",
			frontmatter: map[string]any{"check-for-updates": false},
			strictMode:  false,
			wantErr:     false,
			wantWarning: true,
		},
		{
			name:        "check-for-updates: false in strict mode produces error",
			frontmatter: map[string]any{"check-for-updates": false},
			strictMode:  true,
			wantErr:     true,
			errContains: "strict mode",
		},
		{
			name:        "strict mode error message mentions the flag name",
			frontmatter: map[string]any{"check-for-updates": false},
			strictMode:  true,
			wantErr:     true,
			errContains: "check-for-updates: false",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compiler := NewCompiler(WithFailFast(false))
			compiler.strictMode = tt.strictMode

			initialWarnings := compiler.GetWarningCount()
			err := compiler.validateUpdateCheck(tt.frontmatter)

			if tt.wantErr {
				require.Error(t, err, "Expected an error but got none")
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains,
						"Error should contain %q, got: %s", tt.errContains, err.Error())
				}
			} else {
				require.NoError(t, err, "Expected no error but got: %v", err)
			}

			if tt.wantWarning {
				assert.Greater(t, compiler.GetWarningCount(), initialWarnings,
					"Expected a warning to be emitted")
			} else if !tt.wantErr {
				assert.Equal(t, initialWarnings, compiler.GetWarningCount(),
					"Expected no warnings to be emitted")
			}
		})
	}
}

// TestUpdateCheckInActivationJob tests that the update check step is correctly
// added (or omitted) in the activation job based on the UpdateCheckDisabled flag
// and whether the build is a release build.
func TestUpdateCheckInActivationJob(t *testing.T) {
	baseWorkflowMD := `---
engine: copilot
strict: false
on:
  issues:
    types: [opened]
---
Test workflow for update check step.
`
	disabledWorkflowMD := `---
engine: copilot
check-for-updates: false
strict: false
on:
  issues:
    types: [opened]
---
Test workflow for update check step disabled.
`

	tests := []struct {
		name       string
		workflowMD string
		isRelease  bool
		wantStep   bool
	}{
		{
			name:       "step present when enabled and release build",
			workflowMD: baseWorkflowMD,
			isRelease:  true,
			wantStep:   true,
		},
		{
			name:       "step absent when disabled via check-for-updates: false",
			workflowMD: disabledWorkflowMD,
			isRelease:  true,
			wantStep:   false,
		},
		{
			name:       "step absent for dev build (non-release)",
			workflowMD: baseWorkflowMD,
			isRelease:  false,
			wantStep:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save and restore global release state
			originalRelease := isReleaseBuild
			isReleaseBuild = tt.isRelease
			t.Cleanup(func() { isReleaseBuild = originalRelease })

			tmpDir := testutil.TempDir(t, "update-check-test")
			testFile := filepath.Join(tmpDir, "test-workflow.md")
			require.NoError(t, os.WriteFile(testFile, []byte(tt.workflowMD), 0644))

			compiler := NewCompiler()
			err := compiler.CompileWorkflow(testFile)
			require.NoError(t, err, "Workflow should compile without errors")

			lockFile := stringutil.MarkdownToLockFile(testFile)
			lockContent, err := os.ReadFile(lockFile)
			require.NoError(t, err, "Lock file should be readable")
			lockStr := string(lockContent)

			hasStep := strings.Contains(lockStr, "Check compile-agentic version")
			if tt.wantStep {
				assert.True(t, hasStep,
					"Expected 'Check compile-agentic version' step in activation job but not found")
			} else {
				assert.False(t, hasStep,
					"Expected no 'Check compile-agentic version' step in activation job but it was found")
			}
		})
	}
}
