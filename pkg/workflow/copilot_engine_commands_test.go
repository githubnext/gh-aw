//go:build !integration

package workflow

import (
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/semverutil"

	"github.com/github/gh-aw/pkg/constants"
)

func TestCopilotEngineExecutionStepsWithToolArguments(t *testing.T) {
	engine := NewCopilotEngine()
	workflowData := &WorkflowData{
		Name: "test-workflow",
		Tools: map[string]any{
			"bash": []any{"echo", "git status"},
			"edit": nil,
		},
		ParsedTools: NewTools(map[string]any{
			"bash": []any{"echo", "git status"},
			"edit": nil,
		}),
	}
	steps := engine.GetExecutionSteps(workflowData, "/tmp/gh-aw/test.log")

	if len(steps) != 1 {
		t.Fatalf("Expected 1 execution step, got %d", len(steps))
	}

	// Check the execution step contains tool arguments
	stepContent := strings.Join([]string(steps[0]), "\n")

	// Should contain the tool arguments in the command line
	if !strings.Contains(stepContent, "--allow-tool shell(echo)") {
		t.Errorf("Expected step to contain '--allow-tool shell(echo)' in command:\n%s", stepContent)
	}

	if !strings.Contains(stepContent, "--allow-tool shell(git status)") {
		t.Errorf("Expected step to contain '--allow-tool shell(git status)' in command:\n%s", stepContent)
	}

	if !strings.Contains(stepContent, "--allow-tool write") {
		t.Errorf("Expected step to contain '--allow-tool write' in command:\n%s", stepContent)
	}

	// Should contain the comment showing the tool arguments
	if !strings.Contains(stepContent, "# Copilot CLI tool arguments (sorted):") {
		t.Errorf("Expected step to contain tool arguments comment:\n%s", stepContent)
	}

	if !strings.Contains(stepContent, "# --allow-tool shell(echo)") {
		t.Errorf("Expected step to contain comment for shell(echo):\n%s", stepContent)
	}

	if !strings.Contains(stepContent, "# --allow-tool write") {
		t.Errorf("Expected step to contain comment for write:\n%s", stepContent)
	}

	// Should contain --allow-all-paths for edit tool
	if !strings.Contains(stepContent, "--allow-all-paths") {
		t.Errorf("Expected step to contain '--allow-all-paths' for edit tool:\n%s", stepContent)
	}
}

func TestCopilotEngineEditToolAddsAllowAllPaths(t *testing.T) {
	engine := NewCopilotEngine()

	tests := []struct {
		name       string
		tools      map[string]any
		shouldHave bool
	}{
		{
			name: "edit tool present",
			tools: map[string]any{
				"edit": nil,
			},
			shouldHave: true,
		},
		{
			name: "edit tool with other tools",
			tools: map[string]any{
				"edit": nil,
				"bash": []any{"echo"},
			},
			shouldHave: true,
		},
		{
			name: "no edit tool",
			tools: map[string]any{
				"bash": []any{"echo"},
			},
			shouldHave: false,
		},
		{
			name:       "empty tools",
			tools:      map[string]any{},
			shouldHave: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			workflowData := &WorkflowData{
				Name:        "test-workflow",
				Tools:       tt.tools,
				ParsedTools: NewTools(tt.tools), // Populate ParsedTools from Tools map
			}
			steps := engine.GetExecutionSteps(workflowData, "/tmp/gh-aw/test.log")

			if len(steps) != 1 {
				t.Fatalf("Expected 1 step, got %d", len(steps))
			}

			stepContent := strings.Join([]string(steps[0]), "\n")

			// Check for --allow-all-paths flag
			hasAllowAllPaths := strings.Contains(stepContent, "--allow-all-paths")

			if tt.shouldHave && !hasAllowAllPaths {
				t.Errorf("Expected step to contain '--allow-all-paths' when edit tool is present, but it was missing:\n%s", stepContent)
			}

			if !tt.shouldHave && hasAllowAllPaths {
				t.Errorf("Expected step to NOT contain '--allow-all-paths' when edit tool is absent, but it was present:\n%s", stepContent)
			}

			// When edit tool is present, verify it's in the command line
			if tt.shouldHave {
				lines := strings.Split(stepContent, "\n")
				foundInCommand := false
				for _, line := range lines {
					// When firewall is disabled, it uses 'copilot' instead of 'npx'
					if strings.Contains(line, "copilot") && strings.Contains(line, "--allow-all-paths") {
						foundInCommand = true
						break
					}
				}
				if !foundInCommand {
					t.Errorf("Expected '--allow-all-paths' in copilot command line:\n%s", stepContent)
				}
			}
		})
	}
}

