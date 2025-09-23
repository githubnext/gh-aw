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
    allowed: [list_issues]
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
    allowed: [list_issues]
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
tools:
  github:
    allowed: [list_issues]
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
tools:
  github:
    allowed: [list_issues]
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
    allowed: [list_issues]
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
    allowed: [list_issues]
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
steps:
  - name: Custom checkout
    uses: actions/checkout@v4
    with:
      token: ${{ secrets.CUSTOM_TOKEN }}
  - name: Setup
    run: echo "custom setup"
tools:
  github:
    allowed: [list_issues]
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
steps:
  - name: Setup Node
    uses: actions/setup-node@v4
    with:
      node-version: '18'
  - name: Install deps
    run: npm install
tools:
  github:
    allowed: [list_issues]
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
    allowed: [list_issues]
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

			// Check if checkout step is present
			hasCheckout := strings.Contains(lockContentStr, "actions/checkout@v5")

			if hasCheckout != tt.expectedHasCheckout {
				t.Errorf("%s: Expected hasCheckout=%v, got %v\nLock file content:\n%s",
					tt.description, tt.expectedHasCheckout, hasCheckout, lockContentStr)
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
			customSteps: "steps:\n  - uses: actions/checkout@v5",
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
