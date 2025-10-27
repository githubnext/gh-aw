package workflow

import (
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/githubnext/gh-aw/pkg/constants"
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

	if !engine.SupportsToolsAllowlist() {
		t.Error("Codex engine should support MCP tools")
	}

	// Test installation steps
	steps := engine.GetInstallationSteps(&WorkflowData{})
	expectedStepCount := 3 // Secret validation + Node.js setup + Install Codex
	if len(steps) != expectedStepCount {
		t.Errorf("Expected %d installation steps, got %d", expectedStepCount, len(steps))
	}

	// Verify first step is secret validation
	if len(steps) > 0 && len(steps[0]) > 0 {
		if !strings.Contains(steps[0][0], "Validate CODEX_API_KEY or OPENAI_API_KEY secret") {
			t.Errorf("Expected first step to contain 'Validate CODEX_API_KEY or OPENAI_API_KEY secret', got '%s'", steps[0][0])
		}
	}

	// Verify second step is Node.js setup
	if len(steps) > 1 && len(steps[1]) > 0 {
		if !strings.Contains(steps[1][0], "Setup Node.js") {
			t.Errorf("Expected second step to contain 'Setup Node.js', got '%s'", steps[1][0])
		}
	}

	// Verify third step is Install Codex
	if len(steps) > 2 && len(steps[2]) > 0 {
		if !strings.Contains(steps[2][0], "Install Codex") {
			t.Errorf("Expected third step to contain 'Install Codex', got '%s'", steps[2][0])
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

	if !strings.Contains(stepContent, "codex") {
		t.Errorf("Expected command to contain 'codex' in step content:\n%s", stepContent)
	}

	if !strings.Contains(stepContent, "exec") {
		t.Errorf("Expected command to contain 'exec' in step content:\n%s", stepContent)
	}

	if !strings.Contains(stepContent, "test-log") {
		t.Errorf("Expected command to contain log file name in step content:\n%s", stepContent)
	}

	// Check that pipefail is enabled to preserve exit codes
	if !strings.Contains(stepContent, "set -o pipefail") {
		t.Errorf("Expected command to contain 'set -o pipefail' to preserve exit codes in step content:\n%s", stepContent)
	}

	// Check environment variables
	if !strings.Contains(stepContent, "CODEX_API_KEY: ${{ secrets.CODEX_API_KEY || secrets.OPENAI_API_KEY }}") {
		t.Errorf("Expected CODEX_API_KEY environment variable in step content:\n%s", stepContent)
	}
}

func TestCodexEngineWithVersion(t *testing.T) {
	engine := NewCodexEngine()

	// Test installation steps without version (should use pinned default version)
	stepsNoVersion := engine.GetInstallationSteps(&WorkflowData{})
	foundNoVersionInstall := false
	expectedVersion := fmt.Sprintf("npm install -g @openai/codex@%s", constants.DefaultCodexVersion)
	for _, step := range stepsNoVersion {
		for _, line := range step {
			if strings.Contains(line, expectedVersion) {
				foundNoVersionInstall = true
				break
			}
		}
	}
	if !foundNoVersionInstall {
		t.Errorf("Expected npm install command with @%s when no version specified", constants.DefaultCodexVersion)
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
		"uses":              "actions/checkout@08c6903cd8c0fde910a37f88322edcfb5dd907a8",
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

	steps := engine.GetExecutionSteps(workflowData, "/tmp/gh-aw/test.log")

	// Should have at least one step
	if len(steps) == 0 {
		t.Error("Expected at least one execution step")
		return
	}

	// Check that GH_AW_PROMPT environment variable is included
	foundPromptEnv := false
	foundMCPConfigEnv := false
	for _, step := range steps {
		stepContent := strings.Join([]string(step), "\n")
		if strings.Contains(stepContent, "GH_AW_PROMPT: /tmp/gh-aw/aw-prompts/prompt.txt") {
			foundPromptEnv = true
		}
		if strings.Contains(stepContent, "GH_AW_MCP_CONFIG: /tmp/gh-aw/mcp-config/config.toml") {
			foundMCPConfigEnv = true
		}
	}

	if !foundPromptEnv {
		t.Error("Expected GH_AW_PROMPT environment variable in codex execution steps")
	}

	if !foundMCPConfigEnv {
		t.Error("Expected GH_AW_MCP_CONFIG environment variable in codex execution steps")
	}
}

func TestCodexEngineConvertStepToYAMLWithSection(t *testing.T) {
	engine := NewCodexEngine()

	// Test step with 'with' section to ensure keys are sorted
	stepMap := map[string]any{
		"name": "Test step with sorted with section",
		"uses": "actions/checkout@08c6903cd8c0fde910a37f88322edcfb5dd907a8",
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

func TestCodexEngineRenderMCPConfig(t *testing.T) {
	engine := NewCodexEngine()

	tests := []struct {
		name     string
		tools    map[string]any
		mcpTools []string
		expected []string
	}{
		{
			name: "github tool with user_agent",
			tools: map[string]any{
				"github": map[string]any{},
			},
			mcpTools: []string{"github"},
			expected: []string{
				"cat > /tmp/gh-aw/mcp-config/config.toml << EOF",
				"[history]",
				"persistence = \"none\"",
				"",
				"[mcp_servers.github]",
				"user_agent = \"test-workflow\"",
				"startup_timeout_sec = 120",
				"tool_timeout_sec = 60",
				"command = \"docker\"",
				"args = [",
				"\"run\",",
				"\"-i\",",
				"\"--rm\",",
				"\"-e\",",
				"\"GITHUB_PERSONAL_ACCESS_TOKEN\",",
				"\"-e\",",
				"\"GITHUB_READ_ONLY=1\",",
				"\"-e\",",
				"\"GITHUB_TOOLSETS=default\",",
				"\"ghcr.io/github/github-mcp-server:v0.20.0\"",
				"]",
				"",
				"[mcp_servers.github.env]",
				"GITHUB_PERSONAL_ACCESS_TOKEN = \"${{ secrets.GH_AW_GITHUB_TOKEN || secrets.GITHUB_TOKEN }}\"",
				"EOF",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var yaml strings.Builder
			workflowData := &WorkflowData{Name: "test-workflow"}
			engine.RenderMCPConfig(&yaml, tt.tools, tt.mcpTools, workflowData)

			result := yaml.String()
			lines := strings.Split(strings.TrimSpace(result), "\n")

			// Remove indentation from both expected and actual lines for comparison
			var normalizedResult []string
			for _, line := range lines {
				normalizedResult = append(normalizedResult, strings.TrimSpace(line))
			}

			var normalizedExpected []string
			for _, line := range tt.expected {
				normalizedExpected = append(normalizedExpected, strings.TrimSpace(line))
			}

			if len(normalizedResult) != len(normalizedExpected) {
				t.Errorf("Expected %d lines, got %d", len(normalizedExpected), len(normalizedResult))
				t.Errorf("Expected:\n%s", strings.Join(normalizedExpected, "\n"))
				t.Errorf("Got:\n%s", strings.Join(normalizedResult, "\n"))
				return
			}

			for i, expectedLine := range normalizedExpected {
				if i < len(normalizedResult) {
					actualLine := normalizedResult[i]
					if actualLine != expectedLine {
						t.Errorf("Line %d mismatch:\nExpected: %s\nActual:   %s", i+1, expectedLine, actualLine)
					}
				}
			}
		})
	}
}

func TestCodexEngineUserAgentIdentifierConversion(t *testing.T) {
	engine := NewCodexEngine()

	tests := []struct {
		name         string
		workflowName string
		expectedUA   string
	}{
		{
			name:         "workflow name with spaces",
			workflowName: "Test Codex Create Issue",
			expectedUA:   "test-codex-create-issue",
		},
		{
			name:         "workflow name with underscores",
			workflowName: "Test_Workflow_Name",
			expectedUA:   "test-workflow-name",
		},
		{
			name:         "already identifier format",
			workflowName: "test-workflow",
			expectedUA:   "test-workflow",
		},
		{
			name:         "empty workflow name",
			workflowName: "",
			expectedUA:   "github-agentic-workflow",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var yaml strings.Builder
			workflowData := &WorkflowData{Name: tt.workflowName}

			tools := map[string]any{"github": map[string]any{}}
			mcpTools := []string{"github"}

			engine.RenderMCPConfig(&yaml, tools, mcpTools, workflowData)

			result := yaml.String()
			expectedUserAgentLine := "user_agent = \"" + tt.expectedUA + "\""

			if !strings.Contains(result, expectedUserAgentLine) {
				t.Errorf("Expected MCP config to contain %q, got:\n%s", expectedUserAgentLine, result)
			}
		})
	}
}

func TestCodexEngineRenderMCPConfigUserAgentFromConfig(t *testing.T) {
	engine := NewCodexEngine()

	tests := []struct {
		name         string
		workflowName string
		configuredUA string
		expectedUA   string
		description  string
	}{
		{
			name:         "configured user_agent overrides workflow name",
			workflowName: "Test Workflow Name",
			configuredUA: "my-custom-agent",
			expectedUA:   "my-custom-agent",
			description:  "When user_agent is configured, it should be used instead of the converted workflow name",
		},
		{
			name:         "configured user_agent with spaces",
			workflowName: "test-workflow",
			configuredUA: "My Custom User Agent",
			expectedUA:   "My Custom User Agent",
			description:  "Configured user_agent should be used as-is, without identifier conversion",
		},
		{
			name:         "empty configured user_agent falls back to workflow name",
			workflowName: "Test Workflow",
			configuredUA: "",
			expectedUA:   "test-workflow",
			description:  "Empty configured user_agent should fall back to workflow name conversion",
		},
		{
			name:         "no workflow name and no configured user_agent uses default",
			workflowName: "",
			configuredUA: "",
			expectedUA:   "github-agentic-workflow",
			description:  "Should use default when neither workflow name nor user_agent is configured",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var yaml strings.Builder

			engineConfig := &EngineConfig{
				ID: "codex",
			}
			if tt.configuredUA != "" {
				engineConfig.UserAgent = tt.configuredUA
			}

			workflowData := &WorkflowData{
				Name:         tt.workflowName,
				EngineConfig: engineConfig,
			}

			tools := map[string]any{"github": map[string]any{}}
			mcpTools := []string{"github"}

			engine.RenderMCPConfig(&yaml, tools, mcpTools, workflowData)

			result := yaml.String()
			expectedUserAgentLine := "user_agent = \"" + tt.expectedUA + "\""

			if !strings.Contains(result, expectedUserAgentLine) {
				t.Errorf("Test case: %s\nExpected MCP config to contain %q, got:\n%s", tt.description, expectedUserAgentLine, result)
			}
		})
	}
}

func TestSanitizeIdentifier(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple name with spaces",
			input:    "Test Codex Create Issue",
			expected: "test-codex-create-issue",
		},
		{
			name:     "name with underscores",
			input:    "Test_Workflow_Name",
			expected: "test-workflow-name",
		},
		{
			name:     "name with mixed separators",
			input:    "Test Workflow_Name With Spaces",
			expected: "test-workflow-name-with-spaces",
		},
		{
			name:     "name with special characters",
			input:    "Test@Workflow#With$Special%Characters!",
			expected: "testworkflowwithspecialcharacters",
		},
		{
			name:     "name with multiple spaces",
			input:    "Test   Multiple    Spaces",
			expected: "test-multiple-spaces",
		},
		{
			name:     "empty name",
			input:    "",
			expected: "github-agentic-workflow",
		},
		{
			name:     "name with only special characters",
			input:    "@#$%!",
			expected: "github-agentic-workflow",
		},
		{
			name:     "already lowercase with hyphens",
			input:    "already-lowercase-name",
			expected: "already-lowercase-name",
		},
		{
			name:     "name with leading/trailing spaces",
			input:    "  Test Workflow  ",
			expected: "test-workflow",
		},
		{
			name:     "name with hyphens and underscores",
			input:    "Test-Workflow_Name",
			expected: "test-workflow-name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeIdentifier(tt.input)
			if result != tt.expected {
				t.Errorf("SanitizeIdentifier(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestCodexEngineRenderMCPConfigUserAgentWithHyphen(t *testing.T) {
	engine := NewCodexEngine()

	// Test that "user-agent" field name works
	tests := []struct {
		name             string
		engineConfigFunc func() *EngineConfig
		expectedUA       string
		description      string
	}{
		{
			name: "user-agent field gets parsed as user_agent (hyphen)",
			engineConfigFunc: func() *EngineConfig {
				// This simulates the parsing of "user-agent" from frontmatter
				// which gets stored in the UserAgent field
				return &EngineConfig{
					ID:        "codex",
					UserAgent: "custom-agent-hyphen",
				}
			},
			expectedUA:  "custom-agent-hyphen",
			description: "user-agent field with hyphen should be parsed and work",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var yaml strings.Builder

			workflowData := &WorkflowData{
				Name:         "test-workflow",
				EngineConfig: tt.engineConfigFunc(),
			}

			tools := map[string]any{"github": map[string]any{}}
			mcpTools := []string{"github"}

			engine.RenderMCPConfig(&yaml, tools, mcpTools, workflowData)

			result := yaml.String()
			expectedUserAgentLine := "user_agent = \"" + tt.expectedUA + "\""

			if !strings.Contains(result, expectedUserAgentLine) {
				t.Errorf("Test case: %s\nExpected MCP config to contain %q, got:\n%s", tt.description, expectedUserAgentLine, result)
			}
		})
	}
}

func TestCodexEngineErrorPatterns(t *testing.T) {
	engine := NewCodexEngine()
	patterns := engine.GetErrorPatterns()

	// Test Rust format patterns (new format)
	rustFormatTests := []struct {
		name          string
		logLine       string
		shouldMatch   bool
		expectedLevel string
	}{
		{
			name:          "Rust ERROR format with milliseconds",
			logLine:       "2025-08-31T12:37:08.123Z ERROR This is a test error message",
			shouldMatch:   true,
			expectedLevel: "ERROR",
		},
		{
			name:          "Rust WARN format with milliseconds",
			logLine:       "2025-08-31T12:37:09.456Z WARN This is a test warning message",
			shouldMatch:   true,
			expectedLevel: "WARN",
		},
		{
			name:          "Rust WARNING format with milliseconds",
			logLine:       "2025-08-31T12:37:10.789Z WARNING This is a test warning message",
			shouldMatch:   true,
			expectedLevel: "WARNING",
		},
		{
			name:        "Rust INFO format should not match error patterns",
			logLine:     "2025-08-31T12:37:11.012Z INFO This is a test info message",
			shouldMatch: false,
		},
		{
			name:        "Rust DEBUG format should not match error patterns",
			logLine:     "2025-08-31T12:37:12.345Z DEBUG This is a test debug message",
			shouldMatch: false,
		},
	}

	for _, tt := range rustFormatTests {
		t.Run(tt.name, func(t *testing.T) {
			matched := false
			matchedLevel := ""
			for _, pattern := range patterns {
				if strings.Contains(pattern.Description, "Codex ERROR messages with timestamp") ||
					strings.Contains(pattern.Description, "Codex warning messages with timestamp") {
					re := compilePattern(t, pattern.Pattern)
					if re.MatchString(tt.logLine) {
						matched = true
						matches := re.FindStringSubmatch(tt.logLine)
						if pattern.LevelGroup > 0 && pattern.LevelGroup < len(matches) {
							matchedLevel = matches[pattern.LevelGroup]
						}
						break
					}
				}
			}

			if matched != tt.shouldMatch {
				t.Errorf("Pattern matching mismatch for %q: expected match=%v, got match=%v", tt.logLine, tt.shouldMatch, matched)
			}

			if tt.shouldMatch && matchedLevel != tt.expectedLevel {
				t.Errorf("Level extraction mismatch for %q: expected %q, got %q", tt.logLine, tt.expectedLevel, matchedLevel)
			}
		})
	}
}

func compilePattern(t *testing.T, pattern string) *regexp.Regexp {
	re, err := regexp.Compile(pattern)
	if err != nil {
		t.Fatalf("Failed to compile pattern %q: %v", pattern, err)
	}
	return re
}
