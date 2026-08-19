//go:build !integration

package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/github/gh-aw/pkg/testutil"
)

func TestUpdateManifestWorkflowGroup_AddsUpdatesRemoves(t *testing.T) {
	originalResolveLatestRef := resolveLatestRefFn
	originalDownloadPackage := downloadPackageFileFromGitHubForHost
	originalListPackage := listPackageWorkflowFilesForHost
	originalDefaultBranch := getRepositoryPackageDefaultBranch
	originalDownloadWorkflow := downloadWorkflowContentFn
	originalDownloadImport := downloadRemoteImportFile
	originalDirSubdirs := listPackageDirSubdirsForHost
	originalDirFiles := listPackageDirFilesForHost
	t.Cleanup(func() {
		resolveLatestRefFn = originalResolveLatestRef
		downloadPackageFileFromGitHubForHost = originalDownloadPackage
		listPackageWorkflowFilesForHost = originalListPackage
		getRepositoryPackageDefaultBranch = originalDefaultBranch
		downloadWorkflowContentFn = originalDownloadWorkflow
		downloadRemoteImportFile = originalDownloadImport
		listPackageDirSubdirsForHost = originalDirSubdirs
		listPackageDirFilesForHost = originalDirFiles
	})

	resolveLatestRefFn = func(ctx context.Context, repo, currentRef string, allowMajor, verbose bool, coolDown time.Duration) (latestRefResolution, error) {
		return latestRefResolution{Ref: "v2.0.0"}, nil
	}
	getRepositoryPackageDefaultBranch = func(_ context.Context, repoSlug, host string) (string, error) {
		return "main", nil
	}
	downloadPackageFileFromGitHubForHost = func(_ context.Context, owner, repo, path, ref, host string) ([]byte, error) {
		switch path {
		case "aw.yml":
			if ref == "v1.0.0" {
				return []byte("name: Test Package\nfiles:\n  - workflows/existing.md\n  - workflows/removed.md\n"), nil
			}
			if ref == "v2.0.0" {
				return []byte("name: Test Package\nfiles:\n  - workflows/existing.md\n  - workflows/new.md\n"), nil
			}
		case "README.md":
			return []byte("# Test Package\n"), nil
		}
		return nil, createRepositoryPackageNotFoundError(path)
	}
	listPackageWorkflowFilesForHost = func(_ context.Context, owner, repo, ref, workflowPath, host string) ([]string, error) {
		return nil, errors.New("unexpected scan")
	}
	// Return not-found so skill/agent auto-scan skips gracefully (no real network needed)
	listPackageDirSubdirsForHost = func(_ context.Context, owner, repo, ref, dirPath, host string) ([]string, error) {
		return nil, createRepositoryPackageNotFoundError(dirPath)
	}
	listPackageDirFilesForHost = func(_ context.Context, owner, repo, ref, dirPath, host string) ([]string, error) {
		return nil, createRepositoryPackageNotFoundError(dirPath)
	}

	downloadWorkflowContentFn = func(_ context.Context, repo, path, ref string, _ bool) ([]byte, error) {
		if repo != "owner/repo" {
			return nil, fmt.Errorf("unexpected repo %s", repo)
		}
		switch path + "@" + ref {
		case "workflows/existing.md@v1.0.0":
			return []byte("---\non: push\n---\n\n# Existing old\n"), nil
		case "workflows/existing.md@v2.0.0":
			return []byte("---\non: push\nimports:\n  - shared/control.md\n---\n\n# Existing new\n"), nil
		case "workflows/new.md@v2.0.0":
			return []byte("---\non: push\nimports:\n  - shared/new-helper.md\n---\n\n# New workflow\n"), nil
		}
		return nil, fmt.Errorf("unexpected download %s@%s", path, ref)
	}
	downloadRemoteImportFile = func(_ context.Context, owner, repo, path, ref string) ([]byte, error) {
		if owner != "owner" || repo != "repo" || ref != "v2.0.0" {
			return nil, fmt.Errorf("unexpected import source %s/%s@%s", owner, repo, ref)
		}
		switch path {
		case "workflows/shared/control.md":
			return []byte("---\nimports:\n  - control-precompute.md\n---\n\n# Control v2\n"), nil
		case "workflows/shared/control-precompute.md":
			return []byte("# Control precompute v2\n"), nil
		case "workflows/shared/new-helper.md":
			return []byte("# New helper v2\n"), nil
		default:
			return nil, fmt.Errorf("unexpected import download %s", path)
		}
	}

	tmpDir := testutil.TempDir(t, "manifest-update-*")
	existingPath := filepath.Join(tmpDir, "existing.md")
	removedPath := filepath.Join(tmpDir, "removed.md")
	sharedDir := filepath.Join(tmpDir, "shared")
	if err := os.MkdirAll(sharedDir, 0o755); err != nil {
		t.Fatalf("create shared directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(sharedDir, "control.md"), []byte("# Control v1\n"), 0o644); err != nil {
		t.Fatalf("write stale shared control: %v", err)
	}
	if err := os.WriteFile(filepath.Join(sharedDir, "control-precompute.md"), []byte("# Control precompute v1\n"), 0o644); err != nil {
		t.Fatalf("write stale shared precompute: %v", err)
	}
	if err := os.WriteFile(existingPath, []byte("---\nsource: owner/repo@v1.0.0\n---\n\n# Existing old\n"), 0o644); err != nil {
		t.Fatalf("write existing: %v", err)
	}
	if err := os.WriteFile(removedPath, []byte("---\nsource: owner/repo@v1.0.0\n---\n\n# Removed old\n"), 0o644); err != nil {
		t.Fatalf("write removed: %v", err)
	}

	successes, failures := updateManifestWorkflowGroup(context.Background(), "owner/repo@v1.0.0", []*workflowWithSource{
		{Name: "existing", Path: existingPath, SourceSpec: "owner/repo@v1.0.0"},
		{Name: "removed", Path: removedPath, SourceSpec: "owner/repo@v1.0.0"},
	}, UpdateWorkflowsOptions{
		NoMerge:                true,
		NoCompile:              true,
		DisableSecurityScanner: true,
	})
	if len(failures) > 0 {
		t.Fatalf("unexpected failures: %+v", failures)
	}
	if len(successes) != 3 {
		t.Fatalf("expected 3 successful operations, got %d", len(successes))
	}

	if _, err := os.Stat(removedPath); !os.IsNotExist(err) {
		t.Fatalf("expected removed workflow to be deleted, got err=%v", err)
	}
	updatedExisting, err := os.ReadFile(existingPath)
	if err != nil {
		t.Fatalf("read existing: %v", err)
	}
	if !strings.Contains(string(updatedExisting), "# Existing new") || !strings.Contains(string(updatedExisting), "source: owner/repo@v2.0.0") {
		t.Fatalf("existing workflow not updated as expected:\n%s", string(updatedExisting))
	}
	newPath := filepath.Join(tmpDir, "new.md")
	newContent, err := os.ReadFile(newPath)
	if err != nil {
		t.Fatalf("new workflow not added: %v", err)
	}
	if !strings.Contains(string(newContent), "# New workflow") || !strings.Contains(string(newContent), "source: owner/repo@v2.0.0") {
		t.Fatalf("new workflow content unexpected:\n%s", string(newContent))
	}
	updatedControl, err := os.ReadFile(filepath.Join(sharedDir, "control.md"))
	if err != nil {
		t.Fatalf("read updated shared control: %v", err)
	}
	if !strings.Contains(string(updatedControl), "# Control v2") {
		t.Fatalf("shared control was not updated:\n%s", string(updatedControl))
	}
	updatedPrecompute, err := os.ReadFile(filepath.Join(sharedDir, "control-precompute.md"))
	if err != nil {
		t.Fatalf("read updated shared precompute: %v", err)
	}
	if !strings.Contains(string(updatedPrecompute), "# Control precompute v2") {
		t.Fatalf("transitive shared dependency was not updated:\n%s", string(updatedPrecompute))
	}
	newHelper, err := os.ReadFile(filepath.Join(sharedDir, "new-helper.md"))
	if err != nil {
		t.Fatalf("read new workflow dependency: %v", err)
	}
	if !strings.Contains(string(newHelper), "# New helper v2") {
		t.Fatalf("new workflow dependency was not installed:\n%s", string(newHelper))
	}

	downloadRemoteImportFile = func(_ context.Context, owner, repo, path, ref string) ([]byte, error) {
		t.Errorf("unexpected dependency fetch for unchanged workflow: %s/%s/%s@%s", owner, repo, path, ref)
		return nil, errors.New("unexpected dependency fetch")
	}
	if err := updateManifestManagedWorkflow(context.Background(), manifestManagedWorkflowUpdate{
		wf:             &workflowWithSource{Name: "existing", Path: existingPath},
		repo:           "owner/repo",
		currentPath:    "workflows/existing.md",
		latestPath:     "workflows/existing.md",
		currentRef:     "v2.0.0",
		latestRef:      "v2.0.0",
		manifestSource: "owner/repo@v2.0.0",
	}, UpdateWorkflowsOptions{
		NoMerge:                true,
		NoCompile:              true,
		DisableSecurityScanner: true,
	}); err != nil {
		t.Fatalf("update unchanged workflow: %v", err)
	}
}
