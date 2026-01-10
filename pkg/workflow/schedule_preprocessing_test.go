package workflow

import (
	"fmt"
	"strings"
	"testing"
)

// TestScheduleWorkflowDispatchAutomatic verifies that workflow_dispatch is automatically
// added to workflows with schedule triggers in object format
func TestScheduleWorkflowDispatchAutomatic(t *testing.T) {
	tests := []struct {
		name                   string
		frontmatter            map[string]any
		expectedCron           string
		expectWorkflowDispatch bool
	}{
		{
			name: "schedule array format - should add workflow_dispatch",
			frontmatter: map[string]any{
				"on": map[string]any{
					"schedule": []any{
						map[string]any{
							"cron": "daily at 02:00",
						},
					},
				},
			},
			expectedCron:           "0 2 * * *",
			expectWorkflowDispatch: true,
		},
		{
			name: "schedule string format - should add workflow_dispatch",
			frontmatter: map[string]any{
				"on": map[string]any{
					"schedule": "daily at 02:00",
				},
			},
			expectedCron:           "0 2 * * *",
			expectWorkflowDispatch: true,
		},
		{
			name: "schedule with existing workflow_dispatch - should keep it",
			frontmatter: map[string]any{
				"on": map[string]any{
					"schedule": []any{
						map[string]any{
							"cron": "daily at 02:00",
						},
					},
					"workflow_dispatch": map[string]any{
						"inputs": map[string]any{
							"test": map[string]any{
								"description": "test input",
							},
						},
					},
				},
			},
			expectedCron:           "0 2 * * *",
			expectWorkflowDispatch: true,
		},
		{
			name: "multiple schedules - should add workflow_dispatch",
			frontmatter: map[string]any{
				"on": map[string]any{
					"schedule": []any{
						map[string]any{
							"cron": "daily at 02:00",
						},
						map[string]any{
							"cron": "weekly on friday",
						},
					},
				},
			},
			expectedCron:           "0 2 * * *",
			expectWorkflowDispatch: true,
		},
		{
			name: "schedule with other triggers - should add workflow_dispatch",
			frontmatter: map[string]any{
				"on": map[string]any{
					"push": map[string]any{
						"branches": []any{"main"},
					},
					"schedule": []any{
						map[string]any{
							"cron": "0 9 * * 1",
						},
					},
				},
			},
			expectedCron:           "0 9 * * 1",
			expectWorkflowDispatch: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compiler := NewCompiler(false, "", "test")
			// Set workflow identifier for fuzzy schedule scattering
			compiler.SetWorkflowIdentifier("test-workflow.md")

			err := compiler.preprocessScheduleFields(tt.frontmatter, "", "")
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			// Check that "on" was converted to a map
			onValue, exists := tt.frontmatter["on"]
			if !exists {
				t.Error("expected 'on' field to exist")
				return
			}

			onMap, ok := onValue.(map[string]any)
			if !ok {
				t.Errorf("expected 'on' to be a map, got %T", onValue)
				return
			}

			// Check schedule field exists
			scheduleValue, hasSchedule := onMap["schedule"]
			if !hasSchedule {
				t.Error("expected 'schedule' field in 'on' map")
				return
			}

			// Check the cron expression
			scheduleArray, ok := scheduleValue.([]any)
			if !ok {
				t.Errorf("expected schedule to be array, got %T", scheduleValue)
				return
			}

			if len(scheduleArray) == 0 {
				t.Error("expected at least one schedule item")
				return
			}

			firstSchedule, ok := scheduleArray[0].(map[string]any)
			if !ok {
				t.Errorf("expected first schedule to be map, got %T", scheduleArray[0])
				return
			}

			actualCron, ok := firstSchedule["cron"].(string)
			if !ok {
				t.Errorf("expected cron to be string, got %T", firstSchedule["cron"])
				return
			}

			if tt.expectedCron != "" && actualCron != tt.expectedCron {
				t.Errorf("expected cron '%s', got '%s'", tt.expectedCron, actualCron)
			}

			// Check workflow_dispatch field
			if tt.expectWorkflowDispatch {
				if _, hasWorkflowDispatch := onMap["workflow_dispatch"]; !hasWorkflowDispatch {
					t.Error("expected 'workflow_dispatch' field in 'on' map but it was not added")
					return
				}
			}
		})
	}
}

