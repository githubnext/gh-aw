package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStrictModeTimeout(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		expectError bool
		errorMsg    string
	}{
		{
			name: "timeout not required in strict mode",
			content: `---
on: push
permissions:
  contents: read
engine: copilot
network:
  allowed:
    - "api.example.com"
---

# Test Workflow`,
			expectError: false,
		},
		{
			name: "timeout still valid in strict mode when specified",
			content: `---
on: push
permissions:
  contents: read
timeout_minutes: 10
engine: copilot
network:
  allowed:
    - "api.example.com"
---

# Test Workflow`,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir, err := os.MkdirTemp("", "strict-timeout-test")
			if err != nil {
				t.Fatal(err)
			}
			defer os.RemoveAll(tmpDir)

			testFile := filepath.Join(tmpDir, "test-workflow.md")
			if err := os.WriteFile(testFile, []byte(tt.content), 0644); err != nil {
				t.Fatal(err)
			}

			compiler := NewCompiler(false, "", "")
			compiler.SetStrictMode(true)
			err = compiler.CompileWorkflow(testFile)

			if tt.expectError && err == nil {
				t.Error("Expected compilation to fail but it succeeded")
			} else if !tt.expectError && err != nil {
				t.Errorf("Expected compilation to succeed but it failed: %v", err)
			} else if tt.expectError && err != nil && tt.errorMsg != "" {
				if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error containing '%s', got '%s'", tt.errorMsg, err.Error())
				}
			}
		})
	}
}

func TestStrictModePermissions(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		expectError bool
		errorMsg    string
	}{
		{
			name: "read permissions allowed in strict mode",
			content: `---
on: push
permissions:
  contents: read
  issues: read
timeout_minutes: 10
engine: copilot
---

# Test Workflow`,
			expectError: false,
		},
		{
			name: "contents write permission refused in strict mode",
			content: `---
on: push
permissions:
  contents: write
timeout_minutes: 10
engine: copilot
---

# Test Workflow`,
			expectError: true,
			errorMsg:    "strict mode: write permission 'contents: write' is not allowed",
		},
		{
			name: "issues write permission refused in strict mode",
			content: `---
on: push
permissions:
  issues: write
timeout_minutes: 10
engine: copilot
---

# Test Workflow`,
			expectError: true,
			errorMsg:    "strict mode: write permission 'issues: write' is not allowed",
		},
		{
			name: "pull-requests write permission refused in strict mode",
			content: `---
on: push
permissions:
  pull-requests: write
timeout_minutes: 10
engine: copilot
---

# Test Workflow`,
			expectError: true,
			errorMsg:    "strict mode: write permission 'pull-requests: write' is not allowed",
		},
		{
			name: "no permissions specified allowed in strict mode",
			content: `---
on: push
timeout_minutes: 10
engine: copilot
---

# Test Workflow`,
			expectError: false,
		},
		{
			name: "shorthand write permission refused in strict mode",
			content: `---
on: push
permissions: write
timeout_minutes: 10
engine: copilot
---

# Test Workflow`,
			expectError: true,
			errorMsg:    "strict mode: write permission 'contents: write' is not allowed",
		},
		{
			name: "shorthand write-all permission refused in strict mode",
			content: `---
on: push
permissions: write-all
timeout_minutes: 10
engine: copilot
---

# Test Workflow`,
			expectError: true,
			errorMsg:    "strict mode: write permission 'contents: write' is not allowed",
		},

		{
			name: "shorthand read-all permission allowed in strict mode",
			content: `---
on: push
permissions: read-all
timeout_minutes: 10
engine: copilot
network:
  allowed:
    - "api.example.com"
---

# Test Workflow`,
			expectError: false,
		},
		{
			name: "write permission with inline comment refused in strict mode",
			content: `---
on: push
permissions:
  contents: write # NOT IN STRICT MODE
timeout_minutes: 10
engine: copilot
---

# Test Workflow`,
			expectError: true,
			errorMsg:    "strict mode: write permission 'contents: write' is not allowed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir, err := os.MkdirTemp("", "strict-permissions-test")
			if err != nil {
				t.Fatal(err)
			}
			defer os.RemoveAll(tmpDir)

			testFile := filepath.Join(tmpDir, "test-workflow.md")
			if err := os.WriteFile(testFile, []byte(tt.content), 0644); err != nil {
				t.Fatal(err)
			}

			compiler := NewCompiler(false, "", "")
			compiler.SetStrictMode(true)
			err = compiler.CompileWorkflow(testFile)

			if tt.expectError && err == nil {
				t.Error("Expected compilation to fail but it succeeded")
			} else if !tt.expectError && err != nil {
				t.Errorf("Expected compilation to succeed but it failed: %v", err)
			} else if tt.expectError && err != nil && tt.errorMsg != "" {
				if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error containing '%s', got '%s'", tt.errorMsg, err.Error())
				}
			}
		})
	}
}

