package workflow

import (
	"strings"
	"testing"
)

func TestAddCommentsConfigHideOlderComments(t *testing.T) {
	compiler := NewCompiler(false, "", "")

	tests := []struct {
		name                      string
		configMap                 map[string]any
		expectedHideOlderComments bool
		shouldBeNil               bool
	}{
		{
			name: "hide-older-comments set to true",
			configMap: map[string]any{
				"add-comment": map[string]any{
					"max":                 1,
					"hide-older-comments": true,
				},
			},
			expectedHideOlderComments: true,
			shouldBeNil:               false,
		},
		{
			name: "hide-older-comments set to false",
			configMap: map[string]any{
				"add-comment": map[string]any{
					"max":                 1,
					"hide-older-comments": false,
				},
			},
			expectedHideOlderComments: false,
			shouldBeNil:               false,
		},
		{
			name: "hide-older-comments not specified (default false)",
			configMap: map[string]any{
				"add-comment": map[string]any{
					"max": 1,
				},
			},
			expectedHideOlderComments: false,
			shouldBeNil:               false,
		},
		{
			name: "hide-older-comments with other fields",
			configMap: map[string]any{
				"add-comment": map[string]any{
					"max":                 3,
					"target":              "*",
					"hide-older-comments": true,
				},
			},
			expectedHideOlderComments: true,
			shouldBeNil:               false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := compiler.parseCommentsConfig(tt.configMap)

			if tt.shouldBeNil {
				if config != nil {
					t.Errorf("Expected config to be nil, but got %+v", config)
				}
				return
			}

			if config == nil {
				t.Fatal("Expected valid config, but got nil")
			}

			if config.HideOlderComments != tt.expectedHideOlderComments {
				t.Errorf("Expected HideOlderComments = %v, got %v", tt.expectedHideOlderComments, config.HideOlderComments)
			}
		})
	}
}

func TestAddCommentHideOlderCommentsEnvVar(t *testing.T) {
	compiler := NewCompiler(false, "", "")

	// Create a minimal workflow with hide-older-comments enabled
	data := &WorkflowData{
		SafeOutputs: &SafeOutputsConfig{
			AddComments: &AddCommentsConfig{
				BaseSafeOutputConfig: BaseSafeOutputConfig{
					Max: 1,
				},
				HideOlderComments: true,
			},
		},
	}

	job, err := compiler.buildCreateOutputAddCommentJob(data, "run_agent", "", "", "")
	if err != nil {
		t.Fatalf("Failed to build add_comment job: %v", err)
	}

	if job == nil {
		t.Fatal("Expected job to be created, but got nil")
	}

	// Check that the environment variable is set in the job
	found := false
	for _, step := range job.Steps {
		if strings.Contains(step, "GH_AW_HIDE_OLDER_COMMENTS: \"true\"") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected GH_AW_HIDE_OLDER_COMMENTS environment variable in job steps, but not found.\nJob steps:\n%v", job.Steps)
	}
}

func TestAddCommentHideOlderCommentsNotSetByDefault(t *testing.T) {
	compiler := NewCompiler(false, "", "")

	// Create a minimal workflow without hide-older-comments
	data := &WorkflowData{
		SafeOutputs: &SafeOutputsConfig{
			AddComments: &AddCommentsConfig{
				BaseSafeOutputConfig: BaseSafeOutputConfig{
					Max: 1,
				},
				HideOlderComments: false,
			},
		},
	}

	job, err := compiler.buildCreateOutputAddCommentJob(data, "run_agent", "", "", "")
	if err != nil {
		t.Fatalf("Failed to build add_comment job: %v", err)
	}

	if job == nil {
		t.Fatal("Expected job to be created, but got nil")
	}

	// Check that the environment variable is NOT set in the job
	for _, step := range job.Steps {
		// Look for the actual env var declaration, not just code that references it
		if strings.Contains(step, "GH_AW_HIDE_OLDER_COMMENTS:") {
			t.Errorf("Expected GH_AW_HIDE_OLDER_COMMENTS environment variable NOT to be declared in job steps when false, but found it in step:\n%s", step)
		}
	}
}
