package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/githubnext/gh-aw/pkg/console"
)

func TestStatusWorkflows_JSONOutput(t *testing.T) {
	// Save current directory
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}

	// Change to repository root
	repoRoot := filepath.Join(originalDir, "..", "..")
	if err := os.Chdir(repoRoot); err != nil {
		t.Fatalf("Failed to change to repository root: %v", err)
	}
	defer os.Chdir(originalDir)

	// Test JSON output without pattern
	t.Run("JSON output without pattern", func(t *testing.T) {
		err := StatusWorkflows("", false, true)
		if err != nil {
			t.Errorf("StatusWorkflows with JSON flag failed: %v", err)
		}
		// Note: We can't easily capture stdout in this test,
		// but we verify it doesn't error
	})

	// Test JSON output with pattern
	t.Run("JSON output with pattern", func(t *testing.T) {
		err := StatusWorkflows("smoke", false, true)
		if err != nil {
			t.Errorf("StatusWorkflows with JSON flag and pattern failed: %v", err)
		}
	})
}

func TestWorkflowStatus_JSONMarshaling(t *testing.T) {
	// Test that WorkflowStatus can be marshaled to JSON
	status := WorkflowStatus{
		Workflow:      "test-workflow",
		Agent:         "copilot",
		Compiled:      "Yes",
		Status:        "active",
		TimeRemaining: "N/A",
		Frontmatter: map[string]any{
			"on":     "push",
			"engine": "copilot",
		},
		Prompt: "# Test Workflow\n\nThis is a test prompt.",
	}

	jsonBytes, err := json.Marshal(status)
	if err != nil {
		t.Fatalf("Failed to marshal WorkflowStatus: %v", err)
	}

	// Verify JSON contains expected fields
	var unmarshaled map[string]interface{}
	if err := json.Unmarshal(jsonBytes, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	if unmarshaled["workflow"] != "test-workflow" {
		t.Errorf("Expected workflow='test-workflow', got %v", unmarshaled["workflow"])
	}
	if unmarshaled["agent"] != "copilot" {
		t.Errorf("Expected agent='copilot', got %v", unmarshaled["agent"])
	}
	if unmarshaled["compiled"] != "Yes" {
		t.Errorf("Expected compiled='Yes', got %v", unmarshaled["compiled"])
	}
	if unmarshaled["status"] != "active" {
		t.Errorf("Expected status='active', got %v", unmarshaled["status"])
	}
	if unmarshaled["time_remaining"] != "N/A" {
		t.Errorf("Expected time_remaining='N/A', got %v", unmarshaled["time_remaining"])
	}

	// Verify frontmatter field is included
	frontmatter, ok := unmarshaled["frontmatter"].(map[string]interface{})
	if !ok {
		t.Fatalf("Expected frontmatter to be a map, got %T", unmarshaled["frontmatter"])
	}
	if frontmatter["on"] != "push" {
		t.Errorf("Expected frontmatter.on='push', got %v", frontmatter["on"])
	}
	if frontmatter["engine"] != "copilot" {
		t.Errorf("Expected frontmatter.engine='copilot', got %v", frontmatter["engine"])
	}

	// Verify prompt field is included
	prompt, ok := unmarshaled["prompt"].(string)
	if !ok {
		t.Fatalf("Expected prompt to be a string, got %T", unmarshaled["prompt"])
	}
	if prompt != "# Test Workflow\n\nThis is a test prompt." {
		t.Errorf("Expected prompt to match, got %s", prompt)
	}
}

// TestStatusCommand_JSONOutputValidation tests that the status command with --json flag returns valid JSON
func TestStatusCommand_JSONOutputValidation(t *testing.T) {
	// Skip if the binary doesn't exist
	binaryPath := "../../gh-aw"
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		t.Skip("Skipping test: gh-aw binary not found. Run 'make build' first.")
	}

	// Get the current directory for proper path resolution
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}

	// Change to repository root
	repoRoot := filepath.Join(originalDir, "..", "..")
	if err := os.Chdir(repoRoot); err != nil {
		t.Fatalf("Failed to change to repository root: %v", err)
	}
	defer os.Chdir(originalDir)

	// Run the status command with --json flag
	cmd := exec.Command(filepath.Join(originalDir, binaryPath), "status", "--json")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()
	if err != nil {
		t.Logf("Command stderr: %s", stderr.String())
		t.Fatalf("Failed to run status command: %v", err)
	}

	// Verify the output is valid JSON
	output := stdout.String()
	if output == "" {
		t.Fatal("Expected non-empty JSON output")
	}

	// Try to parse as JSON array
	var statuses []WorkflowStatus
	if err := json.Unmarshal([]byte(output), &statuses); err != nil {
		t.Fatalf("Output is not valid JSON: %v\nOutput: %s", err, output)
	}

	// Verify we got an array (even if empty)
	if statuses == nil {
		t.Error("Expected JSON array, got nil")
	}

	// If we have workflows, verify structure
	if len(statuses) > 0 {
		firstStatus := statuses[0]

		// Verify all required fields are present
		if firstStatus.Workflow == "" {
			t.Error("Expected 'workflow' field to be non-empty")
		}
		if firstStatus.Agent == "" {
			t.Error("Expected 'agent' field to be non-empty")
		}
		if firstStatus.Compiled == "" {
			t.Error("Expected 'compiled' field to be non-empty")
		}
		if firstStatus.Status == "" {
			t.Error("Expected 'status' field to be non-empty")
		}
		if firstStatus.TimeRemaining == "" {
			t.Error("Expected 'time_remaining' field to be non-empty")
		}

		t.Logf("Successfully parsed %d workflow status entries", len(statuses))
		t.Logf("First entry: workflow=%s, agent=%s, compiled=%s",
			firstStatus.Workflow, firstStatus.Agent, firstStatus.Compiled)
	}
}

