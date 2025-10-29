package constants

import (
	"path/filepath"
	"testing"
)

func TestGetWorkflowDir(t *testing.T) {
	expected := filepath.Join(".github", "workflows")
	result := GetWorkflowDir()

	if result != expected {
		t.Errorf("GetWorkflowDir() = %q, want %q", result, expected)
	}
}

func TestDefaultAllowedDomains(t *testing.T) {
	if len(DefaultAllowedDomains) == 0 {
		t.Error("DefaultAllowedDomains should not be empty")
	}

	expectedDomains := []string{"localhost", "localhost:*", "127.0.0.1", "127.0.0.1:*"}
	if len(DefaultAllowedDomains) != len(expectedDomains) {
		t.Errorf("DefaultAllowedDomains length = %d, want %d", len(DefaultAllowedDomains), len(expectedDomains))
	}

	for i, domain := range expectedDomains {
		if DefaultAllowedDomains[i] != domain {
			t.Errorf("DefaultAllowedDomains[%d] = %q, want %q", i, DefaultAllowedDomains[i], domain)
		}
	}
}

func TestSafeWorkflowEvents(t *testing.T) {
	if len(SafeWorkflowEvents) == 0 {
		t.Error("SafeWorkflowEvents should not be empty")
	}

	expectedEvents := []string{"workflow_dispatch", "workflow_run", "schedule"}
	if len(SafeWorkflowEvents) != len(expectedEvents) {
		t.Errorf("SafeWorkflowEvents length = %d, want %d", len(SafeWorkflowEvents), len(expectedEvents))
	}

	for i, event := range expectedEvents {
		if SafeWorkflowEvents[i] != event {
			t.Errorf("SafeWorkflowEvents[%d] = %q, want %q", i, SafeWorkflowEvents[i], event)
		}
	}
}

func TestAllowedExpressions(t *testing.T) {
	if len(AllowedExpressions) == 0 {
		t.Error("AllowedExpressions should not be empty")
	}

	// Test a few key expressions are present
	requiredExpressions := []string{
		"github.event.issue.number",
		"github.event.pull_request.number",
		"github.repository",
		"github.run_id",
		"github.workspace",
	}

	expressionsMap := make(map[string]bool)
	for _, expr := range AllowedExpressions {
		expressionsMap[expr] = true
	}

	for _, required := range requiredExpressions {
		if !expressionsMap[required] {
			t.Errorf("AllowedExpressions missing required expression: %q", required)
		}
	}
}

func TestAgenticEngines(t *testing.T) {
	if len(AgenticEngines) == 0 {
		t.Error("AgenticEngines should not be empty")
	}

	expectedEngines := []string{"claude", "codex", "copilot"}
	if len(AgenticEngines) != len(expectedEngines) {
		t.Errorf("AgenticEngines length = %d, want %d", len(AgenticEngines), len(expectedEngines))
	}

	for i, engine := range expectedEngines {
		if AgenticEngines[i] != engine {
			t.Errorf("AgenticEngines[%d] = %q, want %q", i, AgenticEngines[i], engine)
		}
	}
}

func TestDefaultGitHubTools(t *testing.T) {
	if len(DefaultGitHubToolsLocal) == 0 {
		t.Error("DefaultGitHubToolsLocal should not be empty")
	}

	if len(DefaultGitHubToolsRemote) == 0 {
		t.Error("DefaultGitHubToolsRemote should not be empty")
	}

	// Test that DefaultGitHubTools defaults to local mode
	if len(DefaultGitHubTools) != len(DefaultGitHubToolsLocal) {
		t.Errorf("DefaultGitHubTools should default to DefaultGitHubToolsLocal")
	}

	// Test a few key tools are present in both local and remote
	requiredTools := []string{
		"get_me",
		"list_issues",
		"pull_request_read",
		"get_file_contents",
		"search_code",
	}

	for _, tools := range [][]string{DefaultGitHubToolsLocal, DefaultGitHubToolsRemote} {
		toolsMap := make(map[string]bool)
		for _, tool := range tools {
			toolsMap[tool] = true
		}

		for _, required := range requiredTools {
			if !toolsMap[required] {
				t.Errorf("GitHub tools missing required tool: %q", required)
			}
		}
	}
}

