package workflow

import (
	"testing"
)

func TestAddCommentsConfigTargetRepo(t *testing.T) {
	compiler := NewCompiler(false, "", "")

	tests := []struct {
		name           string
		configMap      map[string]any
		expectedTarget string
		expectedRepo   string
		shouldBeNil    bool
	}{
		{
			name: "basic target-repo configuration",
			configMap: map[string]any{
				"add-comment": map[string]any{
					"max":         5,
					"target":      "*",
					"target-repo": "github/customer-feedback",
				},
			},
			expectedTarget: "*",
			expectedRepo:   "github/customer-feedback",
			shouldBeNil:    false,
		},
		{
			name: "target-repo with wildcard should be rejected",
			configMap: map[string]any{
				"add-comment": map[string]any{
					"max":         5,
					"target":      "123",
					"target-repo": "*",
				},
			},
			shouldBeNil: true, // Configuration should be nil due to validation
		},
		{
			name: "target-repo without target field",
			configMap: map[string]any{
				"add-comment": map[string]any{
					"max":         1,
					"target-repo": "owner/repo",
				},
			},
			expectedTarget: "",
			expectedRepo:   "owner/repo",
			shouldBeNil:    false,
		},
		{
			name: "no target-repo field",
			configMap: map[string]any{
				"add-comment": map[string]any{
					"max":    2,
					"target": "triggering",
				},
			},
			expectedTarget: "triggering",
			expectedRepo:   "",
			shouldBeNil:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := compiler.parseCommentsConfig(tt.configMap)

			if tt.shouldBeNil {
				if config != nil {
					t.Errorf("Expected config to be nil for invalid target-repo, but got %+v", config)
				}
				return
			}

			if config == nil {
				t.Fatal("Expected valid config, but got nil")
			}

			if config.Target != tt.expectedTarget {
				t.Errorf("Expected Target = %q, got %q", tt.expectedTarget, config.Target)
			}

			if config.TargetRepoSlug != tt.expectedRepo {
				t.Errorf("Expected TargetRepoSlug = %q, got %q", tt.expectedRepo, config.TargetRepoSlug)
			}
		})
	}
}
