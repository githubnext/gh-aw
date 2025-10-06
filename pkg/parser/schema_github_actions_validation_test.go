package parser

import (
	"encoding/json"
	"os"
	"testing"
)

// TestSchemaAlignmentWithGitHubActions validates that our schema aligns with the official GitHub Actions schema
func TestSchemaAlignmentWithGitHubActions(t *testing.T) {
	// Load our schema
	ourSchemaBytes, err := os.ReadFile("schemas/main_workflow_schema.json")
	if err != nil {
		t.Fatalf("Failed to read our schema: %v", err)
	}

	var ourSchema map[string]any
	if err := json.Unmarshal(ourSchemaBytes, &ourSchema); err != nil {
		t.Fatalf("Failed to parse our schema: %v", err)
	}

	// Load GitHub Actions schema
	githubSchemaBytes, err := os.ReadFile("../workflow/schemas/github-workflow.json")
	if err != nil {
		t.Fatalf("Failed to read GitHub Actions schema: %v", err)
	}

	var githubSchema map[string]any
	if err := json.Unmarshal(githubSchemaBytes, &githubSchema); err != nil {
		t.Fatalf("Failed to parse GitHub Actions schema: %v", err)
	}

	// Extract event properties from both schemas
	ourOn := getNestedValue(ourSchema, "properties", "on", "oneOf").([]any)
	ourEventObj := ourOn[1].(map[string]any) // The object variant
	ourEvents := getNestedValue(ourEventObj, "properties").(map[string]any)

	githubOn := getNestedValue(githubSchema, "properties", "on", "oneOf").([]any)
	githubEventObj := githubOn[2].(map[string]any) // The object variant (index 2 in GitHub schema)
	githubEvents := getNestedValue(githubEventObj, "properties").(map[string]any)

	// Test push event properties
	t.Run("push event properties match GitHub Actions", func(t *testing.T) {
		ourPush := getNestedValue(ourEvents, "push", "properties").(map[string]any)
		githubPush := getNestedValue(githubEvents, "push", "oneOf").([]any)[1].(map[string]any)
		githubPushProps := getNestedValue(githubPush, "allOf").([]any)[0].(map[string]any)
		githubPushProps = getNestedValue(githubPushProps, "properties").(map[string]any)

		// Check that all GitHub Actions properties are present in our schema
		for prop := range githubPushProps {
			if _, exists := ourPush[prop]; !exists {
				t.Errorf("Missing property in our push event: %s", prop)
			}
		}

		// Verify push event has additionalProperties: false
		ourPushObj := getNestedValue(ourEvents, "push").(map[string]any)
		if addProps, ok := ourPushObj["additionalProperties"].(bool); !ok || addProps {
			t.Error("Push event should have additionalProperties: false")
		}
	})

	// Test pull_request event properties
	t.Run("pull_request event properties", func(t *testing.T) {
		ourPR := getNestedValue(ourEvents, "pull_request", "properties").(map[string]any)
		githubPR := getNestedValue(githubEvents, "pull_request", "oneOf").([]any)[1].(map[string]any)
		githubPRProps := getNestedValue(githubPR, "allOf").([]any)[0].(map[string]any)
		githubPRProps = getNestedValue(githubPRProps, "properties").(map[string]any)

		// Standard GitHub Actions properties that should be present
		standardProps := []string{"branches", "branches-ignore", "paths", "paths-ignore", "types"}
		for _, prop := range standardProps {
			if _, exists := ourPR[prop]; !exists {
				t.Errorf("Missing standard property in our pull_request event: %s", prop)
			}
		}

		// Note: tags and tags-ignore are in the GitHub schema but are not actually supported
		// for pull_request events in practice. We intentionally don't include them.

		// Verify pull_request has additionalProperties: false
		ourPRObj := getNestedValue(ourEvents, "pull_request").(map[string]any)
		if addProps, ok := ourPRObj["additionalProperties"].(bool); !ok || addProps {
			t.Error("Pull request event should have additionalProperties: false")
		}

		// Verify we have our custom extensions (these are processed before YAML generation)
		customProps := []string{"draft", "forks", "names"}
		for _, prop := range customProps {
			if _, exists := ourPR[prop]; !exists {
				t.Errorf("Missing custom property in our pull_request event: %s", prop)
			}
		}
	})

	// Test that both schemas support standard event types
	t.Run("standard event types are supported", func(t *testing.T) {
		commonEvents := []string{"push", "pull_request", "issues", "workflow_dispatch", "schedule"}

		for _, event := range commonEvents {
			if _, exists := ourEvents[event]; !exists {
				t.Errorf("Our schema is missing standard event: %s", event)
			}
			if _, exists := githubEvents[event]; !exists {
				t.Errorf("GitHub schema is missing standard event (this is unexpected): %s", event)
			}
		}
	})

	// Test that event structures have proper validation
	t.Run("event structures have additionalProperties validation", func(t *testing.T) {
		eventsToCheck := []string{"push", "pull_request", "issues", "schedule"}

		for _, eventName := range eventsToCheck {
			event, exists := ourEvents[eventName].(map[string]any)
			if !exists {
				continue // Skip if event doesn't exist
			}

			// For events with direct object type
			if eventType, ok := event["type"].(string); ok && eventType == "object" {
				if addProps, ok := event["additionalProperties"].(bool); !ok || addProps {
					t.Errorf("Event '%s' should have additionalProperties: false", eventName)
				}
			}

			// For events with properties (object structure)
			if props, ok := event["properties"].(map[string]any); ok && len(props) > 0 {
				if addProps, ok := event["additionalProperties"].(bool); !ok || addProps {
					t.Errorf("Event '%s' should have additionalProperties: false", eventName)
				}
			}
		}
	})
}

