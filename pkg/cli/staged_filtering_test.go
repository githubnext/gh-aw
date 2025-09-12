package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestCheckIfWorkflowIsStaged(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir := t.TempDir()

	t.Run("staged true as boolean", func(t *testing.T) {
		// Create aw_info.json with staged: true as boolean
		infoData := map[string]interface{}{
			"engine_id":     "claude",
			"staged":        true,
			"workflow_name": "test-workflow",
		}
		infoBytes, _ := json.Marshal(infoData)
		infoPath := filepath.Join(tmpDir, "aw_info_staged_true.json")
		err := os.WriteFile(infoPath, infoBytes, 0644)
		if err != nil {
			t.Fatalf("Failed to write test file: %v", err)
		}

		result := checkIfWorkflowIsStaged(infoPath, false)
		if !result {
			t.Error("Expected checkIfWorkflowIsStaged to return true for staged: true")
		}
	})

	t.Run("staged false as boolean", func(t *testing.T) {
		// Create aw_info.json with staged: false as boolean
		infoData := map[string]interface{}{
			"engine_id":     "claude",
			"staged":        false,
			"workflow_name": "test-workflow",
		}
		infoBytes, _ := json.Marshal(infoData)
		infoPath := filepath.Join(tmpDir, "aw_info_staged_false.json")
		err := os.WriteFile(infoPath, infoBytes, 0644)
		if err != nil {
			t.Fatalf("Failed to write test file: %v", err)
		}

		result := checkIfWorkflowIsStaged(infoPath, false)
		if result {
			t.Error("Expected checkIfWorkflowIsStaged to return false for staged: false")
		}
	})

	t.Run("staged true as string", func(t *testing.T) {
		// Create aw_info.json with staged: "true" as string
		infoData := map[string]interface{}{
			"engine_id":     "claude",
			"staged":        "true",
			"workflow_name": "test-workflow",
		}
		infoBytes, _ := json.Marshal(infoData)
		infoPath := filepath.Join(tmpDir, "aw_info_staged_string_true.json")
		err := os.WriteFile(infoPath, infoBytes, 0644)
		if err != nil {
			t.Fatalf("Failed to write test file: %v", err)
		}

		result := checkIfWorkflowIsStaged(infoPath, false)
		if !result {
			t.Error("Expected checkIfWorkflowIsStaged to return true for staged: \"true\"")
		}
	})

	t.Run("staged false as string", func(t *testing.T) {
		// Create aw_info.json with staged: "false" as string
		infoData := map[string]interface{}{
			"engine_id":     "claude",
			"staged":        "false",
			"workflow_name": "test-workflow",
		}
		infoBytes, _ := json.Marshal(infoData)
		infoPath := filepath.Join(tmpDir, "aw_info_staged_string_false.json")
		err := os.WriteFile(infoPath, infoBytes, 0644)
		if err != nil {
			t.Fatalf("Failed to write test file: %v", err)
		}

		result := checkIfWorkflowIsStaged(infoPath, false)
		if result {
			t.Error("Expected checkIfWorkflowIsStaged to return false for staged: \"false\"")
		}
	})

	t.Run("no staged field", func(t *testing.T) {
		// Create aw_info.json without staged field
		infoData := map[string]interface{}{
			"engine_id":     "claude",
			"workflow_name": "test-workflow",
		}
		infoBytes, _ := json.Marshal(infoData)
		infoPath := filepath.Join(tmpDir, "aw_info_no_staged.json")
		err := os.WriteFile(infoPath, infoBytes, 0644)
		if err != nil {
			t.Fatalf("Failed to write test file: %v", err)
		}

		result := checkIfWorkflowIsStaged(infoPath, false)
		if result {
			t.Error("Expected checkIfWorkflowIsStaged to return false when staged field is missing")
		}
	})

	t.Run("missing file", func(t *testing.T) {
		// Test with non-existent file
		nonExistentPath := filepath.Join(tmpDir, "nonexistent.json")
		result := checkIfWorkflowIsStaged(nonExistentPath, false)
		if result {
			t.Error("Expected checkIfWorkflowIsStaged to return false for missing file")
		}
	})

	t.Run("invalid json", func(t *testing.T) {
		// Create invalid JSON file
		invalidPath := filepath.Join(tmpDir, "invalid.json")
		err := os.WriteFile(invalidPath, []byte("invalid json content"), 0644)
		if err != nil {
			t.Fatalf("Failed to write test file: %v", err)
		}

		result := checkIfWorkflowIsStaged(invalidPath, false)
		if result {
			t.Error("Expected checkIfWorkflowIsStaged to return false for invalid JSON")
		}
	})

	t.Run("staged as directory with nested file", func(t *testing.T) {
		// Create a directory with the same name and nested aw_info.json
		dirPath := filepath.Join(tmpDir, "aw_info_dir")
		err := os.MkdirAll(dirPath, 0755)
		if err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}

		// Create nested aw_info.json with staged: true
		infoData := map[string]interface{}{
			"engine_id":     "claude",
			"staged":        true,
			"workflow_name": "test-workflow",
		}
		infoBytes, _ := json.Marshal(infoData)
		nestedPath := filepath.Join(dirPath, "aw_info.json")
		err = os.WriteFile(nestedPath, infoBytes, 0644)
		if err != nil {
			t.Fatalf("Failed to write nested test file: %v", err)
		}

		result := checkIfWorkflowIsStaged(dirPath, false)
		if !result {
			t.Error("Expected checkIfWorkflowIsStaged to return true for nested staged: true")
		}
	})
}
