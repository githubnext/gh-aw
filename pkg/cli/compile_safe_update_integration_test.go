//go:build integration

package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// safeUpdateWorkflowWithSecret is a minimal workflow that includes a custom job
// with a non-GITHUB_TOKEN secret in its environment.  The secret reference will
// appear in the compiled YAML body and be detected by CollectSecretReferences.
const safeUpdateWorkflowWithSecret = `---
name: Safe Update Secret Test
on:
  workflow_dispatch:
permissions:
  contents: read
engine: copilot
jobs:
  secret-job:
    runs-on: ubuntu-latest
    needs: [activation]
    env:
      MY_API_SECRET: ${{ secrets.MY_API_SECRET }}
    steps:
      - run: echo "hello"
---

# Safe Update Secret Test

Test workflow that uses a custom secret in a custom job.
`

// safeUpdateWorkflowWithCustomAction is a minimal workflow that includes a custom
// job using a non-actions/* action reference.  The uses: line will appear in the
// compiled YAML body and be detected by CollectActionReferences.
const safeUpdateWorkflowWithCustomAction = `---
name: Safe Update Action Test
on:
  workflow_dispatch:
permissions:
  contents: read
engine: copilot
jobs:
  action-job:
    runs-on: ubuntu-latest
    needs: [activation]
    steps:
      - uses: my-org/custom-action@v1
---

# Safe Update Action Test

Test workflow that uses a custom action in a custom job.
`

// safeUpdateWorkflowBasic is a minimal workflow that uses only GITHUB_TOKEN and
// actions/* actions.  Safe update mode should allow it on a first compile.
const safeUpdateWorkflowBasic = `---
name: Safe Update Basic Test
on:
  workflow_dispatch:
permissions:
  contents: read
engine: copilot
---

# Safe Update Basic Test

Test workflow that uses only GITHUB_TOKEN.
`

// manifestWithAPISecret is a minimal lock file content containing a gh-aw-manifest
// that pre-approves MY_API_SECRET.  Writing this to the lock file path
// before compilation simulates a workflow that was previously compiled and approved.
func manifestLockFileWithSecret(secretName string) string {
	return fmt.Sprintf(
		"# gh-aw-metadata: {\"schema_version\":\"v3\",\"frontmatter_hash\":\"abc\",\"agent_id\":\"copilot\"}\n"+
			"# gh-aw-manifest: {\"version\":1,\"secrets\":[\"%s\",\"GITHUB_TOKEN\"],\"actions\":[]}\n"+
			"name: placeholder\n",
		secretName,
	)
}

// TestSafeUpdateWarnOnNewSecretOnFirstCompile verifies that --safe-update emits a
// warning (not an error) when a first compilation introduces a non-GITHUB_TOKEN
// secret and no prior manifest exists.  The compilation must succeed and the lock
// file must be written so that the agent receives the actionable warning.
func TestSafeUpdateWarnsOnNewSecretOnFirstCompile(t *testing.T) {
	setup := setupIntegrationTest(t)
	defer setup.cleanup()

	workflowPath := filepath.Join(setup.workflowsDir, "safe-update-secret.md")
	require.NoError(t, os.WriteFile(workflowPath, []byte(safeUpdateWorkflowWithSecret), 0o644),
		"should write workflow file")

	cmd := exec.Command(setup.binaryPath, "compile", workflowPath, "--safe-update")
	cmd.Env = append(os.Environ(), "GH_AW_ACTION_MODE=release")
	output, err := cmd.CombinedOutput()
	outputStr := string(output)

	// Compilation must succeed (warning, not error)
	assert.NoError(t, err, "compile should succeed in safe update mode (violation emits warning, not error)\nOutput:\n%s", outputStr)
	// Warning must reference the violation and request a security review
	assert.Contains(t, outputStr, "safe update mode", "output should mention safe update mode")
	assert.Contains(t, outputStr, "MY_API_SECRET", "warning should name the offending secret")
	assert.Contains(t, outputStr, "SECURITY REVIEW REQUIRED", "warning should request a security review")
	// Lock file must still be written
	lockFilePath := filepath.Join(setup.workflowsDir, "safe-update-secret.lock.yml")
	_, statErr := os.Stat(lockFilePath)
	assert.NoError(t, statErr, "lock file should be written even when a safe-update warning is emitted")
	t.Logf("Safe update correctly emitted warning for new secret.\nOutput:\n%s", outputStr)
}

