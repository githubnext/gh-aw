package workflow

import (
	"strings"
	"testing"
)

// TestGenerateSerenaLanguageServiceStepsDeterministicOrder verifies that
// GenerateSerenaLanguageServiceSteps produces steps in a consistent order
// across multiple invocations, ensuring deterministic compilation
func TestGenerateSerenaLanguageServiceStepsDeterministicOrder(t *testing.T) {
	// Create a Serena configuration with multiple languages in map form
	// Maps have non-deterministic iteration order in Go, so this tests
	// that we properly sort the languages before generating steps
	tools := map[string]any{
		"serena": map[string]any{
			"languages": map[string]any{
				"typescript": map[string]any{},
				"go":         map[string]any{},
				"python":     map[string]any{},
				"rust":       map[string]any{},
			},
		},
	}

	// Generate steps multiple times and verify they're always in the same order
	var firstResult []GitHubActionStep
	const iterations = 10

	for i := 0; i < iterations; i++ {
		steps := GenerateSerenaLanguageServiceSteps(tools)

		if i == 0 {
			firstResult = steps
		} else {
			// Verify the order matches the first result
			if len(steps) != len(firstResult) {
				t.Errorf("Iteration %d: Got %d steps, expected %d", i, len(steps), len(firstResult))
				continue
			}

			for j := range steps {
				if len(steps[j]) != len(firstResult[j]) {
					t.Errorf("Iteration %d, step %d: Got %d lines, expected %d", i, j, len(steps[j]), len(firstResult[j]))
					continue
				}
				for k := range steps[j] {
					if steps[j][k] != firstResult[j][k] {
						t.Errorf("Iteration %d, step %d, line %d: Got %q, expected %q", i, j, k, steps[j][k], firstResult[j][k])
					}
				}
			}
		}
	}

	// Additionally verify that the steps are in alphabetical order by language
	// Expected order: go, python, rust, typescript (alphabetically)
	expectedLanguages := []string{"go", "python", "rust", "typescript"}
	if len(firstResult) != len(expectedLanguages) {
		t.Fatalf("Expected %d steps for languages %v, got %d steps", len(expectedLanguages), expectedLanguages, len(firstResult))
	}

	// Check that each step name contains the expected language in alphabetical order
	for i, expectedLang := range expectedLanguages {
		stepName := firstResult[i][0] // First line contains "- name: Install X language service"

		var langInStep string
		switch expectedLang {
		case "go":
			langInStep = "Go"
		case "python":
			langInStep = "Python"
		case "rust":
			langInStep = "Rust"
		case "typescript":
			langInStep = "TypeScript"
		}

		if !strings.Contains(stepName, langInStep) {
			t.Errorf("Step %d should be for %s language, but got: %s", i, langInStep, stepName)
		}
	}
}
