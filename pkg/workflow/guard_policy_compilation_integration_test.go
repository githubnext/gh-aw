//go:build integration

package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/stringutil"
	"github.com/github/gh-aw/pkg/testutil"
)

// TestGuardPolicyYAMLCompilationIntegration tests end-to-end guard-policy YAML compilation.
// Each test case compiles a workflow frontmatter and verifies that the resulting lock YAML
// contains the expected guard-policy fragments.
func TestGuardPolicyYAMLCompilationIntegration(t *testing.T) {
	tests := []struct {
		name           string
		workflowMD     string
		expectedInYAML []string
		notInYAML      []string
		description    string
	}{
		{
			name: "allowed-repos all produces write-sink accept star",
			workflowMD: `---
on: workflow_dispatch
engine: copilot
permissions:
  contents: read
tools:
  github:
    repos: all
    min-integrity: unapproved
---

# Test guard policy — repos all
`,
			expectedInYAML: []string{
				`"guard-policies"`,
				`"write-sink"`,
				`"accept"`,
				`"*"`,
			},
			description: "repos=all should produce a write-sink guard policy with accept=[\"*\"]",
		},
		{
			name: "allowed-repos public produces write-sink accept star",
			workflowMD: `---
on: workflow_dispatch
engine: copilot
permissions:
  contents: read
tools:
  github:
    repos: public
    min-integrity: none
---

# Test guard policy — repos public
`,
			expectedInYAML: []string{
				`"guard-policies"`,
				`"write-sink"`,
				`"accept"`,
				`"*"`,
			},
			description: "repos=public should produce a write-sink guard policy with accept=[\"*\"]",
		},
		{
			name: "single specific repo as array produces private-scoped write-sink",
			workflowMD: `---
on: workflow_dispatch
engine: copilot
permissions:
  contents: read
tools:
  github:
    repos:
      - github/gh-aw
    min-integrity: approved
---

# Test guard policy — specific repo
`,
			expectedInYAML: []string{
				`"guard-policies"`,
				`"write-sink"`,
				`"accept"`,
				`"private:github/gh-aw"`,
			},
			description: "Single specific repo (as array) should produce write-sink with private:owner/repo",
		},
		{
			name: "owner wildcard repo produces stripped private-scoped write-sink",
			workflowMD: `---
on: workflow_dispatch
engine: copilot
permissions:
  contents: read
tools:
  github:
    repos:
      - myorg/*
    min-integrity: approved
---

# Test guard policy — owner wildcard
`,
			expectedInYAML: []string{
				`"guard-policies"`,
				`"write-sink"`,
				`"accept"`,
				`"private:myorg"`,
			},
			notInYAML: []string{
				`"private:myorg/*"`,
			},
			description: "Owner wildcard (myorg/*) should produce private:myorg (/* stripped)",
		},
		{
			name: "multiple repos produce multiple private-scoped accept entries",
			workflowMD: `---
on: workflow_dispatch
engine: copilot
permissions:
  contents: read
tools:
  github:
    repos:
      - github/gh-aw
      - github/copilot-cli
    min-integrity: merged
---

# Test guard policy — multiple repos
`,
			expectedInYAML: []string{
				`"guard-policies"`,
				`"write-sink"`,
				`"accept"`,
				`"private:github/gh-aw"`,
				`"private:github/copilot-cli"`,
			},
			description: "Multiple repos should produce multiple private: accept entries in write-sink",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := testutil.TempDir(t, "guard-policy-compilation-test-*")
			workflowPath := filepath.Join(tmpDir, "test-workflow.md")
			if err := os.WriteFile(workflowPath, []byte(tt.workflowMD), 0o644); err != nil {
				t.Fatalf("Failed to write workflow file: %v", err)
			}

			compiler := NewCompiler()
			if err := compiler.CompileWorkflow(workflowPath); err != nil {
				t.Fatalf("Failed to compile workflow (%s): %v", tt.description, err)
			}

			lockPath := stringutil.MarkdownToLockFile(workflowPath)
			lockContent, err := os.ReadFile(lockPath)
			if err != nil {
				t.Fatalf("Failed to read compiled lock file: %v", err)
			}
			yaml := string(lockContent)

			for _, expected := range tt.expectedInYAML {
				if !strings.Contains(yaml, expected) {
					t.Errorf("%s: expected lock YAML to contain %q\nFull YAML:\n%s", tt.description, expected, yaml)
				}
			}
			for _, notExpected := range tt.notInYAML {
				if strings.Contains(yaml, notExpected) {
					t.Errorf("%s: expected lock YAML NOT to contain %q\nFull YAML:\n%s", tt.description, notExpected, yaml)
				}
			}
		})
	}
}