// TestSafeUpdateWarnOnNewCustomActionOnFirstCompile verifies that --safe-update emits
// a warning (not an error) when a first compilation introduces a non-actions/* action
// reference and no prior manifest exists.  Compilation must succeed.
func TestSafeUpdateWarnsOnNewCustomActionOnFirstCompile(t *testing.T) {
	setup := setupIntegrationTest(t)
	defer setup.cleanup()

	workflowPath := filepath.Join(setup.workflowsDir, "safe-update-action.md")
	require.NoError(t, os.WriteFile(workflowPath, []byte(safeUpdateWorkflowWithCustomAction), 0o644),
		"should write workflow file")

	cmd := exec.Command(setup.binaryPath, "compile", workflowPath, "--safe-update")
	cmd.Env = append(os.Environ(), "GH_AW_ACTION_MODE=release")
	output, err := cmd.CombinedOutput()
	outputStr := string(output)

	// Compilation must succeed (warning, not error)
	assert.NoError(t, err, "compile should succeed in safe update mode (violation emits warning, not error)\nOutput:\n%s", outputStr)
	// Warning must reference the violation and request a security review
	assert.Contains(t, outputStr, "safe update mode", "output should mention safe update mode")
	assert.Contains(t, outputStr, "my-org/custom-action", "warning should name the offending action")
	assert.Contains(t, outputStr, "SECURITY REVIEW REQUIRED", "warning should request a security review")
	// Lock file must still be written
	lockFilePath := filepath.Join(setup.workflowsDir, "safe-update-action.lock.yml")
	_, statErr := os.Stat(lockFilePath)
	assert.NoError(t, statErr, "lock file should be written even when a safe-update warning is emitted")
	t.Logf("Safe update correctly emitted warning for new custom action.\nOutput:\n%s", outputStr)
}

// TestSafeUpdateAllowsKnownSecretWithPriorManifest verifies that --safe-update
// allows a compilation when the secret is already recorded in the prior manifest
// embedded in the existing lock file.
//
// The test uses a two-step approach: first compile without --safe-update to produce
// a complete lock file with the full manifest (including engine-internal secrets and
// actions), then compile again with --safe-update. Since nothing changed between the
// two compilations, no new secrets or actions are introduced and the second compile
// must succeed.
func TestSafeUpdateAllowsKnownSecretWithPriorManifest(t *testing.T) {
	setup := setupIntegrationTest(t)
	defer setup.cleanup()

	workflowPath := filepath.Join(setup.workflowsDir, "safe-update-known-secret.md")
	require.NoError(t, os.WriteFile(workflowPath, []byte(safeUpdateWorkflowWithSecret), 0o644),
		"should write workflow file")

	// Step 1: Compile without --safe-update to generate the full lock file + manifest.
	// (Engine-internal secrets such as COPILOT_GITHUB_TOKEN are also captured here.)
	step1Cmd := exec.Command(setup.binaryPath, "compile", workflowPath)
	step1Cmd.Env = append(os.Environ(), "GH_AW_ACTION_MODE=release")
	step1Out, step1Err := step1Cmd.CombinedOutput()
	require.NoError(t, step1Err,
		"first compilation (no --safe-update) should succeed\nOutput:\n%s", string(step1Out))

	lockFilePath := filepath.Join(setup.workflowsDir, "safe-update-known-secret.lock.yml")
	lockContent, readErr := os.ReadFile(lockFilePath)
	require.NoError(t, readErr, "should read lock file after first compile")
	require.Contains(t, string(lockContent), "MY_API_SECRET",
		"lock file manifest should include MY_API_SECRET after first compile")

	// Step 2: Compile the identical workflow with --safe-update. The lock file from
	// step 1 acts as the prior manifest. Nothing changed, so this must succeed.
	step2Cmd := exec.Command(setup.binaryPath, "compile", workflowPath, "--safe-update")
	step2Cmd.Env = append(os.Environ(), "GH_AW_ACTION_MODE=release")
	output, err := step2Cmd.CombinedOutput()
	outputStr := string(output)

	assert.NoError(t, err, "compile should succeed when the secret is in the prior manifest\nOutput:\n%s", outputStr)
	t.Logf("Safe update correctly allowed known secret.\nOutput:\n%s", outputStr)
}

