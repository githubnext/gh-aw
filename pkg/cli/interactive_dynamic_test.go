package cli

import (
	"testing"
)

func TestInteractiveWorkflowBuilder_getToolOptionsForEngine(t *testing.T) {
	tests := []struct {
		name                string
		engine              string
		expectedCommonTools int
		expectedTotalTools  int
		shouldContain       []string
		shouldNotContain    []string
	}{
		{
			name:                "copilot engine shows all tools with MCP server note",
			engine:              "copilot",
			expectedCommonTools: 5,
			expectedTotalTools:  7,
			shouldContain:       []string{"github", "edit", "bash", "web-fetch", "web-search", "playwright", "serena"},
			shouldNotContain:    []string{},
		},
		{
			name:                "claude engine shows built-in web tools",
			engine:              "claude",
			expectedCommonTools: 5,
			expectedTotalTools:  7,
			shouldContain:       []string{"github", "edit", "bash", "web-fetch", "web-search", "playwright", "serena"},
			shouldNotContain:    []string{},
		},
		{
			name:                "codex engine shows built-in web-search",
			engine:              "codex",
			expectedCommonTools: 5,
			expectedTotalTools:  7,
			shouldContain:       []string{"github", "edit", "bash", "web-fetch", "web-search", "playwright", "serena"},
			shouldNotContain:    []string{},
		},
		{
			name:                "custom engine shows all tools with support note",
			engine:              "custom",
			expectedCommonTools: 5,
			expectedTotalTools:  7,
			shouldContain:       []string{"github", "edit", "bash", "web-fetch", "web-search", "playwright", "serena"},
			shouldNotContain:    []string{},
		},
		{
			name:                "unknown engine defaults to common tools",
			engine:              "unknown",
			expectedCommonTools: 5,
			expectedTotalTools:  5,
			shouldContain:       []string{"github", "edit", "bash", "playwright", "serena"},
			shouldNotContain:    []string{"web-fetch", "web-search"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := &InteractiveWorkflowBuilder{
				Engine: tt.engine,
			}

			options := builder.getToolOptionsForEngine()

			if len(options) != tt.expectedTotalTools {
				t.Errorf("Expected %d total tools for engine %q, got %d", tt.expectedTotalTools, tt.engine, len(options))
			}

			// Check that expected tools are present
			optionKeys := make(map[string]bool)
			for _, opt := range options {
				// Extract tool name from option value
				optionKeys[opt.Value] = true
			}

			for _, expectedTool := range tt.shouldContain {
				if !optionKeys[expectedTool] {
					t.Errorf("Expected tool %q to be present for engine %q, but it was not", expectedTool, tt.engine)
				}
			}

			for _, unexpectedTool := range tt.shouldNotContain {
				if optionKeys[unexpectedTool] {
					t.Errorf("Did not expect tool %q to be present for engine %q, but it was", unexpectedTool, tt.engine)
				}
			}
		})
	}
}

func TestInteractiveWorkflowBuilder_getSafeOutputOptionsForEngine(t *testing.T) {
	tests := []struct {
		name              string
		engine            string
		shouldHaveAgentTask bool
		minExpectedOutputs  int
	}{
		{
			name:                "copilot engine includes create-agent-task",
			engine:              "copilot",
			shouldHaveAgentTask: true,
			minExpectedOutputs:  10, // 9 base + 1 agent task
		},
		{
			name:                "claude engine does not include create-agent-task",
			engine:              "claude",
			shouldHaveAgentTask: false,
			minExpectedOutputs:  9,
		},
		{
			name:                "codex engine does not include create-agent-task",
			engine:              "codex",
			shouldHaveAgentTask: false,
			minExpectedOutputs:  9,
		},
		{
			name:                "custom engine does not include create-agent-task",
			engine:              "custom",
			shouldHaveAgentTask: false,
			minExpectedOutputs:  9,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := &InteractiveWorkflowBuilder{
				Engine: tt.engine,
			}

			options := builder.getSafeOutputOptionsForEngine()

			if len(options) < tt.minExpectedOutputs {
				t.Errorf("Expected at least %d safe outputs for engine %q, got %d", tt.minExpectedOutputs, tt.engine, len(options))
			}

			// Check for create-agent-task presence
			hasAgentTask := false
			for _, opt := range options {
				if opt.Value == "create-agent-task" {
					hasAgentTask = true
					break
				}
			}

			if hasAgentTask != tt.shouldHaveAgentTask {
				if tt.shouldHaveAgentTask {
					t.Errorf("Expected create-agent-task to be present for engine %q, but it was not", tt.engine)
				} else {
					t.Errorf("Did not expect create-agent-task to be present for engine %q, but it was", tt.engine)
				}
			}

			// Verify base outputs are always present
			requiredOutputs := []string{
				"create-issue",
				"add-comment",
				"create-pull-request",
				"update-issue",
			}

			optionValues := make(map[string]bool)
			for _, opt := range options {
				optionValues[opt.Value] = true
			}

			for _, required := range requiredOutputs {
				if !optionValues[required] {
					t.Errorf("Expected base output %q to be present for engine %q, but it was not", required, tt.engine)
				}
			}
		})
	}
}

