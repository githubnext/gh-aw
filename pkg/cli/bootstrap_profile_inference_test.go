package cli

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"
)

func TestMergeBootstrapPermissionLevel(t *testing.T) {
	tests := []struct {
		name     string
		existing string
		incoming string
		want     string
	}{
		{name: "empty existing takes incoming", existing: "", incoming: "read", want: "read"},
		{name: "write beats read", existing: "read", incoming: "write", want: "write"},
		{name: "read does not downgrade write", existing: "write", incoming: "read", want: "write"},
		{name: "equal levels unchanged", existing: "write", incoming: "write", want: "write"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mergeBootstrapPermissionLevel(tt.existing, tt.incoming)
			if got != tt.want {
				t.Fatalf("mergeBootstrapPermissionLevel(%q, %q) = %q, want %q", tt.existing, tt.incoming, got, tt.want)
			}
		})
	}
}

func TestBootstrapEventNamesFromOn(t *testing.T) {
	tests := []struct {
		name string
		raw  any
		want []string
	}{
		{name: "string trigger", raw: "issues", want: []string{"issues"}},
		{name: "list of triggers", raw: []any{"issues", "pull_request"}, want: []string{"issues", "pull_request"}},
		{
			name: "map of triggers excludes non-webhook events",
			raw: map[string]any{
				"issues":              map[string]any{"types": []any{"opened"}},
				"schedule":            []any{map[string]any{"cron": "0 0 * * *"}},
				"workflow_dispatch":   nil,
				"repository_dispatch": nil,
			},
			want: []string{"issues"},
		},
		{name: "nil value", raw: nil, want: nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := bootstrapEventNamesFromOn(tt.raw)
			sort.Strings(got)
			want := append([]string(nil), tt.want...)
			sort.Strings(want)
			if len(got) != len(want) {
				t.Fatalf("bootstrapEventNamesFromOn(%v) = %v, want %v", tt.raw, got, want)
			}
			for i := range got {
				if got[i] != want[i] {
					t.Fatalf("bootstrapEventNamesFromOn(%v) = %v, want %v", tt.raw, got, want)
				}
			}
		})
	}
}

func TestMergeBootstrapGitHubAppRequirements(t *testing.T) {
	declaredPermissions := map[string]string{"contents": "read"}
	declaredEvents := []string{"push"}
	inferredPermissions := map[string]string{"contents": "write", "issues": "write"}
	inferredEvents := []string{"issues", "push"}

	mergedPermissions, mergedEvents := mergeBootstrapGitHubAppRequirements(declaredPermissions, declaredEvents, inferredPermissions, inferredEvents)

	wantPermissions := map[string]string{"contents": "write", "issues": "write"}
	if !reflect.DeepEqual(mergedPermissions, wantPermissions) {
		t.Fatalf("merged permissions = %v, want %v", mergedPermissions, wantPermissions)
	}
	wantEvents := []string{"issues", "push"}
	if !reflect.DeepEqual(mergedEvents, wantEvents) {
		t.Fatalf("merged events = %v, want %v", mergedEvents, wantEvents)
	}

	// Declared-only permissions/events with no inference produce the declared set unchanged.
	mergedPermissions, mergedEvents = mergeBootstrapGitHubAppRequirements(declaredPermissions, declaredEvents, nil, nil)
	if !reflect.DeepEqual(mergedPermissions, declaredPermissions) {
		t.Fatalf("merged permissions with no inference = %v, want %v", mergedPermissions, declaredPermissions)
	}
	if !reflect.DeepEqual(mergedEvents, declaredEvents) {
		t.Fatalf("merged events with no inference = %v, want %v", mergedEvents, declaredEvents)
	}

	// No declared or inferred requirements yields nil (not empty maps/slices).
	mergedPermissions, mergedEvents = mergeBootstrapGitHubAppRequirements(nil, nil, nil, nil)
	if mergedPermissions != nil {
		t.Fatalf("merged permissions = %v, want nil", mergedPermissions)
	}
	if mergedEvents != nil {
		t.Fatalf("merged events = %v, want nil", mergedEvents)
	}
}

