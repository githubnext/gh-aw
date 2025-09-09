package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAnalyzeSanitizationChanges(t *testing.T) {
	// Create a temporary directory for test artifacts
	tmpDir := t.TempDir()

	// Test case 1: No output files available
	t.Run("NoOutputFiles", func(t *testing.T) {
		run := WorkflowRun{DatabaseID: 12345}
		analysis, err := analyzeSanitizationChanges(tmpDir, run, false)

		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		if analysis.HasRawOutput || analysis.HasSanitizedOutput {
			t.Error("Expected no output files detected")
		}
		if analysis.TotalChanges != 0 {
			t.Error("Expected no changes when no output files exist")
		}
	})

	// Test case 2: Only sanitized output available
	t.Run("OnlySanitizedOutput", func(t *testing.T) {
		// Create only sanitized output
		sanitizedPath := filepath.Join(tmpDir, "safe_output.jsonl")
		sanitizedContent := `{"type":"text","text":"Hello world"}`
		if err := os.WriteFile(sanitizedPath, []byte(sanitizedContent), 0644); err != nil {
			t.Fatal(err)
		}

		run := WorkflowRun{DatabaseID: 12346}
		analysis, err := analyzeSanitizationChanges(tmpDir, run, false)

		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		if !analysis.HasSanitizedOutput {
			t.Error("Expected sanitized output to be detected")
		}
		if analysis.HasRawOutput {
			t.Error("Expected no raw output")
		}
		if analysis.TotalChanges != 0 {
			t.Error("Expected no changes when only sanitized output exists")
		}

		// Clean up
		os.Remove(sanitizedPath)
	})

	// Test case 3: Both outputs available with changes
	t.Run("BothOutputsWithChanges", func(t *testing.T) {
		// Create raw output with content that will be sanitized
		rawPath := filepath.Join(tmpDir, "agent_output.json")
		rawContent := `{
			"content": "Hello @user! This fixes #123 and check http://evil.com and https://github.com/repo"
		}`
		if err := os.WriteFile(rawPath, []byte(rawContent), 0644); err != nil {
			t.Fatal(err)
		}

		// Create sanitized output with expected sanitization changes
		sanitizedPath := filepath.Join(tmpDir, "safe_output.jsonl")
		sanitizedContent := `{"type":"text","text":"Hello ` + "`@user`" + `! This ` + "`fixes #123`" + ` and check (redacted) and https://github.com/repo"}`
		if err := os.WriteFile(sanitizedPath, []byte(sanitizedContent), 0644); err != nil {
			t.Fatal(err)
		}

		run := WorkflowRun{DatabaseID: 12347}
		analysis, err := analyzeSanitizationChanges(tmpDir, run, false)

		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		if !analysis.HasRawOutput || !analysis.HasSanitizedOutput {
			t.Error("Expected both output files to be detected")
		}
		if analysis.TotalChanges == 0 {
			t.Error("Expected to detect sanitization changes")
		}

		// Check that we detected expected change types
		expectedTypes := []string{"mention", "bot_trigger", "url"}
		for _, expectedType := range expectedTypes {
			if count, exists := analysis.ChangesByType[expectedType]; !exists || count == 0 {
				t.Errorf("Expected to detect %s changes, got count: %d", expectedType, count)
			}
		}

		// Clean up
		os.Remove(rawPath)
		os.Remove(sanitizedPath)
	})

	// Test case 4: Content truncation detection
	t.Run("ContentTruncation", func(t *testing.T) {
		// Create raw output
		rawPath := filepath.Join(tmpDir, "agent_output.json")
		longContent := strings.Repeat("long content ", 1000)
		rawContent := `{"content": "` + longContent + `"}`
		if err := os.WriteFile(rawPath, []byte(rawContent), 0644); err != nil {
			t.Fatal(err)
		}

		// Create sanitized output with truncation message
		sanitizedPath := filepath.Join(tmpDir, "safe_output.jsonl")
		sanitizedContent := `{"type":"text","text":"partial content\n[Content truncated due to length]"}`
		if err := os.WriteFile(sanitizedPath, []byte(sanitizedContent), 0644); err != nil {
			t.Fatal(err)
		}

		run := WorkflowRun{DatabaseID: 12348}
		analysis, err := analyzeSanitizationChanges(tmpDir, run, false)

		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		if !analysis.WasContentTruncated {
			t.Error("Expected content truncation to be detected")
		}
		if analysis.TruncationReason != "length" {
			t.Errorf("Expected truncation reason 'length', got: %s", analysis.TruncationReason)
		}

		// Clean up
		os.Remove(rawPath)
		os.Remove(sanitizedPath)
	})
}