// TestSafeUpdateAllowsGitHubTokenOnFirstCompile verifies that --safe-update allows
// a compilation that introduces no new non-GITHUB_TOKEN secrets compared to a
// previously recorded manifest.
//
// Uses a two-step approach: step 1 compiles without --safe-update to record the
// baseline manifest (which includes engine-internal secrets in release mode); step 2
// recompiles the same workflow with --safe-update and expects success because the
// manifest is unchanged.
func TestSafeUpdateAllowsGitHubTokenOnFirstCompile(t *testing.T) {
	setup := setupIntegrationTest(t)
	defer setup.cleanup()

	workflowPath := filepath.Join(setup.workflowsDir, "safe-update-basic.md")
	require.NoError(t, os.WriteFile(workflowPath, []byte(safeUpdateWorkflowBasic), 0o644),
		"should write workflow file")

	// Step 1: Establish the baseline manifest with a normal compile.
	step1Cmd := exec.Command(setup.binaryPath, "compile", workflowPath)
	step1Cmd.Env = append(os.Environ(), "GH_AW_ACTION_MODE=release")
	step1Out, step1Err := step1Cmd.CombinedOutput()
	require.NoError(t, step1Err,
		"first compilation (no --safe-update) should succeed\nOutput:\n%s", string(step1Out))

	lockFilePath := filepath.Join(setup.workflowsDir, "safe-update-basic.lock.yml")
	lockContent, readErr := os.ReadFile(lockFilePath)
	require.NoError(t, readErr, "should read lock file after first compile")
	require.Contains(t, string(lockContent), "gh-aw-manifest:",
		"lock file should contain a gh-aw-manifest header after first compile")

	// Step 2: Re-compile with --safe-update. No secrets were added so this must succeed.
	step2Cmd := exec.Command(setup.binaryPath, "compile", workflowPath, "--safe-update")
	step2Cmd.Env = append(os.Environ(), "GH_AW_ACTION_MODE=release")
	output, err := step2Cmd.CombinedOutput()
	outputStr := string(output)

	assert.NoError(t, err, "compile should succeed when no new secrets are introduced\nOutput:\n%s", outputStr)

	// Verify the manifest is still present in the (re-)generated lock file.
	updatedLock, readErr2 := os.ReadFile(lockFilePath)
	require.NoError(t, readErr2, "should read updated lock file")
	assert.Contains(t, string(updatedLock), "gh-aw-manifest:", "lock file should still contain a gh-aw-manifest header")
	assert.NotContains(t, string(updatedLock), "MY_API_SECRET", "lock file manifest should not contain unapproved secrets")
	t.Logf("Safe update correctly allowed GITHUB_TOKEN-only workflow.\nOutput:\n%s", outputStr)
}

// safeUpdateWorkflowNonStrict is a minimal workflow that explicitly opts out of
// strict mode.  Because safe update mode == strict mode, setting strict: false
// also disables safe update enforcement, letting new secrets compile freely.
const safeUpdateWorkflowNonStrict = `---
name: Non Strict Workflow
on:
  workflow_dispatch:
permissions:
  contents: read
engine: copilot
strict: false
jobs:
  secret-job:
    runs-on: ubuntu-latest
    needs: [activation]
    env:
      MY_API_SECRET: ${{ secrets.MY_API_SECRET }}
    steps:
      - run: echo "hello"
---

# Non Strict Workflow

Workflow with strict: false, which also disables safe update enforcement.
`

// TestSafeUpdateNoFlagAllowsNewSecret verifies that when strict mode is disabled
// (strict: false in frontmatter) safe update enforcement is also disabled — new
// secrets compile without any safe-update warning.
func TestSafeUpdateNoFlagAllowsNewSecret(t *testing.T) {
	setup := setupIntegrationTest(t)
	defer setup.cleanup()

	workflowPath := filepath.Join(setup.workflowsDir, "no-safe-update.md")
	require.NoError(t, os.WriteFile(workflowPath, []byte(safeUpdateWorkflowNonStrict), 0o644),
		"should write workflow file")

	// strict: false in frontmatter disables strict mode and therefore safe update mode.
	cmd := exec.Command(setup.binaryPath, "compile", workflowPath)
	cmd.Env = append(os.Environ(), "GH_AW_ACTION_MODE=release")
	output, err := cmd.CombinedOutput()
	outputStr := string(output)

	assert.NoError(t, err, "compile with strict: false should succeed without safe update warning\nOutput:\n%s", outputStr)
	assert.False(t, strings.Contains(outputStr, "safe update mode"),
		"output should not mention safe update mode when strict mode is disabled")
	t.Logf("Compilation without safe update enforcement succeeded as expected.\nOutput:\n%s", outputStr)
}

