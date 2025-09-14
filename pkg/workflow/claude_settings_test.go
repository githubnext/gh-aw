package workflow

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestClaudeSettingsStructures(t *testing.T) {
	t.Run("Default permissions match TVS specification", func(t *testing.T) {
		generator := &ClaudeSettingsGenerator{}
		jsonStr := generator.GenerateSettingsJSON()

		var settings map[string]interface{}
		err := json.Unmarshal([]byte(jsonStr), &settings)
		if err != nil {
			t.Fatalf("Failed to unmarshal settings: %v", err)
		}

		permissions, exists := settings["permissions"]
		if !exists {
			t.Fatal("Settings should contain permissions section")
		}

		permissionsMap, ok := permissions.(map[string]interface{})
		if !ok {
			t.Fatal("Permissions should be an object")
		}

		// Verify allow permissions exactly match specification
		allow, exists := permissionsMap["allow"]
		if !exists {
			t.Fatal("Permissions should contain allow section")
		}

		allowArray, ok := allow.([]interface{})
		if !ok {
			t.Fatal("Allow should be an array")
		}

		expectedAllowPermissions := []string{
			"Bash(npm run lint)",
			"Bash(npm run test:*)",
			"Read(~/.zshrc)",
		}

		if len(allowArray) != len(expectedAllowPermissions) {
			t.Errorf("Expected %d allow permissions, got %d", len(expectedAllowPermissions), len(allowArray))
		}

		for i, expected := range expectedAllowPermissions {
			if i >= len(allowArray) {
				t.Errorf("Missing expected allow permission: %s", expected)
				continue
			}
			actual, ok := allowArray[i].(string)
			if !ok {
				t.Errorf("Allow permission at index %d should be string, got %T", i, allowArray[i])
				continue
			}
			if actual != expected {
				t.Errorf("Allow permission at index %d: expected '%s', got '%s'", i, expected, actual)
			}
		}

		// Verify deny permissions exactly match specification
		deny, exists := permissionsMap["deny"]
		if !exists {
			t.Fatal("Permissions should contain deny section")
		}

		denyArray, ok := deny.([]interface{})
		if !ok {
			t.Fatal("Deny should be an array")
		}

		expectedDenyPermissions := []string{
			"Bash(curl:*)",
			"Read(./.env)",
			"Read(./.env.*)",
			"Read(./secrets/**)",
		}

		if len(denyArray) != len(expectedDenyPermissions) {
			t.Errorf("Expected %d deny permissions, got %d", len(expectedDenyPermissions), len(denyArray))
		}

		for i, expected := range expectedDenyPermissions {
			if i >= len(denyArray) {
				t.Errorf("Missing expected deny permission: %s", expected)
				continue
			}
			actual, ok := denyArray[i].(string)
			if !ok {
				t.Errorf("Deny permission at index %d should be string, got %T", i, denyArray[i])
				continue
			}
			if actual != expected {
				t.Errorf("Deny permission at index %d: expected '%s', got '%s'", i, expected, actual)
			}
		}
	})

	t.Run("ClaudeSettings JSON marshaling with permissions", func(t *testing.T) {
		settings := ClaudeSettings{
			Permissions: &PermissionsConfiguration{
				Allow: []string{
					"Bash(npm run lint)",
					"Bash(npm run test:*)",
					"Read(~/.zshrc)",
				},
				Deny: []string{
					"Bash(curl:*)",
					"Read(./.env)",
					"Read(./.env.*)",
					"Read(./secrets/**)",
				},
			},
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

		// Test permissions section
		if !strings.Contains(jsonStr, `"permissions"`) {
			t.Error("JSON should contain permissions field")
		}
		if !strings.Contains(jsonStr, `"allow"`) {
			t.Error("JSON should contain allow field")
		}
		if !strings.Contains(jsonStr, `"deny"`) {
			t.Error("JSON should contain deny field")
		}
		if !strings.Contains(jsonStr, `"Bash(npm run lint)"`) {
			t.Error("JSON should contain npm lint permission")
		}
		if !strings.Contains(jsonStr, `"Read(./.env)"`) {
			t.Error("JSON should contain env file denial")
		}

		// Test existing hooks section
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
		if strings.Contains(jsonStr, `"permissions"`) {
			t.Error("Empty settings should not contain permissions field due to omitempty")
		}
		if strings.Contains(jsonStr, `"hooks"`) {
			t.Error("Empty settings should not contain hooks field due to omitempty")
		}
	})

	t.Run("Default permissions structure validation", func(t *testing.T) {
		generator := &ClaudeSettingsGenerator{}
		jsonStr := generator.GenerateSettingsJSON()

		var settings ClaudeSettings
		err := json.Unmarshal([]byte(jsonStr), &settings)
		if err != nil {
			t.Fatalf("Failed to unmarshal generated settings: %v", err)
		}

		// Verify permissions structure is present
		if settings.Permissions == nil {
			t.Error("Generated settings should have permissions")
		}

		// Verify allow permissions contain expected values
		expectedAllowItems := []string{
			"Bash(npm run lint)",
			"Bash(npm run test:*)",
			"Read(~/.zshrc)",
		}
		for _, expected := range expectedAllowItems {
			found := false
			for _, item := range settings.Permissions.Allow {
				if item == expected {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Expected allow permission '%s' not found", expected)
			}
		}

		// Verify deny permissions contain expected values
		expectedDenyItems := []string{
			"Bash(curl:*)",
			"Read(./.env)",
			"Read(./.env.*)",
			"Read(./secrets/**)",
		}
		for _, expected := range expectedDenyItems {
			found := false
			for _, item := range settings.Permissions.Deny {
				if item == expected {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Expected deny permission '%s' not found", expected)
			}
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

		// Verify permissions structure is preserved
		if settings.Permissions == nil {
			t.Error("Unmarshaled settings should have permissions")
		}
		if len(settings.Permissions.Allow) != 3 {
			t.Errorf("Expected 3 allow permissions, got %d", len(settings.Permissions.Allow))
		}
		if len(settings.Permissions.Deny) != 4 {
			t.Errorf("Expected 4 deny permissions, got %d", len(settings.Permissions.Deny))
		}

		// Verify hooks structure is preserved
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

		// Verify the JSON content is embedded and contains permissions
		if !strings.Contains(stepStr, `"permissions"`) {
			t.Error("Step should contain embedded permissions in JSON settings")
		}
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

		// Check permissions structure
		permissions, exists := settings["permissions"]
		if !exists {
			t.Error("Settings should contain permissions section")
		}

		permissionsMap, ok := permissions.(map[string]interface{})
		if !ok {
			t.Error("Permissions should be an object")
		}

		allow, exists := permissionsMap["allow"]
		if !exists {
			t.Error("Permissions should contain allow section")
		}

		allowArray, ok := allow.([]interface{})
		if !ok {
			t.Error("Allow should be an array")
		}

		if len(allowArray) != 3 {
			t.Errorf("Allow should contain 3 permissions, got %d", len(allowArray))
		}

		deny, exists := permissionsMap["deny"]
		if !exists {
			t.Error("Permissions should contain deny section")
		}

		denyArray, ok := deny.([]interface{})
		if !ok {
			t.Error("Deny should be an array")
		}

		if len(denyArray) != 4 {
			t.Errorf("Deny should contain 4 permissions, got %d", len(denyArray))
		}

		// Check hooks structure
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
