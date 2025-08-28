package workflow

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestClaudeSettingsStructures(t *testing.T) {
	t.Run("ClaudeSettings JSON marshaling", func(t *testing.T) {
		settings := ClaudeSettings{
			Hooks: &HookConfiguration{
				PreToolUse: []PreToolUseHook{
					{
						Matcher: "WebFetch|WebSearch",
						Hooks: []HookEntry{
							{
								Type:    "command",
								Command: ".claude/hooks/network_permissions.py",
							},
						},
					},
				},
			},
		}

		jsonData, err := json.Marshal(settings)
		if err != nil {
			t.Fatalf("Failed to marshal settings: %v", err)
		}

		jsonStr := string(jsonData)
		if !strings.Contains(jsonStr, `"hooks"`) {
			t.Error("JSON should contain hooks field")
		}
		if !strings.Contains(jsonStr, `"PreToolUse"`) {
			t.Error("JSON should contain PreToolUse field")
		}
		if !strings.Contains(jsonStr, `"WebFetch|WebSearch"`) {
			t.Error("JSON should contain matcher pattern")
		}
		if !strings.Contains(jsonStr, `"command"`) {
			t.Error("JSON should contain hook type")
		}
		if !strings.Contains(jsonStr, `.claude/hooks/network_permissions.py`) {
			t.Error("JSON should contain hook command path")
		}
	})

	t.Run("Empty settings", func(t *testing.T) {
		settings := ClaudeSettings{}
		jsonData, err := json.Marshal(settings)
		if err != nil {
			t.Fatalf("Failed to marshal empty settings: %v", err)
		}

		jsonStr := string(jsonData)
		if strings.Contains(jsonStr, `"hooks"`) {
			t.Error("Empty settings should not contain hooks field due to omitempty")
		}
	})

	t.Run("JSON unmarshal round-trip", func(t *testing.T) {
		generator := &ClaudeSettingsGenerator{}
		originalJSON := generator.GenerateSettingsJSON()

		var settings ClaudeSettings
		err := json.Unmarshal([]byte(originalJSON), &settings)
		if err != nil {
			t.Fatalf("Failed to unmarshal settings: %v", err)
		}

		// Verify structure is preserved
		if settings.Hooks == nil {
			t.Error("Unmarshaled settings should have hooks")
		}
		if len(settings.Hooks.PreToolUse) != 1 {
			t.Errorf("Expected 1 PreToolUse hook, got %d", len(settings.Hooks.PreToolUse))
		}

		hook := settings.Hooks.PreToolUse[0]
		if hook.Matcher != "WebFetch|WebSearch" {
			t.Errorf("Expected matcher 'WebFetch|WebSearch', got '%s'", hook.Matcher)
		}
		if len(hook.Hooks) != 1 {
			t.Errorf("Expected 1 hook entry, got %d", len(hook.Hooks))
		}

		entry := hook.Hooks[0]
		if entry.Type != "command" {
			t.Errorf("Expected hook type 'command', got '%s'", entry.Type)
		}
		if entry.Command != ".claude/hooks/network_permissions.py" {
			t.Errorf("Expected command '.claude/hooks/network_permissions.py', got '%s'", entry.Command)
		}
	})
}

func TestClaudeSettingsWorkflowGeneration(t *testing.T) {
	generator := &ClaudeSettingsGenerator{}

	t.Run("Workflow step format", func(t *testing.T) {
		step := generator.GenerateSettingsWorkflowStep()

		if len(step) == 0 {
			t.Fatal("Generated step should not be empty")
		}

		stepStr := strings.Join(step, "\n")

		// Check step name
		if !strings.Contains(stepStr, "- name: Generate Claude Settings") {
			t.Error("Step should have correct name")
		}

		// Check run command structure
		if !strings.Contains(stepStr, "run: |") {
			t.Error("Step should use multi-line run format")
		}

		// Check file creation
		if !strings.Contains(stepStr, "cat > .claude/settings.json") {
			t.Error("Step should create .claude/settings.json file")
		}

		// Check heredoc usage
		if !strings.Contains(stepStr, "EOF") {
			t.Error("Step should use heredoc for JSON content")
		}

		// Check indentation
		lines := strings.Split(stepStr, "\n")
		foundRunLine := false
		for _, line := range lines {
			if strings.Contains(line, "run: |") {
				foundRunLine = true
				continue
			}
			if foundRunLine && strings.TrimSpace(line) != "" {
				if !strings.HasPrefix(line, "          ") {
					t.Errorf("Run command lines should be indented with 10 spaces, got line: '%s'", line)
				}
				break // Only check the first non-empty line after run:
			}
		}

		// Verify the JSON content is embedded
		if !strings.Contains(stepStr, `"hooks"`) {
			t.Error("Step should contain embedded JSON settings")
		}
	})

	t.Run("Generated JSON validity", func(t *testing.T) {
		jsonStr := generator.GenerateSettingsJSON()

		var settings map[string]interface{}
		err := json.Unmarshal([]byte(jsonStr), &settings)
		if err != nil {
			t.Fatalf("Generated JSON should be valid: %v", err)
		}

		// Check structure
		hooks, exists := settings["hooks"]
		if !exists {
			t.Error("Settings should contain hooks section")
		}

		hooksMap, ok := hooks.(map[string]interface{})
		if !ok {
			t.Error("Hooks should be an object")
		}

		preToolUse, exists := hooksMap["PreToolUse"]
		if !exists {
			t.Error("Hooks should contain PreToolUse section")
		}

		preToolUseArray, ok := preToolUse.([]interface{})
		if !ok {
			t.Error("PreToolUse should be an array")
		}

		if len(preToolUseArray) != 1 {
			t.Errorf("PreToolUse should contain 1 hook, got %d", len(preToolUseArray))
		}
	})
}