func TestDefaultBashTools(t *testing.T) {
	if len(DefaultBashTools) == 0 {
		t.Error("DefaultBashTools should not be empty")
	}

	// Test a few key bash tools are present
	requiredTools := []string{
		"echo",
		"ls",
		"cat",
		"grep",
	}

	toolsMap := make(map[string]bool)
	for _, tool := range DefaultBashTools {
		toolsMap[tool] = true
	}

	for _, required := range requiredTools {
		if !toolsMap[required] {
			t.Errorf("DefaultBashTools missing required tool: %q", required)
		}
	}
}

func TestPriorityFields(t *testing.T) {
	if len(PriorityStepFields) == 0 {
		t.Error("PriorityStepFields should not be empty")
	}

	if len(PriorityJobFields) == 0 {
		t.Error("PriorityJobFields should not be empty")
	}

	if len(PriorityWorkflowFields) == 0 {
		t.Error("PriorityWorkflowFields should not be empty")
	}

	// Test that "name" is first in step fields
	if PriorityStepFields[0] != "name" {
		t.Errorf("PriorityStepFields[0] = %q, want %q", PriorityStepFields[0], "name")
	}

	// Test that "name" is first in job fields
	if PriorityJobFields[0] != "name" {
		t.Errorf("PriorityJobFields[0] = %q, want %q", PriorityJobFields[0], "name")
	}

	// Test that "on" is first in workflow fields
	if PriorityWorkflowFields[0] != "on" {
		t.Errorf("PriorityWorkflowFields[0] = %q, want %q", PriorityWorkflowFields[0], "on")
	}
}

func TestConstantValues(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		expected string
	}{
		{"CLIExtensionPrefix", CLIExtensionPrefix, "gh aw"},
		{"DefaultMCPRegistryURL", DefaultMCPRegistryURL, "https://api.mcp.github.com/v0"},
		{"AgentJobName", AgentJobName, "agent"},
		{"ActivationJobName", ActivationJobName, "activation"},
		{"PreActivationJobName", PreActivationJobName, "pre_activation"},
		{"DetectionJobName", DetectionJobName, "detection"},
		{"SafeOutputArtifactName", SafeOutputArtifactName, "safe_output.jsonl"},
		{"AgentOutputArtifactName", AgentOutputArtifactName, "agent_output.json"},
		{"SafeOutputsMCPServerID", SafeOutputsMCPServerID, "safeoutputs"},
		{"CheckMembershipStepID", CheckMembershipStepID, "check_membership"},
		{"CheckStopTimeStepID", CheckStopTimeStepID, "check_stop_time"},
		{"CheckCommandPositionStepID", CheckCommandPositionStepID, "check_command_position"},
		{"IsTeamMemberOutput", IsTeamMemberOutput, "is_team_member"},
		{"StopTimeOkOutput", StopTimeOkOutput, "stop_time_ok"},
		{"CommandPositionOkOutput", CommandPositionOkOutput, "command_position_ok"},
		{"ActivatedOutput", ActivatedOutput, "activated"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.value != tt.expected {
				t.Errorf("%s = %q, want %q", tt.name, tt.value, tt.expected)
			}
		})
	}
}

func TestNumericConstants(t *testing.T) {
	tests := []struct {
		name     string
		value    int
		minValue int
	}{
		{"MaxExpressionLineLength", MaxExpressionLineLength, 1},
		{"ExpressionBreakThreshold", ExpressionBreakThreshold, 1},
		{"DefaultAgenticWorkflowTimeoutMinutes", DefaultAgenticWorkflowTimeoutMinutes, 1},
		{"DefaultToolTimeoutSeconds", DefaultToolTimeoutSeconds, 1},
		{"DefaultMCPStartupTimeoutSeconds", DefaultMCPStartupTimeoutSeconds, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.value < tt.minValue {
				t.Errorf("%s = %d, should be >= %d", tt.name, tt.value, tt.minValue)
			}
		})
	}
}
