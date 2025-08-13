package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// FileTracker keeps track of files created or modified during workflow operations
// to enable proper staging and rollback functionality
type FileTracker struct {
	CreatedFiles    []string
	ModifiedFiles   []string
	OriginalContent map[string][]byte // Store original content for rollback
	gitRoot         string
}

// NewFileTracker creates a new file tracker
func NewFileTracker() (*FileTracker, error) {
	gitRoot, err := findGitRoot()
	if err != nil {
		return nil, fmt.Errorf("file tracker requires being in a git repository: %w", err)
	}
	return &FileTracker{
		CreatedFiles:    make([]string, 0),
		ModifiedFiles:   make([]string, 0),
		OriginalContent: make(map[string][]byte),
		gitRoot:         gitRoot,
	}, nil
}

// TrackCreated adds a file to the created files list
func (ft *FileTracker) TrackCreated(filePath string) {
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		absPath = filePath
	}
	ft.CreatedFiles = append(ft.CreatedFiles, absPath)
}

// TrackModified adds a file to the modified files list and stores its original content
func (ft *FileTracker) TrackModified(filePath string) {
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		absPath = filePath
	}

	// Store original content if not already stored
	if _, exists := ft.OriginalContent[absPath]; !exists {
		if content, err := os.ReadFile(absPath); err == nil {
			ft.OriginalContent[absPath] = content
		}
	}

	ft.ModifiedFiles = append(ft.ModifiedFiles, absPath)
}

// GetAllFiles returns all tracked files (created and modified)
func (ft *FileTracker) GetAllFiles() []string {
	all := make([]string, 0, len(ft.CreatedFiles)+len(ft.ModifiedFiles))
	all = append(all, ft.CreatedFiles...)
	all = append(all, ft.ModifiedFiles...)
	return all
}

// StageAllFiles stages all tracked files using git add
func (ft *FileTracker) StageAllFiles(verbose bool) error {
	allFiles := ft.GetAllFiles()
	if len(allFiles) == 0 {
		if verbose {
			fmt.Println("No files to stage")
		}
		return nil
	}

	if verbose {
		fmt.Printf("Staging %d files...\n", len(allFiles))
		for _, file := range allFiles {
			fmt.Printf("  - %s\n", file)
		}
	}

	// Stage all files in a single git add command
	args := append([]string{"add"}, allFiles...)
	cmd := exec.Command("git", args...)
	cmd.Dir = ft.gitRoot
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to stage files: %w", err)
	}

	return nil
}

// RollbackCreatedFiles deletes all files that were created during the operation
func (ft *FileTracker) RollbackCreatedFiles(verbose bool) error {
	if len(ft.CreatedFiles) == 0 {
		return nil
	}

	if verbose {
		fmt.Printf("Rolling back %d created files...\n", len(ft.CreatedFiles))
	}

	var errors []string
	for _, file := range ft.CreatedFiles {
		if verbose {
			fmt.Printf("  - Deleting %s\n", file)
		}
		if err := os.Remove(file); err != nil && !os.IsNotExist(err) {
			errors = append(errors, fmt.Sprintf("failed to delete %s: %v", file, err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("rollback errors: %s", strings.Join(errors, "; "))
	}

	return nil
}

// RollbackModifiedFiles restores all modified files to their original state
func (ft *FileTracker) RollbackModifiedFiles(verbose bool) error {
	if len(ft.ModifiedFiles) == 0 {
		return nil
	}

	if verbose {
		fmt.Printf("Rolling back %d modified files...\n", len(ft.ModifiedFiles))
	}

	var errors []string
	for _, file := range ft.ModifiedFiles {
		if verbose {
			fmt.Printf("  - Restoring %s\n", file)
		}

		// Restore original content if we have it
		if originalContent, exists := ft.OriginalContent[file]; exists {
			if err := os.WriteFile(file, originalContent, 0644); err != nil {
				errors = append(errors, fmt.Sprintf("failed to restore %s: %v", file, err))
			}
		} else {
			if verbose {
				fmt.Printf("    Warning: No original content stored for %s\n", file)
			}
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("rollback errors: %s", strings.Join(errors, "; "))
	}

	return nil
}

// RollbackAllFiles rolls back both created and modified files
func (ft *FileTracker) RollbackAllFiles(verbose bool) error {
	var errors []string

	if err := ft.RollbackCreatedFiles(verbose); err != nil {
		errors = append(errors, fmt.Sprintf("created files rollback: %v", err))
	}

	if err := ft.RollbackModifiedFiles(verbose); err != nil {
		errors = append(errors, fmt.Sprintf("modified files rollback: %v", err))
	}

	if len(errors) > 0 {
		return fmt.Errorf("rollback errors: %s", strings.Join(errors, "; "))
	}

	return nil
}
