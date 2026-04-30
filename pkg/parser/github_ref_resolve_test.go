//go:build !integration

package parser

import (
	"testing"
)

// mockGitHubRefPinner is a test implementation of GitHubRefPinner that
// replaces values with a predictable "pinned:<value>" string.
type mockGitHubRefPinner struct{}

func (m *mockGitHubRefPinner) PinGitHubRef(value string) string {
	return "pinned:" + value
}

func TestResolveGitHubRefInputs_NoSchema(t *testing.T) {
	inputs := map[string]any{"pkg": "owner/repo"}
	fm := map[string]any{} // no import-schema
	pinner := &mockGitHubRefPinner{}

	result := resolveGitHubRefInputs(inputs, fm, pinner)
	if result["pkg"] != "owner/repo" {
		t.Errorf("expected unchanged value without schema, got %q", result["pkg"])
	}
}

func TestResolveGitHubRefInputs_NilPinner(t *testing.T) {
	inputs := map[string]any{"pkg": "owner/repo"}
	fm := map[string]any{
		"import-schema": map[string]any{
			"pkg": map[string]any{
				"type":       "string",
				"github_ref": true,
			},
		},
	}

	result := resolveGitHubRefInputs(inputs, fm, nil)
	if result["pkg"] != "owner/repo" {
		t.Errorf("expected unchanged value with nil pinner, got %q", result["pkg"])
	}
}

func TestResolveGitHubRefInputs_StringField(t *testing.T) {
	inputs := map[string]any{
		"pkg":   "owner/repo",
		"other": "plain-value",
	}
	fm := map[string]any{
		"import-schema": map[string]any{
			"pkg": map[string]any{
				"type":       "string",
				"github_ref": true,
			},
			"other": map[string]any{
				"type": "string",
			},
		},
	}
	pinner := &mockGitHubRefPinner{}

	result := resolveGitHubRefInputs(inputs, fm, pinner)
	if result["pkg"] != "pinned:owner/repo" {
		t.Errorf("expected pinned value for github_ref field, got %q", result["pkg"])
	}
	if result["other"] != "plain-value" {
		t.Errorf("expected unchanged value for non-github_ref field, got %q", result["other"])
	}
}

func TestResolveGitHubRefInputs_ArrayField(t *testing.T) {
	inputs := map[string]any{
		"packages": []any{"owner/pkg1", "owner/pkg2"},
	}
	fm := map[string]any{
		"import-schema": map[string]any{
			"packages": map[string]any{
				"type": "array",
				"items": map[string]any{
					"type":       "string",
					"github_ref": true,
				},
			},
		},
	}
	pinner := &mockGitHubRefPinner{}

	result := resolveGitHubRefInputs(inputs, fm, pinner)
	arr, ok := result["packages"].([]any)
	if !ok {
		t.Fatalf("expected packages to be []any, got %T", result["packages"])
	}
	if len(arr) != 2 {
		t.Fatalf("expected 2 items, got %d", len(arr))
	}
	if arr[0] != "pinned:owner/pkg1" {
		t.Errorf("expected arr[0] to be pinned, got %q", arr[0])
	}
	if arr[1] != "pinned:owner/pkg2" {
		t.Errorf("expected arr[1] to be pinned, got %q", arr[1])
	}
}

func TestResolveGitHubRefInputs_ArrayFieldNoGitHubRef(t *testing.T) {
	inputs := map[string]any{
		"languages": []any{"go", "typescript"},
	}
	fm := map[string]any{
		"import-schema": map[string]any{
			"languages": map[string]any{
				"type": "array",
				"items": map[string]any{
					"type": "string",
					// no github_ref
				},
			},
		},
	}
	pinner := &mockGitHubRefPinner{}

	result := resolveGitHubRefInputs(inputs, fm, pinner)
	arr, ok := result["languages"].([]any)
	if !ok {
		t.Fatalf("expected languages to be []any, got %T", result["languages"])
	}
	if arr[0] != "go" || arr[1] != "typescript" {
		t.Errorf("expected unchanged values for non-github_ref array, got %v", arr)
	}
}
