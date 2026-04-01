//go:build !integration

package cli

import (
	"os"
	"strings"
	"testing"
)

func TestProcessIncludesWithWorkflowSpec_NewSyntax(t *testing.T) {
	// Test with new {{#import}} syntax
	content := `---
engine: claude
---

# Test Workflow

Some content here.

{{#import? agentics/weekly-research.config}}

More content.
`

	workflow := &WorkflowSpec{
		RepoSpec: RepoSpec{
			RepoSlug: "githubnext/agentics",
			Version:  "main",
		},
	}

	result, err := processIncludesWithWorkflowSpec(content, workflow, "", "/tmp/package", false)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Should convert to @include with workflowspec
	expectedInclude := "{{#import? githubnext/agentics/agentics/weekly-research.config@main}}"
	if !strings.Contains(result, expectedInclude) {
		t.Errorf("Expected result to contain '%s'\nGot:\n%s", expectedInclude, result)
	}

	// Should NOT contain the malformed path
	malformedPath := "githubnext/agentics/@"
	if strings.Contains(result, malformedPath) {
		t.Errorf("Result should NOT contain malformed path '%s'\nGot:\n%s", malformedPath, result)
	}
}

func TestProcessIncludesWithWorkflowSpec_LegacySyntax(t *testing.T) {
	// Test with legacy @include syntax
	content := `---
engine: claude
---

# Test Workflow

Some content here.

@include? shared/config.md

More content.
`

	workflow := &WorkflowSpec{
		RepoSpec: RepoSpec{
			RepoSlug: "githubnext/agentics",
			Version:  "main",
		},
	}

	result, err := processIncludesWithWorkflowSpec(content, workflow, "", "/tmp/package", false)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Should convert to @include with workflowspec
	expectedInclude := "{{#import? githubnext/agentics/shared/config.md@main}}"
	if !strings.Contains(result, expectedInclude) {
		t.Errorf("Expected result to contain '%s'\nGot:\n%s", expectedInclude, result)
	}
}

func TestProcessIncludesWithWorkflowSpec_WithCommitSHA(t *testing.T) {
	// Test with commit SHA
	content := `---
engine: claude
---

# Test Workflow

{{#import agentics/config.md}}
`

	workflow := &WorkflowSpec{
		RepoSpec: RepoSpec{
			RepoSlug: "githubnext/agentics",
		},
	}

	commitSHA := "e2770974a7eaccb58ddafd5606c38a05ba52c631"

	result, err := processIncludesWithWorkflowSpec(content, workflow, commitSHA, "/tmp/package", false)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Should use commit SHA instead of version
	expectedInclude := "{{#import githubnext/agentics/agentics/config.md@e2770974a7eaccb58ddafd5606c38a05ba52c631}}"
	if !strings.Contains(result, expectedInclude) {
		t.Errorf("Expected result to contain '%s'\nGot:\n%s", expectedInclude, result)
	}
}

func TestProcessIncludesWithWorkflowSpec_EmptyFilePath(t *testing.T) {
	// Test with section-only reference (should be skipped/passed through)
	content := `---
engine: claude
---

# Test Workflow

{{#import? #SectionName}}

More content.
`

	workflow := &WorkflowSpec{
		RepoSpec: RepoSpec{
			RepoSlug: "githubnext/agentics",
			Version:  "main",
		},
	}

	result, err := processIncludesWithWorkflowSpec(content, workflow, "", "/tmp/package", false)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Should preserve the original line when filePath is empty
	if !strings.Contains(result, "{{#import? #SectionName}}") {
		t.Errorf("Expected result to preserve original line\nGot:\n%s", result)
	}

	// Should NOT generate malformed workflowspec
	malformedPath := "githubnext/agentics/@"
	if strings.Contains(result, malformedPath) {
		t.Errorf("Result should NOT contain malformed path '%s'\nGot:\n%s", malformedPath, result)
	}
}

