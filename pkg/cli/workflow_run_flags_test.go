//go:build !integration

package cli

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWorkflowRunFlagsJSON(t *testing.T) {
	const sharedJSON = `"Verbose":true,"EngineOverride":"claude","RepoOverride":"owner/repo","RefOverride":"feature","AutoMergePRs":true,"Push":true,"DryRun":true,"Approve":true`
	flags := WorkflowRunFlags{
		Verbose:        true,
		EngineOverride: "claude",
		RepoOverride:   "owner/repo",
		RefOverride:    "feature",
		AutoMergePRs:   true,
		Push:           true,
		DryRun:         true,
		Approve:        true,
	}

	t.Run("non-interactive", func(t *testing.T) {
		opts := RunOptions{
			WorkflowRunFlags:  flags,
			Enable:            true,
			WaitForCompletion: true,
			RepeatCount:       3,
			Inputs:            []string{"name=value"},
			JSON:              true,
		}
		data, err := json.Marshal(opts)
		require.NoError(t, err)
		assert.JSONEq(t, `{`+sharedJSON+`,"Enable":true,"WaitForCompletion":true,"RepeatCount":3,"Inputs":["name=value"],"JSON":true}`, string(data))

		var decoded RunOptions
		require.NoError(t, json.Unmarshal(data, &decoded))
		assert.Equal(t, opts, decoded)
	})

	t.Run("interactive", func(t *testing.T) {
		opts := RunWorkflowOptions{
			WorkflowRunFlags: flags,
			WorkflowName:     "test-workflow",
		}
		data, err := json.Marshal(opts)
		require.NoError(t, err)
		assert.JSONEq(t, `{`+sharedJSON+`,"WorkflowName":"test-workflow"}`, string(data))

		var decoded RunWorkflowOptions
		require.NoError(t, json.Unmarshal(data, &decoded))
		assert.Equal(t, opts, decoded)
	})
}

func TestWorkflowRunFlagsDefaults(t *testing.T) {
	var run RunOptions
	var interactive RunWorkflowOptions
	assert.Equal(t, WorkflowRunFlags{}, run.WorkflowRunFlags)
	assert.Equal(t, WorkflowRunFlags{}, interactive.WorkflowRunFlags)

	run.Approve = true
	interactive.RefOverride = "feature"
	assert.Equal(t, WorkflowRunFlags{Approve: true}, run.WorkflowRunFlags)
	assert.Equal(t, WorkflowRunFlags{RefOverride: "feature"}, interactive.WorkflowRunFlags)
	assert.False(t, interactive.Approve)
	assert.Empty(t, run.RefOverride)
}

func TestWorkflowRunFlagsDispatchArgs(t *testing.T) {
	for _, ref := range []string{"", "feature"} {
		t.Run("ref="+ref, func(t *testing.T) {
			opts := RunOptions{
				WorkflowRunFlags: WorkflowRunFlags{
					RepoOverride: "owner/repo",
					RefOverride:  ref,
					DryRun:       true,
				},
				Inputs: []string{"name=value", "message=hello world"},
			}
			expected := []string{"workflow", "run", "test.lock.yml", "--repo", "owner/repo"}
			if ref != "" {
				expected = append(expected, "--ref", ref)
			}
			expected = append(expected, "-f", "name=value", "-f", "message=hello world")

			args, resolvedRef := buildWorkflowRunArgs("test.lock.yml", opts)
			assert.Equal(t, expected, args)
			assert.Equal(t, ref, resolvedRef)
		})
	}
}
