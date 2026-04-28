//go:build !integration

package yamlpostcheck

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// buildTree builds a minimal workflow tree with a single job containing the
// given steps for use in table-driven tests.
func buildTree(steps []any) map[string]any {
	return map[string]any{
		"jobs": map[string]any{
			"agent": map[string]any{
				"steps": steps,
			},
		},
	}
}

// buildStep builds a step map from the provided key-value pairs (alternating
// string keys and any values) for use in table-driven tests.
func buildStep(kvs ...any) map[string]any {
	if len(kvs)%2 != 0 {
		panic("buildStep requires an even number of arguments (key-value pairs)")
	}
	step := make(map[string]any)
	for i := 0; i+1 < len(kvs); i += 2 {
		key, ok := kvs[i].(string)
		if !ok {
			panic(fmt.Sprintf("buildStep: key at position %d must be a string, got %T", i, kvs[i]))
		}
		step[key] = kvs[i+1]
	}
	return step
}

func TestSecretsInRunChecker_Name(t *testing.T) {
	c := NewSecretsInRunChecker()
	assert.Equal(t, "secrets-in-run", c.Name(), "checker name should match")
}

func TestSecretsInRunChecker_NoJobs(t *testing.T) {
	c := NewSecretsInRunChecker()
	tree := map[string]any{}
	result, err := c.Check(tree)
	require.NoError(t, err, "empty tree should not error")
	assert.False(t, result.Changed, "empty tree should not be changed")
	assert.Empty(t, result.Fixes, "empty tree should have no fixes")
}

func TestSecretsInRunChecker_NoRunStep(t *testing.T) {
	c := NewSecretsInRunChecker()
	tree := buildTree([]any{
		buildStep("name", "checkout", "uses", "actions/checkout@v4"),
	})
	result, err := c.Check(tree)
	require.NoError(t, err, "uses-only step should not error")
	assert.False(t, result.Changed, "uses-only step should not be changed")
}

func TestSecretsInRunChecker_SecretInRun(t *testing.T) {
	c := NewSecretsInRunChecker()
	step := buildStep(
		"name", "Call API",
		"run", `curl -H "Authorization: Bearer ${{ secrets.API_TOKEN }}" https://example.com`,
	)
	tree := buildTree([]any{step})

	result, err := c.Check(tree)
	require.NoError(t, err, "should not error on fixable step")
	assert.True(t, result.Changed, "tree should be marked as changed")
	require.Len(t, result.Fixes, 1, "should have exactly one fix")
	assert.Contains(t, result.Fixes[0], "API_TOKEN", "fix should mention the env var name")

	// The run: script should now reference $API_TOKEN, not the expression.
	updatedStep := tree["jobs"].(map[string]any)["agent"].(map[string]any)["steps"].([]any)[0].(map[string]any)
	assert.Contains(t, updatedStep["run"].(string), "$API_TOKEN",
		"run: should use shell variable reference")
	assert.NotContains(t, updatedStep["run"].(string), "${{",
		"run: should not contain any expression syntax")

	// The env: map should contain the mapping.
	envMap, ok := updatedStep["env"].(map[string]any)
	require.True(t, ok, "step should have an env: map after fix")
	assert.Equal(t, "${{ secrets.API_TOKEN }}", envMap["API_TOKEN"],
		"env: should map API_TOKEN to the secret expression")
}

func TestSecretsInRunChecker_GithubToken(t *testing.T) {
	c := NewSecretsInRunChecker()
	step := buildStep(
		"name", "Auth",
		"run", `gh auth login --with-token <<< "${{ github.token }}"`,
	)
	tree := buildTree([]any{step})

	result, err := c.Check(tree)
	require.NoError(t, err, "should not error")
	assert.True(t, result.Changed, "tree should be changed")

	updatedStep := tree["jobs"].(map[string]any)["agent"].(map[string]any)["steps"].([]any)[0].(map[string]any)
	envMap, ok := updatedStep["env"].(map[string]any)
	require.True(t, ok, "step should have env: map")
	assert.Equal(t, "${{ github.token }}", envMap["GITHUB_TOKEN"],
		"GITHUB_TOKEN should be mapped to github.token expression")
	assert.Contains(t, updatedStep["run"].(string), "$GITHUB_TOKEN",
		"run: should use $GITHUB_TOKEN")
}

