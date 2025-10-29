package constants

import (
	"path/filepath"
	"testing"
)

func TestGetWorkflowDir(t *testing.T) {
	got := GetWorkflowDir()
	expected := filepath.Join(".github", "workflows")

	if got != expected {
		t.Errorf("GetWorkflowDir() = %v, want %v", got, expected)
	}
}

func TestConstants(t *testing.T) {
	t.Run("CLIExtensionPrefix", func(t *testing.T) {
		if CLIExtensionPrefix != "gh aw" {
			t.Errorf("CLIExtensionPrefix = %v, want 'gh aw'", CLIExtensionPrefix)
		}
	})

	t.Run("MaxExpressionLineLength", func(t *testing.T) {
		if MaxExpressionLineLength != 120 {
			t.Errorf("MaxExpressionLineLength = %v, want 120", MaxExpressionLineLength)
		}
	})

	t.Run("ExpressionBreakThreshold", func(t *testing.T) {
		if ExpressionBreakThreshold != 100 {
			t.Errorf("ExpressionBreakThreshold = %v, want 100", ExpressionBreakThreshold)
		}
	})

	t.Run("DefaultMCPRegistryURL", func(t *testing.T) {
		expected := "https://api.mcp.github.com/v0"
		if DefaultMCPRegistryURL != expected {
			t.Errorf("DefaultMCPRegistryURL = %v, want %v", DefaultMCPRegistryURL, expected)
		}
	})

	t.Run("DefaultClaudeCodeVersion", func(t *testing.T) {
		if DefaultClaudeCodeVersion == "" {
			t.Error("DefaultClaudeCodeVersion should not be empty")
		}
	})

	t.Run("DefaultCopilotVersion", func(t *testing.T) {
		if DefaultCopilotVersion == "" {
			t.Error("DefaultCopilotVersion should not be empty")
		}
	})

	t.Run("DefaultCodexVersion", func(t *testing.T) {
		if DefaultCodexVersion == "" {
			t.Error("DefaultCodexVersion should not be empty")
		}
	})

	t.Run("DefaultAgenticWorkflowTimeoutMinutes", func(t *testing.T) {
		if DefaultAgenticWorkflowTimeoutMinutes <= 0 {
			t.Errorf("DefaultAgenticWorkflowTimeoutMinutes = %v, should be positive", DefaultAgenticWorkflowTimeoutMinutes)
		}
	})

	t.Run("DefaultToolTimeoutSeconds", func(t *testing.T) {
		if DefaultToolTimeoutSeconds <= 0 {
			t.Errorf("DefaultToolTimeoutSeconds = %v, should be positive", DefaultToolTimeoutSeconds)
		}
	})

	t.Run("AgentJobName", func(t *testing.T) {
		if AgentJobName != "agent" {
			t.Errorf("AgentJobName = %v, want 'agent'", AgentJobName)
		}
	})

	t.Run("ActivationJobName", func(t *testing.T) {
		if ActivationJobName != "activation" {
			t.Errorf("ActivationJobName = %v, want 'activation'", ActivationJobName)
		}
	})
}

func TestDefaultAllowedDomains(t *testing.T) {
	if len(DefaultAllowedDomains) == 0 {
		t.Error("DefaultAllowedDomains should not be empty")
	}

	expectedDomains := map[string]bool{
		"localhost":   true,
		"localhost:*": true,
		"127.0.0.1":   true,
		"127.0.0.1:*": true,
	}

	for _, domain := range DefaultAllowedDomains {
		if !expectedDomains[domain] {
			t.Errorf("Unexpected domain in DefaultAllowedDomains: %v", domain)
		}
		delete(expectedDomains, domain)
	}

	if len(expectedDomains) > 0 {
		t.Errorf("Missing expected domains in DefaultAllowedDomains: %v", expectedDomains)
	}
}

func TestSafeWorkflowEvents(t *testing.T) {
	if len(SafeWorkflowEvents) == 0 {
		t.Error("SafeWorkflowEvents should not be empty")
	}

	expectedEvents := map[string]bool{
		"workflow_dispatch": true,
		"workflow_run":      true,
		"schedule":          true,
	}

	for _, event := range SafeWorkflowEvents {
		if !expectedEvents[event] {
			t.Errorf("Unexpected event in SafeWorkflowEvents: %v", event)
		}
		delete(expectedEvents, event)
	}

	if len(expectedEvents) > 0 {
		t.Errorf("Missing expected events in SafeWorkflowEvents: %v", expectedEvents)
	}
}

