package workflow

import (
	"strings"
	"testing"
)

func TestCodexEngine(t *testing.T) {
	engine := NewCodexEngine()

	// Test basic properties
	if engine.GetID() != "codex" {
		t.Errorf("Expected ID 'codex', got '%s'", engine.GetID())
	}

	if engine.GetDisplayName() != "Codex" {
		t.Errorf("Expected display name 'Codex', got '%s'", engine.GetDisplayName())
	}

	if !engine.IsExperimental() {
		t.Error("Codex engine should be experimental")
	}

	if !engine.SupportsToolsWhitelist() {
		t.Error("Codex engine should support MCP tools")
	}

	// Test installation steps
	steps := engine.GetInstallationSteps(&WorkflowData{})
	expectedStepCount := 2 // Setup Node.js and Install Codex
	if len(steps) != expectedStepCount {
		t.Errorf("Expected %d installation steps, got %d", expectedStepCount, len(steps))
	}

	// Verify first step is Setup Node.js
	if len(steps) > 0 && len(steps[0]) > 0 {
		if !strings.Contains(steps[0][0], "Setup Node.js") {
			t.Errorf("Expected first step to contain 'Setup Node.js', got '%s'", steps[0][0])
		}
	}

	// Verify second step is Install Codex
	if len(steps) > 1 && len(steps[1]) > 0 {
		if !strings.Contains(steps[1][0], "Install Codex") {
			t.Errorf("Expected second step to contain 'Install Codex', got '%s'", steps[1][0])
		}
	}

	// Test execution steps
	workflowData := &WorkflowData{
		Name: "test-workflow",
	}
	execSteps := engine.GetExecutionSteps(workflowData, "test-log")
	if len(execSteps) != 1 {
		t.Fatalf("Expected 1 step for Codex execution, got %d", len(execSteps))
	}

	// Check the execution step
	stepContent := strings.Join([]string(execSteps[0]), "\n")

	if !strings.Contains(stepContent, "name: Run Codex") {
		t.Errorf("Expected step name 'Run Codex' in step content:\n%s", stepContent)
	}

	if strings.Contains(stepContent, "uses:") {
		t.Errorf("Expected no action for Codex (uses command), got step with 'uses:' in:\n%s", stepContent)
	}

	if !strings.Contains(stepContent, "codex exec") {
		t.Errorf("Expected command to contain 'codex exec' in step content:\n%s", stepContent)
	}

	if !strings.Contains(stepContent, "test-log") {
		t.Errorf("Expected command to contain log file name in step content:\n%s", stepContent)
	}

	// Check that pipefail is enabled to preserve exit codes
	if !strings.Contains(stepContent, "set -o pipefail") {
		t.Errorf("Expected command to contain 'set -o pipefail' to preserve exit codes in step content:\n%s", stepContent)
	}

	// Check environment variables
	if !strings.Contains(stepContent, "OPENAI_API_KEY: ${{ secrets.OPENAI_API_KEY }}") {
		t.Errorf("Expected OPENAI_API_KEY environment variable in step content:\n%s", stepContent)
	}
}

func TestCodexEngineWithVersion(t *testing.T) {
	engine := NewCodexEngine()

	// Test installation steps without version
	stepsNoVersion := engine.GetInstallationSteps(&WorkflowData{})
	foundNoVersionInstall := false
	for _, step := range stepsNoVersion {
		for _, line := range step {
			if strings.Contains(line, "npm install -g @openai/codex") && !strings.Contains(line, "@openai/codex@") {
				foundNoVersionInstall = true
				break
			}
		}
	}
	if !foundNoVersionInstall {
		t.Error("Expected default npm install command without version")
	}

	// Test installation steps with version
	engineConfig := &EngineConfig{
		ID:      "codex",
		Version: "3.0.1",
	}
	workflowData := &WorkflowData{
		EngineConfig: engineConfig,
	}
	stepsWithVersion := engine.GetInstallationSteps(workflowData)
	foundVersionInstall := false
	for _, step := range stepsWithVersion {
		for _, line := range step {
			if strings.Contains(line, "npm install -g @openai/codex@3.0.1") {
				foundVersionInstall = true
				break
			}
		}
	}
	if !foundVersionInstall {
		t.Error("Expected versioned npm install command with @openai/codex@3.0.1")
	}
}

