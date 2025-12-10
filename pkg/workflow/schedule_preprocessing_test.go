package workflow

import (
	"strings"
	"testing"
)

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
							"cron": "weekly on friday",
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
			err := compiler.preprocessScheduleFields(tt.frontmatter)

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
	err := compiler.preprocessScheduleFields(frontmatter)
	if err != nil {
		t.Fatalf("preprocessing failed: %v", err)
	}

	// Create test YAML output
	yamlStr := `"on":
  schedule:
  - cron: "0 2 * * *"
  workflow_dispatch: null`

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