func TestProcessIncludesInContent_NewSyntax(t *testing.T) {
	// Test processIncludesInContent with new syntax
	content := `---
engine: claude
---

# Test Workflow

{{#import? config/settings.md}}
`

	workflow := &WorkflowSpec{
		RepoSpec: RepoSpec{
			RepoSlug: "owner/repo",
			Version:  "v1.0.0",
		},
	}

	result, err := processIncludesInContent(content, workflow, "", "", false)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Should convert to workflowspec format
	expectedInclude := "{{#import? owner/repo/config/settings.md@v1.0.0}}"
	if !strings.Contains(result, expectedInclude) {
		t.Errorf("Expected result to contain '%s'\nGot:\n%s", expectedInclude, result)
	}
}

func TestProcessIncludesInContent_EmptyFilePath(t *testing.T) {
	// Test processIncludesInContent with empty file path
	content := `---
engine: claude
---

# Test Workflow

@include? #JustASection
`

	workflow := &WorkflowSpec{
		RepoSpec: RepoSpec{
			RepoSlug: "owner/repo",
			Version:  "v1.0.0",
		},
	}

	result, err := processIncludesInContent(content, workflow, "", "", false)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Should preserve the original line
	if !strings.Contains(result, "@include? #JustASection") {
		t.Errorf("Expected result to preserve original line\nGot:\n%s", result)
	}

	// Should NOT generate malformed workflowspec
	malformedPath := "owner/repo/@"
	if strings.Contains(result, malformedPath) {
		t.Errorf("Result should NOT contain malformed path '%s'\nGot:\n%s", malformedPath, result)
	}
}

func TestProcessIncludesWithWorkflowSpec_RealWorldScenario(t *testing.T) {
	// Test the exact scenario from the weekly-research workflow bug report
	// The workflow has: {{#import? agentics/weekly-research.config}}
	// Previously this would generate: githubnext/agentics/@e2770974...
	// Now it should generate: githubnext/agentics/agentics/weekly-research.config@e2770974...

	content := `---
on:
  schedule:
    - cron: "0 9 * * 1"

tools:
  web-fetch:
  web-search:
---

# Weekly Research

Do research.

{{#import? agentics/weekly-research.config}}
`

	workflow := &WorkflowSpec{
		RepoSpec: RepoSpec{
			RepoSlug: "githubnext/agentics",
		},
	}

	commitSHA := "e2770974a7eaccb58ddafd5606c38a05ba52c631"

	result, err := processIncludesWithWorkflowSpec(content, workflow, commitSHA, "/tmp/package", false)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Should convert to proper workflowspec
	expectedInclude := "{{#import? githubnext/agentics/agentics/weekly-research.config@e2770974a7eaccb58ddafd5606c38a05ba52c631}}"
	if !strings.Contains(result, expectedInclude) {
		t.Errorf("Expected result to contain '%s'\nGot:\n%s", expectedInclude, result)
	}

	// Should NOT contain the malformed path from the bug report
	malformedPath := "githubnext/agentics/@e2770974"
	if strings.Contains(result, malformedPath) {
		t.Errorf("Result should NOT contain malformed path '%s' (the original bug)\nGot:\n%s", malformedPath, result)
	}
}

func TestIsWorkflowSpecFormat(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{
			name:     "workflowspec with SHA",
			path:     "owner/repo/path/file.md@abc123",
			expected: true,
		},
		{
			name:     "workflowspec with version tag",
			path:     "owner/repo/file.md@v1.0.0",
			expected: true,
		},
		{
			name:     "workflowspec without version - NOT a workflowspec",
			path:     "owner/repo/path/file.md",
			expected: false, // Without @, it's not detected as a workflowspec
		},
		{
			name:     "three-part relative path - NOT a workflowspec",
			path:     "shared/mcp/arxiv.md",
			expected: false, // Local path, not a workflowspec
		},
		{
			name:     "two-part relative path",
			path:     "shared/file.md",
			expected: false,
		},
		{
			name:     "relative path with ./",
			path:     "./shared/file.md",
			expected: false,
		},
		{
			name:     "absolute path",
			path:     "/shared/file.md",
			expected: false,
		},
		{
			name:     "workflowspec with section and version",
			path:     "owner/repo/path/file.md@sha#section",
			expected: true,
		},
		{
			name:     "simple filename",
			path:     "file.md",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isWorkflowSpecFormat(tt.path)
			if result != tt.expected {
				t.Errorf("isWorkflowSpecFormat(%q) = %v, want %v", tt.path, result, tt.expected)
			}
		})
	}
}

