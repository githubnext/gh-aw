//go:build !integration

package cli

import (
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/testutil"
)

// Note: The following tests exist in other test files and are not duplicated here:
// - TestIsGitRepo is in commands_utils_test.go (tests isGitRepo utility)
// - TestFindGitRoot is in gitroot_test.go (tests findGitRoot utility)
// - TestEnsureGitAttributes is in gitattributes_test.go (comprehensive gitattributes tests)
//
// Note: The following tests remain in commands_compile_workflow_test.go because they test
// compile-specific workflow behavior, not just Git operations:
// - TestStageWorkflowChanges (tests staging behavior during workflow compilation)
// - TestStageGitAttributesIfChanged (tests conditional staging during compilation)

func TestGetCurrentBranch(t *testing.T) {
	tmpDir := testutil.TempDir(t, "test-*")

	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer func() {
		_ = os.Chdir(originalDir)
	}()

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	// Initialize git repo
	if err := exec.Command("git", "init").Run(); err != nil {
		t.Skip("Git not available")
	}

	// Configure git user for commits
	exec.Command("git", "config", "user.name", "Test User").Run()
	exec.Command("git", "config", "user.email", "test@example.com").Run()

	// Create initial commit to establish branch
	if err := os.WriteFile("test.txt", []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	exec.Command("git", "add", "test.txt").Run()
	if err := exec.Command("git", "commit", "-m", "Initial commit").Run(); err != nil {
		t.Skip("Failed to create initial commit")
	}

	// Get current branch
	branch, err := getCurrentBranch()
	if err != nil {
		t.Fatalf("getCurrentBranch() failed: %v", err)
	}

	// Should be on main or master branch
	if branch != "main" && branch != "master" {
		t.Logf("Note: branch name is %q (expected 'main' or 'master')", branch)
	}

	// Verify it's not empty
	if branch == "" {
		t.Error("getCurrentBranch() returned empty branch name")
	}
}

func TestGetCurrentBranchNotInRepo(t *testing.T) {
	tmpDir := testutil.TempDir(t, "test-*")

	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer func() {
		_ = os.Chdir(originalDir)
	}()

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	// Don't initialize git - should error
	_, err = getCurrentBranch()
	if err == nil {
		t.Error("getCurrentBranch() should return error when not in git repo")
	}
}

func TestCreateAndSwitchBranch(t *testing.T) {
	tmpDir := testutil.TempDir(t, "test-*")

	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer func() {
		_ = os.Chdir(originalDir)
	}()

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	// Initialize git repo
	if err := exec.Command("git", "init").Run(); err != nil {
		t.Skip("Git not available")
	}

	// Configure git
	exec.Command("git", "config", "user.name", "Test User").Run()
	exec.Command("git", "config", "user.email", "test@example.com").Run()

	// Create initial commit
	if err := os.WriteFile("test.txt", []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	exec.Command("git", "add", "test.txt").Run()
	if err := exec.Command("git", "commit", "-m", "Initial commit").Run(); err != nil {
		t.Skip("Failed to create initial commit")
	}

	// Create and switch to new branch
	branchName := "test-branch"
	err = createAndSwitchBranch(branchName, false)
	if err != nil {
		t.Fatalf("createAndSwitchBranch() failed: %v", err)
	}

	// Verify we're on the new branch
	currentBranch, err := getCurrentBranch()
	if err != nil {
		t.Fatalf("getCurrentBranch() failed: %v", err)
	}

	if currentBranch != branchName {
		t.Errorf("Expected to be on branch %q, got %q", branchName, currentBranch)
	}
}

func TestSwitchBranch(t *testing.T) {
	tmpDir := testutil.TempDir(t, "test-*")

	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer func() {
		_ = os.Chdir(originalDir)
	}()

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	// Initialize git repo
	if err := exec.Command("git", "init").Run(); err != nil {
		t.Skip("Git not available")
	}

	// Configure git
	exec.Command("git", "config", "user.name", "Test User").Run()
	exec.Command("git", "config", "user.email", "test@example.com").Run()

	// Create initial commit
	if err := os.WriteFile("test.txt", []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	exec.Command("git", "add", "test.txt").Run()
	if err := exec.Command("git", "commit", "-m", "Initial commit").Run(); err != nil {
		t.Skip("Failed to create initial commit")
	}

	// Get initial branch name
	initialBranch, err := getCurrentBranch()
	if err != nil {
		t.Fatalf("getCurrentBranch() failed: %v", err)
	}

	// Create a new branch
	newBranch := "feature-branch"
	if err := exec.Command("git", "checkout", "-b", newBranch).Run(); err != nil {
		t.Fatalf("Failed to create new branch: %v", err)
	}

	// Switch back to initial branch
	err = switchBranch(initialBranch, false)
	if err != nil {
		t.Fatalf("switchBranch() failed: %v", err)
	}

	// Verify we're on the initial branch
	currentBranch, err := getCurrentBranch()
	if err != nil {
		t.Fatalf("getCurrentBranch() failed: %v", err)
	}

	if currentBranch != initialBranch {
		t.Errorf("Expected to be on branch %q, got %q", initialBranch, currentBranch)
	}
}

func TestCommitChanges(t *testing.T) {
	tmpDir := testutil.TempDir(t, "test-*")

	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer func() {
		_ = os.Chdir(originalDir)
	}()

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	// Initialize git repo
	if err := exec.Command("git", "init").Run(); err != nil {
		t.Skip("Git not available")
	}

	// Configure git
	exec.Command("git", "config", "user.name", "Test User").Run()
	exec.Command("git", "config", "user.email", "test@example.com").Run()

	// Create and stage a file
	if err := os.WriteFile("test.txt", []byte("test content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	if err := exec.Command("git", "add", "test.txt").Run(); err != nil {
		t.Fatalf("Failed to stage file: %v", err)
	}

	// Commit changes
	commitMessage := "Test commit"
	err = commitChanges(commitMessage, false)
	if err != nil {
		t.Fatalf("commitChanges() failed: %v", err)
	}

	// Verify commit was created
	cmd := exec.Command("git", "log", "--oneline", "-1")
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("Failed to get git log: %v", err)
	}

	if !strings.Contains(string(output), commitMessage) {
		t.Errorf("Expected commit message %q not found in git log", commitMessage)
	}
}

// Note: TestStageWorkflowChanges is in commands_compile_workflow_test.go
// Note: TestStageGitAttributesIfChanged is in commands_compile_workflow_test.go

func TestPushBranchNotImplemented(t *testing.T) {
	// This test verifies the function signature exists
	// We skip actual push testing as it requires remote repository setup
	tmpDir := testutil.TempDir(t, "test-*")

	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer func() {
		_ = os.Chdir(originalDir)
	}()

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	// Initialize git repo
	if err := exec.Command("git", "init").Run(); err != nil {
		t.Skip("Git not available")
	}

	// pushBranch will fail without a remote, which is expected
	err = pushBranch("test-branch", false)
	if err == nil {
		t.Log("pushBranch() succeeded unexpectedly (might have remote configured)")
	}
	// We expect this to fail in test environment, which is fine
}

func TestCheckWorkflowFileStatus(t *testing.T) {
	tmpDir := testutil.TempDir(t, "test-*")

	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer func() {
		_ = os.Chdir(originalDir)
	}()

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	// Initialize git repo
	if err := exec.Command("git", "init").Run(); err != nil {
		t.Skip("Git not available")
	}

	// Configure git
	exec.Command("git", "config", "user.name", "Test User").Run()
	exec.Command("git", "config", "user.email", "test@example.com").Run()

	// Create .github/workflows directory
	workflowDir := ".github/workflows"
	if err := os.MkdirAll(workflowDir, 0755); err != nil {
		t.Fatalf("Failed to create workflow directory: %v", err)
	}

	workflowFile := ".github/workflows/test.md"

	// Test 1: File doesn't exist - should return empty status
	t.Run("file_not_tracked", func(t *testing.T) {
		status, err := checkWorkflowFileStatus(workflowFile)
		if err != nil {
			t.Fatalf("checkWorkflowFileStatus() failed: %v", err)
		}
		if status.IsModified || status.IsStaged || status.HasUnpushedCommits {
			t.Error("Expected empty status for untracked file")
		}
	})

	// Create and commit a workflow file
	if err := os.WriteFile(workflowFile, []byte("# Test Workflow\n"), 0644); err != nil {
		t.Fatalf("Failed to create workflow file: %v", err)
	}
	exec.Command("git", "add", workflowFile).Run()
	if err := exec.Command("git", "commit", "-m", "Add workflow").Run(); err != nil {
		t.Skip("Failed to create initial commit")
	}

	// Test 2: Clean file - no changes
	t.Run("clean_file", func(t *testing.T) {
		status, err := checkWorkflowFileStatus(workflowFile)
		if err != nil {
			t.Fatalf("checkWorkflowFileStatus() failed: %v", err)
		}
		if status.IsModified || status.IsStaged || status.HasUnpushedCommits {
			t.Error("Expected empty status for clean file")
		}
	})

	// Test 3: Modified file (unstaged changes)
	t.Run("modified_file", func(t *testing.T) {
		if err := os.WriteFile(workflowFile, []byte("# Modified Workflow\n"), 0644); err != nil {
			t.Fatalf("Failed to modify workflow file: %v", err)
		}

		status, err := checkWorkflowFileStatus(workflowFile)
		if err != nil {
			t.Fatalf("checkWorkflowFileStatus() failed: %v", err)
		}

		if !status.IsModified {
			t.Error("Expected IsModified to be true for modified file")
		}
		if status.IsStaged {
			t.Error("Expected IsStaged to be false for unstaged file")
		}

		// Clean up - restore file
		exec.Command("git", "checkout", workflowFile).Run()
	})

	// Test 4: Staged file
	t.Run("staged_file", func(t *testing.T) {
		if err := os.WriteFile(workflowFile, []byte("# Staged Workflow\n"), 0644); err != nil {
			t.Fatalf("Failed to modify workflow file: %v", err)
		}
		exec.Command("git", "add", workflowFile).Run()

		status, err := checkWorkflowFileStatus(workflowFile)
		if err != nil {
			t.Fatalf("checkWorkflowFileStatus() failed: %v", err)
		}

		if !status.IsStaged {
			t.Error("Expected IsStaged to be true for staged file")
		}

		// Clean up - unstage and restore file
		exec.Command("git", "reset", "HEAD", workflowFile).Run()
		exec.Command("git", "checkout", workflowFile).Run()
	})

	// Test 5: Both staged and modified
	t.Run("staged_and_modified", func(t *testing.T) {
		// Modify and stage
		if err := os.WriteFile(workflowFile, []byte("# Staged content\n"), 0644); err != nil {
			t.Fatalf("Failed to modify workflow file: %v", err)
		}
		exec.Command("git", "add", workflowFile).Run()

		// Modify again (unstaged change)
		if err := os.WriteFile(workflowFile, []byte("# Staged and modified\n"), 0644); err != nil {
			t.Fatalf("Failed to modify workflow file again: %v", err)
		}

		status, err := checkWorkflowFileStatus(workflowFile)
		if err != nil {
			t.Fatalf("checkWorkflowFileStatus() failed: %v", err)
		}

		if !status.IsStaged {
			t.Error("Expected IsStaged to be true")
		}
		if !status.IsModified {
			t.Error("Expected IsModified to be true")
		}

		// Clean up - unstage and restore file
		exec.Command("git", "reset", "HEAD", workflowFile).Run()
		exec.Command("git", "checkout", workflowFile).Run()
	})
}

func TestCheckWorkflowFileStatusNotInRepo(t *testing.T) {
	tmpDir := testutil.TempDir(t, "test-*")

	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer func() {
		_ = os.Chdir(originalDir)
	}()

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	// Don't initialize git - should return empty status without error
	status, err := checkWorkflowFileStatus("test.md")
	if err != nil {
		t.Fatalf("checkWorkflowFileStatus() failed: %v", err)
	}

	// Should return empty status for non-git directory
	if status.IsModified || status.IsStaged || status.HasUnpushedCommits {
		t.Error("Expected empty status when not in git repository")
	}
}

func TestExtractHostFromRemoteURL(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected string
	}{
		{
			name:     "public GitHub HTTPS",
			url:      "https://github.com/owner/repo.git",
			expected: "github.com",
		},
		{
			name:     "public GitHub SSH scp-like",
			url:      "git@github.com:owner/repo.git",
			expected: "github.com",
		},
		{
			name:     "GHES HTTPS",
			url:      "https://ghes.example.com/org/repo.git",
			expected: "ghes.example.com",
		},
		{
			name:     "GHES SSH scp-like",
			url:      "git@ghes.example.com:org/repo.git",
			expected: "ghes.example.com",
		},
		{
			name:     "GHES HTTPS without .git suffix",
			url:      "https://ghes.example.com/org/repo",
			expected: "ghes.example.com",
		},
		{
			name:     "SSH URL format with user",
			url:      "ssh://git@ghes.example.com/org/repo.git",
			expected: "ghes.example.com",
		},
		{
			name:     "SSH URL format without user",
			url:      "ssh://ghes.example.com/org/repo.git",
			expected: "ghes.example.com",
		},
		{
			name:     "HTTP URL",
			url:      "http://ghes.example.com/org/repo.git",
			expected: "ghes.example.com",
		},
		{
			name:     "empty URL defaults to github.com",
			url:      "",
			expected: "github.com",
		},
		{
			name:     "unrecognized URL defaults to github.com",
			url:      "not-a-url",
			expected: "github.com",
		},
		{
			name:     "GHES with port",
			url:      "https://ghes.example.com:8443/org/repo.git",
			expected: "ghes.example.com:8443",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractHostFromRemoteURL(tt.url)
			if got != tt.expected {
				t.Errorf("extractHostFromRemoteURL(%q) = %q, want %q", tt.url, got, tt.expected)
			}
		})
	}
}

func TestGetHostFromOriginRemote(t *testing.T) {
	tmpDir := testutil.TempDir(t, "test-get-host-*")

	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer func() {
		if err := os.Chdir(originalDir); err != nil {
			t.Logf("Warning: failed to restore directory: %v", err)
		}
	}()

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	// Initialize a git repo
	if err := exec.Command("git", "init").Run(); err != nil {
		t.Skip("Git not available")
	}

	t.Run("no remote defaults to github.com", func(t *testing.T) {
		got := getHostFromOriginRemote()
		if got != "github.com" {
			t.Errorf("getHostFromOriginRemote() without remote = %q, want %q", got, "github.com")
		}
	})

	t.Run("public GitHub remote", func(t *testing.T) {
		if err := exec.Command("git", "remote", "add", "origin", "https://github.com/owner/repo.git").Run(); err != nil {
			t.Fatalf("Failed to add remote: %v", err)
		}
		defer func() { _ = exec.Command("git", "remote", "remove", "origin").Run() }()

		got := getHostFromOriginRemote()
		if got != "github.com" {
			t.Errorf("getHostFromOriginRemote() = %q, want %q", got, "github.com")
		}
	})

	t.Run("GHES remote", func(t *testing.T) {
		if err := exec.Command("git", "remote", "add", "origin", "https://ghes.example.com/org/repo.git").Run(); err != nil {
			t.Fatalf("Failed to add remote: %v", err)
		}
		defer func() { _ = exec.Command("git", "remote", "remove", "origin").Run() }()

		got := getHostFromOriginRemote()
		if got != "ghes.example.com" {
			t.Errorf("getHostFromOriginRemote() = %q, want %q", got, "ghes.example.com")
		}
	})

	t.Run("non-origin single remote falls back to that remote", func(t *testing.T) {
		if err := exec.Command("git", "remote", "add", "upstream", "https://github.com/owner/repo.git").Run(); err != nil {
			t.Fatalf("Failed to add remote: %v", err)
		}
		defer func() { _ = exec.Command("git", "remote", "remove", "upstream").Run() }()

		got := getHostFromOriginRemote()
		if got != "github.com" {
			t.Errorf("getHostFromOriginRemote() with non-origin remote = %q, want %q", got, "github.com")
		}
	})

	t.Run("multiple remotes without origin defaults to github.com", func(t *testing.T) {
		if err := exec.Command("git", "remote", "add", "myorg", "https://github.com/myorg/repo.git").Run(); err != nil {
			t.Fatalf("Failed to add first remote: %v", err)
		}
		defer func() { _ = exec.Command("git", "remote", "remove", "myorg").Run() }()
		if err := exec.Command("git", "remote", "add", "other", "https://github.com/other/repo.git").Run(); err != nil {
			t.Fatalf("Failed to add second remote: %v", err)
		}
		defer func() { _ = exec.Command("git", "remote", "remove", "other").Run() }()

		got := getHostFromOriginRemote()
		if got != "github.com" {
			t.Errorf("getHostFromOriginRemote() with multiple non-origin remotes = %q, want %q", got, "github.com")
		}
	})
}

func TestResolveRemoteURL(t *testing.T) {
	tmpDir := testutil.TempDir(t, "test-resolve-remote-*")

	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer func() {
		if err := os.Chdir(originalDir); err != nil {
			t.Logf("Warning: failed to restore directory: %v", err)
		}
	}()

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	// Initialize a git repo
	if err := exec.Command("git", "init").Run(); err != nil {
		t.Skip("Git not available")
	}

	t.Run("no remotes returns error", func(t *testing.T) {
		_, _, err := resolveRemoteURL("")
		if err == nil {
			t.Error("resolveRemoteURL() should return error when no remotes are configured")
		}
	})

	t.Run("origin remote is used when present", func(t *testing.T) {
		if err := exec.Command("git", "remote", "add", "origin", "https://github.com/owner/repo.git").Run(); err != nil {
			t.Fatalf("Failed to add remote: %v", err)
		}
		defer func() { _ = exec.Command("git", "remote", "remove", "origin").Run() }()

		url, name, err := resolveRemoteURL("")
		if err != nil {
			t.Fatalf("resolveRemoteURL() failed: %v", err)
		}
		if name != "origin" {
			t.Errorf("resolveRemoteURL() remote name = %q, want %q", name, "origin")
		}
		if url != "https://github.com/owner/repo.git" {
			t.Errorf("resolveRemoteURL() URL = %q, want %q", url, "https://github.com/owner/repo.git")
		}
	})

	t.Run("single non-origin remote is used as fallback", func(t *testing.T) {
		if err := exec.Command("git", "remote", "add", "myorg", "https://github.com/myorg/repo.git").Run(); err != nil {
			t.Fatalf("Failed to add remote: %v", err)
		}
		defer func() { _ = exec.Command("git", "remote", "remove", "myorg").Run() }()

		url, name, err := resolveRemoteURL("")
		if err != nil {
			t.Fatalf("resolveRemoteURL() failed: %v", err)
		}
		if name != "myorg" {
			t.Errorf("resolveRemoteURL() remote name = %q, want %q", name, "myorg")
		}
		if url != "https://github.com/myorg/repo.git" {
			t.Errorf("resolveRemoteURL() URL = %q, want %q", url, "https://github.com/myorg/repo.git")
		}
	})

	t.Run("multiple non-origin remotes returns error", func(t *testing.T) {
		if err := exec.Command("git", "remote", "add", "remote1", "https://github.com/org1/repo.git").Run(); err != nil {
			t.Fatalf("Failed to add first remote: %v", err)
		}
		defer func() { _ = exec.Command("git", "remote", "remove", "remote1").Run() }()
		if err := exec.Command("git", "remote", "add", "remote2", "https://github.com/org2/repo.git").Run(); err != nil {
			t.Fatalf("Failed to add second remote: %v", err)
		}
		defer func() { _ = exec.Command("git", "remote", "remove", "remote2").Run() }()

		_, _, err := resolveRemoteURL("")
		if err == nil {
			t.Error("resolveRemoteURL() should return error when multiple non-origin remotes are configured")
		}
	})

	t.Run("origin takes precedence when multiple remotes exist", func(t *testing.T) {
		if err := exec.Command("git", "remote", "add", "origin", "https://github.com/owner/repo.git").Run(); err != nil {
			t.Fatalf("Failed to add origin remote: %v", err)
		}
		defer func() { _ = exec.Command("git", "remote", "remove", "origin").Run() }()
		if err := exec.Command("git", "remote", "add", "upstream", "https://github.com/upstream/repo.git").Run(); err != nil {
			t.Fatalf("Failed to add upstream remote: %v", err)
		}
		defer func() { _ = exec.Command("git", "remote", "remove", "upstream").Run() }()

		url, name, err := resolveRemoteURL("")
		if err != nil {
			t.Fatalf("resolveRemoteURL() failed: %v", err)
		}
		if name != "origin" {
			t.Errorf("resolveRemoteURL() remote name = %q, want %q", name, "origin")
		}
		if url != "https://github.com/owner/repo.git" {
			t.Errorf("resolveRemoteURL() URL = %q, want %q", url, "https://github.com/owner/repo.git")
		}
	})
}

func TestGetRepositorySlugFromRemoteFallback(t *testing.T) {
	tmpDir := testutil.TempDir(t, "test-slug-fallback-*")

	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer func() {
		if err := os.Chdir(originalDir); err != nil {
			t.Logf("Warning: failed to restore directory: %v", err)
		}
	}()

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	// Initialize a git repo
	if err := exec.Command("git", "init").Run(); err != nil {
		t.Skip("Git not available")
	}

	t.Run("single non-origin remote provides repo slug", func(t *testing.T) {
		if err := exec.Command("git", "remote", "add", "myorg", "https://github.com/myorg/myrepo.git").Run(); err != nil {
			t.Fatalf("Failed to add remote: %v", err)
		}
		defer func() { _ = exec.Command("git", "remote", "remove", "myorg").Run() }()

		slug := getRepositorySlugFromRemote()
		if slug != "myorg/myrepo" {
			t.Errorf("getRepositorySlugFromRemote() = %q, want %q", slug, "myorg/myrepo")
		}
	})

	t.Run("multiple non-origin remotes returns empty slug", func(t *testing.T) {
		if err := exec.Command("git", "remote", "add", "remote1", "https://github.com/org1/repo1.git").Run(); err != nil {
			t.Fatalf("Failed to add first remote: %v", err)
		}
		defer func() { _ = exec.Command("git", "remote", "remove", "remote1").Run() }()
		if err := exec.Command("git", "remote", "add", "remote2", "https://github.com/org2/repo2.git").Run(); err != nil {
			t.Fatalf("Failed to add second remote: %v", err)
		}
		defer func() { _ = exec.Command("git", "remote", "remove", "remote2").Run() }()

		slug := getRepositorySlugFromRemote()
		if slug != "" {
			t.Errorf("getRepositorySlugFromRemote() with multiple non-origin remotes = %q, want %q", slug, "")
		}
	})
}