func TestInferBootstrapGitHubAppRequirements_MergesAcrossWorkflows(t *testing.T) {
	dir := t.TempDir()
	first := filepath.Join(dir, "first.md")
	second := filepath.Join(dir, "second.md")

	firstContent := "---\non:\n  issues:\n    types: [opened]\npermissions:\n  contents: read\n  issues: write\n---\n\n# First\n"
	secondContent := "---\non:\n  pull_request:\n    types: [opened]\n  schedule:\n    - cron: \"0 0 * * *\"\npermissions:\n  contents: write\n---\n\n# Second\n"

	if err := os.WriteFile(first, []byte(firstContent), 0o644); err != nil {
		t.Fatalf("failed to write first workflow: %v", err)
	}
	if err := os.WriteFile(second, []byte(secondContent), 0o644); err != nil {
		t.Fatalf("failed to write second workflow: %v", err)
	}

	permissions, events, err := inferBootstrapGitHubAppRequirements(context.Background(), []string{first, second})
	if err != nil {
		t.Fatalf("inferBootstrapGitHubAppRequirements returned error: %v", err)
	}

	wantPermissions := map[string]string{"contents": "write", "issues": "write"}
	if !reflect.DeepEqual(permissions, wantPermissions) {
		t.Fatalf("permissions = %v, want %v", permissions, wantPermissions)
	}
	wantEvents := []string{"issues", "pull_request"}
	if !reflect.DeepEqual(events, wantEvents) {
		t.Fatalf("events = %v, want %v (schedule must be excluded)", events, wantEvents)
	}
}

func TestInferBootstrapGitHubAppRequirements_NoSources(t *testing.T) {
	permissions, events, err := inferBootstrapGitHubAppRequirements(context.Background(), nil)
	if err != nil {
		t.Fatalf("inferBootstrapGitHubAppRequirements returned error: %v", err)
	}
	if permissions != nil || events != nil {
		t.Fatalf("expected nil permissions/events for no sources, got %v / %v", permissions, events)
	}
}

// TestInferBootstrapGitHubAppRequirements_SingleWorkflow covers the simplest case of a
// single workflow with a string "on" trigger and a small permissions block.
func TestInferBootstrapGitHubAppRequirements_SingleWorkflow(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "solo.md")
	content := "---\non: issues\npermissions:\n  issues: write\n---\n\n# Solo\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write workflow: %v", err)
	}

	permissions, events, err := inferBootstrapGitHubAppRequirements(context.Background(), []string{path})
	if err != nil {
		t.Fatalf("inferBootstrapGitHubAppRequirements returned error: %v", err)
	}

	wantPermissions := map[string]string{"issues": "write"}
	if !reflect.DeepEqual(permissions, wantPermissions) {
		t.Fatalf("permissions = %v, want %v", permissions, wantPermissions)
	}
	wantEvents := []string{"issues"}
	if !reflect.DeepEqual(events, wantEvents) {
		t.Fatalf("events = %v, want %v", events, wantEvents)
	}
}

// TestInferBootstrapGitHubAppRequirements_NoPermissionsOrEvents covers a workflow that
// declares neither a "permissions" block nor an "on" trigger recognized as a webhook
// event; both inferred maps/slices must come back nil rather than empty.
func TestInferBootstrapGitHubAppRequirements_NoPermissionsOrEvents(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bare.md")
	content := "---\non:\n  schedule:\n    - cron: \"0 0 * * *\"\n  workflow_dispatch:\n---\n\n# Bare\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write workflow: %v", err)
	}

	permissions, events, err := inferBootstrapGitHubAppRequirements(context.Background(), []string{path})
	if err != nil {
		t.Fatalf("inferBootstrapGitHubAppRequirements returned error: %v", err)
	}
	if permissions != nil {
		t.Fatalf("permissions = %v, want nil", permissions)
	}
	if len(events) != 0 {
		t.Fatalf("events = %v, want empty", events)
	}
}

