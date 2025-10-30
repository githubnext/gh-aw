package workflow

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestActionPinManagerLoadBuiltinPins tests loading builtin pins
func TestActionPinManagerLoadBuiltinPins(t *testing.T) {
	tmpDir := t.TempDir()
	manager := NewActionPinManager(tmpDir)

	err := manager.LoadBuiltinPins()
	if err != nil {
		t.Fatalf("Failed to load builtin pins: %v", err)
	}

	// Verify that builtin pins were loaded
	if len(manager.builtinPins) == 0 {
		t.Error("Expected builtin pins to be loaded, but got empty map")
	}

	// Check for a specific builtin pin
	if _, exists := manager.builtinPins["actions/checkout"]; !exists {
		t.Error("Expected actions/checkout in builtin pins")
	}
}

// TestActionPinManagerLoadCustomPins tests loading custom pins
func TestActionPinManagerLoadCustomPins(t *testing.T) {
	tmpDir := t.TempDir()
	manager := NewActionPinManager(tmpDir)

	// Create custom pins file
	customPinsPath := filepath.Join(tmpDir, ".github", "aw", CustomActionPinsFileName)
	if err := os.MkdirAll(filepath.Dir(customPinsPath), 0755); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}

	customPins := ActionPinsData{
		Version:     "1.0",
		Description: "Test custom pins",
		Actions: map[string]ActionPin{
			"custom/action": {
				Repo:    "custom/action",
				Version: "v1",
				SHA:     "1234567890123456789012345678901234567890",
			},
		},
	}

	data, err := json.MarshalIndent(customPins, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal custom pins: %v", err)
	}

	if err := os.WriteFile(customPinsPath, data, 0644); err != nil {
		t.Fatalf("Failed to write custom pins file: %v", err)
	}

	// Load custom pins
	err = manager.LoadCustomPins()
	if err != nil {
		t.Fatalf("Failed to load custom pins: %v", err)
	}

	// Verify custom pins were loaded
	if len(manager.customPins) != 1 {
		t.Errorf("Expected 1 custom pin, got %d", len(manager.customPins))
	}

	if pin, exists := manager.customPins["custom/action"]; !exists {
		t.Error("Expected custom/action in custom pins")
	} else if pin.SHA != "1234567890123456789012345678901234567890" {
		t.Errorf("Expected custom pin SHA to be 1234567890123456789012345678901234567890, got %s", pin.SHA)
	}
}

// TestActionPinManagerLoadCustomPinsNoFile tests loading when no custom pins file exists
func TestActionPinManagerLoadCustomPinsNoFile(t *testing.T) {
	tmpDir := t.TempDir()
	manager := NewActionPinManager(tmpDir)

	// Load custom pins (file doesn't exist)
	err := manager.LoadCustomPins()
	if err != nil {
		t.Fatalf("Expected no error when custom pins file doesn't exist, got: %v", err)
	}

	// Verify no custom pins were loaded
	if len(manager.customPins) != 0 {
		t.Errorf("Expected 0 custom pins, got %d", len(manager.customPins))
	}
}

// TestActionPinManagerMergePins tests merging builtin and custom pins
func TestActionPinManagerMergePins(t *testing.T) {
	tmpDir := t.TempDir()
	manager := NewActionPinManager(tmpDir)

	// Load builtin pins
	if err := manager.LoadBuiltinPins(); err != nil {
		t.Fatalf("Failed to load builtin pins: %v", err)
	}

	// Add a custom pin
	manager.customPins["custom/action"] = ActionPin{
		Repo:    "custom/action",
		Version: "v1",
		SHA:     "1234567890123456789012345678901234567890",
	}

	// Merge pins
	err := manager.MergePins()
	if err != nil {
		t.Fatalf("Failed to merge pins: %v", err)
	}

	// Verify merged pins contain both builtin and custom
	if len(manager.mergedPins) < len(manager.builtinPins) {
		t.Error("Merged pins should contain at least all builtin pins")
	}

	// Check that custom pin is in merged pins
	if _, exists := manager.mergedPins["custom/action"]; !exists {
		t.Error("Expected custom/action in merged pins")
	}

	// Check that a builtin pin is still in merged pins
	if _, exists := manager.mergedPins["actions/checkout"]; !exists {
		t.Error("Expected actions/checkout in merged pins")
	}
}

// TestActionPinManagerMergePinsConflict tests conflict detection
func TestActionPinManagerMergePinsConflict(t *testing.T) {
	tmpDir := t.TempDir()
	manager := NewActionPinManager(tmpDir)

	// Load builtin pins
	if err := manager.LoadBuiltinPins(); err != nil {
		t.Fatalf("Failed to load builtin pins: %v", err)
	}

	// Add a custom pin that conflicts with a builtin pin
	builtinCheckout := manager.builtinPins["actions/checkout"]
	manager.customPins["actions/checkout"] = ActionPin{
		Repo:    "actions/checkout",
		Version: builtinCheckout.Version, // Same version
		SHA:     "0000000000000000000000000000000000000000", // Different SHA
	}

	// Merge pins - should fail due to conflict
	err := manager.MergePins()
	if err == nil {
		t.Fatal("Expected error when merging conflicting pins, got nil")
	}

	if !strings.Contains(err.Error(), "conflict") {
		t.Errorf("Expected error message to contain 'conflict', got: %v", err)
	}
}

