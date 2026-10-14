//go:build !integration

package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/goccy/go-yaml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckoutMultiRequiresLiteralBoolean(t *testing.T) {
	for _, value := range []any{"true", "${{ vars.MULTI }}", 1, nil} {
		_, err := checkoutConfigFromMap(map[string]any{"multi": value})
		require.ErrorContains(t, err, "checkout.multi must be a literal boolean")
	}
}

func TestSingleCheckoutDoesNotApplyMultiRestrictions(t *testing.T) {
	for _, multi := range []any{nil, false} {
		for _, repository := range []string{"org/repo", "${{ inputs.repo }}", "org/${{ vars.REPO }}", "${{ github.repository }}"} {
			values := map[string]any{
				"repo": repository, "path": ".", "current": true, "wiki": true,
				"lfs": true, "submodules": "recursive",
				"allowed-repos": "${{ inputs.ignored_policy }}",
				"github-app":    map[string]any{"client-id": "123", "private-key": "${{ secrets.KEY }}"},
			}
			if multi != nil {
				values["multi"] = multi
			}
			cfg, err := checkoutConfigFromMap(values)
			require.NoError(t, err)
			assert.False(t, cfg.Multi)
			assert.Empty(t, dynamicCheckoutFailureOutput([]*CheckoutConfig{cfg}))
			pin := func(action string) string { return action + "@pin" }
			manager := NewCheckoutManager([]*CheckoutConfig{cfg})
			steps := strings.Join(manager.GenerateAdditionalCheckoutSteps(pin), "")
			assert.NotContains(t, steps, "uses: ./.gh-aw/checkouts/")
			assert.Contains(t, steps, "submodules: recursive")
			assert.Contains(t, steps, "lfs: true")
			assert.NotContains(t, buildCheckoutsPromptContent([]*CheckoutConfig{cfg}), "<owner>/<repo>")
			if containsExpression(repository) {
				assert.Contains(t, steps, "validateSingleRepository")
				assert.Less(t, strings.Index(steps, "validateSingleRepository"), strings.Index(steps, "uses: actions/checkout@"))
				assert.Contains(t, steps, "GH_AW_CHECKOUT_REPOSITORY:")
			}
		}
	}
}

func TestSingleCheckoutRejectsLiteralLists(t *testing.T) {
	for _, field := range []string{"repo", "repository"} {
		for _, multi := range []bool{false, true} {
			values := map[string]any{field: "org/a\norg/b", "path": "targets", "allowed-repos": "org/*", "multi": multi}
			cfg, err := checkoutConfigFromMap(values)
			if multi {
				require.NoError(t, err)
				assert.True(t, cfg.Multi)
			} else {
				require.ErrorContains(t, err, "single repository unless checkout.multi is true")
			}
		}
	}
}

func TestCompileSingleCheckoutExpression(t *testing.T) {
	for _, multi := range []string{"", "  multi: false\n"} {
		file := filepath.Join(t.TempDir(), "worker.md")
		content := "---\non: workflow_dispatch\nengine: copilot\ncheckout:\n" + multi +
			"  repo: ${{ vars.REPO }}\n  path: .\n  current: true\n  lfs: true\n  submodules: recursive\n" +
			"safe-outputs:\n  create-pull-request:\n    target-repo: org/repo\n---\nProcess the selected repository.\n"
		compiler := NewCompiler()
		require.NoError(t, os.WriteFile(file, []byte(content), 0o644))
		require.NoError(t, compiler.CompileWorkflow(file))
		lock, err := os.ReadFile(strings.TrimSuffix(file, ".md") + ".lock.yml")
		require.NoError(t, err)
		compiled := string(lock)
		data, err := compiler.ParseWorkflowString(content, file)
		require.NoError(t, err)
		_, err = compiler.CompileToYAML(data, file)
		require.NoError(t, err)
		var workflow struct {
			Jobs map[string]struct {
				Steps []struct {
					Name string
				}
			}
		}
		require.NoError(t, yaml.Unmarshal([]byte(compiled), &workflow))
		validations := 0
		for _, job := range workflow.Jobs {
			for _, step := range job.Steps {
				if strings.HasPrefix(step.Name, "Validate single checkout repository") {
					validations++
				}
			}
		}
		assert.Equal(t, 2, validations, "agent and safe-output jobs must both validate single-repo expressions")
		assert.NotContains(t, compiled, "GH_AW_DYNAMIC_ALLOWED_REPOS")
		assert.Contains(t, compiled, "repository: ${{ vars.REPO }}")
	}
}

func TestCheckoutModesDoNotMerge(t *testing.T) {
	manager := NewCheckoutManager([]*CheckoutConfig{
		{Repository: "${{ vars.REPOS }}", Path: "targets"},
		{Repository: "${{ vars.REPOS }}", Path: "targets", Multi: true, AllowedRepos: []string{"org/*"}},
	})
	require.Len(t, manager.ordered, 2)
	assert.False(t, manager.ordered[0].key.multi)
	assert.True(t, manager.ordered[1].key.multi)
}
