package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/githubnext/gh-aw/pkg/testutil"
)

func TestAllowGitHubReferencesEnvVar(t *testing.T) {
	tests := []struct {
		name                  string
		workflow              string
		expectedEnvVarPresent bool
		expectedEnvVarValue   string
	}{
		{
			name: "env var set with single repo",
			workflow: `---
on: push
engine: copilot
strict: false
permissions:
  contents: read
  issues: write
safe-outputs:
  allow-github-references: ["repo"]
  create-issue: {}
---

# Test Workflow

Test workflow with allow-github-references.
`,
			expectedEnvVarPresent: true,
			expectedEnvVarValue:   "repo",
		},
		{
			name: "env var set with multiple repos",
			workflow: `---
on: push
engine: copilot
strict: false
permissions:
  contents: read
  issues: write
safe-outputs:
  allow-github-references: ["repo", "org/repo2", "org/repo3"]
  create-issue: {}
---

# Test Workflow

Test workflow with multiple allowed repos.
`,
			expectedEnvVarPresent: true,
			expectedEnvVarValue:   "repo,org/repo2,org/repo3",
		},
		{
			name: "env var not set when allow-github-references is absent",
			workflow: `---
on: push
engine: copilot
strict: false
permissions:
  contents: read
  issues: write
safe-outputs:
  create-issue: {}
---

# Test Workflow

Test workflow without allow-github-references.
`,
			expectedEnvVarPresent: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp directory for test files
			tmpDir := testutil.TempDir(t, "allow-github-refs-test")

			// Write workflow file
			testFile := filepath.Join(tmpDir, "test-workflow.md")
			if err := os.WriteFile(testFile, []byte(tt.workflow), 0644); err != nil {
				t.Fatal(err)
			}

			// Compile the workflow
			compiler := NewCompiler(false, "", "test")
			if err := compiler.CompileWorkflow(testFile); err != nil {
				t.Fatalf("Failed to compile workflow: %v", err)
			}

			// Read the generated lock file
			lockFile := strings.Replace(testFile, ".md", ".lock.yml", 1)
			lockContent, err := os.ReadFile(lockFile)
			if err != nil {
				t.Fatalf("Failed to read lock file: %v", err)
			}

			lockStr := string(lockContent)

			// Check for env var presence
			if tt.expectedEnvVarPresent {
				if !strings.Contains(lockStr, "GH_AW_ALLOWED_GITHUB_REFS:") {
					t.Error("Expected GH_AW_ALLOWED_GITHUB_REFS environment variable in lock file")
				}

				// Verify the value
				expectedLine := `GH_AW_ALLOWED_GITHUB_REFS: "` + tt.expectedEnvVarValue + `"`
				if !strings.Contains(lockStr, expectedLine) {
					t.Errorf("Expected GH_AW_ALLOWED_GITHUB_REFS value to be %q, but it was not found in lock file", tt.expectedEnvVarValue)
				}
			} else {
				if strings.Contains(lockStr, "GH_AW_ALLOWED_GITHUB_REFS:") {
					t.Error("Expected no GH_AW_ALLOWED_GITHUB_REFS environment variable in lock file")
				}
			}
		})
	}
}
