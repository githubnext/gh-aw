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
	for _, topology := range []string{"", string(RunnerTopologyArcDind)} {
		for _, driver := range []string{"", "pi_agent_core_driver.cjs", "pi_rpc_driver.cjs"} {
			t.Run(topology+"/"+driver, func(t *testing.T) {
				runner := ""
				commandLog := PiStreamingLogFile
				artifactLog := PiStreamingLogFile
				if topology != "" {
					runner = "runner:\n  topology: " + topology + "\n"
					commandLog = rewriteArcDindPath(PiStreamingLogFile)
					artifactLog = "${{ runner.temp }}/gh-aw/sandbox/agent/logs/pi-streaming.jsonl"
				}
				driverConfig := ""
				if driver != "" {
					driverConfig = "  driver: " + driver + "\n"
				}
				source := filepath.Join(t.TempDir(), "pi-logs.md")
				markdown := fmt.Sprintf("---\non: workflow_dispatch\npermissions:\n  contents: read\nstrict: false\n%sengine:\n  id: pi\n%s  config: '{\"session\":{\"enabled\":true,\"export\":true}}'\n---\nReport status.\n", runner, driverConfig)
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
				assert.Contains(t, summary, "GH_AW_AGENT_OUTPUT: "+artifactLog)
				upload := extractWorkflowStepByName(t, lock, "Upload agent artifacts")
				assert.Contains(t, upload, artifactLog)
				assert.Equal(t, constants.TmpSandboxAgentLogsDir+"pi-streaming.jsonl", PiStreamingLogFile)
				if topology == string(RunnerTopologyArcDind) {
					assert.Contains(t, execution, `--mount "${RUNNER_TEMP}/gh-aw/sandbox/agent:${RUNNER_TEMP}/gh-aw/sandbox/agent:rw"`)
					assert.Contains(t, commandLog, "${RUNNER_TEMP}/gh-aw/sandbox/agent/logs/")
				}
			})
		}
	}
}
