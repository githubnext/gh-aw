//go:build !integration

package cli

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveAddWorkflowSpecAndContent(t *testing.T) {
	originalFetchFn := fetchWorkflowFromSourceWithContextFn
	t.Cleanup(func() {
		fetchWorkflowFromSourceWithContextFn = originalFetchFn
	})

	t.Run("follows redirect chain for remote workflows", func(t *testing.T) {
		fetchWorkflowFromSourceWithContextFn = func(_ context.Context, spec *WorkflowSpec, _ bool) (*FetchedWorkflow, error) {
			switch spec.String() {
			case "owner/repo/workflows/original.md@main":
				return &FetchedWorkflow{
					Content:    []byte("---\nredirect: owner/repo/workflows/new.md@main\n---\n"),
					IsLocal:    false,
					SourcePath: "workflows/original.md",
				}, nil
			case "owner/repo/workflows/new.md@main":
				return &FetchedWorkflow{
					Content:    []byte("---\nname: New Workflow\n---\n"),
					IsLocal:    false,
					SourcePath: "workflows/new.md",
				}, nil
			default:
				return nil, fmt.Errorf("unexpected fetch spec %s", spec.String())
			}
		}

		resolvedSpec, fetched, err := resolveAddWorkflowSpecAndContent(context.Background(), &WorkflowSpec{
			RepoSpec: RepoSpec{
				RepoSlug: "owner/repo",
				Version:  "main",
			},
			WorkflowPath: "workflows/original.md",
			WorkflowName: "original",
		}, false)
		require.NoError(t, err, "redirect chain should resolve")
		require.NotNil(t, fetched, "resolved fetch should be returned")
		assert.Equal(t, "owner/repo", resolvedSpec.RepoSlug, "final repo should be redirected repo")
		assert.Equal(t, "workflows/new.md", resolvedSpec.WorkflowPath, "final path should be redirected path")
		assert.Equal(t, "original", resolvedSpec.WorkflowName, "workflow name should be preserved from the original request")
	})

	t.Run("detects redirect loops", func(t *testing.T) {
		fetchWorkflowFromSourceWithContextFn = func(_ context.Context, spec *WorkflowSpec, _ bool) (*FetchedWorkflow, error) {
			switch spec.String() {
			case "owner/repo/workflows/a.md@main":
				return &FetchedWorkflow{Content: []byte("---\nredirect: owner/repo/workflows/b.md@main\n---\n"), IsLocal: false, SourcePath: "workflows/a.md"}, nil
			case "owner/repo/workflows/b.md@main":
				return &FetchedWorkflow{Content: []byte("---\nredirect: owner/repo/workflows/a.md@main\n---\n"), IsLocal: false, SourcePath: "workflows/b.md"}, nil
			default:
				return nil, fmt.Errorf("unexpected fetch spec %s", spec.String())
			}
		}

		_, _, err := resolveAddWorkflowSpecAndContent(context.Background(), &WorkflowSpec{
			RepoSpec:     RepoSpec{RepoSlug: "owner/repo", Version: "main"},
			WorkflowPath: "workflows/a.md",
			WorkflowName: "a",
		}, false)
		require.Error(t, err, "redirect loop should fail")
		assert.Contains(t, err.Error(), "redirect loop detected", "error should mention loop detection")
	})

	t.Run("local workflows are not redirected", func(t *testing.T) {
		fetchWorkflowFromSourceWithContextFn = func(_ context.Context, _ *WorkflowSpec, _ bool) (*FetchedWorkflow, error) {
			return &FetchedWorkflow{
				Content:    []byte("---\nredirect: owner/repo/workflows/new.md@main\n---\n"),
				IsLocal:    true,
				SourcePath: "./local.md",
			}, nil
		}

		resolvedSpec, _, err := resolveAddWorkflowSpecAndContent(context.Background(), &WorkflowSpec{
			WorkflowPath: "./local.md",
			WorkflowName: "local",
		}, false)
		require.NoError(t, err, "local workflow resolution should succeed")
		assert.Equal(t, "./local.md", resolvedSpec.WorkflowPath, "local workflow path should be preserved")
	})

	t.Run("ResolveWorkflows uses redirected spec", func(t *testing.T) {
		fetchWorkflowFromSourceWithContextFn = func(_ context.Context, spec *WorkflowSpec, _ bool) (*FetchedWorkflow, error) {
			switch spec.String() {
			case "owner/repo/workflows/original.md@main":
				return &FetchedWorkflow{Content: []byte("---\nredirect: owner/repo/workflows/new.md@main\n---\n"), IsLocal: false}, nil
			case "owner/repo/workflows/new.md@main":
				return &FetchedWorkflow{Content: []byte("---\nname: New Workflow\non: push\n---\n"), IsLocal: false}, nil
			default:
				return nil, fmt.Errorf("unexpected fetch spec %s", spec.String())
			}
		}

		resolved, err := ResolveWorkflows(context.Background(), []string{"owner/repo/workflows/original.md@main"}, false)
		require.NoError(t, err, "ResolveWorkflows should follow redirects")
		require.Len(t, resolved.Workflows, 1, "one workflow should resolve")
		assert.Equal(t, "workflows/new.md", resolved.Workflows[0].Spec.WorkflowPath, "resolved spec should point to redirect destination")
		assert.Equal(t, "original", resolved.Workflows[0].Spec.WorkflowName, "resolved workflow name should be preserved from the original request")
	})

	t.Run("ResolveWorkflows adds JSON import refinement suggestion", func(t *testing.T) {
		fetchWorkflowFromSourceWithContextFn = func(_ context.Context, spec *WorkflowSpec, _ bool) (*FetchedWorkflow, error) {
			// Simulate JSON conversion selecting a better local filename from
			// the source payload's "name" field.
			spec.WorkflowName = "haiku"
			return &FetchedWorkflow{
				Content:           []byte("---\ndescription: Imported workflow\non: push\n---\n"),
				IsLocal:           true,
				SourcePath:        "https://example.com/b5a3f76a-3d8f-4790-b7e2-f2886f784345.json",
				ConvertedFromJSON: true,
			}, nil
		}

		resolved, err := ResolveWorkflows(context.Background(), []string{"https://example.com/b5a3f76a-3d8f-4790-b7e2-f2886f784345.json"}, false)
		require.NoError(t, err, "ResolveWorkflows should accept converted JSON workflow payloads")
		require.NotNil(t, resolved, "resolved workflows should be returned")
		require.Len(t, resolved.Workflows, 1, "one workflow should resolve")
		assert.Equal(t, "haiku", resolved.Workflows[0].Spec.WorkflowName, "resolved workflow name should come from JSON name")
		require.NotEmpty(t, resolved.Warnings, "JSON conversion should surface follow-up suggestion")
		assert.Condition(t, func() bool {
			for _, warning := range resolved.Warnings {
				if strings.Contains(warning, `JSON workflow import for "haiku" was best-effort`) &&
					strings.Contains(warning, "run an agentic prompt") &&
					strings.Contains(warning, ".github/workflows/haiku.md") {
					return true
				}
			}
			return false
		}, "should suggest agentic refinement after JSON import using workflow name")
	})
}
