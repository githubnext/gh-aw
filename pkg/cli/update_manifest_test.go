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
	originalDirSubdirs := listPackageDirSubdirsForHost
	originalDirFiles := listPackageDirFilesForHost
	t.Cleanup(func() {
		resolveLatestRefFn = originalResolveLatestRef
		downloadPackageFileFromGitHubForHost = originalDownloadPackage
		listPackageWorkflowFilesForHost = originalListPackage
		getRepositoryPackageDefaultBranch = originalDefaultBranch
		downloadWorkflowContentFn = originalDownloadWorkflow
		listPackageDirSubdirsForHost = originalDirSubdirs
		listPackageDirFilesForHost = originalDirFiles
	})

	resolveLatestRefFn = func(ctx context.Context, repo, currentRef string, allowMajor, verbose bool, coolDown time.Duration) (string, error) {
		return "v2.0.0", nil
	}
	getRepositoryPackageDefaultBranch = func(repoSlug, host string) (string, error) {
		return "main", nil
	}
	downloadPackageFileFromGitHubForHost = func(owner, repo, path, ref, host string) ([]byte, error) {
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
	listPackageWorkflowFilesForHost = func(owner, repo, ref, workflowPath, host string) ([]string, error) {
		return nil, errors.New("unexpected scan")
	}
	// Return not-found so skill/agent auto-scan skips gracefully (no real network needed)
	listPackageDirSubdirsForHost = func(owner, repo, ref, dirPath, host string) ([]string, error) {
		return nil, createRepositoryPackageNotFoundError(dirPath)
	}
	listPackageDirFilesForHost = func(owner, repo, ref, dirPath, host string) ([]string, error) {
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
			return []byte("---\non: push\n---\n\n# Existing new\n"), nil
		case "workflows/new.md@v2.0.0":
			return []byte("---\non: push\n---\n\n# New workflow\n"), nil
		}
		return nil, fmt.Errorf("unexpected download %s@%s", path, ref)
	}

	tmpDir := testutil.TempDir(t, "manifest-update-*")
	existingPath := filepath.Join(tmpDir, "existing.md")
	removedPath := filepath.Join(tmpDir, "removed.md")
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
}
