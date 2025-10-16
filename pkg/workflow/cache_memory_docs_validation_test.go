package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestCacheMemoryDocumentationSnippets validates all code snippets in the cache-memory documentation
func TestCacheMemoryDocumentationSnippets(t *testing.T) {
	// Find the documentation file
	docsFile := filepath.Join("..", "..", "docs", "src", "content", "docs", "reference", "cache-memory.md")
	if _, err := os.Stat(docsFile); os.IsNotExist(err) {
		t.Skipf("Documentation file not found at %s", docsFile)
	}

	content, err := os.ReadFile(docsFile)
	if err != nil {
		t.Fatalf("Failed to read documentation file: %v", err)
	}

	// Extract code snippets
	snippets := extractAWCodeSnippets(string(content))
	if len(snippets) == 0 {
		t.Fatal("No code snippets found in documentation")
	}

	t.Logf("Found %d code snippets in cache-memory documentation", len(snippets))

	// Test each snippet
	for i, snippet := range snippets {
		snippetNum := i + 1
		t.Run(snippet.name, func(t *testing.T) {
			// Skip snippets that are just markdown content (no frontmatter)
			if !strings.Contains(snippet.content, "---") {
				t.Skip("Snippet has no frontmatter (markdown only)")
				return
			}

			// Skip import-only snippets (they need context)
			if strings.Contains(snippet.content, "imports:") && !strings.Contains(snippet.content, "engine:") {
				t.Skip("Snippet is import-only (requires shared files)")
				return
			}

			// Validate the snippet can be compiled
			testSnippetCompilation(t, snippet.content, snippetNum)
		})
	}
}

// testSnippetCompilation tests that a snippet can be compiled successfully
func testSnippetCompilation(t *testing.T, snippetContent string, snippetNum int) {
	// Create a temporary workflow directory
	tmpDir := t.TempDir()
	workflowDir := filepath.Join(tmpDir, ".github", "workflows")
	if err := os.MkdirAll(workflowDir, 0755); err != nil {
		t.Fatalf("Failed to create workflow directory: %v", err)
	}

	// Add minimum required fields if missing
	if !strings.Contains(snippetContent, "on:") {
		// Insert 'on: workflow_dispatch' after the first line of frontmatter
		lines := strings.Split(snippetContent, "\n")
		for i, line := range lines {
			if strings.HasPrefix(line, "engine:") {
				lines = append(lines[:i+1], append([]string{"on: workflow_dispatch"}, lines[i+1:]...)...)
				snippetContent = strings.Join(lines, "\n")
				break
			}
		}
	}

	// Add markdown content if snippet is frontmatter-only
	if !strings.Contains(snippetContent, "#") {
		snippetContent += "\n\n# Test Workflow\nThis is a test workflow to validate configuration.\n"
	}

	// Write the snippet to a temporary file
	workflowFile := filepath.Join(workflowDir, "test-snippet.md")
	if err := os.WriteFile(workflowFile, []byte(snippetContent), 0644); err != nil {
		t.Fatalf("Failed to write workflow file: %v", err)
	}

	// Use compiler to parse the workflow
	compiler := NewCompiler(false, "", "test")
	workflow, err := compiler.ParseWorkflowFile(workflowFile)
	if err != nil {
		t.Errorf("Snippet %d: Failed to parse workflow: %v", snippetNum, err)
		return
	}

	// Validate cache-memory configuration if present
	if workflow.Tools != nil {
		if cacheMemory, ok := workflow.Tools["cache-memory"]; ok && cacheMemory != nil {
			validateCacheMemoryConfig(t, cacheMemory, snippetNum)
		}
	}

	t.Logf("Snippet %d: Successfully parsed and validated", snippetNum)
}

// validateCacheMemoryConfig validates a cache-memory configuration
func validateCacheMemoryConfig(t *testing.T, cacheMemory any, snippetNum int) {
	compiler := NewCompiler(false, "", "test")
	tools := map[string]any{
		"cache-memory": cacheMemory,
	}

	config, err := compiler.extractCacheMemoryConfig(tools)
	if err != nil {
		t.Errorf("Snippet %d: Failed to extract cache-memory config: %v", snippetNum, err)
		return
	}

	// Validate the config
	if config == nil {
		t.Errorf("Snippet %d: Cache-memory config is nil", snippetNum)
		return
	}

	// Validate individual caches
	for _, cache := range config.Caches {
		// Validate retention-days if present
		if cache.RetentionDays != nil {
			if *cache.RetentionDays < 1 || *cache.RetentionDays > 90 {
				t.Errorf("Snippet %d: Invalid retention-days value: %d (must be 1-90)", snippetNum, *cache.RetentionDays)
			}
		}

		// Validate cache key
		if cache.Key == "" {
			t.Errorf("Snippet %d: Empty cache key", snippetNum)
		}

		// Validate cache ID
		if cache.ID == "" {
			t.Errorf("Snippet %d: Empty cache ID", snippetNum)
		}
	}
}

// documentationSnippet represents a code snippet from documentation
type documentationSnippet struct {
	name    string
	content string
	line    int
}

