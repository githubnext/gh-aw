package workflow

import (
	"strings"
	"testing"

	"github.com/githubnext/gh-aw/pkg/constants"
)

func TestCopilotFirewallIntegration(t *testing.T) {
	t.Run("firewall installation step is added when network permissions are configured", func(t *testing.T) {
		workflowData := &WorkflowData{
			NetworkPermissions: &NetworkPermissions{
				Mode: "defaults",
			},
		}

		engine := NewCopilotEngine()
		steps := engine.GetInstallationSteps(workflowData)

		// Convert steps to string for easier checking
		var allLines []string
		for _, step := range steps {
			allLines = append(allLines, step...)
		}
		stepsStr := strings.Join(allLines, "\n")

		// Should contain firewall installation step
		if !strings.Contains(stepsStr, "Install gh-aw-firewall") {
			t.Error("Should contain firewall installation step when network permissions are configured")
		}

		// Should download firewall from GitHub releases
		if !strings.Contains(stepsStr, "github.com/githubnext/gh-aw-firewall/releases") {
			t.Error("Should download firewall from GitHub releases")
		}

		// Should use correct version
		if !strings.Contains(stepsStr, constants.DefaultFirewallVersion) {
			t.Errorf("Should use firewall version %s", constants.DefaultFirewallVersion)
		}

		// Should make firewall executable
		if !strings.Contains(stepsStr, "chmod +x /tmp/gh-aw-firewall") {
			t.Error("Should make firewall executable")
		}
	})

	t.Run("firewall installation step is not added when no network permissions", func(t *testing.T) {
		workflowData := &WorkflowData{
			NetworkPermissions: nil, // No network restrictions
		}

		engine := NewCopilotEngine()
		steps := engine.GetInstallationSteps(workflowData)

		var allLines []string
		for _, step := range steps {
			allLines = append(allLines, step...)
		}
		stepsStr := strings.Join(allLines, "\n")

		// Should NOT contain firewall installation step
		if strings.Contains(stepsStr, "Install gh-aw-firewall") {
			t.Error("Should not contain firewall installation step when no network permissions are configured")
		}
	})

	t.Run("copilot command is wrapped with firewall when network permissions are configured", func(t *testing.T) {
		workflowData := &WorkflowData{
			NetworkPermissions: &NetworkPermissions{
				Allowed: []string{"example.com", "test.org"},
			},
			Tools: map[string]any{
				"github": nil,
			},
		}

		engine := NewCopilotEngine()
		steps := engine.GetExecutionSteps(workflowData, "/tmp/test.log")

		var allLines []string
		for _, step := range steps {
			allLines = append(allLines, step...)
		}
		stepsStr := strings.Join(allLines, "\n")

		// Should wrap copilot with firewall
		if !strings.Contains(stepsStr, "/tmp/gh-aw-firewall") {
			t.Error("Should wrap copilot command with firewall")
		}

		// Should use --allowed-domains flag
		if !strings.Contains(stepsStr, "--allowed-domains") {
			t.Error("Should use --allowed-domains flag")
		}

		// Should use --env-all flag
		if !strings.Contains(stepsStr, "--env-all") {
			t.Error("Should use --env-all flag")
		}

		// Should include the allowed domains
		if !strings.Contains(stepsStr, "example.com") || !strings.Contains(stepsStr, "test.org") {
			t.Error("Should include allowed domains in firewall command")
		}

		// Should still have the copilot command after --
		if !strings.Contains(stepsStr, "-- copilot") {
			t.Error("Should have copilot command after firewall wrapper")
		}
	})

	t.Run("copilot command is not wrapped when no network permissions", func(t *testing.T) {
		workflowData := &WorkflowData{
			NetworkPermissions: nil, // No network restrictions
			Tools: map[string]any{
				"github": nil,
			},
		}

		engine := NewCopilotEngine()
		steps := engine.GetExecutionSteps(workflowData, "/tmp/test.log")

		var allLines []string
		for _, step := range steps {
			allLines = append(allLines, step...)
		}
		stepsStr := strings.Join(allLines, "\n")

		// Should NOT wrap copilot with firewall
		if strings.Contains(stepsStr, "/tmp/gh-aw-firewall") {
			t.Error("Should not wrap copilot command with firewall when no network permissions")
		}

		// Should have direct copilot command
		if !strings.Contains(stepsStr, "copilot --add-dir") {
			t.Error("Should have direct copilot command when no network restrictions")
		}
	})

	t.Run("firewall handles ecosystem identifiers correctly", func(t *testing.T) {
		workflowData := &WorkflowData{
			NetworkPermissions: &NetworkPermissions{
				Allowed: []string{"defaults", "node"},
			},
			Tools: map[string]any{
				"github": nil,
			},
		}

		engine := NewCopilotEngine()
		steps := engine.GetExecutionSteps(workflowData, "/tmp/test.log")

		var allLines []string
		for _, step := range steps {
			allLines = append(allLines, step...)
		}
		stepsStr := strings.Join(allLines, "\n")

		// Should expand ecosystem identifiers
		// "defaults" and "node" should be expanded to actual domain lists
		if !strings.Contains(stepsStr, "npmjs.org") {
			t.Error("Should expand 'node' ecosystem to include npmjs.org")
		}

		if !strings.Contains(stepsStr, "json-schema.org") {
			t.Error("Should expand 'defaults' ecosystem to include json-schema.org")
		}
	})

	t.Run("firewall is not used when wildcard allows all domains", func(t *testing.T) {
		workflowData := &WorkflowData{
			NetworkPermissions: &NetworkPermissions{
				Allowed: []string{"*"},
			},
			Tools: map[string]any{
				"github": nil,
			},
		}

		engine := NewCopilotEngine()

		// Check installation steps - firewall should NOT be installed
		installSteps := engine.GetInstallationSteps(workflowData)
		var installLines []string
		for _, step := range installSteps {
			installLines = append(installLines, step...)
		}
		installStr := strings.Join(installLines, "\n")

		if strings.Contains(installStr, "Install gh-aw-firewall") {
			t.Error("Should not install firewall when wildcard allows all domains")
		}

		// Check execution steps - firewall should NOT be used
		execSteps := engine.GetExecutionSteps(workflowData, "/tmp/test.log")
		var execLines []string
		for _, step := range execSteps {
			execLines = append(execLines, step...)
		}
		execStr := strings.Join(execLines, "\n")

		if strings.Contains(execStr, "/tmp/gh-aw-firewall") {
			t.Error("Should not use firewall when wildcard allows all domains")
		}

		// Should have direct copilot command
		if !strings.Contains(execStr, "copilot --add-dir") {
			t.Error("Should have direct copilot command when wildcard allows all")
		}
	})
}

