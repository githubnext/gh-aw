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
