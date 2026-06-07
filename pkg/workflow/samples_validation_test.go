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

// TestValidateSafeOutputsSamples_RuntimeExpressionsBypassValidation verifies
// that sample values containing `${{ ... }}` GitHub Actions expressions
// (e.g. `item_number: ${{ github.event.inputs.issue_number }}`) bypass
// compile-time schema validation, since GitHub Actions substitutes them on
// the runner before apply_samples.cjs reads GH_AW_SAMPLES.
//
// Regression for https://github.com/github/gh-aw/issues/37532.
func TestValidateSafeOutputsSamples_RuntimeExpressionsBypassValidation(t *testing.T) {
	cfg := &SafeOutputsConfig{
		AddLabels: &AddLabelsConfig{
			BaseSafeOutputConfig: BaseSafeOutputConfig{
				Samples: []map[string]any{
					{
						// item_number's pattern is `^(\d+|#?aw_[A-Za-z0-9_]{3,12})$`,
						// so the raw expression string would otherwise fail validation.
						"item_number": "${{ github.event.inputs.issue_number }}",
						"labels":      []any{"runtime-sample"},
					},
				},
			},
		},
	}
	if err := validateSafeOutputsSamples(cfg); err != nil {
		t.Fatalf("expected runtime expression in sample value to bypass validation, got: %v", err)
	}

	// Original sample must be preserved (validation must not mutate) so that
	// generateSamplesReplayStep emits the live `${{ ... }}` expression for
	// GitHub Actions to substitute at runtime.
	got := cfg.AddLabels.Samples[0]["item_number"]
	if got != "${{ github.event.inputs.issue_number }}" {
		t.Errorf("validation must not mutate the original sample; got item_number=%v", got)
	}
}

// TestValidateSafeOutputsSamples_RuntimeExpressionsInNestedValues verifies
// that runtime-expression substitution works for nested arrays and objects
// (e.g. fields inside create_issue.fields[*].value).
func TestValidateSafeOutputsSamples_RuntimeExpressionsInNestedValues(t *testing.T) {
	cfg := &SafeOutputsConfig{
		CreateIssues: &CreateIssuesConfig{
			BaseSafeOutputConfig: BaseSafeOutputConfig{
				Samples: []map[string]any{
					{
						"title": "Issue ${{ github.event.inputs.title_suffix }}",
						"body":  "Body",
						"labels": []any{
							"static-label",
							"${{ github.event.inputs.dynamic_label }}",
						},
					},
				},
			},
		},
	}
	if err := validateSafeOutputsSamples(cfg); err != nil {
		t.Fatalf("expected nested runtime expressions to bypass validation, got: %v", err)
	}
}

// TestValidateSafeOutputsSamples_NonExpressionErrorsStillReported verifies
// that swapping in the runtime-expression placeholder does NOT mask genuine
// validation errors on adjacent fields (e.g. a still-missing required field).
func TestValidateSafeOutputsSamples_NonExpressionErrorsStillReported(t *testing.T) {
	cfg := &SafeOutputsConfig{
		CreateIssues: &CreateIssuesConfig{
			BaseSafeOutputConfig: BaseSafeOutputConfig{
				Samples: []map[string]any{
					{
						// title is required and is missing; an expression on
						// the body must not paper over that.
						"body": "${{ github.event.inputs.body }}",
					},
				},
			},
		},
	}
	if err := validateSafeOutputsSamples(cfg); err == nil {
		t.Fatal("expected missing-title error to still surface even though body is a runtime expression")
	}
}

// TestSubstituteRuntimeExpressionsForValidation_LeavesLiteralsUntouched
// verifies that the substitution helper only touches strings containing
// `${{ ... }}` and otherwise returns equivalent values.
func TestSubstituteRuntimeExpressionsForValidation_LeavesLiteralsUntouched(t *testing.T) {
	in := map[string]any{
		"title": "literal title",
		"count": float64(5),
		"flags": []any{"a", "${{ github.run_id }}", "b"},
		"nested": map[string]any{
			"id": "${{ inputs.id }}",
		},
	}
	out := substituteRuntimeExpressionsForValidation(in, nil).(map[string]any)

	if out["title"] != "literal title" {
		t.Errorf("expected literal string to be unchanged, got %v", out["title"])
	}
	if out["count"] != float64(5) {
		t.Errorf("expected numeric value to be unchanged, got %v", out["count"])
	}
	flags := out["flags"].([]any)
	if flags[0] != "a" || flags[2] != "b" {
		t.Errorf("expected literal array elements to be unchanged, got %v", flags)
	}
	if flags[1] != sampleRuntimeExpressionPlaceholder {
		t.Errorf("expected array element with ${{...}} to be substituted, got %v", flags[1])
	}
	nested := out["nested"].(map[string]any)
	if nested["id"] != sampleRuntimeExpressionPlaceholder {
		t.Errorf("expected nested ${{...}} string to be substituted, got %v", nested["id"])
	}

	// Original input must not be mutated.
	if in["nested"].(map[string]any)["id"] != "${{ inputs.id }}" {
		t.Error("substituteRuntimeExpressionsForValidation must not mutate its input")
	}
}

