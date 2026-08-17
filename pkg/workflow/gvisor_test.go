//go:build !integration

package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGenerateGVisorInstallStep verifies that the gVisor install step references
// the correct shell script with the pinned version and required runner-guard suppression.
func TestGenerateGVisorInstallStep(t *testing.T) {
	step := generateGVisorInstallStep()
	require.NotEmpty(t, step, "gVisor install step must not be empty")

	content := strings.Join(step, "\n")

	assert.Contains(t, content, "Install gVisor (runsc)", "step should have a recognizable name")
	assert.Contains(t, content, "runner-guard:ignore RGS-012",
		"step should include runner-guard suppression for verified gVisor download false-positive")
	assert.Contains(t, content, "sudo_gvisor_install.sh", "step must reference the sudo gVisor install script")
	assert.Contains(t, content, "${RUNNER_TEMP}/gh-aw/actions/", "must use RUNNER_TEMP script path")
	assert.Contains(t, content, constants.DefaultGVisorVersion,
		"must pass pinned gVisor release version to the script")
	assert.NotContains(t, content, "latest", "must NOT reference the mutable 'latest' release")
}

// TestGVisorInstallScriptContent verifies that the sudo_gvisor_install.sh
// shell script exists and contains the expected key operations.
func TestGVisorInstallScriptContent(t *testing.T) {
	wd, err := os.Getwd()
	require.NoError(t, err)
	script := filepath.Join(wd, "..", "..", "actions", "setup", "sh", "sudo_gvisor_install.sh")
	content, err := os.ReadFile(script)
	require.NoError(t, err, "sudo_gvisor_install.sh must exist")
	s := string(content)

	// Must detect architecture dynamically, not hardcode amd64/arm64.
	assert.Contains(t, s, "uname -m", "must detect architecture via uname -m")
	assert.NotContains(t, s, "amd64", "must NOT remap architecture to amd64")
	assert.NotContains(t, s, "arm64", "must NOT remap architecture to arm64")

	// Both binaries must be downloaded and integrity-verified.
	assert.Contains(t, s, "runsc", "must download runsc binary")
	assert.Contains(t, s, "containerd-shim-runsc-v1", "must download containerd-shim-runsc-v1")
	assert.Contains(t, s, "storage.googleapis.com/gvisor", "must use official gVisor download URL")
	assert.Contains(t, s, "runsc.sha512", "must download SHA-512 for runsc")
	assert.Contains(t, s, "containerd-shim-runsc-v1.sha512", "must download SHA-512 for containerd-shim-runsc-v1")
	assert.Contains(t, s, "sha512sum -c", "must verify SHA-512 checksums before installing")

	// Must install with sudo (hence the sudo_ prefix).
	assert.Contains(t, s, "sudo install", "must install binaries with sudo")
	assert.Contains(t, s, "/usr/local/bin/runsc", "must install runsc to /usr/local/bin")
	assert.Contains(t, s, "sudo runsc install", "must register runsc with Docker via runsc install")

	// Must restart Docker (not reload).
	assert.Contains(t, s, "systemctl restart docker", "must restart Docker with systemctl restart")
	assert.NotContains(t, s, "systemctl reload docker", "must NOT use systemctl reload (breaks host-gateway DNS)")

	// Must verify the runtime works end-to-end.
	assert.Contains(t, s, "docker pull hello-world", "must pre-pull hello-world image")
	assert.Contains(t, s, "docker run --rm --runtime=runsc", "must verify gVisor runtime with a test container")
}

// TestGVisorInstallStepOrderInBuildNpmEngineInstallStepsWithAWF verifies that the
// gVisor install step is emitted BEFORE the AWF install step.
func TestGVisorInstallStepOrderInBuildNpmEngineInstallStepsWithAWF(t *testing.T) {
	workflowData := &WorkflowData{
		SandboxConfig: &SandboxConfig{
			Agent: &AgentSandboxConfig{
				ID:      "awf",
				Runtime: AgentRuntimeGVisor,
			},
		},
		NetworkPermissions: &NetworkPermissions{
			Firewall: &FirewallConfig{Enabled: true},
		},
	}

	steps := BuildNpmEngineInstallStepsWithAWF(nil, workflowData)
	require.NotEmpty(t, steps, "must generate installation steps")

	// Find index of gVisor step and AWF step.
	gvisorIdx := -1
	awfIdx := -1
	for i, step := range steps {
		content := strings.Join(step, "\n")
		if strings.Contains(content, "Install gVisor") {
			gvisorIdx = i
		}
		if strings.Contains(content, "install_awf_binary.sh") {
			awfIdx = i
		}
	}

	require.NotEqual(t, -1, gvisorIdx, "gVisor install step must be present")
	require.NotEqual(t, -1, awfIdx, "AWF install step must be present")
	assert.Less(t, gvisorIdx, awfIdx, "gVisor step must come BEFORE AWF install step")
}

