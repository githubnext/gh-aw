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
		"repo": "${{ inputs.repos }}", "path": "targets", "allowed-repos": "${{ vars.ALLOWED_REPOS }}",
	})
	require.NoError(t, err)
	assert.Equal(t, "${{ inputs.repos }}", cfg.Repository)
	assert.Equal(t, []string{"${{ vars.ALLOWED_REPOS }}"}, cfg.AllowedRepos)

	for name, values := range map[string]map[string]any{
		"both aliases":           {"repository": "org/repo", "repo": "org/other"},
		"list without allowlist": {"repository": "org/a\norg/b", "path": "targets"},
		"wrong alias type":       {"repo": true},
		"old allowlist key":      {"repository": "${{ inputs.repos }}", "path": "targets", "allow-repos": "org/*"},
		"empty allowlist":        {"repository": "${{ inputs.repos }}", "path": "targets", "allowed-repos": []any{}},
		"wrong allowlist":        {"repository": "${{ inputs.repos }}", "path": "targets", "allowed-repos": true},
		"invalid entry":          {"repository": "${{ inputs.repos }}", "path": "targets", "allowed-repos": []any{true}},
		"missing repos":          {"path": "targets", "allowed-repos": "org/*"},
		"missing prefix":         {"repository": "${{ inputs.repos }}", "allowed-repos": "org/*"},
		"root prefix":            {"repository": "${{ inputs.repos }}", "path": ".", "allowed-repos": "org/*"},
		"expression prefix":      {"repository": "${{ inputs.repos }}", "path": "${{ inputs.path }}", "allowed-repos": "org/*"},
		"ambiguous current":      {"repository": "${{ inputs.repos }}", "path": "targets", "allowed-repos": "org/*", "current": true},
		"wiki":                   {"repository": "${{ inputs.repos }}", "path": "targets", "allowed-repos": "org/*", "wiki": true},
		"implicit app": {"repository": "${{ inputs.repos }}", "path": "targets", "allowed-repos": "org/*",
			"github-app": map[string]any{"client-id": "123", "private-key": "${{ secrets.KEY }}"}},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := checkoutConfigFromMap(values)
			require.Error(t, err)
		})
	}
}

func TestRuntimeCheckoutRepositoryWildcards(t *testing.T) {
	for _, field := range []string{"repo", "repository"} {
		t.Run(field, func(t *testing.T) {
			for _, repository := range []string{"*", "org/*", "*/repo", "org/repo*", "org/repo?", "org/[ab]", "org/a\norg/*", "${{ inputs.owner }}/*"} {
				for _, allowlist := range []bool{false, true} {
					values := map[string]any{field: repository, "path": "targets"}
					if allowlist {
						values["allowed-repos"] = "org/*"
					}
					_, err := checkoutConfigFromMap(values)
					require.ErrorContains(t, err, "wildcards are only supported in checkout.allowed-repos")
				}
			}
			for _, repository := range []string{"org/repo", "org/a\norg/b", "${{ inputs.repos }}", "${{ join(needs.select.outputs.repos.*.name, '\n') }}"} {
				_, err := checkoutConfigFromMap(map[string]any{
					field: repository, "path": "targets", "allowed-repos": "org/*",
				})
				require.NoError(t, err)
			}
		})
	}
}

func TestCompileRuntimeCheckoutRequiresAllowlist(t *testing.T) {
	for _, field := range []string{"repository", "repo"} {
		for _, value := range []string{"${{ inputs.repos }}", "${{ github.repository }}", "org/${{ vars.REPO }}", "${{\n  vars.REPOS\n}}"} {
			t.Run(field+"/"+value, func(t *testing.T) {
				_, err := checkoutConfigFromMap(map[string]any{field: value, "path": "targets"})
				require.ErrorContains(t, err, "expressions require checkout.allowed-repos")
			})
		}
		t.Run(field+"/compiler", func(t *testing.T) {
			dir := t.TempDir()
			file := filepath.Join(dir, "worker.md")
			content := "---\non: workflow_dispatch\nengine: copilot\ncheckout:\n  " + field + ": ${{ github.repository }}\n  path: targets\n---\nProcess selected repositories.\n"
			require.NoError(t, os.WriteFile(file, []byte(content), 0o644))
			err := NewCompiler().CompileWorkflow(file)
			require.ErrorContains(t, err, "expressions require checkout.allowed-repos")
			require.NoFileExists(t, filepath.Join(dir, "worker.lock.yml"))
			_, err = NewCompiler().ParseWorkflowString(content, file)
			require.ErrorContains(t, err, "expressions require checkout.allowed-repos")
			_, err = NewCompiler().CompileToYAML(&WorkflowData{
				CheckoutConfigs: []*CheckoutConfig{{Repository: "${{ github.repository }}", Path: "targets", PathExplicit: true}},
			}, file)
			require.ErrorContains(t, err, "expressions require checkout.allowed-repos")
		})
	}
}

func TestCompileRuntimeCheckoutRejectsInvalidImportedAllowlist(t *testing.T) {
	dir := t.TempDir()
	shared := "---\ncheckout:\n  repo: ${{ vars.REPOS }}\n  path: targets\n---\nShared checkout.\n"
	require.NoError(t, os.WriteFile(filepath.Join(dir, "shared.md"), []byte(shared), 0o644))
	main := "---\non: workflow_dispatch\nengine: copilot\nimports:\n  - ./shared.md\n---\nProcess selected repositories.\n"
	file := filepath.Join(dir, "worker.md")
	require.NoError(t, os.WriteFile(file, []byte(main), 0o644))
	err := NewCompiler().CompileWorkflow(file)
	require.ErrorContains(t, err, "expressions require checkout.allowed-repos")
	require.NoFileExists(t, filepath.Join(dir, "worker.lock.yml"))
}

