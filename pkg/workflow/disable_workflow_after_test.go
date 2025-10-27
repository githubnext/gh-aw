package workflow

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestDisableWorkflowAfter tests the new disable-workflow-after field
func TestDisableWorkflowAfter(t *testing.T) {
	tests := []struct {
		name               string
		frontmatter        string
		markdown           string
		expectStopTime     bool
		expectDeprecation  bool
		shouldCompile      bool
	}{
		{
			name: "disable-workflow-after with absolute time",
			frontmatter: `---
engine: copilot
on:
  schedule:
    - cron: "0 9 * * 1"
  disable-workflow-after: "2025-12-31 23:59:59"
---`,
			markdown:          "# Test Workflow\n\nThis is a test workflow.",
			expectStopTime:    true,
			expectDeprecation: false,
			shouldCompile:     true,
		},
		{
			name: "disable-workflow-after with relative time",
			frontmatter: `---
engine: copilot
on:
  schedule:
    - cron: "0 9 * * 1"
  disable-workflow-after: "+7d"
---`,
			markdown:          "# Test Workflow\n\nThis is a test workflow.",
			expectStopTime:    true,
			expectDeprecation: false,
			shouldCompile:     true,
		},
		{
			name: "stop-after shows deprecation warning",
			frontmatter: `---
engine: copilot
on:
  schedule:
    - cron: "0 9 * * 1"
  stop-after: "+7d"
---`,
			markdown:          "# Test Workflow\n\nThis is a test workflow.",
			expectStopTime:    true,
			expectDeprecation: true,
			shouldCompile:     true,
		},
		{
			name: "disable-workflow-after takes precedence over stop-after",
			frontmatter: `---
engine: copilot
on:
  schedule:
    - cron: "0 9 * * 1"
  disable-workflow-after: "2025-12-31 23:59:59"
  stop-after: "2026-01-01 00:00:00"
---`,
			markdown:          "# Test Workflow\n\nThis is a test workflow.",
			expectStopTime:    true,
			expectDeprecation: false,
			shouldCompile:     true,
		},
		{
			name: "no disable-workflow-after or stop-after",
			frontmatter: `---
engine: copilot
on:
  schedule:
    - cron: "0 9 * * 1"
---`,
			markdown:          "# Test Workflow\n\nThis is a test workflow.",
			expectStopTime:    false,
			expectDeprecation: false,
			shouldCompile:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary files
			tmpDir := t.TempDir()
			mdFile := filepath.Join(tmpDir, "test-workflow.md")
			lockFile := filepath.Join(tmpDir, "test-workflow.lock.yml")

			// Write the test workflow
			content := tt.frontmatter + "\n\n" + tt.markdown
			err := os.WriteFile(mdFile, []byte(content), 0644)
			if err != nil {
				t.Fatalf("Failed to write test file: %v", err)
			}

			// Capture stderr to check for deprecation warning
			oldStderr := os.Stderr
			r, w, _ := os.Pipe()
			os.Stderr = w

			// Compile the workflow
			compiler := NewCompiler(false, "", "test-version")
			err = compiler.CompileWorkflow(mdFile)

			// Restore stderr and read what was written
			w.Close()
			os.Stderr = oldStderr
			var buf bytes.Buffer
			buf.ReadFrom(r)
			stderrOutput := buf.String()

			if tt.shouldCompile {
				if err != nil {
					t.Fatalf("Failed to compile workflow: %v", err)
				}

				// Check that the lock file was created
				if _, err := os.Stat(lockFile); os.IsNotExist(err) {
					t.Fatalf("Lock file was not created: %s", lockFile)
				}

				// Read the compiled workflow
				compiledContent, err := os.ReadFile(lockFile)
				if err != nil {
					t.Fatalf("Failed to read compiled workflow: %v", err)
				}

				compiledStr := string(compiledContent)

				if tt.expectStopTime {
					// Should contain stop-time check
					if !strings.Contains(compiledStr, "GH_AW_STOP_TIME:") {
						t.Error("Compiled workflow should contain stop-time check but doesn't")
					}
				} else {
					// Should not contain stop-time check
					if strings.Contains(compiledStr, "GH_AW_STOP_TIME:") {
						t.Error("Compiled workflow should not contain stop-time check but does")
					}
				}

				if tt.expectDeprecation {
					// Should have deprecation warning in stderr
					if !strings.Contains(stderrOutput, "deprecated") && !strings.Contains(stderrOutput, "DEPRECATED") {
						t.Error("Expected deprecation warning but didn't find it")
					}
					if !strings.Contains(stderrOutput, "disable-workflow-after") {
						t.Error("Deprecation warning should mention 'disable-workflow-after'")
					}
				} else {
					// Should not have deprecation warning
					if strings.Contains(stderrOutput, "deprecated") || strings.Contains(stderrOutput, "DEPRECATED") {
						t.Errorf("Unexpected deprecation warning: %s", stderrOutput)
					}
				}
			} else {
				if err == nil {
					t.Error("Expected compilation to fail but it succeeded")
				}
			}
		})
	}
}