func TestStrictModeNetwork(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		expectError bool
		errorMsg    string
	}{
		{
			name: "defaults network allowed in strict mode",
			content: `---
on: push
permissions:
  contents: read
timeout_minutes: 10
engine: copilot
network: defaults
---

# Test Workflow`,
			expectError: false,
		},
		{
			name: "specific domains allowed in strict mode",
			content: `---
on: push
permissions:
  contents: read
timeout_minutes: 10
engine: copilot
network:
  allowed:
    - "api.example.com"
    - "*.trusted.com"
---

# Test Workflow`,
			expectError: false,
		},
		{
			name: "wildcard star refused in strict mode",
			content: `---
on: push
permissions:
  contents: read
timeout_minutes: 10
engine: copilot
network:
  allowed:
    - "*"
---

# Test Workflow`,
			expectError: true,
			errorMsg:    "strict mode: wildcard '*' is not allowed in network.allowed domains",
		},
		{
			name: "empty network object allowed in strict mode",
			content: `---
on: push
permissions:
  contents: read
timeout_minutes: 10
engine: copilot
network: {}
---

# Test Workflow`,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir, err := os.MkdirTemp("", "strict-network-test")
			if err != nil {
				t.Fatal(err)
			}
			defer os.RemoveAll(tmpDir)

			testFile := filepath.Join(tmpDir, "test-workflow.md")
			if err := os.WriteFile(testFile, []byte(tt.content), 0644); err != nil {
				t.Fatal(err)
			}

			compiler := NewCompiler(false, "", "")
			compiler.SetStrictMode(true)
			err = compiler.CompileWorkflow(testFile)

			if tt.expectError && err == nil {
				t.Error("Expected compilation to fail but it succeeded")
			} else if !tt.expectError && err != nil {
				t.Errorf("Expected compilation to succeed but it failed: %v", err)
			} else if tt.expectError && err != nil && tt.errorMsg != "" {
				if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error containing '%s', got '%s'", tt.errorMsg, err.Error())
				}
			}
		})
	}
}

func TestStrictModeMCPNetwork(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		expectError bool
		errorMsg    string
	}{
		{
			name: "built-in tools do not require network configuration",
			content: `---
on: push
permissions:
  contents: read
timeout_minutes: 10
engine: copilot
tools:
  github:
    allowed: [get_issue]
  bash:
---

# Test Workflow`,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir, err := os.MkdirTemp("", "strict-mcp-test")
			if err != nil {
				t.Fatal(err)
			}
			defer os.RemoveAll(tmpDir)

			testFile := filepath.Join(tmpDir, "test-workflow.md")
			if err := os.WriteFile(testFile, []byte(tt.content), 0644); err != nil {
				t.Fatal(err)
			}

			compiler := NewCompiler(false, "", "")
			compiler.SetStrictMode(true)
			err = compiler.CompileWorkflow(testFile)

			if tt.expectError && err == nil {
				t.Error("Expected compilation to fail but it succeeded")
			} else if !tt.expectError && err != nil {
				t.Errorf("Expected compilation to succeed but it failed: %v", err)
			} else if tt.expectError && err != nil && tt.errorMsg != "" {
				if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error containing '%s', got '%s'", tt.errorMsg, err.Error())
				}
			}
		})
	}
}

