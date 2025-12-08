package testutil

import "testing"

func TestTestFeature(t *testing.T) {
	result := TestFeature()
	expected := "test feature working"
	
	if result != expected {
		t.Errorf("TestFeature() = %q, want %q", result, expected)
	}
}
