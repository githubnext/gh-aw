package cli

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestStatusWorkflows_JSONOutput(t *testing.T) {
	// Save current directory
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}

	// Change to repository root
	repoRoot := filepath.Join(originalDir, "..", "..")
	if err := os.Chdir(repoRoot); err != nil {
		t.Fatalf("Failed to change to repository root: %v", err)
	}
	defer os.Chdir(originalDir)

	// Test JSON output without pattern
	t.Run("JSON output without pattern", func(t *testing.T) {
		err := StatusWorkflows("", false, true, "")
		if err != nil {
			t.Errorf("StatusWorkflows with JSON flag failed: %v", err)
		}
		// Note: We can't easily capture stdout in this test,
		// but we verify it doesn't error
	})

	// Test JSON output with pattern
	t.Run("JSON output with pattern", func(t *testing.T) {
		err := StatusWorkflows("smoke", false, true, "")
		if err != nil {
			t.Errorf("StatusWorkflows with JSON flag and pattern failed: %v", err)
		}
	})
}

func TestStatusWorkflows_JqFilter(t *testing.T) {
	// Check if jq is available
	if _, err := exec.LookPath("jq"); err != nil {
		t.Skip("jq not found in PATH, skipping jq filter tests")
	}

	// Save current directory
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}

	// Change to repository root
	repoRoot := filepath.Join(originalDir, "..", "..")
	if err := os.Chdir(repoRoot); err != nil {
		t.Fatalf("Failed to change to repository root: %v", err)
	}
	defer os.Chdir(originalDir)

	// Test jq filter
	t.Run("Valid jq filter", func(t *testing.T) {
		err := StatusWorkflows("", false, true, ".[0:1]")
		if err != nil {
			t.Errorf("StatusWorkflows with jq filter failed: %v", err)
		}
	})

	// Test invalid jq filter
	t.Run("Invalid jq filter", func(t *testing.T) {
		err := StatusWorkflows("", false, true, ".[invalid")
		if err == nil {
			t.Error("StatusWorkflows should fail with invalid jq filter")
		}
	})
}

func TestWorkflowStatus_JSONMarshaling(t *testing.T) {
	// Test that WorkflowStatus can be marshaled to JSON
	status := WorkflowStatus{
		Workflow:      "test-workflow",
		Agent:         "copilot",
		Compiled:      "Yes",
		Status:        "active",
		TimeRemaining: "N/A",
	}

	jsonBytes, err := json.Marshal(status)
	if err != nil {
		t.Fatalf("Failed to marshal WorkflowStatus: %v", err)
	}

	// Verify JSON contains expected fields
	var unmarshaled map[string]interface{}
	if err := json.Unmarshal(jsonBytes, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	if unmarshaled["workflow"] != "test-workflow" {
		t.Errorf("Expected workflow='test-workflow', got %v", unmarshaled["workflow"])
	}
	if unmarshaled["agent"] != "copilot" {
		t.Errorf("Expected agent='copilot', got %v", unmarshaled["agent"])
	}
	if unmarshaled["compiled"] != "Yes" {
		t.Errorf("Expected compiled='Yes', got %v", unmarshaled["compiled"])
	}
	if unmarshaled["status"] != "active" {
		t.Errorf("Expected status='active', got %v", unmarshaled["status"])
	}
	if unmarshaled["time_remaining"] != "N/A" {
		t.Errorf("Expected time_remaining='N/A', got %v", unmarshaled["time_remaining"])
	}
}

func TestOutputJSON(t *testing.T) {
	// Check if jq is available for jq filter tests
	jqAvailable := false
	if _, err := exec.LookPath("jq"); err == nil {
		jqAvailable = true
	}

	statuses := []WorkflowStatus{
		{
			Workflow:      "workflow1",
			Agent:         "copilot",
			Compiled:      "Yes",
			Status:        "active",
			TimeRemaining: "N/A",
		},
		{
			Workflow:      "workflow2",
			Agent:         "claude",
			Compiled:      "No",
			Status:        "disabled",
			TimeRemaining: "1h 30m",
		},
	}

	// Test outputJSON without jq filter
	t.Run("Output JSON without jq filter", func(t *testing.T) {
		err := outputJSON(statuses, "")
		if err != nil {
			t.Errorf("outputJSON failed: %v", err)
		}
	})

	// Test outputJSON with jq filter (if available)
	if jqAvailable {
		t.Run("Output JSON with jq filter", func(t *testing.T) {
			err := outputJSON(statuses, ".[0]")
			if err != nil {
				t.Errorf("outputJSON with jq filter failed: %v", err)
			}
		})

		t.Run("Output JSON with invalid jq filter", func(t *testing.T) {
			err := outputJSON(statuses, ".[invalid")
			if err == nil {
				t.Error("outputJSON should fail with invalid jq filter")
			}
		})
	} else {
		t.Run("Output JSON with jq filter when jq not available", func(t *testing.T) {
			err := outputJSON(statuses, ".[0]")
			if err == nil {
				t.Error("outputJSON should fail when jq is not available")
			}
		})
	}
}
