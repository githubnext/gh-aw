//go:build !integration

package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseRuntimeCheckout(t *testing.T) {
	cfg, err := checkoutConfigFromMap(map[string]any{
		"repo": "${{ inputs.repos }}", "path": "targets", "allow-repos": "${{ vars.ALLOWED_REPOS }}",
	})
	require.NoError(t, err)
	assert.Equal(t, "${{ inputs.repos }}", cfg.Repository)
	assert.Equal(t, []string{"${{ vars.ALLOWED_REPOS }}"}, cfg.AllowRepos)

	for name, values := range map[string]map[string]any{
		"both aliases":           {"repository": "org/repo", "repo": "org/other"},
		"list without allowlist": {"repository": "org/a\norg/b", "path": "targets"},
		"wrong alias type":       {"repo": true},
		"empty allowlist":        {"repository": "${{ inputs.repos }}", "path": "targets", "allow-repos": []any{}},
		"wrong allowlist":        {"repository": "${{ inputs.repos }}", "path": "targets", "allow-repos": true},
		"invalid entry":          {"repository": "${{ inputs.repos }}", "path": "targets", "allow-repos": []any{true}},
		"missing repos":          {"path": "targets", "allow-repos": "*"},
		"missing prefix":         {"repository": "${{ inputs.repos }}", "allow-repos": "*"},
		"root prefix":            {"repository": "${{ inputs.repos }}", "path": ".", "allow-repos": "*"},
		"expression prefix":      {"repository": "${{ inputs.repos }}", "path": "${{ inputs.path }}", "allow-repos": "*"},
		"ambiguous current":      {"repository": "${{ inputs.repos }}", "path": "targets", "allow-repos": "*", "current": true},
		"wiki":                   {"repository": "${{ inputs.repos }}", "path": "targets", "allow-repos": "*", "wiki": true},
		"implicit app": {"repository": "${{ inputs.repos }}", "path": "targets", "allow-repos": "*",
			"github-app": map[string]any{"client-id": "123", "private-key": "${{ secrets.KEY }}"}},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := checkoutConfigFromMap(values)
			require.Error(t, err)
		})
	}
}

func TestGenerateRuntimeCheckoutSteps(t *testing.T) {
	manager := NewCheckoutManager([]*CheckoutConfig{{
		Repository: "${{ inputs.repos }}", Path: "targets", AllowRepos: []string{"org/*"},
		GitHubToken: "${{ secrets.CHECKOUT_TOKEN }}", Fetch: []string{"main"},
	}})
	pin := func(action string) string { return action + "@" + strings.Repeat("a", 40) + " # v7" }
	steps := strings.Join(manager.GenerateAdditionalCheckoutSteps(pin), "")
	assert.Contains(t, steps, "uses: ./.gh-aw/checkouts/0")
	assert.Contains(t, steps, `GH_AW_DYNAMIC_REPOSITORIES: "${{ inputs.repos }}"`)
	assert.Contains(t, steps, `GH_AW_DYNAMIC_ALLOW_REPOS: "org/*"`)
	assert.Contains(t, steps, "GH_AW_DYNAMIC_TOKEN: ${{ secrets.CHECKOUT_TOKEN }}")
	assert.Contains(t, steps, `GH_AW_DYNAMIC_PERSIST: "false"`)
	assert.NotContains(t, steps, "repository: ${{ inputs.repos }}")
	assert.Contains(t, steps, `GH_AW_DYNAMIC_CHECKOUT_ACTION: "actions/checkout@`+strings.Repeat("a", 40)+`"`)
	assert.Empty(t, manager.credentialSubrepositories())
	manifest := strings.Join(manager.GenerateCheckoutManifestStep(pin), "")
	assert.Contains(t, manifest, `GH_AW_DYNAMIC_CHECKOUT_INDEXES: "0"`)
	assert.NotContains(t, manifest, "GH_AW_CHECKOUT_REPO_0")

	manager.SetKeepCredentialsForPush(true)
	steps = strings.Join(manager.GenerateAdditionalCheckoutSteps(pin), "")
	assert.Contains(t, steps, `GH_AW_DYNAMIC_PERSIST: "true"`)
	prompt := buildCheckoutsPromptContent([]*CheckoutConfig{{
		Repository: "${{ inputs.repos }}", Path: "targets", AllowRepos: []string{"org/*"},
	}})
	assert.Contains(t, prompt, "targets/<owner>/<repo>")
	assert.NotContains(t, prompt, "${{ inputs.repos }}")
}

func TestCompileRuntimeCheckout(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "worker.md")
	require.NoError(t, os.WriteFile(file, []byte(`---
on:
  schedule: daily
permissions:
  contents: read
engine: copilot
checkout:
  - repo: ${{ vars.WORKER_REPOS }}
    path: targets
    allow-repos: ["org/*"]
safe-outputs:
  create-pull-request:
    target-repo: "*"
    allowed-repos: ["org/*"]
---
Process each selected repository under targets.
`), 0o644))
	compiler := NewCompiler()
	compiler.SetWorkflowIdentifier("worker.md")
	require.NoError(t, compiler.CompileWorkflow(file))
	lock, err := os.ReadFile(filepath.Join(dir, "worker.lock.yml"))
	require.NoError(t, err)
	yaml := string(lock)
	assert.Contains(t, yaml, `GH_AW_DYNAMIC_REPOSITORIES: "${{ vars.WORKER_REPOS }}"`)
	assert.Contains(t, yaml, "checkout_dynamic.cjs")
	assert.Equal(t, 2, strings.Count(yaml, "uses: ./.gh-aw/checkouts/0"), "agent and safe_outputs must share checkout layout")
	assert.NotContains(t, yaml, "repository: ${{ vars.WORKER_REPOS }}")
}

func TestRuntimeCheckoutAllowlistMerging(t *testing.T) {
	manager := NewCheckoutManager([]*CheckoutConfig{
		{Repository: "${{ inputs.repos }}", Path: "targets", AllowRepos: []string{"org/a"}},
		{Repository: "${{ inputs.repos }}", Path: "targets", AllowRepos: []string{"*"}},
	})
	require.Len(t, manager.ordered, 1)
	assert.Equal(t, []string{"org/a"}, manager.ordered[0].allowRepos, "imports must not broaden the main workflow's allowlist")
}

func TestCompileRuntimeCheckoutInputExpressions(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "worker.md")
	require.NoError(t, os.WriteFile(file, []byte(`---
on:
  workflow_dispatch:
    inputs:
      repos:
        type: string
        required: true
      allowed:
        type: string
        required: true
engine: copilot
checkout:
  - repository: ${{ inputs.repos }}
    path: targets
    allow-repos: ${{ inputs.allowed }}
safe-outputs:
  noop:
    report-as-issue: false
---
Process the selected repositories.
`), 0o644))
	require.NoError(t, NewCompiler().CompileWorkflow(file))
	lock, err := os.ReadFile(filepath.Join(dir, "worker.lock.yml"))
	require.NoError(t, err)
	assert.Contains(t, string(lock), `GH_AW_DYNAMIC_REPOSITORIES: "${{ inputs.repos }}"`)
	assert.Contains(t, string(lock), `GH_AW_DYNAMIC_ALLOW_REPOS: "${{ inputs.allowed }}"`)
}
