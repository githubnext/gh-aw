//go:build !integration

package cli

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createRepositoryPackageNotFoundError(path string) error {
	return normalizeRepositoryPackageRemoteError(fmt.Errorf("404 not found: %s", path))
}

func TestResolveRepositoryPackage(t *testing.T) {
	originalVersion := GetVersion()
	originalDownload := downloadPackageFileFromGitHubForHost
	originalList := listPackageWorkflowFilesForHost
	originalDirFiles := listPackageDirFilesForHost
	originalDirSubdirs := listPackageDirSubdirsForHost
	originalDefaultBranch := getRepositoryPackageDefaultBranch
	originalLatestRelease := getRepositoryPackageLatestRelease
	t.Cleanup(func() {
		SetVersionInfo(originalVersion)
		downloadPackageFileFromGitHubForHost = originalDownload
		listPackageWorkflowFilesForHost = originalList
		listPackageDirFilesForHost = originalDirFiles
		listPackageDirSubdirsForHost = originalDirSubdirs
		getRepositoryPackageDefaultBranch = originalDefaultBranch
		getRepositoryPackageLatestRelease = originalLatestRelease
	})
	SetVersionInfo("v1.2.3")
	getRepositoryPackageDefaultBranch = func(repoSlug, host string) (string, error) {
		return "main", nil
	}

	getRepositoryPackageLatestRelease = func(repoSlug, host string) (string, error) {
		return "", errors.New("no releases found")
	}
	listPackageDirFilesForHost = func(owner, repo, ref, dirPath, host string) ([]string, error) {
		return nil, createRepositoryPackageNotFoundError(dirPath)
	}
	listPackageDirSubdirsForHost = func(owner, repo, ref, dirPath, host string) ([]string, error) {
		return nil, createRepositoryPackageNotFoundError(dirPath)
	}

	t.Run("uses aw manifest files and README docs", func(t *testing.T) {
		downloadPackageFileFromGitHubForHost = func(owner, repo, path, ref, host string) ([]byte, error) {
			switch path {
			case "aw.yml":
				return []byte(`name: Repo Assist
emoji: 🤖
description: Friendly repository automation
license: MIT
files:
  - workflows/review.md
  - .github/workflows/nightly-review.md
  - README.md
`), nil
			case "README.md":
				return []byte("# Repo Assist\n"), nil
			default:
				return nil, createRepositoryPackageNotFoundError(path)
			}
		}
		listPackageWorkflowFilesForHost = func(owner, repo, ref, workflowPath, host string) ([]string, error) {
			t.Fatalf("unexpected scan of %s", workflowPath)
			return nil, nil
		}

		pkg, err := resolveRepositoryPackage(&RepoSpec{RepoSlug: "owner/repo"}, "")
		require.NoError(t, err)
		assert.Equal(t, "aw.yml", pkg.ManifestPath)
		assert.Equal(t, "Repo Assist", pkg.Name)
		assert.Equal(t, "🤖", pkg.Emoji)
		assert.Equal(t, "MIT", pkg.License)
		assert.Equal(t, "README.md", pkg.DocsPath)
		assert.Equal(t, []string{"workflows/review.md", ".github/workflows/nightly-review.md"}, pkg.InstallationSource)
		require.NotEmpty(t, pkg.Warnings)
		assert.Contains(t, strings.Join(pkg.Warnings, "\n"), "Ignoring files entry")
	})

	t.Run("uses repository default branch when version is omitted", func(t *testing.T) {
		previousDefaultBranch := getRepositoryPackageDefaultBranch
		previousLatestRelease := getRepositoryPackageLatestRelease
		t.Cleanup(func() {
			getRepositoryPackageDefaultBranch = previousDefaultBranch
			getRepositoryPackageLatestRelease = previousLatestRelease
		})
		getRepositoryPackageLatestRelease = func(repoSlug, host string) (string, error) {
			assert.Equal(t, "owner/repo", repoSlug)
			assert.Equal(t, "github.com", host)
			return "", errors.New("no releases found")
		}
		getRepositoryPackageDefaultBranch = func(repoSlug, host string) (string, error) {
			assert.Equal(t, "owner/repo", repoSlug)
			assert.Equal(t, "github.com", host)
			return "master", nil
		}
		downloadPackageFileFromGitHubForHost = func(owner, repo, path, ref, host string) ([]byte, error) {
			assert.Equal(t, "master", ref)
			switch path {
			case "aw.yml":
				return []byte("name: Repo Assist\n"), nil
			case "README.md":
				return []byte("# Repo Assist\n"), nil
			default:
				return nil, createRepositoryPackageNotFoundError(path)
			}
		}
		listPackageWorkflowFilesForHost = func(owner, repo, ref, workflowPath, host string) ([]string, error) {
			assert.Equal(t, "master", ref)
			switch workflowPath {
			case "workflows":
				return []string{"workflows/review.md"}, nil
			case ".github/workflows":
				return nil, createRepositoryPackageNotFoundError(workflowPath)
			default:
				return nil, fmt.Errorf("unexpected workflow path %s", workflowPath)
			}
		}

		pkg, err := resolveRepositoryPackage(&RepoSpec{RepoSlug: "owner/repo"}, "github.com")
		require.NoError(t, err)
		assert.Equal(t, []string{"workflows/review.md"}, pkg.InstallationSource)
	})

	t.Run("uses latest release for github/gh-aw when version is omitted", func(t *testing.T) {
		previousDefaultBranch := getRepositoryPackageDefaultBranch
		previousLatestRelease := getRepositoryPackageLatestRelease
		t.Cleanup(func() {
			getRepositoryPackageDefaultBranch = previousDefaultBranch
			getRepositoryPackageLatestRelease = previousLatestRelease
		})
		getRepositoryPackageLatestRelease = func(repoSlug, host string) (string, error) {
			assert.Equal(t, "github/gh-aw", repoSlug)
			assert.Equal(t, "github.com", host)
			return "v1.2.3", nil
		}
		getRepositoryPackageDefaultBranch = func(repoSlug, host string) (string, error) {
			t.Fatalf("default branch lookup should not be called when latest release is available")
			return "", nil
		}
		downloadPackageFileFromGitHubForHost = func(owner, repo, path, ref, host string) ([]byte, error) {
			assert.Equal(t, "v1.2.3", ref)
			switch path {
			case "aw.yml":
				return []byte("name: gh-aw package\n"), nil
			case "README.md":
				return []byte("# gh-aw package\n"), nil
			default:
				return nil, createRepositoryPackageNotFoundError(path)
			}
		}
		listPackageWorkflowFilesForHost = func(owner, repo, ref, workflowPath, host string) ([]string, error) {
			assert.Equal(t, "v1.2.3", ref)
			switch workflowPath {
			case "workflows":
				return []string{"workflows/review.md"}, nil
			case ".github/workflows":
				return nil, createRepositoryPackageNotFoundError(workflowPath)
			default:
				return nil, fmt.Errorf("unexpected workflow path %s", workflowPath)
			}
		}

		pkg, err := resolveRepositoryPackage(&RepoSpec{RepoSlug: "github/gh-aw"}, "github.com")
		require.NoError(t, err)
		assert.Equal(t, "v1.2.3", pkg.ResolvedRef)
		assert.Equal(t, []string{"workflows/review.md"}, pkg.InstallationSource)
	})

	t.Run("falls back to default branch for github/gh-aw when latest release lookup fails", func(t *testing.T) {
		previousDefaultBranch := getRepositoryPackageDefaultBranch
		previousLatestRelease := getRepositoryPackageLatestRelease
		t.Cleanup(func() {
			getRepositoryPackageDefaultBranch = previousDefaultBranch
			getRepositoryPackageLatestRelease = previousLatestRelease
		})
		getRepositoryPackageLatestRelease = func(repoSlug, host string) (string, error) {
			assert.Equal(t, "github/gh-aw", repoSlug)
			assert.Equal(t, "github.com", host)
			return "", errors.New("release lookup failed")
		}
		getRepositoryPackageDefaultBranch = func(repoSlug, host string) (string, error) {
			assert.Equal(t, "github/gh-aw", repoSlug)
			assert.Equal(t, "github.com", host)
			return "main", nil
		}
		downloadPackageFileFromGitHubForHost = func(owner, repo, path, ref, host string) ([]byte, error) {
			assert.Equal(t, "main", ref)
			switch path {
			case "aw.yml":
				return []byte("name: gh-aw package\n"), nil
			case "README.md":
				return []byte("# gh-aw package\n"), nil
			default:
				return nil, createRepositoryPackageNotFoundError(path)
			}
		}
		listPackageWorkflowFilesForHost = func(owner, repo, ref, workflowPath, host string) ([]string, error) {
			assert.Equal(t, "main", ref)
			switch workflowPath {
			case "workflows":
				return []string{"workflows/review.md"}, nil
			case ".github/workflows":
				return nil, createRepositoryPackageNotFoundError(workflowPath)
			default:
				return nil, fmt.Errorf("unexpected workflow path %s", workflowPath)
			}
		}

		pkg, err := resolveRepositoryPackage(&RepoSpec{RepoSlug: "github/gh-aw"}, "github.com")
		require.NoError(t, err)
		assert.Equal(t, "main", pkg.ResolvedRef)
		assert.Equal(t, []string{"workflows/review.md"}, pkg.InstallationSource)
	})

	t.Run("uses slash branch ref from manifest route", func(t *testing.T) {
		previousDefaultBranch := getRepositoryPackageDefaultBranch
		t.Cleanup(func() {
			getRepositoryPackageDefaultBranch = previousDefaultBranch
		})
		getRepositoryPackageDefaultBranch = func(repoSlug, host string) (string, error) {
			t.Fatalf("default branch lookup should not be called when version is provided")
			return "", nil
		}
		downloadPackageFileFromGitHubForHost = func(owner, repo, path, ref, host string) ([]byte, error) {
			assert.Equal(t, "feature/github-agentic-workflow", ref)
			switch path {
			case "agentic-workflows/aw.yml":
				return []byte("name: Repo Assist\nfiles:\n  - workflows/review.md\n"), nil
			case "agentic-workflows/README.md":
				return []byte("# Repo Assist\n"), nil
			default:
				return nil, createRepositoryPackageNotFoundError(path)
			}
		}
		listPackageWorkflowFilesForHost = func(owner, repo, ref, workflowPath, host string) ([]string, error) {
			t.Fatalf("unexpected scan of %s", workflowPath)
			return nil, nil
		}

		pkg, err := resolveRepositoryPackage(&RepoSpec{
			RepoSlug:    "owner/repo",
			PackagePath: "agentic-workflows",
			Version:     "feature/github-agentic-workflow",
		}, "github.com")
		require.NoError(t, err)
		assert.Equal(t, []string{"agentic-workflows/workflows/review.md"}, pkg.InstallationSource)
	})

	t.Run("falls back to scanning supported workflow directories", func(t *testing.T) {
		downloadPackageFileFromGitHubForHost = func(owner, repo, path, ref, host string) ([]byte, error) {
			switch path {
			case "aw.yml":
				return []byte("name: Repo Assist\n"), nil
			case "README.md":
				return []byte("# Repo Assist\n"), nil
			default:
				return nil, createRepositoryPackageNotFoundError(path)
			}
		}
		listPackageWorkflowFilesForHost = func(owner, repo, ref, workflowPath, host string) ([]string, error) {
			assert.Empty(t, host)
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
		assert.Equal(t, "README.md", pkg.DocsPath)
		assert.Equal(t, []string{"workflows/review.md", ".github/workflows/nightly-review.md"}, pkg.InstallationSource)
	})

	t.Run("passes explicit host to scanning fallback", func(t *testing.T) {
		downloadPackageFileFromGitHubForHost = func(owner, repo, path, ref, host string) ([]byte, error) {
			switch path {
			case "aw.yml":
				assert.Equal(t, "github.com", host)
				return []byte("name: Repo Assist\n"), nil
			case "README.md":
				assert.Equal(t, "github.com", host)
				return []byte("# Repo Assist\n"), nil
			default:
				return nil, createRepositoryPackageNotFoundError(path)
			}
		}
		listPackageWorkflowFilesForHost = func(owner, repo, ref, workflowPath, host string) ([]string, error) {
			assert.Equal(t, "github.com", host)
			switch workflowPath {
			case "workflows":
				return []string{"workflows/review.md"}, nil
			case ".github/workflows":
				return nil, createRepositoryPackageNotFoundError(workflowPath)
			default:
				return nil, fmt.Errorf("unexpected workflow path %s", workflowPath)
			}
		}

		pkg, err := resolveRepositoryPackage(&RepoSpec{RepoSlug: "owner/repo"}, "github.com")
		require.NoError(t, err)
		assert.Equal(t, []string{"workflows/review.md"}, pkg.InstallationSource)
	})

	t.Run("rejects manifest without name field", func(t *testing.T) {
		downloadPackageFileFromGitHubForHost = func(owner, repo, path, ref, host string) ([]byte, error) {
			if path == "aw.yml" {
				return []byte("description: missing name\n"), nil
			}
			return nil, createRepositoryPackageNotFoundError(path)
		}
		listPackageWorkflowFilesForHost = func(owner, repo, ref, workflowPath, host string) ([]string, error) {
			t.Fatalf("unexpected scan of %s", workflowPath)
			return nil, nil
		}

		_, err := resolveRepositoryPackage(&RepoSpec{RepoSlug: "owner/repo"}, "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), `name must be a non-empty string`)
	})

	t.Run("requires aw manifest when only legacy alias exists", func(t *testing.T) {
		var requestedPaths []string
		downloadPackageFileFromGitHubForHost = func(owner, repo, path, ref, host string) ([]byte, error) {
			requestedPaths = append(requestedPaths, path)
			if path == "agents.yml" {
				return []byte("name: Legacy Alias\n"), nil
			}
			return nil, createRepositoryPackageNotFoundError(path)
		}
		listPackageWorkflowFilesForHost = func(owner, repo, ref, workflowPath, host string) ([]string, error) {
			t.Fatalf("unexpected scan of %s", workflowPath)
			return nil, nil
		}

		_, err := resolveRepositoryPackage(&RepoSpec{RepoSlug: "owner/repo"}, "")
		require.Error(t, err)
		assert.Equal(t, []string{"aw.yml"}, requestedPaths)
		assert.Contains(t, err.Error(), `no aw.yml manifest found`)
	})

	t.Run("accepts manifest-version and compatible min-version", func(t *testing.T) {
		downloadPackageFileFromGitHubForHost = func(owner, repo, path, ref, host string) ([]byte, error) {
			switch path {
			case "aw.yml":
				return []byte(`manifest-version: "1"
min-version: v1.0.0
name: Repo Assist
files:
  - workflows/review.md
`), nil
			case "README.md":
				return []byte("# Repo Assist\n"), nil
			default:
				return nil, createRepositoryPackageNotFoundError(path)
			}
		}
		listPackageWorkflowFilesForHost = func(owner, repo, ref, workflowPath, host string) ([]string, error) {
			t.Fatalf("unexpected scan of %s", workflowPath)
			return nil, nil
		}

		pkg, err := resolveRepositoryPackage(&RepoSpec{RepoSlug: "owner/repo"}, "")
		require.NoError(t, err)
		assert.Equal(t, []string{"workflows/review.md"}, pkg.InstallationSource)
	})

	t.Run("rejects unsupported manifest-version", func(t *testing.T) {
		downloadPackageFileFromGitHubForHost = func(owner, repo, path, ref, host string) ([]byte, error) {
			if path == "aw.yml" {
				return []byte(`manifest-version: "2"
name: Repo Assist
`), nil
			}
			return nil, createRepositoryPackageNotFoundError(path)
		}

		_, err := resolveRepositoryPackage(&RepoSpec{RepoSlug: "owner/repo"}, "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), `manifest-version`)
	})

	t.Run("rejects docs field", func(t *testing.T) {
		downloadPackageFileFromGitHubForHost = func(owner, repo, path, ref, host string) ([]byte, error) {
			if path == "aw.yml" {
				return []byte(`name: Repo Assist
docs: docs/overview.md
`), nil
			}
			return nil, createRepositoryPackageNotFoundError(path)
		}

		_, err := resolveRepositoryPackage(&RepoSpec{RepoSlug: "owner/repo"}, "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), `docs`)
	})

	t.Run("rejects non-string emoji field", func(t *testing.T) {
		downloadPackageFileFromGitHubForHost = func(owner, repo, path, ref, host string) ([]byte, error) {
			if path == "aw.yml" {
				return []byte(`name: Repo Assist
emoji:
  icon: 🤖
`), nil
			}
			return nil, createRepositoryPackageNotFoundError(path)
		}

		_, err := resolveRepositoryPackage(&RepoSpec{RepoSlug: "owner/repo"}, "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), `emoji`)
	})

	t.Run("rejects non-string license field", func(t *testing.T) {
		downloadPackageFileFromGitHubForHost = func(owner, repo, path, ref, host string) ([]byte, error) {
			if path == "aw.yml" {
				return []byte(`name: Repo Assist
license:
  id: MIT
`), nil
			}
			return nil, createRepositoryPackageNotFoundError(path)
		}

		_, err := resolveRepositoryPackage(&RepoSpec{RepoSlug: "owner/repo"}, "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), `license`)
	})

	t.Run("rejects incompatible min-version", func(t *testing.T) {
		downloadPackageFileFromGitHubForHost = func(owner, repo, path, ref, host string) ([]byte, error) {
			if path == "aw.yml" {
				return []byte(`min-version: v9.9.9
name: Repo Assist
`), nil
			}
			return nil, createRepositoryPackageNotFoundError(path)
		}

		_, err := resolveRepositoryPackage(&RepoSpec{RepoSlug: "owner/repo"}, "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), `requires gh-aw`)
	})

	t.Run("requires package README", func(t *testing.T) {
		downloadPackageFileFromGitHubForHost = func(owner, repo, path, ref, host string) ([]byte, error) {
			if path == "aw.yml" {
				return []byte(`name: Repo Assist
files:
  - workflows/review.md
`), nil
			}
			return nil, createRepositoryPackageNotFoundError(path)
		}
		listPackageWorkflowFilesForHost = func(owner, repo, ref, workflowPath, host string) ([]string, error) {
			t.Fatalf("unexpected scan of %s", workflowPath)
			return nil, nil
		}

		_, err := resolveRepositoryPackage(&RepoSpec{RepoSlug: "owner/repo"}, "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), `missing required README.md`)
	})

	t.Run("reports nested package path when README is missing", func(t *testing.T) {
		downloadPackageFileFromGitHubForHost = func(owner, repo, path, ref, host string) ([]byte, error) {
			if path == "packages/repo-assist/aw.yml" {
				return []byte(`name: Repo Assist
files:
  - workflows/review.md
`), nil
			}
			return nil, createRepositoryPackageNotFoundError(path)
		}
		listPackageWorkflowFilesForHost = func(owner, repo, ref, workflowPath, host string) ([]string, error) {
			t.Fatalf("unexpected scan of %s", workflowPath)
			return nil, nil
		}

		_, err := resolveRepositoryPackage(&RepoSpec{RepoSlug: "owner/repo", PackagePath: "packages/repo-assist"}, "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), `owner/repo/packages/repo-assist`)
		assert.Contains(t, err.Error(), `packages/repo-assist/README.md`)
	})

	t.Run("rejects unknown manifest fields", func(t *testing.T) {
		downloadPackageFileFromGitHubForHost = func(owner, repo, path, ref, host string) ([]byte, error) {
			if path == "aw.yml" {
				return []byte(`name: Repo Assist
unknown-field: true
`), nil
			}
			return nil, createRepositoryPackageNotFoundError(path)
		}

		_, err := resolveRepositoryPackage(&RepoSpec{RepoSlug: "owner/repo"}, "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), `unknown-field`)
	})

	t.Run("resolves nested package manifests", func(t *testing.T) {
		downloadPackageFileFromGitHubForHost = func(owner, repo, path, ref, host string) ([]byte, error) {
			switch path {
			case "packages/repo-assist/aw.yml":
				return []byte(`name: Repo Assist
files:
  - workflows/review.md
`), nil
			case "packages/repo-assist/README.md":
				return []byte("# Repo Assist\n"), nil
			default:
				return nil, createRepositoryPackageNotFoundError(path)
			}
		}
		listPackageWorkflowFilesForHost = func(owner, repo, ref, workflowPath, host string) ([]string, error) {
			t.Fatalf("unexpected scan of %s", workflowPath)
			return nil, nil
		}

		pkg, err := resolveRepositoryPackage(&RepoSpec{RepoSlug: "owner/repo", PackagePath: "packages/repo-assist"}, "")
		require.NoError(t, err)
		assert.Equal(t, "packages/repo-assist/aw.yml", pkg.ManifestPath)
		assert.Equal(t, "packages/repo-assist/README.md", pkg.DocsPath)
		assert.Equal(t, []string{"packages/repo-assist/workflows/review.md"}, pkg.InstallationSource)
	})
}

func TestResolveWorkflows_RepositoryPackage(t *testing.T) {
	originalFetchFn := fetchWorkflowFromSourceWithContextFn
	originalDownload := downloadPackageFileFromGitHubForHost
	originalList := listPackageWorkflowFilesForHost
	originalDirFiles := listPackageDirFilesForHost
	originalDirSubdirs := listPackageDirSubdirsForHost
	originalDefaultBranch := getRepositoryPackageDefaultBranch
	t.Cleanup(func() {
		fetchWorkflowFromSourceWithContextFn = originalFetchFn
		downloadPackageFileFromGitHubForHost = originalDownload
		listPackageWorkflowFilesForHost = originalList
		listPackageDirFilesForHost = originalDirFiles
		listPackageDirSubdirsForHost = originalDirSubdirs
		getRepositoryPackageDefaultBranch = originalDefaultBranch
	})
	getRepositoryPackageDefaultBranch = func(repoSlug, host string) (string, error) {
		return "main", nil
	}
	listPackageDirFilesForHost = func(owner, repo, ref, dirPath, host string) ([]string, error) {
		return nil, createRepositoryPackageNotFoundError(dirPath)
	}
	listPackageDirSubdirsForHost = func(owner, repo, ref, dirPath, host string) ([]string, error) {
		return nil, createRepositoryPackageNotFoundError(dirPath)
	}

	downloadPackageFileFromGitHubForHost = func(owner, repo, path, ref, host string) ([]byte, error) {
		switch path {
		case "aw.yml":
			return []byte(`name: Repo Assist
files:
  - workflows/review.md
  - .github/workflows/nightly-review.md
`), nil
		case "README.md":
			return []byte("# Repo Assist\n"), nil
		}
		return nil, createRepositoryPackageNotFoundError(path)
	}
	listPackageWorkflowFilesForHost = func(owner, repo, ref, workflowPath, host string) ([]string, error) {
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

func TestResolveWorkflows_RepositoryPackageRejectsPrivateTrue(t *testing.T) {
	originalFetchFn := fetchWorkflowFromSourceWithContextFn
	originalDownload := downloadPackageFileFromGitHubForHost
	originalList := listPackageWorkflowFilesForHost
	originalDirFiles := listPackageDirFilesForHost
	originalDirSubdirs := listPackageDirSubdirsForHost
	originalDefaultBranch := getRepositoryPackageDefaultBranch
	t.Cleanup(func() {
		fetchWorkflowFromSourceWithContextFn = originalFetchFn
		downloadPackageFileFromGitHubForHost = originalDownload
		listPackageWorkflowFilesForHost = originalList
		listPackageDirFilesForHost = originalDirFiles
		listPackageDirSubdirsForHost = originalDirSubdirs
		getRepositoryPackageDefaultBranch = originalDefaultBranch
	})
	getRepositoryPackageDefaultBranch = func(repoSlug, host string) (string, error) {
		return "main", nil
	}
	listPackageDirFilesForHost = func(owner, repo, ref, dirPath, host string) ([]string, error) {
		return nil, createRepositoryPackageNotFoundError(dirPath)
	}
	listPackageDirSubdirsForHost = func(owner, repo, ref, dirPath, host string) ([]string, error) {
		return nil, createRepositoryPackageNotFoundError(dirPath)
	}
	downloadPackageFileFromGitHubForHost = func(owner, repo, path, ref, host string) ([]byte, error) {
		switch path {
		case "aw.yml":
			return []byte("name: Repo Assist\nfiles:\n  - workflows/review.md\n"), nil
		case "README.md":
			return []byte("# Repo Assist\n"), nil
		case "workflows/review.md":
			return []byte("---\nprivate: true\n---\n\n# Review\n"), nil
		default:
			return nil, createRepositoryPackageNotFoundError(path)
		}
	}
	listPackageWorkflowFilesForHost = func(owner, repo, ref, workflowPath, host string) ([]string, error) {
		t.Fatalf("unexpected scan of %s", workflowPath)
		return nil, nil
	}
	fetchWorkflowFromSourceWithContextFn = func(_ context.Context, spec *WorkflowSpec, _ bool) (*FetchedWorkflow, error) {
		return &FetchedWorkflow{
			Content:    []byte("---\nprivate: true\n---\n\n# Review\n"),
			CommitSHA:  "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			IsLocal:    false,
			SourcePath: spec.WorkflowPath,
		}, nil
	}

	_, err := ResolveWorkflows(context.Background(), []string{"owner/repo"}, false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), `workflow "workflows/review.md" sets private: true`)
}

func TestResolveWorkflows_NestedRepositoryPackage(t *testing.T) {
	originalFetchFn := fetchWorkflowFromSourceWithContextFn
	originalDownload := downloadPackageFileFromGitHubForHost
	originalList := listPackageWorkflowFilesForHost
	originalDirFiles := listPackageDirFilesForHost
	originalDirSubdirs := listPackageDirSubdirsForHost
	originalDefaultBranch := getRepositoryPackageDefaultBranch
	t.Cleanup(func() {
		fetchWorkflowFromSourceWithContextFn = originalFetchFn
		downloadPackageFileFromGitHubForHost = originalDownload
		listPackageWorkflowFilesForHost = originalList
		listPackageDirFilesForHost = originalDirFiles
		listPackageDirSubdirsForHost = originalDirSubdirs
		getRepositoryPackageDefaultBranch = originalDefaultBranch
	})
	getRepositoryPackageDefaultBranch = func(repoSlug, host string) (string, error) {
		return "main", nil
	}
	listPackageDirFilesForHost = func(owner, repo, ref, dirPath, host string) ([]string, error) {
		return nil, createRepositoryPackageNotFoundError(dirPath)
	}
	listPackageDirSubdirsForHost = func(owner, repo, ref, dirPath, host string) ([]string, error) {
		return nil, createRepositoryPackageNotFoundError(dirPath)
	}

	downloadPackageFileFromGitHubForHost = func(owner, repo, path, ref, host string) ([]byte, error) {
		switch path {
		case "folder/aw.yml":
			return []byte(`name: Repo Assist
files:
  - workflows/review.md
`), nil
		case "folder/README.md":
			return []byte("# Repo Assist\n"), nil
		}
		return nil, createRepositoryPackageNotFoundError(path)
	}
	listPackageWorkflowFilesForHost = func(owner, repo, ref, workflowPath, host string) ([]string, error) {
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

	resolved, err := ResolveWorkflows(context.Background(), []string{"owner/repo/folder"}, false)
	require.NoError(t, err)
	require.Len(t, resolved.Workflows, 1)
	assert.Equal(t, "folder/workflows/review.md", resolved.Workflows[0].Spec.WorkflowPath)
}

func TestResolveWorkflows_FallsBackToWorkflowWhenNestedManifestMissing(t *testing.T) {
	originalFetchFn := fetchWorkflowFromSourceWithContextFn
	originalDownload := downloadPackageFileFromGitHubForHost
	originalDefaultBranch := getRepositoryPackageDefaultBranch
	t.Cleanup(func() {
		fetchWorkflowFromSourceWithContextFn = originalFetchFn
		downloadPackageFileFromGitHubForHost = originalDownload
		getRepositoryPackageDefaultBranch = originalDefaultBranch
	})
	getRepositoryPackageDefaultBranch = func(repoSlug, host string) (string, error) {
		return "main", nil
	}

	downloadPackageFileFromGitHubForHost = func(owner, repo, path, ref, host string) ([]byte, error) {
		return nil, createRepositoryPackageNotFoundError(path)
	}
	fetchWorkflowFromSourceWithContextFn = func(_ context.Context, spec *WorkflowSpec, _ bool) (*FetchedWorkflow, error) {
		return &FetchedWorkflow{
			Content:    []byte("---\nname: Test\non: push\n---\n"),
			CommitSHA:  "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			IsLocal:    false,
			SourcePath: spec.WorkflowPath,
		}, nil
	}

	resolved, err := ResolveWorkflows(context.Background(), []string{"owner/repo/review"}, false)
	require.NoError(t, err)
	require.Len(t, resolved.Workflows, 1)
	assert.Equal(t, "workflows/review.md", resolved.Workflows[0].Spec.WorkflowPath)
}

func TestParseRepositoryPackageSpec(t *testing.T) {
	tests := []struct {
		name            string
		spec            string
		wantOK          bool
		wantErr         string
		wantRepoSlug    string
		wantPackagePath string
		wantVersion     string
	}{
		{
			name:         "repo only package",
			spec:         "owner/repo",
			wantOK:       true,
			wantRepoSlug: "owner/repo",
		},
		{
			name:         "repo only package with slash branch ref",
			spec:         "owner/repo@feature/github-agentic-workflow",
			wantOK:       true,
			wantRepoSlug: "owner/repo",
			wantVersion:  "feature/github-agentic-workflow",
		},
		{
			name:         "repo only package with sanitized branch characters",
			spec:         "owner/repo@release/2026.05.27-rc_1",
			wantOK:       true,
			wantRepoSlug: "owner/repo",
			wantVersion:  "release/2026.05.27-rc_1",
		},
		{
			name:            "nested package path",
			spec:            "owner/repo/packages/repo-assist",
			wantOK:          true,
			wantRepoSlug:    "owner/repo",
			wantPackagePath: "packages/repo-assist",
		},
		{
			name:            "nested package path with slash branch ref",
			spec:            "owner/repo/agentic-workflows@feature/github-agentic-workflow",
			wantOK:          true,
			wantRepoSlug:    "owner/repo",
			wantPackagePath: "agentic-workflows",
			wantVersion:     "feature/github-agentic-workflow",
		},
		{
			name:            "nested package path with sanitized branch characters",
			spec:            "owner/repo/agentic-workflows@hotfix/github-aw_fix-1.2.3",
			wantOK:          true,
			wantRepoSlug:    "owner/repo",
			wantPackagePath: "agentic-workflows",
			wantVersion:     "hotfix/github-aw_fix-1.2.3",
		},
		{
			name:   "workflow path is not package",
			spec:   "owner/repo/workflows/review.md",
			wantOK: false,
		},
		{
			name:   "workflow path with branch ref is not package",
			spec:   "owner/repo/agentic-workflows/pr-review.md@feature/github-agentic-workflows",
			wantOK: false,
		},
		{
			name:   "url is not package",
			spec:   "https://github.com/owner/repo",
			wantOK: false,
		},
		{
			name:    "rejects path traversal",
			spec:    "owner/repo/../secrets",
			wantOK:  true,
			wantErr: `invalid repository package path "../secrets"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repoSpec, ok, err := parseRepositoryPackageSpec(tt.spec)
			assert.Equal(t, tt.wantOK, ok)
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}
			require.NoError(t, err)
			if !tt.wantOK {
				assert.Nil(t, repoSpec)
				return
			}
			require.NotNil(t, repoSpec)
			assert.Equal(t, tt.wantRepoSlug, repoSpec.RepoSlug)
			assert.Equal(t, tt.wantPackagePath, repoSpec.PackagePath)
			assert.Equal(t, tt.wantVersion, repoSpec.Version)
		})
	}
}

func TestIsSupportedPackageInstallablePath(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		// .md files: allowed under workflows/, agentic-workflows/, and .github/workflows/
		{"workflows/review.md", true},
		{"agentic-workflows/review.md", true},
		{".github/workflows/nightly-review.md", true},
		// .yml action workflow files: allowed only under .github/workflows/ (direct children only)
		{".github/workflows/deploy.yml", true},
		{".github/workflows/ci.yml", true},
		// mixed-case extensions are accepted
		{".github/workflows/CI.YML", true},
		// .yml files under workflows/ and agentic-workflows/ are NOT supported
		{"workflows/deploy.yml", false},
		{"agentic-workflows/deploy.yml", false},
		// nested subdirectories under .github/workflows/ are NOT supported for .yml
		{".github/workflows/subdir/ci.yml", false},
		// .lock.yml files are NOT supported (generated artifacts)
		{".github/workflows/deploy.lock.yml", false},
		{"workflows/deploy.lock.yml", false},
		// unsupported extensions
		{".github/workflows/script.sh", false},
		{"README.md", false},
		{".github/workflows/config.yaml", false},
		// path traversal
		{"../evil.md", false},
		{"workflows/../README.md", false},
		{".github/workflows/../x.yml", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			assert.Equal(t, tt.want, isSupportedPackageInstallablePath(tt.path))
		})
	}
}

func TestExtractManifestIncludes(t *testing.T) {
	includes, warnings := extractManifestIncludes([]any{
		"workflows/review.md",
		"agentic-workflows/review.md",
		"skills/code-review",
		"agents/reviewer.md",
		".github/workflows/ci.yml",
	}, "aw.yml")
	assert.Equal(t, []string{
		"workflows/review.md",
		"agentic-workflows/review.md",
		"skills/code-review",
		"agents/reviewer.md",
		".github/workflows/ci.yml",
	}, includes)
	assert.Empty(t, warnings)
}

func TestCodemodManifestFilesToIncludes(t *testing.T) {
	converted := codemodManifestFilesToIncludes([]string{
		"workflows/review.md",
		".github/workflows/ci.yml",
	})
	assert.Equal(t, []string{
		"workflows/review.md",
		".github/workflows/ci.yml",
	}, converted)
}

func TestResolveRepositoryPackage_ActionWorkflowYML(t *testing.T) {
	originalVersion := GetVersion()
	originalDownload := downloadPackageFileFromGitHubForHost
	originalList := listPackageWorkflowFilesForHost
	originalDirFiles := listPackageDirFilesForHost
	originalDirSubdirs := listPackageDirSubdirsForHost
	originalDefaultBranch := getRepositoryPackageDefaultBranch
	t.Cleanup(func() {
		SetVersionInfo(originalVersion)
		downloadPackageFileFromGitHubForHost = originalDownload
		listPackageWorkflowFilesForHost = originalList
		listPackageDirFilesForHost = originalDirFiles
		listPackageDirSubdirsForHost = originalDirSubdirs
		getRepositoryPackageDefaultBranch = originalDefaultBranch
	})
	SetVersionInfo("v1.2.3")
	getRepositoryPackageDefaultBranch = func(repoSlug, host string) (string, error) {
		return "main", nil
	}
	listPackageDirFilesForHost = func(owner, repo, ref, dirPath, host string) ([]string, error) {
		return nil, createRepositoryPackageNotFoundError(dirPath)
	}
	listPackageDirSubdirsForHost = func(owner, repo, ref, dirPath, host string) ([]string, error) {
		return nil, createRepositoryPackageNotFoundError(dirPath)
	}

	t.Run("includes yml action workflow from files list", func(t *testing.T) {
		downloadPackageFileFromGitHubForHost = func(owner, repo, path, ref, host string) ([]byte, error) {
			switch path {
			case "aw.yml":
				return []byte(`name: CI Pack
files:
  - workflows/triage.md
  - .github/workflows/ci.yml
`), nil
			case "README.md":
				return []byte("# CI Pack\n"), nil
			default:
				return nil, createRepositoryPackageNotFoundError(path)
			}
		}
		listPackageWorkflowFilesForHost = func(owner, repo, ref, workflowPath, host string) ([]string, error) {
			t.Fatalf("unexpected scan of %s", workflowPath)
			return nil, nil
		}

		pkg, err := resolveRepositoryPackage(&RepoSpec{RepoSlug: "owner/repo"}, "")
		require.NoError(t, err)
		assert.Equal(t, []string{"workflows/triage.md", ".github/workflows/ci.yml"}, pkg.InstallationSource)
		assert.Contains(t, strings.Join(pkg.Warnings, "\n"), "Field 'files'")
	})

	t.Run("rejects yml files outside .github/workflows with warning", func(t *testing.T) {
		downloadPackageFileFromGitHubForHost = func(owner, repo, path, ref, host string) ([]byte, error) {
			switch path {
			case "aw.yml":
				return []byte(`name: CI Pack
files:
  - workflows/triage.md
  - workflows/deploy.yml
`), nil
			case "README.md":
				return []byte("# CI Pack\n"), nil
			default:
				return nil, createRepositoryPackageNotFoundError(path)
			}
		}
		listPackageWorkflowFilesForHost = func(owner, repo, ref, workflowPath, host string) ([]string, error) {
			t.Fatalf("unexpected scan of %s", workflowPath)
			return nil, nil
		}

		pkg, err := resolveRepositoryPackage(&RepoSpec{RepoSlug: "owner/repo"}, "")
		require.NoError(t, err)
		// Only the .md file should be accepted; the yml under workflows/ is rejected
		assert.Equal(t, []string{"workflows/triage.md"}, pkg.InstallationSource)
		require.NotEmpty(t, pkg.Warnings)
		assert.Contains(t, strings.Join(pkg.Warnings, "\n"), "Ignoring files entry")
	})

	t.Run("rejects duplicate markdown filenames across different folders", func(t *testing.T) {
		downloadPackageFileFromGitHubForHost = func(owner, repo, path, ref, host string) ([]byte, error) {
			switch path {
			case "aw.yml":
				return []byte(`name: Duplicate Filenames
files:
  - workflows/triage.md
  - workflows/subdir/triage.md
`), nil
			case "README.md":
				return []byte("# Duplicate Filenames\n"), nil
			default:
				return nil, createRepositoryPackageNotFoundError(path)
			}
		}
		listPackageWorkflowFilesForHost = func(owner, repo, ref, workflowPath, host string) ([]string, error) {
			t.Fatalf("unexpected scan of %s", workflowPath)
			return nil, nil
		}

		_, err := resolveRepositoryPackage(&RepoSpec{RepoSlug: "owner/repo"}, "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "duplicate workflow filename")
	})
}

func TestResolveWorkflows_ActionWorkflowYML(t *testing.T) {
	originalFetchFn := fetchWorkflowFromSourceWithContextFn
	originalDownload := downloadPackageFileFromGitHubForHost
	originalList := listPackageWorkflowFilesForHost
	originalDirFiles := listPackageDirFilesForHost
	originalDirSubdirs := listPackageDirSubdirsForHost
	originalDefaultBranch := getRepositoryPackageDefaultBranch
	t.Cleanup(func() {
		fetchWorkflowFromSourceWithContextFn = originalFetchFn
		downloadPackageFileFromGitHubForHost = originalDownload
		listPackageWorkflowFilesForHost = originalList
		listPackageDirFilesForHost = originalDirFiles
		listPackageDirSubdirsForHost = originalDirSubdirs
		getRepositoryPackageDefaultBranch = originalDefaultBranch
	})
	getRepositoryPackageDefaultBranch = func(repoSlug, host string) (string, error) {
		return "main", nil
	}
	listPackageDirFilesForHost = func(owner, repo, ref, dirPath, host string) ([]string, error) {
		return nil, createRepositoryPackageNotFoundError(dirPath)
	}
	listPackageDirSubdirsForHost = func(owner, repo, ref, dirPath, host string) ([]string, error) {
		return nil, createRepositoryPackageNotFoundError(dirPath)
	}

	rawYML := []byte("name: CI\non: [push]\njobs:\n  build:\n    runs-on: ubuntu-latest\n    steps: []\n")

	downloadPackageFileFromGitHubForHost = func(owner, repo, path, ref, host string) ([]byte, error) {
		switch path {
		case "aw.yml":
			return []byte(`name: CI Pack
files:
  - workflows/triage.md
  - .github/workflows/ci.yml
`), nil
		case "README.md":
			return []byte("# CI Pack\n"), nil
		}
		return nil, createRepositoryPackageNotFoundError(path)
	}
	listPackageWorkflowFilesForHost = func(owner, repo, ref, workflowPath, host string) ([]string, error) {
		t.Fatalf("unexpected scan of %s", workflowPath)
		return nil, nil
	}
	fetchWorkflowFromSourceWithContextFn = func(_ context.Context, spec *WorkflowSpec, _ bool) (*FetchedWorkflow, error) {
		switch spec.WorkflowPath {
		case "workflows/triage.md":
			return &FetchedWorkflow{
				Content:    []byte("---\nname: Triage\non: issues\n---\n"),
				CommitSHA:  "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
				IsLocal:    false,
				SourcePath: spec.WorkflowPath,
			}, nil
		case ".github/workflows/ci.yml":
			return &FetchedWorkflow{
				Content:    rawYML,
				CommitSHA:  "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
				IsLocal:    false,
				SourcePath: spec.WorkflowPath,
			}, nil
		}
		return nil, fmt.Errorf("unexpected path: %s", spec.WorkflowPath)
	}

	resolved, err := ResolveWorkflows(context.Background(), []string{"owner/repo"}, false)
	require.NoError(t, err)
	require.Len(t, resolved.Workflows, 2)

	// First workflow is the .md agentic workflow
	assert.Equal(t, "workflows/triage.md", resolved.Workflows[0].Spec.WorkflowPath)
	assert.Equal(t, "triage", resolved.Workflows[0].Spec.WorkflowName)
	assert.False(t, resolved.Workflows[0].IsActionWorkflow)

	// Second workflow is the .yml action workflow
	assert.Equal(t, ".github/workflows/ci.yml", resolved.Workflows[1].Spec.WorkflowPath)
	assert.Equal(t, "ci", resolved.Workflows[1].Spec.WorkflowName)
	assert.True(t, resolved.Workflows[1].IsActionWorkflow)
	assert.YAMLEq(t, string(rawYML), string(resolved.Workflows[1].Content))
}

func TestIsSupportedSkillDirPath(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		// Valid: direct children of skills/
		{"skills/my-skill", true},
		{"skills/code-review", true},
		{"skills/a", true},
		// Invalid: nested further
		{"skills/my-skill/subdir", false},
		// Invalid: wrong prefix
		{"agents/my-skill", false},
		{"workflows/my-skill", false},
		{".github/skills/my-skill", true},
		// Invalid: empty
		{"", false},
		// Invalid: path traversal
		{"skills/../evil", false},
		// Invalid: just the prefix
		{"skills/", false},
		{"skills", false},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			assert.Equal(t, tt.want, isSupportedSkillDirPath(tt.path))
		})
	}
}

func TestIsSupportedAgentFilePath(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		// Valid: .md files directly under agents/
		{"agents/my-agent.md", true},
		{"agents/code-review.md", true},
		{"agents/a.md", true},
		// Invalid: non-.md extension
		{"agents/my-agent.sh", false},
		{"agents/my-agent.yml", false},
		{"agents/my-agent", false},
		// Invalid: nested subdirectory
		{"agents/subdir/my-agent.md", false},
		// Invalid: wrong prefix
		{"skills/my-agent.md", false},
		{"workflows/my-agent.md", false},
		{".github/agents/my-agent.md", true},
		// Invalid: empty
		{"", false},
		// Invalid: path traversal
		{"agents/../evil.md", false},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			assert.Equal(t, tt.want, isSupportedAgentFilePath(tt.path))
		})
	}
}

func TestExtractManifestSkillDirs(t *testing.T) {
	t.Run("valid entries are accepted", func(t *testing.T) {
		dirs, warnings := extractManifestSkillDirs([]any{"skills/review", "skills/triage"}, "aw.yml")
		assert.Equal(t, []string{"skills/review", "skills/triage"}, dirs)
		assert.Empty(t, warnings)
	})

	t.Run("invalid entries produce warnings", func(t *testing.T) {
		dirs, warnings := extractManifestSkillDirs([]any{"skills/valid", "not-skills/bad", ".github/skills/bad"}, "aw.yml")
		assert.Equal(t, []string{"skills/valid", ".github/skills/bad"}, dirs)
		require.Len(t, warnings, 1)
		assert.Contains(t, warnings[0], "not-skills/bad")
	})

	t.Run("duplicate entries are deduplicated", func(t *testing.T) {
		dirs, warnings := extractManifestSkillDirs([]any{"skills/review", "skills/review"}, "aw.yml")
		assert.Equal(t, []string{"skills/review"}, dirs)
		assert.Empty(t, warnings)
	})

	t.Run("non-list value produces warning", func(t *testing.T) {
		dirs, warnings := extractManifestSkillDirs("not-a-list", "aw.yml")
		assert.Empty(t, dirs)
		require.Len(t, warnings, 1)
		assert.Contains(t, warnings[0], "not a list of strings")
	})
}

func TestExtractManifestAgentFiles(t *testing.T) {
	t.Run("valid entries are accepted", func(t *testing.T) {
		files, warnings := extractManifestAgentFiles([]any{"agents/review.md", "agents/triage.md"}, "aw.yml")
		assert.Equal(t, []string{"agents/review.md", "agents/triage.md"}, files)
		assert.Empty(t, warnings)
	})

	t.Run("invalid entries produce warnings", func(t *testing.T) {
		files, warnings := extractManifestAgentFiles([]any{"agents/valid.md", "agents/bad.sh", "skills/bad.md"}, "aw.yml")
		assert.Equal(t, []string{"agents/valid.md"}, files)
		require.Len(t, warnings, 2)
		assert.Contains(t, warnings[0], "agents/bad.sh")
		assert.Contains(t, warnings[1], "skills/bad.md")
	})

	t.Run("duplicate entries are deduplicated", func(t *testing.T) {
		files, warnings := extractManifestAgentFiles([]any{"agents/review.md", "agents/review.md"}, "aw.yml")
		assert.Equal(t, []string{"agents/review.md"}, files)
		assert.Empty(t, warnings)
	})

	t.Run("non-list value produces warning", func(t *testing.T) {
		files, warnings := extractManifestAgentFiles("not-a-list", "aw.yml")
		assert.Empty(t, files)
		require.Len(t, warnings, 1)
		assert.Contains(t, warnings[0], "not a list of strings")
	})
}

func TestResolveRepositoryPackage_SkillsAndAgents(t *testing.T) {
	originalVersion := GetVersion()
	originalDownload := downloadPackageFileFromGitHubForHost
	originalList := listPackageWorkflowFilesForHost
	originalDirFiles := listPackageDirFilesForHost
	originalDirFilesRecursively := listPackageDirFilesRecursivelyForHost
	originalDirSubdirs := listPackageDirSubdirsForHost
	originalDefaultBranch := getRepositoryPackageDefaultBranch
	t.Cleanup(func() {
		SetVersionInfo(originalVersion)
		downloadPackageFileFromGitHubForHost = originalDownload
		listPackageWorkflowFilesForHost = originalList
		listPackageDirFilesForHost = originalDirFiles
		listPackageDirFilesRecursivelyForHost = originalDirFilesRecursively
		listPackageDirSubdirsForHost = originalDirSubdirs
		getRepositoryPackageDefaultBranch = originalDefaultBranch
	})
	SetVersionInfo("v1.2.3")
	getRepositoryPackageDefaultBranch = func(repoSlug, host string) (string, error) {
		return "main", nil
	}

	t.Run("explicit skills and agents from manifest", func(t *testing.T) {
		downloadPackageFileFromGitHubForHost = func(owner, repo, filePath, ref, host string) ([]byte, error) {
			switch filePath {
			case "aw.yml":
				return []byte(`name: My Package
skills:
  - skills/code-review
agents:
  - agents/triage.md
files:
  - workflows/review.md
`), nil
			case "README.md":
				return []byte("# My Package\n"), nil
			case "skills/code-review/SKILL.md":
				return []byte("# skill\n"), nil
			default:
				return nil, createRepositoryPackageNotFoundError(filePath)
			}
		}
		listPackageWorkflowFilesForHost = func(owner, repo, ref, workflowPath, host string) ([]string, error) {
			t.Fatalf("unexpected workflow scan of %s", workflowPath)
			return nil, nil
		}
		listPackageDirFilesRecursivelyForHost = func(owner, repo, ref, dirPath, host string) ([]string, error) {
			if dirPath == "skills/code-review" {
				return []string{"skills/code-review/SKILL.md", "skills/code-review/prompt.sh"}, nil
			}
			return nil, createRepositoryPackageNotFoundError(dirPath)
		}
		// Auto-scan runs after manifest skills; return empty so no extras are added.
		listPackageDirSubdirsForHost = func(owner, repo, ref, dirPath, host string) ([]string, error) {
			return nil, createRepositoryPackageNotFoundError(dirPath)
		}

		pkg, err := resolveRepositoryPackage(&RepoSpec{RepoSlug: "owner/repo"}, "")
		require.NoError(t, err)
		assert.Equal(t, []string{"workflows/review.md"}, pkg.InstallationSource)
		require.Len(t, pkg.SkillFiles, 2)
		assert.Equal(t, "skills/code-review/SKILL.md", pkg.SkillFiles[0].SourcePath)
		assert.Equal(t, "code-review", pkg.SkillFiles[0].SkillName)
		assert.Equal(t, "skills/code-review/prompt.sh", pkg.SkillFiles[1].SourcePath)
		assert.Equal(t, "code-review", pkg.SkillFiles[1].SkillName)
		assert.Equal(t, []string{"agents/triage.md"}, pkg.AgentFiles)
		assert.Contains(t, strings.Join(pkg.Warnings, "\n"), "Field 'files'")
	})

	t.Run("includes field infers workflow skill and agent types", func(t *testing.T) {
		downloadPackageFileFromGitHubForHost = func(owner, repo, filePath, ref, host string) ([]byte, error) {
			switch filePath {
			case "aw.yml":
				return []byte(`name: My Package
includes:
  - workflows/review.md
  - skills/code-review
  - .github/agents/triage.md
`), nil
			case "README.md":
				return []byte("# My Package\n"), nil
			case "skills/code-review/SKILL.md":
				return []byte("# skill\n"), nil
			default:
				return nil, createRepositoryPackageNotFoundError(filePath)
			}
		}
		listPackageWorkflowFilesForHost = func(owner, repo, ref, workflowPath, host string) ([]string, error) {
			t.Fatalf("unexpected workflow scan of %s", workflowPath)
			return nil, nil
		}
		listPackageDirFilesRecursivelyForHost = func(owner, repo, ref, dirPath, host string) ([]string, error) {
			if dirPath == "skills/code-review" {
				return []string{"skills/code-review/SKILL.md", "skills/code-review/prompt.md"}, nil
			}
			return nil, createRepositoryPackageNotFoundError(dirPath)
		}
		// Auto-scan runs after manifest skills; return empty so no extras are added.
		listPackageDirSubdirsForHost = func(owner, repo, ref, dirPath, host string) ([]string, error) {
			return nil, createRepositoryPackageNotFoundError(dirPath)
		}

		pkg, err := resolveRepositoryPackage(&RepoSpec{RepoSlug: "owner/repo"}, "")
		require.NoError(t, err)
		assert.Equal(t, []string{"workflows/review.md"}, pkg.InstallationSource)
		require.Len(t, pkg.SkillFiles, 2)
		assert.Equal(t, []string{".github/agents/triage.md"}, pkg.AgentFiles)
	})

	t.Run("auto-scans skills and agents when absent from manifest", func(t *testing.T) {
		downloadPackageFileFromGitHubForHost = func(owner, repo, filePath, ref, host string) ([]byte, error) {
			switch filePath {
			case "aw.yml":
				return []byte(`name: My Package
files:
  - workflows/review.md
`), nil
			case "README.md":
				return []byte("# My Package\n"), nil
			// SKILL.md marker check
			case "skills/auto-skill/SKILL.md":
				return []byte("# Auto Skill\n"), nil
			default:
				return nil, createRepositoryPackageNotFoundError(filePath)
			}
		}
		listPackageWorkflowFilesForHost = func(owner, repo, ref, workflowPath, host string) ([]string, error) {
			t.Fatalf("unexpected workflow scan of %s", workflowPath)
			return nil, nil
		}
		// Agent files are listed with the non-recursive helper.
		listPackageDirFilesForHost = func(owner, repo, ref, dirPath, host string) ([]string, error) {
			if dirPath == "agents" {
				return []string{"agents/my-agent.md", "agents/helper.sh"}, nil
			}
			return nil, createRepositoryPackageNotFoundError(dirPath)
		}
		// Skill files are listed with the recursive helper.
		listPackageDirFilesRecursivelyForHost = func(owner, repo, ref, dirPath, host string) ([]string, error) {
			if dirPath == "skills/auto-skill" {
				return []string{"skills/auto-skill/SKILL.md"}, nil
			}
			return nil, createRepositoryPackageNotFoundError(dirPath)
		}
		listPackageDirSubdirsForHost = func(owner, repo, ref, dirPath, host string) ([]string, error) {
			if dirPath == "skills" {
				return []string{"skills/auto-skill"}, nil
			}
			return nil, createRepositoryPackageNotFoundError(dirPath)
		}

		pkg, err := resolveRepositoryPackage(&RepoSpec{RepoSlug: "owner/repo"}, "")
		require.NoError(t, err)
		require.Len(t, pkg.SkillFiles, 1)
		assert.Equal(t, "skills/auto-skill/SKILL.md", pkg.SkillFiles[0].SourcePath)
		assert.Equal(t, "auto-skill", pkg.SkillFiles[0].SkillName)
		// Only .md files from agents/ are included; helper.sh is filtered out
		assert.Equal(t, []string{"agents/my-agent.md"}, pkg.AgentFiles)
	})

	t.Run("missing skill directory produces warning not error", func(t *testing.T) {
		downloadPackageFileFromGitHubForHost = func(owner, repo, filePath, ref, host string) ([]byte, error) {
			switch filePath {
			case "aw.yml":
				return []byte(`name: My Package
skills:
  - skills/missing-skill
files:
  - workflows/review.md
`), nil
			case "README.md":
				return []byte("# My Package\n"), nil
			default:
				return nil, createRepositoryPackageNotFoundError(filePath)
			}
		}
		listPackageWorkflowFilesForHost = func(owner, repo, ref, workflowPath, host string) ([]string, error) {
			t.Fatalf("unexpected workflow scan of %s", workflowPath)
			return nil, nil
		}
		listPackageDirFilesForHost = func(owner, repo, ref, dirPath, host string) ([]string, error) {
			return nil, createRepositoryPackageNotFoundError(dirPath)
		}
		listPackageDirFilesRecursivelyForHost = func(owner, repo, ref, dirPath, host string) ([]string, error) {
			return nil, createRepositoryPackageNotFoundError(dirPath)
		}
		// Auto-scan runs after manifest skills; return empty so no extras are added.
		listPackageDirSubdirsForHost = func(owner, repo, ref, dirPath, host string) ([]string, error) {
			return nil, createRepositoryPackageNotFoundError(dirPath)
		}

		pkg, err := resolveRepositoryPackage(&RepoSpec{RepoSlug: "owner/repo"}, "")
		require.NoError(t, err)
		assert.Empty(t, pkg.SkillFiles)
		require.NotEmpty(t, pkg.Warnings)
		assert.Contains(t, strings.Join(pkg.Warnings, "\n"), "skills/missing-skill")
	})

	t.Run("explicit skill without marker produces warning", func(t *testing.T) {
		downloadPackageFileFromGitHubForHost = func(owner, repo, filePath, ref, host string) ([]byte, error) {
			switch filePath {
			case "aw.yml":
				return []byte(`name: My Package
skills:
  - skills/no-marker
files:
  - workflows/review.md
`), nil
			case "README.md":
				return []byte("# My Package\n"), nil
			default:
				return nil, createRepositoryPackageNotFoundError(filePath)
			}
		}
		listPackageWorkflowFilesForHost = func(owner, repo, ref, workflowPath, host string) ([]string, error) {
			t.Fatalf("unexpected workflow scan of %s", workflowPath)
			return nil, nil
		}
		listPackageDirFilesForHost = func(owner, repo, ref, dirPath, host string) ([]string, error) {
			return nil, createRepositoryPackageNotFoundError(dirPath)
		}
		listPackageDirFilesRecursivelyForHost = func(owner, repo, ref, dirPath, host string) ([]string, error) {
			if dirPath == "skills/no-marker" {
				return []string{"skills/no-marker/prompt.md"}, nil
			}
			return nil, createRepositoryPackageNotFoundError(dirPath)
		}
		listPackageDirSubdirsForHost = func(owner, repo, ref, dirPath, host string) ([]string, error) {
			return nil, createRepositoryPackageNotFoundError(dirPath)
		}

		pkg, err := resolveRepositoryPackage(&RepoSpec{RepoSlug: "owner/repo"}, "")
		require.NoError(t, err)
		assert.Empty(t, pkg.SkillFiles)
		assert.Contains(t, strings.Join(pkg.Warnings, "\n"), "missing required SKILL.md")
	})

	t.Run("no skills or agents when directories absent", func(t *testing.T) {
		downloadPackageFileFromGitHubForHost = func(owner, repo, filePath, ref, host string) ([]byte, error) {
			switch filePath {
			case "aw.yml":
				return []byte(`name: My Package
files:
  - workflows/review.md
`), nil
			case "README.md":
				return []byte("# My Package\n"), nil
			default:
				return nil, createRepositoryPackageNotFoundError(filePath)
			}
		}
		listPackageWorkflowFilesForHost = func(owner, repo, ref, workflowPath, host string) ([]string, error) {
			// Auto-scan of agents/ returns not-found
			return nil, createRepositoryPackageNotFoundError(workflowPath)
		}
		listPackageDirFilesForHost = func(owner, repo, ref, dirPath, host string) ([]string, error) {
			return nil, createRepositoryPackageNotFoundError(dirPath)
		}
		listPackageDirFilesRecursivelyForHost = func(owner, repo, ref, dirPath, host string) ([]string, error) {
			return nil, createRepositoryPackageNotFoundError(dirPath)
		}
		listPackageDirSubdirsForHost = func(owner, repo, ref, dirPath, host string) ([]string, error) {
			return nil, createRepositoryPackageNotFoundError(dirPath)
		}

		pkg, err := resolveRepositoryPackage(&RepoSpec{RepoSlug: "owner/repo"}, "")
		require.NoError(t, err)
		assert.Empty(t, pkg.SkillFiles)
		assert.Empty(t, pkg.AgentFiles)
		assert.Contains(t, strings.Join(pkg.Warnings, "\n"), "Field 'files'")
	})

	t.Run("manifest skills copied first then auto-scanned additional skills appended", func(t *testing.T) {
		downloadPackageFileFromGitHubForHost = func(owner, repo, filePath, ref, host string) ([]byte, error) {
			switch filePath {
			case "aw.yml":
				return []byte(`name: My Package
skills:
  - skills/review
files:
  - workflows/review.md
`), nil
			case "README.md":
				return []byte("# My Package\n"), nil
			case "skills/review/SKILL.md":
				return []byte("# Review Skill\n"), nil
			// SKILL.md marker check for auto-scanned skill
			case "skills/triage/SKILL.md":
				return []byte("# Triage Skill\n"), nil
			default:
				return nil, createRepositoryPackageNotFoundError(filePath)
			}
		}
		listPackageWorkflowFilesForHost = func(owner, repo, ref, workflowPath, host string) ([]string, error) {
			t.Fatalf("unexpected workflow scan of %s", workflowPath)
			return nil, nil
		}
		listPackageDirFilesForHost = func(owner, repo, ref, dirPath, host string) ([]string, error) {
			return nil, createRepositoryPackageNotFoundError(dirPath)
		}
		listPackageDirFilesRecursivelyForHost = func(owner, repo, ref, dirPath, host string) ([]string, error) {
			switch dirPath {
			case "skills/review":
				return []string{"skills/review/SKILL.md"}, nil
			case "skills/triage":
				return []string{"skills/triage/SKILL.md", "skills/triage/prompts/detailed.md"}, nil
			}
			return nil, createRepositoryPackageNotFoundError(dirPath)
		}
		// Auto-scan finds "triage" in addition to the manifest-specified "review".
		listPackageDirSubdirsForHost = func(owner, repo, ref, dirPath, host string) ([]string, error) {
			if dirPath == "skills" {
				return []string{"skills/review", "skills/triage"}, nil
			}
			return nil, createRepositoryPackageNotFoundError(dirPath)
		}

		pkg, err := resolveRepositoryPackage(&RepoSpec{RepoSlug: "owner/repo"}, "")
		require.NoError(t, err)
		// Manifest skill "review" appears first; auto-scanned "triage" appended after.
		require.Len(t, pkg.SkillFiles, 3)
		assert.Equal(t, "review", pkg.SkillFiles[0].SkillName)
		assert.Equal(t, "triage", pkg.SkillFiles[1].SkillName)
		assert.Equal(t, "triage", pkg.SkillFiles[2].SkillName)
		assert.Equal(t, "skills/triage/prompts/detailed.md", pkg.SkillFiles[2].SourcePath)
		assert.Contains(t, strings.Join(pkg.Warnings, "\n"), "Field 'files'")
	})

	t.Run("auto-scan errors are warnings when manifest skills are explicit", func(t *testing.T) {
		downloadPackageFileFromGitHubForHost = func(owner, repo, filePath, ref, host string) ([]byte, error) {
			switch filePath {
			case "aw.yml":
				return []byte(`name: My Package
skills:
  - skills/review
files:
  - workflows/review.md
`), nil
			case "README.md":
				return []byte("# My Package\n"), nil
			case "skills/review/SKILL.md":
				return []byte("# Review Skill\n"), nil
			default:
				return nil, createRepositoryPackageNotFoundError(filePath)
			}
		}
		listPackageWorkflowFilesForHost = func(owner, repo, ref, workflowPath, host string) ([]string, error) {
			t.Fatalf("unexpected workflow scan of %s", workflowPath)
			return nil, nil
		}
		listPackageDirFilesForHost = func(owner, repo, ref, dirPath, host string) ([]string, error) {
			return nil, createRepositoryPackageNotFoundError(dirPath)
		}
		listPackageDirFilesRecursivelyForHost = func(owner, repo, ref, dirPath, host string) ([]string, error) {
			if dirPath == "skills/review" {
				return []string{"skills/review/SKILL.md", "skills/review/prompts/default.md"}, nil
			}
			return nil, createRepositoryPackageNotFoundError(dirPath)
		}
		listPackageDirSubdirsForHost = func(owner, repo, ref, dirPath, host string) ([]string, error) {
			return nil, errors.New("rate limit")
		}

		pkg, err := resolveRepositoryPackage(&RepoSpec{RepoSlug: "owner/repo"}, "")
		require.NoError(t, err)
		require.Len(t, pkg.SkillFiles, 2)
		assert.Equal(t, "skills/review/SKILL.md", pkg.SkillFiles[0].SourcePath)
		assert.Contains(t, strings.Join(pkg.Warnings, "\n"), "failed to auto-scan skills directory")
	})

	t.Run("skill folder nested files are included recursively", func(t *testing.T) {
		downloadPackageFileFromGitHubForHost = func(owner, repo, filePath, ref, host string) ([]byte, error) {
			switch filePath {
			case "aw.yml":
				return []byte(`name: My Package
skills:
  - skills/my-skill
includes:
  - workflows/review.md
`), nil
			case "README.md":
				return []byte("# My Package\n"), nil
			case "skills/my-skill/SKILL.md":
				return []byte("# My Skill\n"), nil
			default:
				return nil, createRepositoryPackageNotFoundError(filePath)
			}
		}
		listPackageWorkflowFilesForHost = func(owner, repo, ref, workflowPath, host string) ([]string, error) {
			t.Fatalf("unexpected workflow scan of %s", workflowPath)
			return nil, nil
		}
		listPackageDirFilesForHost = func(owner, repo, ref, dirPath, host string) ([]string, error) {
			return nil, createRepositoryPackageNotFoundError(dirPath)
		}
		listPackageDirFilesRecursivelyForHost = func(owner, repo, ref, dirPath, host string) ([]string, error) {
			if dirPath == "skills/my-skill" {
				// Simulate a skill with nested files in subdirectories.
				return []string{
					"skills/my-skill/SKILL.md",
					"skills/my-skill/ssl.json",
					"skills/my-skill/scripts/query.sh",
					"skills/my-skill/scripts/helpers/util.sh",
				}, nil
			}
			return nil, createRepositoryPackageNotFoundError(dirPath)
		}
		listPackageDirSubdirsForHost = func(owner, repo, ref, dirPath, host string) ([]string, error) {
			return nil, createRepositoryPackageNotFoundError(dirPath)
		}

		pkg, err := resolveRepositoryPackage(&RepoSpec{RepoSlug: "owner/repo"}, "")
		require.NoError(t, err)
		require.Len(t, pkg.SkillFiles, 4)
		assert.Equal(t, "skills/my-skill/SKILL.md", pkg.SkillFiles[0].SourcePath)
		assert.Equal(t, "skills/my-skill/ssl.json", pkg.SkillFiles[1].SourcePath)
		assert.Equal(t, "skills/my-skill/scripts/query.sh", pkg.SkillFiles[2].SourcePath)
		assert.Equal(t, "skills/my-skill/scripts/helpers/util.sh", pkg.SkillFiles[3].SourcePath)
		for _, sf := range pkg.SkillFiles {
			assert.Equal(t, "my-skill", sf.SkillName)
		}
	})
}

func TestResolveWorkflows_SkillsAndAgents(t *testing.T) {
	originalFetchFn := fetchWorkflowFromSourceWithContextFn
	originalDownload := downloadPackageFileFromGitHubForHost
	originalList := listPackageWorkflowFilesForHost
	originalDirFiles := listPackageDirFilesForHost
	originalDirFilesRecursively := listPackageDirFilesRecursivelyForHost
	originalDirSubdirs := listPackageDirSubdirsForHost
	originalDefaultBranch := getRepositoryPackageDefaultBranch
	t.Cleanup(func() {
		fetchWorkflowFromSourceWithContextFn = originalFetchFn
		downloadPackageFileFromGitHubForHost = originalDownload
		listPackageWorkflowFilesForHost = originalList
		listPackageDirFilesForHost = originalDirFiles
		listPackageDirFilesRecursivelyForHost = originalDirFilesRecursively
		listPackageDirSubdirsForHost = originalDirSubdirs
		getRepositoryPackageDefaultBranch = originalDefaultBranch
	})
	getRepositoryPackageDefaultBranch = func(repoSlug, host string) (string, error) {
		return "main", nil
	}

	skillMD := []byte("# My Skill\nThis is a skill.\n")
	agentMD := []byte("# My Agent\nThis is an agent.\n")
	workflowMD := []byte("---\nname: Review\non: issues\n---\n")

	downloadPackageFileFromGitHubForHost = func(owner, repo, filePath, ref, host string) ([]byte, error) {
		switch filePath {
		case "aw.yml":
			return []byte(`name: Full Package
skills:
  - skills/my-skill
agents:
  - agents/my-agent.md
files:
  - workflows/review.md
`), nil
		case "README.md":
			return []byte("# Full Package\n"), nil
		case "skills/my-skill/SKILL.md":
			return []byte("# My Skill\nThis is a skill.\n"), nil
		default:
			return nil, createRepositoryPackageNotFoundError(filePath)
		}
	}
	listPackageWorkflowFilesForHost = func(owner, repo, ref, workflowPath, host string) ([]string, error) {
		t.Fatalf("unexpected workflow scan of %s", workflowPath)
		return nil, nil
	}
	listPackageDirFilesRecursivelyForHost = func(owner, repo, ref, dirPath, host string) ([]string, error) {
		if dirPath == "skills/my-skill" {
			return []string{"skills/my-skill/SKILL.md"}, nil
		}
		return nil, createRepositoryPackageNotFoundError(dirPath)
	}
	// Auto-scan runs after manifest skills; return empty so no extras are added.
	listPackageDirSubdirsForHost = func(owner, repo, ref, dirPath, host string) ([]string, error) {
		return nil, createRepositoryPackageNotFoundError(dirPath)
	}
	fetchWorkflowFromSourceWithContextFn = func(_ context.Context, spec *WorkflowSpec, _ bool) (*FetchedWorkflow, error) {
		switch spec.WorkflowPath {
		case "workflows/review.md":
			return &FetchedWorkflow{Content: workflowMD, CommitSHA: "aaaa", IsLocal: false, SourcePath: spec.WorkflowPath}, nil
		case "skills/my-skill/SKILL.md":
			return &FetchedWorkflow{Content: skillMD, CommitSHA: "bbbb", IsLocal: false, SourcePath: spec.WorkflowPath}, nil
		case "agents/my-agent.md":
			return &FetchedWorkflow{Content: agentMD, CommitSHA: "cccc", IsLocal: false, SourcePath: spec.WorkflowPath}, nil
		}
		return nil, fmt.Errorf("unexpected path: %s", spec.WorkflowPath)
	}

	resolved, err := ResolveWorkflows(context.Background(), []string{"owner/repo"}, false)
	require.NoError(t, err)
	require.Len(t, resolved.Workflows, 3)

	// Workflow
	wf := resolved.Workflows[0]
	assert.Equal(t, "workflows/review.md", wf.Spec.WorkflowPath)
	assert.False(t, wf.IsPackageSkillFile)
	assert.False(t, wf.IsPackageAgentFile)

	// Skill file
	skill := resolved.Workflows[1]
	assert.Equal(t, "skills/my-skill/SKILL.md", skill.Spec.WorkflowPath)
	assert.True(t, skill.IsPackageSkillFile)
	assert.False(t, skill.IsPackageAgentFile)
	assert.Equal(t, "my-skill", skill.SkillName)
	assert.Equal(t, skillMD, skill.Content)

	// Agent file
	agent := resolved.Workflows[2]
	assert.Equal(t, "agents/my-agent.md", agent.Spec.WorkflowPath)
	assert.False(t, agent.IsPackageSkillFile)
	assert.True(t, agent.IsPackageAgentFile)
	assert.Equal(t, agentMD, agent.Content)
}

func TestIsGhAwRepository(t *testing.T) {
	tests := []struct {
		name     string
		repoSlug string
		want     bool
	}{
		{name: "exact match", repoSlug: "github/gh-aw", want: true},
		{name: "case-insensitive with whitespace", repoSlug: "  GitHub/GH-AW  ", want: true},
		{name: "different repository", repoSlug: "github/other", want: false},
		{name: "fork-like suffix does not match", repoSlug: "github/gh-aw-fork", want: false},
		{name: "different owner does not match", repoSlug: "other/gh-aw", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isGhAwRepository(tt.repoSlug)
			assert.Equal(t, tt.want, got)
		})
	}
}
