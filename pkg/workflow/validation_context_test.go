package workflow

import (
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewValidationContext(t *testing.T) {
	workflowData := &WorkflowData{
		Name: "test-workflow",
	}

	ctx := NewValidationContext("/path/to/workflow.md", workflowData)

	require.NotNil(t, ctx)
	assert.Equal(t, PhaseParseTime, ctx.GetPhase())
	assert.Equal(t, "/path/to/workflow.md", ctx.GetMarkdownPath())
	assert.Equal(t, workflowData, ctx.GetWorkflowData())
	assert.False(t, ctx.HasErrors())
	assert.False(t, ctx.HasWarnings())
	assert.Equal(t, 0, ctx.ErrorCount())
	assert.Equal(t, 0, ctx.WarningCount())
}

func TestValidationPhaseString(t *testing.T) {
	tests := []struct {
		phase    ValidationPhase
		expected string
	}{
		{PhaseParseTime, "ParseTime"},
		{PhasePreCompile, "PreCompile"},
		{PhasePostYAMLGeneration, "PostYAMLGeneration"},
		{PhasePreEmit, "PreEmit"},
		{ValidationPhase(999), "Unknown(999)"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := tt.phase.String()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSetAndGetPhase(t *testing.T) {
	ctx := NewValidationContext("test.md", &WorkflowData{})

	assert.Equal(t, PhaseParseTime, ctx.GetPhase())

	ctx.SetPhase(PhasePreCompile)
	assert.Equal(t, PhasePreCompile, ctx.GetPhase())

	ctx.SetPhase(PhasePostYAMLGeneration)
	assert.Equal(t, PhasePostYAMLGeneration, ctx.GetPhase())

	ctx.SetPhase(PhasePreEmit)
	assert.Equal(t, PhasePreEmit, ctx.GetPhase())
}

func TestAddError(t *testing.T) {
	ctx := NewValidationContext("test.md", &WorkflowData{})

	// Add single error
	ctx.AddError("validator1", errors.New("test error"))

	assert.True(t, ctx.HasErrors())
	assert.Equal(t, 1, ctx.ErrorCount())
	assert.False(t, ctx.HasWarnings())

	errors := ctx.GetErrors()
	require.Len(t, errors, 1)
	assert.Equal(t, "validator1", errors[0].Validator)
	assert.Equal(t, "test error", errors[0].Message)
	assert.Equal(t, "test.md", errors[0].File)
	assert.Equal(t, 1, errors[0].Line)
	assert.Equal(t, 1, errors[0].Column)
}

func TestAddMultipleErrors(t *testing.T) {
	ctx := NewValidationContext("test.md", &WorkflowData{})

	// Add multiple errors
	ctx.AddError("validator1", errors.New("error 1"))
	ctx.AddError("validator2", errors.New("error 2"))
	ctx.AddError("validator3", errors.New("error 3"))

	assert.True(t, ctx.HasErrors())
	assert.Equal(t, 3, ctx.ErrorCount())

	errors := ctx.GetErrors()
	require.Len(t, errors, 3)
	assert.Equal(t, "validator1", errors[0].Validator)
	assert.Equal(t, "validator2", errors[1].Validator)
	assert.Equal(t, "validator3", errors[2].Validator)
}

func TestAddErrorWithNil(t *testing.T) {
	ctx := NewValidationContext("test.md", &WorkflowData{})

	// Adding nil error should not add anything
	ctx.AddError("validator1", nil)

	assert.False(t, ctx.HasErrors())
	assert.Equal(t, 0, ctx.ErrorCount())
}

func TestAddErrorWithPosition(t *testing.T) {
	ctx := NewValidationContext("test.md", &WorkflowData{})

	ctx.AddErrorWithPosition("validator1", "test error", 42, 10)

	errors := ctx.GetErrors()
	require.Len(t, errors, 1)
	assert.Equal(t, "validator1", errors[0].Validator)
	assert.Equal(t, "test error", errors[0].Message)
	assert.Equal(t, 42, errors[0].Line)
	assert.Equal(t, 10, errors[0].Column)
}

func TestAddWarning(t *testing.T) {
	ctx := NewValidationContext("test.md", &WorkflowData{})

	ctx.AddWarning("validator1", "test warning")

	assert.False(t, ctx.HasErrors())
	assert.True(t, ctx.HasWarnings())
	assert.Equal(t, 1, ctx.WarningCount())

	warnings := ctx.GetWarnings()
	require.Len(t, warnings, 1)
	assert.Equal(t, "validator1", warnings[0].Validator)
	assert.Equal(t, "test warning", warnings[0].Message)
	assert.Equal(t, "test.md", warnings[0].File)
}

func TestAddMultipleWarnings(t *testing.T) {
	ctx := NewValidationContext("test.md", &WorkflowData{})

	ctx.AddWarning("validator1", "warning 1")
	ctx.AddWarning("validator2", "warning 2")

	assert.Equal(t, 2, ctx.WarningCount())

	warnings := ctx.GetWarnings()
	require.Len(t, warnings, 2)
	assert.Equal(t, "warning 1", warnings[0].Message)
	assert.Equal(t, "warning 2", warnings[1].Message)
}

func TestAddWarningWithPosition(t *testing.T) {
	ctx := NewValidationContext("test.md", &WorkflowData{})

	ctx.AddWarningWithPosition("validator1", "test warning", 15, 5)

	warnings := ctx.GetWarnings()
	require.Len(t, warnings, 1)
	assert.Equal(t, "test warning", warnings[0].Message)
	assert.Equal(t, 15, warnings[0].Line)
	assert.Equal(t, 5, warnings[0].Column)
}

func TestMixedErrorsAndWarnings(t *testing.T) {
	ctx := NewValidationContext("test.md", &WorkflowData{})

	ctx.AddError("validator1", errors.New("error 1"))
	ctx.AddWarning("validator2", "warning 1")
	ctx.AddError("validator3", errors.New("error 2"))
	ctx.AddWarning("validator4", "warning 2")

	assert.True(t, ctx.HasErrors())
	assert.True(t, ctx.HasWarnings())
	assert.Equal(t, 2, ctx.ErrorCount())
	assert.Equal(t, 2, ctx.WarningCount())
}

func TestFormatReportSingleError(t *testing.T) {
	ctx := NewValidationContext("test.md", &WorkflowData{})

	ctx.AddError("validator1", errors.New("single error"))

	report := ctx.FormatReport()

	assert.Contains(t, report, "error:")
	assert.Contains(t, report, "single error")
	assert.Contains(t, report, "test.md")
}

func TestFormatReportMultipleErrors(t *testing.T) {
	ctx := NewValidationContext("test.md", &WorkflowData{})

	ctx.AddError("validator1", errors.New("error 1"))
	ctx.AddError("validator2", errors.New("error 2"))
	ctx.AddError("validator3", errors.New("error 3"))

	report := ctx.FormatReport()

	assert.Contains(t, report, "Found 3 validation errors")
	assert.Contains(t, report, "error 1")
	assert.Contains(t, report, "error 2")
	assert.Contains(t, report, "error 3")
}

func TestFormatReportWarningsNotShownWithErrors(t *testing.T) {
	ctx := NewValidationContext("test.md", &WorkflowData{})

	ctx.AddError("validator1", errors.New("error 1"))
	ctx.AddWarning("validator2", "warning 1")

	report := ctx.FormatReport()

	// Warnings should not be shown when there are errors (unless verbose)
	assert.Contains(t, report, "error 1")
	assert.NotContains(t, report, "warning 1")
}

func TestFormatReportWarningsShownWithVerbose(t *testing.T) {
	ctx := NewValidationContext("test.md", &WorkflowData{})
	ctx.SetVerbose(true)

	ctx.AddError("validator1", errors.New("error 1"))
	ctx.AddWarning("validator2", "warning 1")

	report := ctx.FormatReport()

	// Warnings should be shown when verbose is enabled
	assert.Contains(t, report, "error 1")
	assert.Contains(t, report, "warning 1")
}

func TestFormatReportWarningsOnlyWithoutErrors(t *testing.T) {
	ctx := NewValidationContext("test.md", &WorkflowData{})

	ctx.AddWarning("validator1", "warning 1")
	ctx.AddWarning("validator2", "warning 2")

	report := ctx.FormatReport()

	// Warnings should be shown when there are no errors
	assert.Contains(t, report, "Found 2 validation warnings")
	assert.Contains(t, report, "warning 1")
	assert.Contains(t, report, "warning 2")
}

func TestErrorMethod(t *testing.T) {
	ctx := NewValidationContext("test.md", &WorkflowData{})

	// No errors - should return empty string
	assert.Equal(t, "", ctx.Error())

	// Add error
	ctx.AddError("validator1", errors.New("test error"))

	// Should return formatted report
	errorMsg := ctx.Error()
	assert.NotEmpty(t, errorMsg)
	assert.Contains(t, errorMsg, "test error")
}

func TestClear(t *testing.T) {
	ctx := NewValidationContext("test.md", &WorkflowData{})

	ctx.AddError("validator1", errors.New("error 1"))
	ctx.AddWarning("validator2", "warning 1")

	assert.True(t, ctx.HasErrors())
	assert.True(t, ctx.HasWarnings())

	ctx.Clear()

	assert.False(t, ctx.HasErrors())
	assert.False(t, ctx.HasWarnings())
	assert.Equal(t, 0, ctx.ErrorCount())
	assert.Equal(t, 0, ctx.WarningCount())
}

func TestSummary(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(*ValidationContext)
		expected string
	}{
		{
			name:     "no issues",
			setup:    func(ctx *ValidationContext) {},
			expected: "No validation issues found",
		},
		{
			name: "only errors",
			setup: func(ctx *ValidationContext) {
				ctx.AddError("validator1", errors.New("error 1"))
				ctx.AddError("validator2", errors.New("error 2"))
			},
			expected: "2 error(s)",
		},
		{
			name: "only warnings",
			setup: func(ctx *ValidationContext) {
				ctx.AddWarning("validator1", "warning 1")
			},
			expected: "1 warning(s)",
		},
		{
			name: "errors and warnings",
			setup: func(ctx *ValidationContext) {
				ctx.AddError("validator1", errors.New("error 1"))
				ctx.AddWarning("validator2", "warning 1")
				ctx.AddWarning("validator3", "warning 2")
			},
			expected: "1 error(s), 2 warning(s)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := NewValidationContext("test.md", &WorkflowData{})
			tt.setup(ctx)

			summary := ctx.Summary()
			assert.Equal(t, tt.expected, summary)
		})
	}
}

func TestSetAndGetYAMLContent(t *testing.T) {
	ctx := NewValidationContext("test.md", &WorkflowData{})

	yamlContent := "name: test\non: push\njobs:\n  test:\n    runs-on: ubuntu-latest"
	ctx.SetYAMLContent(yamlContent)

	assert.Equal(t, yamlContent, ctx.GetYAMLContent())
}

func TestVerboseFlag(t *testing.T) {
	ctx := NewValidationContext("test.md", &WorkflowData{})

	assert.False(t, ctx.IsVerbose())

	ctx.SetVerbose(true)
	assert.True(t, ctx.IsVerbose())

	ctx.SetVerbose(false)
	assert.False(t, ctx.IsVerbose())
}

func TestSkipExternalAPI(t *testing.T) {
	ctx := NewValidationContext("test.md", &WorkflowData{})

	assert.False(t, ctx.ShouldSkipExternalAPI())

	ctx.SetSkipExternalAPI(true)
	assert.True(t, ctx.ShouldSkipExternalAPI())

	ctx.SetSkipExternalAPI(false)
	assert.False(t, ctx.ShouldSkipExternalAPI())
}

func TestErrorAggregationScenario(t *testing.T) {
	// Simulate a realistic scenario with multiple validation errors
	ctx := NewValidationContext("workflow.md", &WorkflowData{})
	ctx.SetPhase(PhasePreCompile)

	// Multiple validators report errors
	ctx.AddError("strict_mode_validation", errors.New("strict mode: write permission 'contents: write' is not allowed"))
	ctx.AddError("features_validation", errors.New("invalid feature flag: unknown-feature"))
	ctx.AddError("sandbox_validation", errors.New("mount path must be absolute: ./relative/path"))

	// One validator reports a warning
	ctx.AddWarning("container_validation", "container image validation may fail due to auth issues")

	// Check results
	assert.True(t, ctx.HasErrors())
	assert.True(t, ctx.HasWarnings())
	assert.Equal(t, 3, ctx.ErrorCount())
	assert.Equal(t, 1, ctx.WarningCount())

	// Get formatted report
	report := ctx.FormatReport()
	assert.Contains(t, report, "Found 3 validation errors")
	assert.Contains(t, report, "strict mode")
	assert.Contains(t, report, "invalid feature flag")
	assert.Contains(t, report, "mount path")

	// Summary
	summary := ctx.Summary()
	assert.Equal(t, "3 error(s), 1 warning(s)", summary)
}

func TestPhaseProgression(t *testing.T) {
	ctx := NewValidationContext("test.md", &WorkflowData{})

	// Verify phase progression
	assert.Equal(t, PhaseParseTime, ctx.GetPhase())

	ctx.SetPhase(PhasePreCompile)
	ctx.AddError("validator1", errors.New("error in pre-compile"))

	ctx.SetPhase(PhasePostYAMLGeneration)
	ctx.AddError("validator2", errors.New("error in post-yaml"))

	ctx.SetPhase(PhasePreEmit)
	ctx.AddError("validator3", errors.New("error in pre-emit"))

	// All errors should be collected
	assert.Equal(t, 3, ctx.ErrorCount())
}

func TestErrorReportFormat(t *testing.T) {
	// Test that error report is IDE-parseable (file:line:column: error: message)
	ctx := NewValidationContext("workflow.md", &WorkflowData{})

	ctx.AddErrorWithPosition("validator1", "test error message", 5, 10)

	report := ctx.FormatReport()

	// Check for IDE-parseable format
	lines := strings.Split(report, "\n")
	require.Greater(t, len(lines), 0)

	// First line should contain file:line:column: error:
	assert.Contains(t, lines[0], "workflow.md:5:10:")
	assert.Contains(t, lines[0], "error:")
	assert.Contains(t, lines[0], "test error message")
}