func TestProcessImportsWithWorkflowSpec_ThreePartPath(t *testing.T) {
	// Test that three-part paths like "shared/mcp/arxiv.md" are correctly converted
	// to workflowspecs, not skipped as if they were already workflowspecs
	content := `---
engine: copilot
imports:
  - shared/mcp/arxiv.md
  - shared/reporting.md
  - shared/mcp/brave.md
---

# Test Workflow

Test content.
`

	workflow := &WorkflowSpec{
		RepoSpec: RepoSpec{
			RepoSlug: "github/gh-aw",
			Version:  "main",
		},
		WorkflowPath: ".github/workflows/test-workflow.md",
	}

	commitSHA := "abc123def456"

	result, err := processImportsWithWorkflowSpec(content, workflow, commitSHA, "", false)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// All imports should be converted to workflowspecs with the commit SHA
	expectedImports := []string{
		"github/gh-aw/.github/workflows/shared/mcp/arxiv.md@abc123def456",
		"github/gh-aw/.github/workflows/shared/reporting.md@abc123def456",
		"github/gh-aw/.github/workflows/shared/mcp/brave.md@abc123def456",
	}

	for _, expected := range expectedImports {
		if !strings.Contains(result, expected) {
			t.Errorf("Expected result to contain '%s'\nGot:\n%s", expected, result)
		}
	}

	// The original paths should NOT appear unchanged
	unchangedPaths := []string{
		"- shared/mcp/arxiv.md",
		"- shared/reporting.md",
		"- shared/mcp/brave.md",
	}

	for _, unchanged := range unchangedPaths {
		if strings.Contains(result, unchanged) {
			t.Errorf("Did not expect result to contain unchanged path '%s'\nGot:\n%s", unchanged, result)
		}
	}
}

// TestProcessImportsWithWorkflowSpec_PreservesLocalRelativePaths tests that when
// localWorkflowDir is provided and import files exist on disk, the relative paths
// are kept as-is and NOT rewritten to cross-repo workflowspec references.
// This is the fix for: gh aw update rewrites local imports: to cross-repo paths.
func TestProcessImportsWithWorkflowSpec_PreservesLocalRelativePaths(t *testing.T) {
	// Create a temporary directory to act as the local workflow directory
	tmpDir := t.TempDir()

	// Create the shared import files locally
	for _, rel := range []string{"shared/team-config.md", "shared/aor-index.md"} {
		dir := tmpDir + "/" + rel[:strings.LastIndex(rel, "/")]
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("Failed to create dir %s: %v", dir, err)
		}
		if err := os.WriteFile(tmpDir+"/"+rel, []byte("# Shared content"), 0644); err != nil {
			t.Fatalf("Failed to create file %s: %v", rel, err)
		}
	}

	content := `---
engine: copilot
imports:
  - shared/team-config.md
  - shared/aor-index.md
---

# Investigate
`

	workflow := &WorkflowSpec{
		RepoSpec: RepoSpec{
			RepoSlug: "github/identity-core",
			Version:  "cd32c168",
		},
		WorkflowPath: ".github/workflows/investigate.md",
	}

	result, err := processImportsWithWorkflowSpec(content, workflow, "cd32c168", tmpDir, false)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Local paths must be preserved as-is
	if !strings.Contains(result, "- shared/team-config.md") {
		t.Errorf("Expected local import 'shared/team-config.md' to be preserved, got:\n%s", result)
	}
	if !strings.Contains(result, "- shared/aor-index.md") {
		t.Errorf("Expected local import 'shared/aor-index.md' to be preserved, got:\n%s", result)
	}

	// Cross-repo refs must NOT appear
	if strings.Contains(result, "github/identity-core") {
		t.Errorf("Cross-repo ref should NOT appear when local file exists, got:\n%s", result)
	}
}

