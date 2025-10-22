package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFindAvailableScheduleHour(t *testing.T) {
	// Create a temporary directory structure for testing
	tmpDir, err := os.MkdirTemp("", "workflow-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Initialize a git repository in the temp directory
	if err := os.MkdirAll(filepath.Join(tmpDir, ".git"), 0755); err != nil {
		t.Fatalf("Failed to create .git dir: %v", err)
	}

	workflowsDir := filepath.Join(tmpDir, ".github", "workflows")
	if err := os.MkdirAll(workflowsDir, 0755); err != nil {
		t.Fatalf("Failed to create workflows dir: %v", err)
	}

	// Save current directory
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer os.Chdir(originalDir)

	// Change to temp directory
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	tests := []struct {
		name          string
		workflows     map[string]string
		expectedHours []int // Possible expected hours
	}{
		{
			name:          "no existing workflows",
			workflows:     map[string]string{},
			expectedHours: []int{9}, // Default hour
		},
		{
			name: "9 AM already used",
			workflows: map[string]string{
				"daily-1.md": `---
on:
  schedule:
    - cron: "0 9 * * 1-5"
---
# Test Workflow
`,
			},
			expectedHours: []int{10, 11, 14, 13, 8, 15, 16, 17}, // Next business hour
		},
		{
			name: "multiple hours used",
			workflows: map[string]string{
				"daily-1.md": `---
on:
  schedule:
    - cron: "0 9 * * *"
---
# Test Workflow 1
`,
				"daily-2.md": `---
on:
  schedule:
    - cron: "0 10 * * 1-5"
---
# Test Workflow 2
`,
				"daily-3.md": `---
on:
  schedule:
    - cron: "0 11 * * *"
---
# Test Workflow 3
`,
			},
			expectedHours: []int{14, 13, 8, 15, 16, 17}, // Next available business hour
		},
		{
			name: "all business hours used",
			workflows: map[string]string{
				"daily-1.md": `---
on:
  schedule:
    - cron: "0 8 * * 1-5"
---`,
				"daily-2.md": `---
on:
  schedule:
    - cron: "0 9 * * 1-5"
---`,
				"daily-3.md": `---
on:
  schedule:
    - cron: "0 10 * * 1-5"
---`,
				"daily-4.md": `---
on:
  schedule:
    - cron: "0 11 * * 1-5"
---`,
				"daily-5.md": `---
on:
  schedule:
    - cron: "0 13 * * 1-5"
---`,
				"daily-6.md": `---
on:
  schedule:
    - cron: "0 14 * * 1-5"
---`,
				"daily-7.md": `---
on:
  schedule:
    - cron: "0 15 * * 1-5"
---`,
				"daily-8.md": `---
on:
  schedule:
    - cron: "0 16 * * 1-5"
---`,
				"daily-9.md": `---
on:
  schedule:
    - cron: "0 17 * * 1-5"
---`,
			},
			expectedHours: []int{0, 1, 2, 3, 4, 5, 6, 7, 12, 18, 19, 20, 21, 22, 23}, // Any hour outside business hours
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean workflows directory
			os.RemoveAll(workflowsDir)
			os.MkdirAll(workflowsDir, 0755)

			// Create test workflows
			for filename, content := range tt.workflows {
				if err := os.WriteFile(filepath.Join(workflowsDir, filename), []byte(content), 0644); err != nil {
					t.Fatalf("Failed to write workflow file %s: %v", filename, err)
				}
			}

			// Find available hour using the test directory
			hour := findAvailableScheduleHourInDir(tmpDir)

			// Check if hour is in expected range
			if hour < 0 || hour >= 24 {
				t.Errorf("Invalid hour returned: %d (must be 0-23)", hour)
			}

			// Check if hour is in the expected hours list
			found := false
			for _, expectedHour := range tt.expectedHours {
				if hour == expectedHour {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Hour %d not in expected hours %v", hour, tt.expectedHours)
			}
		})
	}
}

func TestGenerateTriggerConfigScheduleDaily(t *testing.T) {
	builder := &InteractiveWorkflowBuilder{
		Trigger: "schedule_daily",
	}

	config := builder.generateTriggerConfig()

	// Verify the config contains schedule trigger
	if !strings.Contains(config, "schedule:") {
		t.Error("Config should contain 'schedule:' trigger")
	}

	// Verify it uses weekdays only (1-5)
	if !strings.Contains(config, "1-5") {
		t.Error("Config should use weekday-only schedule (1-5)")
	}

	// Verify it doesn't use the old pattern with all days (* * *)
	if strings.Contains(config, "* * *") {
		t.Error("Config should not use '* * *' pattern (includes weekends)")
	}

	// Verify the comment mentions weekdays only
	if !strings.Contains(config, "weekdays only") {
		t.Error("Config comment should mention 'weekdays only'")
	}
}

