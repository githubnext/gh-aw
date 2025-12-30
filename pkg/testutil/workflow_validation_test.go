package testutil

import (
	"testing"
)

// TestValidateWorkflowExecution tests that the workflow validation function
// works correctly. This is a basic test to validate the spec-kit workflow.
func TestValidateWorkflowExecution(t *testing.T) {
	result := ValidateWorkflowExecution()
	
	if !result {
		t.Error("ValidateWorkflowExecution() = false, want true")
	}
}

// TestValidateWorkflowExecutionAlwaysSucceeds ensures the function consistently
// returns true when called multiple times.
func TestValidateWorkflowExecutionAlwaysSucceeds(t *testing.T) {
	for i := 0; i < 5; i++ {
		result := ValidateWorkflowExecution()
		if !result {
			t.Errorf("ValidateWorkflowExecution() call %d = false, want true", i+1)
		}
	}
}
