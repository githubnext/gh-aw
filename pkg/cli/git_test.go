package cli

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

// Note: TestIsGitRepo is in commands_utils_test.go
// Note: TestFindGitRoot is in gitroot_test.go
// Note: TestEnsureGitAttributes is in gitattributes_test.go

func TestGetCurrentBranch(t *testing.T) {
	tmpDir := t.TempDir()

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
	tmpDir := t.TempDir()

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
	tmpDir := t.TempDir()

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
	tmpDir := t.TempDir()

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
	tmpDir := t.TempDir()

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
	tmpDir := t.TempDir()

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
