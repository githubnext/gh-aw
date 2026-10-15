//go:build !integration

package workflow

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/goccy/go-yaml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWorkflowCallLocalArtifactPrefixes(t *testing.T) {
	for _, trigger := range []string{"workflow_call", "workflow_dispatch"} {
		t.Run(trigger, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "local-metadata.md")
			require.NoError(t, os.WriteFile(path, []byte(`---
on: `+trigger+`
engine: copilot
permissions:
  contents: read
tools:
  cache-memory: true
  repo-memory: true
safe-outputs:
  github-app:
    app-id: ${{ vars.APP_ID }}
    private-key: ${{ secrets.APP_PRIVATE_KEY }}
  create-issue:
  create-code-scanning-alert:
  upload-code-coverage:
  upload-asset:
  jobs:
    notify:
      steps:
        - run: echo done
---
Report findings.
`), 0600))
			c := NewCompiler(WithNoEmit(true))
			require.NoError(t, c.CompileWorkflow(path))
			for name, job := range c.jobManager.GetAllJobs() {
				steps := strings.Join(job.Steps, "")
				assert.NotContains(t, steps, "needs.activation.outputs.artifact_prefix", name)
				assert.NotContains(t, steps, "needs.agent.outputs.artifact_prefix", name)
				assert.NotContains(t, steps, "needs.activation.outputs.target_repo", name)
				assert.NotContains(t, steps, "needs.activation.outputs.target_ref", name)
				if trigger != "workflow_call" {
					assert.NotContains(t, steps, "id: artifact-prefix", name)
					continue
				}
				if strings.Contains(steps, "steps.artifact-prefix.outputs.prefix") {
					assert.Equal(t, 1, strings.Count(steps, "id: artifact-prefix\n"), name)
					assert.Less(t, strings.Index(steps, "id: artifact-prefix"), strings.Index(steps, "steps.artifact-prefix.outputs.prefix"), name)
				}
				if strings.Contains(steps, "steps.resolve-host-repo.outputs.") {
					assert.Equal(t, 1, strings.Count(steps, "id: resolve-host-repo\n"), name)
					assert.Less(t, strings.Index(steps, "id: resolve-host-repo"), strings.Index(steps, "steps.resolve-host-repo.outputs."), name)
				}
				protocolSteps, err := parseArtifactProtocolSteps(job)
				require.NoError(t, err)
				for _, step := range protocolSteps {
					if strings.HasPrefix(step.Uses, "actions/download-artifact@") {
						for _, key := range []string{"name", "pattern"} {
							if value, ok := step.With[key].(string); ok {
								assert.NotContains(t, value, localArtifactPrefix, "cross-job downloads must use producer attempts: %s", name)
							}
						}
					}
					if !strings.HasPrefix(step.ID, "artifact-source-") {
						continue
					}
					producer := strings.TrimPrefix(step.ID, "artifact-source-")
					assert.Contains(t, job.Needs, producer)
					source, exists := c.jobManager.GetJob(producer)
					require.True(t, exists)
					assert.Contains(t, strings.Join(source.Steps, ""), "name: "+artifactMetadataName(producer))
				}
			}
			for _, name := range []string{"activation", "agent", "detection", "conclusion", "safe_outputs", "upload_code_scanning_sarif", "upload_code_coverage", "upload_assets", "notify", "update_cache_memory", "push_repo_memory"} {
				_, exists := c.jobManager.GetJob(name)
				assert.True(t, exists, "fixture must exercise %s", name)
			}
		})
	}
}

func TestSafeOutputJobLocalMetadataPreSteps(t *testing.T) {
	for _, tc := range []struct {
		name           string
		artifactPrefix bool
		resolveHost    bool
	}{
		{name: "no metadata"},
		{name: "artifact prefix", artifactPrefix: true},
		{name: "host repository", resolveHost: true},
		{name: "both metadata steps", artifactPrefix: true, resolveHost: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := NewCompiler()
			data := &WorkflowData{On: "workflow_call"}
			var preSteps []string
			if tc.artifactPrefix {
				preSteps = append(preSteps, generateArtifactPrefixStep()...)
			}
			if tc.resolveHost {
				preSteps = append(preSteps, c.generateResolveHostRepoStep(data))
			}
			job, err := c.buildSafeOutputJob(data, SafeOutputJobConfig{
				JobName:     "test_output",
				StepName:    "Test output",
				StepID:      "test_output",
				MainJobName: "agent",
				Permissions: NewPermissions(),
				PreSteps:    preSteps,
				Script:      "console.log('test')",
			})
			require.NoError(t, err)
			steps := strings.Join(job.Steps, "")
			assert.Equal(t, 1, strings.Count(steps, "id: artifact-prefix\n"))
			assert.Equal(t, 1, strings.Count(steps, "id: resolve-host-repo\n"))
			assert.Less(t, strings.Index(steps, "id: artifact-prefix\n"), strings.Index(steps, "id: test_output\n"))
			assert.Less(t, strings.Index(steps, "id: resolve-host-repo\n"), strings.Index(steps, "id: test_output\n"))
		})
	}
}

