package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/githubnext/gh-aw/pkg/constants"
)

func TestCompileWorkflow(t *testing.T) {
	tests := []struct {
		name           string
		setupWorkflow  func(string) (string, error)
		verbose        bool
		engineOverride string
		expectError    bool
		errorContains  string
	}{
		{
			name: "successful compilation with valid workflow",
			setupWorkflow: func(tmpDir string) (string, error) {
				workflowContent := `---
name: Test Workflow
on:
  push:
    branches: [main]
permissions:
  contents: read
---

# Test Workflow

This is a test workflow for compilation.
`
				workflowsDir := filepath.Join(tmpDir, constants.GetWorkflowDir())
				err := os.MkdirAll(workflowsDir, 0755)
				if err != nil {
					return "", err
				}

				workflowFile := filepath.Join(workflowsDir, "test.md")
				err = os.WriteFile(workflowFile, []byte(workflowContent), 0644)
				return workflowFile, err
			},
			verbose:        false,
			engineOverride: "",
			expectError:    false,
		},
		{
			name: "successful compilation with verbose mode",
			setupWorkflow: func(tmpDir string) (string, error) {
				workflowContent := `---
name: Verbose Test
on:
  schedule:
    - cron: "0 9 * * 1"
permissions:
  contents: write
---

# Verbose Test Workflow

Test workflow with verbose compilation.
`
				workflowsDir := filepath.Join(tmpDir, constants.GetWorkflowDir())
				err := os.MkdirAll(workflowsDir, 0755)
				if err != nil {
					return "", err
				}

				workflowFile := filepath.Join(workflowsDir, "verbose-test.md")
				err = os.WriteFile(workflowFile, []byte(workflowContent), 0644)
				return workflowFile, err
			},
			verbose:        true,
			engineOverride: "",
			expectError:    false,
		},
		{
			name: "compilation with engine override",
			setupWorkflow: func(tmpDir string) (string, error) {
				workflowContent := `---
name: Engine Override Test
on:
  push:
    branches: [main]
permissions:
  contents: read
---

# Engine Override Test

Test compilation with specific engine.
`
				workflowsDir := filepath.Join(tmpDir, constants.GetWorkflowDir())
				err := os.MkdirAll(workflowsDir, 0755)
				if err != nil {
					return "", err
				}

				workflowFile := filepath.Join(workflowsDir, "engine-test.md")
				err = os.WriteFile(workflowFile, []byte(workflowContent), 0644)
				return workflowFile, err
			},
			verbose:        false,
			engineOverride: "claude",
			expectError:    false,
		},
		{
			name: "compilation with invalid workflow file",
			setupWorkflow: func(tmpDir string) (string, error) {
				workflowContent := `---
invalid yaml: [unclosed
---

# Invalid Workflow

This workflow has invalid frontmatter.
`
				workflowsDir := filepath.Join(tmpDir, constants.GetWorkflowDir())
				err := os.MkdirAll(workflowsDir, 0755)
				if err != nil {
					return "", err
				}

				workflowFile := filepath.Join(workflowsDir, "invalid.md")
				err = os.WriteFile(workflowFile, []byte(workflowContent), 0644)
				return workflowFile, err
			},
			verbose:        false,
			engineOverride: "",
			expectError:    true,
			errorContains:  "yaml",
		},
		{
			name: "compilation with nonexistent file",
			setupWorkflow: func(tmpDir string) (string, error) {
				return filepath.Join(tmpDir, "nonexistent.md"), nil
			},
			verbose:        false,
			engineOverride: "",
			expectError:    true,
			errorContains:  "no such file",
		},
		{
			name: "compilation with invalid engine override",
			setupWorkflow: func(tmpDir string) (string, error) {
				workflowContent := `---
name: Invalid Engine Test
on:
  push:
    branches: [main]
permissions:
  contents: read
---

# Invalid Engine Test

Test compilation with invalid engine.
`
				workflowsDir := filepath.Join(tmpDir, constants.GetWorkflowDir())
				err := os.MkdirAll(workflowsDir, 0755)
				if err != nil {
					return "", err
				}

				workflowFile := filepath.Join(workflowsDir, "invalid-engine.md")
				err = os.WriteFile(workflowFile, []byte(workflowContent), 0644)
				return workflowFile, err
			},
			verbose:        false,
			engineOverride: "invalid-engine",
			expectError:    true,
			errorContains:  "invalid engine",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory
			tmpDir := t.TempDir()

			// Initialize git repository in tmp directory
			if err := initTestGitRepo(tmpDir); err != nil {
				t.Fatalf("Failed to initialize git repo: %v", err)
			}

			// Change to temporary directory
			oldDir, err := os.Getwd()
			if err != nil {
				t.Fatalf("Failed to get current directory: %v", err)
			}
			defer func() {
				if err := os.Chdir(oldDir); err != nil {
					t.Errorf("Failed to restore directory: %v", err)
				}
			}()

			if err := os.Chdir(tmpDir); err != nil {
				t.Fatalf("Failed to change to temp directory: %v", err)
			}

			// Setup workflow file
			workflowFile, err := tt.setupWorkflow(tmpDir)
			if err != nil {
				t.Fatalf("Failed to setup workflow: %v", err)
			}

			// Test compileWorkflow function
			err = compileWorkflow(workflowFile, tt.verbose, tt.engineOverride)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("Expected error to contain '%s', but got: %v", tt.errorContains, err)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				} else {
					// Verify lock file was created
					lockFile := strings.TrimSuffix(workflowFile, ".md") + ".lock.yml"
					if _, err := os.Stat(lockFile); os.IsNotExist(err) {
						t.Errorf("Expected lock file %s to be created", lockFile)
					}
				}
			}
		})
	}
}

