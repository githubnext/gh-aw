//go:build integration

package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAPMDependenciesArrayFormat(t *testing.T) {
	tmpDir := testutil.TempDir(t, "apm-array-test")

	tests := []struct {
		name             string
		workflow         string
		expectedPackages []string
	}{
		{
			name: "Single APM package",
			workflow: `---
engine: copilot
on: workflow_dispatch
permissions:
  issues: read
  pull-requests: read
dependencies:
  - microsoft/apm-sample-package
---

Test single APM package
`,
			expectedPackages: []string{"- microsoft/apm-sample-package"},
		},
		{
			name: "Multiple APM packages",
			workflow: `---
engine: copilot
on: workflow_dispatch
permissions:
  issues: read
  pull-requests: read
dependencies:
  - microsoft/apm-sample-package
  - acme/custom-tools
  - org/another-package
---

Test multiple APM packages
`,
			expectedPackages: []string{
				"- microsoft/apm-sample-package",
				"- acme/custom-tools",
				"- org/another-package",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testFile := filepath.Join(tmpDir, "test-apm-array-"+strings.ReplaceAll(tt.name, " ", "-")+".md")
			err := os.WriteFile(testFile, []byte(tt.workflow), 0644)
			require.NoError(t, err, "Failed to write test file")

			compiler := NewCompiler()
			err = compiler.CompileWorkflow(testFile)
			require.NoError(t, err, "Compilation should succeed")

			lockFile := strings.Replace(testFile, ".md", ".lock.yml", 1)
			content, err := os.ReadFile(lockFile)
			require.NoError(t, err, "Failed to read lock file")

			lockContent := string(content)

			// Verify the APM setup step is present
			assert.Contains(t, lockContent, "Setup APM dependencies",
				"Lock file should contain APM setup step name")
			assert.Contains(t, lockContent, "microsoft/apm-action",
				"Lock file should use microsoft/apm-action")
			assert.Contains(t, lockContent, "dependencies:",
				"Lock file should have dependencies input")

			// Verify all expected package entries are present
			for _, expectedPkg := range tt.expectedPackages {
				assert.Contains(t, lockContent, expectedPkg,
					"Lock file should contain package: %s", expectedPkg)
			}
		})
	}
}

func TestAPMDependenciesObjectFormat(t *testing.T) {
	tmpDir := testutil.TempDir(t, "apm-object-test")

	workflow := `---
engine: copilot
on: workflow_dispatch
permissions:
  issues: read
  pull-requests: read
dependencies:
  apm:
    - microsoft/apm-sample-package
    - acme/custom-tools
---

Test APM object format
`
	testFile := filepath.Join(tmpDir, "test-apm-object.md")
	err := os.WriteFile(testFile, []byte(workflow), 0644)
	require.NoError(t, err, "Failed to write test file")

	compiler := NewCompiler()
	err = compiler.CompileWorkflow(testFile)
	require.NoError(t, err, "Compilation should succeed")

	lockFile := strings.Replace(testFile, ".md", ".lock.yml", 1)
	content, err := os.ReadFile(lockFile)
	require.NoError(t, err, "Failed to read lock file")

	lockContent := string(content)

	assert.Contains(t, lockContent, "Setup APM dependencies",
		"Lock file should contain APM setup step name")
	assert.Contains(t, lockContent, "microsoft/apm-action",
		"Lock file should use microsoft/apm-action")
	assert.Contains(t, lockContent, "- microsoft/apm-sample-package",
		"Lock file should contain first package")
	assert.Contains(t, lockContent, "- acme/custom-tools",
		"Lock file should contain second package")
}

func TestAPMDependenciesNoDependencies(t *testing.T) {
	tmpDir := testutil.TempDir(t, "apm-no-deps-test")

	workflow := `---
engine: copilot
on: workflow_dispatch
permissions:
  issues: read
  pull-requests: read
---

Test without APM dependencies
`
	testFile := filepath.Join(tmpDir, "test-no-apm.md")
	err := os.WriteFile(testFile, []byte(workflow), 0644)
	require.NoError(t, err, "Failed to write test file")

	compiler := NewCompiler()
	err = compiler.CompileWorkflow(testFile)
	require.NoError(t, err, "Compilation should succeed")

	lockFile := strings.Replace(testFile, ".md", ".lock.yml", 1)
	content, err := os.ReadFile(lockFile)
	require.NoError(t, err, "Failed to read lock file")

	lockContent := string(content)

	// Verify no APM setup step is emitted when no dependencies are specified
	assert.NotContains(t, lockContent, "microsoft/apm-action",
		"Lock file should not contain APM action when no dependencies are specified")
	assert.NotContains(t, lockContent, "Setup APM dependencies",
		"Lock file should not contain APM setup step when no dependencies are specified")
}

func TestAPMDependenciesWithAllEngines(t *testing.T) {
	tmpDir := testutil.TempDir(t, "apm-engines-test")

	engines := []string{"copilot", "claude", "codex"}

	for _, engine := range engines {
		t.Run(engine, func(t *testing.T) {
			workflow := `---
engine: ` + engine + `
on: workflow_dispatch
permissions:
  issues: read
  pull-requests: read
dependencies:
  - microsoft/apm-sample-package
---

Test APM with ` + engine + `
`
			testFile := filepath.Join(tmpDir, "test-apm-"+engine+".md")
			err := os.WriteFile(testFile, []byte(workflow), 0644)
			require.NoError(t, err, "Failed to write test file")

			compiler := NewCompiler()
			err = compiler.CompileWorkflow(testFile)
			require.NoError(t, err, "Compilation should succeed for engine: %s", engine)

			lockFile := strings.Replace(testFile, ".md", ".lock.yml", 1)
			content, err := os.ReadFile(lockFile)
			require.NoError(t, err, "Failed to read lock file")

			lockContent := string(content)

			assert.Contains(t, lockContent, "microsoft/apm-action",
				"Lock file should use microsoft/apm-action for engine: %s", engine)
			assert.Contains(t, lockContent, "- microsoft/apm-sample-package",
				"Lock file should contain package for engine: %s", engine)
		})
	}
}
