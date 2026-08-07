//go:build integration

package workflow

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/github/gh-aw/pkg/stringutil"
	"github.com/github/gh-aw/pkg/testutil"
)

// assertScriptUsesRequire asserts that the compiled lock file content sets the
// GH_AW_TRACKER_ID environment variable and loads scripts via require() (file
// mode, not inline).
func assertScriptUsesRequire(t *testing.T, contentStr string) {
	t.Helper()
	assert.Contains(t, contentStr, "GH_AW_TRACKER_ID", "expected GH_AW_TRACKER_ID environment variable to be set")
	assert.Contains(t, contentStr, "require(", "expected scripts to be loaded using require()")
}

func TestTrackerIDIntegration(t *testing.T) {
	tests := []struct {
		name               string
		workflowContent    string
		shouldCompile      bool
		shouldHaveEnvVar   bool
		shouldHaveInScript bool
		expectedTrackerID  string
	}{
		{
			name: "Workflow with valid tracker-id",
			workflowContent: `---
on: workflow_dispatch
permissions:
  contents: read
tracker-id: test-fp-12345
safe-outputs:
  create-issue:
---

# Test Tracker ID

Create a test issue.
`,
			shouldCompile:      true,
			shouldHaveEnvVar:   true,
			shouldHaveInScript: true,
			expectedTrackerID:  "test-fp-12345",
		},
		{
			name: "Workflow without tracker-id",
			workflowContent: `---
on: workflow_dispatch
permissions:
  contents: read
safe-outputs:
  create-issue:
---

# Test No Tracker ID

Create a test issue without tracker-id.
`,
			shouldCompile:      true,
			shouldHaveEnvVar:   false,
			shouldHaveInScript: false,
		},
		{
			name: "Workflow with tracker-id in pull request",
			workflowContent: `---
on: push
permissions:
  contents: read
tracker-id: pr-tracker-123
safe-outputs:
  create-pull-request:
---

# Test PR Tracker ID

Create a pull request.
`,
			shouldCompile:      true,
			shouldHaveEnvVar:   true,
			shouldHaveInScript: true,
			expectedTrackerID:  "pr-tracker-123",
		},
		{
			name: "Workflow with tracker-id and multiple safe-outputs",
			workflowContent: `---
on: workflow_dispatch
permissions:
  contents: read
tracker-id: multi-output-1
safe-outputs:
  create-issue:
  create-pull-request:
---

# Test Multiple Safe Outputs

Create an issue and a pull request.
`,
			shouldCompile:      true,
			shouldHaveEnvVar:   true,
			shouldHaveInScript: true,
			expectedTrackerID:  "multi-output-1",
		},
		{
			name: "Workflow with too-short tracker-id",
			workflowContent: `---
on: workflow_dispatch
permissions:
  contents: read
tracker-id: short
safe-outputs:
  create-issue:
---

# Test Short Tracker ID

Create a test issue.
`,
			shouldCompile: false,
		},
		{
			name: "Workflow with tracker-id containing spaces",
			workflowContent: `---
on: workflow_dispatch
permissions:
  contents: read
tracker-id: has spaces
safe-outputs:
  create-issue:
---

# Test Tracker ID With Spaces

Create a test issue.
`,
			shouldCompile: false,
		},
		{
			name: "Workflow with tracker-id containing invalid characters",
			workflowContent: `---
on: workflow_dispatch
permissions:
  contents: read
tracker-id: bad!chars!
safe-outputs:
  create-issue:
---

# Test Tracker ID With Invalid Characters

Create a test issue.
`,
			shouldCompile: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Each subtest gets its own tmpDir, which testutil.TempDir already
			// registers for cleanup via t.Cleanup, so generated workflow and
			// lock files are removed automatically even on early failures.
			tmpDir := testutil.TempDir(t, "test-*")
			workflowFile := filepath.Join(tmpDir, "test.md")
			require.NoError(t, os.WriteFile(workflowFile, []byte(tt.workflowContent), 0644))

			compiler := NewCompiler()
			// Use dev mode to test with local action paths
			compiler.SetActionMode(ActionModeDev)
			compiler.verbose = false

			err := compiler.CompileWorkflow(workflowFile)

			if tt.shouldCompile {
				require.NoError(t, err, "expected compilation to succeed")
			} else {
				require.Error(t, err, "expected compilation to fail")
				return
			}

			lockFile := stringutil.MarkdownToLockFile(workflowFile)

			content, err := os.ReadFile(lockFile)
			require.NoError(t, err, "failed to read lock file")

			contentStr := string(content)

			if tt.shouldHaveEnvVar {
				envVarLine := "GH_AW_TRACKER_ID: \"" + tt.expectedTrackerID + "\""
				assert.Contains(t, contentStr, envVarLine, "expected lock file to contain tracker-id env var")
			} else {
				// The JavaScript code will always read process.env.GH_AW_TRACKER_ID
				// but the environment variable should not be set
				assert.NotContains(t, contentStr, "GH_AW_TRACKER_ID: \"", "expected lock file to NOT set GH_AW_TRACKER_ID env var")
			}

			if tt.shouldHaveInScript {
				assertScriptUsesRequire(t, contentStr)
			}
		})
	}
}
