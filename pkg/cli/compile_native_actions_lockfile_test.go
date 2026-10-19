package cli

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestGenerateNativeActionsLockfileIncludesTransitiveDependencies(t *testing.T) {
	repoRoot := t.TempDir()
	workflowPath := filepath.Join(repoRoot, ".github", "workflows", "agent.lock.yml")
	require.NoError(t, os.MkdirAll(filepath.Dir(workflowPath), 0o755))

	checkoutSHA := strings.Repeat("a", 40)
	reusableSHA := strings.Repeat("b", 40)
	workflowContent := `# gh-aw-manifest: {"version":1,"actions":[{"repo":"actions/checkout","sha":"` + checkoutSHA + `","version":"v6.0.2"}]}
name: Agent
jobs:
  test:
    uses: org/reusable/.github/workflows/test.yml@` + reusableSHA + `
  build:
    steps:
      - uses: actions/checkout@` + checkoutSHA + `
      - uses: ./local-action
      - uses: docker://alpine:latest
`
	require.NoError(t, os.WriteFile(workflowPath, []byte(workflowContent), 0o644))

	existingLock := `version: v0.0.2
workflows:
  .github/workflows/manual.yml:
    - actions/setup-go@v5
  .github/workflows/agent.yml:
    - org/stale@v1
dependencies:
  actions/setup-go@v5:
    ref: v5
    commit: sha1-` + strings.Repeat("c", 40) + `
    owner_id: 1
    repo_id: 2
  org/stale@v1:
    ref: v1
    commit: sha1-` + strings.Repeat("f", 40) + `
    owner_id: 3
    repo_id: 4
`
	lockPath := filepath.Join(repoRoot, filepath.FromSlash(nativeActionsLockfilePath))
	require.NoError(t, os.WriteFile(lockPath, []byte(existingLock), 0o644))

	shaFor := map[string]string{
		"actions/checkout@" + checkoutSHA: strings.Repeat("a", 40),
		"org/reusable@" + reusableSHA:     strings.Repeat("b", 40),
		"org/composite@v1":                strings.Repeat("d", 40),
		"actions/setup-node@v6":           strings.Repeat("e", 40),
	}
	resolve := func(_ context.Context, use nativeDependencyUse, refHint string) (nativeDependencyResolution, error) {
		key := nativeDependencyKey(use)
		resolution := nativeDependencyResolution{
			ref:     use.ref,
			commit:  "sha1-" + shaFor[key],
			ownerID: 10,
			repoID:  20,
		}
		if refHint != "" {
			resolution.ref = refHint
		}
		switch key {
		case "actions/checkout@" + checkoutSHA:
			resolution.uses = []nativeDependencyUse{{repository: "org/composite", ref: "v1"}}
		case "org/reusable@" + reusableSHA:
			resolution.uses = []nativeDependencyUse{{repository: "actions/setup-node", ref: "v6"}}
		}
		return resolution, nil
	}

	require.NoError(t, generateNativeActionsLockfileWithResolver(context.Background(), repoRoot, []string{workflowPath}, resolve))
	firstOutput, err := os.ReadFile(lockPath)
	require.NoError(t, err)

	var lockfile nativeActionsLockfile
	require.NoError(t, yaml.Unmarshal(firstOutput, &lockfile))
	require.Equal(t, "v0.0.3", lockfile.Version)
	require.Equal(t, []string{
		"actions/checkout@" + checkoutSHA,
		"actions/setup-node@v6",
		"org/composite@v1",
		"org/reusable@" + reusableSHA,
	}, lockfile.Workflows[".github/workflows/agent.yml"])
	require.Equal(t, []string{"org/composite@v1"}, lockfile.Dependencies["actions/checkout@"+checkoutSHA].Uses)
	require.Equal(t, "v6.0.2", lockfile.Dependencies["actions/checkout@"+checkoutSHA].Ref)
	require.Equal(t, []string{"actions/setup-node@v6"}, lockfile.Dependencies["org/reusable@"+reusableSHA].Uses)
	require.Equal(t, []string{"actions/setup-go@v5"}, lockfile.Workflows[".github/workflows/manual.yml"])
	require.Contains(t, lockfile.Dependencies, "actions/setup-go@v5")
	require.NotContains(t, lockfile.Dependencies, "org/stale@v1")

	require.NoError(t, generateNativeActionsLockfileWithResolver(context.Background(), repoRoot, []string{workflowPath}, resolve))
	secondOutput, err := os.ReadFile(lockPath)
	require.NoError(t, err)
	require.True(t, reflect.DeepEqual(firstOutput, secondOutput), "lockfile output should be byte-stable")
}

func TestParseNativeWorkflowUsesSkipsLocalAndContainerRefs(t *testing.T) {
	content := []byte(`jobs:
  build:
    uses: org/workflows/.github/workflows/reusable.yml@main
    steps:
      - uses: actions/checkout@v4
      - uses: ./local
      - uses: docker://alpine:3.20
`)
	uses, err := parseNativeWorkflowUses(content)
	require.NoError(t, err)
	require.Equal(t, []nativeDependencyUse{
		{repository: "actions/checkout", ref: "v4"},
		{repository: "org/workflows", ref: "main", actionPath: ".github/workflows/reusable.yml"},
	}, uses)
}

func TestParseNativeActionUsesOnlyReadsCompositeSteps(t *testing.T) {
	content := []byte(`name: composite
inputs:
  uses:
    description: unrelated input
runs:
  using: composite
  steps:
    - uses: actions/checkout@v4
    - uses: ./local
`)
	uses, err := parseNativeActionUses(content)
	require.NoError(t, err)
	require.Equal(t, []nativeDependencyUse{{repository: "actions/checkout", ref: "v4"}}, uses)
}

func TestParseNativeWorkflowUsesRejectsUnsafeRef(t *testing.T) {
	content := []byte("jobs:\n  build:\n    steps:\n      - uses: org/action@main?query=value\n")
	_, err := parseNativeWorkflowUses(content)
	require.ErrorContains(t, err, "unsupported GitHub dependency reference")
}

func TestGenerateNativeActionsLockfileLeavesExistingFileOnResolutionFailure(t *testing.T) {
	repoRoot := t.TempDir()
	workflowPath := filepath.Join(repoRoot, ".github", "workflows", "agent.lock.yml")
	require.NoError(t, os.MkdirAll(filepath.Dir(workflowPath), 0o755))
	require.NoError(t, os.WriteFile(workflowPath, []byte("jobs:\n  test:\n    steps:\n      - uses: actions/checkout@v4\n"), 0o644))

	lockPath := filepath.Join(repoRoot, filepath.FromSlash(nativeActionsLockfilePath))
	existing := []byte("version: v0.0.3\nworkflows: {}\ndependencies: {}\n")
	require.NoError(t, os.WriteFile(lockPath, existing, 0o644))
	resolve := func(context.Context, nativeDependencyUse, string) (nativeDependencyResolution, error) {
		return nativeDependencyResolution{}, os.ErrPermission
	}

	err := generateNativeActionsLockfileWithResolver(context.Background(), repoRoot, []string{workflowPath}, resolve)
	require.Error(t, err)
	actual, err := os.ReadFile(lockPath)
	require.NoError(t, err)
	require.Equal(t, existing, actual)
}
