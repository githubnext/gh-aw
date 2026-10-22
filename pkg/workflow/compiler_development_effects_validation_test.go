//go:build !integration

package workflow

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDryRunRejectsResolvedSafeOutputHostEffects(t *testing.T) {
	write := "write"
	for _, test := range []struct {
		field string
		data  WorkflowData
	}{
		{"safe-outputs.steps", WorkflowData{SafeOutputs: &SafeOutputsConfig{Steps: []any{map[string]any{"run": "echo unsafe"}}}}},
		{"safe-outputs.threat-detection.steps", WorkflowData{SafeOutputs: &SafeOutputsConfig{ThreatDetection: &ThreatDetectionConfig{Steps: []any{map[string]any{"run": "echo unsafe"}}}}}},
		{"safe-outputs.threat-detection.post-steps", WorkflowData{SafeOutputs: &SafeOutputsConfig{ThreatDetection: &ThreatDetectionConfig{PostSteps: []any{map[string]any{"run": "echo unsafe"}}}}}},
		{"secret-masking.steps", WorkflowData{SecretMasking: &SecretMaskingConfig{Steps: []map[string]any{{"run": "echo unsafe"}}}}},
		{"safe-outputs.actions", WorkflowData{SafeOutputs: &SafeOutputsConfig{Actions: map[string]*SafeOutputActionConfig{"publish": {Uses: "./action", Env: map[string]string{"TOKEN": "${{ secrets.TOKEN }}"}}}}}},
		{"safe-outputs.id-token", WorkflowData{SafeOutputs: &SafeOutputsConfig{IDToken: &write}}},
	} {
		t.Run(test.field, func(t *testing.T) {
			c := NewCompiler()
			require.NoError(t, c.validateDryRunEffects(&test.data), "normal compilation is unchanged")
			c.SetDryRun(true)
			err := c.validateDryRunEffects(&test.data)
			require.ErrorContains(t, err, test.field)
			assert.NotContains(t, err.Error(), "echo unsafe")
			assert.NotContains(t, err.Error(), "secrets.TOKEN")
		})
	}
	none := "none"
	c := NewCompiler()
	c.SetDryRun(true)
	require.NoError(t, c.validateDryRunEffects(&WorkflowData{
		SafeOutputs:   &SafeOutputsConfig{IDToken: &none, ThreatDetection: &ThreatDetectionConfig{}},
		SecretMasking: &SecretMaskingConfig{},
	}))
}

func TestDryRunBuiltinJobCustomizationFailsClosed(t *testing.T) {
	for _, name := range []string{"agent", "upload_assets", "safe-outputs", "pre-activation"} {
		for _, field := range []string{
			"setup-steps", "pre-steps", "steps", "permissions", "uses", "with", "secrets",
			"env", "container", "services", "environment", "concurrency", "restore-memory",
			"runs-on", "future-augmentation-field",
		} {
			t.Run(name+"/"+field, func(t *testing.T) {
				data := &WorkflowData{Jobs: map[string]any{name: map[string]any{field: true}}}
				c := NewCompiler()
				require.NoError(t, c.validateDryRunEffects(data), "normal compilation is unchanged")
				c.SetDryRun(true)
				require.ErrorContains(t, c.validateDryRunEffects(data), "jobs."+name+"."+field)
			})
		}
	}
	assert.Empty(t, dryRunBuiltinJobEffects(map[string]any{
		"agent": map[string]any{
			"needs": []any{"activation"}, "if": "always()",
			"timeout-minutes": 5, "continue-on-error": true,
		},
		"custom": map[string]any{"restore-memory": true, "env": map[string]any{"BASH_ENV": "unsafe"}},
	}))
}

func TestDryRunRejectsDeclaredAndImportedResolvedHostEffects(t *testing.T) {
	for _, test := range []struct {
		field       string
		declaration string
	}{
		{"safe-outputs.steps", "safe-outputs:\n  steps:\n    - run: echo unsafe\n"},
		{"safe-outputs.threat-detection.steps", "safe-outputs:\n  threat-detection:\n    steps:\n      - run: echo unsafe\n"},
		{"safe-outputs.threat-detection.post-steps", "safe-outputs:\n  threat-detection:\n    post-steps:\n      - run: echo unsafe\n"},
		{"secret-masking.steps", "secret-masking:\n  steps:\n    - run: echo unsafe\n"},
		{"safe-outputs.actions", "safe-outputs:\n  actions:\n    publish:\n      uses: ./action\n      env:\n        TOKEN: ${{ secrets.TOKEN }}\n"},
		{"safe-outputs.id-token", "safe-outputs:\n  id-token: write\n"},
	} {
		for _, imported := range []bool{false, true} {
			t.Run(test.field+map[bool]string{false: "/declared", true: "/imported"}[imported], func(t *testing.T) {
				dir := t.TempDir()
				path := filepath.Join(dir, "workflow.md")
				declaration := test.declaration
				if imported {
					require.NoError(t, os.WriteFile(filepath.Join(dir, "shared.md"), []byte("---\n"+declaration+"---\nShared configuration.\n"), 0600))
					declaration = "imports: [shared.md]\n"
				}
				require.NoError(t, os.Mkdir(filepath.Join(dir, "action"), 0700))
				require.NoError(t, os.WriteFile(filepath.Join(dir, "action", "action.yml"), []byte("name: Publish\ndescription: Publish results\ninputs: {}\nruns:\n  using: composite\n  steps: []\n"), 0600))
				require.NoError(t, os.WriteFile(path, []byte("---\non: workflow_dispatch\nengine: copilot\n"+declaration+"---\nAnalyze the repository.\n"), 0600))
				c := NewCompiler()
				c.SetSkipValidation(true)
				c.SetApprove(true)
				c.SetDryRun(true)
				if imported && (test.field == "safe-outputs.actions" || test.field == "safe-outputs.id-token") {
					// These globals are not propagated by the import merger.
					data, err := c.ParseWorkflowFile(path)
					require.NoError(t, err)
					require.NotNil(t, data.SafeOutputs)
					assert.Empty(t, data.SafeOutputs.Actions)
					assert.Nil(t, data.SafeOutputs.IDToken)
					require.NoError(t, c.CompileWorkflow(path))
					return
				}
				require.ErrorContains(t, c.CompileWorkflow(path), test.field)
				require.NoFileExists(t, filepath.Join(dir, "workflow.lock.yml"))
				c.SetDryRun(false)
				require.NoError(t, c.CompileWorkflow(path), "normal compilation still supports the declarations")
			})
		}
	}
}

func TestDryRunRejectsMemoryRestoreUnderOptionalBuiltinName(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "workflow.md")
	require.NoError(t, os.WriteFile(path, []byte(`---
on: workflow_dispatch
engine: copilot
tools:
  cache-memory: true
jobs:
  upload_assets:
    restore-memory: true
    env:
      BASH_ENV: ./unsafe.sh
---
Analyze the repository.
`), 0600))
	c := NewCompiler()
	c.SetSkipValidation(true)
	c.SetApprove(true)
	c.SetDryRun(true)
	require.ErrorContains(t, c.CompileWorkflow(path), "jobs.upload_assets.restore-memory")
	require.NoFileExists(t, filepath.Join(dir, "workflow.lock.yml"))
	c.SetDryRun(false)
	require.ErrorContains(t, c.CompileWorkflow(path), "jobs.upload_assets: this name is reserved")
	require.NoFileExists(t, filepath.Join(dir, "workflow.lock.yml"))
}