// TestStatusCommand_JSONOutputWithPattern tests that status --json works with a pattern filter
func TestStatusCommand_JSONOutputWithPattern(t *testing.T) {
	// Skip if the binary doesn't exist
	binaryPath := "../../gh-aw"
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		t.Skip("Skipping test: gh-aw binary not found. Run 'make build' first.")
	}

	// Get the current directory for proper path resolution
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}

	// Change to repository root
	repoRoot := filepath.Join(originalDir, "..", "..")
	if err := os.Chdir(repoRoot); err != nil {
		t.Fatalf("Failed to change to repository root: %v", err)
	}
	defer os.Chdir(originalDir)

	// Run the status command with --json flag and pattern
	cmd := exec.Command(filepath.Join(originalDir, binaryPath), "status", "smoke", "--json")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()
	if err != nil {
		t.Logf("Command stderr: %s", stderr.String())
		t.Fatalf("Failed to run status command with pattern: %v", err)
	}

	// Verify the output is valid JSON
	output := stdout.String()
	if output == "" {
		t.Fatal("Expected non-empty JSON output")
	}

	// Try to parse as JSON array
	var statuses []WorkflowStatus
	if err := json.Unmarshal([]byte(output), &statuses); err != nil {
		t.Fatalf("Output is not valid JSON: %v\nOutput: %s", err, output)
	}

	// All filtered results should contain "smoke" in the workflow name
	for _, status := range statuses {
		if !strings.Contains(strings.ToLower(status.Workflow), "smoke") {
			t.Errorf("Expected workflow name to contain 'smoke', got: %s", status.Workflow)
		}
	}

	t.Logf("Successfully parsed %d filtered workflow status entries", len(statuses))
}

