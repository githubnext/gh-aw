package workflow

import (
	"strings"
	"testing"
)

// TestValidateSafeOutputsSamples_Valid covers the happy path for the
// strict schema validation of samples entries. We use create_issue (no
// sidecars, just title/body) and create_pull_request (with the `patch` sidecar
// that must be stripped before validation).
func TestValidateSafeOutputsSamples_Valid(t *testing.T) {
	cfg := &SafeOutputsConfig{
		CreateIssues: &CreateIssuesConfig{
			BaseSafeOutputConfig: BaseSafeOutputConfig{
				Samples: []map[string]any{
					{
						"title": "Sample issue",
						"body":  "Sample body",
					},
				},
			},
		},
		CreatePullRequests: &CreatePullRequestsConfig{
			BaseSafeOutputConfig: BaseSafeOutputConfig{
				Samples: []map[string]any{
					{
						"title":  "Sample PR",
						"body":   "Sample PR body",
						"branch": "gh-aw-sample-pr",
						// patch is a sidecar — must be stripped before validation
						// and must NOT cause an `additionalProperties` failure.
						"patch": "diff --git a/foo b/foo\nnew file mode 100644\n--- /dev/null\n+++ b/foo\n@@ -0,0 +1 @@\n+hi\n",
					},
				},
			},
		},
	}
	if err := validateSafeOutputsSamples(cfg); err != nil {
		t.Fatalf("expected no validation error, got: %v", err)
	}
}

// TestValidateSafeOutputsSamples_MissingRequired verifies that omitting a
// required field (title) surfaces a stable, parseable error.
func TestValidateSafeOutputsSamples_MissingRequired(t *testing.T) {
	cfg := &SafeOutputsConfig{
		CreateIssues: &CreateIssuesConfig{
			BaseSafeOutputConfig: BaseSafeOutputConfig{
				Samples: []map[string]any{
					{
						// title intentionally missing
						"body": "Body without title",
					},
				},
			},
		},
	}
	err := validateSafeOutputsSamples(cfg)
	if err == nil {
		t.Fatal("expected validation error for missing title, got nil")
	}
	msg := err.Error()
	if !strings.Contains(msg, "create-issue") {
		t.Errorf("expected error to reference the YAML key `create-issue`, got: %s", msg)
	}
	if !strings.Contains(msg, "samples[0]") {
		t.Errorf("expected error to reference `samples[0]`, got: %s", msg)
	}
}

// TestValidateSafeOutputsSamples_SidecarStripped verifies that the `patch`
// sidecar is stripped before validation, so a create_pull_request sample with
// only the schema-required fields PLUS a patch validates cleanly.
func TestValidateSafeOutputsSamples_SidecarStripped(t *testing.T) {
	cfg := &SafeOutputsConfig{
		CreatePullRequests: &CreatePullRequestsConfig{
			BaseSafeOutputConfig: BaseSafeOutputConfig{
				Samples: []map[string]any{
					{
						"title":  "PR",
						"body":   "PR body",
						"branch": "gh-aw-x",
						"patch":  "diff --git a/x b/x\n",
					},
				},
			},
		},
	}
	if err := validateSafeOutputsSamples(cfg); err != nil {
		t.Fatalf("expected sidecar to be stripped and validation to pass, got: %v", err)
	}
}

// TestCollectSampleEntries_DeterministicOrdering verifies that entries are
// emitted in a stable order across runs (sorted by SafeOutputsConfig field name)
// so that compiled YAML is deterministic.
func TestCollectSampleEntries_DeterministicOrdering(t *testing.T) {
	cfg := &SafeOutputsConfig{
		CreateIssues: &CreateIssuesConfig{
			BaseSafeOutputConfig: BaseSafeOutputConfig{
				Samples: []map[string]any{
					{"title": "A", "body": "A"},
				},
			},
		},
		AddComments: &AddCommentsConfig{
			BaseSafeOutputConfig: BaseSafeOutputConfig{
				Samples: []map[string]any{
					{"body": "comment-A"},
				},
			},
		},
	}
	first := collectSampleEntries(cfg)
	second := collectSampleEntries(cfg)

	if len(first) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(first))
	}
	if first[0].Tool != second[0].Tool || first[1].Tool != second[1].Tool {
		t.Errorf("expected deterministic ordering across runs, got first=%v second=%v", first, second)
	}
	// Sorted by struct field name: AddComments < CreateIssues.
	if first[0].Tool != "add_comment" {
		t.Errorf("expected first entry tool to be add_comment (alphabetical struct field order), got %q", first[0].Tool)
	}
	if first[1].Tool != "create_issue" {
		t.Errorf("expected second entry tool to be create_issue, got %q", first[1].Tool)
	}
}

// TestCollectSampleEntries_SidecarPartitioning verifies that sidecar fields
// land in Sidecars (not Arguments) so the driver knows what to pre-stage.
func TestCollectSampleEntries_SidecarPartitioning(t *testing.T) {
	cfg := &SafeOutputsConfig{
		CreatePullRequests: &CreatePullRequestsConfig{
			BaseSafeOutputConfig: BaseSafeOutputConfig{
				Samples: []map[string]any{
					{
						"title":  "PR",
						"body":   "Body",
						"branch": "br",
						"patch":  "diff --git a/x b/x\n",
					},
				},
			},
		},
	}
	entries := collectSampleEntries(cfg)
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	e := entries[0]
	if e.Tool != "create_pull_request" {
		t.Errorf("expected tool create_pull_request, got %q", e.Tool)
	}
	if _, hasPatchInArgs := e.Arguments["patch"]; hasPatchInArgs {
		t.Error("expected patch to be stripped from Arguments")
	}
	if e.Arguments["title"] != "PR" || e.Arguments["body"] != "Body" || e.Arguments["branch"] != "br" {
		t.Errorf("expected title/body/branch to remain in Arguments, got %#v", e.Arguments)
	}
	if e.Sidecars == nil {
		t.Fatal("expected Sidecars to be non-nil")
	}
	if patch, ok := e.Sidecars["patch"].(string); !ok || !strings.HasPrefix(patch, "diff --git") {
		t.Errorf("expected patch to be present in Sidecars as a git diff string, got %#v", e.Sidecars["patch"])
	}
}
