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

	// APM job should have the pack step (not the activation job)
	assert.Contains(t, lockContent, "Install and pack APM dependencies",
		"Lock file should contain APM pack step in APM job")
	assert.Contains(t, lockContent, "microsoft/apm-action",
		"Lock file should reference the microsoft/apm-action action")
	assert.Contains(t, lockContent, "- microsoft/apm-sample-package",
		"Lock file should list the dependency package")
	assert.Contains(t, lockContent, "id: apm_pack",
		"Lock file should have apm_pack step ID")
	assert.Contains(t, lockContent, "pack: 'true'",
		"Lock file should include pack input")
	assert.Contains(t, lockContent, "target: copilot",
		"Lock file should include target inferred from copilot engine")

	// APM artifact upload in APM job
	assert.Contains(t, lockContent, "Upload APM bundle artifact",
		"Lock file should upload APM bundle as separate artifact")
	assert.Contains(t, lockContent, "name: apm",
		"Lock file should name the APM artifact 'apm'")

	// APM job should exist with minimal permissions and needs activation
	assert.Contains(t, lockContent, "apm:",
		"Lock file should contain a dedicated APM job")
	assert.Contains(t, lockContent, "permissions: {}",
		"APM job should have minimal (empty) permissions")

	// Agent job should have download + restore steps
	assert.Contains(t, lockContent, "Download APM bundle artifact",
		"Lock file should download APM bundle in agent job")
	assert.Contains(t, lockContent, "Restore APM dependencies",
		"Lock file should contain APM restore step in agent job")
	assert.Contains(t, lockContent, "APM_BUNDLE_DIR: /tmp/gh-aw/apm-bundle",
		"Lock file should configure bundle directory for JS unpacker")
	assert.Contains(t, lockContent, "apm_unpack.cjs",
		"Lock file should use JS unpacker script")

	// Old install step should NOT appear
	assert.NotContains(t, lockContent, "Install APM dependencies",
		"Lock file should not contain the old install step name")
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

	assert.Contains(t, lockContent, "Install and pack APM dependencies",
		"Lock file should contain APM pack step")
	assert.Contains(t, lockContent, "microsoft/apm-action",
		"Lock file should reference the microsoft/apm-action action")
	assert.Contains(t, lockContent, "- microsoft/apm-sample-package",
		"Lock file should include first dependency")
	assert.Contains(t, lockContent, "- github/awesome-copilot/skills/review-and-refactor",
		"Lock file should include second dependency")
	assert.Contains(t, lockContent, "- anthropics/skills/skills/frontend-design",
		"Lock file should include third dependency")
	assert.Contains(t, lockContent, "Restore APM dependencies",
		"Lock file should contain APM restore step")
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

	assert.NotContains(t, lockContent, "Install and pack APM dependencies",
		"Lock file should not contain APM pack step when no dependencies specified")
	assert.NotContains(t, lockContent, "Restore APM dependencies",
		"Lock file should not contain APM restore step when no dependencies specified")
	assert.NotContains(t, lockContent, "microsoft/apm-action",
		"Lock file should not reference microsoft/apm-action when no dependencies specified")
}

func TestAPMDependenciesCompilationObjectFormatIsolated(t *testing.T) {
	tmpDir := testutil.TempDir(t, "apm-deps-isolated-test")

	workflow := `---
engine: copilot
on: workflow_dispatch
permissions:
  issues: read
  pull-requests: read
dependencies:
  packages:
    - microsoft/apm-sample-package
  isolated: true
---

Test with isolated APM dependencies
`

	testFile := filepath.Join(tmpDir, "test-apm-isolated.md")
	err := os.WriteFile(testFile, []byte(workflow), 0644)
	require.NoError(t, err, "Failed to write test file")

	compiler := NewCompiler()
	err = compiler.CompileWorkflow(testFile)
	require.NoError(t, err, "Compilation should succeed")

	lockFile := strings.Replace(testFile, ".md", ".lock.yml", 1)
	content, err := os.ReadFile(lockFile)
	require.NoError(t, err, "Failed to read lock file")

	lockContent := string(content)

	assert.Contains(t, lockContent, "Install and pack APM dependencies",
		"Lock file should contain APM pack step")
	assert.Contains(t, lockContent, "Restore APM dependencies",
		"Lock file should contain APM restore step")
	// Restore step uses the JS unpacker (isolated flag not required for JS implementation)
	assert.Contains(t, lockContent, "apm_unpack.cjs",
		"Lock file restore step should use the JS unpacker")
}

func TestAPMDependenciesCompilationClaudeEngineTarget(t *testing.T) {
	tmpDir := testutil.TempDir(t, "apm-deps-claude-test")

	workflow := `---
engine: claude
on: workflow_dispatch
permissions:
  issues: read
  pull-requests: read
dependencies:
  - microsoft/apm-sample-package
---

Test with Claude engine target inference
`

	testFile := filepath.Join(tmpDir, "test-apm-claude.md")
	err := os.WriteFile(testFile, []byte(workflow), 0644)
	require.NoError(t, err, "Failed to write test file")

	compiler := NewCompiler()
	err = compiler.CompileWorkflow(testFile)
	require.NoError(t, err, "Compilation should succeed")

	lockFile := strings.Replace(testFile, ".md", ".lock.yml", 1)
	content, err := os.ReadFile(lockFile)
	require.NoError(t, err, "Failed to read lock file")

	lockContent := string(content)

	assert.Contains(t, lockContent, "target: claude",
		"Lock file should use claude target for claude engine")
}

func TestAPMDependenciesCompilationWithEnv(t *testing.T) {
	tmpDir := testutil.TempDir(t, "apm-deps-env-test")

	workflow := `---
engine: copilot
on: workflow_dispatch
permissions:
  issues: read
  pull-requests: read
dependencies:
  packages:
    - microsoft/apm-sample-package
  env:
    MY_TOKEN: ${{ secrets.MY_TOKEN }}
    REGISTRY: https://registry.example.com
---

Test with env vars on APM pack step
`

	testFile := filepath.Join(tmpDir, "test-apm-env.md")
	err := os.WriteFile(testFile, []byte(workflow), 0644)
	require.NoError(t, err, "Failed to write test file")

	compiler := NewCompiler()
	err = compiler.CompileWorkflow(testFile)
	require.NoError(t, err, "Compilation should succeed")

	lockFile := strings.Replace(testFile, ".md", ".lock.yml", 1)
	content, err := os.ReadFile(lockFile)
	require.NoError(t, err, "Failed to read lock file")

	lockContent := string(content)

	assert.Contains(t, lockContent, "Install and pack APM dependencies",
		"Lock file should contain APM pack step")
	assert.Contains(t, lockContent, "MY_TOKEN:",
		"Lock file should include MY_TOKEN env var on pack step")
	assert.Contains(t, lockContent, "REGISTRY: https://registry.example.com",
		"Lock file should include REGISTRY env var on pack step")
}
