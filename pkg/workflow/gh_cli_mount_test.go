package workflow

import (
	"strings"
	"testing"
)

// TestGhCLIMountInAWFContainer tests that gh CLI binary is mounted in AWF container
func TestGhCLIMountInAWFContainer(t *testing.T) {
	t.Run("gh CLI is mounted when firewall is enabled", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name: "test-workflow",
			EngineConfig: &EngineConfig{
				ID: "copilot",
			},
			NetworkPermissions: &NetworkPermissions{
					Enabled: true,
				},
			},
		}

		engine := NewCopilotEngine()
		steps := engine.GetExecutionSteps(workflowData, "test.log")

		if len(steps) == 0 {
			t.Fatal("Expected at least one execution step")
		}

		stepContent := strings.Join(steps[0], "\n")

		// Check that gh CLI binary mount is included in AWF command
		expectedMount := "--mount /usr/bin/gh:/usr/bin/gh:ro"
		if !strings.Contains(stepContent, expectedMount) {
			t.Errorf("Expected AWF command to contain gh CLI binary mount '%s', but it was not found", expectedMount)
		}

		// Verify mount is read-only
		if !strings.Contains(stepContent, "/usr/bin/gh:ro") {
			t.Error("Expected gh CLI mount to be read-only (:ro)")
		}
	})

	t.Run("gh CLI is NOT mounted when firewall is disabled", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name: "test-workflow",
			EngineConfig: &EngineConfig{
				ID: "copilot",
			},
			SandboxConfig: &SandboxConfig{
				Agent: &AgentSandboxConfig{
					Disabled: true,
				},
			},
		}

		engine := NewCopilotEngine()
		steps := engine.GetExecutionSteps(workflowData, "test.log")

		if len(steps) == 0 {
			t.Fatal("Expected at least one execution step")
		}

		stepContent := strings.Join(steps[0], "\n")

		// Check that AWF command is not used
		if strings.Contains(stepContent, "awf") {
			t.Error("Expected no AWF command when firewall is disabled")
		}

		// Check that gh CLI mount is not present
		if strings.Contains(stepContent, "/usr/bin/gh") {
			t.Error("Expected no gh CLI mount when firewall is disabled")
		}
	})

	t.Run("gh CLI mount is positioned after workspace mounts", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name: "test-workflow",
			EngineConfig: &EngineConfig{
				ID: "copilot",
			},
			NetworkPermissions: &NetworkPermissions{
					Enabled: true,
				},
			},
		}

		engine := NewCopilotEngine()
		steps := engine.GetExecutionSteps(workflowData, "test.log")

		if len(steps) == 0 {
			t.Fatal("Expected at least one execution step")
		}

		stepContent := strings.Join(steps[0], "\n")

		// Find positions of mounts in the command
		tmpMountPos := strings.Index(stepContent, "--mount /tmp:/tmp:rw")
		workspaceMountPos := strings.Index(stepContent, "--mount \"${GITHUB_WORKSPACE}:${GITHUB_WORKSPACE}:rw\"")
		ghMountPos := strings.Index(stepContent, "--mount /usr/bin/gh:/usr/bin/gh:ro")

		if tmpMountPos == -1 || workspaceMountPos == -1 || ghMountPos == -1 {
			t.Fatal("Not all expected mounts were found in the command")
		}

		// Verify order: /tmp < workspace < gh
		if tmpMountPos >= workspaceMountPos || workspaceMountPos >= ghMountPos {
			t.Error("Expected mount order: /tmp, workspace, gh CLI")
		}
	})

	t.Run("gh CLI mount works with custom firewall args", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name: "test-workflow",
			EngineConfig: &EngineConfig{
				ID: "copilot",
			},
			NetworkPermissions: &NetworkPermissions{
					Enabled: true,
					Args:    []string{"--custom-flag", "value"},
				},
			},
		}

		engine := NewCopilotEngine()
		steps := engine.GetExecutionSteps(workflowData, "test.log")

		if len(steps) == 0 {
			t.Fatal("Expected at least one execution step")
		}

		stepContent := strings.Join(steps[0], "\n")

		// Verify both gh mount and custom args are present
		if !strings.Contains(stepContent, "--mount /usr/bin/gh:/usr/bin/gh:ro") {
			t.Error("Expected gh CLI mount to be present with custom firewall args")
		}

		if !strings.Contains(stepContent, "--custom-flag") {
			t.Error("Expected custom firewall args to be present with gh CLI mount")
		}
	})

	t.Run("gh CLI mount works with SRT disabled", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name: "test-workflow",
			EngineConfig: &EngineConfig{
				ID: "copilot",
			},
			NetworkPermissions: &NetworkPermissions{
					Enabled: true,
				},
			},
			// Explicitly ensure SRT is not enabled
			SandboxConfig: &SandboxConfig{
				Agent: &AgentSandboxConfig{
					ID: "awf",
				},
			},
		}

		engine := NewCopilotEngine()
		steps := engine.GetExecutionSteps(workflowData, "test.log")

		if len(steps) == 0 {
			t.Fatal("Expected at least one execution step")
		}

		stepContent := strings.Join(steps[0], "\n")

		// Verify gh CLI mount is present
		if !strings.Contains(stepContent, "--mount /usr/bin/gh:/usr/bin/gh:ro") {
			t.Error("Expected gh CLI mount to be present when using AWF (not SRT)")
		}

		// Verify AWF is being used
		if !strings.Contains(stepContent, "awf") {
			t.Error("Expected AWF to be used when firewall is enabled")
		}
	})
}
