//go:build !integration

package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/testutil"
)

// TestPullRequestTargetValidation tests security validation for the pull_request_target trigger.
func TestPullRequestTargetValidation(t *testing.T) {
	tmpDir := testutil.TempDir(t, "prt-validation-test")

	tests := []struct {
		name          string
		frontmatter   string
		filename      string
		strictMode    bool
		expectError   bool
		expectWarning bool
		errorContains string
		warningCount  int
	}{
		// ---- non-strict mode ----

		{
			name: "pull_request_target with checkout disabled - non-strict - sandbox warning only",
			frontmatter: `---
strict: false
on:
  pull_request_target:
    types: [opened]
tools:
  github: false
sandbox:
  agent: false
checkout: false
---

# PR Target Workflow
Test workflow content.`,
			filename:      "prt-checkout-false-non-strict.md",
			strictMode:    false,
			expectError:   false,
			expectWarning: true,
			warningCount:  1, // sandbox.agent: false
		},
		{
			name: "pull_request_target with checkout enabled - non-strict - should warn",
			frontmatter: `---
strict: false
on:
  pull_request_target:
    types: [opened]
tools:
  github: false
sandbox:
  agent: false
---

# PR Target Workflow
Test workflow content.`,
			filename:      "prt-checkout-enabled-non-strict.md",
			strictMode:    false,
			expectError:   false,
			expectWarning: true,
			warningCount:  2, // 1 for insecure checkout + 1 for sandbox.agent: false
		},
		{
			name: "pull_request trigger (not target) - non-strict - no diagnostic",
			frontmatter: `---
strict: false
on:
  pull_request:
    types: [opened]
tools:
  github: false
sandbox:
  agent: false
---

# PR Workflow
Test workflow content.`,
			filename:      "pr-non-strict.md",
			strictMode:    false,
			expectError:   false,
			expectWarning: true,
			warningCount:  1, // sandbox.agent: false only
		},
		{
			name: "push trigger - non-strict - no diagnostic",
			frontmatter: `---
strict: false
on:
  push:
    branches: [main]
tools:
  github: false
sandbox:
  agent: false
---

# Push Workflow
Test workflow content.`,
			filename:      "push-non-strict.md",
			strictMode:    false,
			expectError:   false,
			expectWarning: true,
			warningCount:  1, // sandbox.agent: false only
		},

		// ---- strict mode ----

		{
			name: "pull_request_target with checkout disabled - strict - dangerous-trigger warning",
			frontmatter: `---
on:
  pull_request_target:
    types: [opened]
tools:
  github:
    toolsets: [pull_requests]
permissions:
  pull-requests: read
checkout: false
---

# PR Target Strict Workflow
Test workflow content.`,
			filename:      "prt-checkout-false-strict.md",
			strictMode:    true,
			expectError:   false,
			expectWarning: true,
			warningCount:  1, // dangerous-trigger warning
		},
		{
			name: "pull_request_target with checkout enabled - strict - error (extremely insecure)",
			frontmatter: `---
on:
  pull_request_target:
    types: [opened]
tools:
  github:
    toolsets: [pull_requests]
permissions:
  pull-requests: read
---

# PR Target Strict No Checkout
Test workflow content.`,
			filename:      "prt-checkout-enabled-strict.md",
			strictMode:    true,
			expectError:   true,
			expectWarning: true, // dangerous-trigger warning is still emitted before the error
			errorContains: "pull_request_target trigger with checkout enabled is extremely insecure",
			warningCount:  1, // dangerous-trigger warning
		},
		{
			name: "pull_request_target with checkout enabled - strict CLI + frontmatter strict false - warning only",
			frontmatter: `---
strict: false
on:
  pull_request_target:
    types: [opened]
tools:
  github:
    toolsets: [pull_requests]
permissions:
  pull-requests: read
---

# PR Target Strict Opt-Out
Test workflow content.`,
			filename:      "prt-checkout-enabled-strict-frontmatter-opt-out.md",
			strictMode:    true,
			expectError:   false,
			expectWarning: true,
			warningCount:  1, // insecure-checkout warning only
		},
		{
			name: "pull_request trigger (not target) - strict - no diagnostic",
			frontmatter: `---
on:
  pull_request:
    types: [opened]
tools:
  github:
    toolsets: [pull_requests]
permissions:
  pull-requests: read
---

# PR Strict Workflow
Test workflow content.`,
			filename:      "pr-strict.md",
			strictMode:    true,
			expectError:   false,
			expectWarning: false,
			warningCount:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mdFile := filepath.Join(tmpDir, tt.filename)
			if err := os.WriteFile(mdFile, []byte(tt.frontmatter), 0644); err != nil {
				t.Fatal(err)
			}

			compiler := NewCompiler()
			compiler.SetStrictMode(tt.strictMode)
			compiler.SetNoEmit(true)

			err := compiler.CompileWorkflow(mdFile)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("Expected error to contain %q but got: %v", tt.errorContains, err)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}

			if compiler.GetWarningCount() != tt.warningCount {
				t.Errorf("Expected %d warnings but got %d", tt.warningCount, compiler.GetWarningCount())
			}
		})
	}
}
