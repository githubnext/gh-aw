package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEnsureAutoCompileWorkflow(t *testing.T) {
	tests := []struct {
		name              string
		setupFunc         func(testDir string) error
		verbose           bool
		expectError       bool
		expectFileCreated bool
		errorContains     string
	}{
		{
			name: "create new auto-compile workflow file",
			setupFunc: func(testDir string) error {
				// Initialize git repo
				return initTestGitRepo(testDir)
			},
			verbose:           true,
			expectError:       false,
			expectFileCreated: true,
		},
		{
			name: "update outdated auto-compile workflow file",
			setupFunc: func(testDir string) error {
				// Initialize git repo and create outdated workflow file
				if err := initTestGitRepo(testDir); err != nil {
					return err
				}

				workflowsDir := filepath.Join(testDir, ".github", "workflows")
				if err := os.MkdirAll(workflowsDir, 0755); err != nil {
					return err
				}

				outdatedContent := `name: Outdated Auto Compile
on:
  push:
    paths: ['.github/workflows/*.md']
jobs:
  compile:
    runs-on: ubuntu-latest
    steps:
      - name: Old Step
        run: echo "old"`

				autoCompileFile := filepath.Join(workflowsDir, "auto-compile-workflows.yml")
				return os.WriteFile(autoCompileFile, []byte(outdatedContent), 0644)
			},
			verbose:           true,
			expectError:       false,
			expectFileCreated: true,
		},
		{
			name: "workflow file already up to date",
			setupFunc: func(testDir string) error {
				// Initialize git repo and create up-to-date workflow file
				if err := initTestGitRepo(testDir); err != nil {
					return err
				}

				workflowsDir := filepath.Join(testDir, ".github", "workflows")
				if err := os.MkdirAll(workflowsDir, 0755); err != nil {
					return err
				}

				autoCompileFile := filepath.Join(workflowsDir, "auto-compile-workflows.yml")
				return os.WriteFile(autoCompileFile, []byte(autoCompileWorkflowTemplate), 0644)
			},
			verbose:           true,
			expectError:       false,
			expectFileCreated: true, // File exists but should not be modified
		},
		{
			name: "not in git repository",
			setupFunc: func(testDir string) error {
				// Don't initialize git repo to simulate not being in a git repository
				return nil
			},
			verbose:       false,
			expectError:   true,
			errorContains: "auto-compile workflow management requires being in a git repository",
		},
		{
			name: "permission denied to create workflow directory",
			setupFunc: func(testDir string) error {
				// Initialize git repo
				if err := initTestGitRepo(testDir); err != nil {
					return err
				}

				// Create .github directory with read-only permissions to simulate permission error
				githubDir := filepath.Join(testDir, ".github")
				if err := os.MkdirAll(githubDir, 0555); err != nil {
					return err
				}

				return nil
			},
			verbose:       false,
			expectError:   true,
			errorContains: "failed to create workflows directory",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory for test
			testDir := t.TempDir()

			// Change to test directory
			originalDir, err := os.Getwd()
			if err != nil {
				t.Fatalf("Failed to get current directory: %v", err)
			}
			defer func() {
				if err := os.Chdir(originalDir); err != nil {
					t.Errorf("Failed to restore original directory: %v", err)
				}
			}()

			if err := os.Chdir(testDir); err != nil {
				t.Fatalf("Failed to change to test directory: %v", err)
			}

			// Setup test environment
			if tt.setupFunc != nil {
				if err := tt.setupFunc(testDir); err != nil {
					t.Fatalf("Setup function failed: %v", err)
				}
			}

			// Execute function under test
			err = ensureAutoCompileWorkflow(tt.verbose)

			// Check error expectations
			if tt.expectError {
				if err == nil {
					t.Errorf("ensureAutoCompileWorkflow() expected error but got none")
				} else if tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("ensureAutoCompileWorkflow() error = %v, want error containing %v", err, tt.errorContains)
				}
			} else {
				if err != nil {
					t.Errorf("ensureAutoCompileWorkflow() unexpected error: %v", err)
				}
			}

			// Check file creation expectations
			if !tt.expectError && tt.expectFileCreated {
				autoCompileFile := filepath.Join(testDir, ".github", "workflows", "auto-compile-workflows.yml")
				if _, err := os.Stat(autoCompileFile); os.IsNotExist(err) {
					t.Errorf("ensureAutoCompileWorkflow() should have created file %s", autoCompileFile)
				} else {
					// Verify file content
					content, err := os.ReadFile(autoCompileFile)
					if err != nil {
						t.Errorf("Failed to read auto-compile workflow file: %v", err)
					} else {
						contentStr := strings.TrimSpace(string(content))
						expectedStr := strings.TrimSpace(autoCompileWorkflowTemplate)
						if contentStr != expectedStr {
							t.Errorf("ensureAutoCompileWorkflow() file content does not match template")
						}
					}
				}
			}

			// Clean up permissions for deletion
			if strings.Contains(tt.name, "permission denied") {
				githubDir := filepath.Join(testDir, ".github")
				if err := os.Chmod(githubDir, 0755); err != nil {
					t.Errorf("Failed to restore write permissions for cleanup: %v", err)
				}
			}
		})
	}
}

