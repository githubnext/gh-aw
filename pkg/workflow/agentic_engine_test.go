package workflow

import (
	"testing"
)

func TestEngineRegistry(t *testing.T) {
	registry := NewEngineRegistry()

	// Test that built-in engines are registered
	supportedEngines := registry.GetSupportedEngines()
	if len(supportedEngines) != 5 {
		t.Errorf("Expected 5 supported engines, got %d", len(supportedEngines))
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

	opencodeEngine, err := registry.GetEngine("opencode")
	if err != nil {
		t.Errorf("Expected to find opencode engine, got error: %v", err)
	}
	if opencodeEngine.GetID() != "opencode" {
		t.Errorf("Expected opencode engine ID, got '%s'", opencodeEngine.GetID())
	}

	geminiEngine, err := registry.GetEngine("gemini")
	if err != nil {
		t.Errorf("Expected to find gemini engine, got error: %v", err)
	}
	if geminiEngine.GetID() != "gemini" {
		t.Errorf("Expected gemini engine ID, got '%s'", geminiEngine.GetID())
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

	if !registry.IsValidEngine("opencode") {
		t.Error("Expected opencode to be valid engine")
	}

	if !registry.IsValidEngine("gemini") {
		t.Error("Expected gemini to be valid engine")
	}

	if registry.IsValidEngine("nonexistent") {
		t.Error("Expected nonexistent to be invalid engine")
	}

	// Test GetDefaultEngine
	defaultEngine := registry.GetDefaultEngine()
	if defaultEngine.GetID() != "claude" {
		t.Errorf("Expected default engine to be claude, got '%s'", defaultEngine.GetID())
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

func TestEngineRegistryCustomEngine(t *testing.T) {
	registry := NewEngineRegistry()

	// Create a custom engine for testing
	customEngine := &ClaudeEngine{
		BaseEngine: BaseEngine{
			id:                     "custom",
			displayName:            "Custom Engine",
			description:            "A custom test engine",
			experimental:           true,
			supportsToolsWhitelist: false,
		},
	}

	// Register the custom engine
	registry.Register(customEngine)

	// Test that it's now available
	engine, err := registry.GetEngine("custom")
	if err != nil {
		t.Errorf("Expected to find custom engine, got error: %v", err)
	}

	if engine.GetID() != "custom" {
		t.Errorf("Expected custom engine ID, got '%s'", engine.GetID())
	}

	if !engine.IsExperimental() {
		t.Error("Expected custom engine to be experimental")
	}

	// Test that supported engines list is updated
	supportedEngines := registry.GetSupportedEngines()
	if len(supportedEngines) != 6 {
		t.Errorf("Expected 6 supported engines after adding custom, got %d", len(supportedEngines))
	}
}
