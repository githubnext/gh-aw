package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestSafeOutputJSONMarshaling tests that SafeOutput marshals and unmarshals correctly
func TestSafeOutputJSONMarshaling(t *testing.T) {
	tests := []struct {
		name       string
		safeOutput SafeOutput
		wantJSON   string
	}{
		{
			name: "full safe output",
			safeOutput: SafeOutput{
				Type: "issue",
				ID:   123,
				URL:  "https://github.com/owner/repo/issues/123",
				Metadata: map[string]interface{}{
					"title": "Test Issue",
					"state": "open",
				},
			},
			wantJSON: `{"type":"issue","id":123,"url":"https://github.com/owner/repo/issues/123","metadata":{"state":"open","title":"Test Issue"}}`,
		},
		{
			name: "minimal safe output",
			safeOutput: SafeOutput{
				Type: "discussion",
			},
			wantJSON: `{"type":"discussion"}`,
		},
		{
			name: "safe output with metadata only",
			safeOutput: SafeOutput{
				Type: "comment",
				Metadata: map[string]interface{}{
					"body": "Test comment",
				},
			},
			wantJSON: `{"type":"comment","metadata":{"body":"Test comment"}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test marshaling
			jsonBytes, err := json.Marshal(tt.safeOutput)
			if err != nil {
				t.Fatalf("Failed to marshal SafeOutput: %v", err)
			}

			// Unmarshal both expected and actual to compare as maps (order-independent)
			var expected, actual map[string]interface{}
			if err := json.Unmarshal([]byte(tt.wantJSON), &expected); err != nil {
				t.Fatalf("Failed to unmarshal expected JSON: %v", err)
			}
			if err := json.Unmarshal(jsonBytes, &actual); err != nil {
				t.Fatalf("Failed to unmarshal actual JSON: %v", err)
			}

			// Compare the maps
			if !mapsEqual(expected, actual) {
				t.Errorf("JSON mismatch:\nExpected: %s\nActual: %s", tt.wantJSON, string(jsonBytes))
			}

			// Test unmarshaling
			var unmarshaled SafeOutput
			if err := json.Unmarshal(jsonBytes, &unmarshaled); err != nil {
				t.Fatalf("Failed to unmarshal SafeOutput: %v", err)
			}

			// Verify fields
			if unmarshaled.Type != tt.safeOutput.Type {
				t.Errorf("Type mismatch: got %q, want %q", unmarshaled.Type, tt.safeOutput.Type)
			}
			if unmarshaled.ID != tt.safeOutput.ID {
				t.Errorf("ID mismatch: got %d, want %d", unmarshaled.ID, tt.safeOutput.ID)
			}
			if unmarshaled.URL != tt.safeOutput.URL {
				t.Errorf("URL mismatch: got %q, want %q", unmarshaled.URL, tt.safeOutput.URL)
			}
		})
	}
}

// TestAgenticRunInfoJSONMarshaling tests that AgenticRunInfo marshals and unmarshals correctly
func TestAgenticRunInfoJSONMarshaling(t *testing.T) {
	tests := []struct {
		name     string
		runInfo  AgenticRunInfo
		wantJSON string
	}{
		{
			name: "full agentic run info",
			runInfo: AgenticRunInfo{
				TotalTurns:   5,
				TokensUsed:   1500,
				Duration:     time.Minute * 3,
				Engine:       "copilot",
				ModelVersion: "gpt-4",
			},
			wantJSON: `{"total_turns":5,"tokens_used":1500,"duration":180000000000,"engine":"copilot","model_version":"gpt-4"}`,
		},
		{
			name: "minimal agentic run info",
			runInfo: AgenticRunInfo{
				TotalTurns: 1,
			},
			wantJSON: `{"total_turns":1}`,
		},
		{
			name: "partial agentic run info",
			runInfo: AgenticRunInfo{
				TotalTurns: 3,
				TokensUsed: 500,
				Engine:     "claude",
			},
			wantJSON: `{"total_turns":3,"tokens_used":500,"engine":"claude"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test marshaling
			jsonBytes, err := json.Marshal(tt.runInfo)
			if err != nil {
				t.Fatalf("Failed to marshal AgenticRunInfo: %v", err)
			}

			// Unmarshal both expected and actual to compare as maps (order-independent)
			var expected, actual map[string]interface{}
			if err := json.Unmarshal([]byte(tt.wantJSON), &expected); err != nil {
				t.Fatalf("Failed to unmarshal expected JSON: %v", err)
			}
			if err := json.Unmarshal(jsonBytes, &actual); err != nil {
				t.Fatalf("Failed to unmarshal actual JSON: %v", err)
			}

			// Compare the maps
			if !mapsEqual(expected, actual) {
				t.Errorf("JSON mismatch:\nExpected: %s\nActual: %s", tt.wantJSON, string(jsonBytes))
			}

			// Test unmarshaling
			var unmarshaled AgenticRunInfo
			if err := json.Unmarshal(jsonBytes, &unmarshaled); err != nil {
				t.Fatalf("Failed to unmarshal AgenticRunInfo: %v", err)
			}

			// Verify fields
			if unmarshaled.TotalTurns != tt.runInfo.TotalTurns {
				t.Errorf("TotalTurns mismatch: got %d, want %d", unmarshaled.TotalTurns, tt.runInfo.TotalTurns)
			}
			if unmarshaled.TokensUsed != tt.runInfo.TokensUsed {
				t.Errorf("TokensUsed mismatch: got %d, want %d", unmarshaled.TokensUsed, tt.runInfo.TokensUsed)
			}
			if unmarshaled.Duration != tt.runInfo.Duration {
				t.Errorf("Duration mismatch: got %v, want %v", unmarshaled.Duration, tt.runInfo.Duration)
			}
			if unmarshaled.Engine != tt.runInfo.Engine {
				t.Errorf("Engine mismatch: got %q, want %q", unmarshaled.Engine, tt.runInfo.Engine)
			}
			if unmarshaled.ModelVersion != tt.runInfo.ModelVersion {
				t.Errorf("ModelVersion mismatch: got %q, want %q", unmarshaled.ModelVersion, tt.runInfo.ModelVersion)
			}
		})
	}
}