func TestStrictModeBashTools(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		expectError bool
		errorMsg    string
	}{
		{
			name: "specific bash commands allowed in strict mode",
			content: `---
on: push
permissions:
  contents: read
timeout_minutes: 10
engine: copilot
tools:
  bash: ["echo", "ls", "pwd"]
network:
  allowed:
    - "api.example.com"
---

# Test Workflow`,
			expectError: false,
		},
		{
			name: "bash null allowed in strict mode",
			content: `---
on: push
permissions:
  contents: read
timeout_minutes: 10
engine: copilot
tools:
  bash:
network:
  allowed:
    - "api.example.com"
---

# Test Workflow`,
			expectError: false,
		},
		{
			name: "bash empty array allowed in strict mode",
			content: `---
on: push
permissions:
  contents: read
timeout_minutes: 10
engine: copilot
tools:
  bash: []
network:
  allowed:
    - "api.example.com"
---

# Test Workflow`,
			expectError: false,
		},
		{
			name: "bash wildcard star refused in strict mode",
			content: `---
on: push
permissions:
  contents: read
timeout_minutes: 10
engine: copilot
tools:
  bash: ["*"]
network:
  allowed:
    - "api.example.com"
---

# Test Workflow`,
			expectError: true,
			errorMsg:    "strict mode: bash wildcard '*' is not allowed - use specific commands instead",
		},
		{
			name: "bash wildcard colon-star refused in strict mode",
			content: `---
on: push
permissions:
  contents: read
timeout_minutes: 10
engine: copilot
tools:
  bash: [":*"]
network:
  allowed:
    - "api.example.com"
---

# Test Workflow`,
			expectError: true,
			errorMsg:    "strict mode: bash wildcard ':*' is not allowed - use specific commands instead",
		},
		{
			name: "bash wildcard star mixed with commands refused in strict mode",
			content: `---
on: push
permissions:
  contents: read
timeout_minutes: 10
engine: copilot
tools:
  bash: ["echo", "ls", "*", "pwd"]
network:
  allowed:
    - "api.example.com"
---

# Test Workflow`,
			expectError: true,
			errorMsg:    "strict mode: bash wildcard '*' is not allowed - use specific commands instead",
		},
		{
			name: "bash command wildcards like git:* are allowed in strict mode",
			content: `---
on: push
permissions:
  contents: read
timeout_minutes: 10
engine: copilot
tools:
  bash: ["git:*", "npm:*"]
network:
  allowed:
    - "api.example.com"
---

# Test Workflow`,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir, err := os.MkdirTemp("", "strict-bash-test")
			if err != nil {
				t.Fatal(err)
			}
			defer os.RemoveAll(tmpDir)

			testFile := filepath.Join(tmpDir, "test-workflow.md")
			if err := os.WriteFile(testFile, []byte(tt.content), 0644); err != nil {
				t.Fatal(err)
			}

			compiler := NewCompiler(false, "", "")
			compiler.SetStrictMode(true)
			err = compiler.CompileWorkflow(testFile)

			if tt.expectError && err == nil {
				t.Error("Expected compilation to fail but it succeeded")
			} else if !tt.expectError && err != nil {
				t.Errorf("Expected compilation to succeed but it failed: %v", err)
			} else if tt.expectError && err != nil && tt.errorMsg != "" {
				if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error containing '%s', got '%s'", tt.errorMsg, err.Error())
				}
			}
		})
	}
}

// Note: Detailed MCP network validation tests are skipped because they require
// complex schema validation setup. The validation logic is implemented in
// validateStrictMCPNetwork() and will be triggered during actual workflow compilation
// when custom MCP servers with containers are detected.

