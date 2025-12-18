package testutil

import "testing"

// TestSpecKitFeature tests the basic functionality of the spec-kit test feature.
func TestSpecKitFeature(t *testing.T) {
	result := SpecKitTestFeature()
	expected := "Spec-Kit Test Feature: OK"

	if result != expected {
		t.Errorf("SpecKitTestFeature() = %q; want %q", result, expected)
	}
}

// TestSpecKitFeatureGreeting tests the greeting functionality.
func TestSpecKitFeatureGreeting(t *testing.T) {
	name := "Spec-Kit Workflow"
	result := SpecKitTestFeatureGreeting(name)
	expected := "Hello, Spec-Kit Workflow! This is a test feature."

	if result != expected {
		t.Errorf("SpecKitTestFeatureGreeting(%q) = %q; want %q", name, result, expected)
	}
}
