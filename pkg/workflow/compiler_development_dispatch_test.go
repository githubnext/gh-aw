//go:build !integration

package workflow

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/goccy/go-yaml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDryRunWorkflowDispatch(t *testing.T) {
	for _, tc := range []struct {
		name, on string
		roles    []string
	}{
		{name: "push", on: "on: push\n"},
		{name: "dispatch", on: "on: workflow_dispatch\n"},
		{name: "dispatch-list", on: "on: [push, workflow_dispatch]\n"},
		{name: "events", on: "on: [push, pull_request]\n"},
		{name: "schedule", on: "on:\n  schedule:\n    - cron: '0 0 * * *'\n"},
		{name: "existing-inputs", on: "on:\n  workflow_dispatch:\n    inputs:\n      note:\n        type: string\n        required: true\n        default: 2026-01-01\n"},
		{name: "reusable-inputs", on: "on:\n  workflow_call:\n    inputs:\n      note:\n        type: string\n        required: true\n"},
		{name: "unrestricted", on: "on: push\n", roles: []string{"all"}},
		{name: "admin-only", on: "on: push\n", roles: []string{"admin"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			data := &WorkflowData{
				On: tc.on, Roles: tc.roles, Bots: []string{"trusted[bot]"},
				RawFrontmatter: map[string]any{"on": map[string]any{"roles": "all", "bots": []string{"trusted[bot]"}}},
			}
			compiler := NewCompiler()
			compiler.SetDryRun(true)
			result, err := compiler.prepareDryRunWorkflowData(data)
			require.NoError(t, err)
			var compiled struct{ On map[string]any }
			require.NoError(t, yaml.Unmarshal([]byte(result.On), &compiled))
			assert.Contains(t, compiled.On, "workflow_dispatch")
			if tc.name == "existing-inputs" || tc.name == "reusable-inputs" {
				assert.Contains(t, result.On, "note:")
				assert.Contains(t, result.On, "required: true")
			}
			if tc.name == "existing-inputs" {
				assert.Contains(t, result.On, `default: 2026-01-01`)
				assert.NotContains(t, result.On, "T00:00:00")
			}
			if tc.name == "admin-only" {
				assert.Equal(t, []string{"admin"}, result.Roles)
			} else {
				assert.Equal(t, []string{"admin", "maintainer"}, result.Roles)
			}
			assert.Empty(t, result.Bots)
			assert.True(t, compiler.needsRoleCheck(result, data.RawFrontmatter))
			assert.Equal(t, tc.on, data.On)
			assert.Equal(t, tc.roles, data.Roles)
			assert.Equal(t, []string{"trusted[bot]"}, data.Bots)
			assert.Equal(t, "all", data.RawFrontmatter["on"].(map[string]any)["roles"])
		})
	}
}

func TestDryRunWorkflowDispatchRejectsInvalidTriggers(t *testing.T) {
	for _, source := range []string{"on: 42\n", "on: true\n", "on: [push, 42]\n", "on: [push, {}]\n", "- push\n"} {
		t.Run(source, func(t *testing.T) {
			require.Error(t, prepareDryRunWorkflowDispatch(&WorkflowData{On: source}))
		})
	}
}

