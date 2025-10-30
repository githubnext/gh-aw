package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCheckoutOptimization(t *testing.T) {
	tests := []struct {
		name                string
		frontmatter         string
		expectedHasCheckout bool
		description         string
	}{
		{
			name: "no permissions defaults to read-all should include checkout",
			frontmatter: `---
on:
  issues:
    types: [opened]
tools:
  github:
    toolsets: [issues]
engine: claude
---`,
			expectedHasCheckout: true,
			description:         "When no permissions are specified, default read-all grants checkout",
		},
		{
			name: "permissions without contents should omit checkout",
			frontmatter: `---
on:
  issues:
    types: [opened]
permissions:
  issues: write
  pull-requests: read
tools:
  github:
    toolsets: [issues, pull_requests]
engine: claude
---`,
			expectedHasCheckout: false,
			description:         "When permissions don't include contents, checkout should be omitted",
		},
		{
			name: "permissions with contents read should include checkout",
			frontmatter: `---
on:
  issues:
    types: [opened]
permissions:
  contents: read
  issues: write
  pull-requests: read
tools:
  github:
    toolsets: [repos, issues, pull_requests]
engine: claude
---`,
			expectedHasCheckout: true,
			description:         "When permissions include contents: read, checkout should be included",
		},
		{
			name: "permissions with contents write should include checkout",
			frontmatter: `---
on:
  issues:
    types: [opened]
permissions:
  contents: write
  issues: write
  pull-requests: read
tools:
  github:
    toolsets: [repos, issues, pull_requests]
engine: claude
---`,
			expectedHasCheckout: true,
			description:         "When permissions include contents: write, checkout should be included",
		},
		{
			name: "shorthand read-all should include checkout",
			frontmatter: `---
on:
  issues:
    types: [opened]
permissions: read-all
tools:
  github:
    toolsets: [issues]
engine: claude
---`,
			expectedHasCheckout: true,
			description:         "When permissions is read-all, checkout should be included",
		},
		{
			name: "shorthand write-all should include checkout",
			frontmatter: `---
on:
  issues:
    types: [opened]
permissions: write-all
tools:
  github:
    toolsets: [issues]
engine: claude
---`,
			expectedHasCheckout: true,
			description:         "When permissions is write-all, checkout should be included",
		},
		{
			name: "custom steps with checkout should omit default checkout",
			frontmatter: `---
on:
  issues:
    types: [opened]
permissions:
  contents: read
  issues: write
  pull-requests: read
steps:
  - name: Custom checkout
    uses: actions/checkout@08c6903cd8c0fde910a37f88322edcfb5dd907a8
    with:
      token: ${{ secrets.CUSTOM_TOKEN }}
  - name: Setup
    run: echo "custom setup"
tools:
  github:
    toolsets: [issues]
engine: claude
---`,
			expectedHasCheckout: false,
			description:         "When custom steps already contain checkout, default checkout should be omitted",
		},
		{
			name: "custom steps without checkout but with contents permission should include checkout",
			frontmatter: `---
on:
  issues:
    types: [opened]
permissions:
  contents: read
  issues: write
  pull-requests: read
steps:
  - name: Setup Node
    uses: actions/setup-node@v4
    with:
      node-version: '18'
  - name: Install deps
    run: npm install
tools:
  github:
    toolsets: [issues]
engine: claude
---`,
			expectedHasCheckout: true,
			description:         "When custom steps don't contain checkout but have contents permission, checkout should be included",
		},
		{
			name: "custom steps without checkout and no contents permission should omit checkout",
			frontmatter: `---
on:
  issues:
    types: [opened]
permissions:
  issues: write
  pull-requests: read
steps:
  - name: Setup Node
    uses: actions/setup-node@v4
    with:
      node-version: '18'
  - name: Install deps
    run: npm install
tools:
  github:
    toolsets: [issues, pull_requests]
engine: claude
---`,
			expectedHasCheckout: false,
			description:         "When custom steps don't contain checkout and no contents permission, checkout should be omitted",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir, err := os.MkdirTemp("", "checkout-optimization-test")
			if err != nil {
				t.Fatal(err)
			}
			defer os.RemoveAll(tmpDir)

			// Create test workflow file
			testContent := tt.frontmatter + "\n\n# Test Workflow\n\nThis is a test workflow to check checkout optimization.\n"
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

			// For the test case with custom checkout, we need to verify that
			// only the custom checkout is present, not a default generated one
			if tt.name == "custom steps with checkout should omit default checkout" {
				// Check that the custom checkout with token is present
				hasCustomCheckout := strings.Contains(lockContentStr, "token: ${{ secrets.CUSTOM_TOKEN }}")
				// Check that there's no "Checkout repository" step (which is the default name)
				hasDefaultCheckout := strings.Contains(lockContentStr, "name: Checkout repository")

				if !hasCustomCheckout {
					t.Errorf("%s: Custom checkout with token not found", tt.description)
				}
				if hasDefaultCheckout {
					t.Errorf("%s: Default checkout step should not be present when custom steps have checkout", tt.description)
				}
			} else {
				// For other test cases, check if checkout step is present
				hasCheckout := strings.Contains(lockContentStr, "actions/checkout@08c6903cd8c0fde910a37f88322edcfb5dd907a8")

				if hasCheckout != tt.expectedHasCheckout {
					t.Errorf("%s: Expected hasCheckout=%v, got %v\nLock file content:\n%s",
						tt.description, tt.expectedHasCheckout, hasCheckout, lockContentStr)
				}
			}
		})
	}
}

func TestShouldAddCheckoutStep(t *testing.T) {
	tests := []struct {
		name        string
		permissions string
		customSteps string
		expected    bool
	}{
		{
			name:        "default permissions should include checkout",
			permissions: "permissions: read-all", // Default applied by compiler
			customSteps: "",
			expected:    true,
		},
		{
			name:        "contents read permission specified, no custom steps",
			permissions: "permissions:\n  contents: read",
			customSteps: "",
			expected:    true,
		},
		{
			name:        "contents write permission specified, no custom steps",
			permissions: "permissions:\n  contents: write",
			customSteps: "",
			expected:    true,
		},
		{
			name:        "no contents permission specified, no custom steps",
			permissions: "permissions:\n  issues: write",
			customSteps: "",
			expected:    false,
		},
		{
			name:        "contents read permission, custom steps with checkout",
			permissions: "permissions:\n  contents: read",
			customSteps: "steps:\n  - uses: actions/checkout@08c6903cd8c0fde910a37f88322edcfb5dd907a8",
			expected:    false,
		},
		{
			name:        "contents read permission, custom steps without checkout",
			permissions: "permissions:\n  contents: read",
			customSteps: "steps:\n  - uses: actions/setup-node@v4",
			expected:    true,
		},
		{
			name:        "read-all shorthand permission specified",
			permissions: "permissions: read-all",
			customSteps: "",
			expected:    true,
		},
		{
			name:        "write-all shorthand permission specified",
			permissions: "permissions: write-all",
			customSteps: "",
			expected:    true,
		},
	}

	compiler := NewCompiler(false, "", "test")

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := &WorkflowData{
				Permissions: tt.permissions,
				CustomSteps: tt.customSteps,
			}

			result := compiler.shouldAddCheckoutStep(data)
			if result != tt.expected {
				t.Errorf("shouldAddCheckoutStep() = %v, expected %v", result, tt.expected)
			}
		})
	}
}