// --- Transitive import tests -------------------------------------------------
//
// The following tests verify that the gh-aw-manifest embedded in a compiled
// lock file captures the *transitive closure* of all secrets and actions
// referenced by the workflow, including those introduced by imported shared
// workflow files.

// safeUpdateSharedMCPConfig is a shared workflow file that declares an MCP
// server whose env references a non-GITHUB_TOKEN secret.  It is imported by
// safeUpdateWorkflowWithImport below.
const safeUpdateSharedMCPConfig = `---
mcp-servers:
  shared-mcp:
    container: "mcp/shared"
    env:
      SHARED_API_KEY: "${{ secrets.SHARED_API_KEY }}"
    allowed:
      - "shared_op"
---
`

// safeUpdateWorkflowWithImport is a workflow that imports safeUpdateSharedMCPConfig.
// After compilation the manifest should include SHARED_API_KEY even though
// the secret is declared in the imported file, not the top-level workflow.
const safeUpdateWorkflowWithImport = `---
name: Safe Update Import Test
on:
  workflow_dispatch:
permissions:
  contents: read
engine: copilot
imports:
  - shared/shared-mcp.md
---

# Safe Update Import Test

Test workflow that imports a shared config containing a secret.
`

// safeUpdateSharedLevel2Config is a second-level shared workflow that itself
// imports safeUpdateSharedLevel3Config.  This is used to verify 3-level
// transitive import chains.
const safeUpdateSharedLevel2Config = `---
imports:
  - shared/level3.md
---
`

// safeUpdateSharedLevel3Config is a third-level shared workflow that declares
// an MCP server env with a deeply nested secret.
const safeUpdateSharedLevel3Config = `---
mcp-servers:
  deep-mcp:
    container: "mcp/deep"
    env:
      DEEP_NESTED_SECRET: "${{ secrets.DEEP_NESTED_SECRET }}"
    allowed:
      - "deep_op"
---
`

// safeUpdateWorkflowWithTransitiveImport is a workflow that imports level2,
// which imports level3.  The manifest must include DEEP_NESTED_SECRET.
const safeUpdateWorkflowWithTransitiveImport = `---
name: Safe Update Transitive Import Test
on:
  workflow_dispatch:
permissions:
  contents: read
engine: copilot
imports:
  - shared/level2.md
---

# Safe Update Transitive Import Test

Test workflow that uses a 3-level transitive import chain.
`

// writeSharedImportFiles is a helper that creates the shared/ directory and
// writes the shared import files for import-related integration tests.
func writeSharedImportFiles(t *testing.T, workflowsDir string) {
	t.Helper()
	sharedDir := filepath.Join(workflowsDir, "shared")
	require.NoError(t, os.MkdirAll(sharedDir, 0o755), "should create shared dir")
	require.NoError(t,
		os.WriteFile(filepath.Join(sharedDir, "shared-mcp.md"), []byte(safeUpdateSharedMCPConfig), 0o644),
		"should write shared MCP config")
	require.NoError(t,
		os.WriteFile(filepath.Join(sharedDir, "level2.md"), []byte(safeUpdateSharedLevel2Config), 0o644),
		"should write level-2 shared config")
	require.NoError(t,
		os.WriteFile(filepath.Join(sharedDir, "level3.md"), []byte(safeUpdateSharedLevel3Config), 0o644),
		"should write level-3 shared config")
}

