//go:build !integration

package cli

import (
	"encoding/json"
	"testing"
)

func TestInjectDockerUnavailableWarning_AddsWarningToValidResults(t *testing.T) {
	// Simulate compile output where both workflows compiled successfully.
	inputJSON := `[{"workflow":"a.md","valid":true,"errors":[],"warnings":[]},{"workflow":"b.md","valid":true,"errors":[],"warnings":[]}]`
	warningMsg := "docker is not available (cannot connect to Docker daemon). actionlint requires Docker."

	output := injectDockerUnavailableWarning(inputJSON, warningMsg)

	var results []ValidationResult
	if err := json.Unmarshal([]byte(output), &results); err != nil {
		t.Fatalf("Failed to parse injected output: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("Expected 2 results, got %d", len(results))
	}

	for _, r := range results {
		if !r.Valid {
			t.Errorf("Workflow %s should still be valid after Docker unavailable warning", r.Workflow)
		}
		if len(r.Warnings) != 1 {
			t.Errorf("Workflow %s should have 1 warning, got %d", r.Workflow, len(r.Warnings))
			continue
		}
		if r.Warnings[0].Type != "docker_unavailable" {
			t.Errorf("Expected warning type 'docker_unavailable', got '%s'", r.Warnings[0].Type)
		}
		if r.Warnings[0].Message != warningMsg {
			t.Errorf("Expected warning message %q, got %q", warningMsg, r.Warnings[0].Message)
		}
	}
}

func TestInjectDockerUnavailableWarning_PreservesInvalidResults(t *testing.T) {
	// One workflow failed to compile; the other succeeded.
	inputJSON := `[{"workflow":"bad.md","valid":false,"errors":[{"type":"parse_error","message":"syntax error"}],"warnings":[]},{"workflow":"good.md","valid":true,"errors":[],"warnings":[]}]`
	warningMsg := "docker is not available"

	output := injectDockerUnavailableWarning(inputJSON, warningMsg)

	var results []ValidationResult
	if err := json.Unmarshal([]byte(output), &results); err != nil {
		t.Fatalf("Failed to parse injected output: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("Expected 2 results, got %d", len(results))
	}

	// bad.md should remain invalid and still carry its original error.
	if results[0].Valid {
		t.Error("bad.md should remain invalid")
	}
	if len(results[0].Errors) != 1 || results[0].Errors[0].Type != "parse_error" {
		t.Error("bad.md should still have its original parse_error")
	}
	// good.md should be valid with the warning appended.
	if !results[1].Valid {
		t.Error("good.md should still be valid")
	}
	if len(results[1].Warnings) != 1 || results[1].Warnings[0].Type != "docker_unavailable" {
		t.Error("good.md should have the docker_unavailable warning")
	}
}

func TestInjectDockerUnavailableWarning_InvalidJSONReturnedUnchanged(t *testing.T) {
	invalidJSON := "not-valid-json"
	output := injectDockerUnavailableWarning(invalidJSON, "some warning")
	if output != invalidJSON {
		t.Errorf("Expected original output to be returned unchanged for invalid JSON, got: %s", output)
	}
}
