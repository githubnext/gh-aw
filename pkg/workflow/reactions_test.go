package workflow

import (
	"sort"
	"testing"
)

func TestIsValidReaction(t *testing.T) {
	tests := []struct {
		name     string
		reaction string
		expected bool
	}{
		{"+1 is valid", "+1", true},
		{"-1 is valid", "-1", true},
		{"laugh is valid", "laugh", true},
		{"confused is valid", "confused", true},
		{"heart is valid", "heart", true},
		{"hooray is valid", "hooray", true},
		{"rocket is valid", "rocket", true},
		{"eyes is valid", "eyes", true},
		{"none is valid", "none", true},
		{"invalid reaction", "thumbsup", false},
		{"empty string is invalid", "", false},
		{"random string is invalid", "random", false},
		{"case sensitive - uppercase invalid", "HEART", false},
		{"case sensitive - mixed case invalid", "Laugh", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidReaction(tt.reaction)
			if result != tt.expected {
				t.Errorf("isValidReaction(%q) = %v, want %v", tt.reaction, result, tt.expected)
			}
		})
	}
}

func TestGetValidReactions(t *testing.T) {
	reactions := getValidReactions()

	if len(reactions) == 0 {
		t.Error("getValidReactions() returned empty slice")
	}

	expectedReactions := []string{"+1", "-1", "laugh", "confused", "heart", "hooray", "rocket", "eyes", "none"}
	if len(reactions) != len(expectedReactions) {
		t.Errorf("getValidReactions() returned %d reactions, want %d", len(reactions), len(expectedReactions))
	}

	// Sort both slices for comparison
	sort.Strings(reactions)
	sort.Strings(expectedReactions)

	for i, expected := range expectedReactions {
		if reactions[i] != expected {
			t.Errorf("getValidReactions()[%d] = %q, want %q", i, reactions[i], expected)
		}
	}

	// Verify all returned reactions are valid
	for _, reaction := range reactions {
		if !isValidReaction(reaction) {
			t.Errorf("getValidReactions() returned invalid reaction: %q", reaction)
		}
	}
}

func TestValidReactionsMap(t *testing.T) {
	// Test that the validReactions map contains expected entries
	expectedCount := 9 // +1, -1, laugh, confused, heart, hooray, rocket, eyes, none
	if len(validReactions) != expectedCount {
		t.Errorf("validReactions map has %d entries, want %d", len(validReactions), expectedCount)
	}

	// Test that all entries in the map have value true
	for reaction, valid := range validReactions {
		if !valid {
			t.Errorf("validReactions[%q] = false, expected true", reaction)
		}
	}
}
