//go:build !integration

package workflow

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/github/gh-aw/pkg/constants"
)

// ---------------------------------------------------------------------------
// buildEvalsJob — job not created when no evals
// ---------------------------------------------------------------------------

func TestBuildEvalsJob_NoEvals_ReturnsNil(t *testing.T) {
	c := NewCompiler()
	data := &WorkflowData{
		Name: "test-workflow",
	}

	job, err := c.buildEvalsJob(data)
	require.NoError(t, err)
	assert.Nil(t, job, "buildEvalsJob should return nil when no evals are configured")
}

func TestBuildEvalsJob_EmptyEvalsConfig_ReturnsNil(t *testing.T) {
	c := NewCompiler()
	data := &WorkflowData{
		Name:  "test-workflow",
		Evals: &EvalsConfig{Questions: []EvalDefinition{}},
	}

	job, err := c.buildEvalsJob(data)
	require.NoError(t, err)
	assert.Nil(t, job)
}

// ---------------------------------------------------------------------------
// buildEvalsJob — job structure
// ---------------------------------------------------------------------------

func TestBuildEvalsJob_JobName(t *testing.T) {
	c := NewCompiler()
	data := makeEvalsWorkflowData()

	job, err := c.buildEvalsJob(data)
	require.NoError(t, err)
	require.NotNil(t, job)
	assert.Equal(t, string(constants.EvalsJobName), job.Name)
}

func TestBuildEvalsJob_DependsOnAgentAndActivation(t *testing.T) {
	c := NewCompiler()
	data := makeEvalsWorkflowData()

	job, err := c.buildEvalsJob(data)
	require.NoError(t, err)
	require.NotNil(t, job)

	assert.Contains(t, job.Needs, string(constants.AgentJobName), "evals job must depend on agent")
	assert.Contains(t, job.Needs, string(constants.ActivationJobName), "evals job must depend on activation")
}

func TestBuildEvalsJob_DependsOnSafeOutputsWhenConfigured(t *testing.T) {
	c := NewCompiler()
	data := makeEvalsWorkflowData()
	data.SafeOutputs = &SafeOutputsConfig{}

	job, err := c.buildEvalsJob(data)
	require.NoError(t, err)
	require.NotNil(t, job)

	assert.Contains(t, job.Needs, string(constants.SafeOutputsJobName), "evals job must depend on safe_outputs when it is configured")
}

func TestBuildEvalsJob_DoesNotDependOnSafeOutputsWhenAbsent(t *testing.T) {
	c := NewCompiler()
	data := makeEvalsWorkflowData() // SafeOutputs is nil

	job, err := c.buildEvalsJob(data)
	require.NoError(t, err)
	require.NotNil(t, job)

	for _, need := range job.Needs {
		assert.NotEqual(t, string(constants.SafeOutputsJobName), need, "evals job must NOT depend on safe_outputs when it is not configured")
	}
}

func TestBuildEvalsJob_HasAlwaysCondition(t *testing.T) {
	c := NewCompiler()
	data := makeEvalsWorkflowData()

	job, err := c.buildEvalsJob(data)
	require.NoError(t, err)
	require.NotNil(t, job)

	assert.Contains(t, job.If, "always()", "evals job condition should include always()")
}

func TestBuildEvalsJob_HasContentsWritePermission(t *testing.T) {
	c := NewCompiler()
	data := makeEvalsWorkflowData()

	job, err := c.buildEvalsJob(data)
	require.NoError(t, err)
	require.NotNil(t, job)

	assert.Contains(t, job.Permissions, "contents: write", "evals job needs contents:write for git branch persistence")
}

func TestBuildEvalsJob_HasSteps(t *testing.T) {
	c := NewCompiler()
	data := makeEvalsWorkflowData()

	job, err := c.buildEvalsJob(data)
	require.NoError(t, err)
	require.NotNil(t, job)

	assert.NotEmpty(t, job.Steps, "evals job should have steps")
}

func TestBuildEvalsJob_StepsContainHarness(t *testing.T) {
	c := NewCompiler()
	data := makeEvalsWorkflowData()

	job, err := c.buildEvalsJob(data)
	require.NoError(t, err)
	require.NotNil(t, job)

	allSteps := strings.Join(job.Steps, "")
	assert.Contains(t, allSteps, "BinEval", "steps should reference BinEval harness")
}

func TestBuildEvalsJob_StepsContainArtifactUpload(t *testing.T) {
	c := NewCompiler()
	data := makeEvalsWorkflowData()

	job, err := c.buildEvalsJob(data)
	require.NoError(t, err)
	require.NotNil(t, job)

	allSteps := strings.Join(job.Steps, "")
	assert.Contains(t, allSteps, "upload-artifact", "steps should upload eval artifact")
}

func TestBuildEvalsJob_StepsContainBranchPersistence(t *testing.T) {
	c := NewCompiler()
	data := makeEvalsWorkflowData()

	job, err := c.buildEvalsJob(data)
	require.NoError(t, err)
	require.NotNil(t, job)

	allSteps := strings.Join(job.Steps, "")
	assert.Contains(t, allSteps, "evals/", "steps should persist to evals/ branch")
}

func TestBuildEvalsJob_DefaultRunsOn(t *testing.T) {
	c := NewCompiler()
	data := makeEvalsWorkflowData() // no RunsOn override

	job, err := c.buildEvalsJob(data)
	require.NoError(t, err)
	require.NotNil(t, job)

	assert.Contains(t, job.RunsOn, "ubuntu-latest")
}

func TestBuildEvalsJob_CustomRunsOn(t *testing.T) {
	c := NewCompiler()
	data := makeEvalsWorkflowData()
	data.Evals.RunsOn = "ubuntu-latest"

	job, err := c.buildEvalsJob(data)
	require.NoError(t, err)
	require.NotNil(t, job)

	// RunsOn should still reference ubuntu-latest
	assert.Contains(t, job.RunsOn, "ubuntu-latest")
}

// ---------------------------------------------------------------------------
// deduplicateStrings
// ---------------------------------------------------------------------------

func TestDeduplicateStrings(t *testing.T) {
	input := []string{"agent", "activation", "agent", "safe_outputs", "activation"}
	got := deduplicateStrings(input)
	assert.Equal(t, []string{"agent", "activation", "safe_outputs"}, got)
}

func TestDeduplicateStrings_EmptyInput(t *testing.T) {
	assert.Empty(t, deduplicateStrings(nil))
	assert.Empty(t, deduplicateStrings([]string{}))
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func makeEvalsWorkflowData() *WorkflowData {
	return &WorkflowData{
		Name:       "test-workflow",
		WorkflowID: "test-workflow-123",
		Evals: &EvalsConfig{
			Questions: []EvalDefinition{
				{ID: "builds", Question: "Does the code compile?"},
				{ID: "tests", Question: "Do all tests pass?"},
			},
		},
	}
}