// TestWorkflowTrialResultJSONMarshaling tests that WorkflowTrialResult marshals and unmarshals correctly
func TestWorkflowTrialResultJSONMarshaling(t *testing.T) {
	timestamp := time.Date(2024, 1, 15, 12, 30, 0, 0, time.UTC)

	result := WorkflowTrialResult{
		WorkflowName: "test-workflow",
		RunID:        "12345",
		SafeOutputs: []SafeOutput{
			{
				Type: "issue",
				ID:   123,
				URL:  "https://github.com/owner/repo/issues/123",
			},
			{
				Type: "discussion",
				ID:   456,
				URL:  "https://github.com/owner/repo/discussions/456",
			},
		},
		AgenticRunInfo: &AgenticRunInfo{
			TotalTurns:   3,
			TokensUsed:   1000,
			Duration:     time.Minute * 2,
			Engine:       "copilot",
			ModelVersion: "gpt-4",
		},
		AdditionalArtifacts: map[string]interface{}{
			"logs": "some log content",
		},
		Timestamp: timestamp,
	}

	// Test marshaling
	jsonBytes, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("Failed to marshal WorkflowTrialResult: %v", err)
	}

	// Test unmarshaling
	var unmarshaled WorkflowTrialResult
	if err := json.Unmarshal(jsonBytes, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal WorkflowTrialResult: %v", err)
	}

	// Verify fields
	if unmarshaled.WorkflowName != result.WorkflowName {
		t.Errorf("WorkflowName mismatch: got %q, want %q", unmarshaled.WorkflowName, result.WorkflowName)
	}
	if unmarshaled.RunID != result.RunID {
		t.Errorf("RunID mismatch: got %q, want %q", unmarshaled.RunID, result.RunID)
	}
	if len(unmarshaled.SafeOutputs) != len(result.SafeOutputs) {
		t.Errorf("SafeOutputs length mismatch: got %d, want %d", len(unmarshaled.SafeOutputs), len(result.SafeOutputs))
	}
	if unmarshaled.AgenticRunInfo == nil {
		t.Fatal("AgenticRunInfo is nil after unmarshaling")
	}
	if unmarshaled.AgenticRunInfo.TotalTurns != result.AgenticRunInfo.TotalTurns {
		t.Errorf("AgenticRunInfo.TotalTurns mismatch: got %d, want %d", unmarshaled.AgenticRunInfo.TotalTurns, result.AgenticRunInfo.TotalTurns)
	}
	if !unmarshaled.Timestamp.Equal(result.Timestamp) {
		t.Errorf("Timestamp mismatch: got %v, want %v", unmarshaled.Timestamp, result.Timestamp)
	}
}

