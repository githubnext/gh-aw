//go:build !integration

package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

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
