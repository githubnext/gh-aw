//go:build !integration

package cli

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSkillCommand(t *testing.T) {
	t.Chdir(t.TempDir())
	t.Setenv("GH_TOKEN", "test-token-not-for-target-org")
	originalList := listAgenticWorkflowsMarkdownFiles
	t.Cleanup(func() { listAgenticWorkflowsMarkdownFiles = originalList })
	listAgenticWorkflowsMarkdownFiles = func(context.Context) ([]string, error) {
		t.Fatal("skill must use bundled content without network access")
		return nil, nil
	}

	var stdout, stderr bytes.Buffer
	cmd := newSkillCommand(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs(nil)
	require.NoError(t, cmd.Execute())

	expected, err := renderAgenticWorkflowsSkillContent(embeddedFallbackAWMarkdownFiles())
	require.NoError(t, err)
	require.Equal(t, expected, stdout.String())
	require.Contains(t, stdout.String(), "---\nname: agentic-workflows\n")
	require.Contains(t, stdout.String(), ".github/aw/create-agentic-workflow.md")
	require.Contains(t, stdout.String(), "`.github/aw/debug-agentic-workflow.md#model-and-engine-misconfiguration`")
	require.NotContains(t, stdout.String(), agenticWorkflowsSkillFileListPlaceholder)
	require.Empty(t, stderr.String())

	files, err := os.ReadDir(".")
	require.NoError(t, err)
	require.Empty(t, files, "skill must not create repository files")
}

func TestShippedMarkdownTemplatesAvoidParentRelativeLinks(t *testing.T) {
	paths, err := filepath.Glob("data/*.md")
	require.NoError(t, err)
	require.NotEmpty(t, paths)

	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			content, err := os.ReadFile(path)
			require.NoError(t, err)
			require.NotRegexp(t, `(?m)(\]\(\s*<?|^\s*\[[^\]]+\]:\s*<?)(\./)?\.\./`, string(content),
				"shipped templates must not link to parent paths that are unavailable in consumer repositories")
		})
	}
}

func TestSkillCommandRejectsArguments(t *testing.T) {
	var output bytes.Buffer
	cmd := newSkillCommand(&output)
	cmd.SetErr(io.Discard)
	cmd.SetArgs([]string{"unexpected"})
	require.Error(t, cmd.Execute())
	require.Empty(t, output.String())
}

func TestSkillCommandWriteError(t *testing.T) {
	cmd := newSkillCommand(skillCommandErrorWriter{})
	cmd.SetErr(io.Discard)
	cmd.SetArgs(nil)
	require.ErrorContains(t, cmd.Execute(), "failed to write agentic-workflows skill")
}

type skillCommandErrorWriter struct{}

func (skillCommandErrorWriter) Write([]byte) (int, error) {
	return 0, errors.New("output unavailable")
}
