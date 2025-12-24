package testutil

import "testing"

// TestSpecKitBasicFeature tests the basic spec-kit test feature
func TestSpecKitBasicFeature(t *testing.T) {
	result := SpecKitBasicFeature()
	expected := "spec-kit test feature works"

	if result != expected {
		t.Errorf("SpecKitBasicFeature() = %q, want %q", result, expected)
	}
}

// TestSpecKitBasicFeatureNotEmpty ensures the function returns non-empty result
func TestSpecKitBasicFeatureNotEmpty(t *testing.T) {
	result := SpecKitBasicFeature()

	if result == "" {
		t.Error("SpecKitBasicFeature() returned empty string")
	}
}