func TestRuntimeCheckoutFailureOutputs(t *testing.T) {
	configs := []*CheckoutConfig{
		{Repository: "org/static", Path: "static"},
		{Repository: "${{ vars.REPOS }}", Path: "targets", AllowedRepos: []string{"org/*"}},
		{Repository: "${{ vars.REPOS }}", Path: "targets", AllowedRepos: []string{"org/*"}},
		{Repository: "${{ vars.OTHER_REPOS }}", Path: "others", AllowedRepos: []string{"other/*"}},
	}
	expected := "${{ steps.checkout-validate-1.outputs.checkout_error || steps.checkout-validate-2.outputs.checkout_error }}"
	assert.Equal(t, expected, dynamicCheckoutFailureOutput(configs))
	assert.Equal(t, expected, NewCompiler().buildMainJobOutputs(&WorkflowData{CheckoutConfigs: configs})["checkout_error"])
	assert.Empty(t, dynamicCheckoutFailureOutput(nil))
	assert.Empty(t, dynamicCheckoutFailureOutput(configs[:1]))
	assert.Equal(t, []string{"          GH_AW_CHECKOUT_ERROR: ${{ needs.worker.outputs.checkout_error }}\n"},
		buildDynamicCheckoutFailureEnv(&WorkflowData{CheckoutConfigs: configs}, "worker"))
	assert.Empty(t, buildDynamicCheckoutFailureEnv(&WorkflowData{}, "worker"))
}

func TestGenerateRuntimeCheckoutSteps(t *testing.T) {
	manager := NewCheckoutManager([]*CheckoutConfig{{
		Repository: "${{ inputs.repos }}", Path: "targets", AllowedRepos: []string{"org/*"},
		GitHubToken: "${{ secrets.CHECKOUT_TOKEN }}", Fetch: []string{"main"},
	}})
	pin := func(action string) string { return action + "@" + strings.Repeat("a", 40) + " # v7" }
	steps := strings.Join(manager.GenerateAdditionalCheckoutSteps(pin), "")
	assert.Contains(t, steps, "uses: ./.gh-aw/checkouts/0")
	assert.Contains(t, steps, "id: checkout-validate-0")
	assert.Contains(t, steps, `GH_AW_DYNAMIC_REPOSITORIES: "${{ inputs.repos }}"`)
	assert.Contains(t, steps, `GH_AW_DYNAMIC_ALLOWED_REPOS: "org/*"`)
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
		Repository: "${{ inputs.repos }}", Path: "targets", AllowedRepos: []string{"org/*"},
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
    allowed-repos: ["org/*"]
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
	assert.Equal(t, 2, strings.Count(yaml, "checkout_error: ${{ steps.checkout-validate-0.outputs.checkout_error }}"))
	assert.Contains(t, yaml, "GH_AW_CHECKOUT_ERROR: ${{ needs.agent.outputs.checkout_error || needs.safe_outputs.outputs.checkout_error }}")
	_, failureHandler, found := strings.Cut(yaml, "        id: handle_agent_failure\n")
	require.True(t, found)
	failureHandler, _, _ = strings.Cut(failureHandler, "\n      - name:")
	assert.Contains(t, failureHandler, "GH_AW_CHECKOUT_ERROR: ${{ needs.agent.outputs.checkout_error || needs.safe_outputs.outputs.checkout_error }}",
		"checkout rejection must reach the agent failure handler, not another conclusion step")
}

func TestRuntimeCheckoutAllowlistMerging(t *testing.T) {
	manager := NewCheckoutManager([]*CheckoutConfig{
		{Repository: "${{ inputs.repos }}", Path: "targets", AllowedRepos: []string{"org/a"}},
		{Repository: "${{ inputs.repos }}", Path: "targets", AllowedRepos: []string{"org/*"}},
	})
	require.Len(t, manager.ordered, 1)
	assert.Equal(t, []string{"org/a"}, manager.ordered[0].allowedRepos, "imports must not broaden the main workflow's allowlist")
}

func TestRuntimeCheckoutAllowedRepoPatterns(t *testing.T) {
	for _, value := range []any{"*", " * ", "*/*", "**/**", "org/*\n*", "org/*,*", []any{"org/*", "*"}} {
		_, err := checkoutConfigFromMap(map[string]any{
			"repository": "${{ inputs.repos }}", "path": "targets", "allowed-repos": value,
		})
		require.ErrorContains(t, err, "must not allow every repository")
	}
	for _, value := range []any{"org/*", "*/repo", "org/repo", []any{"org/*", "*/repo"}, "${{ inputs.allowed }}"} {
		_, err := checkoutConfigFromMap(map[string]any{
			"repository": "${{ inputs.repos }}", "path": "targets", "allowed-repos": value,
		})
		require.NoError(t, err)
	}
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
    allowed-repos: ${{ inputs.allowed }}
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
	assert.Contains(t, string(lock), `GH_AW_DYNAMIC_ALLOWED_REPOS: "${{ inputs.allowed }}"`)
	assert.Contains(t, string(lock), "GH_AW_CHECKOUT_ERROR: ${{ needs.agent.outputs.checkout_error }}")
}