func TestCopilotEngineShellEscaping(t *testing.T) {
	engine := NewCopilotEngine()
	workflowData := &WorkflowData{
		Name: "test-workflow",
		Tools: map[string]any{
			"bash": []any{"git add:*", "git commit:*"},
		},
	}
	steps := engine.GetExecutionSteps(workflowData, "/tmp/gh-aw/test.log")

	if len(steps) != 1 {
		t.Fatalf("Expected 1 step, got %d", len(steps))
	}

	// Get the full command from the execution step
	stepContent := strings.Join([]string(steps[0]), "\n")

	// Find the line that contains the copilot command
	// When firewall is disabled, it uses 'copilot' instead of 'npx'
	lines := strings.Split(stepContent, "\n")
	var copilotCommand string
	for _, line := range lines {
		if strings.Contains(line, "copilot") && strings.Contains(line, "--allow-tool") {
			copilotCommand = strings.TrimSpace(line)
			break
		}
	}

	if copilotCommand == "" {
		t.Fatalf("Could not find copilot command in step content:\n%s", stepContent)
	}

	// Verify that arguments with special characters are properly quoted
	// This test should fail initially, showing the need for escaping
	t.Logf("Generated command: %s", copilotCommand)

	// The command should contain properly escaped arguments with single quotes
	if !strings.Contains(copilotCommand, "'shell(git add:*)'") {
		t.Errorf("Expected 'shell(git add:*)' to be single-quoted in command: %s", copilotCommand)
	}

	if !strings.Contains(copilotCommand, "'shell(git commit:*)'") {
		t.Errorf("Expected 'shell(git commit:*)' to be single-quoted in command: %s", copilotCommand)
	}
}

func TestCopilotEnginePromptFilePath(t *testing.T) {
	engine := NewCopilotEngine()
	workflowData := &WorkflowData{
		Name: "test-workflow",
		Tools: map[string]any{
			"bash": []any{"git status"},
		},
	}
	steps := engine.GetExecutionSteps(workflowData, "/tmp/gh-aw/test.log")

	if len(steps) != 1 {
		t.Fatalf("Expected 1 step, got %d", len(steps))
	}

	// Get the full command from the execution step
	stepContent := strings.Join([]string(steps[0]), "\n")

	// Find the line that contains the copilot command
	// When firewall is disabled, it uses 'copilot' instead of 'npx'
	lines := strings.Split(stepContent, "\n")
	var copilotCommand string
	for _, line := range lines {
		if strings.Contains(line, "copilot") && strings.Contains(line, "--prompt") {
			copilotCommand = strings.TrimSpace(line)
			break
		}
	}

	if copilotCommand == "" {
		t.Fatalf("Could not find copilot command in step content:\n%s", stepContent)
	}

	if !strings.Contains(copilotCommand, "--prompt-file /tmp/gh-aw/aw-prompts/prompt.txt") {
		t.Errorf("Expected prompt to be passed via --prompt-file, got: %s", copilotCommand)
	}

	if strings.Contains(copilotCommand, "--prompt ") {
		t.Errorf("Expected no inline --prompt argument expansion, got: %s", copilotCommand)
	}
}