func TestSecretsInRunChecker_EnvGithubToken(t *testing.T) {
	c := NewSecretsInRunChecker()
	step := buildStep(
		"name", "Push",
		"run", `git push https://x-access-token:${{ env.GITHUB_TOKEN }}@github.com/owner/repo.git`,
	)
	tree := buildTree([]any{step})

	result, err := c.Check(tree)
	require.NoError(t, err, "should not error")
	assert.True(t, result.Changed, "tree should be changed")

	updatedStep := tree["jobs"].(map[string]any)["agent"].(map[string]any)["steps"].([]any)[0].(map[string]any)
	envMap, ok := updatedStep["env"].(map[string]any)
	require.True(t, ok, "step should have env: map")
	assert.Equal(t, "${{ env.GITHUB_TOKEN }}", envMap["GITHUB_TOKEN"],
		"GITHUB_TOKEN env var should be set")
	assert.Contains(t, updatedStep["run"].(string), "$GITHUB_TOKEN",
		"run: should reference $GITHUB_TOKEN")
	assert.NotContains(t, updatedStep["run"].(string), "${{",
		"run: should not contain expression syntax")
}

func TestSecretsInRunChecker_MultipleSecrets(t *testing.T) {
	c := NewSecretsInRunChecker()
	step := buildStep(
		"name", "Deploy",
		"run", `
deploy.sh \
  --token "${{ secrets.DEPLOY_TOKEN }}" \
  --key "${{ secrets.SSH_KEY }}"`,
	)
	tree := buildTree([]any{step})

	result, err := c.Check(tree)
	require.NoError(t, err, "should not error")
	assert.True(t, result.Changed, "tree should be changed")
	assert.Len(t, result.Fixes, 2, "should have two fixes (one per unique secret)")

	updatedStep := tree["jobs"].(map[string]any)["agent"].(map[string]any)["steps"].([]any)[0].(map[string]any)
	envMap, ok := updatedStep["env"].(map[string]any)
	require.True(t, ok, "step should have env: map")
	assert.Equal(t, "${{ secrets.DEPLOY_TOKEN }}", envMap["DEPLOY_TOKEN"], "DEPLOY_TOKEN should be set")
	assert.Equal(t, "${{ secrets.SSH_KEY }}", envMap["SSH_KEY"], "SSH_KEY should be set")
}

func TestSecretsInRunChecker_DuplicateExpression(t *testing.T) {
	// Same expression appears multiple times — should get a single env var and
	// both occurrences replaced.
	c := NewSecretsInRunChecker()
	step := buildStep(
		"name", "Double use",
		"run", `echo "${{ secrets.TOKEN }}" && echo "${{ secrets.TOKEN }}"`,
	)
	tree := buildTree([]any{step})

	result, err := c.Check(tree)
	require.NoError(t, err, "should not error")
	assert.True(t, result.Changed, "tree should be changed")
	// Two occurrences of the same expression → still only one fix entry.
	assert.Len(t, result.Fixes, 1, "duplicate expression should produce one fix entry")

	updatedStep := tree["jobs"].(map[string]any)["agent"].(map[string]any)["steps"].([]any)[0].(map[string]any)
	// Both occurrences should be replaced.
	runScript := updatedStep["run"].(string)
	assert.NotContains(t, runScript, "${{", "run: should have no remaining expressions")
	assert.Equal(t, `echo "$TOKEN" && echo "$TOKEN"`, runScript, "both occurrences should be replaced")
}

func TestSecretsInRunChecker_ExistingEnvMapPreserved(t *testing.T) {
	// A step that already has some env: vars — they should be kept.
	c := NewSecretsInRunChecker()
	step := buildStep(
		"name", "With existing env",
		"env", map[string]any{"EXISTING": "value"},
		"run", `echo "${{ secrets.MY_SECRET }}"`,
	)
	tree := buildTree([]any{step})

	result, err := c.Check(tree)
	require.NoError(t, err, "should not error")
	assert.True(t, result.Changed, "tree should be changed")

	updatedStep := tree["jobs"].(map[string]any)["agent"].(map[string]any)["steps"].([]any)[0].(map[string]any)
	envMap, ok := updatedStep["env"].(map[string]any)
	require.True(t, ok, "env: map should be present")
	assert.Equal(t, "value", envMap["EXISTING"], "pre-existing env var should be preserved")
	assert.Equal(t, "${{ secrets.MY_SECRET }}", envMap["MY_SECRET"], "new env var should be added")
}

