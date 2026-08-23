//go:build !integration

package workflow

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/testutil"
	"github.com/goccy/go-yaml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveAgentJobTimeoutValue(t *testing.T) {
	t.Run("defaults to the built-in positive integer", func(t *testing.T) {
		assert.Equal(t, "60", resolveAgentJobTimeoutValue(&WorkflowData{}))
	})

	t.Run("ignores the agentic step timeout default", func(t *testing.T) {
		data := &WorkflowData{TimeoutMinutes: "timeout-minutes: ${{ fromJSON(vars.GH_AW_DEFAULT_TIMEOUT_MINUTES || '20') }}"}
		assert.Equal(t, "60", resolveAgentJobTimeoutValue(data))
	})

	t.Run("keeps the job default when the step timeout is shorter", func(t *testing.T) {
		data := &WorkflowData{TimeoutMinutes: "timeout-minutes: 30"}
		assert.Equal(t, "60", resolveAgentJobTimeoutValue(data))
	})

	t.Run("raises the job default to cover a longer step timeout", func(t *testing.T) {
		data := &WorkflowData{TimeoutMinutes: "timeout-minutes: 120"}
		assert.Equal(t, "120", resolveAgentJobTimeoutValue(data))
	})

	t.Run("uses jobs.agent.timeout-minutes when configured", func(t *testing.T) {
		data := &WorkflowData{
			TimeoutMinutes: "timeout-minutes: 120",
			Jobs: map[string]any{
				string(constants.AgentJobName): map[string]any{"timeout-minutes": 90},
			},
		}
		assert.Equal(t, "90", resolveAgentJobTimeoutValue(data))
	})

	t.Run("ignores a jobs.agent.timeout-minutes expression override", func(t *testing.T) {
		data := &WorkflowData{
			Jobs: map[string]any{
				string(constants.AgentJobName): map[string]any{"timeout-minutes": "${{ inputs.agent-timeout }}"},
			},
		}
		assert.Equal(t, "60", resolveAgentJobTimeoutValue(data))
	})
}

func TestResolveDetectionJobTimeoutValue(t *testing.T) {
	t.Run("defaults to the built-in positive integer", func(t *testing.T) {
		assert.Equal(t, "10", resolveDetectionJobTimeoutValue(&WorkflowData{}))
	})

	t.Run("is independent from jobs.agent.timeout-minutes", func(t *testing.T) {
		data := &WorkflowData{
			TimeoutMinutes: "timeout-minutes: 120",
			Jobs: map[string]any{
				string(constants.AgentJobName): map[string]any{"timeout-minutes": 90},
			},
		}
		assert.Equal(t, "10", resolveDetectionJobTimeoutValue(data))
	})

	t.Run("uses jobs.detection.timeout-minutes when configured", func(t *testing.T) {
		data := &WorkflowData{
			Jobs: map[string]any{
				string(constants.DetectionJobName): map[string]any{"timeout-minutes": 25},
			},
		}
		assert.Equal(t, "25", resolveDetectionJobTimeoutValue(data))
		assert.Equal(t, "25", resolveStepTimeoutValue(&WorkflowData{IsDetectionRun: true, Jobs: data.Jobs}))
	})
}

func TestResolveStepTimeoutValueIgnoresAgentJobOverride(t *testing.T) {
	data := &WorkflowData{
		TimeoutMinutes: "timeout-minutes: 30",
		Jobs: map[string]any{
			string(constants.AgentJobName): map[string]any{"timeout-minutes": 90},
		},
	}
	assert.Equal(t, "30", resolveStepTimeoutValue(data))
}

// TestGeneratedAgentJobTimeoutMinutes verifies that the generated agent job is
// bounded by its own timeout, independently from the agentic step timeout.
func TestGeneratedAgentJobTimeoutMinutes(t *testing.T) {
	tests := []struct {
		name            string
		frontmatter     string
		wantJobTimeout  any
		wantStepTimeout any
	}{
		{
			name:            "defaults",
			frontmatter:     "",
			wantJobTimeout:  uint64(60),
			wantStepTimeout: "${{ fromJSON(vars.GH_AW_DEFAULT_TIMEOUT_MINUTES || '20') }}",
		},
		{
			name:            "top-level timeout bounds the step only",
			frontmatter:     "timeout-minutes: 45\n",
			wantJobTimeout:  uint64(60),
			wantStepTimeout: uint64(45),
		},
		{
			name:            "jobs.agent.timeout-minutes bounds the job only",
			frontmatter:     "jobs:\n  agent:\n    timeout-minutes: 90\n",
			wantJobTimeout:  uint64(90),
			wantStepTimeout: "${{ fromJSON(vars.GH_AW_DEFAULT_TIMEOUT_MINUTES || '20') }}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := testutil.TempDir(t, "agent-job-timeout")
			workflowPath := filepath.Join(tmpDir, "agent-job-timeout.md")
			workflow := "---\non: workflow_dispatch\npermissions:\n  contents: read\nengine: copilot\n" + tt.frontmatter + "---\n\n# Test workflow\n"
			require.NoError(t, os.WriteFile(workflowPath, []byte(workflow), 0o644))
			require.NoError(t, NewCompiler().CompileWorkflow(workflowPath))

			lockContent, err := os.ReadFile(filepath.Join(tmpDir, "agent-job-timeout.lock.yml"))
			require.NoError(t, err)

			var lock map[string]any
			require.NoError(t, yaml.Unmarshal(lockContent, &lock))
			jobs, ok := lock["jobs"].(map[string]any)
			require.True(t, ok)
			agent, ok := jobs[string(constants.AgentJobName)].(map[string]any)
			require.True(t, ok)
			assert.Equal(t, tt.wantJobTimeout, agent["timeout-minutes"])

			steps, ok := agent["steps"].([]any)
			require.True(t, ok)
			for _, rawStep := range steps {
				step, ok := rawStep.(map[string]any)
				if !ok || step["id"] != "agentic_execution" {
					continue
				}
				assert.Equal(t, tt.wantStepTimeout, step["timeout-minutes"])
				return
			}
			t.Fatal("agentic_execution step not found")
		})
	}
}

func TestGeneratedAgentJobTimeoutMinutes_ImportedFromSharedWorkflow(t *testing.T) {
	tmpDir := testutil.TempDir(t, "agent-job-timeout-imported")
	sharedPath := filepath.Join(tmpDir, "shared-timeout.md")
	sharedWorkflow := `---
jobs:
  agent:
    timeout-minutes: 75
---
`
	require.NoError(t, os.WriteFile(sharedPath, []byte(sharedWorkflow), 0o644))

	mainWorkflowPath := filepath.Join(tmpDir, "agent-job-timeout-imported.md")
	mainWorkflow := `---
on: workflow_dispatch
permissions:
  contents: read
engine: copilot
imports:
  - ./shared-timeout.md
---

# Test workflow
`
	require.NoError(t, os.WriteFile(mainWorkflowPath, []byte(mainWorkflow), 0o644))
	require.NoError(t, NewCompiler().CompileWorkflow(mainWorkflowPath))

	lockContent, err := os.ReadFile(filepath.Join(tmpDir, "agent-job-timeout-imported.lock.yml"))
	require.NoError(t, err)

	var lock map[string]any
	require.NoError(t, yaml.Unmarshal(lockContent, &lock))
	jobs, ok := lock["jobs"].(map[string]any)
	require.True(t, ok)
	agent, ok := jobs[string(constants.AgentJobName)].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, uint64(75), agent["timeout-minutes"])
}
