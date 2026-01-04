package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestLogsCommandAgentTaskFlag tests the agent-task flag
func TestLogsCommandAgentTaskFlag(t *testing.T) {
	cmd := NewLogsCommand()
	flags := cmd.Flags()

	// Check agent-task flag exists
	agentTaskFlag := flags.Lookup("agent-task")
	require.NotNil(t, agentTaskFlag, "Should have 'agent-task' flag")
	assert.Equal(t, "bool", agentTaskFlag.Value.Type(), "agent-task should be boolean type")
	assert.Equal(t, "false", agentTaskFlag.DefValue, "agent-task should default to false")
}

// TestLogsCommandAgentTaskFlagInHelp tests that agent-task flag is documented
func TestLogsCommandAgentTaskFlagInHelp(t *testing.T) {
	cmd := NewLogsCommand()

	// Check that help text mentions agent-task
	assert.Contains(t, cmd.Long, "--agent-task", "Help text should mention --agent-task flag")
}

// TestListWorkflowRunsFilteringWithAgentTaskFalse tests that when agentTask=false,
// only agentic workflows (those with .lock.yml files) are included
func TestListWorkflowRunsFilteringWithAgentTaskFalse(t *testing.T) {
	// Note: This test is primarily documentation of expected behavior
	// since listWorkflowRunsWithPagination requires GitHub CLI authentication

	// When agentTask=false (default), the function should:
	// 1. Call getAgenticWorkflowNames() to get list of workflows from .lock.yml files
	// 2. Filter runs to only include those workflows
	// 3. Exclude any workflows not in that list (e.g., "Agent Task" workflows from gh agent task)

	// This is the default behavior to filter out gh agent task runs
	t.Log("When agentTask=false, only workflows with .lock.yml files should be included")
}

// TestListWorkflowRunsFilteringWithAgentTaskTrue tests that when agentTask=true,
// all workflow runs are included (both agentic and agent task)
func TestListWorkflowRunsFilteringWithAgentTaskTrue(t *testing.T) {
	// Note: This test is primarily documentation of expected behavior
	// since listWorkflowRunsWithPagination requires GitHub CLI authentication

	// When agentTask=true, the function should:
	// 1. Skip the filtering step
	// 2. Return all workflow runs from GitHub API
	// 3. Include both agentic workflows AND agent task workflows

	// This allows users to see gh agent task runs in addition to gh aw workflows
	t.Log("When agentTask=true, all workflow runs should be included (agentic + agent task)")
}

// TestListWorkflowRunsWithSpecificWorkflowName tests that when a specific workflow
// name is provided, the agentTask flag is ignored
func TestListWorkflowRunsWithSpecificWorkflowName(t *testing.T) {
	// Note: This test is primarily documentation of expected behavior
	// since listWorkflowRunsWithPagination requires GitHub CLI authentication

	// When a specific workflow name is provided:
	// 1. The GitHub API filters to that workflow
	// 2. The agentTask flag should not matter
	// 3. All runs for that specific workflow are returned

	// This allows users to target a specific workflow regardless of its type
	t.Log("When a specific workflow name is provided, agentTask flag is ignored")
}