func TestSecretsInRunChecker_AlreadySafeStep(t *testing.T) {
	// Step that already uses env: correctly — should not be modified.
	c := NewSecretsInRunChecker()
	step := buildStep(
		"name", "Already safe",
		"env", map[string]any{"API_TOKEN": "${{ secrets.API_TOKEN }}"},
		"run", `curl -H "Authorization: Bearer $API_TOKEN" https://example.com`,
	)
	tree := buildTree([]any{step})

	result, err := c.Check(tree)
	require.NoError(t, err, "should not error on already-safe step")
	assert.False(t, result.Changed, "already-safe step should not be modified")
	assert.Empty(t, result.Fixes, "no fixes should be needed")
}

func TestSecretsInRunChecker_EnvVarNameCollision(t *testing.T) {
	// An existing env: entry with the same key but a different value — the
	// checker must use a disambiguated name.
	c := NewSecretsInRunChecker()
	step := buildStep(
		"name", "Collision",
		"env", map[string]any{"TOKEN": "static-value"},
		"run", `curl -H "Authorization: Bearer ${{ secrets.TOKEN }}" https://example.com`,
	)
	tree := buildTree([]any{step})

	result, err := c.Check(tree)
	require.NoError(t, err, "should not error")
	assert.True(t, result.Changed, "tree should be changed")

	updatedStep := tree["jobs"].(map[string]any)["agent"].(map[string]any)["steps"].([]any)[0].(map[string]any)
	envMap, ok := updatedStep["env"].(map[string]any)
	require.True(t, ok, "env: map should be present")

	// The original static-value entry must be preserved.
	assert.Equal(t, "static-value", envMap["TOKEN"], "original TOKEN entry should be untouched")

	// A disambiguated name should have been created.
	assert.Equal(t, "${{ secrets.TOKEN }}", envMap["TOKEN_1"],
		"disambiguated env var TOKEN_1 should hold the secret expression")

	// The run: script should use the disambiguated variable.
	assert.Contains(t, updatedStep["run"].(string), "$TOKEN_1",
		"run: should reference $TOKEN_1")
}

func TestSecretsInRunChecker_MultipleJobs(t *testing.T) {
	// Secrets in steps across multiple jobs should all be fixed.
	c := NewSecretsInRunChecker()
	tree := map[string]any{
		"jobs": map[string]any{
			"job-a": map[string]any{
				"steps": []any{
					buildStep("name", "A step", "run", `echo "${{ secrets.SECRET_A }}"`),
				},
			},
			"job-b": map[string]any{
				"steps": []any{
					buildStep("name", "B step", "run", `echo "${{ secrets.SECRET_B }}"`),
				},
			},
		},
	}

	result, err := c.Check(tree)
	require.NoError(t, err, "should not error")
	assert.True(t, result.Changed, "tree should be changed")
	assert.Len(t, result.Fixes, 2, "should have two fixes (one per job)")
}

func TestSecretsInRunChecker_NonStringRunValue(t *testing.T) {
	// run: with a non-string value (unexpected) should be skipped gracefully.
	c := NewSecretsInRunChecker()
	step := buildStep("name", "weird", "run", 42)
	tree := buildTree([]any{step})

	result, err := c.Check(tree)
	require.NoError(t, err, "should not error on non-string run: value")
	assert.False(t, result.Changed, "non-string run: value should not trigger a change")
}

func TestSecretsInRunChecker_IdempotentOnAlreadyMapped(t *testing.T) {
	// Running the checker twice should not produce duplicate env: entries or
	// double-replace the script references.
	c := NewSecretsInRunChecker()
	step := buildStep(
		"name", "Idempotency",
		"run", `curl "${{ secrets.API_TOKEN }}"`,
	)
	tree := buildTree([]any{step})

	// First run.
	result1, err := c.Check(tree)
	require.NoError(t, err, "first run should not error")
	assert.True(t, result1.Changed, "first run should change the tree")

	// Second run — the tree is already fixed; no further changes expected.
	result2, err := c.Check(tree)
	require.NoError(t, err, "second run should not error")
	assert.False(t, result2.Changed, "second run should be a no-op")
	assert.Empty(t, result2.Fixes, "second run should produce no fixes")
}

// ---------------------------------------------------------------------------
// Additional tests
// ---------------------------------------------------------------------------

func TestSecretsInRunChecker_AnonymousStep(t *testing.T) {
	// Step without a "name" field — fix label should fall back to job/index form.
	c := NewSecretsInRunChecker()
	step := buildStep(
		"run", `echo "${{ secrets.ANON_SECRET }}"`,
	)
	tree := buildTree([]any{step})

	result, err := c.Check(tree)
	require.NoError(t, err, "should not error on anonymous step")
	assert.True(t, result.Changed, "anonymous step should be fixed")
	require.Len(t, result.Fixes, 1, "should have one fix")
	// Fix label should reference job and index, not a step name.
	assert.Contains(t, result.Fixes[0], "agent", "fix label should mention the job name")
}

