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
on: push
jobs:
  test:
    uses: org/reusable/.github/workflows/test.yml@` + reusableSHA + `
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@` + checkoutSHA + `
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
  org/composite@v1:
    ref: v1
    commit: sha1-` + strings.Repeat("c", 40) + `
    owner_id: 30
    repo_id: 40
`
	lockPath := filepath.Join(repoRoot, filepath.FromSlash(nativeActionsLockfilePath))
	require.NoError(t, os.WriteFile(lockPath, []byte(existingLock), 0o644))

	shaFor := map[string]string{
		"actions/checkout@" + checkoutSHA: strings.Repeat("a", 40),
		"org/reusable@" + reusableSHA:     strings.Repeat("b", 40),
		"org/composite@v1":                strings.Repeat("d", 40),
		"actions/setup-node@v6":           strings.Repeat("e", 40),
	}
	resolve := func(_ context.Context, use nativeDependencyUse, refHint string, existing *nativeDependency) (nativeDependencyResolution, error) {
		key := nativeDependencyKey(use)
		resolution := nativeDependencyResolution{
			hostname: "github.com",
			ref:      use.ref,
			commit:   "sha1-" + shaFor[key],
			ownerID:  10,
			repoID:   20,
		}
		if existing != nil {
			resolution.hostname = existing.Hostname
			resolution.ref = existing.Ref
			resolution.commit = existing.Commit
			resolution.ownerID = existing.OwnerID
			resolution.repoID = existing.RepoID
		}
		if refHint != "" {
			if existing == nil {
				resolution.ref = refHint
			}
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
	}, lockfile.Workflows[".github/workflows/agent.lock.yml"])
	require.Equal(t, []string{"org/composite@v1"}, lockfile.Dependencies["actions/checkout@"+checkoutSHA].Uses)
	require.Equal(t, "v6.0.2", lockfile.Dependencies["actions/checkout@"+checkoutSHA].Ref)
	require.Equal(t, []string{"actions/setup-node@v6"}, lockfile.Dependencies["org/reusable@"+reusableSHA].Uses)
	require.Equal(t, []string{"actions/setup-go@v5"}, lockfile.Workflows[".github/workflows/manual.yml"])
	require.Contains(t, lockfile.Dependencies, "actions/setup-go@v5")
	require.Equal(t, []string{"org/stale@v1"}, lockfile.Workflows[".github/workflows/agent.yml"])
	require.Contains(t, lockfile.Dependencies, "org/stale@v1")
	require.Equal(t, "sha1-"+strings.Repeat("c", 40), lockfile.Dependencies["org/composite@v1"].Commit)
	require.Equal(t, "github.com", lockfile.Dependencies["org/composite@v1"].Hostname)

	require.NoError(t, generateNativeActionsLockfileWithResolver(context.Background(), repoRoot, []string{workflowPath}, resolve))
	secondOutput, err := os.ReadFile(lockPath)
	require.NoError(t, err)
	require.True(t, reflect.DeepEqual(firstOutput, secondOutput), "lockfile output should be byte-stable")
}

func TestParseNativeWorkflowUsesRejectsLocalActionsAndSkipsContainerRefs(t *testing.T) {
	content := []byte(`on: push
jobs:
  reusable:
    uses: org/workflows/.github/workflows/reusable.yml@main
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          uses: org/not-an-action@v2
      - run: echo "not an action reference"
        env:
          USES: org/not-an-action@v2
      - uses: ./local
      - uses: docker://alpine:3.20
`)
	_, err := parseNativeWorkflowUses(content)
	require.ErrorContains(t, err, "local action reference")
	content = []byte(strings.ReplaceAll(string(content), "      - uses: ./local\n", ""))
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
      with:
        uses: org/not-an-action@v2
    - uses: ./local
`)
	_, err := parseNativeActionUses(content)
	require.ErrorContains(t, err, "unsupported GitHub dependency reference")
	content = []byte(strings.ReplaceAll(string(content), "    - uses: ./local\n", ""))
	uses, err := parseNativeActionUses(content)
	require.NoError(t, err)
	require.Equal(t, []nativeDependencyUse{{repository: "actions/checkout", ref: "v4"}}, uses)
}

