package workflow

import (
	"strings"
	"testing"
)

func TestNetworkHookGenerator(t *testing.T) {
	generator := &NetworkHookGenerator{}

	t.Run("GenerateNetworkHookScript", func(t *testing.T) {
		allowedDomains := []string{"example.com", "*.trusted.com", "api.service.org"}
		script := generator.GenerateNetworkHookScript(allowedDomains)

		// Check that script contains the expected domains
		if !strings.Contains(script, `"example.com"`) {
			t.Error("Script should contain example.com")
		}
		if !strings.Contains(script, `"*.trusted.com"`) {
			t.Error("Script should contain *.trusted.com")
		}
		if !strings.Contains(script, `"api.service.org"`) {
			t.Error("Script should contain api.service.org")
		}

		// Check for required Python imports and functions
		if !strings.Contains(script, "import json") {
			t.Error("Script should import json")
		}
		if !strings.Contains(script, "import urllib.parse") {
			t.Error("Script should import urllib.parse")
		}
		if !strings.Contains(script, "def extract_domain") {
			t.Error("Script should define extract_domain function")
		}
		if !strings.Contains(script, "def is_domain_allowed") {
			t.Error("Script should define is_domain_allowed function")
		}

		// Check for WebFetch and WebSearch handling
		if !strings.Contains(script, "WebFetch") && !strings.Contains(script, "WebSearch") {
			t.Error("Script should handle WebFetch and WebSearch tools")
		}
	})

	t.Run("GenerateNetworkHookWorkflowStep", func(t *testing.T) {
		allowedDomains := []string{"example.com", "test.org"}
		step := generator.GenerateNetworkHookWorkflowStep(allowedDomains)

		// Check step structure
		if len(step) == 0 {
			t.Fatal("Step should not be empty")
		}

		stepStr := strings.Join(step, "\n")
		if !strings.Contains(stepStr, "Generate Network Permissions Hook") {
			t.Error("Step should have correct name")
		}
		if !strings.Contains(stepStr, "mkdir -p .claude/hooks") {
			t.Error("Step should create hooks directory")
		}
		if !strings.Contains(stepStr, ".claude/hooks/network_permissions.py") {
			t.Error("Step should create network permissions hook file")
		}
		if !strings.Contains(stepStr, "chmod +x") {
			t.Error("Step should make hook executable")
		}
	})

	t.Run("EmptyDomainsList", func(t *testing.T) {
		script := generator.GenerateNetworkHookScript([]string{})
		if !strings.Contains(script, "ALLOWED_DOMAINS = []") {
			t.Error("Empty domains list should result in empty ALLOWED_DOMAINS array")
		}
	})
}

func TestClaudeSettingsGenerator(t *testing.T) {
	generator := &ClaudeSettingsGenerator{}

	t.Run("GenerateSettingsJSON", func(t *testing.T) {
		settingsJSON := generator.GenerateSettingsJSON()

		// Check JSON structure
		if !strings.Contains(settingsJSON, `"hooks"`) {
			t.Error("Settings should contain hooks section")
		}
		if !strings.Contains(settingsJSON, `"PreToolUse"`) {
			t.Error("Settings should contain PreToolUse hooks")
		}
		if !strings.Contains(settingsJSON, `"WebFetch|WebSearch"`) {
			t.Error("Settings should match WebFetch and WebSearch tools")
		}
		if !strings.Contains(settingsJSON, `.claude/hooks/network_permissions.py`) {
			t.Error("Settings should reference network permissions hook")
		}
		if !strings.Contains(settingsJSON, `"type": "command"`) {
			t.Error("Settings should specify command hook type")
		}
	})

	t.Run("GenerateSettingsWorkflowStep", func(t *testing.T) {
		step := generator.GenerateSettingsWorkflowStep()

		// Check step structure
		if len(step) == 0 {
			t.Fatal("Step should not be empty")
		}

		stepStr := strings.Join(step, "\n")
		if !strings.Contains(stepStr, "Generate Claude Settings") {
			t.Error("Step should have correct name")
		}
		if !strings.Contains(stepStr, ".claude/settings.json") {
			t.Error("Step should create settings.json file")
		}
		if !strings.Contains(stepStr, "EOF") {
			t.Error("Step should use heredoc syntax")
		}
	})
}

