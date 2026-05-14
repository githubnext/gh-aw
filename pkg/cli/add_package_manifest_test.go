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
	originalVersion := GetVersion()
	originalDownload := downloadPackageFileFromGitHubForHost
	originalList := listPackageWorkflowFiles
	t.Cleanup(func() {
		SetVersionInfo(originalVersion)
		downloadPackageFileFromGitHubForHost = originalDownload
		listPackageWorkflowFiles = originalList
	})
	SetVersionInfo("v1.2.3")

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

	t.Run("accepts schema-version and compatible min-version", func(t *testing.T) {
		downloadPackageFileFromGitHubForHost = func(owner, repo, path, ref, host string) ([]byte, error) {
			switch path {
			case "aw.yml":
				return []byte(`schema-version: "1"
min-version: v1.0.0
name: Repo Assist
files:
  - workflows/review.md
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
		assert.Equal(t, []string{"workflows/review.md"}, pkg.InstallationSource)
	})

	t.Run("rejects unsupported schema-version", func(t *testing.T) {
		downloadPackageFileFromGitHubForHost = func(owner, repo, path, ref, host string) ([]byte, error) {
			if path == "aw.yml" {
				return []byte(`schema-version: "2"
name: Repo Assist
`), nil
			}
			return nil, fmt.Errorf("404 not found: %s", path)
		}

		_, err := resolveRepositoryPackage(&RepoSpec{RepoSlug: "owner/repo"}, "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), `schema-version`)
	})

	t.Run("rejects incompatible min-version", func(t *testing.T) {
		downloadPackageFileFromGitHubForHost = func(owner, repo, path, ref, host string) ([]byte, error) {
			if path == "aw.yml" {
				return []byte(`min-version: v9.9.9
name: Repo Assist
`), nil
			}
			return nil, fmt.Errorf("404 not found: %s", path)
		}

		_, err := resolveRepositoryPackage(&RepoSpec{RepoSlug: "owner/repo"}, "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), `requires gh-aw`)
	})

	t.Run("rejects unknown manifest fields", func(t *testing.T) {
		downloadPackageFileFromGitHubForHost = func(owner, repo, path, ref, host string) ([]byte, error) {
			if path == "aw.yml" {
				return []byte(`name: Repo Assist
unknown-field: true
`), nil
			}
			return nil, fmt.Errorf("404 not found: %s", path)
		}

		_, err := resolveRepositoryPackage(&RepoSpec{RepoSlug: "owner/repo"}, "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), `unknown-field`)
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
