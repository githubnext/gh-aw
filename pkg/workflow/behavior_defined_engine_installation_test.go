//go:build !integration

package workflow

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newPreinstalledEngineDefinition returns a minimal EngineDefinition for an engine whose
// CLI is preinstalled in the runtime image (like Pydantic AI), so it declares no
// `installation` block at all.
func newPreinstalledEngineDefinition() *EngineDefinition {
	return &EngineDefinition{
		ID:          "preinstalled",
		DisplayName: "Preinstalled",
		Description: "A test engine that is preinstalled in the runtime image",
		Behaviors: &EngineBehaviorDefinition{
			Execution: &EngineExecutionDefinition{
				CommandName: "uv",
				Args:        []string{"run", "pai", "run"},
				StepName:    "Execute Preinstalled CLI",
			},
		},
	}
}

func sandboxedWorkflowData() *WorkflowData {
	return &WorkflowData{
		Name:          "test",
		SandboxConfig: &SandboxConfig{Agent: &AgentSandboxConfig{ID: "awf"}},
	}
}

// TestBehaviorDefinedEngineInstallsAWFWithoutNpmInstallation verifies that engines which
// declare no `installation` block (or a non-npm one) still get the AWF binary installed
// when the firewall/sandbox is enabled. Without this, the generated workflow fails with
// `awf: command not found`.
func TestBehaviorDefinedEngineInstallsAWFWithoutNpmInstallation(t *testing.T) {
	t.Run("no installation block with sandbox emits AWF install", func(t *testing.T) {
		engine, err := NewBehaviorDefinedEngine(newPreinstalledEngineDefinition())
		require.NoError(t, err)

		steps := engine.GetInstallationSteps(sandboxedWorkflowData())
		require.NotEmpty(t, steps, "expected AWF installation steps when sandbox is enabled")
		assert.Contains(t, joinSteps(steps), "install_awf_binary.sh")
	})

	t.Run("no installation block without firewall emits no steps", func(t *testing.T) {
		engine, err := NewBehaviorDefinedEngine(newPreinstalledEngineDefinition())
		require.NoError(t, err)

		steps := engine.GetInstallationSteps(&WorkflowData{Name: "test"})
		assert.Empty(t, steps, "expected no installation steps when the firewall is disabled")
	})

	t.Run("non-npm installation with sandbox emits AWF install", func(t *testing.T) {
		def := newPreinstalledEngineDefinition()
		def.Behaviors.Installation = &EngineInstallationDefinition{
			PackageManager: "pip",
			PackageName:    "pydantic-ai",
		}
		engine, err := NewBehaviorDefinedEngine(def)
		require.NoError(t, err)

		steps := engine.GetInstallationSteps(sandboxedWorkflowData())
		require.NotEmpty(t, steps, "expected AWF installation steps for non-npm installation")
		assert.Contains(t, joinSteps(steps), "install_awf_binary.sh")
	})

	t.Run("custom engine command skips installation", func(t *testing.T) {
		engine, err := NewBehaviorDefinedEngine(newPreinstalledEngineDefinition())
		require.NoError(t, err)

		workflowData := sandboxedWorkflowData()
		workflowData.EngineConfig = &EngineConfig{Command: "my-custom-cli"}
		assert.Empty(t, engine.GetInstallationSteps(workflowData))
	})
}