// TestStatusCommand_JSONOutputIncludesFrontmatterAndPrompt tests that frontmatter and prompt are included in JSON output
func TestStatusCommand_JSONOutputIncludesFrontmatterAndPrompt(t *testing.T) {
	// Skip if the binary doesn't exist
	binaryPath := "../../gh-aw"
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		t.Skip("Skipping test: gh-aw binary not found. Run 'make build' first.")
	}

	// Get the current directory for proper path resolution
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}

	// Change to repository root
	repoRoot := filepath.Join(originalDir, "..", "..")
	if err := os.Chdir(repoRoot); err != nil {
		t.Fatalf("Failed to change to repository root: %v", err)
	}
	defer os.Chdir(originalDir)

	// Run the status command with --json flag
	cmd := exec.Command(filepath.Join(originalDir, binaryPath), "status", "--json")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()
	if err != nil {
		t.Logf("Command stderr: %s", stderr.String())
		t.Fatalf("Failed to run status command: %v", err)
	}

	// Verify the output is valid JSON
	output := stdout.String()
	if output == "" {
		t.Fatal("Expected non-empty JSON output")
	}

	// Try to parse as JSON array
	var statuses []WorkflowStatus
	if err := json.Unmarshal([]byte(output), &statuses); err != nil {
		t.Fatalf("Output is not valid JSON: %v\nOutput: %s", err, output)
	}

	// If we have workflows, verify frontmatter and prompt fields are present
	if len(statuses) > 0 {
		firstStatus := statuses[0]

		// Verify frontmatter field is present and is a map
		if firstStatus.Frontmatter == nil {
			t.Error("Expected 'frontmatter' field to be present")
		} else {
			t.Logf("Frontmatter keys for workflow '%s': %v", firstStatus.Workflow, getKeys(firstStatus.Frontmatter))

			// Verify frontmatter contains common fields
			if _, ok := firstStatus.Frontmatter["on"]; !ok {
				t.Logf("Warning: frontmatter missing 'on' field for workflow '%s'", firstStatus.Workflow)
			}
		}

		// Verify prompt field is present and non-empty
		if firstStatus.Prompt == "" {
			t.Error("Expected 'prompt' field to be non-empty")
		} else {
			t.Logf("Prompt length for workflow '%s': %d characters", firstStatus.Workflow, len(firstStatus.Prompt))
			// Verify prompt contains markdown content (typically starts with #)
			if !strings.Contains(firstStatus.Prompt, "#") {
				t.Logf("Warning: prompt doesn't appear to contain markdown headers")
			}
		}

		t.Logf("Successfully verified frontmatter and prompt fields for %d workflow(s)", len(statuses))
	} else {
		t.Skip("No workflows found to test frontmatter and prompt fields")
	}
}

// Helper function to get keys from a map
func getKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// TestWorkflowStatus_ConsoleRendering tests that WorkflowStatus uses console.RenderStruct correctly
func TestWorkflowStatus_ConsoleRendering(t *testing.T) {
	// Create test data
	statuses := []WorkflowStatus{
		{
			Workflow:      "test-workflow-1",
			Agent:         "copilot",
			Compiled:      "Yes",
			Status:        "active",
			TimeRemaining: "N/A",
		},
		{
			Workflow:      "test-workflow-2",
			Agent:         "claude",
			Compiled:      "No",
			Status:        "disabled",
			TimeRemaining: "2h 30m",
		},
	}

	// Render using console.RenderStruct
	output := console.RenderStruct(statuses)

	// Verify the output contains table headers from console tags
	expectedHeaders := []string{"Workflow", "Agent", "Compiled", "Status", "Time Remaining"}
	for _, header := range expectedHeaders {
		if !strings.Contains(output, header) {
			t.Errorf("Expected output to contain header '%s', got:\n%s", header, output)
		}
	}

	// Verify the output contains the data values
	expectedValues := []string{
		"test-workflow-1", "copilot", "Yes", "active",
		"test-workflow-2", "claude", "No", "disabled", "2h 30m",
	}
	for _, value := range expectedValues {
		if !strings.Contains(output, value) {
			t.Errorf("Expected output to contain value '%s', got:\n%s", value, output)
		}
	}

	// Verify it's formatted as a table (contains separators)
	if !strings.Contains(output, "-") {
		t.Error("Expected table output to contain separator lines")
	}
}