func TestStageWorkflowChanges(t *testing.T) {
	tests := []struct {
		name          string
		setupRepo     func(string) error
		expectNoError bool
	}{
		{
			name: "successful staging in git repo with workflows",
			setupRepo: func(tmpDir string) error {
				// Initialize git repo
				if err := initTestGitRepo(tmpDir); err != nil {
					return err
				}

				// Create workflows directory with test files
				workflowsDir := filepath.Join(tmpDir, constants.GetWorkflowDir())
				if err := os.MkdirAll(workflowsDir, 0755); err != nil {
					return err
				}

				testFile := filepath.Join(workflowsDir, "test.lock.yml")
				return os.WriteFile(testFile, []byte("test: content"), 0644)
			},
			expectNoError: true,
		},
		{
			name: "staging works even without workflows directory",
			setupRepo: func(tmpDir string) error {
				return initTestGitRepo(tmpDir)
			},
			expectNoError: true,
		},
		{
			name: "staging in non-git directory falls back gracefully",
			setupRepo: func(tmpDir string) error {
				// Don't initialize git repo - should use fallback
				workflowsDir := filepath.Join(tmpDir, constants.GetWorkflowDir())
				if err := os.MkdirAll(workflowsDir, 0755); err != nil {
					return err
				}

				testFile := filepath.Join(workflowsDir, "test.lock.yml")
				return os.WriteFile(testFile, []byte("test: content"), 0644)
			},
			expectNoError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory
			tmpDir := t.TempDir()

			// Setup repository
			if err := tt.setupRepo(tmpDir); err != nil {
				t.Fatalf("Failed to setup repo: %v", err)
			}

			// Change to temporary directory
			oldDir, err := os.Getwd()
			if err != nil {
				t.Fatalf("Failed to get current directory: %v", err)
			}
			defer func() {
				if err := os.Chdir(oldDir); err != nil {
					t.Errorf("Failed to restore directory: %v", err)
				}
			}()

			if err := os.Chdir(tmpDir); err != nil {
				t.Fatalf("Failed to change to temp directory: %v", err)
			}

			// Test stageWorkflowChanges function - should not panic
			func() {
				defer func() {
					if r := recover(); r != nil {
						if tt.expectNoError {
							t.Errorf("Function panicked unexpectedly: %v", r)
						}
					}
				}()
				stageWorkflowChanges()
			}()
		})
	}
}

func TestStageGitAttributesIfChanged(t *testing.T) {
	tests := []struct {
		name          string
		setupRepo     func(string) error
		expectError   bool
		errorContains string
	}{
		{
			name: "successful staging in git repo",
			setupRepo: func(tmpDir string) error {
				if err := initTestGitRepo(tmpDir); err != nil {
					return err
				}

				// Create .gitattributes file
				gitattributesPath := filepath.Join(tmpDir, ".gitattributes")
				return os.WriteFile(gitattributesPath, []byte("*.lock.yml linguist-generated=true"), 0644)
			},
			expectError: false,
		},
		{
			name: "staging without .gitattributes file",
			setupRepo: func(tmpDir string) error {
				return initTestGitRepo(tmpDir)
			},
			expectError:   true, // git add may fail on missing files in some git versions
			errorContains: "exit status",
		},
		{
			name: "error in non-git directory",
			setupRepo: func(tmpDir string) error {
				// Don't initialize git repo
				return nil
			},
			expectError:   true,
			errorContains: "git",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory
			tmpDir := t.TempDir()

			// Setup repository
			if err := tt.setupRepo(tmpDir); err != nil {
				t.Fatalf("Failed to setup repo: %v", err)
			}

			// Change to temporary directory
			oldDir, err := os.Getwd()
			if err != nil {
				t.Fatalf("Failed to get current directory: %v", err)
			}
			defer func() {
				if err := os.Chdir(oldDir); err != nil {
					t.Errorf("Failed to restore directory: %v", err)
				}
			}()

			if err := os.Chdir(tmpDir); err != nil {
				t.Fatalf("Failed to change to temp directory: %v", err)
			}

			// Test stageGitAttributesIfChanged function
			err = stageGitAttributesIfChanged()

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("Expected error to contain '%s', but got: %v", tt.errorContains, err)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
			}
		})
	}
}