// TestParseSafeOutputsArtifact tests the parseSafeOutputsArtifact function
func TestParseSafeOutputsArtifact(t *testing.T) {
	tests := []struct {
		name         string
		jsonContent  string
		wantCount    int
		wantFirstID  int
		wantFirstURL string
	}{
		{
			name: "single issue output",
			jsonContent: `{
				"create_issue": {
					"id": 123,
					"url": "https://github.com/owner/repo/issues/123",
					"title": "Test Issue"
				}
			}`,
			wantCount:    1,
			wantFirstID:  123,
			wantFirstURL: "https://github.com/owner/repo/issues/123",
		},
		{
			name: "multiple outputs",
			jsonContent: `{
				"create_issue": [
					{
						"id": 123,
						"url": "https://github.com/owner/repo/issues/123"
					},
					{
						"id": 124,
						"url": "https://github.com/owner/repo/issues/124"
					}
				]
			}`,
			wantCount:    2,
			wantFirstID:  123,
			wantFirstURL: "https://github.com/owner/repo/issues/123",
		},
		{
			name: "empty artifact",
			jsonContent: `{
			}`,
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp file with JSON content
			tempDir := t.TempDir()
			filePath := filepath.Join(tempDir, "agent_output.json")
			if err := os.WriteFile(filePath, []byte(tt.jsonContent), 0644); err != nil {
				t.Fatalf("Failed to write test file: %v", err)
			}

			// Parse the artifact
			outputs := parseSafeOutputsArtifact(filePath, false)

			// Verify count
			if len(outputs) != tt.wantCount {
				t.Errorf("Output count mismatch: got %d, want %d", len(outputs), tt.wantCount)
			}

			// Verify first output if expected
			if tt.wantCount > 0 {
				if outputs[0].ID != tt.wantFirstID {
					t.Errorf("First output ID mismatch: got %d, want %d", outputs[0].ID, tt.wantFirstID)
				}
				if outputs[0].URL != tt.wantFirstURL {
					t.Errorf("First output URL mismatch: got %q, want %q", outputs[0].URL, tt.wantFirstURL)
				}
			}
		})
	}
}

// TestParseAgenticRunInfoArtifact tests the parseAgenticRunInfoArtifact function
func TestParseAgenticRunInfoArtifact(t *testing.T) {
	tests := []struct {
		name        string
		jsonContent string
		wantNil     bool
		wantTurns   int
		wantTokens  int
		wantEngine  string
	}{
		{
			name: "full run info",
			jsonContent: `{
				"total_turns": 5,
				"tokens_used": 1500,
				"duration": 180000000000,
				"engine": "copilot",
				"model_version": "gpt-4"
			}`,
			wantNil:    false,
			wantTurns:  5,
			wantTokens: 1500,
			wantEngine: "copilot",
		},
		{
			name: "minimal run info",
			jsonContent: `{
				"total_turns": 1
			}`,
			wantNil:   false,
			wantTurns: 1,
		},
		{
			name:        "invalid JSON",
			jsonContent: `{invalid json}`,
			wantNil:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp file with JSON content
			tempDir := t.TempDir()
			filePath := filepath.Join(tempDir, "aw_info.json")
			if err := os.WriteFile(filePath, []byte(tt.jsonContent), 0644); err != nil {
				t.Fatalf("Failed to write test file: %v", err)
			}

			// Parse the artifact
			runInfo := parseAgenticRunInfoArtifact(filePath, false)

			// Verify result
			if tt.wantNil {
				if runInfo != nil {
					t.Errorf("Expected nil, got %+v", runInfo)
				}
				return
			}

			if runInfo == nil {
				t.Fatal("Expected non-nil result, got nil")
			}

			if runInfo.TotalTurns != tt.wantTurns {
				t.Errorf("TotalTurns mismatch: got %d, want %d", runInfo.TotalTurns, tt.wantTurns)
			}
			if runInfo.TokensUsed != tt.wantTokens {
				t.Errorf("TokensUsed mismatch: got %d, want %d", runInfo.TokensUsed, tt.wantTokens)
			}
			if runInfo.Engine != tt.wantEngine {
				t.Errorf("Engine mismatch: got %q, want %q", runInfo.Engine, tt.wantEngine)
			}
		})
	}
}

// mapsEqual compares two maps for equality (deep comparison)
func mapsEqual(a, b map[string]interface{}) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if bv, ok := b[k]; !ok {
			return false
		} else {
			// Handle nested maps
			if vm, ok := v.(map[string]interface{}); ok {
				if bvm, ok := bv.(map[string]interface{}); ok {
					if !mapsEqual(vm, bvm) {
						return false
					}
				} else {
					return false
				}
			} else if v != bv {
				return false
			}
		}
	}
	return true
}
