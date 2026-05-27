//go:build !integration

package cli

import (
	"context"
	"fmt"
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
	originalDefaultBranch := getRepositoryPackageDefaultBranch
	t.Cleanup(func() {
		SetVersionInfo(originalVersion)
		downloadPackageFileFromGitHubForHost = originalDownload
		listPackageWorkflowFilesForHost = originalList
		getRepositoryPackageDefaultBranch = originalDefaultBranch
	})
	SetVersionInfo("v1.2.3")
	getRepositoryPackageDefaultBranch = func(repoSlug, host string) (string, error) {
		return "main", nil
	}

	t.Run("uses aw manifest files and README docs", func(t *testing.T) {
		downloadPackageFileFromGitHubForHost = func(owner, repo, path, ref, host string) ([]byte, error) {
			switch path {
			case "aw.yml":
				return []byte(`name: Repo Assist
emoji: 🤖
description: Friendly repository automation
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
		assert.Equal(t, "README.md", pkg.DocsPath)
		assert.Equal(t, []string{"workflows/review.md", ".github/workflows/nightly-review.md"}, pkg.InstallationSource)
		require.NotEmpty(t, pkg.Warnings)
		assert.Contains(t, pkg.Warnings[0], "Ignoring files entry")
	})

	t.Run("uses repository default branch when version is omitted", func(t *testing.T) {
		previousDefaultBranch := getRepositoryPackageDefaultBranch
		t.Cleanup(func() {
			getRepositoryPackageDefaultBranch = previousDefaultBranch
		})
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
	originalDefaultBranch := getRepositoryPackageDefaultBranch
	t.Cleanup(func() {
		fetchWorkflowFromSourceWithContextFn = originalFetchFn
		downloadPackageFileFromGitHubForHost = originalDownload
		listPackageWorkflowFilesForHost = originalList
		getRepositoryPackageDefaultBranch = originalDefaultBranch
	})
	getRepositoryPackageDefaultBranch = func(repoSlug, host string) (string, error) {
		return "main", nil
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

func TestResolveWorkflows_NestedRepositoryPackage(t *testing.T) {
	originalFetchFn := fetchWorkflowFromSourceWithContextFn
	originalDownload := downloadPackageFileFromGitHubForHost
	originalList := listPackageWorkflowFilesForHost
	originalDefaultBranch := getRepositoryPackageDefaultBranch
	t.Cleanup(func() {
		fetchWorkflowFromSourceWithContextFn = originalFetchFn
		downloadPackageFileFromGitHubForHost = originalDownload
		listPackageWorkflowFilesForHost = originalList
		getRepositoryPackageDefaultBranch = originalDefaultBranch
	})
	getRepositoryPackageDefaultBranch = func(repoSlug, host string) (string, error) {
		return "main", nil
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
		// .md files: allowed under workflows/ and .github/workflows/
		{"workflows/review.md", true},
		{".github/workflows/nightly-review.md", true},
		// .yml action workflow files: allowed only under .github/workflows/ (direct children only)
		{".github/workflows/deploy.yml", true},
		{".github/workflows/ci.yml", true},
		// mixed-case extensions are accepted
		{".github/workflows/CI.YML", true},
		// .yml files under workflows/ are NOT supported
		{"workflows/deploy.yml", false},
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

func TestResolveRepositoryPackage_ActionWorkflowYML(t *testing.T) {
	originalVersion := GetVersion()
	originalDownload := downloadPackageFileFromGitHubForHost
	originalList := listPackageWorkflowFilesForHost
	originalDefaultBranch := getRepositoryPackageDefaultBranch
	t.Cleanup(func() {
		SetVersionInfo(originalVersion)
		downloadPackageFileFromGitHubForHost = originalDownload
		listPackageWorkflowFilesForHost = originalList
		getRepositoryPackageDefaultBranch = originalDefaultBranch
	})
	SetVersionInfo("v1.2.3")
	getRepositoryPackageDefaultBranch = func(repoSlug, host string) (string, error) {
		return "main", nil
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
		assert.Empty(t, pkg.Warnings)
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
		assert.Contains(t, pkg.Warnings[0], "Ignoring files entry")
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
	originalDefaultBranch := getRepositoryPackageDefaultBranch
	t.Cleanup(func() {
		fetchWorkflowFromSourceWithContextFn = originalFetchFn
		downloadPackageFileFromGitHubForHost = originalDownload
		listPackageWorkflowFilesForHost = originalList
		getRepositoryPackageDefaultBranch = originalDefaultBranch
	})
	getRepositoryPackageDefaultBranch = func(repoSlug, host string) (string, error) {
		return "main", nil
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