func TestCompileWorkflowsWithWorkflowID(t *testing.T) {
	tests := []struct {
		name          string
		workflowID    string
		setupWorkflow func(string) error
		expectError   bool
		errorContains string
	}{
		{
			name:       "compile with workflow ID successfully resolves to .md file",
			workflowID: "test-workflow",
			setupWorkflow: func(tmpDir string) error {
				workflowContent := `---
name: Test Workflow
on:
  push:
    branches: [main]
permissions:
  contents: read
---

# Test Workflow

This is a test workflow for compilation.
`
				workflowsDir := filepath.Join(tmpDir, constants.GetWorkflowDir())
				err := os.MkdirAll(workflowsDir, 0755)
				if err != nil {
					return err
				}

				workflowFile := filepath.Join(workflowsDir, "test-workflow.md")
				return os.WriteFile(workflowFile, []byte(workflowContent), 0644)
			},
			expectError: false,
		},
		{
			name:       "compile with nonexistent workflow ID returns error",
			workflowID: "nonexistent",
			setupWorkflow: func(tmpDir string) error {
				// Create workflows directory but no file
				workflowsDir := filepath.Join(tmpDir, constants.GetWorkflowDir())
				return os.MkdirAll(workflowsDir, 0755)
			},
			expectError:   true,
			errorContains: "workflow 'nonexistent' not found",
		},
		{
			name:       "compile with full path still works (backward compatibility)",
			workflowID: ".github/workflows/test-workflow.md",
			setupWorkflow: func(tmpDir string) error {
				workflowContent := `---
name: Test Workflow
on:
  push:
    branches: [main]
permissions:
  contents: read
---

# Test Workflow

This is a test workflow for backward compatibility.
`
				workflowsDir := filepath.Join(tmpDir, constants.GetWorkflowDir())
				err := os.MkdirAll(workflowsDir, 0755)
				if err != nil {
					return err
				}

				workflowFile := filepath.Join(workflowsDir, "test-workflow.md")
				return os.WriteFile(workflowFile, []byte(workflowContent), 0644)
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory
			tmpDir := t.TempDir()

			// Initialize git repository in tmp directory
			if err := initTestGitRepo(tmpDir); err != nil {
				t.Fatalf("Failed to initialize git repo: %v", err)
			}

			// Change to temporary directory
			oldDir, err := os.Getwd()
			if err != nil {
				t.Fatalf("Failed to get current directory: %v", err)
			}
			defer func() {
				if err := os.Chdir(oldDir); err != nil {
					t.Errorf("Failed to restore directory: %v", err)
				}
			}()

			if err := os.Chdir(tmpDir); err != nil {
				t.Fatalf("Failed to change to temp directory: %v", err)
			}

			// Setup workflow file
			if err := tt.setupWorkflow(tmpDir); err != nil {
				t.Fatalf("Failed to setup workflow: %v", err)
			}

			// Test CompileWorkflows function with workflow ID
			var args []string
			if tt.workflowID != "" {
				args = []string{tt.workflowID}
			}
			config := CompileConfig{
				MarkdownFiles:     args,
				Verbose:           false,
				EngineOverride:    "",
				Validate:          false,
				Watch:             false,
				WorkflowDir:       "",
				SkipInstructions:  false,
				NoEmit:            false,
				Purge:             false,
				TrialMode:         false,
				SimulatedRepoSlug: "",
			}
			_, err = CompileWorkflows(config)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("Expected error to contain '%s', but got: %v", tt.errorContains, err)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}

				// Verify the lock file was created
				expectedLockFile := filepath.Join(constants.GetWorkflowDir(), "test-workflow.lock.yml")
				if _, err := os.Stat(expectedLockFile); os.IsNotExist(err) {
					t.Errorf("Expected lock file %s to be created", expectedLockFile)
				}
			}
		})
	}
}