func TestParseNativeWorkflowUsesReportsActionlintParseErrors(t *testing.T) {
	_, err := parseNativeWorkflowUses([]byte("jobs: ["))
	require.ErrorContains(t, err, "failed to parse GitHub Actions workflow")
}

func TestParseNativeDependencyUsesRetainsPathAndRefAtSign(t *testing.T) {
	uses, err := normalizeNativeUses([]string{
		"Org/Actions/first@release@2024",
		"org/actions/second@v1",
	})
	require.NoError(t, err)
	require.Equal(t, []nativeDependencyUse{
		{repository: "org/actions", ref: "release@2024", actionPath: "first"},
		{repository: "org/actions", ref: "v1", actionPath: "second"},
	}, uses)
}

func TestValidateNativeRefHintRequiresCommitMatch(t *testing.T) {
	commit := strings.Repeat("a", 40)
	resolve := func(_ context.Context, _, ref string) (string, error) {
		if ref == "v4.1.0" {
			return commit, nil
		}
		return strings.Repeat("b", 40), nil
	}
	require.Equal(t, "v4.1.0", validateNativeRefHint(context.Background(), "actions/checkout", "v4.1.0", commit, commit, resolve))
	require.Empty(t, validateNativeRefHint(context.Background(), "actions/checkout", "v4.2.0", commit, commit, resolve))
	called := false
	require.Empty(t, validateNativeRefHint(context.Background(), "actions/checkout", "pinned after review", commit, commit, func(context.Context, string, string) (string, error) {
		called = true
		return commit, nil
	}))
	require.False(t, called)
}

func TestParseNativeWorkflowUsesIncludesContextualSelfReferences(t *testing.T) {
	content := []byte(`jobs:
  local:
    uses: ./.github/workflows/child.yml
  self:
    uses: $/.github/workflows/sibling.yml
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: Azure/login@release@2024
`)
	uses, err := parseNativeWorkflowUsesWithContext(content, "org/workflows", strings.Repeat("a", 40))
	require.NoError(t, err)
	require.Equal(t, []nativeDependencyUse{
		{repository: "azure/login", ref: "release@2024"},
		{repository: "org/workflows", ref: strings.Repeat("a", 40), actionPath: ".github/workflows/child.yml", selfRepository: true},
		{repository: "org/workflows", ref: strings.Repeat("a", 40), actionPath: ".github/workflows/sibling.yml", selfRepository: true},
	}, uses)
}

func TestReadNativeWorkflowInputsTraversesLocalReusableWorkflows(t *testing.T) {
	repoRoot := t.TempDir()
	workflowPath := filepath.Join(repoRoot, ".github", "workflows", "agent.lock.yml")
	childPath := filepath.Join(repoRoot, ".github", "workflows", "child.yml")
	require.NoError(t, os.MkdirAll(filepath.Dir(workflowPath), 0o755))
	require.NoError(t, os.WriteFile(workflowPath, []byte(`jobs:
  child:
    uses: ./.github/workflows/child.yml
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
`), 0o644))
	require.NoError(t, os.WriteFile(childPath, []byte(`jobs:
  child:
    runs-on: ubuntu-latest
    steps:
      - uses: other/action@v2
`), 0o644))

	inputs, err := readNativeWorkflowInputs(repoRoot, []string{workflowPath})
	require.NoError(t, err)
	require.Equal(t, []nativeDependencyUse{
		{repository: "actions/checkout", ref: "v4"},
		{repository: "other/action", ref: "v2"},
	}, inputs[0].uses)
}

func TestParseNativeWorkflowUsesCollectsReferencesWithUnknownActionlintFields(t *testing.T) {
	content := []byte(`on: push
concurrency:
  group: lock
  queue:
    max: 1
jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
`)
	uses, err := parseNativeWorkflowUses(content)
	require.NoError(t, err)
	require.Equal(t, []nativeDependencyUse{{repository: "actions/checkout", ref: "v4"}}, uses)
}

func TestParseNativeWorkflowUsesRejectsUnsafeRef(t *testing.T) {
	content := []byte("on: push\njobs:\n  build:\n    runs-on: ubuntu-latest\n    steps:\n      - uses: org/action@main?query=value\n")
	_, err := parseNativeWorkflowUses(content)
	require.ErrorContains(t, err, "unsupported GitHub dependency reference")
}

