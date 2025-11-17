package cli

import (
	"os/exec"
	"testing"
)

// TestValidateNodeEnvironment tests the Node.js environment validation
func TestValidateNodeEnvironment(t *testing.T) {
	// This test verifies that ValidateNodeEnvironment correctly checks for Node.js and npm
	err := ValidateNodeEnvironment(false)

	// Check if Node.js is installed on the system
	_, nodeErr := exec.LookPath("node")
	_, npmErr := exec.LookPath("npm")

	if nodeErr != nil || npmErr != nil {
		// If Node.js or npm is not installed, the validation should fail
		if err == nil {
			t.Error("Expected error when Node.js or npm is not installed, got nil")
		}
	} else {
		// If Node.js and npm are installed, the validation should succeed
		if err != nil {
			t.Errorf("Expected no error when Node.js and npm are installed, got: %v", err)
		}
	}
}

// TestDevDepsCommandCreation tests that the dev-deps command can be created
func TestDevDepsCommandCreation(t *testing.T) {
	cmd := NewDevDepsCommand()

	if cmd == nil {
		t.Fatal("NewDevDepsCommand() returned nil")
	}

	if cmd.Use != "dev-deps" {
		t.Errorf("Expected command Use to be 'dev-deps', got '%s'", cmd.Use)
	}

	if cmd.Short == "" {
		t.Error("Expected command Short description to be non-empty")
	}

	if cmd.Long == "" {
		t.Error("Expected command Long description to be non-empty")
	}
}