func TestNetworkPermissionsHelpers(t *testing.T) {
	t.Run("HasNetworkPermissions", func(t *testing.T) {
		// Test nil config
		if HasNetworkPermissions(nil) {
			t.Error("nil config should not have network permissions")
		}

		// Test config without permissions
		config := &EngineConfig{ID: "claude"}
		if HasNetworkPermissions(config) {
			t.Error("Config without permissions should not have network permissions")
		}

		// Test config with empty permissions
		config.Permissions = &EnginePermissions{}
		if HasNetworkPermissions(config) {
			t.Error("Config with empty permissions should not have network permissions")
		}

		// Test config with empty network permissions (empty struct)
		config.Permissions.Network = &NetworkPermissions{}
		if !HasNetworkPermissions(config) {
			t.Error("Config with empty network permissions struct should have network permissions (deny-all policy)")
		}

		// Test config with network permissions
		config.Permissions.Network.Allowed = []string{"example.com"}
		if !HasNetworkPermissions(config) {
			t.Error("Config with network permissions should have network permissions")
		}

		// Test non-Claude engine with network permissions (should be false)
		nonClaudeConfig := &EngineConfig{
			ID: "codex",
			Permissions: &EnginePermissions{
				Network: &NetworkPermissions{
					Allowed: []string{"example.com"},
				},
			},
		}
		if HasNetworkPermissions(nonClaudeConfig) {
			t.Error("Non-Claude engine should not have network permissions even if configured")
		}
	})

	t.Run("GetAllowedDomains", func(t *testing.T) {
		// Test nil config
		domains := GetAllowedDomains(nil)
		if domains != nil {
			t.Error("nil config should return nil (no restrictions)")
		}

		// Test config without permissions
		config := &EngineConfig{ID: "claude"}
		domains = GetAllowedDomains(config)
		if domains != nil {
			t.Error("Config without permissions should return nil (no restrictions)")
		}

		// Test config with empty network permissions (deny-all policy)
		config.Permissions = &EnginePermissions{
			Network: &NetworkPermissions{
				Allowed: []string{}, // Empty list means deny-all
			},
		}
		domains = GetAllowedDomains(config)
		if domains == nil {
			t.Error("Config with empty network permissions should return empty slice (deny-all policy)")
		}
		if len(domains) != 0 {
			t.Errorf("Expected 0 domains for deny-all policy, got %d", len(domains))
		}

		// Test config with network permissions
		config.Permissions = &EnginePermissions{
			Network: &NetworkPermissions{
				Allowed: []string{"example.com", "*.trusted.com", "api.service.org"},
			},
		}
		domains = GetAllowedDomains(config)
		if len(domains) != 3 {
			t.Errorf("Expected 3 domains, got %d", len(domains))
		}
		if domains[0] != "example.com" {
			t.Errorf("Expected first domain to be 'example.com', got '%s'", domains[0])
		}
		if domains[1] != "*.trusted.com" {
			t.Errorf("Expected second domain to be '*.trusted.com', got '%s'", domains[1])
		}
		if domains[2] != "api.service.org" {
			t.Errorf("Expected third domain to be 'api.service.org', got '%s'", domains[2])
		}

		// Test non-Claude engine with network permissions (should return empty)
		nonClaudeConfig := &EngineConfig{
			ID: "codex",
			Permissions: &EnginePermissions{
				Network: &NetworkPermissions{
					Allowed: []string{"example.com", "test.org"},
				},
			},
		}
		domains = GetAllowedDomains(nonClaudeConfig)
		if len(domains) != 0 {
			t.Error("Non-Claude engine should return empty domains even if configured")
		}
	})
}

func TestEngineConfigParsing(t *testing.T) {
	compiler := &Compiler{}

	t.Run("ParseNetworkPermissions", func(t *testing.T) {
		frontmatter := map[string]any{
			"engine": map[string]any{
				"id":    "claude",
				"model": "claude-3-5-sonnet-20241022",
				"permissions": map[string]any{
					"network": map[string]any{
						"allowed": []any{"example.com", "*.trusted.com", "api.service.org"},
					},
				},
			},
		}

		engineSetting, engineConfig := compiler.extractEngineConfig(frontmatter)

		if engineSetting != "claude" {
			t.Errorf("Expected engine setting 'claude', got '%s'", engineSetting)
		}

		if engineConfig == nil {
			t.Fatal("Engine config should not be nil")
		}

		if engineConfig.ID != "claude" {
			t.Errorf("Expected engine ID 'claude', got '%s'", engineConfig.ID)
		}

		if engineConfig.Model != "claude-3-5-sonnet-20241022" {
			t.Errorf("Expected model 'claude-3-5-sonnet-20241022', got '%s'", engineConfig.Model)
		}

		if !HasNetworkPermissions(engineConfig) {
			t.Error("Engine config should have network permissions")
		}

		domains := GetAllowedDomains(engineConfig)
		expectedDomains := []string{"example.com", "*.trusted.com", "api.service.org"}
		if len(domains) != len(expectedDomains) {
			t.Fatalf("Expected %d domains, got %d", len(expectedDomains), len(domains))
		}

		for i, expected := range expectedDomains {
			if domains[i] != expected {
				t.Errorf("Expected domain %d to be '%s', got '%s'", i, expected, domains[i])
			}
		}
	})

	t.Run("ParseWithoutNetworkPermissions", func(t *testing.T) {
		frontmatter := map[string]any{
			"engine": map[string]any{
				"id":    "claude",
				"model": "claude-3-5-sonnet-20241022",
			},
		}

		engineSetting, engineConfig := compiler.extractEngineConfig(frontmatter)

		if engineSetting != "claude" {
			t.Errorf("Expected engine setting 'claude', got '%s'", engineSetting)
		}

		if engineConfig == nil {
			t.Fatal("Engine config should not be nil")
		}

		if HasNetworkPermissions(engineConfig) {
			t.Error("Engine config should not have network permissions")
		}

		domains := GetAllowedDomains(engineConfig)
		if len(domains) != 0 {
			t.Errorf("Expected 0 domains, got %d", len(domains))
		}
	})
}
