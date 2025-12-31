package constants

import (
	"path/filepath"
	"testing"
	"time"
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

	// workflow_run is intentionally excluded due to HIGH security risks
	expectedEvents := []string{"workflow_dispatch", "schedule"}
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

	if len(DefaultReadOnlyGitHubTools) == 0 {
		t.Error("DefaultReadOnlyGitHubTools should not be empty")
	}

	// Test that DefaultGitHubTools defaults to local mode
	if len(DefaultGitHubTools) != len(DefaultGitHubToolsLocal) {
		t.Errorf("DefaultGitHubTools should default to DefaultGitHubToolsLocal")
	}

	// Test that Local and Remote tools reference the same shared list
	if len(DefaultGitHubToolsLocal) != len(DefaultReadOnlyGitHubTools) {
		t.Errorf("DefaultGitHubToolsLocal should have same length as DefaultReadOnlyGitHubTools, got %d vs %d",
			len(DefaultGitHubToolsLocal), len(DefaultReadOnlyGitHubTools))
	}

	if len(DefaultGitHubToolsRemote) != len(DefaultReadOnlyGitHubTools) {
		t.Errorf("DefaultGitHubToolsRemote should have same length as DefaultReadOnlyGitHubTools, got %d vs %d",
			len(DefaultGitHubToolsRemote), len(DefaultReadOnlyGitHubTools))
	}

	// Test a few key tools are present in all lists
	requiredTools := []string{
		"get_me",
		"list_issues",
		"pull_request_read",
		"get_file_contents",
		"search_code",
	}

	for name, tools := range map[string][]string{
		"DefaultGitHubToolsLocal":    DefaultGitHubToolsLocal,
		"DefaultGitHubToolsRemote":   DefaultGitHubToolsRemote,
		"DefaultReadOnlyGitHubTools": DefaultReadOnlyGitHubTools,
	} {
		toolsMap := make(map[string]bool)
		for _, tool := range tools {
			toolsMap[tool] = true
		}

		for _, required := range requiredTools {
			if !toolsMap[required] {
				t.Errorf("%s missing required tool: %q", name, required)
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
		{"CLIExtensionPrefix", string(CLIExtensionPrefix), "gh aw"},
		{"DefaultMCPRegistryURL", string(DefaultMCPRegistryURL), "https://api.mcp.github.com/v0"},
		{"AgentJobName", string(AgentJobName), "agent"},
		{"ActivationJobName", string(ActivationJobName), "activation"},
		{"PreActivationJobName", string(PreActivationJobName), "pre_activation"},
		{"DetectionJobName", string(DetectionJobName), "detection"},
		{"SafeOutputArtifactName", SafeOutputArtifactName, "safe-output"},
		{"AgentOutputArtifactName", AgentOutputArtifactName, "agent-output"},
		{"SafeOutputsMCPServerID", SafeOutputsMCPServerID, "safeoutputs"},
		{"CheckMembershipStepID", string(CheckMembershipStepID), "check_membership"},
		{"CheckStopTimeStepID", string(CheckStopTimeStepID), "check_stop_time"},
		{"CheckSkipIfMatchStepID", string(CheckSkipIfMatchStepID), "check_skip_if_match"},
		{"CheckCommandPositionStepID", string(CheckCommandPositionStepID), "check_command_position"},
		{"IsTeamMemberOutput", IsTeamMemberOutput, "is_team_member"},
		{"StopTimeOkOutput", StopTimeOkOutput, "stop_time_ok"},
		{"SkipCheckOkOutput", SkipCheckOkOutput, "skip_check_ok"},
		{"CommandPositionOkOutput", CommandPositionOkOutput, "command_position_ok"},
		{"ActivatedOutput", ActivatedOutput, "activated"},
		{"DefaultActivationJobRunnerImage", DefaultActivationJobRunnerImage, "ubuntu-slim"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.value != tt.expected {
				t.Errorf("%s = %q, want %q", tt.name, tt.value, tt.expected)
			}
		})
	}
}

func TestModelNameConstants(t *testing.T) {
	// Test that DefaultCopilotDetectionModel has the correct type and value
	tests := []struct {
		name     string
		value    ModelName
		expected string
	}{
		{"DefaultCopilotDetectionModel", DefaultCopilotDetectionModel, "gpt-5-mini"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.value) != tt.expected {
				t.Errorf("%s = %q, want %q", tt.name, tt.value, tt.expected)
			}
		})
	}
}

