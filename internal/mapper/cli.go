package mapper

import (
	"encoding/json"
	"fmt"
	"os"
)

// CLIError represents an error in CLI format for testing
type CLIError struct {
	InstancePath string    `json:"instancePath"`
	Meta         ErrorMeta `json:"meta"`
}

// TestRunner provides a simple interface for testing the mapper
type TestRunner struct {
	Verbose bool
}

// NewTestRunner creates a new test runner
func NewTestRunner(verbose bool) *TestRunner {
	return &TestRunner{Verbose: verbose}
}

// RunTest executes a mapping test with the given YAML and error
func (tr *TestRunner) RunTest(yamlContent string, cliError CLIError) {
	if tr.Verbose {
		fmt.Printf("=== Testing Mapper ===\n")
		fmt.Printf("YAML Content:\n%s\n", yamlContent)
		fmt.Printf("Instance Path: %s\n", cliError.InstancePath)
		fmt.Printf("Error Kind: %s\n", cliError.Meta.Kind)
		if cliError.Meta.Property != "" {
			fmt.Printf("Property: %s\n", cliError.Meta.Property)
		}
		fmt.Printf("\n")
	}

	spans, err := MapErrorToSpans([]byte(yamlContent), cliError.InstancePath, cliError.Meta)
	if err != nil {
		fmt.Printf("ERROR: %v\n", err)
		return
	}

	if len(spans) == 0 {
		fmt.Printf("No spans returned\n")
		return
	}

	fmt.Printf("Results (%d spans):\n", len(spans))
	for i, span := range spans {
		fmt.Printf("  %d. Line %d:%d - %d:%d (confidence: %.2f)\n",
			i+1, span.StartLine, span.StartCol, span.EndLine, span.EndCol, span.Confidence)
		fmt.Printf("     Reason: %s\n", span.Reason)
	}

	if tr.Verbose {
		fmt.Printf("\n=== End Test ===\n\n")
	}
}

// RunTestFromJSON runs a test from JSON-formatted input
func (tr *TestRunner) RunTestFromJSON(yamlContent, errorJSON string) error {
	var cliError CLIError
	if err := json.Unmarshal([]byte(errorJSON), &cliError); err != nil {
		return fmt.Errorf("failed to parse error JSON: %w", err)
	}

	tr.RunTest(yamlContent, cliError)
	return nil
}

// PrintHelp prints usage information
func (tr *TestRunner) PrintHelp() {
	fmt.Printf(`JSON Schema Error Mapper Test Tool

USAGE:
  Set up YAML content and error JSON, then call RunTest methods.

ERROR JSON FORMAT:
  {
    "instancePath": "/path/to/property",
    "meta": {
      "kind": "type|required|additionalProperties|oneOf|anyOf",
      "property": "property_name",
      "schemaSnippet": "optional schema info"
    }
  }

EXAMPLE ERROR TYPES:
  - Type mismatch: {"instancePath": "/config/port", "meta": {"kind": "type"}}
  - Missing required: {"instancePath": "/config/missing", "meta": {"kind": "required", "property": "missing"}}
  - Additional property: {"instancePath": "/config/extra", "meta": {"kind": "additionalProperties", "property": "extra"}}
`)
}

// Example usage function that can be called from tests or a CLI
func ExampleUsage() {
	runner := NewTestRunner(true)

	// Example 1: Type error
	yaml1 := `config:
  port: "8080"
  host: "localhost"`

	error1 := CLIError{
		InstancePath: "/config/port",
		Meta:         ErrorMeta{Kind: "type"},
	}

	fmt.Println("Example 1: Type Error")
	runner.RunTest(yaml1, error1)

	// Example 2: Missing required property
	yaml2 := `config:
  host: "localhost"`

	error2 := CLIError{
		InstancePath: "/config/port",
		Meta:         ErrorMeta{Kind: "required", Property: "port"},
	}

	fmt.Println("Example 2: Missing Required Property")
	runner.RunTest(yaml2, error2)

	// Example 3: Additional property
	yaml3 := `config:
  port: 8080
  host: "localhost"
  extra: "not allowed"`

	error3 := CLIError{
		InstancePath: "/config/extra",
		Meta:         ErrorMeta{Kind: "additionalProperties", Property: "extra"},
	}

	fmt.Println("Example 3: Additional Property")
	runner.RunTest(yaml3, error3)
}

// RunInteractiveExample runs examples and can be called from main or tests
func RunInteractiveExample() {
	fmt.Println("JSON Schema Error Mapper - Interactive Examples")
	fmt.Println("=" + fmt.Sprintf("%50s", "="))

	ExampleUsage()
}

// SaveExampleToFile saves a test case to a file for external testing
func SaveExampleToFile(filename, yamlContent, errorJSON string) error {
	type TestCase struct {
		YAML  string   `json:"yaml"`
		Error CLIError `json:"error"`
	}

	var cliError CLIError
	if err := json.Unmarshal([]byte(errorJSON), &cliError); err != nil {
		return err
	}

	testCase := TestCase{
		YAML:  yamlContent,
		Error: cliError,
	}

	data, err := json.MarshalIndent(testCase, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filename, data, 0644)
}
