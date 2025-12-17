package testutil

import (
	"testing"
)

func TestNewTestFeature(t *testing.T) {
	name := "test-feature"
	tf := NewTestFeature(name)
	
	if tf == nil {
		t.Fatal("NewTestFeature returned nil")
	}
	
	if tf.GetName() != name {
		t.Errorf("Expected name %q, got %q", name, tf.GetName())
	}
}

func TestTestFeature_GetName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"basic name", "feature-1", "feature-1"},
		{"empty name", "", ""},
		{"special chars", "feature@123", "feature@123"},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tf := NewTestFeature(tt.input)
			if got := tf.GetName(); got != tt.expected {
				t.Errorf("GetName() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestTestFeature_Validate(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"valid name", "test", true},
		{"empty name", "", false},
		{"whitespace only", "   ", true}, // non-empty string
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tf := NewTestFeature(tt.input)
			if got := tf.Validate(); got != tt.expected {
				t.Errorf("Validate() = %v, want %v", got, tt.expected)
			}
		})
	}
}
