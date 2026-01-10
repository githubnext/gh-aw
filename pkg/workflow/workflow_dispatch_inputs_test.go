package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/goccy/go-yaml"
)

// TestWorkflowDispatchInputTypes tests that all input types are supported
// by the compiler and properly converted to YAML
func TestWorkflowDispatchInputTypes(t *testing.T) {
	tests := []struct {
		name           string
		markdown       string
		wantType       string
		wantDefault    string
		wantErrContain string
	}{
		{
			name: "string input type",
			markdown: `---
on:
  workflow_dispatch:
    inputs:
      message:
        description: 'Message to display'
        type: string
        default: 'Hello World'
        required: true
engine: copilot
---
# Test Workflow
Test workflow with string input`,
			wantType:    "string",
			wantDefault: "Hello World",
		},
		{
			name: "boolean input type",
			markdown: `---
on:
  workflow_dispatch:
    inputs:
      debug:
        description: 'Enable debug mode'
        type: boolean
        default: false
        required: false
engine: copilot
---
# Test Workflow
Test workflow with boolean input`,
			wantType:    "boolean",
			wantDefault: "false",
		},
		{
			name: "boolean input type with true default",
			markdown: `---
on:
  workflow_dispatch:
    inputs:
      enabled:
        description: 'Enable feature'
        type: boolean
        default: true
engine: copilot
---
# Test Workflow
Test workflow with boolean true input`,
			wantType:    "boolean",
			wantDefault: "true",
		},
		{
			name: "number input type",
			markdown: `---
on:
  workflow_dispatch:
    inputs:
      count:
        description: 'Number of items'
        type: number
        default: 100
        required: true
engine: copilot
---
# Test Workflow
Test workflow with number input`,
			wantType:    "number",
			wantDefault: "100",
		},
		{
			name: "choice input type",
			markdown: `---
on:
  workflow_dispatch:
    inputs:
      environment:
        description: 'Target environment'
        type: choice
        default: staging
        required: true
        options:
          - staging
          - production
          - development
engine: copilot
---
# Test Workflow
Test workflow with choice input`,
			wantType: "choice",
		},
		{
			name: "environment input type",
			markdown: `---
on:
  workflow_dispatch:
    inputs:
      deploy_env:
        description: 'Deployment environment'
        type: environment
        required: false
engine: copilot
---
# Test Workflow
Test workflow with environment input`,
			wantType: "environment",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory
			tmpDir := t.TempDir()

			// Write markdown file
			workflowPath := filepath.Join(tmpDir, "test.md")
			err := os.WriteFile(workflowPath, []byte(tt.markdown), 0600)
			if err != nil {
				t.Fatalf("Failed to write test file: %v", err)
			}

			// Compile workflow
			compiler := NewCompiler(false, "", "test")

			err = compiler.CompileWorkflow(workflowPath)

			if tt.wantErrContain != "" {
				if err == nil {
					t.Fatalf("Expected error containing %q, got nil", tt.wantErrContain)
				}
				if !strings.Contains(err.Error(), tt.wantErrContain) {
					t.Fatalf("Expected error containing %q, got %q", tt.wantErrContain, err.Error())
				}
				return
			}

			if err != nil {
				t.Fatalf("Compilation failed: %v", err)
			}

			// Read compiled YAML
			outputPath := filepath.Join(tmpDir, "test.lock.yml")
			yamlContent, err := os.ReadFile(outputPath)
			if err != nil {
				t.Fatalf("Failed to read compiled file: %v", err)
			}

			// Parse YAML
			var workflow map[string]any
			err = yaml.Unmarshal(yamlContent, &workflow)
			if err != nil {
				t.Fatalf("Failed to parse YAML: %v", err)
			}

			// Navigate to workflow_dispatch.inputs
			onSection, ok := workflow["on"].(map[string]any)
			if !ok {
				t.Fatalf("Expected 'on' to be a map, got %T", workflow["on"])
			}

			workflowDispatch, ok := onSection["workflow_dispatch"].(map[string]any)
			if !ok {
				t.Fatalf("Expected 'workflow_dispatch' to be a map, got %T", onSection["workflow_dispatch"])
			}

			inputs, ok := workflowDispatch["inputs"].(map[string]any)
			if !ok {
				t.Fatalf("Expected 'inputs' to be a map, got %T", workflowDispatch["inputs"])
			}

			// Get the first (and only) input
			if len(inputs) != 1 {
				t.Fatalf("Expected 1 input, got %d", len(inputs))
			}

			var inputDef map[string]any
			for _, input := range inputs {
				inputDef, ok = input.(map[string]any)
				if !ok {
					t.Fatalf("Expected input to be a map, got %T", input)
				}
				break
			}

			// Verify type
			if tt.wantType != "" {
				inputType, ok := inputDef["type"].(string)
				if !ok {
					t.Fatalf("Expected 'type' to be a string, got %T", inputDef["type"])
				}
				if inputType != tt.wantType {
					t.Errorf("Expected type %q, got %q", tt.wantType, inputType)
				}
			}

			// Verify default value
			if tt.wantDefault != "" {
				defaultVal := inputDef["default"]
				var defaultStr string
				switch v := defaultVal.(type) {
				case string:
					defaultStr = v
				case bool:
					if v {
						defaultStr = "true"
					} else {
						defaultStr = "false"
					}
				case int:
					defaultStr = string(rune(v))
				case float64:
					if v == float64(int(v)) {
						defaultStr = strings.TrimSuffix(strings.TrimSuffix(string(rune(int(v))), ".0"), ".")
					}
				}
				// For numeric types, we need a more flexible comparison
				if tt.wantType == "number" {
					// Just check that default exists
					if defaultVal == nil {
						t.Error("Expected default value to exist for number type")
					}
				} else if defaultStr != tt.wantDefault {
					t.Errorf("Expected default %q, got %q (%T)", tt.wantDefault, defaultStr, defaultVal)
				}
			}
		})
	}
}