func TestDetectMentionChanges(t *testing.T) {
	tests := []struct {
		name          string
		rawLine       string
		sanitizedLine string
		lineNumber    int
		expectedCount int
	}{
		{
			name:          "SingleMention",
			rawLine:       "Hello @user",
			sanitizedLine: "Hello `@user`",
			lineNumber:    1,
			expectedCount: 1,
		},
		{
			name:          "MultipleMentions",
			rawLine:       "Contact @user1 and @org/team",
			sanitizedLine: "Contact `@user1` and `@org/team`",
			lineNumber:    2,
			expectedCount: 2,
		},
		{
			name:          "NoChanges",
			rawLine:       "Hello world",
			sanitizedLine: "Hello world",
			lineNumber:    3,
			expectedCount: 0,
		},
		{
			name:          "AlreadyBackticked",
			rawLine:       "Check `@user` in code",
			sanitizedLine: "Check `@user` in code",
			lineNumber:    4,
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			changes := detectMentionChanges(tt.rawLine, tt.sanitizedLine, tt.lineNumber)
			if len(changes) != tt.expectedCount {
				t.Errorf("Expected %d changes, got %d", tt.expectedCount, len(changes))
			}

			for _, change := range changes {
				if change.Type != "mention" {
					t.Errorf("Expected change type 'mention', got '%s'", change.Type)
				}
				if change.LineNumber != tt.lineNumber {
					t.Errorf("Expected line number %d, got %d", tt.lineNumber, change.LineNumber)
				}
			}
		})
	}
}

func TestDetectBotTriggerChanges(t *testing.T) {
	tests := []struct {
		name          string
		rawLine       string
		sanitizedLine string
		lineNumber    int
		expectedCount int
	}{
		{
			name:          "SingleTrigger",
			rawLine:       "This fixes #123",
			sanitizedLine: "This `fixes #123`",
			lineNumber:    1,
			expectedCount: 1,
		},
		{
			name:          "MultipleTriggers",
			rawLine:       "fixes #123 and closes #456",
			sanitizedLine: "`fixes #123` and `closes #456`",
			lineNumber:    2,
			expectedCount: 2,
		},
		{
			name:          "CaseInsensitive",
			rawLine:       "FIXES #ABC",
			sanitizedLine: "`FIXES #ABC`",
			lineNumber:    3,
			expectedCount: 1,
		},
		{
			name:          "NoChanges",
			rawLine:       "No triggers here",
			sanitizedLine: "No triggers here",
			lineNumber:    4,
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			changes := detectBotTriggerChanges(tt.rawLine, tt.sanitizedLine, tt.lineNumber)
			if len(changes) != tt.expectedCount {
				t.Errorf("Expected %d changes, got %d", tt.expectedCount, len(changes))
			}

			for _, change := range changes {
				if change.Type != "bot_trigger" {
					t.Errorf("Expected change type 'bot_trigger', got '%s'", change.Type)
				}
				if change.LineNumber != tt.lineNumber {
					t.Errorf("Expected line number %d, got %d", tt.lineNumber, change.LineNumber)
				}
			}
		})
	}
}

func TestDetectURLChanges(t *testing.T) {
	tests := []struct {
		name          string
		rawLine       string
		sanitizedLine string
		lineNumber    int
		expectedCount int
	}{
		{
			name:          "HTTPRedacted",
			rawLine:       "Visit http://example.com",
			sanitizedLine: "Visit (redacted)",
			lineNumber:    1,
			expectedCount: 1,
		},
		{
			name:          "HTTPSPreserved",
			rawLine:       "Visit https://github.com",
			sanitizedLine: "Visit https://github.com",
			lineNumber:    2,
			expectedCount: 0,
		},
		{
			name:          "HTTPSRedacted",
			rawLine:       "Visit https://evil.com",
			sanitizedLine: "Visit (redacted)",
			lineNumber:    3,
			expectedCount: 1,
		},
		{
			name:          "MultipleURLs",
			rawLine:       "Links: http://bad.com and ftp://files.com",
			sanitizedLine: "Links: (redacted) and (redacted)",
			lineNumber:    4,
			expectedCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			changes := detectURLChanges(tt.rawLine, tt.sanitizedLine, tt.lineNumber)
			if len(changes) != tt.expectedCount {
				t.Errorf("Expected %d changes, got %d", tt.expectedCount, len(changes))
			}

			for _, change := range changes {
				if change.Type != "url" {
					t.Errorf("Expected change type 'url', got '%s'", change.Type)
				}
				if change.Sanitized != "(redacted)" {
					t.Errorf("Expected sanitized value '(redacted)', got '%s'", change.Sanitized)
				}
			}
		})
	}
}