// TestValidateSafeOutputsSamples_RuntimeExpressionWithEmbeddedBrace covers
// expressions whose body contains `}` characters (e.g. fromJSON literals).
// The substitution regex must use non-greedy `.*?` rather than `[^}]*`, so
// the entire `${{ ... }}` token is recognized and substituted.
func TestValidateSafeOutputsSamples_RuntimeExpressionWithEmbeddedBrace(t *testing.T) {
	cfg := &SafeOutputsConfig{
		AddLabels: &AddLabelsConfig{
			BaseSafeOutputConfig: BaseSafeOutputConfig{
				Samples: []map[string]any{
					{
						"item_number": `${{ fromJSON('{"n":42}').n }}`,
						"labels":      []any{"embedded-brace"},
					},
				},
			},
		},
	}
	if err := validateSafeOutputsSamples(cfg); err != nil {
		t.Fatalf("expected expression containing `}` inside string literal to be substituted, got: %v", err)
	}
	if cfg.AddLabels.Samples[0]["item_number"] != `${{ fromJSON('{"n":42}').n }}` {
		t.Errorf("validation must not mutate the original sample")
	}
}

// TestValidateSafeOutputsSamples_RuntimeExpressionInEnumField verifies that
// substitution is schema-aware: an enum-constrained field (e.g.
// create_code_scanning_alert.severity) gets replaced with the first allowed
// enum value rather than the generic `aw_sample` string, so the substituted
// sample still validates.
func TestValidateSafeOutputsSamples_RuntimeExpressionInEnumField(t *testing.T) {
	cfg := &SafeOutputsConfig{
		CreateCodeScanningAlerts: &CreateCodeScanningAlertsConfig{
			BaseSafeOutputConfig: BaseSafeOutputConfig{
				Samples: []map[string]any{
					{
						"file":     "src/foo.go",
						"line":     1,
						"message":  "demo",
						"severity": "${{ github.event.inputs.severity }}",
					},
				},
			},
		},
	}
	if err := validateSafeOutputsSamples(cfg); err != nil {
		t.Fatalf("expected runtime expression on enum-constrained field to substitute to a valid enum value, got: %v", err)
	}
}

// TestValidateSafeOutputsSamples_RuntimeExpressionInBooleanField covers
// schema-aware substitution for a boolean-typed field
// (create_pull_request.draft).
func TestValidateSafeOutputsSamples_RuntimeExpressionInBooleanField(t *testing.T) {
	cfg := &SafeOutputsConfig{
		CreatePullRequests: &CreatePullRequestsConfig{
			BaseSafeOutputConfig: BaseSafeOutputConfig{
				Samples: []map[string]any{
					{
						"title":  "PR",
						"body":   "Body",
						"branch": "br",
						"draft":  "${{ github.event.inputs.draft }}",
					},
				},
			},
		},
	}
	if err := validateSafeOutputsSamples(cfg); err != nil {
		t.Fatalf("expected runtime expression on boolean field to substitute to a boolean, got: %v", err)
	}
}

// TestPlaceholderForSchema covers the schema-driven placeholder lookup for
// the common shapes used inside safe_outputs_tools.json.
func TestPlaceholderForSchema(t *testing.T) {
	cases := []struct {
		name   string
		schema map[string]any
		want   any
	}{
		{name: "nil schema", schema: nil, want: sampleRuntimeExpressionPlaceholder},
		{
			name:   "enum picks first value",
			schema: map[string]any{"type": "string", "enum": []any{"APPROVE", "REQUEST_CHANGES", "COMMENT"}},
			want:   "APPROVE",
		},
		{
			name:   "boolean",
			schema: map[string]any{"type": "boolean"},
			want:   true,
		},
		{
			name:   "number",
			schema: map[string]any{"type": "number"},
			want:   float64(1),
		},
		{
			name:   "integer",
			schema: map[string]any{"type": "integer"},
			want:   float64(1),
		},
		{
			name:   "type union prefers string",
			schema: map[string]any{"type": []any{"number", "string"}},
			want:   sampleRuntimeExpressionPlaceholder,
		},
		{
			name:   "date format",
			schema: map[string]any{"type": "string", "format": "date"},
			want:   "2024-01-01",
		},
		{
			name:   "uri format",
			schema: map[string]any{"type": "string", "format": "uri"},
			want:   "https://example.com",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := placeholderForSchema(tc.schema)
			if got != tc.want {
				t.Errorf("placeholderForSchema(%v) = %v, want %v", tc.schema, got, tc.want)
			}
		})
	}
}
