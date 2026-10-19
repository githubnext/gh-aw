//go:build !integration

package workflow

import (
	"fmt"
	"strings"
	"testing"

	"github.com/goccy/go-yaml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadRepoConfigAdmission(t *testing.T) {
	for _, content := range []string{`{}`, `{"admission":{"url":"https://cao.example/api/admission"}}`, `{"admission":{"url":"https://cao.example/api/admission","on_outage":"ignore"}}`, `{"admission":{"url":"https://cao.example/api/admission","on_outage":"block"}}`} {
		t.Run(content, func(t *testing.T) {
			dir := t.TempDir()
			writeAWJSON(t, dir, content)
			cfg, err := LoadRepoConfig(dir)
			require.NoError(t, err)
			if content == "{}" {
				assert.Nil(t, cfg.Admission)
			} else {
				require.NotNil(t, cfg.Admission)
				assert.Equal(t, "https://cao.example/api/admission", cfg.Admission.URL)
			}
		})
	}
}

func TestLoadRepoConfigAdmissionRejectsInvalidConfig(t *testing.T) {
	for _, config := range []string{
		`null`, `false`, `true`, `""`, `{}`, `{"url":""}`, `{"url":123}`,
		`{"url":"http://cao.example/admission"}`,
		`{"url":"https://"}`,
		`{"url":"https://user:secret@cao.example/admission"}`,
		`{"url":"https://cao.example/admission?token=secret"}`,
		`{"url":"https://cao.example/admission?"}`,
		`{"url":"https://cao.example/admission#fragment"}`,
		`{"url":"https://cao.example/admission#"}`,
		`{"url":"https://cao.example/ bad"}`,
		`{"url":"https://cao.example/\\evil"}`,
		`{"url":"https://${{ secrets.ENDPOINT }}/admission"}`,
		`{"url":"https://cao.example/admission","token":"OTHER_TOKEN"}`,
		`{"url":"https://cao.example/admission","continue_on_error":true}`,
		`{"url":"https://cao.example/admission","on_outage":"allow"}`,
		`{"url":"https://cao.example/admission","on_outage":null}`,
		`{"url":"https://cao.example/admission","on_outage":false}`,
	} {
		t.Run(config, func(t *testing.T) {
			dir := t.TempDir()
			writeAWJSON(t, dir, `{"admission":`+config+`}`)
			_, err := LoadRepoConfig(dir)
			require.Error(t, err)
			compiler := NewCompiler()
			compiler.gitRoot = dir
			_, err = compiler.buildActivationJob(&WorkflowData{Name: "Admission"}, false, "", "admission.lock.yml")
			require.Error(t, err, "invalid admission configuration must not compile without a gate")
		})
	}
}

func TestActivationAdmissionGate(t *testing.T) {
	for _, configured := range []bool{false, true} {
		t.Run(map[bool]string{false: "disabled", true: "enabled"}[configured], func(t *testing.T) {
			dir := t.TempDir()
			if configured {
				writeAWJSON(t, dir, `{"admission":{"url":"https://cao.example/api/admission"}}`)
			}
			compiler := NewCompiler()
			compiler.gitRoot = dir
			data := &WorkflowData{
				Name: "Admission", WorkflowID: "admission", AIReaction: "eyes",
				ActivationGitHubToken: "${{ secrets.CUSTOM_ACTIVATION_TOKEN }}",
				StaleCheckDisabled:    true,
				RawFrontmatter:        map[string]any{"max-daily-ai-credits": -1},
			}
			job, err := compiler.buildActivationJob(data, false, "", "admission.lock.yml")
			require.NoError(t, err)
			steps := strings.Join(job.Steps, "")
			if !configured {
				assert.NotContains(t, steps, "check_admission.cjs")
				assert.NotContains(t, job.Permissions, "actions: read")
				return
			}
			assert.Contains(t, job.Permissions, "actions: read")
			step := extractWorkflowStepByName(t, steps, "Check workflow admission")
			assert.Contains(t, step, `GH_AW_ADMISSION_URL: "https://cao.example/api/admission"`)
			assert.Contains(t, step, "GH_AW_ADMISSION_TOKEN: ${{ github.token }}")
			assert.Contains(t, step, `GH_AW_ADMISSION_ON_OUTAGE: "ignore"`)
			assert.Contains(t, step, "github-token: ${{ github.token }}")
			assert.Contains(t, step, `GH_AW_WORKFLOW_ID: "admission"`)
			assert.Contains(t, step, "check_admission.cjs")
			assert.NotContains(t, step, "secrets.")
			assert.NotContains(t, step, "continue-on-error")
			assert.NotContains(t, step, "if:")
			assert.Less(t, strings.Index(steps, "Check workflow admission"), strings.Index(steps, "Check for OAuth tokens"))
			assert.Less(t, strings.Index(steps, "Check workflow admission"), strings.Index(steps, "Add eyes reaction"))
		})
	}
}

func TestActivationAdmissionBlocksOutagesWhenConfigured(t *testing.T) {
	dir := t.TempDir()
	writeAWJSON(t, dir, `{"admission":{"url":"https://cao.example/api/admission","on_outage":"block"}}`)
	compiler := NewCompiler()
	compiler.gitRoot = dir
	step, err := compiler.buildAdmissionGateStep(&WorkflowData{WorkflowID: "admission"})
	require.NoError(t, err)
	assert.Contains(t, strings.Join(step, ""), `GH_AW_ADMISSION_ON_OUTAGE: "block"`)
}

func TestCompileToYAMLAdmissionGate(t *testing.T) {
	for _, trigger := range []string{"push", "workflow_dispatch", "workflow_call"} {
		t.Run(trigger, func(t *testing.T) {
			dir := t.TempDir()
			writeAWJSON(t, dir, `{"admission":{"url":"https://cao.example/api/admission","on_outage":"block"}}`)
			compiler := NewCompiler(WithNoEmit(true), WithSkipValidation(true))
			compiler.gitRoot = dir
			markdown := fmt.Sprintf(`---
on:
  %s: {}
engine: copilot
jobs:
  agent:
    if: always()
---
Perform the requested task.
`, trigger)
			data, err := compiler.ParseWorkflowString(markdown, "admission.md")
			require.NoError(t, err)
			compiled, err := compiler.CompileToYAML(data, "admission.md")
			require.NoError(t, err)
			var workflow map[string]any
			require.NoError(t, yaml.Unmarshal([]byte(compiled), &workflow))
			jobs := workflow["jobs"].(map[string]any)
			activation := jobs["activation"].(map[string]any)
			found := false
			for _, rawStep := range activation["steps"].([]any) {
				step := rawStep.(map[string]any)
				if step["id"] == "admission" {
					found = true
					env := step["env"].(map[string]any)
					assert.Equal(t, "${{ github.token }}", env["GH_AW_ADMISSION_TOKEN"])
					assert.NotEmpty(t, env["GH_AW_WORKFLOW_ID"])
					assert.Equal(t, "block", env["GH_AW_ADMISSION_ON_OUTAGE"])
					assert.NotContains(t, step, "continue-on-error")
					assert.NotContains(t, step, "if")
				}
			}
			require.True(t, found, "every trigger must retain the admission step")
			agent := jobs["agent"].(map[string]any)
			assert.Contains(t, agent["if"], "needs.activation.result == 'success'")
		})
	}
}
