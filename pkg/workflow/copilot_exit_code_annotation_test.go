//go:build !integration

// Tests guarding the failure instrumentation of the "Execute GitHub Copilot CLI"
// step. A non-zero exit from the Copilot CLI used to be indistinguishable from a
// successful run in the job log: the agent stream and its post-processing output
// ended normally and nothing reported why the step failed. The EXIT trap therefore
// emits an explicit ::error:: annotation carrying the exit code.
package workflow

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCopilotExitCodeTrap_EmitsErrorAnnotation pins the generated trap so the
// failure annotation cannot be silently dropped.
func TestCopilotExitCodeTrap_EmitsErrorAnnotation(t *testing.T) {
	got := buildCopilotSettingsCleanupAndExitCodeTrap()

	assert.Contains(t, got, `if [ "$gh_aw_exit_code" -ne 0 ]; then`,
		"trap must guard the annotation on a non-zero exit code:\n%s", got)
	assert.Contains(t, got, `echo "::error::Execute GitHub Copilot CLI exited with code $gh_aw_exit_code"`,
		"trap must emit an error annotation naming the step and exit code:\n%s", got)
	assert.Contains(t, got, `> `+agentExecutionExitCodePath,
		"trap must still persist the exit code for the OTLP conclusion span:\n%s", got)
	assert.Contains(t, got, `rm -f "`+copilotSettingsPath+`"`,
		"trap must still clean up the Copilot settings file:\n%s", got)
}

// TestBashIntegration_CopilotExitCodeTrap runs the generated trap through bash to
// confirm the annotation is printed with the real exit code on failure, that the
// exit code file is still written, and that successful runs stay silent.
func TestBashIntegration_CopilotExitCodeTrap(t *testing.T) {
	trap := buildCopilotSettingsCleanupAndExitCodeTrap()

	tests := []struct {
		name          string
		body          string
		wantExitCode  string
		wantErrorLine bool
	}{
		{name: "success is silent", body: "true\n", wantExitCode: "0"},
		{name: "explicit failure", body: "exit 3\n", wantExitCode: "3", wantErrorLine: true},
		{name: "failing command", body: "set -o pipefail\nfalse | cat\n", wantExitCode: "1", wantErrorLine: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			home := t.TempDir()
			exitCodeFile := t.TempDir() + "/agent_execution_exit_code.txt"
			script := strings.Replace(trap, agentExecutionExitCodePath, exitCodeFile, 1) + tt.body

			stdout, stderr, _ := runBashWithHome(t, home, script)

			errorLine := "::error::Execute GitHub Copilot CLI exited with code " + tt.wantExitCode
			if tt.wantErrorLine {
				assert.Contains(t, stdout, errorLine,
					"failing run must surface the exit code:\nstdout=%s\nstderr=%s", stdout, stderr)
			} else {
				assert.NotContains(t, stdout, "::error::",
					"successful run must not emit an error annotation:\nstdout=%s", stdout)
			}

			written, err := os.ReadFile(exitCodeFile)
			require.NoError(t, err, "trap must persist the exit code file")
			assert.Equal(t, tt.wantExitCode, string(written),
				"persisted exit code must match the script exit status")
		})
	}
}

// TestCopilotExecutionStep_ContainsExitCodeAnnotation verifies the annotation is
// present in the generated step for both the direct and firewall command paths.
func TestCopilotExecutionStep_ContainsExitCodeAnnotation(t *testing.T) {
	tests := []struct {
		name string
		wd   *WorkflowData
	}{
		{
			name: "direct command",
			wd:   &WorkflowData{Name: "direct"},
		},
		{
			name: "firewall command",
			wd: &WorkflowData{
				Name: "firewall",
				NetworkPermissions: &NetworkPermissions{
					Firewall: &FirewallConfig{Enabled: true},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine := NewCopilotEngine()
			steps := engine.GetExecutionSteps(tt.wd, "/tmp/gh-aw/test.log")
			stepContent := requireCopilotExecutionStep(t, steps)

			assert.Contains(t, stepContent, "::error::Execute GitHub Copilot CLI exited with code",
				"execution step must surface a non-zero exit code in the log:\n%s", stepContent)
		})
	}
}