// TestGVisorAWFConfigJSON verifies that sandbox.agent.runtime: gvisor causes
// containerRuntime: "gvisor" to appear in the AWF config JSON when the effective AWF
// version is at or above AWFContainerRuntimeMinVersion.
func TestGVisorAWFConfigJSON(t *testing.T) {
	config := AWFCommandConfig{
		EngineName:     "copilot",
		AllowedDomains: "github.com",
		WorkflowData: &WorkflowData{
			EngineConfig: &EngineConfig{ID: "copilot"},
			NetworkPermissions: &NetworkPermissions{
				// Pin to a version that supports containerRuntime (AWFContainerRuntimeMinVersion = v0.27.30).
				Firewall: &FirewallConfig{Enabled: true, Version: "v0.27.30"},
			},
			SandboxConfig: &SandboxConfig{
				Agent: &AgentSandboxConfig{
					ID:      "awf",
					Runtime: AgentRuntimeGVisor,
				},
			},
		},
	}

	jsonStr, err := BuildAWFConfigJSON(config)
	require.NoError(t, err)

	assert.Contains(t, jsonStr, `"containerRuntime":"gvisor"`,
		"AWF config JSON must include containerRuntime: gvisor when version supports it")
}

// TestGVisorAWFConfigJSONVersionGated verifies that containerRuntime is NOT emitted
// when the effective AWF version predates AWFContainerRuntimeMinVersion, even if
// sandbox.agent.runtime: gvisor is set.
func TestGVisorAWFConfigJSONVersionGated(t *testing.T) {
	config := AWFCommandConfig{
		EngineName:     "copilot",
		AllowedDomains: "github.com",
		WorkflowData: &WorkflowData{
			EngineConfig: &EngineConfig{ID: "copilot"},
			NetworkPermissions: &NetworkPermissions{
				// Pin to a version that predates containerRuntime support.
				Firewall: &FirewallConfig{Enabled: true, Version: "v0.27.29"},
			},
			SandboxConfig: &SandboxConfig{
				Agent: &AgentSandboxConfig{
					ID:      "awf",
					Runtime: AgentRuntimeGVisor,
				},
			},
		},
	}

	jsonStr, err := BuildAWFConfigJSON(config)
	require.NoError(t, err)

	assert.NotContains(t, jsonStr, `"containerRuntime"`,
		"containerRuntime must not be emitted when AWF version predates support")
}
func TestGVisorAWFConfigJSONAbsentByDefault(t *testing.T) {
	config := AWFCommandConfig{
		EngineName:     "copilot",
		AllowedDomains: "github.com",
		WorkflowData: &WorkflowData{
			EngineConfig: &EngineConfig{ID: "copilot"},
			NetworkPermissions: &NetworkPermissions{
				Firewall: &FirewallConfig{Enabled: true},
			},
			SandboxConfig: &SandboxConfig{
				Agent: &AgentSandboxConfig{
					ID: "awf",
				},
			},
		},
	}

	jsonStr, err := BuildAWFConfigJSON(config)
	require.NoError(t, err)

	assert.NotContains(t, jsonStr, `"containerRuntime"`,
		"containerRuntime must be absent when runtime is not configured")
}

// TestGVisorValidation_ArcDindIncompatible verifies that gVisor + arc-dind is a
// compile-time error.
func TestGVisorValidation_ArcDindIncompatible(t *testing.T) {
	workflowData := &WorkflowData{
		SandboxConfig: &SandboxConfig{
			Agent: &AgentSandboxConfig{
				ID:      "awf",
				Runtime: AgentRuntimeGVisor,
			},
		},
		RunnerConfig: &RunnerConfig{Topology: RunnerTopologyArcDind},
		NetworkPermissions: &NetworkPermissions{
			Firewall: &FirewallConfig{Enabled: true},
		},
		Tools: map[string]any{"github": map[string]any{"mode": "remote"}},
	}

	err := validateSandboxConfig(workflowData)
	require.Error(t, err, "gVisor + arc-dind must produce a compile-time error")
	require.ErrorContains(t, err, "arc-dind", "error must mention arc-dind")
	require.ErrorContains(t, err, "gvisor", "error must mention gvisor")
}

