package workflow

import (
	"strings"
	"testing"
)

// TestSafeInputsStepCodeGenerationStability verifies that the MCP setup step code generation
// for safe-inputs produces stable, deterministic output when called multiple times.
// This test ensures that tools are sorted before generating cat commands.
func TestSafeInputsStepCodeGenerationStability(t *testing.T) {
	// Create a config with multiple tools to ensure sorting is tested
	safeInputsConfig := &SafeInputsConfig{
		Tools: map[string]*SafeInputToolConfig{
			"zebra-shell": {
				Name:        "zebra-shell",
				Description: "A shell tool that starts with Z",
				Run:         "echo zebra",
			},
			"alpha-js": {
				Name:        "alpha-js",
				Description: "A JS tool that starts with A",
				Script:      "return 'alpha';",
			},
			"middle-shell": {
				Name:        "middle-shell",
				Description: "A shell tool in the middle",
				Run:         "echo middle",
			},
			"beta-js": {
				Name:        "beta-js",
				Description: "A JS tool that starts with B",
				Script:      "return 'beta';",
			},
		},
	}

	workflowData := &WorkflowData{
		SafeInputs: safeInputsConfig,
		Tools:      make(map[string]any),
		Features: map[string]any{
			"safe-inputs": true, // Feature flag is optional now
		},
	}

	// Generate MCP setup code multiple times using the actual compiler method
	iterations := 10
	outputs := make([]string, iterations)
	compiler := &Compiler{}

	// Create a mock engine that does nothing for MCP config
	mockEngine := &CustomEngine{}

	for i := 0; i < iterations; i++ {
		var yaml strings.Builder
		compiler.generateMCPSetup(&yaml, workflowData.Tools, mockEngine, workflowData)
		outputs[i] = yaml.String()
	}

	// All iterations should produce identical output
	for i := 1; i < iterations; i++ {
		if outputs[i] != outputs[0] {
			t.Errorf("generateMCPSetup produced different output on iteration %d", i+1)
			// Find first difference for debugging
			for j := 0; j < len(outputs[0]) && j < len(outputs[i]); j++ {
				if outputs[0][j] != outputs[i][j] {
					start := j - 100
					if start < 0 {
						start = 0
					}
					end := j + 100
					if end > len(outputs[0]) {
						end = len(outputs[0])
					}
					if end > len(outputs[i]) {
						end = len(outputs[i])
					}
					t.Errorf("First difference at position %d:\n  Expected: %q\n  Got: %q", j, outputs[0][start:end], outputs[i][start:end])
					break
				}
			}
		}
	}

	// Verify tools appear in sorted order in the output
	// All tools are sorted alphabetically regardless of type (JavaScript or shell):
	// alpha-js, beta-js, middle-shell, zebra-shell
	alphaPos := strings.Index(outputs[0], "alpha-js")
	betaPos := strings.Index(outputs[0], "beta-js")
	middlePos := strings.Index(outputs[0], "middle-shell")
	zebraPos := strings.Index(outputs[0], "zebra-shell")

	if alphaPos == -1 || betaPos == -1 || middlePos == -1 || zebraPos == -1 {
		t.Error("Output should contain all tool names")
	}

	// Verify alphabetical sorting: alpha < beta < middle < zebra
	if alphaPos >= betaPos || betaPos >= middlePos || middlePos >= zebraPos {
		t.Errorf("Tools should be sorted alphabetically in step code: alpha(%d) < beta(%d) < middle(%d) < zebra(%d)",
			alphaPos, betaPos, middlePos, zebraPos)
	}
}
