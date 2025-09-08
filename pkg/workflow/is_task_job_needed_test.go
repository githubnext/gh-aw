package workflow

import "testing"

func TestIsTaskJobNeeded(t *testing.T) {
	compiler := NewCompiler(false, "", "test")

	t.Run("no_conditions", func(t *testing.T) {
		data := &WorkflowData{
			For: []string{"all"}, // Explicitly disable permission checks
		}
		if compiler.isTaskJobNeeded(data) {
			t.Errorf("Expected isTaskJobNeeded to be false when no alias, no needsTextOutput, no If condition, and roles: all")
		}
	})

	t.Run("if_condition_present", func(t *testing.T) {
		data := &WorkflowData{If: "if: github.ref == 'refs/heads/main'"}
		if !compiler.isTaskJobNeeded(data) {
			t.Errorf("Expected isTaskJobNeeded to be true when If condition is specified")
		}
	})

	t.Run("default_permission_check", func(t *testing.T) {
		data := &WorkflowData{} // No explicit For field, should default to permission checks
		if !compiler.isTaskJobNeeded(data) {
			t.Errorf("Expected isTaskJobNeeded to be true when permission checks are needed by default")
		}
	})
}