func TestSecretsInRunChecker_JobsNotAMap(t *testing.T) {
	// jobs: is present but has unexpected type — checker should skip gracefully.
	c := NewSecretsInRunChecker()
	tree := map[string]any{
		"jobs": "not-a-map",
	}
	result, err := c.Check(tree)
	require.NoError(t, err, "should not error when jobs: is not a map")
	assert.False(t, result.Changed, "should not be changed when jobs: is not a map")
}

func TestSecretsInRunChecker_StepsNotASlice(t *testing.T) {
	// steps: present but not a slice — checker should skip the job gracefully.
	c := NewSecretsInRunChecker()
	tree := map[string]any{
		"jobs": map[string]any{
			"agent": map[string]any{
				"steps": "not-a-slice",
			},
		},
	}
	result, err := c.Check(tree)
	require.NoError(t, err, "should not error when steps: is not a slice")
	assert.False(t, result.Changed, "should not be changed when steps: is not a slice")
}

func TestSecretsInRunChecker_StepNotAMap(t *testing.T) {
	// A step value that is not a map — checker should skip gracefully.
	c := NewSecretsInRunChecker()
	tree := map[string]any{
		"jobs": map[string]any{
			"agent": map[string]any{
				"steps": []any{"string-step", 42},
			},
		},
	}
	result, err := c.Check(tree)
	require.NoError(t, err, "should not error on non-map step")
	assert.False(t, result.Changed, "non-map steps should not be changed")
}

func TestSecretsInRunChecker_WhitespaceVariations(t *testing.T) {
	// ${{ secrets.TOKEN }} with extra whitespace inside the expression.
	c := NewSecretsInRunChecker()
	step := buildStep(
		"name", "Whitespace",
		"run", `echo "${{  secrets.TOKEN  }}"`,
	)
	tree := buildTree([]any{step})

	result, err := c.Check(tree)
	require.NoError(t, err, "should not error on whitespace variant")
	assert.True(t, result.Changed, "whitespace variant should still be fixed")

	updatedStep := tree["jobs"].(map[string]any)["agent"].(map[string]any)["steps"].([]any)[0].(map[string]any)
	assert.NotContains(t, updatedStep["run"].(string), "${{", "run: should have no expression syntax")
	envMap := updatedStep["env"].(map[string]any)
	assert.Equal(t, "${{  secrets.TOKEN  }}", envMap["TOKEN"],
		"env: should preserve the original expression verbatim")
}

func TestSecretsInRunChecker_MixedSecretsAndGithubToken(t *testing.T) {
	// A single run: block containing both ${{ secrets.* }} and ${{ github.token }}.
	c := NewSecretsInRunChecker()
	step := buildStep(
		"name", "Mixed",
		"run", `
gh auth login --with-token <<< "${{ github.token }}"
curl -H "Authorization: Bearer ${{ secrets.API_KEY }}" https://api.example.com`,
	)
	tree := buildTree([]any{step})

	result, err := c.Check(tree)
	require.NoError(t, err, "should not error on mixed expressions")
	assert.True(t, result.Changed, "tree should be changed")
	assert.Len(t, result.Fixes, 2, "should have two fixes (github.token + API_KEY)")

	updatedStep := tree["jobs"].(map[string]any)["agent"].(map[string]any)["steps"].([]any)[0].(map[string]any)
	envMap, ok := updatedStep["env"].(map[string]any)
	require.True(t, ok, "env: map should be present")
	assert.Equal(t, "${{ github.token }}", envMap["GITHUB_TOKEN"], "GITHUB_TOKEN should be set")
	assert.Equal(t, "${{ secrets.API_KEY }}", envMap["API_KEY"], "API_KEY should be set")
	assert.NotContains(t, updatedStep["run"].(string), "${{", "run: should have no expression syntax")
}

