//go:build !integration

package cli

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAddedWorkflowNameRejectsTraversal(t *testing.T) {
	t.Parallel()
	for _, name := range []string{"../../../../../tmp/pwned", "../escape", "nested/workflow", `nested\workflow`, "/tmp/escape", `C:\tmp\escape`, "..", ".", "workflow..name", "bad\x00name"} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			require.Error(t, validateAddedWorkflowName(name))
			_, err := AddWorkflows(context.Background(), []string{"not-a-source"}, AddOptions{Name: name})
			require.ErrorContains(t, err, "invalid workflow name")
			_, err = AddResolvedWorkflows(context.Background(), nil, nil, AddOptions{Name: name})
			require.ErrorContains(t, err, "invalid workflow name")
			base := t.TempDir()
			skip, err := validateWorkflowDestination(base, name, "", AddOptions{Force: true})
			require.Error(t, err)
			assert.False(t, skip)
			err = addActionWorkflowWithTracking(&ResolvedWorkflow{Content: []byte("payload")}, nil, AddOptions{Force: true}, base, name)
			require.Error(t, err)
			entries, err := os.ReadDir(base)
			require.NoError(t, err)
			assert.Empty(t, entries)
		})
	}
	for _, name := range []string{"workflow", "my-workflow_2", "release.v1", "Weekly Report"} {
		assert.NoError(t, validateAddedWorkflowName(name))
	}
}

func TestMCPAddRejectsTraversalNameBeforeExecution(t *testing.T) {
	t.Parallel()
	for _, name := range []string{"../../../../../tmp/pwned", `..\escape`, "nested/workflow"} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			executed := false
			execCmd := mockCommandWithOutput("", "")
			server := mcp.NewServer(&mcp.Implementation{Name: "name-test", Version: "test"}, nil)
			registerAddTool(server, func(ctx context.Context, args ...string) *exec.Cmd {
				executed = true
				return execCmd(ctx, args...)
			})
			result, err := connectInMemory(t, server).CallTool(context.Background(), &mcp.CallToolParams{
				Name: "add", Arguments: map[string]any{"workflows": []string{"https://example.com/payload.md"}, "name": name},
			})
			if err != nil {
				require.ErrorContains(t, err, "invalid workflow name")
			} else {
				require.True(t, result.IsError)
				assert.Contains(t, extractTextResult(t, result), "invalid workflow name")
			}
			assert.False(t, executed)
		})
	}
}

func TestAddedWorkflowDestinationRejectsSymlinkEscape(t *testing.T) {
	t.Parallel()
	for _, extension := range []string{".md", ".lock.yml", ".yml"} {
		t.Run(extension, func(t *testing.T) {
			t.Parallel()
			base, outside := t.TempDir(), t.TempDir()
			target := filepath.Join(outside, "untouched")
			require.NoError(t, os.WriteFile(target, []byte("original"), 0o600))
			require.NoError(t, os.Symlink(target, filepath.Join(base, "workflow"+extension)))
			if extension == ".yml" {
				require.Error(t, addActionWorkflowWithTracking(&ResolvedWorkflow{Content: []byte("payload")}, nil, AddOptions{Force: true}, base, "workflow"))
			} else {
				_, err := validateWorkflowDestination(base, "workflow", "", AddOptions{Force: true})
				require.Error(t, err)
			}
			content, err := os.ReadFile(target)
			require.NoError(t, err)
			assert.Equal(t, "original", string(content))
		})
	}
}

func TestAddedWorkflowDestinationPreservesValidNames(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	for _, name := range []string{"workflow", "my-workflow_2", "release.v1"} {
		skip, err := validateWorkflowDestination(base, name, "", AddOptions{})
		require.NoError(t, err)
		assert.False(t, skip)
		require.NoError(t, addActionWorkflowWithTracking(&ResolvedWorkflow{Content: []byte("name: Test\non: push\njobs: {}\n")}, nil, AddOptions{Quiet: true}, base, name))
		assert.FileExists(t, filepath.Join(base, name+".yml"))
	}

}

func TestAddedWorkflowDestinationRejectsDanglingSymlink(t *testing.T) {
	t.Parallel()
	for _, extension := range []string{".md", ".lock.yml", ".yml"} {
		t.Run(extension, func(t *testing.T) {
			t.Parallel()
			base, outside := t.TempDir(), t.TempDir()
			target := filepath.Join(outside, "must-not-be-created")
			require.NoError(t, os.Symlink(target, filepath.Join(base, "workflow"+extension)))
			if extension == ".yml" {
				require.Error(t, addActionWorkflowWithTracking(&ResolvedWorkflow{Content: []byte("payload")}, nil, AddOptions{Force: true}, base, "workflow"))
			} else {
				_, err := validateWorkflowDestination(base, "workflow", "", AddOptions{Force: true})
				require.Error(t, err)
			}
			assert.NoFileExists(t, target)
		})
	}
}