func TestGenerateNativeActionsLockfileLeavesExistingFileOnResolutionFailure(t *testing.T) {
	repoRoot := t.TempDir()
	workflowPath := filepath.Join(repoRoot, ".github", "workflows", "agent.lock.yml")
	require.NoError(t, os.MkdirAll(filepath.Dir(workflowPath), 0o755))
	require.NoError(t, os.WriteFile(workflowPath, []byte("on: push\njobs:\n  test:\n    runs-on: ubuntu-latest\n    steps:\n      - uses: actions/checkout@v4\n"), 0o644))

	lockPath := filepath.Join(repoRoot, filepath.FromSlash(nativeActionsLockfilePath))
	existing := []byte("version: v0.0.3\nworkflows: {}\ndependencies: {}\n")
	require.NoError(t, os.WriteFile(lockPath, existing, 0o644))
	resolve := func(context.Context, nativeDependencyUse, string, *nativeDependency) (nativeDependencyResolution, error) {
		return nativeDependencyResolution{}, os.ErrPermission
	}

	err := generateNativeActionsLockfileWithResolver(context.Background(), repoRoot, []string{workflowPath}, resolve)
	require.Error(t, err)
	actual, err := os.ReadFile(lockPath)
	require.NoError(t, err)
	require.Equal(t, existing, actual)
}

func TestGenerateNativeActionsLockfileRefusesUnsupportedExistingVersions(t *testing.T) {
	for _, version := range []string{"v0.0.1", "v0.0.4"} {
		t.Run(version, func(t *testing.T) {
			repoRoot := t.TempDir()
			workflowPath := filepath.Join(repoRoot, ".github", "workflows", "agent.lock.yml")
			require.NoError(t, os.MkdirAll(filepath.Dir(workflowPath), 0o755))
			require.NoError(t, os.WriteFile(workflowPath, []byte("on: push\njobs:\n  build:\n    runs-on: ubuntu-latest\n    steps:\n      - uses: actions/checkout@v4\n"), 0o644))
			lockPath := filepath.Join(repoRoot, filepath.FromSlash(nativeActionsLockfilePath))
			existing := []byte("version: " + version + "\nworkflows: {}\ndependencies: {}\n")
			if version == "v0.0.1" {
				existing = []byte("version: v0.0.1\nworkflows:\n  .github/workflows/manual.yml:\n    - actions/checkout@sha1-" + strings.Repeat("a", 40) + "\ndependencies:\n  actions/checkout@sha1-" + strings.Repeat("a", 40) + ":\n    tag: v4\n    branch: v4\n")
			}
			require.NoError(t, os.WriteFile(lockPath, existing, 0o644))
			resolve := func(context.Context, nativeDependencyUse, string, *nativeDependency) (nativeDependencyResolution, error) {
				t.Fatal("resolver must not run for an unsupported existing lockfile")
				return nativeDependencyResolution{}, nil
			}
			err := generateNativeActionsLockfileWithResolver(context.Background(), repoRoot, []string{workflowPath}, resolve)
			require.ErrorContains(t, err, "unsupported GitHub Actions lockfile version")
			actual, err := os.ReadFile(lockPath)
			require.NoError(t, err)
			require.Equal(t, existing, actual)
		})
	}
}

func TestCollectNativeDependencyMergesSameRepositoryPaths(t *testing.T) {
	dependencies := make(map[string]nativeDependency)
	pins := make(map[string]struct{})
	visited := make(map[string]struct{})
	resolve := func(_ context.Context, use nativeDependencyUse, _ string, _ *nativeDependency) (nativeDependencyResolution, error) {
		resolution := nativeDependencyResolution{
			hostname: "github.com",
			ref:      use.ref,
			commit:   "sha1-" + strings.Repeat("a", 40),
			ownerID:  1,
			repoID:   2,
		}
		switch use.actionPath {
		case "parent":
			resolution.uses = []nativeDependencyUse{{repository: "org/actions", ref: "v1", actionPath: "child"}}
		case "child":
			resolution.uses = []nativeDependencyUse{{repository: "other/leaf", ref: "v1"}}
		}
		if use.repository == "other/leaf" {
			resolution.commit = "sha1-" + strings.Repeat("b", 40)
		}
		return resolution, nil
	}

	_, err := collectNativeDependency(context.Background(), nativeDependencyUse{
		repository: "org/actions",
		ref:        "v1",
		actionPath: "parent",
	}, nil, resolve, dependencies, pins, visited)
	require.NoError(t, err)
	require.Equal(t, []string{"other/leaf@v1"}, dependencies["org/actions@v1"].Uses)
	require.NotContains(t, dependencies["org/actions@v1"].Uses, "org/actions@v1")
	require.Contains(t, pins, "other/leaf@v1")
}