func TestNonStrictModeAllowsAll(t *testing.T) {
	// Verify that non-strict mode (default) allows all configurations
	content := `---
on: push
permissions:
  contents: write
  issues: write
engine: copilot
network:
  allowed:
    - "*"
---

# Test Workflow`

	tmpDir, err := os.MkdirTemp("", "non-strict-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	testFile := filepath.Join(tmpDir, "test-workflow.md")
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	compiler := NewCompiler(false, "", "")
	// Do NOT set strict mode - should allow everything
	err = compiler.CompileWorkflow(testFile)

	if err != nil {
		t.Errorf("Non-strict mode should allow all configurations, but got error: %v", err)
	}
}

func TestStrictModeFromFrontmatter(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		expectError bool
		errorMsg    string
	}{
		{
			name: "strict: true in frontmatter enables strict mode",
			content: `---
on: push
strict: true
permissions:
  contents: read
engine: copilot
network:
  allowed:
    - "api.example.com"
---

# Test Workflow`,
			expectError: false,
		},
		{
			name: "strict: false in frontmatter does not enable strict mode",
			content: `---
on: push
strict: false
permissions:
  contents: write
engine: copilot
---

# Test Workflow`,
			expectError: false,
		},
		{
			name: "strict: true with valid configuration passes",
			content: `---
on: push
strict: true
permissions:
  contents: read
timeout_minutes: 10
engine: copilot
network:
  allowed:
    - "api.example.com"
---

# Test Workflow`,
			expectError: false,
		},
		{
			name: "no strict field defaults to non-strict mode",
			content: `---
on: push
permissions:
  contents: write
engine: copilot
---

# Test Workflow`,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir, err := os.MkdirTemp("", "frontmatter-strict-test")
			if err != nil {
				t.Fatal(err)
			}
			defer os.RemoveAll(tmpDir)

			testFile := filepath.Join(tmpDir, "test-workflow.md")
			if err := os.WriteFile(testFile, []byte(tt.content), 0644); err != nil {
				t.Fatal(err)
			}

			compiler := NewCompiler(false, "", "")
			// Do NOT set strict mode via CLI - let frontmatter control it
			err = compiler.CompileWorkflow(testFile)

			if tt.expectError && err == nil {
				t.Error("Expected compilation to fail but it succeeded")
			} else if !tt.expectError && err != nil {
				t.Errorf("Expected compilation to succeed but it failed: %v", err)
			} else if tt.expectError && err != nil && tt.errorMsg != "" {
				if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error containing '%s', got '%s'", tt.errorMsg, err.Error())
				}
			}
		})
	}
}