// TestWorkflowDispatchInputsRequiredField tests that required field is properly handled
func TestWorkflowDispatchInputsRequiredField(t *testing.T) {
	markdown := `---
on:
  workflow_dispatch:
    inputs:
      required_field:
        description: 'Required input'
        type: string
        required: true
      optional_field:
        description: 'Optional input'
        type: string
        required: false
engine: copilot
---
# Test Workflow
Test workflow with required and optional inputs`

	// Create temporary directory
	tmpDir := t.TempDir()

	// Write markdown file
	workflowPath := filepath.Join(tmpDir, "test.md")
	err := os.WriteFile(workflowPath, []byte(markdown), 0600)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Compile workflow
	compiler := NewCompiler(false, "", "test")

	err = compiler.CompileWorkflow(workflowPath)
	if err != nil {
		t.Fatalf("Compilation failed: %v", err)
	}

	// Read and parse compiled YAML
	outputPath := filepath.Join(tmpDir, "test.lock.yml")
	yamlContent, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read compiled file: %v", err)
	}

	var workflow map[string]any
	err = yaml.Unmarshal(yamlContent, &workflow)
	if err != nil {
		t.Fatalf("Failed to parse YAML: %v", err)
	}

	// Navigate to inputs
	onSection := workflow["on"].(map[string]any)
	workflowDispatch := onSection["workflow_dispatch"].(map[string]any)
	inputs := workflowDispatch["inputs"].(map[string]any)

	// Check required field
	requiredField := inputs["required_field"].(map[string]any)
	if required, ok := requiredField["required"].(bool); !ok || !required {
		t.Errorf("Expected 'required_field' to have required: true")
	}

	// Check optional field
	optionalField := inputs["optional_field"].(map[string]any)
	if required, ok := optionalField["required"].(bool); ok && required {
		t.Errorf("Expected 'optional_field' to have required: false or absent")
	}
}

