//go:build !integration

package workflow

import (
	"strings"
	"testing"
)

// TestChrootModeInAWFContainer tests that AWF uses chroot mode (default in v0.15.0+) for transparent host access
func TestChrootModeInAWFContainer(t *testing.T) {
	t.Run("chroot mode is enabled by default when firewall is enabled", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name: "test-workflow",
			EngineConfig: &EngineConfig{
				ID: "copilot",
			},
			NetworkPermissions: &NetworkPermissions{
				Firewall: &FirewallConfig{
					Enabled: true,
				},
			},
		}

		engine := NewCopilotEngine()
		steps := engine.GetExecutionSteps(workflowData, "test.log")

		stepContent := requireCopilotExecutionStep(t, steps)

		// Check that AWF is used (chroot mode is default in v0.15.0+)
		if !strings.Contains(stepContent, "sudo -E awf") {
			t.Error("Expected AWF command for transparent host access")
		}
	})

	t.Run("AWF command is NOT used when firewall is disabled", func(t *testing.T) {
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

		stepContent := requireCopilotExecutionStep(t, steps)

		// Check that AWF command is not used
		if strings.Contains(stepContent, "awf") {
			t.Error("Expected no AWF command when firewall is disabled")
		}
	})

	t.Run("chroot mode replaces individual binary mounts", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name: "test-workflow",
			EngineConfig: &EngineConfig{
				ID: "copilot",
			},
			NetworkPermissions: &NetworkPermissions{
				Firewall: &FirewallConfig{
					Enabled: true,
				},
			},
		}

		engine := NewCopilotEngine()
		steps := engine.GetExecutionSteps(workflowData, "test.log")

		stepContent := requireCopilotExecutionStep(t, steps)

		// Verify AWF is present (chroot mode is default in v0.15.0+)
		if !strings.Contains(stepContent, "sudo -E awf") {
			t.Error("Expected AWF to be present")
		}

		// Verify individual binary mounts are NOT present (replaced by default chroot mode)
		individualMounts := []string{
			"--mount /usr/bin/gh:/usr/bin/gh:ro",
			"--mount /usr/bin/cat:/usr/bin/cat:ro",
			"--mount /usr/bin/jq:/usr/bin/jq:ro",
			"--mount /tmp:/tmp:rw",
			"--mount /opt/hostedtoolcache:/opt/hostedtoolcache:ro",
		}

		for _, mount := range individualMounts {
			if strings.Contains(stepContent, mount) {
				t.Errorf("Individual mount '%s' should be replaced by default chroot mode", mount)
			}
		}
	})

	t.Run("chroot mode works with custom firewall args", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name: "test-workflow",
			EngineConfig: &EngineConfig{
				ID: "copilot",
			},
			NetworkPermissions: &NetworkPermissions{
				Firewall: &FirewallConfig{
					Enabled: true,
					Args:    []string{"--custom-flag", "value"},
				},
			},
		}

		engine := NewCopilotEngine()
		steps := engine.GetExecutionSteps(workflowData, "test.log")

		stepContent := requireCopilotExecutionStep(t, steps)

		// Verify AWF is present with custom args (chroot mode is default in v0.15.0+)
		if !strings.Contains(stepContent, "sudo -E awf") {
			t.Error("Expected AWF to be present with custom firewall args")
		}

		if !strings.Contains(stepContent, "--custom-flag") {
			t.Error("Expected custom firewall args to be present with chroot mode")
		}
	})

	t.Run("chroot mode works with AWF sandbox type", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name: "test-workflow",
			EngineConfig: &EngineConfig{
				ID: "copilot",
			},
			NetworkPermissions: &NetworkPermissions{
				Firewall: &FirewallConfig{
					Enabled: true,
				},
			},
			// Explicitly set AWF sandbox type
			SandboxConfig: &SandboxConfig{
				Agent: &AgentSandboxConfig{
					ID: "awf",
				},
			},
		}

		engine := NewCopilotEngine()
		steps := engine.GetExecutionSteps(workflowData, "test.log")

		stepContent := requireCopilotExecutionStep(t, steps)

		// Verify AWF is being used (chroot mode is default in v0.15.0+)
		if !strings.Contains(stepContent, "awf") {
			t.Error("Expected AWF to be used when firewall is enabled")
		}
	})
}