func TestCodexEngineConvertStepToYAMLWithIdAndContinueOnError(t *testing.T) {
	engine := NewCodexEngine()

	// Test step with id and continue-on-error fields
	stepMap := map[string]any{
		"name":              "Test step with id and continue-on-error",
		"id":                "test-step",
		"continue-on-error": true,
		"run":               "echo 'test'",
	}

	yaml, err := engine.convertStepToYAML(stepMap)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Check that id field is included
	if !strings.Contains(yaml, "id: test-step") {
		t.Errorf("Expected YAML to contain 'id: test-step', got:\n%s", yaml)
	}

	// Check that continue-on-error field is included
	if !strings.Contains(yaml, "continue-on-error: true") {
		t.Errorf("Expected YAML to contain 'continue-on-error: true', got:\n%s", yaml)
	}

	// Test with string continue-on-error
	stepMap2 := map[string]any{
		"name":              "Test step with string continue-on-error",
		"id":                "test-step-2",
		"continue-on-error": "false",
		"uses":              "actions/checkout@v4",
	}

	yaml2, err := engine.convertStepToYAML(stepMap2)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Check that continue-on-error field is included as string
	if !strings.Contains(yaml2, "continue-on-error: \"false\"") {
		t.Errorf("Expected YAML to contain 'continue-on-error: \"false\"', got:\n%s", yaml2)
	}
}

func TestCodexEngineExecutionIncludesGitHubAWPrompt(t *testing.T) {
	engine := NewCodexEngine()

	workflowData := &WorkflowData{
		Name: "test-workflow",
	}

	steps := engine.GetExecutionSteps(workflowData, "/tmp/test.log")

	// Should have at least one step
	if len(steps) == 0 {
		t.Error("Expected at least one execution step")
		return
	}

	// Check that GITHUB_AW_PROMPT environment variable is included
	foundPromptEnv := false
	for _, step := range steps {
		stepContent := strings.Join([]string(step), "\n")
		if strings.Contains(stepContent, "GITHUB_AW_PROMPT: /tmp/aw-prompts/prompt.txt") {
			foundPromptEnv = true
			break
		}
	}

	if !foundPromptEnv {
		t.Error("Expected GITHUB_AW_PROMPT environment variable in codex execution steps")
	}
}

func TestCodexEngineConvertStepToYAMLWithSection(t *testing.T) {
	engine := NewCodexEngine()

	// Test step with 'with' section to ensure keys are sorted
	stepMap := map[string]any{
		"name": "Test step with sorted with section",
		"uses": "actions/checkout@v4",
		"with": map[string]any{
			"zebra": "value-z",
			"alpha": "value-a",
			"beta":  "value-b",
			"gamma": "value-g",
		},
	}

	yaml, err := engine.convertStepToYAML(stepMap)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Verify that the with keys are in alphabetical order
	lines := strings.Split(yaml, "\n")
	withSection := false
	withKeyOrder := []string{}

	for _, line := range lines {
		if strings.TrimSpace(line) == "with:" {
			withSection = true
			continue
		}
		if withSection && strings.HasPrefix(strings.TrimSpace(line), "- ") {
			// End of with section if we hit another top-level key
			break
		}
		if withSection && strings.Contains(line, ":") {
			// Extract the key (before the colon)
			parts := strings.SplitN(strings.TrimSpace(line), ":", 2)
			if len(parts) > 0 {
				withKeyOrder = append(withKeyOrder, strings.TrimSpace(parts[0]))
			}
		}
	}

	expectedOrder := []string{"alpha", "beta", "gamma", "zebra"}
	if len(withKeyOrder) != len(expectedOrder) {
		t.Errorf("Expected %d with keys, got %d", len(expectedOrder), len(withKeyOrder))
	}

	for i, key := range expectedOrder {
		if i >= len(withKeyOrder) || withKeyOrder[i] != key {
			t.Errorf("Expected with key at position %d to be '%s', got '%s'. Full order: %v", i, key, withKeyOrder[i], withKeyOrder)
		}
	}
}