// TestExtractStopAfterFromOn tests the extractStopAfterFromOn function
func TestExtractStopAfterFromOn(t *testing.T) {
	compiler := NewCompiler(false, "", "test-version")

	tests := []struct {
		name              string
		frontmatter       map[string]any
		expectedValue     string
		expectedDeprecated bool
		expectError       bool
	}{
		{
			name: "disable-workflow-after field",
			frontmatter: map[string]any{
				"on": map[string]any{
					"schedule": []any{
						map[string]any{"cron": "0 9 * * 1"},
					},
					"disable-workflow-after": "+7d",
				},
			},
			expectedValue:     "+7d",
			expectedDeprecated: false,
			expectError:       false,
		},
		{
			name: "stop-after field (deprecated)",
			frontmatter: map[string]any{
				"on": map[string]any{
					"schedule": []any{
						map[string]any{"cron": "0 9 * * 1"},
					},
					"stop-after": "+7d",
				},
			},
			expectedValue:     "+7d",
			expectedDeprecated: true,
			expectError:       false,
		},
		{
			name: "both fields - disable-workflow-after takes precedence",
			frontmatter: map[string]any{
				"on": map[string]any{
					"schedule": []any{
						map[string]any{"cron": "0 9 * * 1"},
					},
					"disable-workflow-after": "2025-12-31 23:59:59",
					"stop-after":             "2026-01-01 00:00:00",
				},
			},
			expectedValue:     "2025-12-31 23:59:59",
			expectedDeprecated: false,
			expectError:       false,
		},
		{
			name: "neither field present",
			frontmatter: map[string]any{
				"on": map[string]any{
					"schedule": []any{
						map[string]any{"cron": "0 9 * * 1"},
					},
				},
			},
			expectedValue:     "",
			expectedDeprecated: false,
			expectError:       false,
		},
		{
			name: "simple string on format",
			frontmatter: map[string]any{
				"on": "push",
			},
			expectedValue:     "",
			expectedDeprecated: false,
			expectError:       false,
		},
		{
			name: "invalid disable-workflow-after type",
			frontmatter: map[string]any{
				"on": map[string]any{
					"disable-workflow-after": 123,
				},
			},
			expectedValue:     "",
			expectedDeprecated: false,
			expectError:       true,
		},
		{
			name: "invalid stop-after type",
			frontmatter: map[string]any{
				"on": map[string]any{
					"stop-after": true,
				},
			},
			expectedValue:     "",
			expectedDeprecated: false,
			expectError:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value, deprecated, err := compiler.extractStopAfterFromOn(tt.frontmatter)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if value != tt.expectedValue {
				t.Errorf("Expected value %q but got %q", tt.expectedValue, value)
			}

			if deprecated != tt.expectedDeprecated {
				t.Errorf("Expected deprecated=%v but got %v", tt.expectedDeprecated, deprecated)
			}
		})
	}
}
