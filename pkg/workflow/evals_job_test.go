package workflow

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/github/gh-aw/pkg/constants"
)

func TestBuildEvalsJobNeedsWithoutDetection(t *testing.T) {
	compiler := NewCompiler()

	data := &WorkflowData{
		AI: "copilot",
		Evals: &EvalsConfig{
			Questions: []EvalDefinition{
				{ID: "q1", Question: "Does it build?"},
			},
		},
		SafeOutputs: &SafeOutputsConfig{},
	}

	job, err := compiler.buildEvalsJob(data)
	require.NoError(t, err)
	require.NotNil(t, job)

	assert.ElementsMatch(t, []string{
		string(constants.AgentJobName),
		string(constants.ActivationJobName),
	}, job.Needs)
	assert.NotContains(t, job.Needs, string(constants.SafeOutputsJobName))
	assert.NotContains(t, job.Needs, string(constants.DetectionJobName))
	assert.Contains(t, job.If, "needs.agent.result")
	assert.NotContains(t, job.If, "needs.safe_outputs.result")
	assert.Equal(t, "${{ steps.parse-mcp-gateway.outputs.aic }}", job.Outputs["aic"])
	assert.Contains(t, strings.Join(job.Steps, ""), "id: parse-mcp-gateway\n")
}

func TestBuildEvalsJobNeedsWithDetection(t *testing.T) {
	compiler := NewCompiler()

	data := &WorkflowData{
		AI: "copilot",
		Evals: &EvalsConfig{
			Questions: []EvalDefinition{
				{ID: "q1", Question: "Does it build?"},
			},
		},
		SafeOutputs: &SafeOutputsConfig{
			ThreatDetection: &ThreatDetectionConfig{},
		},
	}

	job, err := compiler.buildEvalsJob(data)
	require.NoError(t, err)
	require.NotNil(t, job)

	assert.ElementsMatch(t, []string{
		string(constants.AgentJobName),
		string(constants.ActivationJobName),
		string(constants.DetectionJobName),
	}, job.Needs)
	assert.NotContains(t, job.Needs, string(constants.SafeOutputsJobName))
	assert.Contains(t, job.If, "needs.agent.result")
	assert.NotContains(t, job.If, "needs.safe_outputs.result")
	assert.Equal(t, "${{ steps.parse-mcp-gateway.outputs.aic }}", job.Outputs["aic"])
	assert.Contains(t, strings.Join(job.Steps, ""), "id: parse-mcp-gateway\n")
}
