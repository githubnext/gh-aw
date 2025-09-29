package constants

import (
	"testing"
)

// TestDefaultBashToolsIncludesMake verifies that the default bash tools include make commands
func TestDefaultBashToolsIncludesMake(t *testing.T) {
	found := false
	for _, tool := range DefaultBashTools {
		if tool == "make:*" {
			found = true
			break
		}
	}
	
	if !found {
		t.Errorf("DefaultBashTools should include 'make:*' but doesn't. Current tools: %v", DefaultBashTools)
	}
}

// TestDefaultBashToolsContainsExpectedCommands verifies that all expected commands are present
func TestDefaultBashToolsContainsExpectedCommands(t *testing.T) {
	expectedTools := []string{
		"echo",
		"ls", 
		"pwd",
		"cat",
		"head",
		"tail",
		"grep",
		"wc",
		"sort",
		"uniq",
		"date",
		"make:*",
	}
	
	toolsMap := make(map[string]bool)
	for _, tool := range DefaultBashTools {
		toolsMap[tool] = true
	}
	
	for _, expected := range expectedTools {
		if !toolsMap[expected] {
			t.Errorf("Expected tool '%s' not found in DefaultBashTools: %v", expected, DefaultBashTools)
		}
	}
	
	// Verify the count matches expectations
	if len(DefaultBashTools) != len(expectedTools) {
		t.Errorf("Expected %d tools in DefaultBashTools, but got %d: %v", len(expectedTools), len(DefaultBashTools), DefaultBashTools)
	}
}