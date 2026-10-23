//go:build integration

package workflow

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/github/gh-aw/pkg/testutil"
	"github.com/goccy/go-yaml"
	"github.com/stretchr/testify/require"
)

func TestWorkQueueE2ESmokeVerification(t *testing.T) {
	dir := testutil.TempDir(t, "work-queue-e2e-smoke-")
	dir = filepath.Join(dir, ".github", "workflows")
	require.NoError(t, os.MkdirAll(dir, 0o700))
	for _, name := range []string{"smoke-work-queue-e2e", "smoke-work-queue-e2e-worker"} {
		source, err := os.ReadFile("../../.github/workflows/" + name + ".md")
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(filepath.Join(dir, name+".md"), source, 0o600))
	}
	for _, worker := range []bool{false, true} {
		name, jobName := "smoke-work-queue-e2e", "verify_smoke_dispatch"
		if worker {
			name, jobName = name+"-worker", "verify_smoke_completion"
		}
		t.Run(name, func(t *testing.T) {
			require.NoError(t, NewCompiler(WithVersion("integration"), WithWorkflowIdentifier(name)).CompileWorkflow(filepath.Join(dir, name+".md")))
			lock, err := os.ReadFile(filepath.Join(dir, name+".lock.yml"))
			require.NoError(t, err)
			var compiled struct {
				Jobs map[string]struct {
					Needs   any
					If      string
					Outputs map[string]string
					Steps   []struct {
						Env map[string]string
						Run string
					}
				}
			}
			require.NoError(t, yaml.Unmarshal(lock, &compiled))
			job, ok := compiled.Jobs[jobName]
			require.True(t, ok)
			require.Contains(t, job.Needs, "safe_outputs")
			require.Contains(t, job.If, "always()")
			require.NotEmpty(t, job.Steps)
			step := job.Steps[len(job.Steps)-1]
			require.Equal(t, "${{ needs.safe_outputs.result }}", step.Env["SAFE_OUTPUTS_RESULT"])
			outputKey := "work_queue_requests"
			envKey := "WORK_QUEUE_REQUESTS"
			outputExpression := "${{ steps.work_queue_controls.outputs.work_queue_requests }}"
			if worker {
				outputKey, envKey = "work_queue_outcome", "WORK_QUEUE_OUTCOME"
				outputExpression = "${{ steps.process_safe_outputs.outputs.work_queue_outcome }}"
			}
			require.Equal(t, "${{ needs.safe_outputs.outputs."+outputKey+" }}", step.Env[envKey])
			require.Equal(t, outputExpression, compiled.Jobs["safe_outputs"].Outputs[outputKey])

			type verificationCase struct {
				name    string
				value   string
				result  string
				success bool
			}
			cases := []verificationCase{
				{"missing evidence", "", "success", false},
				{"failed processing", "", "failure", false},
				{"skipped processing", "", "skipped", false},
				{"cancelled processing", "", "cancelled", false},
			}
			if worker {
				cases = append(cases,
					verificationCase{"completed original Claim", "completed", "success", true},
					verificationCase{"cancelled original Claim", "cancelled", "success", false},
					verificationCase{"completion with failed processing", "completed", "failure", false},
				)
			} else {
				for _, entry := range []struct {
					name     string
					launches string
					success  bool
				}{
					{"bound native dispatch", `[{"state":"bound","dispatched":true}]`, true},
					{"no grant", `[]`, false},
					{"two dispatches", `[{"state":"bound","dispatched":true},{"state":"bound","dispatched":true}]`, false},
					{"staged preview", `[{"state":"staged_preview"}]`, false},
					{"released", `[{"state":"released"}]`, false},
					{"unresolved launch", `[{"state":"pending","dispatched":true}]`, false},
					{"bound without dispatch", `[{"state":"bound","dispatched":false}]`, false},
				} {
					value := `{"version":3,"status":"ok","success":true,"receipts":[{"status":"durable","launches":` + entry.launches + `}]}`
					cases = append(cases, verificationCase{entry.name, value, "success", entry.success})
				}
			}
			for _, tc := range cases {
				t.Run(tc.name, func(t *testing.T) {
					cmd := exec.Command("bash", "-e", "-c", step.Run)
					cmd.Env = append(os.Environ(), "SAFE_OUTPUTS_RESULT="+tc.result, envKey+"="+tc.value)
					output, err := cmd.CombinedOutput()
					if tc.success {
						require.NoError(t, err, "%s", output)
					} else {
						require.Error(t, err, "%s", output)
					}
				})
			}
		})
	}
}