func TestCopilotEngineGitHubToolsShellEscaping(t *testing.T) {
	engine := NewCopilotEngine()
	workflowData := &WorkflowData{
		Name: "test-workflow",
		Tools: map[string]any{
			"github": map[string]any{
				"allowed": []any{"add_issue_comment", "issue_read"},
			},
		},
	}
	steps := engine.GetExecutionSteps(workflowData, "/tmp/gh-aw/test.log")

	if len(steps) != 1 {
		t.Fatalf("Expected 1 step, got %d", len(steps))
	}

	// Get the full command from the execution step
	stepContent := strings.Join([]string(steps[0]), "\n")

	// Find the line that contains the copilot command
	// When firewall is disabled, it uses 'copilot' instead of 'npx'
	lines := strings.Split(stepContent, "\n")
	var copilotCommand string
	for _, line := range lines {
		if strings.Contains(line, "copilot") && strings.Contains(line, "--allow-tool") {
			copilotCommand = strings.TrimSpace(line)
			break
		}
	}

	if copilotCommand == "" {
		t.Fatalf("Could not find copilot command in step content:\n%s", stepContent)
	}

	// Verify that GitHub tool arguments are properly single-quoted
	t.Logf("Generated command: %s", copilotCommand)

	// The command should contain properly escaped GitHub tool arguments with single quotes
	if !strings.Contains(copilotCommand, "'github(add_issue_comment)'") {
		t.Errorf("Expected 'github(add_issue_comment)' to be single-quoted in command: %s", copilotCommand)
	}

	if !strings.Contains(copilotCommand, "'github(issue_read)'") {
		t.Errorf("Expected 'github(issue_read)' to be single-quoted in command: %s", copilotCommand)
	}
}

func TestCopilotEngineNoAskUser(t *testing.T) {
	engine := NewCopilotEngine()

	tests := []struct {
		name         string
		engineConfig *EngineConfig
		safeOutputs  *SafeOutputsConfig
		expectNoAsk  bool
		description  string
	}{
		{
			name:         "default version emits --no-ask-user for agent job",
			engineConfig: nil,
			safeOutputs:  &SafeOutputsConfig{},
			expectNoAsk:  true,
			description:  "default version is >= 1.0.19",
		},
		{
			name:         "latest version emits --no-ask-user for agent job",
			engineConfig: &EngineConfig{Version: "latest"},
			safeOutputs:  &SafeOutputsConfig{},
			expectNoAsk:  true,
			description:  "latest always supports --no-ask-user",
		},
		{
			name:         "version 1.0.19 emits --no-ask-user",
			engineConfig: &EngineConfig{Version: "1.0.19"},
			safeOutputs:  &SafeOutputsConfig{},
			expectNoAsk:  true,
			description:  "1.0.19 is the minimum supported version",
		},
		{
			name:         "version 1.0.20 emits --no-ask-user",
			engineConfig: &EngineConfig{Version: "1.0.20"},
			safeOutputs:  &SafeOutputsConfig{},
			expectNoAsk:  true,
			description:  "1.0.20 > 1.0.19",
		},
		{
			name:         "version 1.0.18 does not emit --no-ask-user",
			engineConfig: &EngineConfig{Version: "1.0.18"},
			safeOutputs:  &SafeOutputsConfig{},
			expectNoAsk:  false,
			description:  "1.0.18 < 1.0.19",
		},
		{
			name:         "version 1.0.0 does not emit --no-ask-user",
			engineConfig: &EngineConfig{Version: "1.0.0"},
			safeOutputs:  &SafeOutputsConfig{},
			expectNoAsk:  false,
			description:  "1.0.0 < 1.0.19",
		},
		{
			name:         "detection job emits --no-ask-user with default version",
			engineConfig: nil,
			safeOutputs:  nil, // nil SafeOutputs = detection job
			expectNoAsk:  true,
			description:  "--no-ask-user is emitted for both agent and detection jobs",
		},
		{
			name:         "detection job with old version does not emit --no-ask-user",
			engineConfig: &EngineConfig{Version: "1.0.18"},
			safeOutputs:  nil,
			expectNoAsk:  false,
			description:  "detection job with old version still respects version gate",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			workflowData := &WorkflowData{
				Name:         "test-workflow",
				EngineConfig: tt.engineConfig,
				SafeOutputs:  tt.safeOutputs,
			}

			steps := engine.GetExecutionSteps(workflowData, "/tmp/gh-aw/agent-stdio.log")
			if len(steps) == 0 {
				t.Fatal("Expected at least one step")
			}

			stepContent := strings.Join([]string(steps[0]), "\n")
			hasNoAsk := strings.Contains(stepContent, "--no-ask-user")

			if tt.expectNoAsk && !hasNoAsk {
				t.Errorf("%s: expected --no-ask-user in step, got:\n%s", tt.description, stepContent)
			}
			if !tt.expectNoAsk && hasNoAsk {
				t.Errorf("%s: expected --no-ask-user NOT in step, got:\n%s", tt.description, stepContent)
			}
		})
	}
}