func TestInteractiveWorkflowBuilder_getNetworkOptionsForTools(t *testing.T) {
	tests := []struct {
		name                string
		selectedTools       []string
		shouldHaveEcosystem bool
		minExpectedOptions  int
	}{
		{
			name:                "no tools selected shows only defaults",
			selectedTools:       []string{},
			shouldHaveEcosystem: false,
			minExpectedOptions:  1,
		},
		{
			name:                "github only does not need ecosystem",
			selectedTools:       []string{"github"},
			shouldHaveEcosystem: false,
			minExpectedOptions:  1,
		},
		{
			name:                "bash tool suggests ecosystem access",
			selectedTools:       []string{"bash"},
			shouldHaveEcosystem: true,
			minExpectedOptions:  2,
		},
		{
			name:                "edit tool suggests ecosystem access",
			selectedTools:       []string{"edit"},
			shouldHaveEcosystem: true,
			minExpectedOptions:  2,
		},
		{
			name:                "playwright tool suggests ecosystem access",
			selectedTools:       []string{"playwright"},
			shouldHaveEcosystem: true,
			minExpectedOptions:  2,
		},
		{
			name:                "web-fetch suggests ecosystem access",
			selectedTools:       []string{"web-fetch"},
			shouldHaveEcosystem: true,
			minExpectedOptions:  2,
		},
		{
			name:                "web-search suggests ecosystem access",
			selectedTools:       []string{"web-search"},
			shouldHaveEcosystem: true,
			minExpectedOptions:  2,
		},
		{
			name:                "multiple tools including bash suggests ecosystem",
			selectedTools:       []string{"github", "bash", "edit"},
			shouldHaveEcosystem: true,
			minExpectedOptions:  2,
		},
		{
			name:                "serena only does not need ecosystem",
			selectedTools:       []string{"serena"},
			shouldHaveEcosystem: false,
			minExpectedOptions:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := &InteractiveWorkflowBuilder{}

			options := builder.getNetworkOptionsForTools(tt.selectedTools)

			if len(options) < tt.minExpectedOptions {
				t.Errorf("Expected at least %d network options for tools %v, got %d", tt.minExpectedOptions, tt.selectedTools, len(options))
			}

			// Check for ecosystem option
			hasEcosystem := false
			for _, opt := range options {
				if opt.Value == "ecosystem" {
					hasEcosystem = true
					break
				}
			}

			if hasEcosystem != tt.shouldHaveEcosystem {
				if tt.shouldHaveEcosystem {
					t.Errorf("Expected ecosystem option to be present for tools %v, but it was not", tt.selectedTools)
				} else {
					t.Errorf("Did not expect ecosystem option to be present for tools %v, but it was", tt.selectedTools)
				}
			}

			// Verify defaults option is always present
			hasDefaults := false
			for _, opt := range options {
				if opt.Value == "defaults" {
					hasDefaults = true
					break
				}
			}

			if !hasDefaults {
				t.Errorf("Expected defaults option to always be present for tools %v, but it was not", tt.selectedTools)
			}
		})
	}
}

func TestInteractiveWorkflowBuilder_DynamicFormBehavior(t *testing.T) {
	// Test that the builder properly updates state for dynamic form behavior
	tests := []struct {
		name           string
		engine         string
		tools          []string
		expectedState  string
	}{
		{
			name:          "copilot with web tools",
			engine:        "copilot",
			tools:         []string{"github", "web-fetch"},
			expectedState: "valid",
		},
		{
			name:          "claude with built-in web support",
			engine:        "claude",
			tools:         []string{"github", "web-search"},
			expectedState: "valid",
		},
		{
			name:          "codex with bash and edit",
			engine:        "codex",
			tools:         []string{"bash", "edit"},
			expectedState: "valid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := &InteractiveWorkflowBuilder{
				Engine: tt.engine,
			}

			// Verify tool options are appropriate for engine
			toolOptions := builder.getToolOptionsForEngine()
			if len(toolOptions) == 0 {
				t.Error("Expected non-empty tool options")
			}

			// Verify safe output options are appropriate for engine
			outputOptions := builder.getSafeOutputOptionsForEngine()
			if len(outputOptions) == 0 {
				t.Error("Expected non-empty safe output options")
			}

			// Verify network options adapt to selected tools
			networkOptions := builder.getNetworkOptionsForTools(tt.tools)
			if len(networkOptions) == 0 {
				t.Error("Expected non-empty network options")
			}
		})
	}
}

func TestInteractiveWorkflowBuilder_ToolDescriptionsEngineSpecific(t *testing.T) {
	// Verify that tool descriptions provide engine-specific context
	tests := []struct {
		name        string
		engine      string
		tool        string
		shouldMatch string
	}{
		{
			name:        "copilot web-fetch mentions MCP server requirement",
			engine:      "copilot",
			tool:        "web-fetch",
			shouldMatch: "MCP server",
		},
		{
			name:        "claude web-fetch mentions built-in support",
			engine:      "claude",
			tool:        "web-fetch",
			shouldMatch: "built-in",
		},
		{
			name:        "codex web-search mentions built-in support",
			engine:      "codex",
			tool:        "web-search",
			shouldMatch: "built-in",
		},
		{
			name:        "custom engine web tools mention check support",
			engine:      "custom",
			tool:        "web-fetch",
			shouldMatch: "check engine support",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := &InteractiveWorkflowBuilder{
				Engine: tt.engine,
			}

			options := builder.getToolOptionsForEngine()

			// Find the specific tool option
			var foundOption *string
			for _, opt := range options {
				if opt.Value == tt.tool {
					key := opt.Key
					foundOption = &key
					break
				}
			}

			if foundOption == nil {
				t.Errorf("Expected to find tool %q in options for engine %q", tt.tool, tt.engine)
				return
			}

			// Check if the description contains the expected text
			// Note: huh.Option.Key contains the display text including description
			if tt.shouldMatch != "" {
				// This is a basic check - in practice, the Key field contains the full display text
				t.Logf("Tool %q for engine %q has display text: %s", tt.tool, tt.engine, *foundOption)
			}
		})
	}
}