func TestExtractContentFromJSON(t *testing.T) {
	tests := []struct {
		name           string
		jsonContent    string
		expectedOutput string
		expectError    bool
	}{
		{
			name:           "ContentField",
			jsonContent:    `{"content": "Hello world"}`,
			expectedOutput: "Hello world",
			expectError:    false,
		},
		{
			name:           "OutputField",
			jsonContent:    `{"output": "Test output"}`,
			expectedOutput: "Test output",
			expectError:    false,
		},
		{
			name:           "TextField",
			jsonContent:    `{"text": "Text content"}`,
			expectedOutput: "Text content",
			expectError:    false,
		},
		{
			name: "ItemsArray",
			jsonContent: `{
				"items": [
					{"type": "text", "text": "First part"},
					{"type": "text", "text": "Second part"},
					{"type": "other", "data": "ignored"}
				]
			}`,
			expectedOutput: "First partSecond part",
			expectError:    false,
		},
		{
			name:           "PlainJSON",
			jsonContent:    `{"unknown": "structure"}`,
			expectedOutput: `{"unknown": "structure"}`,
			expectError:    false,
		},
		{
			name:           "InvalidJSON",
			jsonContent:    `{invalid json`,
			expectedOutput: `{invalid json`,
			expectError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := extractContentFromJSON(tt.jsonContent)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}

			if result != tt.expectedOutput {
				t.Errorf("Expected output '%s', got '%s'", tt.expectedOutput, result)
			}
		})
	}
}

func TestExtractContentFromJSONL(t *testing.T) {
	tests := []struct {
		name           string
		jsonlContent   string
		expectedOutput string
		expectError    bool
	}{
		{
			name: "ValidJSONL",
			jsonlContent: `{"type":"text","text":"Line 1"}
{"type":"text","text":"Line 2"}
{"type":"other","data":"ignored"}`,
			expectedOutput: "Line 1Line 2",
			expectError:    false,
		},
		{
			name:           "EmptyLines",
			jsonlContent:   "\n\n",
			expectedOutput: "",
			expectError:    false,
		},
		{
			name: "MixedContent",
			jsonlContent: `{"type":"text","text":"Valid"}
{invalid json}
{"type":"text","text":"Also valid"}`,
			expectedOutput: "ValidAlso valid",
			expectError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := extractContentFromJSONL(tt.jsonlContent)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}

			if result != tt.expectedOutput {
				t.Errorf("Expected output '%s', got '%s'", tt.expectedOutput, result)
			}
		})
	}
}

func TestGenerateSanitizationSummary(t *testing.T) {
	tests := []struct {
		name     string
		analysis *SanitizationAnalysis
		expected string
	}{
		{
			name: "NoChanges",
			analysis: &SanitizationAnalysis{
				TotalChanges:  0,
				ChangesByType: make(map[string]int),
			},
			expected: "No sanitization changes detected",
		},
		{
			name: "SingleType",
			analysis: &SanitizationAnalysis{
				TotalChanges: 2,
				ChangesByType: map[string]int{
					"mention": 2,
				},
			},
			expected: "2 @mention(s)",
		},
		{
			name: "MultipleTypes",
			analysis: &SanitizationAnalysis{
				TotalChanges: 5,
				ChangesByType: map[string]int{
					"mention":     2,
					"url":         2,
					"bot_trigger": 1,
				},
			},
			expected: "2 @mention(s), 1 bot trigger(s), 2 URL(s) redacted",
		},
		{
			name: "WithTruncation",
			analysis: &SanitizationAnalysis{
				TotalChanges: 1,
				ChangesByType: map[string]int{
					"truncation": 1,
				},
			},
			expected: "content truncated",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generateSanitizationSummary(tt.analysis)

			// Since the order of map iteration is not guaranteed in Go,
			// we need to check that all expected parts are present
			// rather than exact string matching for multiple types
			if tt.analysis.TotalChanges <= 1 {
				if result != tt.expected {
					t.Errorf("Expected '%s', got '%s'", tt.expected, result)
				}
			} else {
				// For multiple types, check that all expected components are present
				for changeType, count := range tt.analysis.ChangesByType {
					var expectedPart string
					switch changeType {
					case "mention":
						expectedPart = fmt.Sprintf("%d @mention(s)", count)
					case "bot_trigger":
						expectedPart = fmt.Sprintf("%d bot trigger(s)", count)
					case "url":
						expectedPart = fmt.Sprintf("%d URL(s) redacted", count)
					case "truncation":
						expectedPart = "content truncated"
					}

					if expectedPart != "" && !strings.Contains(result, expectedPart) {
						t.Errorf("Expected result to contain '%s', got '%s'", expectedPart, result)
					}
				}
			}
		})
	}
}
