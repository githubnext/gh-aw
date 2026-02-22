//go:build !integration

package workflow

import (
	"slices"
	"testing"
)

func TestEngineRegistry(t *testing.T) {
	registry := NewEngineRegistry()

	// Test that built-in engines are registered - check for specific IDs rather than exact count
	// to avoid brittleness when new engines are added
	supportedEngines := registry.GetSupportedEngines()
	expectedEngineIDs := []string{"claude", "codex", "copilot", "gemini"}
	for _, engineID := range expectedEngineIDs {
		found := slices.Contains(supportedEngines, engineID)
		if !found {
			t.Errorf("Expected engine '%s' to be registered", engineID)
		}
	}

	// Test getting engines by ID
	claudeEngine, err := registry.GetEngine("claude")
	if err != nil {
		t.Errorf("Expected to find claude engine, got error: %v", err)
	}
	if claudeEngine.GetID() != "claude" {
		t.Errorf("Expected claude engine ID, got '%s'", claudeEngine.GetID())
	}

	codexEngine, err := registry.GetEngine("codex")
	if err != nil {
		t.Errorf("Expected to find codex engine, got error: %v", err)
	}
	if codexEngine.GetID() != "codex" {
		t.Errorf("Expected codex engine ID, got '%s'", codexEngine.GetID())
	}

	// Test getting non-existent engine
	_, err = registry.GetEngine("nonexistent")
	if err == nil {
		t.Error("Expected error when getting non-existent engine")
	}

	// Test IsValidEngine
	if !registry.IsValidEngine("claude") {
		t.Error("Expected claude to be valid engine")
	}

	if !registry.IsValidEngine("codex") {
		t.Error("Expected codex to be valid engine")
	}

	if registry.IsValidEngine("nonexistent") {
		t.Error("Expected nonexistent to be invalid engine")
	}

	// Test GetDefaultEngine
	defaultEngine := registry.GetDefaultEngine()
	if defaultEngine.GetID() != "copilot" {
		t.Errorf("Expected default engine to be copilot, got '%s'", defaultEngine.GetID())
	}

	// Test GetEngineByPrefix
	engineByPrefix, err := registry.GetEngineByPrefix("codex-experimental")
	if err != nil {
		t.Errorf("Expected to find engine by prefix 'codex-experimental', got error: %v", err)
	}
	if engineByPrefix.GetID() != "codex" {
		t.Errorf("Expected engine ID 'codex' from prefix, got '%s'", engineByPrefix.GetID())
	}

	// Test GetEngineByPrefix with non-matching prefix
	_, err = registry.GetEngineByPrefix("nonexistent-prefix")
	if err == nil {
		t.Error("Expected error when getting engine by non-matching prefix")
	}
}