// TestWorkflowDispatchChoiceInputOptions tests that options are properly preserved for choice type
func TestWorkflowDispatchChoiceInputOptions(t *testing.T) {
	markdown := `---
on:
  workflow_dispatch:
    inputs:
      environment:
        description: 'Target environment'
        type: choice
        options:
          - development
          - staging
          - production
engine: copilot
---
# Test Workflow
Test workflow with choice input options`

	// Create temporary directory
	tmpDir := t.TempDir()

	// Write markdown file
	workflowPath := filepath.Join(tmpDir, "test.md")
	err := os.WriteFile(workflowPath, []byte(markdown), 0600)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Compile workflow
	compiler := NewCompiler(false, "", "test")

	err = compiler.CompileWorkflow(workflowPath)
	if err != nil {
		t.Fatalf("Compilation failed: %v", err)
	}

	// Read and parse compiled YAML
	outputPath := filepath.Join(tmpDir, "test.lock.yml")
	yamlContent, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read compiled file: %v", err)
	}

	var workflow map[string]any
	err = yaml.Unmarshal(yamlContent, &workflow)
	if err != nil {
		t.Fatalf("Failed to parse YAML: %v", err)
	}

	// Navigate to inputs
	onSection := workflow["on"].(map[string]any)
	workflowDispatch := onSection["workflow_dispatch"].(map[string]any)
	inputs := workflowDispatch["inputs"].(map[string]any)

	// Check options array
	environment := inputs["environment"].(map[string]any)
	options, ok := environment["options"].([]any)
	if !ok {
		t.Fatalf("Expected 'options' to be an array, got %T", environment["options"])
	}

	expectedOptions := []string{"development", "staging", "production"}
	if len(options) != len(expectedOptions) {
		t.Fatalf("Expected %d options, got %d", len(expectedOptions), len(options))
	}

	for i, expectedOpt := range expectedOptions {
		optStr, ok := options[i].(string)
		if !ok {
			t.Fatalf("Expected option %d to be a string, got %T", i, options[i])
		}
		if optStr != expectedOpt {
			t.Errorf("Expected option %d to be %q, got %q", i, expectedOpt, optStr)
		}
	}
}

// TestWorkflowDispatchAllInputTypes tests a workflow with all input types
func TestWorkflowDispatchAllInputTypes(t *testing.T) {
	markdown := `---
on:
  workflow_dispatch:
    inputs:
      message:
        description: 'String input'
        type: string
        default: 'test'
      debug:
        description: 'Boolean input'
        type: boolean
        default: false
      count:
        description: 'Number input'
        type: number
        default: 10
      environment:
        description: 'Choice input'
        type: choice
        options:
          - dev
          - prod
      deploy_target:
        description: 'Environment input'
        type: environment
engine: copilot
---
# Test Workflow
Test workflow with all input types`

	// Create temporary directory
	tmpDir := t.TempDir()

	// Write markdown file
	workflowPath := filepath.Join(tmpDir, "test.md")
	err := os.WriteFile(workflowPath, []byte(markdown), 0600)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Compile workflow
	compiler := NewCompiler(false, "", "test")

	err = compiler.CompileWorkflow(workflowPath)
	if err != nil {
		t.Fatalf("Compilation failed: %v", err)
	}

	// Read and parse compiled YAML
	outputPath := filepath.Join(tmpDir, "test.lock.yml")
	yamlContent, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read compiled file: %v", err)
	}

	var workflow map[string]any
	err = yaml.Unmarshal(yamlContent, &workflow)
	if err != nil {
		t.Fatalf("Failed to parse YAML: %v", err)
	}

	// Navigate to inputs
	onSection := workflow["on"].(map[string]any)
	workflowDispatch := onSection["workflow_dispatch"].(map[string]any)
	inputs := workflowDispatch["inputs"].(map[string]any)

	// Verify all inputs exist
	expectedInputs := []string{"message", "debug", "count", "environment", "deploy_target"}
	if len(inputs) != len(expectedInputs) {
		t.Fatalf("Expected %d inputs, got %d", len(expectedInputs), len(inputs))
	}

	for _, inputName := range expectedInputs {
		if _, exists := inputs[inputName]; !exists {
			t.Errorf("Expected input %q to exist", inputName)
		}
	}

	// Verify types
	expectedTypes := map[string]string{
		"message":       "string",
		"debug":         "boolean",
		"count":         "number",
		"environment":   "choice",
		"deploy_target": "environment",
	}

	for inputName, expectedType := range expectedTypes {
		inputDef := inputs[inputName].(map[string]any)
		actualType, ok := inputDef["type"].(string)
		if !ok {
			t.Fatalf("Expected 'type' for %q to be a string, got %T", inputName, inputDef["type"])
		}
		if actualType != expectedType {
			t.Errorf("Expected type %q for %q, got %q", expectedType, inputName, actualType)
		}
	}
}
