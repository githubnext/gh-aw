//go:build !integration

package workflow

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompilePiStreamingLogPaths(t *testing.T) {
	for _, tc := range []struct {
		name            string
		topology        string
		sandboxDisabled bool
	}{
		{name: "default"},
		{name: "arc-dind", topology: string(RunnerTopologyArcDind)},
		{name: "arc-dind-sandbox-disabled", topology: string(RunnerTopologyArcDind), sandboxDisabled: true},
	} {
		for _, driver := range []string{"", "pi_agent_core_driver.cjs", "pi_rpc_driver.cjs"} {
			t.Run(tc.name+"/"+driver, func(t *testing.T) {
				runner := ""
				commandLog := PiStreamingLogFile
				parserLog := PiStreamingLogFile
				artifactLog := PiStreamingLogFile
				if tc.topology != "" {
					runner = "runner:\n  topology: " + tc.topology + "\n"
					artifactLog = "${{ runner.temp }}/gh-aw/sandbox/agent/logs/pi-streaming.jsonl"
					if !tc.sandboxDisabled {
						commandLog = rewriteArcDindPath(PiStreamingLogFile)
						parserLog = artifactLog
					}
				}
				sandboxConfig := ""
				if tc.sandboxDisabled {
					sandboxConfig = "features:\n  dangerously-disable-sandbox-agent: true\nsandbox:\n  agent: false\n"
				}
				driverConfig := ""
				if driver != "" {
					driverConfig = "  driver: " + driver + "\n"
				}
				source := filepath.Join(t.TempDir(), "pi-logs.md")
				markdown := fmt.Sprintf("---\non: workflow_dispatch\npermissions:\n  contents: read\nstrict: false\n%s%sengine:\n  id: pi\n%s  config: '{\"session\":{\"enabled\":true,\"export\":true}}'\n---\nReport status.\n", runner, sandboxConfig, driverConfig)
				require.NoError(t, os.WriteFile(source, []byte(markdown), 0o600))
				require.NoError(t, NewCompiler(WithVersion("dev")).CompileWorkflow(source))
				lockBytes, err := os.ReadFile(filepath.Join(filepath.Dir(source), "pi-logs.lock.yml"))
				require.NoError(t, err)
				lock := string(lockBytes)
				execution := extractWorkflowStepByName(t, lock, "Execute Pi CLI")
				assert.Contains(t, execution, "mkdir -p "+filepath.Dir(commandLog)+"/")
				assert.Contains(t, execution, "tee "+commandLog)
				assert.NotContains(t, lock, "/gh-aw/pi-streaming.jsonl")
				if driver == "" {
					assert.Contains(t, execution, `export GH_AW_AGENT_OUTPUT="`+commandLog+`"`)
					assert.Contains(t, execution, `/pi_session_export.cjs")`)
				} else {
					assert.Contains(t, execution, "/"+driver)
				}
				summary := extractWorkflowStepByName(t, lock, "Parse agent logs for step summary")
				assert.Contains(t, summary, "GH_AW_AGENT_OUTPUT: "+parserLog)
				upload := extractWorkflowStepByName(t, lock, "Upload agent artifacts")
				assert.Contains(t, upload, artifactLog)
				assert.Equal(t, constants.TmpSandboxAgentLogsDir+"pi-streaming.jsonl", PiStreamingLogFile)
				if tc.topology == string(RunnerTopologyArcDind) && !tc.sandboxDisabled {
					assert.Contains(t, execution, `--mount "${RUNNER_TEMP}/gh-aw/sandbox/agent:${RUNNER_TEMP}/gh-aw/sandbox/agent:rw"`)
					assert.Contains(t, commandLog, "${RUNNER_TEMP}/gh-aw/sandbox/agent/logs/")
				}
				if tc.sandboxDisabled {
					assert.NotContains(t, execution, "awf ")
					assert.NotContains(t, summary, "GH_AW_AGENT_OUTPUT: "+artifactLog)
				}
			})
		}
	}
}
