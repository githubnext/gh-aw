//go:build !integration

package workflow

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/testutil"
)

// TestPushBranchScopeStrictVsNonStrict verifies that an unscoped push trigger
// (missing branch/tag ref filters) is an error in strict mode and a warning in non-strict mode.
func TestPushBranchScopeStrictVsNonStrict(t *testing.T) {
	tests := []struct {
		name          string
		content       string
		expectError   bool
		expectWarning bool
		warningText   string
	}{
		{
			name: "unscoped push in strict mode (default) is an error",
			content: `---
name: Unscoped Push Strict
on:
  push:
    paths:
      - src/**
---

# Test Workflow
`,
			expectError:   true,
			expectWarning: false,
		},
		{
			name: "unscoped push in non-strict mode is a warning",
			content: `---
name: Unscoped Push Non-Strict
strict: false
on:
  push:
    paths:
      - src/**
---

# Test Workflow
`,
			expectError:   false,
			expectWarning: true,
			warningText:   "branches",
		},
		{
			name: "scoped push with branches in strict mode is allowed",
			content: `---
name: Scoped Push Strict
on:
  push:
    branches:
      - main
---

# Test Workflow
`,
			expectError:   false,
			expectWarning: false,
		},
		{
			name: "scoped push with tags in strict mode is allowed",
			content: `---
name: Tag Scoped Push Strict
on:
  push:
    tags:
      - 'v*.*.*'
---

# Test Workflow
`,
			expectError:   false,
			expectWarning: false,
		},
		{
			name: "empty push map in non-strict mode is a warning",
			content: `---
name: Empty Push Non-Strict
strict: false
on:
  push: {}
---

# Test Workflow
`,
			expectError:   false,
			expectWarning: true,
			warningText:   "branches",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := testutil.TempDir(t, "push-branch-scope-*")
			testFile := filepath.Join(tmpDir, "test-workflow.md")
			if err := os.WriteFile(testFile, []byte(tt.content), 0644); err != nil {
				t.Fatal(err)
			}

			// Capture stderr to check for warnings
			oldStderr := os.Stderr
			r, w, _ := os.Pipe()
			os.Stderr = w

			compiler := NewCompiler()
			err := compiler.CompileWorkflow(testFile)

			// Restore stderr
			w.Close()
			os.Stderr = oldStderr
			var buf bytes.Buffer
			io.Copy(&buf, r) //nolint:errcheck
			stderrOutput := buf.String()

			if tt.expectError {
				if err == nil {
					t.Fatal("expected compilation to fail but got no error")
				}
				return
			}

			if err != nil {
				t.Fatalf("expected compilation to succeed but got error: %v", err)
			}

			if tt.expectWarning {
				if !strings.Contains(stderrOutput, tt.warningText) {
					t.Errorf("expected warning containing %q\ngot stderr:\n%s", tt.warningText, stderrOutput)
				}
			} else {
				if strings.Contains(stderrOutput, "push event must specify") {
					t.Errorf("unexpected push branch scope warning in stderr:\n%s", stderrOutput)
				}
			}
		})
	}
}
