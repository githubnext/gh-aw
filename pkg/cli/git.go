package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func isGitRepo() bool {
	cmd := exec.Command("git", "rev-parse", "--git-dir")
	return cmd.Run() == nil
}

// findGitRoot finds the root directory of the git repository
func findGitRoot() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("not in a git repository or git command failed: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}

func stageWorkflowChanges() {
	// Find git root and add .github/workflows relative to it
	if gitRoot, err := findGitRoot(); err == nil {
		workflowsPath := filepath.Join(gitRoot, ".github/workflows/")
		_ = exec.Command("git", "-C", gitRoot, "add", workflowsPath).Run()

		// Also stage .gitattributes if it was modified
		_ = stageGitAttributesIfChanged()
	} else {
		// Fallback to relative path if git root can't be found
		_ = exec.Command("git", "add", ".github/workflows/").Run()
		_ = exec.Command("git", "add", ".gitattributes").Run()
	}
}

// ensureGitAttributes ensures that .gitattributes contains the entry to mark .lock.yml files as generated
func ensureGitAttributes() error {
	gitRoot, err := findGitRoot()
	if err != nil {
		return err // Not in a git repository, skip
	}

	gitAttributesPath := filepath.Join(gitRoot, ".gitattributes")
	lockYmlEntry := ".github/workflows/*.lock.yml linguist-generated=true merge=ours"

	// Read existing .gitattributes file if it exists
	var lines []string
	if content, err := os.ReadFile(gitAttributesPath); err == nil {
		lines = strings.Split(string(content), "\n")
	}

	// Check if the entry already exists or needs updating
	found := false
	for i, line := range lines {
		trimmedLine := strings.TrimSpace(line)
		if trimmedLine == lockYmlEntry {
			return nil // Entry already exists with correct format
		}
		// Check for old format entry that needs updating
		if strings.HasPrefix(trimmedLine, ".github/workflows/*.lock.yml") {
			lines[i] = lockYmlEntry
			found = true
			break
		}
	}

	// Add the entry if not found
	if !found {
		if len(lines) > 0 && lines[len(lines)-1] != "" {
			lines = append(lines, "") // Add empty line before our entry if file doesn't end with newline
		}
		lines = append(lines, lockYmlEntry)
	}

	// Write back to file
	content := strings.Join(lines, "\n")
	if err := os.WriteFile(gitAttributesPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write .gitattributes: %w", err)
	}

	return nil
}

// stageGitAttributesIfChanged stages .gitattributes if it was modified
func stageGitAttributesIfChanged() error {
	gitRoot, err := findGitRoot()
	if err != nil {
		return err
	}
	gitAttributesPath := filepath.Join(gitRoot, ".gitattributes")
	return exec.Command("git", "-C", gitRoot, "add", gitAttributesPath).Run()
}

// getCurrentBranch gets the current git branch name
func getCurrentBranch() (string, error) {
	cmd := exec.Command("git", "branch", "--show-current")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get current branch: %w", err)
	}

	branch := strings.TrimSpace(string(output))
	if branch == "" {
		return "", fmt.Errorf("could not determine current branch")
	}

	return branch, nil
}

// createAndSwitchBranch creates a new branch and switches to it
func createAndSwitchBranch(branchName string, verbose bool) error {
	if verbose {
		fmt.Printf("Creating and switching to branch: %s\n", branchName)
	}

	cmd := exec.Command("git", "checkout", "-b", branchName)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create and switch to branch %s: %w", branchName, err)
	}

	return nil
}

// switchBranch switches to the specified branch
func switchBranch(branchName string, verbose bool) error {
	if verbose {
		fmt.Printf("Switching to branch: %s\n", branchName)
	}

	cmd := exec.Command("git", "checkout", branchName)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to switch to branch %s: %w", branchName, err)
	}

	return nil
}

// commitChanges commits all staged changes with the given message
func commitChanges(message string, verbose bool) error {
	if verbose {
		fmt.Printf("Committing changes with message: %s\n", message)
	}

	cmd := exec.Command("git", "commit", "-m", message)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to commit changes: %w", err)
	}

	return nil
}

// pushBranch pushes the specified branch to origin
func pushBranch(branchName string, verbose bool) error {
	if verbose {
		fmt.Printf("Pushing branch: %s\n", branchName)
	}

	cmd := exec.Command("git", "push", "-u", "origin", branchName)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to push branch %s: %w", branchName, err)
	}

	return nil
}