// TestSafeUpdateManifestIncludesImportedSecret verifies that compiling a
// workflow that imports a shared config containing a secret embeds that secret
// in the gh-aw-manifest of the generated lock file.
func TestSafeUpdateManifestIncludesImportedSecret(t *testing.T) {
	setup := setupIntegrationTest(t)
	defer setup.cleanup()

	writeSharedImportFiles(t, setup.workflowsDir)

	workflowPath := filepath.Join(setup.workflowsDir, "import-secret.md")
	require.NoError(t, os.WriteFile(workflowPath, []byte(safeUpdateWorkflowWithImport), 0o644),
		"should write workflow file")

	// Compile without --safe-update so we can inspect the manifest freely.
	cmd := exec.Command(setup.binaryPath, "compile", workflowPath)
	cmd.Env = append(os.Environ(), "GH_AW_ACTION_MODE=release")
	output, err := cmd.CombinedOutput()
	outputStr := string(output)

	require.NoError(t, err, "compilation should succeed\nOutput:\n%s", outputStr)

	lockPath := filepath.Join(setup.workflowsDir, "import-secret.lock.yml")
	lockContent, readErr := os.ReadFile(lockPath)
	require.NoError(t, readErr, "should read lock file")

	assert.Contains(t, string(lockContent), "SHARED_API_KEY",
		"manifest should include the secret from the imported shared config")
	lines := strings.Split(string(lockContent), "\n")
	if len(lines) > 1 {
		t.Logf("Manifest correctly includes imported secret.\nLock file header:\n%s", lines[1])
	}
}

// TestSafeUpdateRejectsNewSecretFromImport verifies that --safe-update emits a warning
// when the new secret is introduced via an imported workflow rather than directly in
// the top-level workflow frontmatter.  Compilation must still succeed.
func TestSafeUpdateWarnsOnNewSecretFromImport(t *testing.T) {
	setup := setupIntegrationTest(t)
	defer setup.cleanup()

	writeSharedImportFiles(t, setup.workflowsDir)

	workflowPath := filepath.Join(setup.workflowsDir, "import-safe-update.md")
	require.NoError(t, os.WriteFile(workflowPath, []byte(safeUpdateWorkflowWithImport), 0o644),
		"should write workflow file")

	// No prior lock file — safe update treats this as an empty manifest.
	cmd := exec.Command(setup.binaryPath, "compile", workflowPath, "--safe-update")
	cmd.Env = append(os.Environ(), "GH_AW_ACTION_MODE=release")
	output, err := cmd.CombinedOutput()
	outputStr := string(output)

	// Compilation must succeed (warning, not error)
	assert.NoError(t, err,
		"compile should succeed (violation emits warning, not error) when import introduces a new secret under --safe-update\nOutput:\n%s", outputStr)
	assert.Contains(t, outputStr, "safe update mode",
		"warning should mention safe update mode")
	assert.Contains(t, outputStr, "SHARED_API_KEY",
		"warning should name the secret that came from the import")
	assert.Contains(t, outputStr, "SECURITY REVIEW REQUIRED",
		"warning should request a security review")
	t.Logf("Safe update correctly warned about secret introduced via import.\nOutput:\n%s", outputStr)
}

// TestSafeUpdateAllowsImportedSecretWithPriorManifest verifies that --safe-update
// allows compilation when the secret introduced by an import is already recorded
// in the prior lock file's gh-aw-manifest.
//
// The test uses a two-step approach to avoid hard-coding the full set of
// engine-required secrets in the prior manifest:
//  1. Compile without --safe-update to produce a lock file with the full manifest.
//  2. Compile again with --safe-update; the existing lock file (from step 1) acts as
//     the prior manifest and the compilation should succeed since no new
//     secrets or actions are being introduced.
func TestSafeUpdateAllowsImportedSecretWithPriorManifest(t *testing.T) {
	setup := setupIntegrationTest(t)
	defer setup.cleanup()

	writeSharedImportFiles(t, setup.workflowsDir)

	workflowPath := filepath.Join(setup.workflowsDir, "import-approved.md")
	require.NoError(t, os.WriteFile(workflowPath, []byte(safeUpdateWorkflowWithImport), 0o644),
		"should write workflow file")

	// Step 1: Compile without --safe-update to generate the lock file + manifest.
	step1Cmd := exec.Command(setup.binaryPath, "compile", workflowPath)
	step1Cmd.Env = append(os.Environ(), "GH_AW_ACTION_MODE=release")
	step1Out, step1Err := step1Cmd.CombinedOutput()
	require.NoError(t, step1Err,
		"first compilation (no --safe-update) should succeed\nOutput:\n%s", string(step1Out))

	// Verify the lock file was created and contains the manifest.
	lockPath := filepath.Join(setup.workflowsDir, "import-approved.lock.yml")
	lockContent, readErr := os.ReadFile(lockPath)
	require.NoError(t, readErr, "should read lock file after first compile")
	require.Contains(t, string(lockContent), "SHARED_API_KEY",
		"lock file manifest should include the imported secret after first compile")

	// Step 2: Compile again with --safe-update. The lock file from step 1 serves
	// as the prior manifest. No new secrets or actions are introduced so this must succeed.
	step2Cmd := exec.Command(setup.binaryPath, "compile", workflowPath, "--safe-update")
	step2Cmd.Env = append(os.Environ(), "GH_AW_ACTION_MODE=release")
	step2Out, step2Err := step2Cmd.CombinedOutput()
	outputStr := string(step2Out)

	assert.NoError(t, step2Err,
		"second compilation (with --safe-update) should succeed when imported secret is already in the manifest\nOutput:\n%s", outputStr)
	t.Logf("Safe update correctly allowed pre-approved imported secret.\nOutput:\n%s", outputStr)
}

