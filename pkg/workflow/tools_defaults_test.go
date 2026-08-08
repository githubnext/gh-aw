package workflow

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/testutil"
	"github.com/github/gh-aw/pkg/workflow/compilerenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApplyDefaults_DefaultTimeoutMinutesFromEnv(t *testing.T) {
	t.Setenv(compilerenv.DefaultTimeoutMinutes, "45")

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
	assert.Equal(t, "timeout-minutes: 45", data.TimeoutMinutes)
}

func TestApplyDefaults_DefaultTimeoutMinutesFallback(t *testing.T) {
	t.Setenv(compilerenv.DefaultTimeoutMinutes, "0")

	tmpDir := testutil.TempDir(t, "tools-default-timeout-fallback")
	markdownPath := filepath.Join(tmpDir, "workflow.md")
	require.NoError(t, os.WriteFile(markdownPath, []byte("# Test"), 0644))

	data := &WorkflowData{
		Name: "Test",
		On: `on:
  workflow_dispatch:`,
	}

	compiler := NewCompiler()
	require.NoError(t, compiler.applyDefaults(data, markdownPath))
	expectedDefaultTimeout := "timeout-minutes: " + strconv.Itoa(int(constants.DefaultAgenticWorkflowTimeout/time.Minute))
	assert.Equal(t, expectedDefaultTimeout, data.TimeoutMinutes)
}

func TestApplyDefaults_DockerSbxRuntimeDefault(t *testing.T) {
	tmpDir := testutil.TempDir(t, "tools-default-docker-sbx")
	markdownPath := filepath.Join(tmpDir, "workflow.md")
	require.NoError(t, os.WriteFile(markdownPath, []byte("# Test"), 0644))

	tests := []struct {
		name    string
		runsOn  string
		runtime AgentRuntime
		want    AgentRuntime
		def     bool
	}{
		{name: "GitHub-hosted Ubuntu", runsOn: "runs-on: ubuntu-latest", want: AgentRuntimeDockerSbx, def: true},
		{name: "explicit runtime", runsOn: "runs-on: ubuntu-latest", runtime: AgentRuntimeGVisor, want: AgentRuntimeGVisor},
		{name: "self-hosted", runsOn: "runs-on: self-hosted"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := &WorkflowData{
				Name:   "Test",
				On:     "on:\n  workflow_dispatch:",
				RunsOn: tt.runsOn,
				SandboxConfig: &SandboxConfig{Agent: &AgentSandboxConfig{
					ID:      "awf",
					Runtime: tt.runtime,
				}},
			}
			require.NoError(t, NewCompiler().applyDefaults(data, markdownPath))
			assert.Equal(t, tt.want, data.SandboxConfig.Agent.Runtime)
			assert.Equal(t, tt.def, data.SandboxConfig.Agent.RuntimeDefaulted)
		})
	}
}
