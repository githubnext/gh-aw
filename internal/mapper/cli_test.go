package mapper

import (
	"testing"
)

func TestCLIFunctionality(t *testing.T) {
	runner := NewTestRunner(false) // Non-verbose for test output

	yaml := `config:
  port: "8080"
  host: "localhost"`

	error1 := CLIError{
		InstancePath: "/config/port",
		Meta:         ErrorMeta{Kind: "type"},
	}

	// Test that RunTest doesn't panic
	runner.RunTest(yaml, error1)

	// Test JSON parsing
	errorJSON := `{
		"instancePath": "/config/port",
		"meta": {
			"kind": "type"
		}
	}`

	err := runner.RunTestFromJSON(yaml, errorJSON)
	if err != nil {
		t.Fatalf("RunTestFromJSON failed: %v", err)
	}

	// Test invalid JSON
	invalidJSON := `{invalid json}`
	err = runner.RunTestFromJSON(yaml, invalidJSON)
	if err == nil {
		t.Error("Expected error for invalid JSON")
	}
}

func TestExampleUsage(t *testing.T) {
	// Test that ExampleUsage runs without panicking
	ExampleUsage()
}

// TestInteractiveExample tests the interactive example
func TestInteractiveExample(t *testing.T) {
	// Test that RunInteractiveExample runs without panicking
	RunInteractiveExample()
}
