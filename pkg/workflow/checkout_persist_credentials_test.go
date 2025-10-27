package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCheckoutPersistCredentials(t *testing.T) {
	tests := []struct {
		name        string
		frontmatter string
		description string
	}{
		{
			name: "main job checkout includes persist-credentials false",
			frontmatter: `---
on:
  issues:
    types: [opened]
permissions:
  contents: read
  issues: write
tools:
  github:
    allowed: [list_issues]
engine: claude
---`,
			description: "Main job checkout step should include persist-credentials: false",
		},
		{
			name: "safe output create-issue checkout includes persist-credentials false",
			frontmatter: `---
on:
  issues:
    types: [opened]
permissions:
  contents: read
  actions: read
safe-outputs:
  create-issue:
    assignees: [user1]
engine: claude
---`,
			description: "Create issue job checkout should include persist-credentials: false",
		},
		{
			name: "safe output create-pull-request checkout includes persist-credentials false",
			frontmatter: `---
on:
  push:
    branches: [main]
permissions:
  contents: read
  actions: read
safe-outputs:
  create-pull-request:
engine: claude
---`,
			description: "Create pull request job checkout should include persist-credentials: false",
		},
		{
			name: "safe output push-to-pull-request-branch checkout includes persist-credentials false",
			frontmatter: `---
on:
  pull_request:
    types: [opened]
permissions:
  contents: read
  actions: read
safe-outputs:
  push-to-pull-request-branch:
engine: claude
---`,
			description: "Push to PR branch job checkout should include persist-credentials: false",
		},
		{
			name: "safe output create-agent-task checkout includes persist-credentials false",
			frontmatter: `---
on:
  issues:
    types: [opened]
permissions:
  contents: read
  actions: read
safe-outputs:
  create-agent-task:
engine: claude
---`,
			description: "Create agent task job checkout should include persist-credentials: false",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir, err := os.MkdirTemp("", "checkout-persist-credentials-test")
			if err != nil {
				t.Fatal(err)
			}
			defer os.RemoveAll(tmpDir)

			// Create test workflow file
			testContent := tt.frontmatter + "\n\n# Test Workflow\n\nThis is a test workflow to check persist-credentials.\n"
			testFile := filepath.Join(tmpDir, "test-workflow.md")
			if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
				t.Fatal(err)
			}

			compiler := NewCompiler(false, "", "test")

			// Compile the workflow
			err = compiler.CompileWorkflow(testFile)
			if err != nil {
				t.Fatalf("Failed to compile workflow: %v", err)
			}

			// Calculate the lock file path
			lockFile := strings.TrimSuffix(testFile, ".md") + ".lock.yml"

			// Read the generated lock file
			lockContent, err := os.ReadFile(lockFile)
			if err != nil {
				t.Fatalf("Failed to read lock file: %v", err)
			}

			lockContentStr := string(lockContent)

			// Check that all checkout steps have persist-credentials: false
			// We need to find all occurrences of actions/checkout and verify each has persist-credentials
			checkoutLines := []string{}
			lines := strings.Split(lockContentStr, "\n")
			for i, line := range lines {
				if strings.Contains(line, "actions/checkout@") {
					// Collect the next several lines to check for persist-credentials
					context := ""
					for j := i; j < len(lines) && j < i+10; j++ {
						context += lines[j] + "\n"
						if strings.TrimSpace(lines[j]) != "" && !strings.HasPrefix(strings.TrimSpace(lines[j]), "-") && j > i {
							// Stop if we hit a non-indented line or a new step
							if !strings.HasPrefix(lines[j], "      ") && !strings.HasPrefix(lines[j], "        ") {
								break
							}
						}
					}
					checkoutLines = append(checkoutLines, context)
				}
			}

			if len(checkoutLines) == 0 {
				t.Logf("Note: No checkout steps found in workflow, which may be expected for some configurations")
				return
			}

			// Verify each checkout has persist-credentials: false
			for idx, checkoutContext := range checkoutLines {
				if !strings.Contains(checkoutContext, "persist-credentials: false") {
					t.Errorf("%s: Checkout #%d missing persist-credentials: false\nContext:\n%s\nFull workflow:\n%s",
						tt.description, idx+1, checkoutContext, lockContentStr)
				}
			}
		})
	}
}
