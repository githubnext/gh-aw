package workflow

import (
	"strings"
	"testing"
)

func TestGenerateWorkflowComplete(t *testing.T) {
	tests := []struct {
		name           string
		expectedSteps  []string
		unexpectedSteps []string
	}{
		{
			name: "generates simplified workflow complete step",
			expectedSteps: []string{
				"- name: Upload workflow-complete.txt",
				"uses: actions/upload-artifact@v4",
				"name: workflow-complete",
				"path: workflow-complete.txt", 
				"if-no-files-found: ignore",
			},
			unexpectedSteps: []string{
				"Check if workflow-complete.txt exists", // Should not have shell check
				"id: check_file",                         // Should not have check step
				"if: steps.check_file.outputs.upload == 'true'", // Should not have conditional
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compiler := NewCompiler(false, "", "test")
			var yaml strings.Builder

			// Call the function we're testing
			compiler.generateWorkflowComplete(&yaml)

			result := yaml.String()

			// Check for expected content
			for _, expected := range tt.expectedSteps {
				if !strings.Contains(result, expected) {
					t.Errorf("Expected to find '%s' in generated YAML, but it was missing", expected)
				}
			}

			// Check for unexpected content (from the old implementation)
			for _, unexpected := range tt.unexpectedSteps {
				if strings.Contains(result, unexpected) {
					t.Errorf("Did not expect to find '%s' in generated YAML, but it was present", unexpected)
				}
			}

			// Verify the output has the correct structure
			lines := strings.Split(strings.TrimSpace(result), "\n")
			if len(lines) == 0 {
				t.Error("Generated YAML should not be empty")
				return
			}

			// Check that it starts with the step name
			if !strings.Contains(lines[0], "- name: Upload workflow-complete.txt") {
				t.Errorf("First line should be the step name, got: %s", lines[0])
			}
		})
	}
}