// TestPushEventValidation tests that push event validates correctly with both schemas
func TestPushEventValidation(t *testing.T) {
	tests := []struct {
		name        string
		event       map[string]any
		wantErr     bool
		errContains string
	}{
		{
			name: "valid push with branches",
			event: map[string]any{
				"on": map[string]any{
					"push": map[string]any{
						"branches": []string{"main", "develop"},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "valid push with tags",
			event: map[string]any{
				"on": map[string]any{
					"push": map[string]any{
						"tags": []string{"v*"},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "valid push with paths",
			event: map[string]any{
				"on": map[string]any{
					"push": map[string]any{
						"paths": []string{"src/**"},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "invalid push with unknown property",
			event: map[string]any{
				"on": map[string]any{
					"push": map[string]any{
						"branches":     []string{"main"},
						"invalid_prop": "value",
					},
				},
			},
			wantErr:     true,
			errContains: "additional properties 'invalid_prop' not allowed",
		},
		{
			name: "valid push with branches and tags",
			event: map[string]any{
				"on": map[string]any{
					"push": map[string]any{
						"branches": []string{"main"},
						"tags":     []string{"v*"},
					},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateMainWorkflowFrontmatterWithSchema(tt.event)

			if tt.wantErr && err == nil {
				t.Error("Expected error but got nil")
			}

			if !tt.wantErr && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if tt.wantErr && err != nil && tt.errContains != "" {
				if !stringContains(err.Error(), tt.errContains) {
					t.Errorf("Error should contain '%s', got: %v", tt.errContains, err)
				}
			}
		})
	}
}

// TestPullRequestEventValidation tests that pull_request event validates correctly
func TestPullRequestEventValidation(t *testing.T) {
	tests := []struct {
		name        string
		event       map[string]any
		wantErr     bool
		errContains string
	}{
		{
			name: "valid pull_request with types",
			event: map[string]any{
				"on": map[string]any{
					"pull_request": map[string]any{
						"types": []string{"opened", "synchronize"},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "valid pull_request with branches",
			event: map[string]any{
				"on": map[string]any{
					"pull_request": map[string]any{
						"branches": []string{"main"},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "valid pull_request with paths",
			event: map[string]any{
				"on": map[string]any{
					"pull_request": map[string]any{
						"paths": []string{"docs/**"},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "valid pull_request with draft filter (custom extension)",
			event: map[string]any{
				"on": map[string]any{
					"pull_request": map[string]any{
						"draft": false,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "valid pull_request with forks filter (custom extension)",
			event: map[string]any{
				"on": map[string]any{
					"pull_request": map[string]any{
						"forks": []string{"org/*"},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "valid pull_request with names filter (custom extension)",
			event: map[string]any{
				"on": map[string]any{
					"pull_request": map[string]any{
						"types": []string{"labeled"},
						"names": []string{"bug", "enhancement"},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "invalid pull_request with unknown property",
			event: map[string]any{
				"on": map[string]any{
					"pull_request": map[string]any{
						"types":        []string{"opened"},
						"invalid_prop": "value",
					},
				},
			},
			wantErr:     true,
			errContains: "additional properties 'invalid_prop' not allowed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateMainWorkflowFrontmatterWithSchema(tt.event)

			if tt.wantErr && err == nil {
				t.Error("Expected error but got nil")
			}

			if !tt.wantErr && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if tt.wantErr && err != nil && tt.errContains != "" {
				if !stringContains(err.Error(), tt.errContains) {
					t.Errorf("Error should contain '%s', got: %v", tt.errContains, err)
				}
			}
		})
	}
}

// Helper function to get nested values from map
func getNestedValue(m map[string]any, keys ...string) any {
	var current any = m
	for _, key := range keys {
		if currentMap, ok := current.(map[string]any); ok {
			current = currentMap[key]
		} else {
			return nil
		}
	}
	return current
}

// Helper function to check if string contains substring
func stringContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
