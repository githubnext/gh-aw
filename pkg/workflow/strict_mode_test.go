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
			name: "valid timeout in strict mode",
			content: `---
on: push
permissions:
  contents: read
timeout_minutes: 10
engine: claude
---

# Test Workflow`,
			expectError: false,
		},
		{
			name: "missing timeout in strict mode",
			content: `---
on: push
permissions:
  contents: read
engine: claude
---

# Test Workflow`,
			expectError: true,
			errorMsg:    "strict mode: 'timeout_minutes' is required",
		},
		{
			name: "zero timeout in strict mode",
			content: `---
on: push
permissions:
  contents: read
timeout_minutes: 0
engine: claude
---

# Test Workflow`,
			expectError: true,
			errorMsg:    "strict mode: 'timeout_minutes' must be a positive integer",
		},
		{
			name: "negative timeout in strict mode",
			content: `---
on: push
permissions:
  contents: read
timeout_minutes: -5
engine: claude
---

# Test Workflow`,
			expectError: true,
			errorMsg:    "strict mode: 'timeout_minutes' must be a positive integer",
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
engine: claude
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
engine: claude
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
engine: claude
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
engine: claude
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
engine: claude
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
engine: claude
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
engine: claude
---

# Test Workflow`,
			expectError: true,
			errorMsg:    "strict mode: write permission 'contents: write' is not allowed",
		},
		{
			name: "shorthand read permission allowed in strict mode",
			content: `---
on: push
permissions: read
timeout_minutes: 10
engine: claude
network:
  allowed:
    - "api.example.com"
---

# Test Workflow`,
			expectError: false,
		},
		{
			name: "shorthand read-all permission allowed in strict mode",
			content: `---
on: push
permissions: read-all
timeout_minutes: 10
engine: claude
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
engine: claude
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
engine: claude
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
engine: claude
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
engine: claude
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
engine: claude
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
engine: claude
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
engine: claude
---

# Test Workflow`,
			expectError: true,
			errorMsg:    "strict mode: 'timeout_minutes' is required",
		},
		{
			name: "strict: false in frontmatter does not enable strict mode",
			content: `---
on: push
strict: false
permissions:
  contents: write
engine: claude
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
engine: claude
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
engine: claude
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
engine: claude
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

	// Should fail because CLI flag enforces strict mode despite frontmatter saying false
	if err == nil {
		t.Error("Expected compilation to fail with CLI --strict flag, but it succeeded")
	} else if !strings.Contains(err.Error(), "timeout_minutes") {
		t.Errorf("Expected timeout error, got: %v", err)
	}
}

func TestStrictModeIsolation(t *testing.T) {
	// Test that strict mode in one workflow doesn't affect other workflows
	tmpDir, err := os.MkdirTemp("", "strict-isolation-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// First workflow with strict: true (missing timeout_minutes - should fail)
	strictWorkflow := `---
on: push
strict: true
permissions:
  contents: read
engine: claude
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
engine: claude
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

	// Compile strict workflow first - should fail
	err = compiler.CompileWorkflow(strictFile)
	if err == nil {
		t.Error("Expected strict workflow to fail due to missing timeout_minutes, but it succeeded")
	} else if !strings.Contains(err.Error(), "timeout_minutes") {
		t.Errorf("Expected timeout_minutes error for strict workflow, got: %v", err)
	}

	// Compile non-strict workflow second - should succeed
	// This tests that strict mode from first workflow doesn't leak
	err = compiler.CompileWorkflow(nonStrictFile)
	if err != nil {
		t.Errorf("Expected non-strict workflow to succeed, but it failed: %v", err)
	}
}