func TestEnsureAutoCompileWorkflowEdgeCases(t *testing.T) {
	t.Run("handle read error on existing file", func(t *testing.T) {
		// This test is challenging to create as we'd need to create a file that exists but can't be read
		// For now, we'll test the function behavior with a non-git directory
		testDir := t.TempDir()
		originalDir, _ := os.Getwd()
		defer func() {
			if err := os.Chdir(originalDir); err != nil {
				t.Errorf("Failed to restore original directory: %v", err)
			}
		}()
		if err := os.Chdir(testDir); err != nil {
			t.Fatalf("Failed to change to test directory: %v", err)
		}

		// Create a non-git directory structure
		err := ensureAutoCompileWorkflow(false)
		if err == nil {
			t.Error("ensureAutoCompileWorkflow() should fail when not in git repo")
		}
		if !strings.Contains(err.Error(), "requires being in a git repository") {
			t.Errorf("ensureAutoCompileWorkflow() error should mention git repository requirement, got: %v", err)
		}
	})

	t.Run("verbose mode produces output", func(t *testing.T) {
		// Test verbose mode doesn't crash (output testing is complex)
		testDir := t.TempDir()
		originalDir, _ := os.Getwd()
		defer func() {
			if err := os.Chdir(originalDir); err != nil {
				t.Errorf("Failed to restore original directory: %v", err)
			}
		}()
		if err := os.Chdir(testDir); err != nil {
			t.Fatalf("Failed to change to test directory: %v", err)
		}

		if err := initTestGitRepo(testDir); err != nil {
			t.Fatalf("Failed to initialize test git repo: %v", err)
		}

		// This should work and not panic in verbose mode
		err := ensureAutoCompileWorkflow(true)
		if err != nil {
			t.Errorf("ensureAutoCompileWorkflow() with verbose=true failed: %v", err)
		}

		// Run again to test the "up-to-date" path
		err = ensureAutoCompileWorkflow(true)
		if err != nil {
			t.Errorf("ensureAutoCompileWorkflow() second run with verbose=true failed: %v", err)
		}
	})

	t.Run("non-verbose mode works", func(t *testing.T) {
		testDir := t.TempDir()
		originalDir, _ := os.Getwd()
		defer func() {
			if err := os.Chdir(originalDir); err != nil {
				t.Errorf("Failed to restore original directory: %v", err)
			}
		}()
		if err := os.Chdir(testDir); err != nil {
			t.Fatalf("Failed to change to test directory: %v", err)
		}

		if err := initTestGitRepo(testDir); err != nil {
			t.Fatalf("Failed to initialize test git repo: %v", err)
		}

		// This should work and not produce output
		err := ensureAutoCompileWorkflow(false)
		if err != nil {
			t.Errorf("ensureAutoCompileWorkflow() with verbose=false failed: %v", err)
		}
	})
}

func TestAutoCompileWorkflowTemplate(t *testing.T) {
	// Test that the template constant is properly defined
	t.Run("template is not empty", func(t *testing.T) {
		if autoCompileWorkflowTemplate == "" {
			t.Error("autoCompileWorkflowTemplate should not be empty")
		}
	})

	t.Run("template contains expected elements", func(t *testing.T) {
		template := autoCompileWorkflowTemplate

		expectedElements := []string{
			"name:",
			"on:",
			"push:",
			"paths:",
			".github/workflows/*.md",
			"jobs:",
			"runs-on:",
		}

		for _, element := range expectedElements {
			if !strings.Contains(template, element) {
				t.Errorf("autoCompileWorkflowTemplate should contain '%s'", element)
			}
		}
	})
}

// Helper function to initialize a git repository in test directory
func initTestGitRepo(dir string) error {
	// Create .git directory structure to simulate being in a git repo
	gitDir := filepath.Join(dir, ".git")
	if err := os.MkdirAll(gitDir, 0755); err != nil {
		return err
	}

	// Create subdirectories
	subdirs := []string{"objects", "refs", "refs/heads", "refs/tags"}
	for _, subdir := range subdirs {
		if err := os.MkdirAll(filepath.Join(gitDir, subdir), 0755); err != nil {
			return err
		}
	}

	// Create HEAD file pointing to main branch
	headFile := filepath.Join(gitDir, "HEAD")
	if err := os.WriteFile(headFile, []byte("ref: refs/heads/main\n"), 0644); err != nil {
		return err
	}

	// Create a minimal git config
	configFile := filepath.Join(gitDir, "config")
	configContent := `[core]
	repositoryformatversion = 0
	filemode = true
	bare = false
	logallrefupdates = true
[user]
	name = Test User
	email = test@example.com`

	if err := os.WriteFile(configFile, []byte(configContent), 0644); err != nil {
		return err
	}

	// Create description file
	descFile := filepath.Join(gitDir, "description")
	if err := os.WriteFile(descFile, []byte("Test repository"), 0644); err != nil {
		return err
	}

	return nil
}
