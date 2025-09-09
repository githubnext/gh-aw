package workflow

import "testing"

func TestIsTaskJobNeeded(t *testing.T) {
	compiler := NewCompiler(false, "", "test")

	t.Run("no_conditions", func(t *testing.T) {
		data := &WorkflowData{
			Roles: []string{"all"}, // Explicitly disable permission checks
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
		data := &WorkflowData{} // No explicit Roles field, permission checks now integrated into individual jobs
		if compiler.isTaskJobNeeded(data) {
			t.Errorf("Expected isTaskJobNeeded to be false - permission checks are now integrated into individual jobs, not task job")
		}
	})
}
