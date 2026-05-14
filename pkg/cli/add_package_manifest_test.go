//go:build !integration

package cli

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveRepositoryPackage(t *testing.T) {
	originalDownload := downloadPackageFileFromGitHubForHost
	originalList := listPackageWorkflowFiles
	t.Cleanup(func() {
		downloadPackageFileFromGitHubForHost = originalDownload
		listPackageWorkflowFiles = originalList
	})

	t.Run("uses aw manifest files and explicit docs", func(t *testing.T) {
		downloadPackageFileFromGitHubForHost = func(owner, repo, path, ref, host string) ([]byte, error) {
			switch path {
			case "aw.yml":
				return []byte(`name: Repo Assist
description: Friendly repository automation
docs: docs/overview.md
files:
  - workflows/review.md
  - .github/workflows/nightly-review.md
  - README.md
`), nil
			default:
				return nil, fmt.Errorf("404 not found: %s", path)
			}
		}
		listPackageWorkflowFiles = func(owner, repo, ref, workflowPath string) ([]string, error) {
			t.Fatalf("unexpected scan of %s", workflowPath)
			return nil, nil
		}

		pkg, err := resolveRepositoryPackage(&RepoSpec{RepoSlug: "owner/repo"}, "")
		require.NoError(t, err)
		assert.Equal(t, "aw.yml", pkg.ManifestPath)
		assert.Equal(t, "Repo Assist", pkg.Name)
		assert.Equal(t, "docs/overview.md", pkg.DocsPath)
		assert.Equal(t, []string{"workflows/review.md", ".github/workflows/nightly-review.md"}, pkg.InstallationSource)
		assert.Contains(t, pkg.Warnings[0], "Ignoring files entry")
	})

	t.Run("falls back to scanning supported workflow directories", func(t *testing.T) {
		downloadPackageFileFromGitHubForHost = func(owner, repo, path, ref, host string) ([]byte, error) {
			switch path {
			case "aw.yml":
				return []byte("name: Repo Assist\n"), nil
			case "docs/repo-assist.md":
				return []byte("# Repo Assist\n"), nil
			default:
				return nil, fmt.Errorf("404 not found: %s", path)
			}
		}
		listPackageWorkflowFiles = func(owner, repo, ref, workflowPath string) ([]string, error) {
			switch workflowPath {
			case "workflows":
				return []string{"workflows/review.md"}, nil
			case ".github/workflows":
				return []string{".github/workflows/nightly-review.md"}, nil
			default:
				return nil, fmt.Errorf("unexpected workflow path %s", workflowPath)
			}
		}

		pkg, err := resolveRepositoryPackage(&RepoSpec{RepoSlug: "owner/repo"}, "")
		require.NoError(t, err)
		assert.Equal(t, "docs/repo-assist.md", pkg.DocsPath)
		assert.Equal(t, []string{"workflows/review.md", ".github/workflows/nightly-review.md"}, pkg.InstallationSource)
	})

	t.Run("rejects manifest without name field", func(t *testing.T) {
		downloadPackageFileFromGitHubForHost = func(owner, repo, path, ref, host string) ([]byte, error) {
			if path == "aw.yml" {
				return []byte("description: missing name\n"), nil
			}
			return nil, fmt.Errorf("404 not found: %s", path)
		}
		listPackageWorkflowFiles = func(owner, repo, ref, workflowPath string) ([]string, error) {
			t.Fatalf("unexpected scan of %s", workflowPath)
			return nil, nil
		}

		_, err := resolveRepositoryPackage(&RepoSpec{RepoSlug: "owner/repo"}, "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), `name must be a non-empty string`)
	})
}

func TestResolveWorkflows_RepositoryPackage(t *testing.T) {
	originalFetchFn := fetchWorkflowFromSourceWithContextFn
	originalDownload := downloadPackageFileFromGitHubForHost
	originalList := listPackageWorkflowFiles
	t.Cleanup(func() {
		fetchWorkflowFromSourceWithContextFn = originalFetchFn
		downloadPackageFileFromGitHubForHost = originalDownload
		listPackageWorkflowFiles = originalList
	})

	downloadPackageFileFromGitHubForHost = func(owner, repo, path, ref, host string) ([]byte, error) {
		if path == "aw.yml" {
			return []byte(`name: Repo Assist
files:
  - workflows/review.md
  - .github/workflows/nightly-review.md
`), nil
		}
		return nil, fmt.Errorf("404 not found: %s", path)
	}
	listPackageWorkflowFiles = func(owner, repo, ref, workflowPath string) ([]string, error) {
		t.Fatalf("unexpected scan of %s", workflowPath)
		return nil, nil
	}
	fetchWorkflowFromSourceWithContextFn = func(_ context.Context, spec *WorkflowSpec, _ bool) (*FetchedWorkflow, error) {
		return &FetchedWorkflow{
			Content:    []byte("---\nname: Test\non: push\n---\n"),
			CommitSHA:  "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			IsLocal:    false,
			SourcePath: spec.WorkflowPath,
		}, nil
	}

	resolved, err := ResolveWorkflows(context.Background(), []string{"owner/repo"}, false)
	require.NoError(t, err)
	require.Len(t, resolved.Workflows, 2)
	assert.Equal(t, "workflows/review.md", resolved.Workflows[0].Spec.WorkflowPath)
	assert.Equal(t, ".github/workflows/nightly-review.md", resolved.Workflows[1].Spec.WorkflowPath)
}