func TestFirewallInstallationStep(t *testing.T) {
	engine := NewCopilotEngine()
	step := engine.generateFirewallInstallationStep()

	stepStr := strings.Join(step, "\n")

	t.Run("contains correct step name", func(t *testing.T) {
		if !strings.Contains(stepStr, "name: Install gh-aw-firewall") {
			t.Error("Should have correct step name")
		}
	})

	t.Run("downloads from correct URL", func(t *testing.T) {
		expectedURL := "https://github.com/githubnext/gh-aw-firewall/releases/download"
		if !strings.Contains(stepStr, expectedURL) {
			t.Errorf("Should download from %s", expectedURL)
		}
	})

	t.Run("uses correct version", func(t *testing.T) {
		if !strings.Contains(stepStr, constants.DefaultFirewallVersion) {
			t.Errorf("Should use version %s", constants.DefaultFirewallVersion)
		}
	})

	t.Run("saves to /tmp directory", func(t *testing.T) {
		if !strings.Contains(stepStr, "/tmp/gh-aw-firewall") {
			t.Error("Should save firewall to /tmp/gh-aw-firewall")
		}
	})

	t.Run("makes firewall executable", func(t *testing.T) {
		if !strings.Contains(stepStr, "chmod +x /tmp/gh-aw-firewall") {
			t.Error("Should make firewall executable")
		}
	})

	t.Run("verifies installation with version check", func(t *testing.T) {
		if !strings.Contains(stepStr, "/tmp/gh-aw-firewall --version") {
			t.Error("Should verify installation with version check")
		}
	})

	t.Run("uses set -e for error handling", func(t *testing.T) {
		if !strings.Contains(stepStr, "set -e") {
			t.Error("Should use set -e for error handling")
		}
	})
}
