//go:build !integration

package parser

import (
	"testing"
)

// TestMainWorkflowSchema_SafeOutputsTargetProperties validates that safe output
// types which support target/target-repo/allowed-repos in the Go code also accept
// those properties in the JSON schema. This is a regression test for cases where
// the Go code supports these fields but the schema was missing them, causing
// "Unknown properties" validation errors at compile time.
func TestMainWorkflowSchema_SafeOutputsTargetProperties(t *testing.T) {
	t.Parallel()

	baseFrontmatter := func(safeOutputs map[string]any) map[string]any {
		return map[string]any{
			"on":           map[string]any{"issues": map[string]any{"types": []any{"opened"}}},
			"engine":       "copilot",
			"safe-outputs": safeOutputs,
		}
	}

	// Safe output types that should accept target, target-repo, and allowed-repos
	// properties in the JSON schema.
	tests := []struct {
		name        string
		safeOutputs map[string]any
	}{
		{
			name: "resolve-pull-request-review-thread with target and target-repo",
			safeOutputs: map[string]any{
				"resolve-pull-request-review-thread": map[string]any{
					"max":           10,
					"target":        "*",
					"target-repo":   "github/github",
					"allowed-repos": []any{"github/docs"},
				},
			},
		},
		{
			name: "submit-pull-request-review with target and target-repo",
			safeOutputs: map[string]any{
				"submit-pull-request-review": map[string]any{
					"max":           1,
					"target":        "*",
					"target-repo":   "github/github",
					"allowed-repos": []any{"github/docs"},
				},
			},
		},
		{
			name: "add-comment with target",
			safeOutputs: map[string]any{
				"add-comment": map[string]any{
					"max":    5,
					"target": "*",
				},
			},
		},
		{
			name: "update-issue with target and target-repo",
			safeOutputs: map[string]any{
				"update-issue": map[string]any{
					"body":          nil,
					"target":        "*",
					"target-repo":   "github/github",
					"allowed-repos": []any{"github/docs"},
				},
			},
		},
		{
			name: "update-pull-request with target and target-repo",
			safeOutputs: map[string]any{
				"update-pull-request": map[string]any{
					"body":          true,
					"target":        "*",
					"target-repo":   "github/github",
					"allowed-repos": []any{"github/docs"},
				},
			},
		},
		{
			name: "update-discussion with target and target-repo",
			safeOutputs: map[string]any{
				"update-discussion": map[string]any{
					"body":          nil,
					"target":        "*",
					"target-repo":   "github/github",
					"allowed-repos": []any{"github/docs"},
				},
			},
		},
		{
			name: "close-issue with target and target-repo",
			safeOutputs: map[string]any{
				"close-issue": map[string]any{
					"target":        "*",
					"target-repo":   "github/github",
					"allowed-repos": []any{"github/docs"},
				},
			},
		},
		{
			name: "close-pull-request with target and target-repo",
			safeOutputs: map[string]any{
				"close-pull-request": map[string]any{
					"target":        "*",
					"target-repo":   "github/github",
					"allowed-repos": []any{"github/docs"},
				},
			},
		},
		{
			name: "close-discussion with target and target-repo",
			safeOutputs: map[string]any{
				"close-discussion": map[string]any{
					"target":        "*",
					"target-repo":   "github/github",
					"allowed-repos": []any{"github/docs"},
				},
			},
		},
		{
			name: "add-labels with target and target-repo",
			safeOutputs: map[string]any{
				"add-labels": map[string]any{
					"allowed":       []any{"bug", "feature"},
					"target":        "*",
					"target-repo":   "github/github",
					"allowed-repos": []any{"github/docs"},
				},
			},
		},
		{
			name: "remove-labels with target and target-repo",
			safeOutputs: map[string]any{
				"remove-labels": map[string]any{
					"allowed":       []any{"bug", "feature"},
					"target":        "*",
					"target-repo":   "github/github",
					"allowed-repos": []any{"github/docs"},
				},
			},
		},
		{
			name: "assign-to-user with target and target-repo",
			safeOutputs: map[string]any{
				"assign-to-user": map[string]any{
					"target":        "*",
					"target-repo":   "github/github",
					"allowed-repos": []any{"github/docs"},
				},
			},
		},
		{
			name: "unassign-from-user with target and target-repo",
			safeOutputs: map[string]any{
				"unassign-from-user": map[string]any{
					"target":        "*",
					"target-repo":   "github/github",
					"allowed-repos": []any{"github/docs"},
				},
			},
		},
		{
			name: "assign-to-agent with target and target-repo",
			safeOutputs: map[string]any{
				"assign-to-agent": map[string]any{
					"target":        "*",
					"target-repo":   "github/github",
					"allowed-repos": []any{"github/docs"},
				},
			},
		},
		{
			name: "assign-milestone with target and target-repo",
			safeOutputs: map[string]any{
				"assign-milestone": map[string]any{
					"allowed":       []any{"v1.0", "v2.0"},
					"target":        "*",
					"target-repo":   "github/github",
					"allowed-repos": []any{"github/docs"},
				},
			},
		},
		{
			name: "set-issue-type with target and target-repo",
			safeOutputs: map[string]any{
				"set-issue-type": map[string]any{
					"target":        "*",
					"target-repo":   "github/github",
					"allowed-repos": []any{"github/docs"},
				},
			},
		},
		{
			name: "set-issue-field with target and target-repo",
			safeOutputs: map[string]any{
				"set-issue-field": map[string]any{
					"target":        "*",
					"target-repo":   "github/github",
					"allowed-repos": []any{"github/docs"},
				},
			},
		},
		{
			name: "add-reviewer with target and target-repo",
			safeOutputs: map[string]any{
				"add-reviewer": map[string]any{
					"target":        "*",
					"target-repo":   "github/github",
					"allowed-repos": []any{"github/docs"},
				},
			},
		},
		{
			name: "hide-comment with target and target-repo",
			safeOutputs: map[string]any{
				"hide-comment": map[string]any{
					"max":           5,
					"target":        "*",
					"target-repo":   "github/github",
					"allowed-repos": []any{"github/docs"},
				},
			},
		},
		{
			name: "link-sub-issue with target and target-repo",
			safeOutputs: map[string]any{
				"link-sub-issue": map[string]any{
					"max":           5,
					"target":        "*",
					"target-repo":   "github/github",
					"allowed-repos": []any{"github/docs"},
				},
			},
		},
		{
			name: "reply-to-pull-request-review-comment with target and target-repo",
			safeOutputs: map[string]any{
				"reply-to-pull-request-review-comment": map[string]any{
					"max":           10,
					"target":        "*",
					"target-repo":   "github/github",
					"allowed-repos": []any{"github/docs"},
				},
			},
		},
		{
			name: "mark-pull-request-as-ready-for-review with target and target-repo",
			safeOutputs: map[string]any{
				"mark-pull-request-as-ready-for-review": map[string]any{
					"target":        "*",
					"target-repo":   "github/github",
					"allowed-repos": []any{"github/docs"},
				},
			},
		},
		{
			name: "create-pull-request-review-comment with target and target-repo",
			safeOutputs: map[string]any{
				"create-pull-request-review-comment": map[string]any{
					"max":           10,
					"target":        "*",
					"target-repo":   "github/github",
					"allowed-repos": []any{"github/docs"},
				},
			},
		},
		{
			name: "merge-pull-request with target, target-repo, and samples",
			safeOutputs: map[string]any{
				"merge-pull-request": map[string]any{
					"target":        "*",
					"target-repo":   "github/github",
					"allowed-repos": []any{"github/docs"},
					"samples": []any{
						map[string]any{
							"merge_method": "squash",
						},
					},
				},
			},
		},
		{
			name: "dispatch-workflow with target-repo and allowed-repos",
			safeOutputs: map[string]any{
				"dispatch-workflow": map[string]any{
					"workflows":     []any{"worker"},
					"target-repo":   "github/github",
					"allowed-repos": []any{"github/docs"},
				},
			},
		},
		{
			name: "push-to-pull-request-branch with target and target-repo",
			safeOutputs: map[string]any{
				"push-to-pull-request-branch": map[string]any{
					"target":        "*",
					"target-repo":   "github/github",
					"allowed-repos": []any{"github/docs"},
				},
			},
		},
		{
			name: "issue-intent toggle accepted for close and assignment tools",
			safeOutputs: map[string]any{
				"close-issue": map[string]any{
					"issue-intent": false,
				},
				"assign-to-user": map[string]any{
					"issue-intent": false,
				},
				"assign-to-agent": map[string]any{
					"issue-intent": false,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			frontmatter := baseFrontmatter(tt.safeOutputs)
			err := ValidateMainWorkflowFrontmatterWithSchemaAndLocation(frontmatter, "/tmp/gh-aw/safe-outputs-target-schema-test.md")
			if err != nil {
				t.Fatalf("expected safe output config to pass schema validation, got: %v", err)
			}
		})
	}
}

// TestMainWorkflowSchema_SafeOutputsRejectsUnknownProperties verifies that the schema
// correctly rejects unknown properties in safe output configurations (the additionalProperties: false
// constraint is working).
func TestMainWorkflowSchema_SafeOutputsRejectsUnknownProperties(t *testing.T) {
	t.Parallel()

	baseFrontmatter := func(safeOutputs map[string]any) map[string]any {
		return map[string]any{
			"on":           map[string]any{"issues": map[string]any{"types": []any{"opened"}}},
			"engine":       "copilot",
			"safe-outputs": safeOutputs,
		}
	}

	tests := []struct {
		name        string
		safeOutputs map[string]any
		errContains string
	}{
		{
			name: "resolve-pull-request-review-thread rejects unknown property",
			safeOutputs: map[string]any{
				"resolve-pull-request-review-thread": map[string]any{
					"max":           10,
					"unknown-field": "value",
				},
			},
			errContains: "unknown-field",
		},
		{
			name: "assign-milestone rejects unknown property",
			safeOutputs: map[string]any{
				"assign-milestone": map[string]any{
					"bogus-property": true,
				},
			},
			errContains: "bogus-property",
		},
		{
			name: "hide-comment rejects unknown property",
			safeOutputs: map[string]any{
				"hide-comment": map[string]any{
					"invalid-key": "value",
				},
			},
			errContains: "invalid-key",
		},
		{
			name: "link-sub-issue rejects unknown property",
			safeOutputs: map[string]any{
				"link-sub-issue": map[string]any{
					"not-a-field": 42,
				},
			},
			errContains: "not-a-field",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			frontmatter := baseFrontmatter(tt.safeOutputs)
			err := ValidateMainWorkflowFrontmatterWithSchemaAndLocation(frontmatter, "/tmp/gh-aw/safe-outputs-reject-test.md")
			if err == nil {
				t.Fatal("expected schema validation to reject unknown properties, but it passed")
			}
			if tt.errContains != "" {
				errStr := err.Error()
				if !contains(errStr, tt.errContains) {
					t.Errorf("expected error to contain %q, got: %v", tt.errContains, err)
				}
			}
		})
	}
}
