package workflow

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/github/gh-aw/pkg/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApplyDefaults_DefaultTimeoutMinutesUsesActionVariable(t *testing.T) {
	tmpDir := testutil.TempDir(t, "tools-default-timeout")
	markdownPath := filepath.Join(tmpDir, "workflow.md")
	require.NoError(t, os.WriteFile(markdownPath, []byte("# Test"), 0644))

	data := &WorkflowData{
		Name: "Test",
		On: `on:
  workflow_dispatch:`,
	}

	compiler := NewCompiler()
	require.NoError(t, compiler.applyDefaults(data, markdownPath))
	assert.Equal(t, "timeout-minutes: ${{ fromJSON(vars.GH_AW_DEFAULT_TIMEOUT_MINUTES || '20') }}", data.TimeoutMinutes)
}