// TestInferBootstrapGitHubAppRequirements_ThreeWorkflowsHighestScopeWins exercises
// merging across three workflows where the same resource is requested at different
// scopes (none/read/write) and events accumulate as a de-duplicated, sorted union.
func TestInferBootstrapGitHubAppRequirements_ThreeWorkflowsHighestScopeWins(t *testing.T) {
	dir := t.TempDir()
	paths := []string{
		filepath.Join(dir, "a.md"),
		filepath.Join(dir, "b.md"),
		filepath.Join(dir, "c.md"),
	}
	contents := []string{
		"---\non: issues\npermissions:\n  contents: read\n  issues: read\n---\n\n# A\n",
		"---\non: pull_request\npermissions:\n  contents: write\n  issues: none\n---\n\n# B\n",
		"---\non:\n  - issues\n  - pull_request\npermissions:\n  contents: read\n  issues: write\n---\n\n# C\n",
	}
	for i, path := range paths {
		if err := os.WriteFile(path, []byte(contents[i]), 0o644); err != nil {
			t.Fatalf("failed to write workflow %s: %v", path, err)
		}
	}

	permissions, events, err := inferBootstrapGitHubAppRequirements(context.Background(), paths)
	if err != nil {
		t.Fatalf("inferBootstrapGitHubAppRequirements returned error: %v", err)
	}

	wantPermissions := map[string]string{"contents": "write", "issues": "write"}
	if !reflect.DeepEqual(permissions, wantPermissions) {
		t.Fatalf("permissions = %v, want %v", permissions, wantPermissions)
	}
	wantEvents := []string{"issues", "pull_request"}
	if !reflect.DeepEqual(events, wantEvents) {
		t.Fatalf("events = %v, want %v", events, wantEvents)
	}
}

// TestInferBootstrapGitHubAppRequirements_MapOnAllExcludedYieldsNoEvents verifies that
// when every trigger in a mapping-style "on" block is excluded from App inference
// (schedule/workflow_dispatch/repository_dispatch), no events are inferred at all.
func TestInferBootstrapGitHubAppRequirements_MapOnAllExcludedYieldsNoEvents(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "excluded.md")
	content := "---\non:\n  schedule:\n    - cron: \"*/5 * * * *\"\n  workflow_dispatch:\n  repository_dispatch:\n    types: [custom]\npermissions:\n  contents: read\n---\n\n# Excluded\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write workflow: %v", err)
	}

	permissions, events, err := inferBootstrapGitHubAppRequirements(context.Background(), []string{path})
	if err != nil {
		t.Fatalf("inferBootstrapGitHubAppRequirements returned error: %v", err)
	}
	wantPermissions := map[string]string{"contents": "read"}
	if !reflect.DeepEqual(permissions, wantPermissions) {
		t.Fatalf("permissions = %v, want %v", permissions, wantPermissions)
	}
	if len(events) != 0 {
		t.Fatalf("events = %v, want empty", events)
	}
}

// TestInferBootstrapGitHubAppRequirements_InvalidFrontmatterSkipped verifies that a
// workflow file with unparsable frontmatter does not fail the whole inference pass;
// its contribution is simply skipped while valid sibling workflows still contribute.
func TestInferBootstrapGitHubAppRequirements_InvalidFrontmatterSkipped(t *testing.T) {
	dir := t.TempDir()
	badPath := filepath.Join(dir, "bad.md")
	goodPath := filepath.Join(dir, "good.md")

	badContent := "---\non: [issues\npermissions:\n  contents: read\n---\n\n# Bad\n"
	goodContent := "---\non: pull_request\npermissions:\n  contents: write\n---\n\n# Good\n"
	if err := os.WriteFile(badPath, []byte(badContent), 0o644); err != nil {
		t.Fatalf("failed to write bad workflow: %v", err)
	}
	if err := os.WriteFile(goodPath, []byte(goodContent), 0o644); err != nil {
		t.Fatalf("failed to write good workflow: %v", err)
	}

	permissions, events, err := inferBootstrapGitHubAppRequirements(context.Background(), []string{badPath, goodPath})
	if err != nil {
		t.Fatalf("inferBootstrapGitHubAppRequirements returned error: %v", err)
	}
	wantPermissions := map[string]string{"contents": "write"}
	if !reflect.DeepEqual(permissions, wantPermissions) {
		t.Fatalf("permissions = %v, want %v", permissions, wantPermissions)
	}
	wantEvents := []string{"pull_request"}
	if !reflect.DeepEqual(events, wantEvents) {
		t.Fatalf("events = %v, want %v", events, wantEvents)
	}
}

// TestInferBootstrapGitHubAppRequirements_ResolutionErrorPropagates verifies that
// errors from resolving the underlying workflow sources (e.g. a nonexistent local
// file) are surfaced to the caller instead of being silently swallowed.
func TestInferBootstrapGitHubAppRequirements_ResolutionErrorPropagates(t *testing.T) {
	dir := t.TempDir()
	missing := filepath.Join(dir, "does-not-exist.md")

	_, _, err := inferBootstrapGitHubAppRequirements(context.Background(), []string{missing})
	if err == nil {
		t.Fatal("expected an error for an unresolvable source, got nil")
	}
}