func TestDryRunWorkflowDispatchPreservesTriggers(t *testing.T) {
	for _, source := range []string{
		`on:
  workflow_run:
    # zizmor: ignore[dangerous-triggers] - secured with role and fork validation
    workflows: [CI, "Lint: checks"]
    types: [completed]
    branches: [main, "release/**"]
  schedule:
    - cron: '0 0 * * *' # keep the schedule
`,
		`on:
  workflow_dispatch: # keep manual inputs
    inputs:
      note:
        type: string
        default: 2026-01-01
  workflow_run:
    # zizmor: ignore[dangerous-triggers] - secured with role and fork validation
    workflows: [CI]
    types: [completed]
`,
		`on: {workflow_run: {workflows: [CI], types: [completed]}} # zizmor: ignore[dangerous-triggers]
`,
	} {
		t.Run(source, func(t *testing.T) {
			data := &WorkflowData{On: source}
			var before struct{ On map[string]any }
			require.NoError(t, yaml.Unmarshal([]byte(source), &before))
			require.NoError(t, prepareDryRunWorkflowDispatch(data))
			var after struct{ On map[string]any }
			require.NoError(t, yaml.Unmarshal([]byte(data.On), &after))
			require.Contains(t, after.On, "workflow_dispatch")
			for event, config := range before.On {
				assert.Equal(t, config, after.On[event], event)
			}
			assert.Contains(t, data.On, "# zizmor: ignore[dangerous-triggers]")
			if _, exists := before.On["workflow_dispatch"]; exists {
				assert.Equal(t, source, data.On, "existing dispatch should not rewrite triggers")
			}
			first := data.On
			require.NoError(t, prepareDryRunWorkflowDispatch(data))
			assert.Equal(t, first, data.On, "dispatch preparation should be idempotent")
		})
	}
}

func TestDryRunCompiledWorkflowDispatchRoleGate(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "dispatch.md")
	source := "---\non:\n  push: {}\n  roles: all\n  bots: [trusted-bot]\nengine: copilot\nstrict: false\npermissions:\n  contents: read\n---\nDiagnostic.\n"
	require.NoError(t, os.WriteFile(path, []byte(source), 0600))
	compiler := NewCompiler()
	compiler.SetSkipValidation(true)
	compiler.SetApprove(true)
	var normal string
	for _, dryRun := range []bool{false, true, false} {
		compiler.SetDryRun(dryRun)
		require.NoError(t, compiler.CompileWorkflow(path))
		content, err := os.ReadFile(filepath.Join(dir, "dispatch.lock.yml"))
		require.NoError(t, err)
		compiled := string(content)
		if dryRun {
			var workflow struct{ On map[string]any }
			require.NoError(t, yaml.Unmarshal(content, &workflow))
			assert.Contains(t, workflow.On, "workflow_dispatch")
			assert.Contains(t, compiled, `GH_AW_REQUIRED_ROLES: "admin,maintainer"`)
			assert.Contains(t, compiled, "check_membership.cjs")
			assert.Contains(t, compiled, "steps.check_membership.outputs.is_team_member")
			assert.NotContains(t, compiled, "GH_AW_ALLOWED_BOTS:")
		} else if normal == "" {
			normal = compiled
		} else {
			assert.Equal(t, normal, compiled)
		}

	}
}

func TestDryRunCompiledDispatchPreservesDangerousTriggerSuppression(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "hawk.md")
	const source = `---
on:
  workflow_run:
    workflows: [CI]
    types: [completed]
    branches: [main]
engine: copilot
strict: false
permissions:
  contents: read
---
Inspect completed runs.
`
	require.NoError(t, os.WriteFile(path, []byte(source), 0600))
	compiler := NewCompiler()
	compiler.SetSkipValidation(true)
	compiler.SetApprove(true)
	for _, dryRun := range []bool{false, true} {
		compiler.SetDryRun(dryRun)
		require.NoError(t, compiler.CompileWorkflow(path))
		content, err := os.ReadFile(filepath.Join(dir, "hawk.lock.yml"))
		require.NoError(t, err)
		assert.Contains(t, string(content), "# zizmor: ignore[dangerous-triggers] - workflow_run trigger is secured with role and fork validation")
		var workflow struct{ On map[string]any }
		require.NoError(t, yaml.Unmarshal(content, &workflow))
		assert.Equal(t, map[string]any{
			"workflows": []any{"CI"},
			"types":     []any{"completed"},
			"branches":  []any{"main"},
		}, workflow.On["workflow_run"])
		if dryRun {
			assert.Contains(t, workflow.On, "workflow_dispatch")
		}
	}
}