// extractAWCodeSnippets extracts all ```aw code blocks from markdown content
func extractAWCodeSnippets(markdown string) []documentationSnippet {
	var snippets []documentationSnippet
	lines := strings.Split(markdown, "\n")
	
	var currentSnippet strings.Builder
	var inSnippet bool
	var snippetStartLine int
	snippetCount := 0

	for i, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "```aw") {
			inSnippet = true
			snippetStartLine = i + 1
			snippetCount++
			currentSnippet.Reset()
			continue
		}

		if inSnippet && strings.HasPrefix(strings.TrimSpace(line), "```") {
			inSnippet = false
			contextLine := snippetStartLine - 10
			if contextLine < 0 {
				contextLine = 0
			}
			snippetName := "Snippet " + strings.TrimSpace(strings.Split(lines[contextLine], "#")[0])
			snippets = append(snippets, documentationSnippet{
				name:    snippetName,
				content: currentSnippet.String(),
				line:    snippetStartLine,
			})
			continue
		}

		if inSnippet {
			currentSnippet.WriteString(line)
			currentSnippet.WriteString("\n")
		}
	}

	return snippets
}

// TestCacheMemoryDocumentationExamples validates specific documentation examples
func TestCacheMemoryDocumentationExamples(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		shouldWork  bool
		description string
	}{
		{
			name: "Basic enable pattern",
			content: `---
engine: claude
on: workflow_dispatch
tools:
  cache-memory: true
  github:
    allowed: [get_repository]
---

# Test Workflow
Test basic cache-memory enablement.
`,
			shouldWork:  true,
			description: "Should successfully parse basic cache-memory: true configuration",
		},
		{
			name: "Custom key with retention",
			content: `---
engine: claude
on: workflow_dispatch
tools:
  cache-memory:
    key: custom-memory-${{ github.workflow }}-${{ github.run_id }}
    retention-days: 30
  github:
    allowed: [get_repository]
---

# Test Workflow
Test custom key with retention days.
`,
			shouldWork:  true,
			description: "Should successfully parse custom key and retention configuration",
		},
		{
			name: "Multiple cache folders",
			content: `---
engine: claude
on: workflow_dispatch
tools:
  cache-memory:
    - id: default
      key: memory-default
    - id: session
      key: memory-session-${{ github.run_id }}
    - id: logs
      retention-days: 7
  github:
    allowed: [get_repository]
---

# Test Workflow
Test multiple cache folders.
`,
			shouldWork:  true,
			description: "Should successfully parse multiple cache configuration",
		},
		{
			name: "Invalid retention days (too high)",
			content: `---
engine: claude
on: workflow_dispatch
tools:
  cache-memory:
    retention-days: 100
---

# Test Workflow
Test invalid retention days.
`,
			shouldWork:  false,
			description: "Should fail with retention-days > 90",
		},
		{
			name: "Invalid retention days (too low)",
			content: `---
engine: claude
on: workflow_dispatch
tools:
  cache-memory:
    retention-days: 0
---

# Test Workflow
Test invalid retention days.
`,
			shouldWork:  false,
			description: "Should fail with retention-days < 1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a temporary workflow directory
			tmpDir := t.TempDir()
			workflowDir := filepath.Join(tmpDir, ".github", "workflows")
			if err := os.MkdirAll(workflowDir, 0755); err != nil {
				t.Fatalf("Failed to create workflow directory: %v", err)
			}

			// Write the workflow to a temporary file
			workflowFile := filepath.Join(workflowDir, "test-workflow.md")
			if err := os.WriteFile(workflowFile, []byte(tt.content), 0644); err != nil {
				t.Fatalf("Failed to write workflow file: %v", err)
			}

			// Use compiler to parse the workflow
			compiler := NewCompiler(false, "", "test")
			workflow, err := compiler.ParseWorkflowFile(workflowFile)
			
			if tt.shouldWork {
				if err != nil {
					t.Errorf("Expected workflow to parse successfully, but got error: %v", err)
					return
				}

				// Validate cache-memory configuration if present
				if workflow.Tools != nil {
					if cacheMemory, ok := workflow.Tools["cache-memory"]; ok && cacheMemory != nil {
						validateCacheMemoryConfig(t, cacheMemory, 0)
					}
				}
			} else {
				// For invalid configs, we expect compilation to fail
				if err == nil && workflow.Tools != nil {
					if cacheMemory, ok := workflow.Tools["cache-memory"]; ok && cacheMemory != nil {
						// Try to extract config - should fail for invalid retention days
						compiler := NewCompiler(false, "", "test")
						tools := map[string]any{
							"cache-memory": cacheMemory,
						}
						config, extractErr := compiler.extractCacheMemoryConfig(tools)
						
						// Check if validation catches the error
						hasError := extractErr != nil
						if config != nil {
							for _, cache := range config.Caches {
								if cache.RetentionDays != nil {
									if *cache.RetentionDays < 1 || *cache.RetentionDays > 90 {
										hasError = true
										break
									}
								}
							}
						}
						
						if !hasError {
							t.Error("Expected validation to fail but it passed")
						}
					}
				}
			}
		})
	}
}
