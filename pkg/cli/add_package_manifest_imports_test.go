//go:build !integration

package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractManifestImports(t *testing.T) {
	manifest, _, err := parseRepositoryPackageManifest("aw.yml", []byte(`name: Bundle
imports:
  - activity/aw.yml
  - dashboard/../activity/aw.yml
`))
	require.NoError(t, err)
	assert.Equal(t, []string{"activity/aw.yml"}, manifest.Imports)

	_, _, err = parseRepositoryPackageManifest("aw.yml", []byte("name: Bundle\nimports:\n  - /tmp/aw.yml\n"))
	require.ErrorContains(t, err, "absolute paths are not allowed")
}

func TestResolveRepositoryPackageManifestGraph(t *testing.T) {
	manifests := map[string]string{
		"aw.yml":            "name: Root\nimports:\n  - packages/a/aw.yml\n",
		"packages/a/aw.yml": "name: A\nimports:\n  - ../b/aw.yml\n",
		"packages/b/aw.yml": "name: B\nincludes:\n  - workflows/b.md\n",
	}
	root, _, err := parseRepositoryPackageManifest("aw.yml", []byte(manifests["aw.yml"]))
	require.NoError(t, err)

	nodes, _, err := resolveRepositoryPackageManifestGraph("aw.yml", root, func(path string) ([]byte, error) {
		content, ok := manifests[path]
		if !ok {
			return nil, os.ErrNotExist
		}
		return []byte(content), nil
	})
	require.NoError(t, err)
	require.Len(t, nodes, 3)
	assert.Equal(t, []string{"packages/b/aw.yml", "packages/a/aw.yml", "aw.yml"}, []string{nodes[0].Path, nodes[1].Path, nodes[2].Path})
}

func TestResolveRepositoryPackageManifestGraphCycle(t *testing.T) {
	manifests := map[string]string{
		"aw.yml":            "name: Root\nimports:\n  - packages/a/aw.yml\n",
		"packages/a/aw.yml": "name: A\nimports:\n  - ../../aw.yml\n",
	}
	root := &repositoryPackageManifest{Name: "Root", Imports: []string{"packages/a/aw.yml"}}

	_, _, err := resolveRepositoryPackageManifestGraph("aw.yml", root, func(path string) ([]byte, error) {
		return []byte(manifests[path]), nil
	})
	require.ErrorContains(t, err, "package manifest import cycle detected")
	require.ErrorContains(t, err, "aw.yml -> packages/a/aw.yml -> aw.yml")
}

func TestResolveLocalRepositoryPackageImports(t *testing.T) {
	packageDir := t.TempDir()
	writePackageTestFile(t, packageDir, "README.md", "# Bundle\n")
	writePackageTestFile(t, packageDir, "aw.yml", `name: Bundle
imports:
  - activity/aw.yml
  - dashboard/aw.yml
`)
	writePackageTestFile(t, packageDir, "activity/aw.yml", `name: Activity
includes:
  - workflows/activity.md
resources:
  - source: config.json
    destination: .github/aw/activity/config.json
`)
	writePackageTestFile(t, packageDir, "activity/workflows/activity.md", "# Activity\n")
	writePackageTestFile(t, packageDir, "activity/config.json", "{}\n")
	writePackageTestFile(t, packageDir, "dashboard/aw.yml", `name: Dashboard
includes:
  - workflows/dashboard.md
`)
	writePackageTestFile(t, packageDir, "dashboard/workflows/dashboard.md", "# Dashboard\n")

	pkg, err := resolveLocalRepositoryPackage(packageDir)
	require.NoError(t, err)
	require.NotNil(t, pkg)
	require.Len(t, pkg.InstallationSource, 2)
	assert.Equal(t, []string{
		filepath.Join(packageDir, "activity", "workflows", "activity.md"),
		filepath.Join(packageDir, "dashboard", "workflows", "dashboard.md"),
	}, packageInstallableSourcePaths(pkg.InstallationSource))
	require.Len(t, pkg.ResourceFiles, 1)
	assert.Equal(t, filepath.Join(packageDir, "activity", "config.json"), pkg.ResourceFiles[0].SourcePath)
}

func TestResolveLocalRepositoryPackageImportClash(t *testing.T) {
	packageDir := t.TempDir()
	writePackageTestFile(t, packageDir, "README.md", "# Bundle\n")
	writePackageTestFile(t, packageDir, "aw.yml", `name: Bundle
imports:
  - first/aw.yml
  - second/aw.yml
`)
	for _, dir := range []string{"first", "second"} {
		writePackageTestFile(t, packageDir, dir+"/aw.yml", `name: Child
includes:
  - workflows/shared.md
`)
		writePackageTestFile(t, packageDir, dir+"/workflows/shared.md", "# Shared\n")
	}

	_, err := resolveLocalRepositoryPackage(packageDir)
	require.ErrorContains(t, err, "both install to")
	require.ErrorContains(t, err, ".github/workflows/shared.md")
}

func writePackageTestFile(t *testing.T, root, relativePath, content string) {
	t.Helper()
	fullPath := filepath.Join(root, filepath.FromSlash(relativePath))
	require.NoError(t, os.MkdirAll(filepath.Dir(fullPath), 0o755))
	require.NoError(t, os.WriteFile(fullPath, []byte(content), 0o644))
}