// TestProcessImportsWithWorkflowSpec_RewritesWhenLocalMissing verifies that imports
// for files that do NOT exist locally are still rewritten to cross-repo refs.
func TestProcessImportsWithWorkflowSpec_RewritesWhenLocalMissing(t *testing.T) {
	// Use a temp dir that has NO shared files
	tmpDir := t.TempDir()

	content := `---
engine: copilot
imports:
  - shared/team-config.md
---

# Investigate
`

	workflow := &WorkflowSpec{
		RepoSpec: RepoSpec{
			RepoSlug: "github/identity-core",
			Version:  "cd32c168",
		},
		WorkflowPath: ".github/workflows/investigate.md",
	}

	result, err := processImportsWithWorkflowSpec(content, workflow, "cd32c168", tmpDir, false)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// File does NOT exist locally → must be rewritten to cross-repo ref
	expectedRef := "github/identity-core/.github/workflows/shared/team-config.md@cd32c168"
	if !strings.Contains(result, expectedRef) {
		t.Errorf("Expected cross-repo ref '%s' when file is missing locally, got:\n%s", expectedRef, result)
	}

	// Original relative path must be gone
	if strings.Contains(result, "- shared/team-config.md") {
		t.Errorf("Relative path should have been rewritten when file is missing locally, got:\n%s", result)
	}
}

// TestProcessIncludesInContent_PreservesLocalIncludeDirectives tests that @include
// directives whose files exist locally are not rewritten to cross-repo refs.
func TestProcessIncludesInContent_PreservesLocalIncludeDirectives(t *testing.T) {
	// Create a temporary directory with the shared include file
	tmpDir := t.TempDir()
	if err := os.MkdirAll(tmpDir+"/shared", 0755); err != nil {
		t.Fatalf("Failed to create shared dir: %v", err)
	}
	if err := os.WriteFile(tmpDir+"/shared/config.md", []byte("# Config"), 0644); err != nil {
		t.Fatalf("Failed to create config file: %v", err)
	}

	content := `---
engine: copilot
---

# Test Workflow

{{#import shared/config.md}}
`

	workflow := &WorkflowSpec{
		RepoSpec: RepoSpec{
			RepoSlug: "github/identity-core",
			Version:  "abc123",
		},
		WorkflowPath: ".github/workflows/test.md",
	}

	result, err := processIncludesInContent(content, workflow, "abc123", tmpDir, false)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Local include directive must be preserved
	if !strings.Contains(result, "{{#import shared/config.md}}") {
		t.Errorf("Expected local @include to be preserved, got:\n%s", result)
	}

	// Cross-repo ref must NOT appear
	if strings.Contains(result, "github/identity-core") {
		t.Errorf("Cross-repo ref should NOT appear when local file exists, got:\n%s", result)
	}
}

// TestIsLocalFileForUpdate_PathTraversal ensures that traversal attempts (e.g.
// "../../etc/passwd") are rejected even if the target path happens to exist.
func TestIsLocalFileForUpdate_PathTraversal(t *testing.T) {
	tmpDir := t.TempDir()

	// Traversal path that would escape tmpDir
	traversal := "../../etc/passwd"
	if isLocalFileForUpdate(tmpDir, traversal) {
		t.Errorf("isLocalFileForUpdate should reject path traversal attempt: %s", traversal)
	}

	// A normal path within tmpDir that doesn't exist should return false
	if isLocalFileForUpdate(tmpDir, "nonexistent.md") {
		t.Errorf("isLocalFileForUpdate should return false for non-existent file")
	}

	// A normal path within tmpDir that DOES exist should return true
	validFile := "shared/file.md"
	if err := os.MkdirAll(tmpDir+"/shared", 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(tmpDir+"/"+validFile, []byte("content"), 0644); err != nil {
		t.Fatal(err)
	}
	if !isLocalFileForUpdate(tmpDir, validFile) {
		t.Errorf("isLocalFileForUpdate should return true for an existing file within tmpDir")
	}
}
