package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/githubnext/gh-aw/pkg/testutil"
)

func TestNoopWithPostAsComment(t *testing.T) {
	// Create temporary directory for test files
	tmpDir := testutil.TempDir(t, "noop-post-as-comment-test")

	// Create a test markdown file with noop safe output and post-as-comment
	testContent := `---
on:
  issues:
    types: [opened]
permissions:
  contents: read
engine: copilot
safe-outputs:
  noop:
    max: 5
    post-as-comment: "https://github.com/owner/repo/issues/123"
---

# Test Noop with Post-as-Comment

Test that noop configuration includes post-as-comment field.
`

	testFile := filepath.Join(tmpDir, "test-noop-post.md")
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatal(err)
	}

	compiler := NewCompiler(false, "", "test")

	// Compile the workflow
	if err := compiler.CompileWorkflow(testFile); err != nil {
		t.Fatalf("Failed to compile workflow: %v", err)
	}

	// Read the compiled workflow
	lockFile := filepath.Join(tmpDir, "test-noop-post.lock.yml")
	compiledBytes, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatalf("Failed to read compiled workflow: %v", err)
	}
	compiled := string(compiledBytes)

	// Verify that conclusion job exists and contains the post-as-comment URL
	if !strings.Contains(compiled, "\n  conclusion:") {
		t.Error("Conclusion job should exist")
	}

	// Verify that the post-as-comment URL is passed as an environment variable
	if !strings.Contains(compiled, "GH_AW_NOOP_POST_AS_COMMENT") {
		t.Error("Conclusion job should contain GH_AW_NOOP_POST_AS_COMMENT environment variable")
	}

	// Verify the URL is present
	if !strings.Contains(compiled, "https://github.com/owner/repo/issues/123") {
		t.Error("Conclusion job should contain the post-as-comment URL")
	}
}

func TestNoopWithoutPostAsComment(t *testing.T) {
	// Create temporary directory for test files
	tmpDir := testutil.TempDir(t, "noop-without-post-test")

	// Create a test markdown file with noop safe output but no post-as-comment
	testContent := `---
on:
  issues:
    types: [opened]
permissions:
  contents: read
engine: copilot
safe-outputs:
  noop:
    max: 5
---

# Test Noop without Post-as-Comment

Test that noop works without post-as-comment field.
`

	testFile := filepath.Join(tmpDir, "test-noop-no-post.md")
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatal(err)
	}

	compiler := NewCompiler(false, "", "test")

	// Compile the workflow
	if err := compiler.CompileWorkflow(testFile); err != nil {
		t.Fatalf("Failed to compile workflow: %v", err)
	}

	// Read the compiled workflow
	lockFile := filepath.Join(tmpDir, "test-noop-no-post.lock.yml")
	compiledBytes, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatalf("Failed to read compiled workflow: %v", err)
	}
	compiled := string(compiledBytes)

	// Verify that conclusion job exists
	if !strings.Contains(compiled, "\n  conclusion:") {
		t.Error("Conclusion job should exist")
	}

	// Verify that the post-as-comment environment variable is NOT present when not configured
	// The JavaScript code will reference the env var, but we should not see it being SET in the env section
	// Look for the pattern "GH_AW_NOOP_POST_AS_COMMENT:" which would indicate it's being set
	if strings.Contains(compiled, "GH_AW_NOOP_POST_AS_COMMENT:") {
		// Print the offending section for debugging
		lines := strings.Split(compiled, "\n")
		for i, line := range lines {
			if strings.Contains(line, "GH_AW_NOOP_POST_AS_COMMENT:") {
				start := i - 5
				if start < 0 {
					start = 0
				}
				end := i + 5
				if end >= len(lines) {
					end = len(lines) - 1
				}
				t.Logf("Found GH_AW_NOOP_POST_AS_COMMENT: at line %d:", i)
				for j := start; j <= end; j++ {
					t.Logf("%4d: %s", j, lines[j])
				}
			}
		}
		t.Error("Conclusion job should NOT set GH_AW_NOOP_POST_AS_COMMENT environment variable when not configured")
	}
}

func TestNoopWithPostAsCommentDiscussion(t *testing.T) {
	// Create temporary directory for test files
	tmpDir := testutil.TempDir(t, "noop-post-discussion-test")

	// Create a test markdown file with noop pointing to a discussion
	testContent := `---
on:
  issues:
    types: [opened]
permissions:
  contents: read
engine: copilot
safe-outputs:
  noop:
    max: 1
    post-as-comment: "https://github.com/owner/repo/discussions/456"
---

# Test Noop with Discussion URL

Test that noop configuration supports discussion URLs.
`

	testFile := filepath.Join(tmpDir, "test-noop-discussion.md")
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatal(err)
	}

	compiler := NewCompiler(false, "", "test")

	// Compile the workflow
	if err := compiler.CompileWorkflow(testFile); err != nil {
		t.Fatalf("Failed to compile workflow: %v", err)
	}

	// Read the compiled workflow
	lockFile := filepath.Join(tmpDir, "test-noop-discussion.lock.yml")
	compiledBytes, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatalf("Failed to read compiled workflow: %v", err)
	}
	compiled := string(compiledBytes)

	// Verify that the discussion URL is present
	if !strings.Contains(compiled, "https://github.com/owner/repo/discussions/456") {
		t.Error("Conclusion job should contain the discussion URL")
	}
}

