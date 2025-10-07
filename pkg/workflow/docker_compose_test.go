package workflow

import (
	"strings"
	"testing"
)

func TestGenerateDockerCompose_SortedEnvironmentVariables(t *testing.T) {
	// Test that environment variables are sorted alphabetically
	envVars := map[string]any{
		"ZEBRA_VAR":   "zebra",
		"ALPHA_VAR":   "alpha",
		"CHARLIE_VAR": "charlie",
		"BRAVO_VAR":   "bravo",
	}

	result := generateDockerCompose("test-image:latest", envVars, "testTool", nil)

	// Check that environment variables appear in sorted order
	envSection := extractSection(result, "environment:", "networks:")
	expectedOrder := []string{
		"ALPHA_VAR=alpha",
		"BRAVO_VAR=bravo",
		"CHARLIE_VAR=charlie",
		"ZEBRA_VAR=zebra",
	}

	for i, expectedVar := range expectedOrder {
		if !strings.Contains(envSection, expectedVar) {
			t.Errorf("Expected environment variable %s not found in section", expectedVar)
		}

		// Check order
		if i > 0 {
			prevVar := expectedOrder[i-1]
			prevIdx := strings.Index(envSection, prevVar)
			currIdx := strings.Index(envSection, expectedVar)
			if prevIdx >= currIdx {
				t.Errorf("Environment variables not in sorted order: %s should come before %s", prevVar, expectedVar)
			}
		}
	}
}

func TestGenerateDockerCompose_EnvironmentBeforeNetworks(t *testing.T) {
	// Test that environment variables are placed in the environment section, not networks
	envVars := map[string]any{
		"TEST_VAR": "value",
	}

	result := generateDockerCompose("test-image:latest", envVars, "testTool", nil)

	// Find the tool service section (after the squid-proxy service)
	toolServicePos := strings.Index(result, "testTool:")
	if toolServicePos == -1 {
		t.Fatal("Tool service section not found")
	}

	// Look for environment and networks sections within the tool service
	toolSection := result[toolServicePos:]
	envPos := strings.Index(toolSection, "environment:")
	networksPos := strings.Index(toolSection, "networks:")

	if envPos == -1 {
		t.Fatal("Environment section not found in tool service")
	}
	if networksPos == -1 {
		t.Fatal("Networks section not found in tool service")
	}

	// Environment section should come before networks section within the tool service
	if envPos >= networksPos {
		t.Error("Environment section should come before networks section in tool service")
	}

	// Extract the environment section
	envSection := toolSection[envPos:networksPos]

	// Check that our custom variable is in the environment section
	if !strings.Contains(envSection, "TEST_VAR=value") {
		t.Error("Custom environment variable should be in environment section")
	}

	// Extract the networks section (from networks: to the next major section)
	nextSectionPos := strings.Index(toolSection[networksPos:], "depends_on:")
	var networksSection string
	if nextSectionPos == -1 {
		networksSection = toolSection[networksPos:]
	} else {
		networksSection = toolSection[networksPos : networksPos+nextSectionPos]
	}

	// Check that environment variables are NOT in the networks section
	if strings.Contains(networksSection, "TEST_VAR=value") {
		t.Error("Environment variable should NOT be in networks section")
	}
}

func TestGenerateDockerCompose_DeterministicOutput(t *testing.T) {
	// Test that generating the same config multiple times produces identical output
	envVars := map[string]any{
		"VAR_A": "a",
		"VAR_B": "b",
		"VAR_C": "c",
	}

	results := make([]string, 5)
	for i := 0; i < 5; i++ {
		results[i] = generateDockerCompose("test-image:latest", envVars, "testTool", nil)
	}

	// All results should be identical
	for i := 1; i < len(results); i++ {
		if results[i] != results[0] {
			t.Errorf("Output %d differs from output 0: determinism violated", i)
		}
	}
}

// extractSection extracts content between two markers
func extractSection(content, startMarker, endMarker string) string {
	startIdx := strings.Index(content, startMarker)
	if startIdx == -1 {
		return ""
	}

	endIdx := strings.Index(content[startIdx:], endMarker)
	if endIdx == -1 {
		return content[startIdx:]
	}

	return content[startIdx : startIdx+endIdx]
}
