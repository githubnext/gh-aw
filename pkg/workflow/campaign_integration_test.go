package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCampaignIntegration(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name               string
		workflowContent    string
		shouldCompile      bool
		shouldHaveEnvVar   bool
		shouldHaveInScript bool
		expectedCampaign   string
	}{
		{
			name: "Workflow with valid campaign",
			workflowContent: `---
on: workflow_dispatch
permissions:
  contents: read
campaign: test-fp-12345
safe-outputs:
  create-issue:
---

# Test Campaign

Create a test issue.
`,
			shouldCompile:      true,
			shouldHaveEnvVar:   true,
			shouldHaveInScript: true,
			expectedCampaign:   "test-fp-12345",
		},
		{
			name: "Workflow without campaign",
			workflowContent: `---
on: workflow_dispatch
permissions:
  contents: read
safe-outputs:
  create-issue:
---

# Test No Campaign

Create a test issue without campaign.
`,
			shouldCompile:      true,
			shouldHaveEnvVar:   false,
			shouldHaveInScript: false,
		},
		{
			name: "Workflow with campaign in discussion",
			workflowContent: `---
on: workflow_dispatch
permissions:
  contents: read
campaign: discussion_fp_001
safe-outputs:
  create-discussion:
---

# Test Discussion Campaign

Create a discussion.
`,
			shouldCompile:      true,
			shouldHaveEnvVar:   true,
			shouldHaveInScript: true,
			expectedCampaign:   "discussion_fp_001",
		},
		{
			name: "Workflow with campaign in comment",
			workflowContent: `---
on:
  issues:
    types: [opened]
permissions:
  contents: read
campaign: comment_fp_2024
safe-outputs:
  add-comment:
---

# Test Comment Campaign

Add a comment.
`,
			shouldCompile:      true,
			shouldHaveEnvVar:   true,
			shouldHaveInScript: true,
			expectedCampaign:   "comment_fp_2024",
		},
		{
			name: "Workflow with campaign in pull request",
			workflowContent: `---
on: push
permissions:
  contents: read
campaign: pr-campaign-123
safe-outputs:
  create-pull-request:
---

# Test PR Campaign

Create a pull request.
`,
			shouldCompile:      true,
			shouldHaveEnvVar:   true,
			shouldHaveInScript: true,
			expectedCampaign:   "pr-campaign-123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			workflowFile := filepath.Join(tmpDir, "test.md")
			err := os.WriteFile(workflowFile, []byte(tt.workflowContent), 0644)
			if err != nil {
				t.Fatalf("Failed to write test workflow: %v", err)
			}

			compiler := NewCompiler(false, "", "test")
			compiler.verbose = false

			err = compiler.CompileWorkflow(workflowFile)

			if tt.shouldCompile && err != nil {
				t.Fatalf("Expected compilation to succeed, got error: %v", err)
			}
			if !tt.shouldCompile && err == nil {
				t.Fatal("Expected compilation to fail, but it succeeded")
			}

			if tt.shouldCompile {
				lockFile := strings.TrimSuffix(workflowFile, ".md") + ".lock.yml"
				content, err := os.ReadFile(lockFile)
				if err != nil {
					t.Fatalf("Failed to read lock file: %v", err)
				}

				contentStr := string(content)

				if tt.shouldHaveEnvVar {
					envVarLine := "GH_AW_CAMPAIGN: \"" + tt.expectedCampaign + "\""
					if !strings.Contains(contentStr, envVarLine) {
						t.Errorf("Expected lock file to contain env var '%s', but it didn't", envVarLine)
					}
				} else {
					// The JavaScript code will always read process.env.GH_AW_CAMPAIGN
					// but the environment variable should not be set
					envVarLine := "GH_AW_CAMPAIGN: \""
					if strings.Contains(contentStr, envVarLine) {
						t.Error("Expected lock file to NOT set GH_AW_CAMPAIGN env var, but it did")
					}
				}

				if tt.shouldHaveInScript {
					// Check that fingerprint is read from environment
					if !strings.Contains(contentStr, "process.env.GH_AW_CAMPAIGN") {
						t.Error("Expected script to read GH_AW_CAMPAIGN from environment")
					}
					// Check that fingerprint is added to body/comment
					if !strings.Contains(contentStr, "<!-- campaign:") {
						t.Error("Expected script to add fingerprint HTML comment")
					}
				}

				// Clean up lock file
				os.Remove(lockFile)
			}

			// Clean up workflow file
			os.Remove(workflowFile)
		})
	}
}