func TestVersionConstants(t *testing.T) {
	tests := []struct {
		name     string
		value    Version
		expected Version
	}{
		{"DefaultClaudeCodeVersion", DefaultClaudeCodeVersion, "2.0.76"},
		{"DefaultCopilotVersion", DefaultCopilotVersion, "0.0.373"},
		{"DefaultCodexVersion", DefaultCodexVersion, "0.77.0"},
		{"DefaultGitHubMCPServerVersion", DefaultGitHubMCPServerVersion, "v0.26.3"},
		{"DefaultFirewallVersion", DefaultFirewallVersion, "v0.7.0"},
		{"DefaultPlaywrightMCPVersion", DefaultPlaywrightMCPVersion, "0.0.54"},
		{"DefaultPlaywrightBrowserVersion", DefaultPlaywrightBrowserVersion, "v1.57.0"},
		{"DefaultBunVersion", DefaultBunVersion, "1.1"},
		{"DefaultNodeVersion", DefaultNodeVersion, "24"},
		{"DefaultPythonVersion", DefaultPythonVersion, "3.12"},
		{"DefaultRubyVersion", DefaultRubyVersion, "3.3"},
		{"DefaultDotNetVersion", DefaultDotNetVersion, "8.0"},
		{"DefaultJavaVersion", DefaultJavaVersion, "21"},
		{"DefaultElixirVersion", DefaultElixirVersion, "1.17"},
		{"DefaultHaskellVersion", DefaultHaskellVersion, "9.10"},
		{"DefaultDenoVersion", DefaultDenoVersion, "2.x"},
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
		value    LineLength
		minValue LineLength
	}{
		{"MaxExpressionLineLength", MaxExpressionLineLength, 1},
		{"ExpressionBreakThreshold", ExpressionBreakThreshold, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.value < tt.minValue {
				t.Errorf("%s = %d, should be >= %d", tt.name, tt.value, tt.minValue)
			}
		})
	}
}

func TestTimeoutConstants(t *testing.T) {
	// Test new time.Duration-based constants
	tests := []struct {
		name     string
		value    time.Duration
		minValue time.Duration
	}{
		{"DefaultAgenticWorkflowTimeout", DefaultAgenticWorkflowTimeout, 1 * time.Minute},
		{"DefaultToolTimeout", DefaultToolTimeout, 1 * time.Second},
		{"DefaultMCPStartupTimeout", DefaultMCPStartupTimeout, 1 * time.Second},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.value < tt.minValue {
				t.Errorf("%s = %v, should be >= %v", tt.name, tt.value, tt.minValue)
			}
		})
	}

	// Test backward compatibility with legacy integer constants
	legacyTests := []struct {
		name     string
		value    int
		minValue int
	}{
		{"DefaultAgenticWorkflowTimeoutMinutes", DefaultAgenticWorkflowTimeoutMinutes, 1},
		{"DefaultToolTimeoutSeconds", DefaultToolTimeoutSeconds, 1},
		{"DefaultMCPStartupTimeoutSeconds", DefaultMCPStartupTimeoutSeconds, 1},
	}

	for _, tt := range legacyTests {
		t.Run(tt.name+"_legacy", func(t *testing.T) {
			if tt.value < tt.minValue {
				t.Errorf("%s = %d, should be >= %d", tt.name, tt.value, tt.minValue)
			}
		})
	}

	// Test that legacy constants match the Duration-based values
	t.Run("legacy_compatibility", func(t *testing.T) {
		if DefaultAgenticWorkflowTimeoutMinutes != int(DefaultAgenticWorkflowTimeout/time.Minute) {
			t.Errorf("DefaultAgenticWorkflowTimeoutMinutes (%d) doesn't match DefaultAgenticWorkflowTimeout (%v)",
				DefaultAgenticWorkflowTimeoutMinutes, DefaultAgenticWorkflowTimeout)
		}
		if DefaultToolTimeoutSeconds != int(DefaultToolTimeout/time.Second) {
			t.Errorf("DefaultToolTimeoutSeconds (%d) doesn't match DefaultToolTimeout (%v)",
				DefaultToolTimeoutSeconds, DefaultToolTimeout)
		}
		if DefaultMCPStartupTimeoutSeconds != int(DefaultMCPStartupTimeout/time.Second) {
			t.Errorf("DefaultMCPStartupTimeoutSeconds (%d) doesn't match DefaultMCPStartupTimeout (%v)",
				DefaultMCPStartupTimeoutSeconds, DefaultMCPStartupTimeout)
		}
	})
}