func TestAllowedExpressions(t *testing.T) {
	if len(AllowedExpressions) == 0 {
		t.Error("AllowedExpressions should not be empty")
	}

	// Test some critical expressions are present
	criticalExpressions := []string{
		"github.event.issue.number",
		"github.event.pull_request.number",
		"github.repository",
		"github.run_id",
		"github.workspace",
	}

	expressionMap := make(map[string]bool)
	for _, expr := range AllowedExpressions {
		expressionMap[expr] = true
	}

	for _, critical := range criticalExpressions {
		if !expressionMap[critical] {
			t.Errorf("Critical expression missing from AllowedExpressions: %v", critical)
		}
	}
}

func TestAgenticEngines(t *testing.T) {
	if len(AgenticEngines) == 0 {
		t.Error("AgenticEngines should not be empty")
	}

	expectedEngines := map[string]bool{
		"claude":  true,
		"codex":   true,
		"copilot": true,
	}

	for _, engine := range AgenticEngines {
		if !expectedEngines[engine] {
			t.Errorf("Unexpected engine in AgenticEngines: %v", engine)
		}
		delete(expectedEngines, engine)
	}

	if len(expectedEngines) > 0 {
		t.Errorf("Missing expected engines in AgenticEngines: %v", expectedEngines)
	}
}

func TestDefaultGitHubTools(t *testing.T) {
	if len(DefaultGitHubToolsLocal) == 0 {
		t.Error("DefaultGitHubToolsLocal should not be empty")
	}

	if len(DefaultGitHubToolsRemote) == 0 {
		t.Error("DefaultGitHubToolsRemote should not be empty")
	}

	// Test that DefaultGitHubTools points to local by default (backward compatibility)
	if len(DefaultGitHubTools) != len(DefaultGitHubToolsLocal) {
		t.Error("DefaultGitHubTools should equal DefaultGitHubToolsLocal for backward compatibility")
	}
}

func TestDefaultBashTools(t *testing.T) {
	if len(DefaultBashTools) == 0 {
		t.Error("DefaultBashTools should not be empty")
	}

	// Test some critical bash commands are present
	criticalCommands := []string{
		"echo",
		"ls",
		"cat",
		"grep",
	}

	commandMap := make(map[string]bool)
	for _, cmd := range DefaultBashTools {
		commandMap[cmd] = true
	}

	for _, critical := range criticalCommands {
		if !commandMap[critical] {
			t.Errorf("Critical bash command missing from DefaultBashTools: %v", critical)
		}
	}
}

func TestPriorityFields(t *testing.T) {
	t.Run("PriorityStepFields", func(t *testing.T) {
		if len(PriorityStepFields) == 0 {
			t.Error("PriorityStepFields should not be empty")
		}
		// Test that 'name' is first
		if len(PriorityStepFields) > 0 && PriorityStepFields[0] != "name" {
			t.Errorf("PriorityStepFields[0] = %v, want 'name'", PriorityStepFields[0])
		}
	})

	t.Run("PriorityJobFields", func(t *testing.T) {
		if len(PriorityJobFields) == 0 {
			t.Error("PriorityJobFields should not be empty")
		}
		// Test that 'name' is first
		if len(PriorityJobFields) > 0 && PriorityJobFields[0] != "name" {
			t.Errorf("PriorityJobFields[0] = %v, want 'name'", PriorityJobFields[0])
		}
	})

	t.Run("PriorityWorkflowFields", func(t *testing.T) {
		if len(PriorityWorkflowFields) == 0 {
			t.Error("PriorityWorkflowFields should not be empty")
		}
		// Test that 'on' is first
		if len(PriorityWorkflowFields) > 0 && PriorityWorkflowFields[0] != "on" {
			t.Errorf("PriorityWorkflowFields[0] = %v, want 'on'", PriorityWorkflowFields[0])
		}
	})
}

func TestIgnoredFrontmatterFields(t *testing.T) {
	if len(IgnoredFrontmatterFields) == 0 {
		t.Error("IgnoredFrontmatterFields should not be empty")
	}

	expectedFields := map[string]bool{
		"description": true,
		"applyTo":     true,
	}

	for _, field := range IgnoredFrontmatterFields {
		if !expectedFields[field] {
			t.Errorf("Unexpected field in IgnoredFrontmatterFields: %v", field)
		}
		delete(expectedFields, field)
	}

	if len(expectedFields) > 0 {
		t.Errorf("Missing expected fields in IgnoredFrontmatterFields: %v", expectedFields)
	}
}