// TestActionPinManagerFindPin tests finding pins
func TestActionPinManagerFindPin(t *testing.T) {
	tmpDir := t.TempDir()
	manager := NewActionPinManager(tmpDir)

	// Load and merge pins
	if err := manager.LoadBuiltinPins(); err != nil {
		t.Fatalf("Failed to load builtin pins: %v", err)
	}
	if err := manager.MergePins(); err != nil {
		t.Fatalf("Failed to merge pins: %v", err)
	}

	// Test finding a pin with exact version match
	pin, found := manager.FindPin("actions/checkout", "v5")
	if !found {
		t.Error("Expected to find actions/checkout@v5")
	}
	if pin.Repo != "actions/checkout" {
		t.Errorf("Expected repo to be actions/checkout, got %s", pin.Repo)
	}
	if pin.Version != "v5" {
		t.Errorf("Expected version to be v5, got %s", pin.Version)
	}

	// Test finding a pin without version (should find any version)
	pin, found = manager.FindPin("actions/checkout", "")
	if !found {
		t.Error("Expected to find actions/checkout without version")
	}

	// Test finding a non-existent pin
	_, found = manager.FindPin("unknown/action", "v1")
	if found {
		t.Error("Expected not to find unknown/action")
	}
}

// TestActionPinManagerSaveNonBuiltinPins tests saving non-builtin pins
func TestActionPinManagerSaveNonBuiltinPins(t *testing.T) {
	tmpDir := t.TempDir()
	manager := NewActionPinManager(tmpDir)

	// Load builtin pins
	if err := manager.LoadBuiltinPins(); err != nil {
		t.Fatalf("Failed to load builtin pins: %v", err)
	}

	// Create some resolved pins (mix of builtin and non-builtin)
	resolvedPins := []ActionPin{
		{
			Repo:    "actions/checkout",
			Version: "v5",
			SHA:     "08c6903cd8c0fde910a37f88322edcfb5dd907a8",
		},
		{
			Repo:    "custom/action",
			Version: "v1",
			SHA:     "1234567890123456789012345678901234567890",
		},
		{
			Repo:    "another/action",
			Version: "v2",
			SHA:     "abcdef1234567890abcdef1234567890abcdef12",
		},
	}

	// Save non-builtin pins
	err := manager.SaveNonBuiltinPins(resolvedPins)
	if err != nil {
		t.Fatalf("Failed to save non-builtin pins: %v", err)
	}

	// Verify file was created
	nonBuiltinPinsPath := filepath.Join(tmpDir, ".github", "aw", NonBuiltinPinsFileName)
	if _, err := os.Stat(nonBuiltinPinsPath); os.IsNotExist(err) {
		t.Error("Expected non-builtin pins file to be created")
	}

	// Load and verify contents
	data, err := os.ReadFile(nonBuiltinPinsPath)
	if err != nil {
		t.Fatalf("Failed to read non-builtin pins file: %v", err)
	}

	var savedData ActionPinsData
	if err := json.Unmarshal(data, &savedData); err != nil {
		t.Fatalf("Failed to unmarshal non-builtin pins: %v", err)
	}

	// Should have 2 non-builtin pins (custom/action and another/action)
	if len(savedData.Actions) != 2 {
		t.Errorf("Expected 2 non-builtin pins, got %d", len(savedData.Actions))
	}

	// Verify custom/action is saved
	if pin, exists := savedData.Actions["custom/action"]; !exists {
		t.Error("Expected custom/action in saved non-builtin pins")
	} else if pin.SHA != "1234567890123456789012345678901234567890" {
		t.Errorf("Expected SHA to be 1234567890123456789012345678901234567890, got %s", pin.SHA)
	}

	// Verify actions/checkout is NOT saved (it's builtin)
	if _, exists := savedData.Actions["actions/checkout"]; exists {
		t.Error("Did not expect actions/checkout in saved non-builtin pins")
	}
}

// TestActionPinManagerSaveNonBuiltinPinsEmpty tests saving when no non-builtin pins
func TestActionPinManagerSaveNonBuiltinPinsEmpty(t *testing.T) {
	tmpDir := t.TempDir()
	manager := NewActionPinManager(tmpDir)

	// Load builtin pins
	if err := manager.LoadBuiltinPins(); err != nil {
		t.Fatalf("Failed to load builtin pins: %v", err)
	}

	// Create resolved pins (all builtin)
	resolvedPins := []ActionPin{
		{
			Repo:    "actions/checkout",
			Version: "v5",
			SHA:     "08c6903cd8c0fde910a37f88322edcfb5dd907a8",
		},
	}

	// Save non-builtin pins
	err := manager.SaveNonBuiltinPins(resolvedPins)
	if err != nil {
		t.Fatalf("Failed to save non-builtin pins: %v", err)
	}

	// Verify file was NOT created (or was removed if it existed)
	nonBuiltinPinsPath := filepath.Join(tmpDir, ".github", "aw", NonBuiltinPinsFileName)
	if _, err := os.Stat(nonBuiltinPinsPath); err == nil {
		t.Error("Expected non-builtin pins file to not exist when there are no non-builtin pins")
	}
}