// TestGVisorValidation_RootlessProfileAllowed verifies that the gvisor runtime profile
// is valid. The gVisor install step invokes sudo in shell commands directly, while AWF
// itself keeps running rootless.
func TestGVisorValidation_RootlessProfileAllowed(t *testing.T) {
	workflowData := &WorkflowData{
		SandboxConfig: &SandboxConfig{
			Agent: &AgentSandboxConfig{
				ID:      "awf",
				Runtime: AgentRuntimeGVisor,
			},
		},
		NetworkPermissions: &NetworkPermissions{
			Firewall: &FirewallConfig{Enabled: true},
		},
		Tools: map[string]any{"github": map[string]any{"mode": "remote"}},
	}

	err := validateSandboxConfig(workflowData)
	require.NoError(t, err, "the gvisor runtime profile must be a valid combination")
}

// TestGVisorValidation_ValidCombination verifies that a valid gVisor configuration
// passes validation.
func TestGVisorValidation_ValidCombination(t *testing.T) {
	workflowData := &WorkflowData{
		SandboxConfig: &SandboxConfig{
			Agent: &AgentSandboxConfig{
				ID:      "awf",
				Runtime: AgentRuntimeGVisor,
			},
		},
		NetworkPermissions: &NetworkPermissions{
			Firewall: &FirewallConfig{Enabled: true},
		},
		Tools: map[string]any{"github": map[string]any{"mode": "remote"}},
	}

	err := validateSandboxConfig(workflowData)
	assert.NoError(t, err, "gVisor on a standard runner must pass validation")
}

// TestGVisorFrontmatterExtraction verifies end-to-end that a workflow with
// sandbox.agent.runtime: gvisor compiles correctly and produces the expected output.
func TestGVisorFrontmatterExtraction(t *testing.T) {
	workflowsDir := t.TempDir()

	markdown := `---
on:
  workflow_dispatch:
engine: copilot
strict: false
network:
  allowed:
    - "example.com"
sandbox:
  agent:
    id: awf
    runtime: gvisor
---

# Test gVisor Runtime
`

	testFile := filepath.Join(workflowsDir, "test-gvisor.md")
	err := os.WriteFile(testFile, []byte(markdown), 0644)
	require.NoError(t, err)

	compiler := NewCompiler()
	compiler.SetSkipValidation(false) // exercise AWF config schema validation; containerRuntime must pass
	err = compiler.CompileWorkflow(testFile)
	require.NoError(t, err, "compilation with runtime: gvisor must succeed")

	lockContent, err := os.ReadFile(filepath.Join(workflowsDir, "test-gvisor.lock.yml"))
	require.NoError(t, err)
	lockStr := string(lockContent)

	// gVisor install step must be present.
	assert.Contains(t, lockStr, "Install gVisor", "compiled workflow must include gVisor install step")

	// AWF install step must also be present.
	assert.Contains(t, lockStr, "Install AWF binary", "compiled workflow must include AWF install step")

	// containerRuntime: gvisor must appear with the default AWF version (v0.27.30),
	// which supports containerRuntime (AWFContainerRuntimeMinVersion = v0.27.30).
	assert.Contains(t, lockStr, `\"containerRuntime\":\"gvisor\"`,
		"containerRuntime must be emitted for default AWF version")

	// gVisor step must appear before AWF step.
	gvisorPos := strings.Index(lockStr, "Install gVisor")
	awfPos := strings.Index(lockStr, "Install AWF binary")
	assert.Less(t, gvisorPos, awfPos, "gVisor install step must precede AWF install step in compiled YAML")
}

// TestIsGVisorRuntime verifies the isGVisorRuntime helper.
func TestIsGVisorRuntime(t *testing.T) {
	t.Run("returns false for nil workflow data", func(t *testing.T) {
		assert.False(t, isGVisorRuntime(nil))
	})

	t.Run("returns false when no sandbox config", func(t *testing.T) {
		assert.False(t, isGVisorRuntime(&WorkflowData{}))
	})

	t.Run("returns false when runtime is not gvisor", func(t *testing.T) {
		assert.False(t, isGVisorRuntime(&WorkflowData{
			SandboxConfig: &SandboxConfig{
				Agent: &AgentSandboxConfig{ID: "awf"},
			},
		}))
	})

	t.Run("returns false when agent is disabled", func(t *testing.T) {
		assert.False(t, isGVisorRuntime(&WorkflowData{
			SandboxConfig: &SandboxConfig{
				Agent: &AgentSandboxConfig{
					ID:       "awf",
					Runtime:  AgentRuntimeGVisor,
					Disabled: true,
				},
			},
		}))
	})

	t.Run("returns true when runtime is gvisor", func(t *testing.T) {
		assert.True(t, isGVisorRuntime(&WorkflowData{
			SandboxConfig: &SandboxConfig{
				Agent: &AgentSandboxConfig{
					ID:      "awf",
					Runtime: AgentRuntimeGVisor,
				},
			},
		}))
	})
}
