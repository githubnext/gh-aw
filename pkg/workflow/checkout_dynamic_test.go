//go:build !integration

package workflow

import (
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRuntimeCheckoutMaxReposParsing(t *testing.T) {
	for _, value := range []any{1, int64(2), uint64(3), float64(4)} {
		cfg, err := checkoutConfigFromMap(map[string]any{
			"repo": "${{ inputs.repos }}", "path": "targets", "allowed-repos": "org/*", "max-repos": value,
		})
		require.NoError(t, err)
		require.NotNil(t, cfg.MaxRepos)
		assert.Positive(t, *cfg.MaxRepos)
	}
	for _, value := range []any{0, -1, 1.5, math.NaN(), math.Inf(1), uint64(math.MaxUint64), float64(1 << 53), "2", true, nil, "${{ inputs.max }}"} {
		_, err := checkoutConfigFromMap(map[string]any{
			"repo": "${{ inputs.repos }}", "path": "targets", "allowed-repos": "org/*", "max-repos": value,
		})
		require.ErrorContains(t, err, "checkout.max-repos must be a positive integer")
	}
	_, err := checkoutConfigFromMap(map[string]any{"repository": "org/a", "max-repos": 1})
	require.ErrorContains(t, err, "checkout.max-repos requires checkout.allowed-repos")
	for _, limit := range []int{0, -1} {
		_, err := NewCompiler().CompileToYAML(&WorkflowData{
			CheckoutConfigs: []*CheckoutConfig{{
				Repository: "${{ vars.REPOS }}", Path: "targets", PathExplicit: true, AllowedRepos: []string{"org/*"}, MaxRepos: &limit,
			}},
		}, filepath.Join(t.TempDir(), "worker.md"))
		require.ErrorContains(t, err, "checkout.max-repos must be a positive integer")
	}
}

func TestRuntimeCheckoutMaxReposMerging(t *testing.T) {
	for _, limits := range [][]int{{3, 20}, {20, 3}, {0, 20}, {20, 0}, {20, 30}} {
		configs := make([]*CheckoutConfig, 0, len(limits))
		expected := 30
		for _, limit := range limits {
			cfg := &CheckoutConfig{Repository: "${{ inputs.repos }}", Path: "targets", AllowedRepos: []string{"org/*"}}
			if limit != 0 {
				cfg.MaxRepos = &limit
			} else {
				limit = defaultCheckoutMaxRepos
			}
			expected = min(expected, limit)
			configs = append(configs, cfg)
		}
		manager := NewCheckoutManager(configs)
		require.Len(t, manager.ordered, 1)
		assert.Equal(t, expected, manager.ordered[0].maxRepos, "merged entries must not broaden the effective cap")
	}
}

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
		for _, value := range []string{"${{ inputs.repos }}", "org/${{ vars.REPO }}", "${{\n  vars.REPOS\n}}",
			"${{ github.repository || inputs.repos }}", "${{ github.repository }}\norg/other", "${{ github.repository }}/other"} {
			t.Run(field+"/"+value, func(t *testing.T) {
				_, err := checkoutConfigFromMap(map[string]any{field: value, "path": "targets"})
				require.ErrorContains(t, err, "expressions require checkout.allowed-repos")
			})
		}
		t.Run(field+"/compiler", func(t *testing.T) {
			dir := t.TempDir()
			file := filepath.Join(dir, "worker.md")
			content := "---\non: workflow_dispatch\nengine: copilot\ncheckout:\n  " + field + ": ${{ vars.REPOS }}\n  path: targets\n---\nProcess selected repositories.\n"
			require.NoError(t, os.WriteFile(file, []byte(content), 0o644))
			err := NewCompiler().CompileWorkflow(file)
			require.ErrorContains(t, err, "expressions require checkout.allowed-repos")
			require.NoFileExists(t, filepath.Join(dir, "worker.lock.yml"))
			_, err = NewCompiler().ParseWorkflowString(content, file)
			require.ErrorContains(t, err, "expressions require checkout.allowed-repos")
			_, err = NewCompiler().CompileToYAML(&WorkflowData{
				CheckoutConfigs: []*CheckoutConfig{{Repository: "${{ vars.REPOS }}", Path: "targets", PathExplicit: true}},
			}, file)
			require.ErrorContains(t, err, "expressions require checkout.allowed-repos")
		})
	}
}

