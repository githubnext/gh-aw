package workflow

import (
	"testing"
)

// TestExtractEngineConfigSessionBoolean tests extracting session configuration in boolean format
func TestExtractEngineConfigSessionBoolean(t *testing.T) {
	compiler := &Compiler{}

	frontmatter := map[string]any{
		"engine": map[string]any{
			"id":      "claude",
			"session": true,
		},
	}

	engineID, config := compiler.ExtractEngineConfig(frontmatter)

	if engineID != "claude" {
		t.Errorf("Expected engine ID 'claude', got '%s'", engineID)
	}

	if config == nil {
		t.Fatal("Expected config to be non-nil")
	}

	if config.Session == nil {
		t.Fatal("Expected session config to be non-nil")
	}

	if !config.Session.Enabled {
		t.Error("Expected session.enabled to be true")
	}

	if config.Session.Resume {
		t.Error("Expected session.resume to be false by default")
	}

	if config.Session.Continue {
		t.Error("Expected session.continue to be false by default")
	}
}

// TestExtractEngineConfigSessionObject tests extracting session configuration in object format
func TestExtractEngineConfigSessionObject(t *testing.T) {
	compiler := &Compiler{}

	frontmatter := map[string]any{
		"engine": map[string]any{
			"id": "claude",
			"session": map[string]any{
				"enabled":  true,
				"continue": true,
			},
		},
	}

	engineID, config := compiler.ExtractEngineConfig(frontmatter)

	if engineID != "claude" {
		t.Errorf("Expected engine ID 'claude', got '%s'", engineID)
	}

	if config == nil {
		t.Fatal("Expected config to be non-nil")
	}

	if config.Session == nil {
		t.Fatal("Expected session config to be non-nil")
	}

	if !config.Session.Enabled {
		t.Error("Expected session.enabled to be true")
	}

	if config.Session.Resume {
		t.Error("Expected session.resume to be false")
	}

	if !config.Session.Continue {
		t.Error("Expected session.continue to be true")
	}
}

// TestExtractEngineConfigSessionWithID tests extracting session configuration with session ID
func TestExtractEngineConfigSessionWithID(t *testing.T) {
	compiler := &Compiler{}

	frontmatter := map[string]any{
		"engine": map[string]any{
			"id": "claude",
			"session": map[string]any{
				"enabled": true,
				"resume":  true,
				"id":      "test-session-123",
			},
		},
	}

	engineID, config := compiler.ExtractEngineConfig(frontmatter)

	if engineID != "claude" {
		t.Errorf("Expected engine ID 'claude', got '%s'", engineID)
	}

	if config == nil {
		t.Fatal("Expected config to be non-nil")
	}

	if config.Session == nil {
		t.Fatal("Expected session config to be non-nil")
	}

	if !config.Session.Enabled {
		t.Error("Expected session.enabled to be true")
	}

	if !config.Session.Resume {
		t.Error("Expected session.resume to be true")
	}

	if config.Session.Continue {
		t.Error("Expected session.continue to be false")
	}

	if config.Session.ID != "test-session-123" {
		t.Errorf("Expected session.id to be 'test-session-123', got '%s'", config.Session.ID)
	}
}

// TestExtractEngineConfigNoSession tests extracting engine config without session
func TestExtractEngineConfigNoSession(t *testing.T) {
	compiler := &Compiler{}

	frontmatter := map[string]any{
		"engine": map[string]any{
			"id": "claude",
		},
	}

	engineID, config := compiler.ExtractEngineConfig(frontmatter)

	if engineID != "claude" {
		t.Errorf("Expected engine ID 'claude', got '%s'", engineID)
	}

	if config == nil {
		t.Fatal("Expected config to be non-nil")
	}

	if config.Session != nil {
		t.Error("Expected session config to be nil when not specified")
	}
}

// TestExtractEngineConfigSessionDisabled tests extracting session configuration when disabled
func TestExtractEngineConfigSessionDisabled(t *testing.T) {
	compiler := &Compiler{}

	frontmatter := map[string]any{
		"engine": map[string]any{
			"id":      "claude",
			"session": false,
		},
	}

	engineID, config := compiler.ExtractEngineConfig(frontmatter)

	if engineID != "claude" {
		t.Errorf("Expected engine ID 'claude', got '%s'", engineID)
	}

	if config == nil {
		t.Fatal("Expected config to be non-nil")
	}

	if config.Session == nil {
		t.Fatal("Expected session config to be non-nil")
	}

	if config.Session.Enabled {
		t.Error("Expected session.enabled to be false")
	}
}