func TestCopilotSupportsNoAskUser(t *testing.T) {
	defaultSupported := semverutil.Compare(
		string(constants.DefaultCopilotVersion),
		string(constants.CopilotNoAskUserMinVersion),
	) >= 0

	tests := []struct {
		name         string
		engineConfig *EngineConfig
		expected     bool
	}{
		{
			name:         "nil config uses default version gate",
			engineConfig: nil,
			expected:     defaultSupported,
		},
		{
			name:         "empty version uses default version gate",
			engineConfig: &EngineConfig{},
			expected:     defaultSupported,
		},
		{
			name:         "latest is always supported",
			engineConfig: &EngineConfig{Version: "latest"},
			expected:     true,
		},
		{
			name:         "LATEST (uppercase) is always supported",
			engineConfig: &EngineConfig{Version: "LATEST"},
			expected:     true,
		},
		{
			name:         "exact minimum version 1.0.19 is supported",
			engineConfig: &EngineConfig{Version: "1.0.19"},
			expected:     true,
		},
		{
			name:         "version with v-prefix v1.0.19 is supported",
			engineConfig: &EngineConfig{Version: "v1.0.19"},
			expected:     true,
		},
		{
			name:         "version above minimum 1.0.20 is supported",
			engineConfig: &EngineConfig{Version: "1.0.20"},
			expected:     true,
		},
		{
			name:         "version below minimum 1.0.18 is not supported",
			engineConfig: &EngineConfig{Version: "1.0.18"},
			expected:     false,
		},
		{
			name:         "version 1.0.0 is not supported",
			engineConfig: &EngineConfig{Version: "1.0.0"},
			expected:     false,
		},
		{
			name:         "non-semver branch name returns false (conservative)",
			engineConfig: &EngineConfig{Version: "main"},
			expected:     false,
		},
		{
			name:         "expression version is treated as supported",
			engineConfig: &EngineConfig{Version: "${{ inputs.engine-version }}"},
			expected:     true,
		},
		{
			name:         "github event input expression is treated as supported",
			engineConfig: &EngineConfig{Version: "${{ github.event.inputs.engine-version }}"},
			expected:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := copilotSupportsNoAskUser(tt.engineConfig)
			if result != tt.expected {
				t.Errorf("copilotSupportsNoAskUser(%v) = %v, want %v", tt.engineConfig, result, tt.expected)
			}
		})
	}
}

func TestSanitizeCopilotShellCommand(t *testing.T) {
	tests := []struct {
		name            string
		input           string
		expectedOutput  string
		expectedChanged bool
	}{
		{
			name:            "no single quotes - unchanged",
			input:           "jq . /tmp/file.json",
			expectedOutput:  "jq . /tmp/file.json",
			expectedChanged: false,
		},
		{
			name:            "single-quoted jq filter - truncated to prefix",
			input:           "jq '.data[] | {id, billing}' /tmp/file.json",
			expectedOutput:  "jq",
			expectedChanged: true,
		},
		{
			name:            "single-quoted jq array filter - truncated to prefix",
			input:           "jq '[.data[] | keys] | add | unique' /tmp/file.json",
			expectedOutput:  "jq",
			expectedChanged: true,
		},
		{
			name:            "plain command without quotes - unchanged",
			input:           "cat /tmp/file.json",
			expectedOutput:  "cat /tmp/file.json",
			expectedChanged: false,
		},
		{
			name:            "empty string - unchanged",
			input:           "",
			expectedOutput:  "",
			expectedChanged: false,
		},
		{
			name:            "single quote at start - empty prefix",
			input:           "'quoted from start'",
			expectedOutput:  "",
			expectedChanged: true,
		},
		{
			name:            "trailing whitespace trimmed after truncation",
			input:           "grep  '.pattern'",
			expectedOutput:  "grep",
			expectedChanged: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, changed := sanitizeCopilotShellCommand(tt.input)
			if output != tt.expectedOutput {
				t.Errorf("sanitizeCopilotShellCommand(%q) output = %q, want %q", tt.input, output, tt.expectedOutput)
			}
			if changed != tt.expectedChanged {
				t.Errorf("sanitizeCopilotShellCommand(%q) changed = %v, want %v", tt.input, changed, tt.expectedChanged)
			}
		})
	}
}
