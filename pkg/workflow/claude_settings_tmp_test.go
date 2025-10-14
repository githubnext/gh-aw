package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestClaudeSettingsTmpPath(t *testing.T) {
	// Create temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "claude-settings-tmp-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a test markdown file with network permissions to trigger settings generation
	testContent := `---
on: push
permissions:
  contents: read
  issues: write
tools:
  github:
    allowed: [list_issues]
engine: claude
network:
  allowed:
    - example.com
---

# Test Claude settings tmp path

This workflow tests that .claude/settings.json is generated in /tmp directory.
`

	testFile := filepath.Join(tmpDir, "test-claude-settings-tmp.md")
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatal(err)
	}

	compiler := NewCompiler(false, "", "test")
	if err := compiler.CompileWorkflow(testFile); err != nil {
		t.Fatalf("Failed to compile workflow: %v", err)
	}

	// Read the generated lock file
	lockFile := strings.Replace(testFile, ".md", ".lock.yml", 1)
	lockContent, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatalf("Failed to read generated lock file: %v", err)
	}

	lockStr := string(lockContent)

	// Test 1: Verify .claude directory is created in /tmp/gh-aw
	if !strings.Contains(lockStr, "mkdir -p /tmp/gh-aw/.claude") {
		t.Error("Expected directory creation 'mkdir -p /tmp/gh-aw/.claude' in generated workflow")
	}

	// Test 2: Verify settings.json is written to /tmp/gh-aw/.claude/settings.json
	if !strings.Contains(lockStr, "cat > /tmp/gh-aw/.claude/settings.json") {
		t.Error("Expected settings file creation 'cat > /tmp/gh-aw/.claude/settings.json' in generated workflow")
	}

	// Test 3: With network permissions, Docker Compose mode is used
	// Settings are copied to container instead of passed as --settings flag
	if !strings.Contains(lockStr, "docker compose") {
		t.Error("Expected Docker Compose execution when network permissions are configured")
	}

	// Test 4: Verify the old paths are not present
	if strings.Contains(lockStr, "mkdir -p .claude") && !strings.Contains(lockStr, "mkdir -p /tmp/gh-aw/.claude") {
		t.Error("Found old directory path '.claude' without /tmp/gh-aw prefix in generated workflow")
	}

	if strings.Contains(lockStr, "cat > .claude/settings.json") {
		t.Error("Found old settings file path '.claude/settings.json' in generated workflow, should use /tmp/gh-aw/.claude/settings.json")
	}

	if strings.Contains(lockStr, "settings: .claude/settings.json") && !strings.Contains(lockStr, "settings: /tmp/gh-aw/.claude/settings.json") {
		t.Error("Found old settings parameter '.claude/settings.json' without /tmp/gh-aw prefix in generated workflow")
	}

	t.Logf("Successfully verified .claude/settings.json is generated in /tmp/gh-aw directory")
}