func TestFeatureFlagConstants(t *testing.T) {
	// Test that feature flag constants have the correct type and values
	tests := []struct {
		name     string
		value    FeatureFlag
		expected string
	}{
		{"SafeInputsFeatureFlag", SafeInputsFeatureFlag, "safe-inputs"},
		{"MCPGatewayFeatureFlag", MCPGatewayFeatureFlag, "mcp-gateway"},
		{"SandboxRuntimeFeatureFlag", SandboxRuntimeFeatureFlag, "sandbox-runtime"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.value) != tt.expected {
				t.Errorf("%s = %q, want %q", tt.name, tt.value, tt.expected)
			}
		})
	}
}

func TestFeatureFlagType(t *testing.T) {
	// Test that FeatureFlag type can be used as expected
	var flag FeatureFlag = "test-flag"
	if string(flag) != "test-flag" {
		t.Errorf("FeatureFlag conversion failed: got %q, want %q", flag, "test-flag")
	}

	// Test that constants can be assigned to FeatureFlag variables
	safeInputsFlag := SafeInputsFeatureFlag
	if safeInputsFlag != "safe-inputs" {
		t.Errorf("SafeInputsFeatureFlag assignment failed: got %q, want %q", safeInputsFlag, "safe-inputs")
	}
}

func TestSemanticTypeAliases(t *testing.T) {
	// Test URL type
	t.Run("URL type", func(t *testing.T) {
		var testURL URL = "https://example.com"
		if string(testURL) != "https://example.com" {
			t.Errorf("URL conversion failed: got %q, want %q", testURL, "https://example.com")
		}

		// Test DefaultMCPRegistryURL has the correct type
		registryURL := DefaultMCPRegistryURL
		if string(registryURL) != "https://api.mcp.github.com/v0" {
			t.Errorf("DefaultMCPRegistryURL = %q, want %q", registryURL, "https://api.mcp.github.com/v0")
		}
	})

	// Test ModelName type
	t.Run("ModelName type", func(t *testing.T) {
		var testModel ModelName = "test-model"
		if string(testModel) != "test-model" {
			t.Errorf("ModelName conversion failed: got %q, want %q", testModel, "test-model")
		}

		// Test DefaultCopilotDetectionModel has the correct type
		detectionModel := DefaultCopilotDetectionModel
		if string(detectionModel) != "gpt-5-mini" {
			t.Errorf("DefaultCopilotDetectionModel = %q, want %q", detectionModel, "gpt-5-mini")
		}
	})

	// Test JobName type
	t.Run("JobName type", func(t *testing.T) {
		var testJob JobName = "test-job"
		if string(testJob) != "test-job" {
			t.Errorf("JobName conversion failed: got %q, want %q", testJob, "test-job")
		}

		// Test job name constants have the correct type
		agentJob := AgentJobName
		if string(agentJob) != "agent" {
			t.Errorf("AgentJobName = %q, want %q", agentJob, "agent")
		}

		activationJob := ActivationJobName
		if string(activationJob) != "activation" {
			t.Errorf("ActivationJobName = %q, want %q", activationJob, "activation")
		}

		preActivationJob := PreActivationJobName
		if string(preActivationJob) != "pre_activation" {
			t.Errorf("PreActivationJobName = %q, want %q", preActivationJob, "pre_activation")
		}

		detectionJob := DetectionJobName
		if string(detectionJob) != "detection" {
			t.Errorf("DetectionJobName = %q, want %q", detectionJob, "detection")
		}
	})

	// Test StepID type
	t.Run("StepID type", func(t *testing.T) {
		var testStep StepID = "test-step"
		if string(testStep) != "test-step" {
			t.Errorf("StepID conversion failed: got %q, want %q", testStep, "test-step")
		}

		// Test step ID constants have the correct type
		membershipStep := CheckMembershipStepID
		if string(membershipStep) != "check_membership" {
			t.Errorf("CheckMembershipStepID = %q, want %q", membershipStep, "check_membership")
		}

		stopTimeStep := CheckStopTimeStepID
		if string(stopTimeStep) != "check_stop_time" {
			t.Errorf("CheckStopTimeStepID = %q, want %q", stopTimeStep, "check_stop_time")
		}

		skipMatchStep := CheckSkipIfMatchStepID
		if string(skipMatchStep) != "check_skip_if_match" {
			t.Errorf("CheckSkipIfMatchStepID = %q, want %q", skipMatchStep, "check_skip_if_match")
		}

		commandPosStep := CheckCommandPositionStepID
		if string(commandPosStep) != "check_command_position" {
			t.Errorf("CheckCommandPositionStepID = %q, want %q", commandPosStep, "check_command_position")
		}
	})

	// Test CommandPrefix type
	t.Run("CommandPrefix type", func(t *testing.T) {
		var testPrefix CommandPrefix = "test-prefix"
		if string(testPrefix) != "test-prefix" {
			t.Errorf("CommandPrefix conversion failed: got %q, want %q", testPrefix, "test-prefix")
		}

		// Test CLIExtensionPrefix has the correct type
		cliPrefix := CLIExtensionPrefix
		if string(cliPrefix) != "gh aw" {
			t.Errorf("CLIExtensionPrefix = %q, want %q", cliPrefix, "gh aw")
		}
	})
}

func TestTypeSafetyBetweenSemanticTypes(t *testing.T) {
	// This test demonstrates that semantic types provide type safety
	// by preventing accidental mixing of different string types

	// These assignments should work (same types)
	job1 := AgentJobName
	job2 := ActivationJobName
	if job1 == job2 {
		t.Error("AgentJobName should not equal ActivationJobName")
	}

	step1 := CheckMembershipStepID
	step2 := CheckStopTimeStepID
	if step1 == step2 {
		t.Error("CheckMembershipStepID should not equal CheckStopTimeStepID")
	}

	// Verify that we can still convert to string when needed
	if string(job1) != "agent" {
		t.Errorf("JobName string conversion failed: got %q, want %q", job1, "agent")
	}

	if string(step1) != "check_membership" {
		t.Errorf("StepID string conversion failed: got %q, want %q", step1, "check_membership")
	}

	// Verify that different semantic types have different underlying types
	// (this is a compile-time check, but we verify the values are correct)
	jobStr := string(AgentJobName)
	stepStr := string(CheckMembershipStepID)
	_ = jobStr  // Used for demonstration
	_ = stepStr // Used for demonstration
	// Different semantic types prevent accidental mixing even if string values match
}