func TestCodexEngineAuthenticationFailureDetection(t *testing.T) {
	engine := NewCodexEngine()

	workflowData := &WorkflowData{
		Name: "test-workflow",
	}

	steps := engine.GetExecutionSteps(workflowData, "/tmp/test.log")

	// Should have at least one step
	if len(steps) == 0 {
		t.Error("Expected at least one execution step")
		return
	}

	// Check that the execution step includes authentication failure detection
	stepContent := strings.Join([]string(steps[0]), "\n")

	// Verify the step contains error detection logic
	if !strings.Contains(stepContent, "401 Unauthorized") {
		t.Errorf("Expected step to contain authentication error detection for '401 Unauthorized' pattern in:\n%s", stepContent)
	}

	if !strings.Contains(stepContent, "exceeded retry limit.*401 Unauthorized") {
		t.Errorf("Expected step to contain authentication error detection for 'exceeded retry limit.*401 Unauthorized' pattern in:\n%s", stepContent)
	}

	if !strings.Contains(stepContent, "Codex authentication failed") {
		t.Errorf("Expected step to contain authentication failure message in:\n%s", stepContent)
	}

	if !strings.Contains(stepContent, "exit 1") {
		t.Errorf("Expected step to exit with error code 1 on authentication failure in:\n%s", stepContent)
	}

	if !strings.Contains(stepContent, "OPENAI_API_KEY secret is properly configured") {
		t.Errorf("Expected step to contain instructions about OPENAI_API_KEY configuration in:\n%s", stepContent)
	}

	// Verify the error detection happens after codex execution
	codexIndex := strings.Index(stepContent, "codex exec")
	errorCheckIndex := strings.Index(stepContent, "401 Unauthorized")

	if codexIndex == -1 {
		t.Error("Expected step to contain 'codex exec' command")
	}

	if errorCheckIndex == -1 {
		t.Error("Expected step to contain authentication error check")
	}

	if codexIndex != -1 && errorCheckIndex != -1 && errorCheckIndex <= codexIndex {
		t.Error("Expected authentication error check to come after codex execution")
	}

	// Verify the error check only runs if log file exists
	if !strings.Contains(stepContent, "if [ -f /tmp/test.log ]; then") {
		t.Error("Expected authentication error check to only run if log file exists")
	}
}

func TestCodexEngineErrorDetectionPatterns(t *testing.T) {
	tests := []struct {
		name        string
		logContent  string
		shouldFail  bool
		description string
	}{
		{
			name:        "no_errors",
			logContent:  "[2025-01-15] Starting Codex execution\n[2025-01-15] Task completed successfully",
			shouldFail:  false,
			description: "Normal execution should not trigger authentication failure",
		},
		{
			name:        "401_unauthorized_direct",
			logContent:  "[2025-01-15] ERROR: 401 Unauthorized",
			shouldFail:  true,
			description: "Direct 401 Unauthorized error should trigger failure",
		},
		{
			name:        "exceeded_retry_limit_with_401",
			logContent:  "[2025-01-15] ERROR: exceeded retry limit, last status: 401 Unauthorized",
			shouldFail:  true,
			description: "Exceeded retry limit with 401 should trigger failure",
		},
		{
			name:        "retry_messages_with_401",
			logContent:  "[2025-01-15] stream error: exceeded retry limit, last status: 401 Unauthorized; retrying 1/5 in 216ms…",
			shouldFail:  true,
			description: "Retry messages with 401 should trigger failure",
		},
		{
			name:        "other_4xx_errors",
			logContent:  "[2025-01-15] ERROR: 403 Forbidden",
			shouldFail:  false,
			description: "Other HTTP errors like 403 should not trigger authentication failure detection",
		},
		{
			name:        "exceeded_retry_limit_other_error",
			logContent:  "[2025-01-15] ERROR: exceeded retry limit, last status: 500 Internal Server Error",
			shouldFail:  false,
			description: "Exceeded retry limit with non-401 errors should not trigger authentication failure detection",
		},
		{
			name:        "mixed_errors_including_401",
			logContent:  "[2025-01-15] ERROR: 500 Internal Server Error\n[2025-01-15] stream error: exceeded retry limit, last status: 401 Unauthorized; retrying 2/5 in 414ms…\n[2025-01-15] Task completed",
			shouldFail:  true,
			description: "Mixed errors including 401 should trigger failure",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test the grep pattern used in the authentication detection
			// This simulates what happens in the actual bash command

			matched := false
			lines := strings.Split(tt.logContent, "\n")
			for _, line := range lines {
				// Simple pattern matching for test purposes
				if strings.Contains(line, "401 Unauthorized") {
					matched = true
					break
				}
				if strings.Contains(line, "exceeded retry limit") && strings.Contains(line, "401 Unauthorized") {
					matched = true
					break
				}
			}

			if matched != tt.shouldFail {
				t.Errorf("Test %s failed: pattern matching result %v, expected %v for content:\n%s",
					tt.name, matched, tt.shouldFail, tt.logContent)
			}
		})
	}
}
