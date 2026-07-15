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
	fakeBinDir := t.TempDir()
	fakeGH := filepath.Join(fakeBinDir, "gh")
	argsLog := filepath.Join(fakeBinDir, "gh-args.log")
	script := "#!/bin/sh\nprintf '%s\\n' \"$*\" > \"" + argsLog + "\"\necho aGVsbG8=\n"
	require.NoError(t, os.WriteFile(fakeGH, []byte(script), 0o755))
	t.Setenv("PATH", fakeBinDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	content, err := readRemoteRepoBranchFileContext(context.Background(), "github/gh-aw", "evals/workflow", "evals.jsonl", "")
	require.NoError(t, err)
	assert.Equal(t, "hello", string(content))

	recordedArgs, err := os.ReadFile(argsLog)
	require.NoError(t, err)
	assert.Contains(t, string(recordedArgs), "--method GET")
}
