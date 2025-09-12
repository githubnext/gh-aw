package workflow

import "testing"

func TestIsTaskJobNeeded(t *testing.T) {
	compiler := NewCompiler(false, "", "test")

	t.Run("no_conditions", func(t *testing.T) {
		data := &WorkflowData{
			Roles: []string{"all"}, // Explicitly disable permission checks
		}
		// Pass false for needsPermissionCheck since roles: all is specified
		if compiler.isTaskJobNeeded(data, false) {
			t.Errorf("Expected isTaskJobNeeded to be false when no alias, no needsTextOutput, no If condition, and roles: all")
		}
	})

	t.Run("if_condition_present", func(t *testing.T) {
		data := &WorkflowData{If: "if: github.ref == 'refs/heads/main'"}
		// Pass false for needsPermissionCheck, but should still return true due to If condition
		if !compiler.isTaskJobNeeded(data, false) {
			t.Errorf("Expected isTaskJobNeeded to be true when If condition is specified")
		}
	})

	t.Run("default_permission_check", func(t *testing.T) {
		data := &WorkflowData{} // No explicit Roles field, permission checks are consolidated in task job
		// Pass true for needsPermissionCheck to simulate permission checks being needed
		if !compiler.isTaskJobNeeded(data, true) {
			t.Errorf("Expected isTaskJobNeeded to be true - permission checks are now consolidated in task job")
		}
	})

	t.Run("permission_check_not_needed", func(t *testing.T) {
		data := &WorkflowData{} // No other conditions that would require task job
		// Pass false for needsPermissionCheck - task job should not be needed
		if compiler.isTaskJobNeeded(data, false) {
			t.Errorf("Expected isTaskJobNeeded to be false when no conditions require task job")
		}
	})
}
