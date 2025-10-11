package cli

import (
	"fmt"
	"strings"
	"testing"
)

func TestFormatPreservationManual(t *testing.T) {
	content := `---
on:
    workflow_dispatch:

    schedule:
        # Daily run at 9 AM UTC on weekdays  
        - cron: "0 9 * * 1-5"

    # Auto-stop workflow after 2 hours
    stop-after: +2h

timeout_minutes: 45

permissions: read-all

network: defaults

engine: claude

tools:
    # Enable web search functionality
    web-search: null
    
    # Memory caching for better performance  
    cache-memory: true

---

# Test Formatting Preservation

This workflow is designed to test whether the formatting is preserved.`

	result, err := addSourceToWorkflow(content, "test/repo/workflow.md@v1.0.0")
	if err != nil {
		t.Fatalf("Error adding source: %v", err)
	}

	fmt.Printf("=== ORIGINAL ===\n%s\n\n", content)
	fmt.Printf("=== RESULT ===\n%s\n", result)

	// Check if comments are preserved
	if !strings.Contains(result, "# Daily run at 9 AM UTC on weekdays") {
		t.Error("Comments were not preserved")
	}

	// Check if blank lines are preserved (look for consecutive newlines)
	if !strings.Contains(result, "\n\n") {
		t.Error("Blank lines were not preserved")
	}

	// Check if inline comments are preserved
	if !strings.Contains(result, "stop-after: +2h") {
		t.Error("Inline comments were not preserved")
	}

	// Check if indentation is preserved (4-space indentation)
	if !strings.Contains(result, "    workflow_dispatch:") {
		t.Error("4-space indentation was not preserved")
	}
}
