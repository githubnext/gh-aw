package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestCommentEnvVarsOnlyWithReaction verifies that GH_AW_COMMENT_ID and GH_AW_COMMENT_REPO
// environment variables are only added to safe output jobs when a reaction is configured
func TestCommentEnvVarsOnlyWithReaction(t *testing.T) {
	tests := []struct {
		name              string
		markdown          string
		expectCommentEnvs bool
		safeOutputType    string
	}{
		{
			name: "create-pull-request with reaction should have comment env vars",
			markdown: `---
on:
  pull_request:
    types: [opened]
  reaction: rocket
safe-outputs:
  create-pull-request:
---

# Test PR with reaction

Should have comment env vars.
`,
			expectCommentEnvs: true,
			safeOutputType:    "create_pull_request",
		},
		{
			name: "create-pull-request without reaction should not have comment env vars",
			markdown: `---
on:
  pull_request:
    types: [opened]
safe-outputs:
  create-pull-request:
---

# Test PR without reaction

Should NOT have comment env vars.
`,
			expectCommentEnvs: false,
			safeOutputType:    "create_pull_request",
		},
		{
			name: "push-to-pull-request-branch with reaction should have comment env vars",
			markdown: `---
on:
  pull_request:
    types: [opened]
  reaction: eyes
safe-outputs:
  push-to-pull-request-branch:
---

# Test push with reaction

Should have comment env vars.
`,
			expectCommentEnvs: true,
			safeOutputType:    "push_to_pull_request_branch",
		},
		{
			name: "push-to-pull-request-branch without reaction should not have comment env vars",
			markdown: `---
on:
  pull_request:
    types: [opened]
safe-outputs:
  push-to-pull-request-branch:
---

# Test push without reaction

Should NOT have comment env vars.
`,
			expectCommentEnvs: false,
			safeOutputType:    "push_to_pull_request_branch",
		},
		{
			name: "create-pull-request with reaction:none should not have comment env vars",
			markdown: `---
on:
  pull_request:
    types: [opened]
  reaction: none
safe-outputs:
  create-pull-request:
---

# Test PR with reaction:none

Should NOT have comment env vars.
`,
			expectCommentEnvs: false,
			safeOutputType:    "create_pull_request",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a temporary directory for the test
			tmpDir := t.TempDir()

			// Write the test file
			mdFile := filepath.Join(tmpDir, "test-workflow.md")
			if err := os.WriteFile(mdFile, []byte(tt.markdown), 0644); err != nil {
				t.Fatalf("Failed to write test markdown file: %v", err)
			}

			// Create compiler and compile the workflow
			compiler := NewCompiler(false, "", "test")

			if err := compiler.CompileWorkflow(mdFile); err != nil {
				t.Fatalf("Failed to compile workflow: %v", err)
			}

			// Read the generated .lock.yml file
			lockFile := strings.TrimSuffix(mdFile, ".md") + ".lock.yml"
			lockContent, err := os.ReadFile(lockFile)
			if err != nil {
				t.Fatalf("Failed to read lock file: %v", err)
			}

			lockContentStr := string(lockContent)

			// Verify that the safe output job is generated
			if !strings.Contains(lockContentStr, tt.safeOutputType+":") {
				t.Errorf("Generated workflow should contain %s job", tt.safeOutputType)
			}

			// Check for comment environment variables
			hasCommentID := strings.Contains(lockContentStr, "GH_AW_COMMENT_ID: ${{ needs.activation.outputs.comment_id }}")
			hasCommentRepo := strings.Contains(lockContentStr, "GH_AW_COMMENT_REPO: ${{ needs.activation.outputs.comment_repo }}")

			if tt.expectCommentEnvs {
				if !hasCommentID {
					t.Errorf("Expected GH_AW_COMMENT_ID environment variable but it was not found")
				}
				if !hasCommentRepo {
					t.Errorf("Expected GH_AW_COMMENT_REPO environment variable but it was not found")
				}
			} else {
				if hasCommentID {
					t.Errorf("Did NOT expect GH_AW_COMMENT_ID environment variable but it was found")
				}
				if hasCommentRepo {
					t.Errorf("Did NOT expect GH_AW_COMMENT_REPO environment variable but it was found")
				}
			}
		})
	}
}

// TestActivationJobOutputsWithReaction verifies that activation job outputs
// include comment_id and comment_repo when a reaction is configured
func TestActivationJobOutputsWithReaction(t *testing.T) {
	tests := []struct {
		name                 string
		markdown             string
		expectCommentOutputs bool
	}{
		{
			name: "activation with reaction should have comment outputs",
			markdown: `---
on:
  issues:
    types: [opened]
  reaction: rocket
---

# Test with reaction

Should have comment outputs in activation job.
`,
			expectCommentOutputs: true,
		},
		{
			name: "activation without reaction should not have comment outputs",
			markdown: `---
on:
  issues:
    types: [opened]
---

# Test without reaction

Should NOT have comment outputs in activation job.
`,
			expectCommentOutputs: false,
		},
		{
			name: "activation with reaction:none should not have comment outputs",
			markdown: `---
on:
  issues:
    types: [opened]
  reaction: none
---

# Test with reaction:none

Should NOT have comment outputs in activation job.
`,
			expectCommentOutputs: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a temporary directory for the test
			tmpDir := t.TempDir()

			// Write the test file
			mdFile := filepath.Join(tmpDir, "test-workflow.md")
			if err := os.WriteFile(mdFile, []byte(tt.markdown), 0644); err != nil {
				t.Fatalf("Failed to write test markdown file: %v", err)
			}

			// Create compiler and compile the workflow
			compiler := NewCompiler(false, "", "test")

			if err := compiler.CompileWorkflow(mdFile); err != nil {
				t.Fatalf("Failed to compile workflow: %v", err)
			}

			// Read the generated .lock.yml file
			lockFile := strings.TrimSuffix(mdFile, ".md") + ".lock.yml"
			lockContent, err := os.ReadFile(lockFile)
			if err != nil {
				t.Fatalf("Failed to read lock file: %v", err)
			}

			lockContentStr := string(lockContent)

			// Check for activation job outputs
			hasCommentIDOutput := strings.Contains(lockContentStr, "comment_id: ${{ steps.react.outputs.comment-id }}")
			hasCommentRepoOutput := strings.Contains(lockContentStr, "comment_repo: ${{ steps.react.outputs.comment-repo }}")

			if tt.expectCommentOutputs {
				if !hasCommentIDOutput {
					t.Errorf("Expected comment_id output in activation job but it was not found")
				}
				if !hasCommentRepoOutput {
					t.Errorf("Expected comment_repo output in activation job but it was not found")
				}
			} else {
				if hasCommentIDOutput {
					t.Errorf("Did NOT expect comment_id output in activation job but it was found")
				}
				if hasCommentRepoOutput {
					t.Errorf("Did NOT expect comment_repo output in activation job but it was found")
				}
			}
		})
	}
}