func TestSchedulePreprocessingShorthandOnString(t *testing.T) {
	tests := []struct {
		name                   string
		frontmatter            map[string]any
		checkScattered         bool // Check if fuzzy was scattered to valid cron
		expectedCron           string
		expectedError          bool
		errorSubstring         string
		expectWorkflowDispatch bool
	}{
		{
			name: "on: daily",
			frontmatter: map[string]any{
				"on": "daily",
			},
			checkScattered:         true, // Fuzzy schedule, should be scattered
			expectWorkflowDispatch: true,
		},
		{
			name: "on: weekly",
			frontmatter: map[string]any{
				"on": "weekly",
			},
			checkScattered:         true, // Fuzzy schedule, should be scattered
			expectWorkflowDispatch: true,
		},
		{
			name: "on: daily at 14:00",
			frontmatter: map[string]any{
				"on": "daily at 14:00",
			},
			expectedCron:           "0 14 * * *",
			expectWorkflowDispatch: true,
		},
		{
			name: "on: weekly on monday",
			frontmatter: map[string]any{
				"on": "weekly on monday",
			},
			checkScattered:         true, // Fuzzy schedule, should be scattered
			expectWorkflowDispatch: true,
		},
		{
			name: "on: every 10 minutes",
			frontmatter: map[string]any{
				"on": "every 10 minutes",
			},
			expectedCron:           "*/10 * * * *",
			expectWorkflowDispatch: true,
		},
		{
			name: "on: 0 9 * * 1 (cron expression)",
			frontmatter: map[string]any{
				"on": "0 9 * * 1",
			},
			expectedCron:           "0 9 * * 1",
			expectWorkflowDispatch: true,
		},
		{
			name: "on: push (not a schedule)",
			frontmatter: map[string]any{
				"on": "push",
			},
			expectedCron:           "",
			expectWorkflowDispatch: false,
		},
		{
			name: "on: invalid schedule",
			frontmatter: map[string]any{
				"on": "invalid schedule format",
			},
			expectedCron:           "",
			expectWorkflowDispatch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compiler := NewCompiler(false, "", "test")
			// Set workflow identifier for fuzzy schedule scattering
			// (required for all schedule tests to avoid fuzzy schedule errors)
			compiler.SetWorkflowIdentifier("test-workflow.md")

			err := compiler.preprocessScheduleFields(tt.frontmatter, "", "")

			if tt.expectedError {
				if err == nil {
					t.Errorf("expected error containing '%s', got nil", tt.errorSubstring)
					return
				}
				if !strings.Contains(err.Error(), tt.errorSubstring) {
					t.Errorf("expected error containing '%s', got '%s'", tt.errorSubstring, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			// If expectWorkflowDispatch is false, "on" should still be a string
			if !tt.expectWorkflowDispatch {
				onValue, exists := tt.frontmatter["on"]
				if !exists {
					t.Error("expected 'on' field to exist")
					return
				}
				if _, ok := onValue.(string); !ok {
					t.Errorf("expected 'on' to remain a string for non-schedule value")
				}
				return
			}

			// Check that "on" was converted to a map with schedule and workflow_dispatch
			onValue, exists := tt.frontmatter["on"]
			if !exists {
				t.Error("expected 'on' field to exist")
				return
			}

			onMap, ok := onValue.(map[string]any)
			if !ok {
				t.Errorf("expected 'on' to be converted to map, got %T", onValue)
				return
			}

			// Check schedule field exists
			scheduleValue, hasSchedule := onMap["schedule"]
			if !hasSchedule {
				t.Error("expected 'schedule' field in 'on' map")
				return
			}

			// Check workflow_dispatch field exists
			if _, hasWorkflowDispatch := onMap["workflow_dispatch"]; !hasWorkflowDispatch {
				t.Error("expected 'workflow_dispatch' field in 'on' map")
				return
			}

			// Check the cron expression
			scheduleArray, ok := scheduleValue.([]any)
			if !ok {
				t.Errorf("expected schedule to be array, got %T", scheduleValue)
				return
			}

			if len(scheduleArray) == 0 {
				t.Error("expected at least one schedule item")
				return
			}

			firstSchedule, ok := scheduleArray[0].(map[string]any)
			if !ok {
				t.Errorf("expected first schedule to be map, got %T", scheduleArray[0])
				return
			}

			actualCron, ok := firstSchedule["cron"].(string)
			if !ok {
				t.Errorf("expected cron to be string, got %T", firstSchedule["cron"])
				return
			}

			if tt.checkScattered {
				// Should be scattered to a valid cron (not fuzzy)
				if strings.HasPrefix(actualCron, "FUZZY:") {
					t.Errorf("expected scattered cron, got fuzzy: %s", actualCron)
				}
				// Verify it's a valid cron expression
				fields := strings.Fields(actualCron)
				if len(fields) != 5 {
					t.Errorf("expected 5 fields in cron expression, got %d: %s", len(fields), actualCron)
				}
				t.Logf("Successfully scattered schedule to: %s", actualCron)
			} else if tt.expectedCron != "" {
				if actualCron != tt.expectedCron {
					t.Errorf("expected cron '%s', got '%s'", tt.expectedCron, actualCron)
				}
			}
		})
	}
}

func TestSchedulePreprocessing(t *testing.T) {
	tests := []struct {
		name           string
		frontmatter    map[string]any
		expectedCron   string
		expectedError  bool
		errorSubstring string
	}{
		{
			name: "daily schedule",
			frontmatter: map[string]any{
				"on": map[string]any{
					"schedule": []any{
						map[string]any{
							"cron": "daily at 02:00",
						},
					},
				},
			},
			expectedCron: "0 2 * * *",
		},
		{
			name: "weekly schedule",
			frontmatter: map[string]any{
				"on": map[string]any{
					"schedule": []any{
						map[string]any{
							"cron": "weekly on monday at 06:30",
						},
					},
				},
			},
			expectedCron: "30 6 * * 1",
		},
		{
			name: "interval schedule",
			frontmatter: map[string]any{
				"on": map[string]any{
					"schedule": []any{
						map[string]any{
							"cron": "every 10 minutes",
						},
					},
				},
			},
			expectedCron: "*/10 * * * *",
		},
		{
			name: "existing cron expression unchanged",
			frontmatter: map[string]any{
				"on": map[string]any{
					"schedule": []any{
						map[string]any{
							"cron": "0 9 * * 1",
						},
					},
				},
			},
			expectedCron: "0 9 * * 1",
		},
		{
			name: "multiple schedules",
			frontmatter: map[string]any{
				"on": map[string]any{
					"schedule": []any{
						map[string]any{
							"cron": "daily at 02:00",
						},
						map[string]any{
							"cron": "weekly on friday at 17:00",
						},
					},
				},
			},
			expectedCron: "0 2 * * *", // First one
		},
		{
			name: "invalid schedule format",
			frontmatter: map[string]any{
				"on": map[string]any{
					"schedule": []any{
						map[string]any{
							"cron": "invalid schedule format",
						},
					},
				},
			},
			expectedError:  true,
			errorSubstring: "invalid schedule expression",
		},
		// New tests for shorthand string format
		{
			name: "shorthand string format - daily",
			frontmatter: map[string]any{
				"on": map[string]any{
					"schedule": "daily at 02:00",
				},
			},
			expectedCron: "0 2 * * *",
		},
		{
			name: "shorthand string format - weekly",
			frontmatter: map[string]any{
				"on": map[string]any{
					"schedule": "weekly on monday at 06:30",
				},
			},
			expectedCron: "30 6 * * 1",
		},
		{
			name: "shorthand string format - interval",
			frontmatter: map[string]any{
				"on": map[string]any{
					"schedule": "every 10 minutes",
				},
			},
			expectedCron: "*/10 * * * *",
		},
		{
			name: "shorthand string format - existing cron",
			frontmatter: map[string]any{
				"on": map[string]any{
					"schedule": "0 9 * * 1",
				},
			},
			expectedCron: "0 9 * * 1",
		},
		{
			name: "shorthand string format - invalid",
			frontmatter: map[string]any{
				"on": map[string]any{
					"schedule": "invalid format",
				},
			},
			expectedError:  true,
			errorSubstring: "invalid schedule expression",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compiler := NewCompiler(false, "", "test")
			err := compiler.preprocessScheduleFields(tt.frontmatter, "", "")

			if tt.expectedError {
				if err == nil {
					t.Errorf("expected error containing '%s', got nil", tt.errorSubstring)
					return
				}
				if !strings.Contains(err.Error(), tt.errorSubstring) {
					t.Errorf("expected error containing '%s', got '%s'", tt.errorSubstring, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			// Check that the cron expression was updated
			onMap := tt.frontmatter["on"].(map[string]any)
			scheduleArray := onMap["schedule"].([]any)
			firstSchedule := scheduleArray[0].(map[string]any)
			actualCron := firstSchedule["cron"].(string)

			if actualCron != tt.expectedCron {
				t.Errorf("expected cron '%s', got '%s'", tt.expectedCron, actualCron)
			}
		})
	}
}

func TestScheduleFriendlyComments(t *testing.T) {
	// Create a test frontmatter with a friendly schedule
	frontmatter := map[string]any{
		"on": map[string]any{
			"schedule": []any{
				map[string]any{
					"cron": "daily at 02:00",
				},
			},
		},
	}

	compiler := NewCompiler(false, "", "test")

	// Preprocess to convert and store friendly formats
	err := compiler.preprocessScheduleFields(frontmatter, "", "")
	if err != nil {
		t.Fatalf("preprocessing failed: %v", err)
	}

	// Create test YAML output
	yamlStr := `"on":
  schedule:
  - cron: "0 2 * * *"
  workflow_dispatch:`

	// Add friendly comments
	result := compiler.addFriendlyScheduleComments(yamlStr, frontmatter)

	// Check that the comment was added
	if !strings.Contains(result, "# Friendly format: daily at 02:00") {
		t.Errorf("expected friendly format comment to be added, got:\n%s", result)
	}

	// Check that the cron expression is still there
	if !strings.Contains(result, `cron: "0 2 * * *"`) {
		t.Errorf("expected cron expression to remain, got:\n%s", result)
	}
}

func TestFuzzyScheduleScattering(t *testing.T) {
	tests := []struct {
		name               string
		frontmatter        map[string]any
		workflowIdentifier string
		checkScattered     bool // If true, verify the result is scattered (not fuzzy)
		expectError        bool // If true, expect an error (fuzzy without identifier)
		errorSubstring     string
	}{
		{
			name: "fuzzy daily schedule with identifier",
			frontmatter: map[string]any{
				"on": map[string]any{
					"schedule": []any{
						map[string]any{
							"cron": "daily",
						},
					},
				},
			},
			workflowIdentifier: "workflow-a.md",
			checkScattered:     true,
			expectError:        false,
		},
		{
			name: "fuzzy daily schedule without identifier",
			frontmatter: map[string]any{
				"on": map[string]any{
					"schedule": []any{
						map[string]any{
							"cron": "daily",
						},
					},
				},
			},
			workflowIdentifier: "", // No identifier, should error
			checkScattered:     false,
			expectError:        true,
			errorSubstring:     "fuzzy cron expression",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compiler := NewCompiler(false, "", "test")
			if tt.workflowIdentifier != "" {
				compiler.SetWorkflowIdentifier(tt.workflowIdentifier)
			}

			err := compiler.preprocessScheduleFields(tt.frontmatter, "", "")

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error containing '%s', got nil", tt.errorSubstring)
					return
				}
				if !strings.Contains(err.Error(), tt.errorSubstring) {
					t.Errorf("expected error containing '%s', got '%s'", tt.errorSubstring, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			// Check that the cron expression was updated
			onMap := tt.frontmatter["on"].(map[string]any)
			scheduleArray := onMap["schedule"].([]any)
			firstSchedule := scheduleArray[0].(map[string]any)
			actualCron := firstSchedule["cron"].(string)

			if tt.checkScattered {
				// Should be scattered (not fuzzy)
				if strings.HasPrefix(actualCron, "FUZZY:") {
					t.Errorf("expected scattered schedule, got fuzzy: %s", actualCron)
				}
				// Should be a valid daily cron
				fields := strings.Fields(actualCron)
				if len(fields) != 5 {
					t.Errorf("expected 5 fields in cron, got %d: %s", len(fields), actualCron)
				}
			}
		})
	}
}

func TestFuzzyScheduleScatteringDeterministic(t *testing.T) {
	// Test that scattering is deterministic - same workflow ID produces same result
	workflows := []string{"workflow-a.md", "workflow-b.md", "workflow-c.md", "workflow-a.md"}

	results := make([]string, len(workflows))
	for i, wf := range workflows {
		frontmatter := map[string]any{
			"on": map[string]any{
				"schedule": []any{
					map[string]any{
						"cron": "daily",
					},
				},
			},
		}

		compiler := NewCompiler(false, "", "test")
		compiler.SetWorkflowIdentifier(wf)

		err := compiler.preprocessScheduleFields(frontmatter, "", "")
		if err != nil {
			t.Fatalf("unexpected error for workflow %s: %v", wf, err)
		}

		onMap := frontmatter["on"].(map[string]any)
		scheduleArray := onMap["schedule"].([]any)
		firstSchedule := scheduleArray[0].(map[string]any)
		results[i] = firstSchedule["cron"].(string)
	}

	// workflow-a.md should produce the same result both times
	if results[0] != results[3] {
		t.Errorf("Scattering not deterministic: workflow-a.md produced %s and %s", results[0], results[3])
	}

	// Different workflows should produce different results (with high probability)
	if results[0] == results[1] && results[1] == results[2] {
		t.Errorf("Scattering produced identical results for all workflows: %s", results[0])
	}
}

func TestSchedulePreprocessingWithFuzzyDaily(t *testing.T) {
	// Test various fuzzy daily schedule formats
	tests := []struct {
		name          string
		frontmatter   map[string]any
		checkScatter  bool
		expectError   bool
		errorContains string
	}{
		{
			name: "fuzzy daily - shorthand string",
			frontmatter: map[string]any{
				"on": map[string]any{
					"schedule": "daily",
				},
			},
			checkScatter: true,
		},
		{
			name: "fuzzy daily - array format",
			frontmatter: map[string]any{
				"on": map[string]any{
					"schedule": []any{
						map[string]any{
							"cron": "daily",
						},
					},
				},
			},
			checkScatter: true,
		},
		{
			name: "fuzzy daily at specific time",
			frontmatter: map[string]any{
				"on": map[string]any{
					"schedule": "daily at 14:30",
				},
			},
			checkScatter: false, // This has a specific time, so not scattered
		},
		{
			name: "fuzzy daily around specific time",
			frontmatter: map[string]any{
				"on": map[string]any{
					"schedule": "daily around 14:30",
				},
			},
			checkScatter: true, // This uses "around", so it should be scattered
		},
		{
			name: "fuzzy daily with multiple schedules",
			frontmatter: map[string]any{
				"on": map[string]any{
					"schedule": []any{
						map[string]any{
							"cron": "daily",
						},
						map[string]any{
							"cron": "weekly on monday",
						},
					},
				},
			},
			checkScatter: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compiler := NewCompiler(false, "", "test")
			compiler.SetWorkflowIdentifier("test-workflow.md")

			err := compiler.preprocessScheduleFields(tt.frontmatter, "", "")

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error containing '%s', got nil", tt.errorContains)
					return
				}
				if !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("expected error containing '%s', got '%s'", tt.errorContains, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			// Extract the cron expression
			onMap := tt.frontmatter["on"].(map[string]any)

			var actualCron string
			switch schedule := onMap["schedule"].(type) {
			case []any:
				firstSchedule := schedule[0].(map[string]any)
				actualCron = firstSchedule["cron"].(string)
			case map[string]any:
				actualCron = schedule["cron"].(string)
			default:
				t.Fatalf("unexpected schedule type: %T", schedule)
			}

			// Verify it's not in fuzzy format anymore (should be scattered)
			if strings.HasPrefix(actualCron, "FUZZY:") {
				t.Errorf("schedule should have been scattered, still in fuzzy format: %s", actualCron)
			}

			// Verify it's a valid cron expression
			fields := strings.Fields(actualCron)
			if len(fields) != 5 {
				t.Errorf("expected 5 fields in cron expression, got %d: %s", len(fields), actualCron)
			}

			if tt.checkScatter {
				// For scattered daily schedules, verify it's a daily pattern
				if fields[2] != "*" || fields[3] != "*" || fields[4] != "*" {
					t.Errorf("expected daily pattern (minute hour * * *), got: %s", actualCron)
				}
				t.Logf("Successfully scattered fuzzy daily schedule to: %s", actualCron)
			}
		})
	}
}

func TestSchedulePreprocessingDailyVariations(t *testing.T) {
	// Test that "daily" produces a valid scattered schedule
	compiler := NewCompiler(false, "", "test")
	compiler.SetWorkflowIdentifier("daily-variation-test.md")

	frontmatter := map[string]any{
		"on": map[string]any{
			"schedule": "daily",
		},
	}

	err := compiler.preprocessScheduleFields(frontmatter, "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Extract and verify the scattered schedule
	onMap := frontmatter["on"].(map[string]any)

	// The schedule will be converted to array format during preprocessing
	var cronExpr string
	switch schedule := onMap["schedule"].(type) {
	case []any:
		firstSchedule := schedule[0].(map[string]any)
		cronExpr = firstSchedule["cron"].(string)
	case map[string]any:
		cronExpr = schedule["cron"].(string)
	default:
		t.Fatalf("unexpected schedule type: %T", schedule)
	}

	// Verify it's a valid daily cron expression
	fields := strings.Fields(cronExpr)
	if len(fields) != 5 {
		t.Fatalf("expected 5 fields in cron expression, got %d: %s", len(fields), cronExpr)
	}

	// Parse hour and minute to ensure they're valid
	var minute, hour int
	if _, err := fmt.Sscanf(fields[0], "%d", &minute); err != nil {
		t.Errorf("invalid minute field: %s", fields[0])
	}
	if _, err := fmt.Sscanf(fields[1], "%d", &hour); err != nil {
		t.Errorf("invalid hour field: %s", fields[1])
	}

	// Verify ranges
	if minute < 0 || minute > 59 {
		t.Errorf("minute should be 0-59, got: %d", minute)
	}
	if hour < 0 || hour > 23 {
		t.Errorf("hour should be 0-23, got: %d", hour)
	}

	// Verify daily pattern
	if fields[2] != "*" || fields[3] != "*" || fields[4] != "*" {
		t.Errorf("expected daily pattern (minute hour * * *), got: %s", cronExpr)
	}

	t.Logf("Successfully compiled 'daily' to valid cron: %s", cronExpr)
}

func TestSlashCommandShorthand(t *testing.T) {
	tests := []struct {
		name                  string
		frontmatter           map[string]any
		expectedCommand       string
		expectWorkflowDispath bool
		expectedError         bool
		errorSubstring        string
	}{
		{
			name: "on: /command",
			frontmatter: map[string]any{
				"on": "/my-bot",
			},
			expectedCommand:       "my-bot",
			expectWorkflowDispath: true,
		},
		{
			name: "on: /another-command",
			frontmatter: map[string]any{
				"on": "/code-review",
			},
			expectedCommand:       "code-review",
			expectWorkflowDispath: true,
		},
		{
			name: "on: / (empty command)",
			frontmatter: map[string]any{
				"on": "/",
			},
			expectedError:  true,
			errorSubstring: "slash command shorthand cannot be empty after '/'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compiler := NewCompiler(false, "", "test")
			compiler.SetWorkflowIdentifier("test-workflow.md")

			err := compiler.preprocessScheduleFields(tt.frontmatter, "", "")

			if tt.expectedError {
				if err == nil {
					t.Errorf("expected error containing '%s', got nil", tt.errorSubstring)
					return
				}
				if !strings.Contains(err.Error(), tt.errorSubstring) {
					t.Errorf("expected error containing '%s', got '%s'", tt.errorSubstring, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			// Check that "on" was converted to a map with slash_command and workflow_dispatch
			onValue, exists := tt.frontmatter["on"]
			if !exists {
				t.Error("expected 'on' field to exist")
				return
			}

			onMap, ok := onValue.(map[string]any)
			if !ok {
				t.Errorf("expected 'on' to be converted to map, got %T", onValue)
				return
			}

			// Check slash_command field exists and has correct value
			slashCommandValue, hasSlashCommand := onMap["slash_command"]
			if !hasSlashCommand {
				t.Error("expected 'slash_command' field in 'on' map")
				return
			}

			slashCommandStr, ok := slashCommandValue.(string)
			if !ok {
				t.Errorf("expected slash_command to be string, got %T", slashCommandValue)
				return
			}

			if slashCommandStr != tt.expectedCommand {
				t.Errorf("expected slash_command '%s', got '%s'", tt.expectedCommand, slashCommandStr)
			}

			// Check workflow_dispatch field exists
			if _, hasWorkflowDispatch := onMap["workflow_dispatch"]; !hasWorkflowDispatch {
				t.Error("expected 'workflow_dispatch' field in 'on' map")
				return
			}

			// Ensure there are no extra fields (should only have slash_command and workflow_dispatch)
			if len(onMap) != 2 {
				t.Errorf("expected exactly 2 fields in 'on' map, got %d: %v", len(onMap), onMap)
			}
		})
	}
}
