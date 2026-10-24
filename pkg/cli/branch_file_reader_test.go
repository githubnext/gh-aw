//go:build !integration

package cli

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadRemoteRepoBranchFileContextUsesGETMethod(t *testing.T) {
	for _, tt := range []struct {
		name     string
		repo     string
		hostname string
		wantHost string
	}{
		{name: "default host", repo: "github/gh-aw"},
		{name: "repository host", repo: "github.example.com/github/gh-aw", wantHost: "github.example.com"},
		{name: "explicit host", repo: "github/gh-aw", hostname: "github.example.com", wantHost: "github.example.com"},
		{name: "explicit public host", repo: "github/gh-aw", hostname: "github.com", wantHost: "github.com"},
		{name: "host override", repo: "github.example.com/github/gh-aw", hostname: "other.example.com", wantHost: "other.example.com"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			fakeBinDir := t.TempDir()
			argsLog := filepath.Join(fakeBinDir, "gh-args.log")
			t.Setenv("EXPERIMENT_GH_ARGS_LOG", argsLog)
			script := "#!/bin/sh\nprintf '%s\\n' \"$@\" > \"$EXPERIMENT_GH_ARGS_LOG\"\nprintf '\\n' >> \"$EXPERIMENT_GH_ARGS_LOG\"\necho aGVsbG8=\n"
			require.NoError(t, os.WriteFile(filepath.Join(fakeBinDir, "gh"), []byte(script), 0o755))
			t.Setenv("PATH", fakeBinDir+string(os.PathListSeparator)+os.Getenv("PATH"))
			t.Setenv("GH_HOST", "ambient.example.com")

			content, err := readRemoteRepoBranchFileContext(context.Background(), tt.repo, "evals/{owner}+test#1", "nested dir/evals#1.jsonl", tt.hostname)
			require.NoError(t, err)
			assert.Equal(t, "hello", string(content))

			expected := []string{"api", "repos/github/gh-aw/contents/nested%20dir/evals%231.jsonl",
				"--method", "GET", "--raw-field", "ref=evals/{owner}+test#1", "--jq", ".content"}
			if tt.wantHost != "" {
				expected = append(expected, "--hostname", tt.wantHost)
			}

			assert.Equal(t, [][]string{expected}, readExperimentGHArgs(t, argsLog))
		})
	}
}

func TestReadRemoteRepoBranchFileNotFound(t *testing.T) {
	fakeBinDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(fakeBinDir, "gh"),
		[]byte("#!/bin/sh\necho 'HTTP 404: Not Found' >&2\nexit 1\n"), 0o755))
	t.Setenv("PATH", fakeBinDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	_, err := readRemoteRepoBranchFile("octo/repo", "experiments/missing", "state.jsonl", "")
	assert.ErrorIs(t, err, os.ErrNotExist)
}
