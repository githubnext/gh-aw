package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/githubnext/gh-aw/pkg/workflow"
)

func TestDependencyGraph_IsTopLevelWorkflow(t *testing.T) {
	tmpDir := t.TempDir()
	workflowsDir := filepath.Join(tmpDir, ".github", "workflows")
	if err := os.MkdirAll(workflowsDir, 0755); err != nil {
		t.Fatal(err)
	}

	graph := NewDependencyGraph(workflowsDir)

	tests := []struct {
		name         string
		path         string
		wantTopLevel bool
	}{
		{
			name:         "top-level workflow",
			path:         filepath.Join(workflowsDir, "main.md"),
			wantTopLevel: true,
		},
		{
			name:         "shared workflow in subdirectory",
			path:         filepath.Join(workflowsDir, "shared", "helper.md"),
			wantTopLevel: false,
		},
		{
			name:         "nested shared workflow",
			path:         filepath.Join(workflowsDir, "shared", "mcp", "tool.md"),
			wantTopLevel: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := graph.isTopLevelWorkflow(tt.path)
			if got != tt.wantTopLevel {
				t.Errorf("isTopLevelWorkflow() = %v, want %v", got, tt.wantTopLevel)
			}
		})
	}
}

func TestDependencyGraph_BuildGraphAndGetAffectedWorkflows(t *testing.T) {
	tmpDir := t.TempDir()
	workflowsDir := filepath.Join(tmpDir, ".github", "workflows")
	sharedDir := filepath.Join(workflowsDir, "shared")
	if err := os.MkdirAll(sharedDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create a shared workflow
	sharedWorkflow := filepath.Join(sharedDir, "helper.md")
	sharedContent := `---
description: Helper workflow
---
# Helper`
	if err := os.WriteFile(sharedWorkflow, []byte(sharedContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create a top-level workflow that imports the shared workflow
	topWorkflow1 := filepath.Join(workflowsDir, "main.md")
	topContent1 := `---
description: Main workflow
imports:
  - shared/helper.md
---
# Main`
	if err := os.WriteFile(topWorkflow1, []byte(topContent1), 0644); err != nil {
		t.Fatal(err)
	}

	// Create another top-level workflow that also imports the shared workflow
	topWorkflow2 := filepath.Join(workflowsDir, "secondary.md")
	topContent2 := `---
description: Secondary workflow
imports:
  - shared/helper.md
---
# Secondary`
	if err := os.WriteFile(topWorkflow2, []byte(topContent2), 0644); err != nil {
		t.Fatal(err)
	}

	// Create a top-level workflow without imports
	topWorkflow3 := filepath.Join(workflowsDir, "standalone.md")
	topContent3 := `---
description: Standalone workflow
---
# Standalone`
	if err := os.WriteFile(topWorkflow3, []byte(topContent3), 0644); err != nil {
		t.Fatal(err)
	}

	// Build dependency graph
	graph := NewDependencyGraph(workflowsDir)
	compiler := workflow.NewCompiler(false, "", "test")
	if err := graph.BuildGraph(compiler); err != nil {
		t.Fatalf("BuildGraph() error = %v", err)
	}

	// Test 1: Modifying the shared workflow should affect both top-level workflows that import it
	t.Run("shared workflow modification affects importers", func(t *testing.T) {
		affected := graph.GetAffectedWorkflows(sharedWorkflow)

		// Should return both main.md and secondary.md
		expectedCount := 2
		if len(affected) != expectedCount {
			t.Errorf("GetAffectedWorkflows() returned %d workflows, want %d", len(affected), expectedCount)
		}

		// Check that both importers are in the list
		affectedMap := make(map[string]bool)
		for _, w := range affected {
			affectedMap[w] = true
		}

		if !affectedMap[topWorkflow1] {
			t.Errorf("GetAffectedWorkflows() should include %s", topWorkflow1)
		}
		if !affectedMap[topWorkflow2] {
			t.Errorf("GetAffectedWorkflows() should include %s", topWorkflow2)
		}
	})

	// Test 2: Modifying a top-level workflow should only affect itself
	t.Run("top-level workflow modification affects only itself", func(t *testing.T) {
		affected := graph.GetAffectedWorkflows(topWorkflow1)

		if len(affected) != 1 {
			t.Errorf("GetAffectedWorkflows() returned %d workflows, want 1", len(affected))
		}

		if len(affected) > 0 && affected[0] != topWorkflow1 {
			t.Errorf("GetAffectedWorkflows() = %v, want [%s]", affected, topWorkflow1)
		}
	})

	// Test 3: Modifying a standalone workflow should only affect itself
	t.Run("standalone workflow modification affects only itself", func(t *testing.T) {
		affected := graph.GetAffectedWorkflows(topWorkflow3)

		if len(affected) != 1 {
			t.Errorf("GetAffectedWorkflows() returned %d workflows, want 1", len(affected))
		}

		if len(affected) > 0 && affected[0] != topWorkflow3 {
			t.Errorf("GetAffectedWorkflows() = %v, want [%s]", affected, topWorkflow3)
		}
	})
}

func TestDependencyGraph_UpdateAndRemoveWorkflow(t *testing.T) {
	tmpDir := t.TempDir()
	workflowsDir := filepath.Join(tmpDir, ".github", "workflows")
	sharedDir := filepath.Join(workflowsDir, "shared")
	if err := os.MkdirAll(sharedDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create a shared workflow
	sharedWorkflow := filepath.Join(sharedDir, "helper.md")
	sharedContent := `---
description: Helper workflow
---
# Helper`
	if err := os.WriteFile(sharedWorkflow, []byte(sharedContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create a top-level workflow that imports the shared workflow
	topWorkflow := filepath.Join(workflowsDir, "main.md")
	topContent := `---
description: Main workflow
imports:
  - shared/helper.md
---
# Main`
	if err := os.WriteFile(topWorkflow, []byte(topContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Build dependency graph
	graph := NewDependencyGraph(workflowsDir)
	compiler := workflow.NewCompiler(false, "", "test")
	if err := graph.BuildGraph(compiler); err != nil {
		t.Fatalf("BuildGraph() error = %v", err)
	}

	// Test: Update workflow to remove import
	t.Run("update workflow removes old dependencies", func(t *testing.T) {
		// Update the workflow to remove the import
		newContent := `---
description: Main workflow
---
# Main (no imports)`
		if err := os.WriteFile(topWorkflow, []byte(newContent), 0644); err != nil {
			t.Fatal(err)
		}

		// Update the workflow in the graph
		if err := graph.UpdateWorkflow(topWorkflow, compiler); err != nil {
			t.Fatalf("UpdateWorkflow() error = %v", err)
		}

		// Now modifying the shared workflow should not affect the top-level workflow
		affected := graph.GetAffectedWorkflows(sharedWorkflow)
		if len(affected) != 0 {
			t.Errorf("After update, GetAffectedWorkflows() returned %d workflows, want 0", len(affected))
		}
	})

	// Test: Remove workflow
	t.Run("remove workflow cleans up dependencies", func(t *testing.T) {
		// Remove the workflow
		graph.RemoveWorkflow(topWorkflow)

		// Check that the workflow is no longer in the graph
		if _, exists := graph.nodes[topWorkflow]; exists {
			t.Error("RemoveWorkflow() did not remove the node from the graph")
		}

		// Check that reverse imports are cleaned up
		if importers, exists := graph.reverseImports[sharedWorkflow]; exists && len(importers) > 0 {
			t.Errorf("RemoveWorkflow() did not clean up reverse imports, still has %d importers", len(importers))
		}
	})
}

func TestDependencyGraph_NestedImports(t *testing.T) {
	tmpDir := t.TempDir()
	workflowsDir := filepath.Join(tmpDir, ".github", "workflows")
	sharedDir := filepath.Join(workflowsDir, "shared")
	if err := os.MkdirAll(sharedDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create a base shared workflow (leaf)
	baseWorkflow := filepath.Join(sharedDir, "base.md")
	baseContent := `---
description: Base workflow
---
# Base`
	if err := os.WriteFile(baseWorkflow, []byte(baseContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create an intermediate shared workflow that imports the base
	intermediateWorkflow := filepath.Join(sharedDir, "intermediate.md")
	intermediateContent := `---
description: Intermediate workflow
imports:
  - base.md
---
# Intermediate`
	if err := os.WriteFile(intermediateWorkflow, []byte(intermediateContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create a top-level workflow that imports the intermediate workflow
	topWorkflow := filepath.Join(workflowsDir, "main.md")
	topContent := `---
description: Main workflow
imports:
  - shared/intermediate.md
---
# Main`
	if err := os.WriteFile(topWorkflow, []byte(topContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Build dependency graph
	graph := NewDependencyGraph(workflowsDir)
	compiler := workflow.NewCompiler(false, "", "test")
	if err := graph.BuildGraph(compiler); err != nil {
		t.Fatalf("BuildGraph() error = %v", err)
	}

	// Test: Modifying the base workflow should transitively affect the top-level workflow
	t.Run("nested import modification affects top-level workflow", func(t *testing.T) {
		affected := graph.GetAffectedWorkflows(baseWorkflow)

		// Should find the top-level workflow through the intermediate workflow
		if len(affected) != 1 {
			t.Errorf("GetAffectedWorkflows() returned %d workflows, want 1", len(affected))
		}

		if len(affected) > 0 && affected[0] != topWorkflow {
			t.Errorf("GetAffectedWorkflows() = %v, want [%s]", affected, topWorkflow)
		}
	})
}
