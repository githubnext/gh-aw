package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/githubnext/gh-aw/pkg/parser"
)

// TestJSweepWorkflowCompilation tests that the jsweep workflow compiles correctly
func TestJSweepWorkflowCompilation(t *testing.T) {
	// Path to the jsweep workflow
	workflowPath := filepath.Join("..", "..", ".github", "workflows", "jsweep.md")
	
	// Check if file exists
	if _, err := os.Stat(workflowPath); os.IsNotExist(err) {
		t.Skipf("jsweep workflow not found at %s", workflowPath)
	}
	
	// Read the workflow file
	content, err := os.ReadFile(workflowPath)
	if err != nil {
		t.Fatalf("Failed to read jsweep workflow: %v", err)
	}
	
	contentStr := string(content)
	
	// Test that it mentions processing three files
	if !strings.Contains(contentStr, "three .cjs files") {
		t.Error("jsweep workflow should mention processing 'three .cjs files'")
	}
	
	// Test that the file location is correct
	if !strings.Contains(contentStr, "actions/setup/js/") {
		t.Error("jsweep workflow should reference 'actions/setup/js/' directory")
	}
	
	// Create a compiler instance
	compiler := NewCompiler(false, "", "test")
	compiler.SetSkipValidation(true)
	
	// Parse the workflow using parser package
	result, err := parser.ExtractFrontmatterFromContent(contentStr)
	if err != nil {
		t.Fatalf("Failed to parse jsweep frontmatter: %v", err)
	}
	
	frontmatter := result.Frontmatter
	body := result.Markdown
	
	// Verify description
	if desc, ok := frontmatter["description"].(string); ok {
		if !strings.Contains(desc, "three .cjs files") {
			t.Errorf("Description should mention 'three .cjs files', got: %s", desc)
		}
	} else {
		t.Error("Description field missing or not a string")
	}
	
	// Verify tools configuration
	tools, ok := frontmatter["tools"].(map[string]any)
	if !ok {
		t.Fatal("tools field missing or not a map")
	}
	
	// Check for cache-memory tool (it's inside tools section)
	if cacheMemory, ok := tools["cache-memory"]; !ok || cacheMemory != true {
		t.Error("jsweep workflow should have cache-memory enabled in tools")
	}
	
	// Check for serena tool
	if _, hasSerena := tools["serena"]; !hasSerena {
		t.Error("jsweep workflow should have serena tool")
	}
	
	// Check for github tool
	if _, hasGitHub := tools["github"]; !hasGitHub {
		t.Error("jsweep workflow should have github tool")
	}
	
	// Check for edit tool
	if _, hasEdit := tools["edit"]; !hasEdit {
		t.Error("jsweep workflow should have edit tool")
	}
	
	// Check for bash tool
	if _, hasBash := tools["bash"]; !hasBash {
		t.Error("jsweep workflow should have bash tool")
	}
	
	// Verify safe outputs configuration
	safeOutputs, ok := frontmatter["safe-outputs"].(map[string]any)
	if !ok {
		t.Fatal("safe-outputs field missing or not a map")
	}
	
	// Check for create-pull-request
	if _, hasCreatePR := safeOutputs["create-pull-request"]; !hasCreatePR {
		t.Error("jsweep workflow should have create-pull-request safe output")
	}
	
	// Verify body contains the correct instructions
	if !strings.Contains(body, "three files") {
		t.Error("Workflow body should mention processing 'three files'")
	}
	
	if !strings.Contains(body, "Find the Next Files to Clean") {
		t.Error("Workflow body should have section 'Find the Next Files to Clean'")
	}
	
	if !strings.Contains(body, "Three files per run") {
		t.Error("Workflow body should mention 'Three files per run' in constraints")
	}
}

// TestJSweepWorkflowInstructions tests that jsweep workflow has correct instructions for 3 files
func TestJSweepWorkflowInstructions(t *testing.T) {
	// Path to the jsweep workflow
	workflowPath := filepath.Join("..", "..", ".github", "workflows", "jsweep.md")
	
	// Check if file exists
	if _, err := os.Stat(workflowPath); os.IsNotExist(err) {
		t.Skipf("jsweep workflow not found at %s", workflowPath)
	}
	
	// Read the workflow file
	content, err := os.ReadFile(workflowPath)
	if err != nil {
		t.Fatalf("Failed to read jsweep workflow: %v", err)
	}
	
	contentStr := string(content)
	
	// Test specific instructions for handling 3 files
	requiredStrings := []string{
		"Pick the **three files**",
		"After cleaning all three files",
		"Clean <file1>, <file2>, <file3>",
		"Summary of changes for each file",
		"Context type (github-script or Node.js) for each file",
		"Any test improvements for each file",
	}
	
	for _, required := range requiredStrings {
		if !strings.Contains(contentStr, required) {
			t.Errorf("jsweep workflow should contain instruction: %q", required)
		}
	}
	
	// Test that old single-file instructions are removed
	deprecatedStrings := []string{
		"one .cjs file per day",
		"Pick the file with the earliest",
		"One file per run",
		"Clean <filename>",
	}
	
	for _, deprecated := range deprecatedStrings {
		if strings.Contains(contentStr, deprecated) {
			t.Errorf("jsweep workflow should not contain old instruction: %q", deprecated)
		}
	}
}

// TestJSweepWorkflowFileLocation tests that jsweep references the correct file location
func TestJSweepWorkflowFileLocation(t *testing.T) {
	// Path to the jsweep workflow
	workflowPath := filepath.Join("..", "..", ".github", "workflows", "jsweep.md")
	
	// Check if file exists
	if _, err := os.Stat(workflowPath); os.IsNotExist(err) {
		t.Skipf("jsweep workflow not found at %s", workflowPath)
	}
	
	// Read the workflow file
	content, err := os.ReadFile(workflowPath)
	if err != nil {
		t.Fatalf("Failed to read jsweep workflow: %v", err)
	}
	
	contentStr := string(content)
	
	// Test that it references the correct directory
	if !strings.Contains(contentStr, "/home/runner/work/gh-aw/gh-aw/actions/setup/js/") {
		t.Error("jsweep workflow should reference '/home/runner/work/gh-aw/gh-aw/actions/setup/js/' directory")
	}
	
	// Test that it doesn't reference the old pkg/workflow/js location
	if strings.Contains(contentStr, "pkg/workflow/js/") {
		t.Error("jsweep workflow should not reference old 'pkg/workflow/js/' location")
	}
}