// TestSafeUpdateManifestIncludesTransitivelyImportedSecret verifies that the
// gh-aw-manifest includes secrets declared in a *transitively* imported workflow
// (A imports B, B imports C, C declares the secret).  This confirms that the
// manifest computation covers the full transitive closure of imports.
func TestSafeUpdateManifestIncludesTransitivelyImportedSecret(t *testing.T) {
	setup := setupIntegrationTest(t)
	defer setup.cleanup()

	writeSharedImportFiles(t, setup.workflowsDir)

	workflowPath := filepath.Join(setup.workflowsDir, "transitive-import.md")
	require.NoError(t,
		os.WriteFile(workflowPath, []byte(safeUpdateWorkflowWithTransitiveImport), 0o644),
		"should write workflow file")

	// Compile without --safe-update so we can freely inspect the manifest.
	cmd := exec.Command(setup.binaryPath, "compile", workflowPath)
	cmd.Env = append(os.Environ(), "GH_AW_ACTION_MODE=release")
	output, err := cmd.CombinedOutput()
	outputStr := string(output)

	require.NoError(t, err, "compilation should succeed\nOutput:\n%s", outputStr)

	lockPath := filepath.Join(setup.workflowsDir, "transitive-import.lock.yml")
	lockContent, readErr := os.ReadFile(lockPath)
	require.NoError(t, readErr, "should read lock file")

	assert.Contains(t, string(lockContent), "DEEP_NESTED_SECRET",
		"manifest should include the secret from the transitively imported (level-3) shared config")
	lines := strings.Split(string(lockContent), "\n")
	if len(lines) > 1 {
		t.Logf("Manifest correctly includes transitively imported secret.\nLock file header:\n%s", lines[1])
	}
}

// TestSafeUpdateRejectsTransitivelyImportedSecretOnFirstCompile verifies that
// --safe-update emits a warning when the new secret is introduced via a transitive
// import (A imports B, B imports C, C declares the secret).  Compilation must succeed.
func TestSafeUpdateWarnsOnTransitivelyImportedSecretOnFirstCompile(t *testing.T) {
	setup := setupIntegrationTest(t)
	defer setup.cleanup()

	writeSharedImportFiles(t, setup.workflowsDir)

	workflowPath := filepath.Join(setup.workflowsDir, "transitive-safe-update.md")
	require.NoError(t,
		os.WriteFile(workflowPath, []byte(safeUpdateWorkflowWithTransitiveImport), 0o644),
		"should write workflow file")

	// No prior lock file — safe update treats this as an empty manifest.
	cmd := exec.Command(setup.binaryPath, "compile", workflowPath, "--safe-update")
	cmd.Env = append(os.Environ(), "GH_AW_ACTION_MODE=release")
	output, err := cmd.CombinedOutput()
	outputStr := string(output)

	// Compilation must succeed (warning, not error)
	assert.NoError(t, err,
		"compile should succeed (violation emits warning, not error) when a transitive import introduces a new secret under --safe-update\nOutput:\n%s", outputStr)
	assert.Contains(t, outputStr, "safe update mode",
		"warning should mention safe update mode")
	assert.Contains(t, outputStr, "DEEP_NESTED_SECRET",
		"warning should name the secret that came from the transitive import")
	assert.Contains(t, outputStr, "SECURITY REVIEW REQUIRED",
		"warning should request a security review")
	t.Logf("Safe update correctly warned about secret from transitive import.\nOutput:\n%s", outputStr)
}
