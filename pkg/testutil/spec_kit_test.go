package testutil

import "testing"

// TestSpecKitWorkflow is a test to validate the spec-kit workflow
func TestSpecKitWorkflow(t *testing.T) {
	message := GetSpecKitMessage()
	if message == "" {
		t.Error("Expected non-empty message")
	}
}
