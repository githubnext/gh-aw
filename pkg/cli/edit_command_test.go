package cli

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEditChangesSupportAssignmentsSchedulesAndImports(t *testing.T) {
	t.Parallel()
	cmd := NewEditCommand()
	require.NoError(t, cmd.Flags().Set("set", "max-turns=20"))
	require.NoError(t, cmd.Flags().Set("schedule", "6h"))
	require.NoError(t, cmd.Flags().Set("add-import", "shared/common.md"))

	changes, err := editChangesFromCommand(cmd, []string{"workflow", "model: small"})
	require.NoError(t, err)
	frontmatter := map[string]any{"on": "workflow_dispatch"}
	for _, change := range changes {
		require.NoError(t, applyEditChange(frontmatter, change))
	}

	assert.Equal(t, "small", frontmatter["model"])
	assert.Equal(t, uint64(20), frontmatter["max-turns"])
	assert.Equal(t, []any{"shared/common.md"}, frontmatter["imports"])
	assert.Equal(t, map[string]any{
		"workflow_dispatch": nil,
		"schedule":          []any{map[string]any{"cron": "FUZZY:HOURLY/6 * * *"}},
	}, frontmatter["on"])
}

func TestEditCommandDryRunPreservesWorkflowFile(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	workflowPath := dir + "/workflow.md"
	original := "---\non: workflow_dispatch\n---\n# Workflow\n"
	require.NoError(t, os.WriteFile(workflowPath, []byte(original), 0o644))

	cmd := NewEditCommand()
	cmd.SetArgs([]string{workflowPath, "max-turns: 20", "--dry-run"})
	var output strings.Builder
	cmd.SetOut(&output)
	require.NoError(t, cmd.Execute())

	assert.Contains(t, output.String(), "max-turns: 20")
	content, err := os.ReadFile(workflowPath)
	require.NoError(t, err)
	assert.Equal(t, original, string(content))
}

func TestEditChangesAddImportsToObjectForm(t *testing.T) {
	t.Parallel()
	frontmatter := map[string]any{"imports": map[string]any{"aw": []any{"shared/base.md"}}}
	require.NoError(t, applyEditChange(frontmatter, editChange{
		kind: "add", path: "imports", value: "shared/extra.md",
	}))
	assert.Equal(t, []any{"shared/base.md", "shared/extra.md"}, frontmatter["imports"].(map[string]any)["aw"])
}

func TestEditAssignmentParsesScheduleShorthands(t *testing.T) {
	t.Parallel()
	change, err := parseEditAssignment("on.schedule: weekdays", ":")
	require.NoError(t, err)
	frontmatter := map[string]any{"on": "workflow_dispatch"}
	require.NoError(t, applyEditChange(frontmatter, change))
	assert.Equal(t, []any{map[string]any{"cron": "FUZZY:DAILY_WEEKDAYS * * *"}}, frontmatter["on"].(map[string]any)["schedule"])
}

func TestReplaceFrontmatterPreservesBodySeparators(t *testing.T) {
	t.Parallel()
	content := "---\non: workflow_dispatch\n---\n# Workflow\n\n---\nBody\n"
	updated, err := replaceFrontmatter(content, map[string]any{"on": "push"})
	require.NoError(t, err)
	assert.Contains(t, updated, "---\n# Workflow\n\n---\nBody\n")
}

func TestEditCommandRejectsSourceManagedWorkflow(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	workflowPath := dir + "/workflow.md"
	require.NoError(t, os.WriteFile(workflowPath, []byte("---\nsource: owner/repo@v1\non: workflow_dispatch\n---\n"), 0o644))

	cmd := NewEditCommand()
	cmd.SetArgs([]string{workflowPath, "max-turns: 20"})
	assert.ErrorContains(t, cmd.Execute(), "source-managed")
}
