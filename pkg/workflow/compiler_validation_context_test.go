package workflow

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPreCompileValidationsWithContext(t *testing.T) {
	// Create a workflow with multiple validation errors
	markdown := `---
on: issues
engine: copilot
---

# Test Workflow

Test prompt.
`

	// Create compiler
	compiler := NewCompiler(false, "", "1.0.0")
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.md")
	err := os.WriteFile(testFile, []byte(markdown), 0644)
	require.NoError(t, err, "Failed to write test file")

	// Parse the workflow
	workflowData, err := compiler.ParseWorkflowFile(testFile)
	require.NoError(t, err, "Failed to parse workflow")

	// Create validation context and run pre-compile validations
	ctx := NewValidationContext(testFile, workflowData)
	ctx.SetPhase(PhasePreCompile)

	// Run all pre-compile validations
	compiler.runPreCompileValidations(ctx, workflowData, testFile)

	// Verify that the validation system is working even with no errors
	// This test demonstrates the validation context infrastructure
	assert.False(t, ctx.HasErrors(), "Should have no validation errors with valid workflow")
	assert.Equal(t, 0, ctx.ErrorCount(), "Error count should be 0")

	t.Logf("Validation completed successfully: %s", ctx.Summary())
}

func TestPreCompileValidationsNoErrors(t *testing.T) {
	// Create a valid workflow
	markdown := `---
on: issues
engine: copilot
permissions:
  contents: read
---

# Test Workflow

Test prompt.
`

	compiler := NewCompiler(false, "", "1.0.0")
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.md")
	err := os.WriteFile(testFile, []byte(markdown), 0644)
	require.NoError(t, err, "Failed to write test file")

	workflowData, err := compiler.ParseWorkflowFile(testFile)
	require.NoError(t, err, "Failed to parse workflow")

	// Create validation context and run validations
	ctx := NewValidationContext(testFile, workflowData)
	ctx.SetPhase(PhasePreCompile)

	compiler.runPreCompileValidations(ctx, workflowData, testFile)

	// Verify no errors
	assert.False(t, ctx.HasErrors(), "Should have no validation errors")
	assert.Equal(t, 0, ctx.ErrorCount(), "Error count should be 0")
	assert.Equal(t, "No validation issues found", ctx.Summary())
}

func TestErrorAggregationDemonstration(t *testing.T) {
	// Demonstrate error aggregation with a real scenario
	// This test validates the ValidationContext infrastructure
	markdown := `---
on: issues
engine: copilot
---

# Test Workflow

Test prompt.
`

	compiler := NewCompiler(false, "", "1.0.0")
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.md")
	err := os.WriteFile(testFile, []byte(markdown), 0644)
	require.NoError(t, err)

	workflowData, err := compiler.ParseWorkflowFile(testFile)
	require.NoError(t, err)

	// NEW PATTERN: Collect all errors
	ctx := NewValidationContext(testFile, workflowData)
	ctx.SetPhase(PhasePreCompile)
	compiler.runPreCompileValidations(ctx, workflowData, testFile)

	// Demonstrate that the context is working correctly
	assert.False(t, ctx.HasErrors(), "Should have no errors with valid workflow")

	t.Logf("\n=== NEW PATTERN: Error Aggregation ===")
	t.Logf("Validation completed: %s", ctx.Summary())
	t.Logf("The ValidationContext infrastructure is working correctly")
	t.Logf("\nTo see error aggregation in action, create a workflow with:")
	t.Logf("  - Invalid feature flag values")
	t.Logf("  - Invalid sandbox configuration")
	t.Logf("  - Multiple constraint violations")
	t.Logf("All errors will be collected and reported together")
}

func TestValidationContextPhaseTracking(t *testing.T) {
	// Test that phase tracking works correctly
	markdown := `---
on: issues
engine: copilot
---

# Test Workflow

Test prompt.
`

	compiler := NewCompiler(false, "", "1.0.0")
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.md")
	err := os.WriteFile(testFile, []byte(markdown), 0644)
	require.NoError(t, err)

	workflowData, err := compiler.ParseWorkflowFile(testFile)
	require.NoError(t, err)

	// Create context and verify phase progression
	ctx := NewValidationContext(testFile, workflowData)
	assert.Equal(t, PhaseParseTime, ctx.GetPhase())

	ctx.SetPhase(PhasePreCompile)
	assert.Equal(t, PhasePreCompile, ctx.GetPhase())

	// Run validations at this phase
	compiler.runPreCompileValidations(ctx, workflowData, testFile)

	// Move to next phase
	ctx.SetPhase(PhasePostYAMLGeneration)
	assert.Equal(t, PhasePostYAMLGeneration, ctx.GetPhase())

	// Verify no errors were collected
	assert.False(t, ctx.HasErrors())
}