func TestRuntimeCheckoutWorkflowRepositoryException(t *testing.T) {
	for _, field := range []string{"repository", "repo"} {
		t.Run(field, func(t *testing.T) {
			for _, value := range []string{"${{ github.repository }}", "${{github.repository}}", " ${{\n github.repository\n}} \n"} {
				cfg, err := checkoutConfigFromMap(map[string]any{field: value})
				require.NoError(t, err)
				assert.Empty(t, cfg.AllowedRepos)
				assert.False(t, cfg.PathExplicit, "the hosting-repository expression must not require a prefix")
				assert.Empty(t, dynamicCheckoutFailureOutput([]*CheckoutConfig{cfg}), "this remains a static single-repository checkout")
			}
			dir := t.TempDir()
			file := filepath.Join(dir, "worker.md")
			content := "---\non: workflow_dispatch\nengine: copilot\ncheckout:\n  " + field + ": ${{ github.repository }}\n---\nProcess the workflow repository.\n"
			require.NoError(t, os.WriteFile(file, []byte(content), 0o644))
			require.NoError(t, NewCompiler().CompileWorkflow(file))
			lock, err := os.ReadFile(filepath.Join(dir, "worker.lock.yml"))
			require.NoError(t, err)
			assert.NotContains(t, string(lock), "checkout_dynamic.cjs")
			data, err := NewCompiler().ParseWorkflowString(content, file)
			require.NoError(t, err)
			_, err = NewCompiler().CompileToYAML(data, file)
			require.NoError(t, err)
		})
	}
	_, err := checkoutConfigFromMap(map[string]any{"repo": "${{ github.repository }}", "allowed-repos": "org/*"})
	require.ErrorContains(t, err, "literal, non-root path prefix", "explicit allowlists still opt into runtime expansion")
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

func TestRuntimeCheckoutRejectsLFS(t *testing.T) {
	for _, repository := range []string{"${{ inputs.repos }}", "org/a\norg/b", "org/a"} {
		_, err := checkoutConfigFromMap(map[string]any{
			"repo": repository, "path": "targets", "allowed-repos": "org/*", "lfs": true,
		})
		require.ErrorContains(t, err, "LFS endpoints can bypass allowed-repos")
		_, err = NewCompiler().CompileToYAML(&WorkflowData{
			CheckoutConfigs: []*CheckoutConfig{{
				Repository: repository, Path: "targets", PathExplicit: true, AllowedRepos: []string{"org/*"}, LFS: true,
			}},
		}, filepath.Join(t.TempDir(), "worker.md"))
		require.ErrorContains(t, err, "LFS endpoints can bypass allowed-repos")
	}
	_, err := checkoutConfigFromMap(map[string]any{"repository": "org/a", "lfs": true})
	require.NoError(t, err, "static checkout LFS must remain supported")
	dir := t.TempDir()
	file := filepath.Join(dir, "worker.md")
	content := "---\non: workflow_dispatch\nengine: copilot\ncheckout:\n  repo: ${{ vars.REPOS }}\n  path: targets\n  allowed-repos: org/*\n  lfs: true\n---\nProcess selected repositories.\n"
	require.NoError(t, os.WriteFile(file, []byte(content), 0o644))
	require.ErrorContains(t, NewCompiler().CompileWorkflow(file), "LFS endpoints can bypass allowed-repos")
	require.NoFileExists(t, filepath.Join(dir, "worker.lock.yml"))
	_, err = NewCompiler().ParseWorkflowString(content, file)
	require.ErrorContains(t, err, "LFS endpoints can bypass allowed-repos")
}

func TestRuntimeCheckoutRejectsSubmodules(t *testing.T) {
	for _, repository := range []string{"${{ inputs.repos }}", "org/a\norg/b", "org/a"} {
		for _, submodules := range []any{true, "true", "recursive", "${{ inputs.submodules }}"} {
			_, err := checkoutConfigFromMap(map[string]any{
				"repo": repository, "path": "targets", "allowed-repos": "org/*", "submodules": submodules,
			})
			require.ErrorContains(t, err, "submodule URLs can bypass allowed-repos")
		}
		for _, submodules := range []string{"true", "recursive", "${{ inputs.submodules }}"} {
			_, err := NewCompiler().CompileToYAML(&WorkflowData{
				CheckoutConfigs: []*CheckoutConfig{{
					Repository: repository, Path: "targets", PathExplicit: true, AllowedRepos: []string{"org/*"}, Submodules: submodules,
				}},
			}, filepath.Join(t.TempDir(), "worker.md"))
			require.ErrorContains(t, err, "submodule URLs can bypass allowed-repos")
		}
		for _, submodules := range []any{false, "false", ""} {
			_, err := checkoutConfigFromMap(map[string]any{
				"repo": repository, "path": "targets", "allowed-repos": "org/*", "submodules": submodules,
			})
			require.NoError(t, err)
		}
	}
	for _, repository := range []string{"org/a", "${{ github.repository }}"} {
		for _, submodules := range []any{true, "recursive"} {
			_, err := checkoutConfigFromMap(map[string]any{"repository": repository, "submodules": submodules})
			require.NoError(t, err, "static checkout submodules must remain supported")
		}
	}
	dir := t.TempDir()
	file := filepath.Join(dir, "worker.md")
	content := "---\non: workflow_dispatch\nengine: copilot\ncheckout:\n  repo: ${{ vars.REPOS }}\n  path: targets\n  allowed-repos: org/*\n  submodules: recursive\n---\nProcess selected repositories.\n"
	require.NoError(t, os.WriteFile(file, []byte(content), 0o644))
	require.ErrorContains(t, NewCompiler().CompileWorkflow(file), "submodule URLs can bypass allowed-repos")
	require.NoFileExists(t, filepath.Join(dir, "worker.lock.yml"))
	_, err := NewCompiler().ParseWorkflowString(content, file)
	require.ErrorContains(t, err, "submodule URLs can bypass allowed-repos")
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
	assert.Contains(t, steps, `GH_AW_DYNAMIC_MAX_REPOS: "10"`)
	assert.Contains(t, steps, `GH_AW_DYNAMIC_ALLOW_PRIVATE_REPOS: "false"`)
	assert.Contains(t, steps, "GH_AW_DYNAMIC_TOKEN: ${{ secrets.CHECKOUT_TOKEN }}")
	assert.Contains(t, steps, `GH_AW_DYNAMIC_PERSIST: "false"`)
	assert.Contains(t, steps, "await finish({ cleanCredentials: true });")
	assert.NotContains(t, steps, "repository: ${{ inputs.repos }}")
	assert.Contains(t, steps, `GH_AW_DYNAMIC_CHECKOUT_ACTION: "actions/checkout@`+strings.Repeat("a", 40)+`"`)
	assert.Empty(t, manager.credentialSubrepositories())
	manifest := strings.Join(manager.GenerateCheckoutManifestStep(pin), "")
	assert.Contains(t, manifest, `GH_AW_DYNAMIC_CHECKOUT_INDEXES: "0"`)
	assert.NotContains(t, manifest, "GH_AW_CHECKOUT_REPO_0")

	manager.SetKeepCredentialsForPush(true)
	steps = strings.Join(manager.GenerateAdditionalCheckoutSteps(pin), "")
	assert.Contains(t, steps, `GH_AW_DYNAMIC_PERSIST: "true"`)
	assert.Contains(t, steps, "await finish({ cleanCredentials: false });", "safe-output write credentials must remain available")
	prompt := buildCheckoutsPromptContent([]*CheckoutConfig{{
		Repository: "${{ inputs.repos }}", Path: "targets", AllowedRepos: []string{"org/*"},
	}})
	assert.Contains(t, prompt, "targets/<owner>/<repo>")
	assert.NotContains(t, prompt, "${{ inputs.repos }}")

	manager = NewCheckoutManager([]*CheckoutConfig{{
		Repository: "${{ inputs.repos }}", Path: "targets", AllowedRepos: []string{"org/*"}, CleanGitCredentials: true,
	}})
	steps = strings.Join(manager.GenerateAdditionalCheckoutSteps(pin), "")
	assert.Contains(t, steps, `GH_AW_DYNAMIC_PERSIST: "true"`)
	assert.Contains(t, steps, "await finish({ cleanCredentials: true });", "persisted agent credentials must be cleaned using explicit checkout paths")
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
    max-repos: 3
    allow-private-repos: true
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
	assert.Equal(t, 6, strings.Count(yaml, `GH_AW_DYNAMIC_MAX_REPOS: "3"`), "preparation, checkout and finalization in both jobs must share the cap")
	assert.Equal(t, 6, strings.Count(yaml, `GH_AW_DYNAMIC_ALLOW_PRIVATE_REPOS: "true"`), "agent and safe_outputs must enforce the same private-repository policy")
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
	for _, value := range []any{"org/*", "*/repo", "org/repo", []any{"org/*", "*/repo"}, "${{ vars.ALLOWED_REPOS }}"} {
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
engine: copilot
checkout:
  - repository: ${{ inputs.repos }}
    path: targets
    allowed-repos: ${{ vars.ALLOWED_REPOS }}
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
	assert.Contains(t, string(lock), `GH_AW_DYNAMIC_ALLOWED_REPOS: "${{ vars.ALLOWED_REPOS }}"`)
	assert.Contains(t, string(lock), "GH_AW_CHECKOUT_ERROR: ${{ needs.agent.outputs.checkout_error }}")
}
