//go:build !integration

package cli

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractManifestIncludesWithWildcard(t *testing.T) {
	t.Parallel()
	includes, warnings, err := extractManifestIncludes([]any{
		"workflows/*",
		".github/workflows/*",
		"workflows/**",
		"workflows/review*.md",
	}, "aw.yml")
	require.NoError(t, err)
	assert.Equal(t, []string{"workflows/*", ".github/workflows/*"}, manifestIncludeSources(includes))
	require.Len(t, warnings, 2)
	assert.Contains(t, warnings[0], "workflows/**")
	assert.Contains(t, warnings[1], "workflows/review*.md")
}

func TestExpandManifestWildcardMatches(t *testing.T) {
	t.Parallel()
	matches := expandManifestWildcardMatches("workflows", []string{
		"workflows/z.md",
		"workflows/nested/ignored.md",
		"workflows/README.txt",
		"workflows/a.md",
	}, isSupportedPackageInstallablePath)
	assert.Equal(t, []string{"workflows/a.md", "workflows/z.md"}, manifestIncludeSources(matches))
}

func TestExpandRepositoryPackageWildcardIncludes(t *testing.T) {
	originalFiles := listPackageDirFilesForHost
	originalSubdirs := listPackageDirSubdirsForHost
	t.Cleanup(func() {
		listPackageDirFilesForHost = originalFiles
		listPackageDirSubdirsForHost = originalSubdirs
	})

	listPackageDirFilesForHost = func(_ context.Context, owner, repo, ref, dirPath, host string) ([]string, error) {
		switch dirPath {
		case "bundle/workflows":
			return []string{
				"bundle/workflows/z.md",
				"bundle/workflows/nested/ignored.md",
				"bundle/workflows/README.txt",
				"bundle/workflows/a.md",
			}, nil
		case ".github/workflows":
			return []string{
				".github/workflows/ci.yml",
				".github/workflows/ci.lock.yml",
			}, nil
		case "bundle/agents":
			return []string{"bundle/agents/reviewer.md"}, nil
		case "bundle/skills":
			return []string{"bundle/skills/README.md"}, nil
		default:
			return nil, createRepositoryPackageNotFoundError(dirPath)
		}
	}
	listPackageDirSubdirsForHost = func(_ context.Context, owner, repo, ref, dirPath, host string) ([]string, error) {
		if dirPath == "bundle/skills" {
			return []string{"bundle/skills/review"}, nil
		}
		return nil, nil
	}

	expanded, err := expandRepositoryPackageWildcardIncludes(
		t.Context(),
		"owner",
		"repo",
		"bundle",
		"main",
		"",
		[]repositoryPackageInclude{
			{Source: "workflows/a.md"},
			{Source: "workflows/*"},
			{Source: ".github/workflows/*"},
			{Source: "agents/*"},
			{Source: "skills/*"},
		},
	)
	require.NoError(t, err)
	assert.Equal(t, []string{
		"workflows/a.md",
		"workflows/z.md",
		".github/workflows/ci.yml",
		"agents/reviewer.md",
		"skills/review",
	}, manifestIncludeSources(expanded))
}

func TestResolveLocalRepositoryPackageWildcardIncludes(t *testing.T) {
	packageDir := t.TempDir()
	writePackageTestFile(t, packageDir, "README.md", "# Wildcard package\n")
	writePackageTestFile(t, packageDir, "aw.yml", `name: Wildcard package
includes:
  - workflows/*
  - .github/workflows/*
`)
	writePackageTestFile(t, packageDir, "workflows/z.md", "# Z\n")
	writePackageTestFile(t, packageDir, "workflows/a.md", "# A\n")
	writePackageTestFile(t, packageDir, "workflows/README.txt", "ignored\n")
	writePackageTestFile(t, packageDir, "workflows/nested/ignored.md", "# Ignored\n")
	writePackageTestFile(t, packageDir, ".github/workflows/ci.yml", "name: CI\n")
	writePackageTestFile(t, packageDir, ".github/workflows/ci.lock.yml", "name: Ignored\n")
	outsideWorkflow := filepath.Join(t.TempDir(), "outside.md")
	require.NoError(t, os.WriteFile(outsideWorkflow, []byte("# Outside\n"), 0o644))
	if err := os.Symlink(outsideWorkflow, filepath.Join(packageDir, "workflows", "linked.md")); err != nil {
		t.Logf("symlinks unavailable; skipping symlink fixture: %v", err)
	}

	pkg, err := resolveLocalRepositoryPackage(packageDir)
	require.NoError(t, err)
	require.NotNil(t, pkg)
	assert.Equal(t, []string{
		filepath.Join(packageDir, "workflows", "a.md"),
		filepath.Join(packageDir, "workflows", "z.md"),
		filepath.Join(packageDir, ".github", "workflows", "ci.yml"),
	}, packageInstallableSourcePaths(pkg.InstallationSource))
}
