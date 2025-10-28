//go:build !integration

package workflow

import (
	"strings"
	"testing"
)

func TestClaudeFirewallIntegration(t *testing.T) {
	t.Run("claude with firewall enabled generates AWF installation steps", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name: "test-claude-firewall",
			NetworkPermissions: &NetworkPermissions{
				Allowed: []string{"api.example.com"},
				Firewall: &FirewallConfig{
					Enabled: true,
				},
			},
		}

		// Get Claude engine and installation steps
		engine := NewClaudeEngine()
		steps := engine.GetInstallationSteps(workflowData)

		// Check that AWF installation step is included
		foundAWFInstall := false
		foundAWFCleanup := false
		for _, step := range steps {
			stepStr := strings.Join(step, "\n")
			if strings.Contains(stepStr, "Install awf binary") {
				foundAWFInstall = true
			}
			if strings.Contains(stepStr, "Cleanup any existing awf resources") {
				foundAWFCleanup = true
			}
		}

		if !foundAWFInstall {
			t.Error("Expected AWF installation step when firewall is enabled for Claude")
		}
		if !foundAWFCleanup {
			t.Error("Expected AWF cleanup step when firewall is enabled for Claude")
		}
	})

	t.Run("claude with firewall enabled wraps execution with AWF", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name: "test-claude-firewall",
			NetworkPermissions: &NetworkPermissions{
				Allowed: []string{"api.example.com"},
				Firewall: &FirewallConfig{
					Enabled: true,
				},
			},
			Tools: make(map[string]any),
		}

		// Get Claude engine and execution steps
		engine := NewClaudeEngine()
		steps := engine.GetExecutionSteps(workflowData, "/tmp/gh-aw/agent-stdio.log")

		// Check that execution step includes AWF wrapper
		foundAWFWrapper := false
		for _, step := range steps {
			stepStr := strings.Join(step, "\n")
			if strings.Contains(stepStr, "sudo -E awf --env-all") {
				foundAWFWrapper = true
			}
		}

		if !foundAWFWrapper {
			t.Error("Expected AWF wrapper in execution step when firewall is enabled for Claude")
		}
	})

	t.Run("claude with firewall disabled does not include AWF", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name: "test-claude-no-firewall",
			NetworkPermissions: &NetworkPermissions{
				Allowed: []string{"api.example.com"},
				Firewall: &FirewallConfig{
					Enabled: false,
				},
			},
			Tools: make(map[string]any),
		}

		// Get Claude engine and installation steps
		engine := NewClaudeEngine()
		steps := engine.GetInstallationSteps(workflowData)

		// Check that AWF installation step is NOT included
		foundAWFInstall := false
		for _, step := range steps {
			stepStr := strings.Join(step, "\n")
			if strings.Contains(stepStr, "Install awf binary") {
				foundAWFInstall = true
			}
		}

		if foundAWFInstall {
			t.Error("Should not include AWF installation when firewall is disabled for Claude")
		}
	})
}

func TestCodexFirewallIntegration(t *testing.T) {
	t.Run("codex with firewall enabled generates AWF installation steps", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name: "test-codex-firewall",
			NetworkPermissions: &NetworkPermissions{
				Allowed: []string{"api.openai.com"},
				Firewall: &FirewallConfig{
					Enabled: true,
				},
			},
		}

		// Get Codex engine and installation steps
		engine := NewCodexEngine()
		steps := engine.GetInstallationSteps(workflowData)

		// Check that AWF installation step is included
		foundAWFInstall := false
		foundAWFCleanup := false
		for _, step := range steps {
			stepStr := strings.Join(step, "\n")
			if strings.Contains(stepStr, "Install awf binary") {
				foundAWFInstall = true
			}
			if strings.Contains(stepStr, "Cleanup any existing awf resources") {
				foundAWFCleanup = true
			}
		}

		if !foundAWFInstall {
			t.Error("Expected AWF installation step when firewall is enabled for Codex")
		}
		if !foundAWFCleanup {
			t.Error("Expected AWF cleanup step when firewall is enabled for Codex")
		}
	})

	t.Run("codex with firewall enabled wraps execution with AWF", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name: "test-codex-firewall",
			NetworkPermissions: &NetworkPermissions{
				Allowed: []string{"api.openai.com"},
				Firewall: &FirewallConfig{
					Enabled: true,
				},
			},
			Tools: make(map[string]any),
		}

		// Get Codex engine and execution steps
		engine := NewCodexEngine()
		steps := engine.GetExecutionSteps(workflowData, "/tmp/gh-aw/agent-stdio.log")

		// Check that execution step includes AWF wrapper
		foundAWFWrapper := false
		for _, step := range steps {
			stepStr := strings.Join(step, "\n")
			if strings.Contains(stepStr, "sudo -E awf --env-all") {
				foundAWFWrapper = true
			}
		}

		if !foundAWFWrapper {
			t.Error("Expected AWF wrapper in execution step when firewall is enabled for Codex")
		}
	})

	t.Run("codex with firewall disabled does not include AWF", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name: "test-codex-no-firewall",
			NetworkPermissions: &NetworkPermissions{
				Allowed: []string{"api.openai.com"},
				Firewall: &FirewallConfig{
					Enabled: false,
				},
			},
			Tools: make(map[string]any),
		}

		// Get Codex engine and installation steps
		engine := NewCodexEngine()
		steps := engine.GetInstallationSteps(workflowData)

		// Check that AWF installation step is NOT included
		foundAWFInstall := false
		for _, step := range steps {
			stepStr := strings.Join(step, "\n")
			if strings.Contains(stepStr, "Install awf binary") {
				foundAWFInstall = true
			}
		}

		if foundAWFInstall {
			t.Error("Should not include AWF installation when firewall is disabled for Codex")
		}
	})
}