func TestCollectNativeDependencyDoesNotEmitCycles(t *testing.T) {
	dependencies := make(map[string]nativeDependency)
	pins := make(map[string]struct{})
	visited := make(map[string]struct{})
	resolve := func(_ context.Context, use nativeDependencyUse, _ string, _ *nativeDependency) (nativeDependencyResolution, error) {
		resolution := nativeDependencyResolution{
			hostname: "github.com",
			ref:      use.ref,
			commit:   "sha1-" + strings.Repeat("a", 40),
			ownerID:  1,
			repoID:   2,
		}
		if use.repository == "org/first" {
			resolution.uses = []nativeDependencyUse{{repository: "org/second", ref: "v1"}}
		} else {
			resolution.commit = "sha1-" + strings.Repeat("b", 40)
			resolution.uses = []nativeDependencyUse{{repository: "org/first", ref: "v1"}}
		}
		return resolution, nil
	}

	_, err := collectNativeDependency(context.Background(), nativeDependencyUse{repository: "org/first", ref: "v1"}, nil, resolve, dependencies, pins, visited)
	require.NoError(t, err)
	require.Equal(t, []string{"org/second@v1"}, dependencies["org/first@v1"].Uses)
	require.Empty(t, dependencies["org/second@v1"].Uses)
}

func TestCollectNativeDependencyTraversesSelfAndRelativeWorkflowReferences(t *testing.T) {
	for _, test := range []struct {
		name     string
		initial  nativeDependencyUse
		selfPath string
	}{
		{name: "self action", initial: nativeDependencyUse{repository: "org/actions", ref: "v1", actionPath: "parent"}, selfPath: "sibling/action.yml"},
		{name: "relative workflow", initial: nativeDependencyUse{repository: "org/workflows", ref: "v1", actionPath: ".github/workflows/parent.yml"}, selfPath: ".github/workflows/child.yml"},
	} {
		t.Run(test.name, func(t *testing.T) {
			dependencies := make(map[string]nativeDependency)
			pins := make(map[string]struct{})
			visited := make(map[string]struct{})
			resolve := func(_ context.Context, use nativeDependencyUse, _ string, _ *nativeDependency) (nativeDependencyResolution, error) {
				resolution := nativeDependencyResolution{
					hostname: "github.com",
					ref:      use.ref,
					commit:   "sha1-" + strings.Repeat("a", 40),
					ownerID:  1,
					repoID:   2,
				}
				if use.actionPath == test.initial.actionPath {
					resolution.uses = []nativeDependencyUse{{
						repository:     use.repository,
						ref:            strings.Repeat("a", 40),
						actionPath:     test.selfPath,
						selfRepository: true,
					}}
				} else if use.selfRepository && use.actionPath == test.selfPath {
					resolution.uses = []nativeDependencyUse{{repository: "other/leaf", ref: "v1"}}
				}
				if use.repository == "other/leaf" {
					resolution.commit = "sha1-" + strings.Repeat("b", 40)
				}
				return resolution, nil
			}

			_, err := collectNativeDependency(context.Background(), test.initial, nil, resolve, dependencies, pins, visited)
			require.NoError(t, err)
			key := nativeDependencyKey(test.initial)
			require.Equal(t, []string{"other/leaf@v1"}, dependencies[key].Uses)
			require.NotContains(t, dependencies, test.initial.repository+"@"+strings.Repeat("a", 40))
			require.Contains(t, pins, "other/leaf@v1")
		})
	}
}
