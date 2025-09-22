package workflow

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestClaudeSettingsStructures(t *testing.T) {
	t.Run("ClaudeSettings JSON marshaling", func(t *testing.T) {
		// Test the basic structure without hooks (JavaScript validation replaces Python hooks)
		settings := ClaudeSettings{}

		jsonData, err := json.Marshal(settings)
		if err != nil {
			t.Fatalf("Failed to marshal settings: %v", err)
		}

		jsonStr := string(jsonData)
		if jsonStr == "" {
			t.Error("JSON should not be empty")
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

		// With JavaScript-based validation, hooks are no longer needed
		// Verify that settings can be marshaled/unmarshaled correctly
		remarshaled, err := json.MarshalIndent(settings, "", "  ")
		if err != nil {
			t.Fatalf("Failed to remarshal settings: %v", err)
		}

		if string(remarshaled) != originalJSON {
			t.Error("JSON round-trip failed")
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

		// Check directory creation command
		if !strings.Contains(stepStr, "mkdir -p /tmp/.claude") {
			t.Error("Step should create /tmp/.claude directory before creating settings file")
		}

		// Check file creation
		if !strings.Contains(stepStr, "cat > /tmp/.claude/settings.json") {
			t.Error("Step should create /tmp/.claude/settings.json file")
		}

		// Verify the order - mkdir should come before cat
		mkdirIndex := strings.Index(stepStr, "mkdir -p /tmp/.claude")
		catIndex := strings.Index(stepStr, "cat > /tmp/.claude/settings.json")
		if mkdirIndex == -1 || catIndex == -1 || mkdirIndex > catIndex {
			t.Error("Directory creation (mkdir) should come before file creation (cat)")
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

		// Verify the JSON content is embedded (even if empty, it should be valid JSON)
		if !strings.Contains(stepStr, "cat > /tmp/.claude/settings.json") {
			t.Error("Step should create settings.json file")
		}
	})

	t.Run("Generated JSON validity", func(t *testing.T) {
		jsonStr := generator.GenerateSettingsJSON()

		var settings map[string]any
		err := json.Unmarshal([]byte(jsonStr), &settings)
		if err != nil {
			t.Fatalf("Generated JSON should be valid: %v", err)
		}

		// With JavaScript-based validation, we expect empty or minimal settings
		// The key requirement is that the JSON is valid
		if len(jsonStr) == 0 {
			t.Error("Generated JSON should not be empty")
		}
	})
}