func TestNoopWithPostAsCommentShortPath(t *testing.T) {
	// Create temporary directory for test files
	tmpDir := testutil.TempDir(t, "noop-post-short-path-test")

	// Create a test markdown file with noop using short path format
	testContent := `---
on:
  issues:
    types: [opened]
permissions:
  contents: read
engine: copilot
safe-outputs:
  noop:
    max: 1
    post-as-comment: "owner/repo/issues/789"
---

# Test Noop with Short Path

Test that noop configuration supports short path format.
`

	testFile := filepath.Join(tmpDir, "test-noop-short.md")
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatal(err)
	}

	compiler := NewCompiler(false, "", "test")

	// Compile the workflow
	if err := compiler.CompileWorkflow(testFile); err != nil {
		t.Fatalf("Failed to compile workflow: %v", err)
	}

	// Read the compiled workflow
	lockFile := filepath.Join(tmpDir, "test-noop-short.lock.yml")
	compiledBytes, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatalf("Failed to read compiled workflow: %v", err)
	}
	compiled := string(compiledBytes)

	// Verify that the short path is present
	if !strings.Contains(compiled, "owner/repo/issues/789") {
		t.Error("Conclusion job should contain the short path")
	}

	// Verify the environment variable is set
	if !strings.Contains(compiled, "GH_AW_NOOP_POST_AS_COMMENT:") {
		t.Error("Conclusion job should set GH_AW_NOOP_POST_AS_COMMENT environment variable")
	}
}

func TestNoopWithPostAsCommentShortPathDiscussion(t *testing.T) {
	// Create temporary directory for test files
	tmpDir := testutil.TempDir(t, "noop-post-short-discussion-test")

	// Create a test markdown file with noop using short path for discussion
	testContent := `---
on:
  issues:
    types: [opened]
permissions:
  contents: read
engine: copilot
safe-outputs:
  noop:
    max: 1
    post-as-comment: "owner/repo/discussions/999"
---

# Test Noop with Short Discussion Path

Test that noop configuration supports short path format for discussions.
`

	testFile := filepath.Join(tmpDir, "test-noop-short-discussion.md")
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatal(err)
	}

	compiler := NewCompiler(false, "", "test")

	// Compile the workflow
	if err := compiler.CompileWorkflow(testFile); err != nil {
		t.Fatalf("Failed to compile workflow: %v", err)
	}

	// Read the compiled workflow
	lockFile := filepath.Join(tmpDir, "test-noop-short-discussion.lock.yml")
	compiledBytes, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatalf("Failed to read compiled workflow: %v", err)
	}
	compiled := string(compiledBytes)

	// Verify that the short discussion path is present
	if !strings.Contains(compiled, "owner/repo/discussions/999") {
		t.Error("Conclusion job should contain the short discussion path")
	}
}

func TestNoopWithPostAsCommentNumberOnly(t *testing.T) {
	// Create temporary directory for test files
	tmpDir := testutil.TempDir(t, "noop-post-number-only-test")

	// Create a test markdown file with noop using just a number
	testContent := `---
on:
  issues:
    types: [opened]
permissions:
  contents: read
engine: copilot
safe-outputs:
  noop:
    max: 1
    post-as-comment: "42"
---

# Test Noop with Number Only

Test that noop configuration supports number-only format.
`

	testFile := filepath.Join(tmpDir, "test-noop-number.md")
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatal(err)
	}

	compiler := NewCompiler(false, "", "test")

	// Compile the workflow
	if err := compiler.CompileWorkflow(testFile); err != nil {
		t.Fatalf("Failed to compile workflow: %v", err)
	}

	// Read the compiled workflow
	lockFile := filepath.Join(tmpDir, "test-noop-number.lock.yml")
	compiledBytes, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatalf("Failed to read compiled workflow: %v", err)
	}
	compiled := string(compiledBytes)

	// Verify that the number is present
	if !strings.Contains(compiled, `GH_AW_NOOP_POST_AS_COMMENT: "42"`) {
		t.Error("Conclusion job should contain the number in the environment variable")
	}
}

func TestNoopWithPostAsCommentHashNumber(t *testing.T) {
	// Create temporary directory for test files
	tmpDir := testutil.TempDir(t, "noop-post-hash-number-test")

	// Create a test markdown file with noop using #number format
	testContent := `---
on:
  issues:
    types: [opened]
permissions:
  contents: read
engine: copilot
safe-outputs:
  noop:
    max: 1
    post-as-comment: "#123"
---

# Test Noop with Hash Number

Test that noop configuration supports #number format.
`

	testFile := filepath.Join(tmpDir, "test-noop-hash.md")
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatal(err)
	}

	compiler := NewCompiler(false, "", "test")

	// Compile the workflow
	if err := compiler.CompileWorkflow(testFile); err != nil {
		t.Fatalf("Failed to compile workflow: %v", err)
	}

	// Read the compiled workflow
	lockFile := filepath.Join(tmpDir, "test-noop-hash.lock.yml")
	compiledBytes, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatalf("Failed to read compiled workflow: %v", err)
	}
	compiled := string(compiledBytes)

	// Verify that the hash number is present
	if !strings.Contains(compiled, `GH_AW_NOOP_POST_AS_COMMENT: "#123"`) {
		t.Error("Conclusion job should contain the hash number in the environment variable")
	}
}
