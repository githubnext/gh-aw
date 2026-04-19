//go:build !integration

package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIssueMonsterAssignToAgentUsesIssueNumberField(t *testing.T) {
	workflowPath := filepath.Join("..", "..", ".github", "workflows", "issue-monster.md")

	content, err := os.ReadFile(workflowPath)
	require.NoError(t, err, "Issue Monster workflow should be readable")

	workflowText := string(content)
	assert.Contains(t, workflowText, "safeoutputs/assign_to_agent(issue_number=<issue_number>, agent=\"copilot\")", "Issue Monster should instruct assign_to_agent with issue_number")
	assert.NotContains(t, workflowText, "safeoutputs/assign_to_agent(issue-number=<issue_number>, agent=\"copilot\")", "Issue Monster should not use issue-number for assign_to_agent")
	assert.NotContains(t, workflowText, "Issue Monster has assigned this to Copilot!", "Issue Monster comment should avoid claiming assignment already succeeded")
}