func TestCLIStrictFlagTakesPrecedence(t *testing.T) {
	// CLI --strict flag should override frontmatter strict: false
	content := `---
on: push
strict: false
permissions:
  contents: write
engine: copilot
---

# Test Workflow`

	tmpDir, err := os.MkdirTemp("", "cli-precedence-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	testFile := filepath.Join(tmpDir, "test-workflow.md")
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	compiler := NewCompiler(false, "", "")
	compiler.SetStrictMode(true) // CLI flag sets strict mode
	err = compiler.CompileWorkflow(testFile)

	// Should fail because CLI flag enforces strict mode and write permission is not allowed
	if err == nil {
		t.Error("Expected compilation to fail with CLI --strict flag, but it succeeded")
	} else if !strings.Contains(err.Error(), "write permission") {
		t.Errorf("Expected write permission error, got: %v", err)
	}
}

func TestStrictModeIsolation(t *testing.T) {
	// Test that strict mode in one workflow doesn't affect other workflows
	tmpDir, err := os.MkdirTemp("", "strict-isolation-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// First workflow with strict: true (should succeed now without timeout)
	strictWorkflow := `---
on: push
strict: true
permissions:
  contents: read
engine: copilot
network:
  allowed:
    - "api.example.com"
---

# Strict Workflow`

	// Second workflow without strict mode (no timeout_minutes - should succeed)
	nonStrictWorkflow := `---
on: push
permissions:
  contents: write
engine: copilot
---

# Non-Strict Workflow`

	strictFile := filepath.Join(tmpDir, "strict-workflow.md")
	nonStrictFile := filepath.Join(tmpDir, "non-strict-workflow.md")

	if err := os.WriteFile(strictFile, []byte(strictWorkflow), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(nonStrictFile, []byte(nonStrictWorkflow), 0644); err != nil {
		t.Fatal(err)
	}

	compiler := NewCompiler(false, "", "")
	// Do NOT set strict mode via CLI - let frontmatter control it

	// Compile strict workflow first - should succeed now
	err = compiler.CompileWorkflow(strictFile)
	if err != nil {
		t.Errorf("Expected strict workflow to succeed, but it failed: %v", err)
	}

	// Compile non-strict workflow second - should also succeed
	// This tests that strict mode from first workflow doesn't leak
	err = compiler.CompileWorkflow(nonStrictFile)
	if err != nil {
		t.Errorf("Expected non-strict workflow to succeed, but it failed: %v", err)
	}
}

func TestStrictModeForbiddenExpressions(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		expectError bool
		errorMsg    string
	}{
		{
			name: "git.workflow expression refused in strict mode",
			content: `---
on: push
strict: true
permissions:
  contents: read
engine: copilot
network:
  allowed:
    - "api.example.com"
---

# Test Workflow

Use ${{ git.workflow }} in this test.`,
			expectError: true,
			errorMsg:    "strict mode: forbidden expressions found",
		},
		{
			name: "git.agent expression refused in strict mode",
			content: `---
on: push
strict: true
permissions:
  contents: read
engine: copilot
network:
  allowed:
    - "api.example.com"
---

# Test Workflow

The agent is ${{ git.agent }}.`,
			expectError: true,
			errorMsg:    "strict mode: forbidden expressions found",
		},
		{
			name: "both git.workflow and git.agent refused in strict mode",
			content: `---
on: push
strict: true
permissions:
  contents: read
engine: copilot
network:
  allowed:
    - "api.example.com"
---

# Test Workflow

Workflow: ${{ git.workflow }}
Agent: ${{ git.agent }}`,
			expectError: true,
			errorMsg:    "strict mode: forbidden expressions found",
		},
		{
			name: "github.workflow allowed in strict mode",
			content: `---
on: push
strict: true
permissions:
  contents: read
engine: copilot
network:
  allowed:
    - "api.example.com"
---

# Test Workflow

Use ${{ github.workflow }} in this test.`,
			expectError: false,
		},
		{
			name: "github.actor allowed in strict mode",
			content: `---
on: push
strict: true
permissions:
  contents: read
engine: copilot
network:
  allowed:
    - "api.example.com"
---

# Test Workflow

The actor is ${{ github.actor }}.`,
			expectError: false,
		},
		{
			name: "git.workflow fails expression validation in non-strict mode",
			content: `---
on: push
permissions:
  contents: read
engine: copilot
---

# Test Workflow

Use ${{ git.workflow }} in this test.`,
			expectError: true, // Still fails because git.workflow is not in allowed expressions
			errorMsg:    "unauthorized expressions",
		},
		{
			name: "git.workflow in complex expression refused in strict mode",
			content: `---
on: push
strict: true
permissions:
  contents: read
engine: copilot
network:
  allowed:
    - "api.example.com"
---

# Test Workflow

Workflow name: ${{ git.workflow || 'default' }}`,
			expectError: true,
			errorMsg:    "strict mode: forbidden expressions found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir, err := os.MkdirTemp("", "strict-expressions-test")
			if err != nil {
				t.Fatal(err)
			}
			defer os.RemoveAll(tmpDir)

			testFile := filepath.Join(tmpDir, "test-workflow.md")
			if err := os.WriteFile(testFile, []byte(tt.content), 0644); err != nil {
				t.Fatal(err)
			}

			compiler := NewCompiler(false, "", "")
			// Don't set strict mode via CLI - let frontmatter control it
			err = compiler.CompileWorkflow(testFile)

			if tt.expectError && err == nil {
				t.Error("Expected compilation to fail but it succeeded")
			} else if !tt.expectError && err != nil {
				t.Errorf("Expected compilation to succeed but it failed: %v", err)
			} else if tt.expectError && err != nil && tt.errorMsg != "" {
				if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error containing '%s', got '%s'", tt.errorMsg, err.Error())
				}
			}
		})
	}
}
