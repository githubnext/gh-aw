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
