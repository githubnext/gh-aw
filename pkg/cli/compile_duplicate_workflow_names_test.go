//go:build !integration

package cli

import (
	"testing"

	"github.com/github/gh-aw/pkg/workflow"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAppendDuplicateWorkflowNameWarnings(t *testing.T) {
	workflowDataList := []*workflow.WorkflowData{
		{Name: "Duplicate", WorkflowID: "first"},
		{Name: "Unique", WorkflowID: "unique"},
		{Name: "Duplicate", WorkflowID: "second"},
	}
	validationResults := []ValidationResult{
		{Workflow: "first.md"},
		{Workflow: "unique.md"},
		{Workflow: "second.md"},
	}

	warnings := appendDuplicateWorkflowNameWarnings(workflowDataList, &validationResults)

	require.Len(t, warnings, 1)
	assert.Equal(t, "duplicate_workflow_name", warnings[0].Type)
	assert.Contains(t, warnings[0].Message, `"Duplicate"`)
	assert.Contains(t, warnings[0].Message, "first, second")
	assert.Len(t, validationResults[0].Warnings, 1)
	assert.Empty(t, validationResults[1].Warnings)
	assert.Len(t, validationResults[2].Warnings, 1)
}