func TestLocalArtifactPrefixSurvivesMaskedJobOutputs(t *testing.T) {
	var steps []map[string]any
	require.NoError(t, yaml.Unmarshal([]byte(strings.Join(generateArtifactPrefixStep(), "")), &steps))
	script, ok := steps[0]["run"].(string)
	require.True(t, ok)
	require.NotContains(t, script, "${{")
	require.NotContains(t, script, "RUNNER_TEMP", "scriptless downstream jobs must not need Setup Scripts")

	compute := func(script, inputs, attempt string) string {
		t.Helper()
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		outputPath := filepath.Join(t.TempDir(), "output")
		cmd := exec.CommandContext(ctx, "bash", "-c", script)
		cmd.Env = append(os.Environ(), "INPUTS_JSON="+inputs, "GITHUB_RUN_ATTEMPT="+attempt, "GITHUB_OUTPUT="+outputPath,
			"GH_AW_NEEDS_ACTIVATION_OUTPUTS_ARTIFACT_PREFIX=", "GH_AW_NEEDS_AGENT_OUTPUTS_ARTIFACT_PREFIX=")
		output, err := cmd.CombinedOutput()
		require.NoError(t, err, "%s", output)
		value, err := os.ReadFile(outputPath)
		require.NoError(t, err)
		return string(value)
	}
	original, err := os.ReadFile("../../actions/setup/sh/compute_artifact_prefix.sh")
	require.NoError(t, err)
	for _, inputs := range []string{`{}`, `{"target":"host/platform"}`, `{"target":"$(touch /tmp/not-executed)"}`} {
		for _, attempt := range []string{"1", "2"} {
			expected := compute(string(original), inputs, attempt)
			assert.Regexp(t, `^prefix=[a-f0-9]{8}-\nidentity_prefix=[a-f0-9]{8}-\n$`, expected)
			assert.Equal(t, expected, compute(script, inputs, attempt))
			assert.Equal(t, expected, compute(script, inputs, attempt), "independent jobs must agree without transported outputs")
		}
	}
	assert.NotEqual(t, compute(script, `{"target":"one"}`, "1"), compute(script, `{"target":"two"}`, "1"))
	assert.NotEqual(t, compute(script, `{}`, "1"), compute(script, `{}`, "2"))
}

func TestWorkflowCallTemplatedStepContinueOnError(t *testing.T) {
	for _, tc := range []struct {
		name   string
		config string
		job    string
	}{
		{
			name: "external detector",
			config: `safe-outputs:
  create-issue:
  threat-detection:
    continue-on-error: ${{ inputs.tolerate }}
features:
  gh-aw-detection: true
`,
			job: "detection",
		},
		{
			name: "custom job",
			config: `jobs:
  optional:
    runs-on: ubuntu-latest
    steps:
      - run: echo optional
        continue-on-error: ${{ inputs.tolerate }}
`,
			job: "optional",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			workflow := filepath.Join(t.TempDir(), "templated-step.md")
			content := `---
on:
  workflow_call:
    inputs:
      tolerate:
        type: boolean
        default: false
engine: copilot
permissions:
  contents: read
` + tc.config + "---\nTest templated step options.\n"
			require.NoError(t, os.WriteFile(workflow, []byte(content), 0600))
			c := NewCompiler(WithNoEmit(true))
			require.NoError(t, c.CompileWorkflow(workflow))
			job, exists := c.jobManager.GetJob(tc.job)
			require.True(t, exists)
			assert.Contains(t, strings.Join(job.Steps, ""), "continue-on-error: ${{ inputs.tolerate }}")
		})
	}
}
