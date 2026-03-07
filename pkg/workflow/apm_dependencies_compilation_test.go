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

func TestAPMDependenciesCompilationSinglePackage(t *testing.T) {
	tmpDir := testutil.TempDir(t, "apm-deps-single-test")

	workflow := `---
engine: copilot
on: workflow_dispatch
permissions:
  issues: read
  pull-requests: read
dependencies:
  - microsoft/apm-sample-package
---

Test with a single APM dependency
`

	testFile := filepath.Join(tmpDir, "test-apm-single.md")
	err := os.WriteFile(testFile, []byte(workflow), 0644)
	require.NoError(t, err, "Failed to write test file")

	compiler := NewCompiler()
	err = compiler.CompileWorkflow(testFile)
	require.NoError(t, err, "Compilation should succeed")

	lockFile := strings.Replace(testFile, ".md", ".lock.yml", 1)
	content, err := os.ReadFile(lockFile)
	require.NoError(t, err, "Failed to read lock file")

	lockContent := string(content)

	assert.Contains(t, lockContent, "Install APM dependencies",
		"Lock file should contain APM dependencies step name")
	assert.Contains(t, lockContent, "microsoft/apm-action",
		"Lock file should reference the microsoft/apm-action action")
	assert.Contains(t, lockContent, "dependencies: |",
		"Lock file should use block scalar for dependencies input")
	assert.Contains(t, lockContent, "- microsoft/apm-sample-package",
		"Lock file should list the dependency package")
}

func TestAPMDependenciesCompilationMultiplePackages(t *testing.T) {
	tmpDir := testutil.TempDir(t, "apm-deps-multi-test")

	workflow := `---
engine: copilot
on: workflow_dispatch
permissions:
  issues: read
  pull-requests: read
dependencies:
  - microsoft/apm-sample-package
  - github/awesome-copilot/skills/review-and-refactor
  - anthropics/skills/skills/frontend-design
---

Test with multiple APM dependencies
`

	testFile := filepath.Join(tmpDir, "test-apm-multi.md")
	err := os.WriteFile(testFile, []byte(workflow), 0644)
	require.NoError(t, err, "Failed to write test file")

	compiler := NewCompiler()
	err = compiler.CompileWorkflow(testFile)
	require.NoError(t, err, "Compilation should succeed")

	lockFile := strings.Replace(testFile, ".md", ".lock.yml", 1)
	content, err := os.ReadFile(lockFile)
	require.NoError(t, err, "Failed to read lock file")

	lockContent := string(content)

	assert.Contains(t, lockContent, "Install APM dependencies",
		"Lock file should contain APM dependencies step name")
	assert.Contains(t, lockContent, "microsoft/apm-action",
		"Lock file should reference the microsoft/apm-action action")
	assert.Contains(t, lockContent, "- microsoft/apm-sample-package",
		"Lock file should include first dependency")
	assert.Contains(t, lockContent, "- github/awesome-copilot/skills/review-and-refactor",
		"Lock file should include second dependency")
	assert.Contains(t, lockContent, "- anthropics/skills/skills/frontend-design",
		"Lock file should include third dependency")
}

func TestAPMDependenciesCompilationNoDependencies(t *testing.T) {
	tmpDir := testutil.TempDir(t, "apm-deps-none-test")

	workflow := `---
engine: copilot
on: workflow_dispatch
permissions:
  issues: read
  pull-requests: read
---

Test without APM dependencies
`

	testFile := filepath.Join(tmpDir, "test-apm-none.md")
	err := os.WriteFile(testFile, []byte(workflow), 0644)
	require.NoError(t, err, "Failed to write test file")

	compiler := NewCompiler()
	err = compiler.CompileWorkflow(testFile)
	require.NoError(t, err, "Compilation should succeed")

	lockFile := strings.Replace(testFile, ".md", ".lock.yml", 1)
	content, err := os.ReadFile(lockFile)
	require.NoError(t, err, "Failed to read lock file")

	lockContent := string(content)

	assert.NotContains(t, lockContent, "Install APM dependencies",
		"Lock file should not contain APM dependencies step when no dependencies specified")
	assert.NotContains(t, lockContent, "microsoft/apm-action",
		"Lock file should not reference microsoft/apm-action when no dependencies specified")
}