// TestChrootModeEnvFlags tests that --env-all is used with chroot mode to pass env vars to AWF
// and that every secret-bearing env var is excluded via --exclude-env
func TestChrootModeEnvFlags(t *testing.T) {
	t.Run("env-all is required for AWF to receive host env vars", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name: "test-workflow",
			EngineConfig: &EngineConfig{
				ID: "copilot",
			},
			NetworkPermissions: &NetworkPermissions{
				Firewall: &FirewallConfig{
					Enabled: true,
				},
			},
		}

		engine := NewCopilotEngine()
		steps := engine.GetExecutionSteps(workflowData, "test.log")

		stepContent := requireCopilotExecutionStep(t, steps)

		// Verify AWF is present (chroot mode is default in v0.15.0+)
		if !strings.Contains(stepContent, "sudo -E awf") {
			t.Error("Expected AWF to be present")
		}

		// Verify --env-all IS used (required for AWF to receive host environment variables)
		if !strings.Contains(stepContent, "--env-all") {
			t.Error("--env-all is required for AWF to receive host environment variables")
		}

		// Verify COPILOT_GITHUB_TOKEN is excluded via --exclude-env (AWF v0.25.3+ security fix)
		// This is always required for Copilot regardless of tool configuration.
		if !strings.Contains(stepContent, "--exclude-env COPILOT_GITHUB_TOKEN") {
			t.Error("COPILOT_GITHUB_TOKEN must be excluded from container env via --exclude-env")
		}
	})

	t.Run("github tool adds GITHUB_MCP_SERVER_TOKEN to exclude list", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name: "test-workflow",
			EngineConfig: &EngineConfig{
				ID: "copilot",
			},
			NetworkPermissions: &NetworkPermissions{
				Firewall: &FirewallConfig{
					Enabled: true,
				},
			},
			ParsedTools: &ToolsConfig{
				GitHub: &GitHubToolConfig{},
			},
		}

		engine := NewCopilotEngine()
		steps := engine.GetExecutionSteps(workflowData, "test.log")

		stepContent := requireCopilotExecutionStep(t, steps)

		// With GitHub tool present, GITHUB_MCP_SERVER_TOKEN must also be excluded
		if !strings.Contains(stepContent, "--exclude-env COPILOT_GITHUB_TOKEN") {
			t.Error("COPILOT_GITHUB_TOKEN must be excluded from container env")
		}
		if !strings.Contains(stepContent, "--exclude-env GITHUB_MCP_SERVER_TOKEN") {
			t.Error("GITHUB_MCP_SERVER_TOKEN must be excluded from container env when GitHub tool is present")
		}
	})
}

// TestCopilotNodePathExportedBeforeAWF verifies that the Copilot engine exports node's
// parent directory to PATH before `sudo -E awf` runs. This ensures AWF captures the
// node bin dir in AWF_HOST_PATH so it is available inside the AWF chroot container —
// critical on GPU runners (e.g. aw-gpu-runner-T4) where sudo's secure_path strips the
// toolcache additions made by actions/setup-node.
func TestCopilotNodePathExportedBeforeAWF(t *testing.T) {
	workflowData := &WorkflowData{
		Name: "test-workflow",
		EngineConfig: &EngineConfig{
			ID: "copilot",
		},
		NetworkPermissions: &NetworkPermissions{
			Firewall: &FirewallConfig{
				Enabled: true,
			},
		},
	}

	engine := NewCopilotEngine()
	steps := engine.GetExecutionSteps(workflowData, "test.log")

	stepContent := requireCopilotExecutionStep(t, steps)

	// GH_AW_NODE_BIN must still be captured for the fallback inside AWF
	if !strings.Contains(stepContent, "GH_AW_NODE_BIN") {
		t.Error("GH_AW_NODE_BIN must be captured before AWF runs")
	}

	// The path setup must export the node bin directory to PATH before sudo -E awf
	// so AWF_HOST_PATH includes it and `node` is available inside the chroot.
	if !strings.Contains(stepContent, `export PATH="$(dirname "${GH_AW_NODE_BIN}")`) {
		t.Error("node's bin directory must be prepended to PATH before sudo -E awf so AWF_HOST_PATH includes it")
	}

	// The export must appear before the sudo -E awf invocation
	pathExportIdx := strings.Index(stepContent, `export PATH="$(dirname "${GH_AW_NODE_BIN}")`)
	awfIdx := strings.Index(stepContent, "sudo -E awf")
	if pathExportIdx >= awfIdx {
		t.Errorf("node PATH export must appear before sudo -E awf (pathExportIdx=%d, awfIdx=%d)", pathExportIdx, awfIdx)
	}
}
