package testutil

import "testing"

// TestTestFeature tests the TestFeature function
func TestTestFeature(t *testing.T) {
	result := TestFeature()
	expected := "test feature works"
	
	if result != expected {
		t.Errorf("TestFeature() = %q, want %q", result, expected)
	}
}
