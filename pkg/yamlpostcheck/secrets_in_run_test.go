//go:build !integration

package yamlpostcheck

import (
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
	step := make(map[string]any)
	for i := 0; i+1 < len(kvs); i += 2 {
		step[kvs[i].(string)] = kvs[i+1]
	}
	return step
}

func TestSecretsInRunChecker_Name(t *testing.T) {
	c := NewSecretsInRunChecker()
	assert.Equal(t, "rgs008-secrets-in-run", c.Name(), "checker name should match")
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
