//go:build !integration

package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGenerateGVisorInstallStep verifies that the gVisor install step contains
// the expected content from the reference implementation.
func TestGenerateGVisorInstallStep(t *testing.T) {
	step := generateGVisorInstallStep()
	require.NotEmpty(t, step, "gVisor install step must not be empty")

	content := strings.Join(step, "\n")

	assert.Contains(t, content, "Install gVisor (runsc)", "step should have a recognizable name")

	// Architecture detection: must use uname -m, not hardcoded amd64/arm64.
	assert.Contains(t, content, "uname -m", "must detect architecture via uname -m")
	assert.NotContains(t, content, "amd64", "must NOT remap architecture to amd64")
	assert.NotContains(t, content, "arm64", "must NOT remap architecture to arm64")

	// Both binaries must be downloaded.
	assert.Contains(t, content, "runsc", "must download runsc binary")
	assert.Contains(t, content, "containerd-shim-runsc-v1", "must download containerd-shim-runsc-v1")

	// Must use gVisor's official download URL.
	assert.Contains(t, content, "storage.googleapis.com/gvisor", "must use official gVisor download URL")

	// Must install binaries to system path (requires sudo).
	assert.Contains(t, content, "sudo install", "must install binaries with sudo")
	assert.Contains(t, content, "/usr/local/bin/runsc", "must install runsc to /usr/local/bin")

	// Must register the runtime with Docker.
	assert.Contains(t, content, "sudo runsc install", "must register runsc with Docker via runsc install")

	// Must use systemctl restart (NOT reload) to restart Docker.
	assert.Contains(t, content, "systemctl restart docker", "must restart Docker with systemctl restart")
	assert.NotContains(t, content, "systemctl reload docker", "must NOT use systemctl reload (breaks host-gateway DNS)")

	// Must verify the runtime works.
	assert.Contains(t, content, "docker run --rm --runtime=runsc", "must verify gVisor runtime with a test container")
}

// TestGVisorInstallStepOrderInBuildNpmEngineInstallStepsWithAWF verifies that the
// gVisor install step is emitted BEFORE the AWF install step.
func TestGVisorInstallStepOrderInBuildNpmEngineInstallStepsWithAWF(t *testing.T) {
	workflowData := &WorkflowData{
		SandboxConfig: &SandboxConfig{
			Agent: &AgentSandboxConfig{
				ID:                    "awf",
				Runtime:               AgentRuntimeGVisor,
				NetworkIsolation:      false, // sudo: true (required for gVisor)
				SudoExplicitlyEnabled: true,
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
// containerRuntime: "gvisor" to appear in the AWF config JSON.
func TestGVisorAWFConfigJSON(t *testing.T) {
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
					ID:                    "awf",
					Runtime:               AgentRuntimeGVisor,
					NetworkIsolation:      false,
					SudoExplicitlyEnabled: true,
				},
			},
		},
	}

	jsonStr, err := BuildAWFConfigJSON(config)
	require.NoError(t, err)

	assert.Contains(t, jsonStr, `"containerRuntime":"gvisor"`,
		"AWF config JSON must include containerRuntime: gvisor")
}

// TestGVisorAWFConfigJSONAbsentByDefault verifies that containerRuntime is not
// emitted when runtime is not configured.
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
				ID:                    "awf",
				Runtime:               AgentRuntimeGVisor,
				NetworkIsolation:      false,
				SudoExplicitlyEnabled: true,
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
	assert.Contains(t, err.Error(), "arc-dind", "error must mention arc-dind")
	assert.Contains(t, err.Error(), "gvisor", "error must mention gvisor")
}

// TestGVisorValidation_SudoFalseIncompatible verifies that gVisor + sudo:false (default)
// is a compile-time error.
func TestGVisorValidation_SudoFalseIncompatible(t *testing.T) {
	workflowData := &WorkflowData{
		SandboxConfig: &SandboxConfig{
			Agent: &AgentSandboxConfig{
				ID:               "awf",
				Runtime:          AgentRuntimeGVisor,
				NetworkIsolation: true, // sudo: false (default or explicit)
			},
		},
		NetworkPermissions: &NetworkPermissions{
			Firewall: &FirewallConfig{Enabled: true},
		},
		Tools: map[string]any{"github": map[string]any{"mode": "remote"}},
	}

	err := validateSandboxConfig(workflowData)
	require.Error(t, err, "gVisor + sudo:false must produce a compile-time error")
	assert.Contains(t, err.Error(), "gvisor", "error must mention gvisor")
	assert.Contains(t, err.Error(), "sudo", "error must mention sudo")
}

// TestGVisorValidation_ValidCombination verifies that a valid gVisor configuration
// (with sudo: true on a non-arc-dind runner) passes validation.
func TestGVisorValidation_ValidCombination(t *testing.T) {
	workflowData := &WorkflowData{
		SandboxConfig: &SandboxConfig{
			Agent: &AgentSandboxConfig{
				ID:                    "awf",
				Runtime:               AgentRuntimeGVisor,
				NetworkIsolation:      false, // sudo: true
				SudoExplicitlyEnabled: true,
			},
		},
		NetworkPermissions: &NetworkPermissions{
			Firewall: &FirewallConfig{Enabled: true},
		},
		Tools: map[string]any{"github": map[string]any{"mode": "remote"}},
	}

	err := validateSandboxConfig(workflowData)
	assert.NoError(t, err, "gVisor + sudo:true on a standard runner must pass validation")
}

// TestGVisorStrictModeSudoTrueSuppressed verifies that sandbox.agent.sudo: true
// does NOT produce a strict-mode error or warning when runtime: gvisor is configured.
func TestGVisorStrictModeSudoTrueSuppressed(t *testing.T) {
	sandboxConfig := &SandboxConfig{
		Agent: &AgentSandboxConfig{
			ID:                    "awf",
			Runtime:               AgentRuntimeGVisor,
			NetworkIsolation:      false,
			SudoExplicitlyEnabled: true,
		},
	}

	compiler := NewCompiler()
	compiler.strictMode = true
	initialWarnings := compiler.GetWarningCount()

	err := compiler.validateStrictSandboxCustomization(sandboxConfig)
	require.NoError(t, err, "sudo:true + runtime:gvisor must not produce a strict-mode error")
	assert.Equal(t, initialWarnings, compiler.GetWarningCount(), "sudo:true + runtime:gvisor must not produce a warning")
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
    sudo: true
---

# Test gVisor Runtime
`

	testFile := filepath.Join(workflowsDir, "test-gvisor.md")
	err := os.WriteFile(testFile, []byte(markdown), 0644)
	require.NoError(t, err)

	compiler := NewCompiler()
	err = compiler.CompileWorkflow(testFile)
	require.NoError(t, err, "compilation with runtime: gvisor must succeed")

	lockContent, err := os.ReadFile(filepath.Join(workflowsDir, "test-gvisor.lock.yml"))
	require.NoError(t, err)
	lockStr := string(lockContent)

	// gVisor install step must be present.
	assert.Contains(t, lockStr, "Install gVisor", "compiled workflow must include gVisor install step")

	// AWF install step must also be present.
	assert.Contains(t, lockStr, "Install AWF binary", "compiled workflow must include AWF install step")

	// containerRuntime: gvisor must appear in the AWF config JSON.
	// The AWF config is embedded as escaped JSON in a printf command in the lock file.
	assert.Contains(t, lockStr, `\"containerRuntime\":\"gvisor\"`,
		"compiled workflow must pass containerRuntime: gvisor to AWF")

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