func TestSecretsInRunChecker_EnvMapUnexpectedType(t *testing.T) {
	// env: key is present but has an unexpected (non-map) type — should be
	// replaced with a fresh map containing the fix.
	c := NewSecretsInRunChecker()
	step := buildStep(
		"name", "Bad env type",
		"env", "not-a-map",
		"run", `echo "${{ secrets.MY_SECRET }}"`,
	)
	tree := buildTree([]any{step})

	result, err := c.Check(tree)
	require.NoError(t, err, "should not error when env: has unexpected type")
	assert.True(t, result.Changed, "tree should be changed even with bad env: type")

	updatedStep := tree["jobs"].(map[string]any)["agent"].(map[string]any)["steps"].([]any)[0].(map[string]any)
	envMap, ok := updatedStep["env"].(map[string]any)
	require.True(t, ok, "env: should be replaced with a map")
	assert.Equal(t, "${{ secrets.MY_SECRET }}", envMap["MY_SECRET"], "new env var should be present")
}

func TestSecretsInRunChecker_MultipleStepsInSameJob(t *testing.T) {
	// Multiple steps in the same job that each need fixing.
	c := NewSecretsInRunChecker()
	tree := buildTree([]any{
		buildStep("name", "Step 1", "run", `echo "${{ secrets.SECRET_ONE }}"`),
		buildStep("name", "Step 2", "uses", "actions/checkout@v4"),
		buildStep("name", "Step 3", "run", `echo "${{ secrets.SECRET_TWO }}"`),
	})

	result, err := c.Check(tree)
	require.NoError(t, err, "should not error")
	assert.True(t, result.Changed, "tree should be changed")
	assert.Len(t, result.Fixes, 2, "should fix both run: steps")

	steps := tree["jobs"].(map[string]any)["agent"].(map[string]any)["steps"].([]any)
	step1 := steps[0].(map[string]any)
	step3 := steps[2].(map[string]any)

	assert.Contains(t, step1["env"].(map[string]any), "SECRET_ONE", "step1 should have SECRET_ONE in env:")
	assert.Contains(t, step3["env"].(map[string]any), "SECRET_TWO", "step3 should have SECRET_TWO in env:")
}

func TestSecretsInRunChecker_GithubTokenAndEnvGithubTokenCollision(t *testing.T) {
	// Both ${{ github.token }} and ${{ env.GITHUB_TOKEN }} appear in the same
	// run: block.  Both map to GITHUB_TOKEN; one gets the canonical name and
	// the other gets TOKEN_1 (collision resolution).
	c := NewSecretsInRunChecker()
	step := buildStep(
		"name", "Double token",
		"run", `echo "${{ github.token }}" && echo "${{ env.GITHUB_TOKEN }}"`,
	)
	tree := buildTree([]any{step})

	result, err := c.Check(tree)
	require.NoError(t, err, "should not error")
	assert.True(t, result.Changed, "tree should be changed")

	updatedStep := tree["jobs"].(map[string]any)["agent"].(map[string]any)["steps"].([]any)[0].(map[string]any)
	envMap, ok := updatedStep["env"].(map[string]any)
	require.True(t, ok, "env: map should be present")

	// First GITHUB_TOKEN wins; second resolves to GITHUB_TOKEN_1.
	assert.Contains(t, envMap, "GITHUB_TOKEN", "GITHUB_TOKEN should be set for the first token expression")
	assert.Contains(t, envMap, "GITHUB_TOKEN_1", "GITHUB_TOKEN_1 should be set for the second token expression")
	assert.NotContains(t, updatedStep["run"].(string), "${{", "run: should have no expression syntax")
}

func TestSecretsInRunChecker_SecretNamePreservesCase(t *testing.T) {
	// env var name is derived from the secret name uppercased.
	c := NewSecretsInRunChecker()
	step := buildStep(
		"name", "Mixed case",
		"run", `echo "${{ secrets.mySecret }}"`,
	)
	tree := buildTree([]any{step})

	result, err := c.Check(tree)
	require.NoError(t, err)
	assert.True(t, result.Changed)

	updatedStep := tree["jobs"].(map[string]any)["agent"].(map[string]any)["steps"].([]any)[0].(map[string]any)
	envMap := updatedStep["env"].(map[string]any)
	// Secret name "mySecret" should become "MYSECRET" (uppercased).
	assert.Contains(t, envMap, "MYSECRET", "env var name should be uppercased from secret name")
}

func TestBuildStep_PanicsOnOddArgCount(t *testing.T) {
	assert.Panics(t, func() {
		buildStep("key") //nolint:staticcheck // intentional odd-arg call to test panic
	}, "buildStep should panic on odd argument count")
}

func TestBuildStep_PanicsOnNonStringKey(t *testing.T) {
	assert.Panics(t, func() {
		buildStep(42, "value") // non-string key
	}, "buildStep should panic on non-string key")
}
