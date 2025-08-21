package workflow

import "testing"

func TestIsTaskJobNeeded(t *testing.T) {
	compiler := NewCompiler(false, "", "test")

	t.Run("no_conditions", func(t *testing.T) {
		data := &WorkflowData{}
		if compiler.isTaskJobNeeded(data) {
			t.Errorf("Expected isTaskJobNeeded to be false when no alias, no needsTextOutput, and no If condition")
		}
	})

	t.Run("if_condition_present", func(t *testing.T) {
		data := &WorkflowData{If: "if: github.ref == 'refs/heads/main'"}
		if !compiler.isTaskJobNeeded(data) {
			t.Errorf("Expected isTaskJobNeeded to be true when If condition is specified")
		}
	})
}
