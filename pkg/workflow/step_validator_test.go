package workflow

import (
	"strings"
	"testing"
)

func TestValidateStepsGHToken(t *testing.T) {
	tests := []struct {
		name        string
		steps       []any
		shouldError bool
		errorMsg    string
	}{
		{
			name: "step with gh command and GH_TOKEN is valid",
			steps: []any{
				map[string]any{
					"name": "Run gh command",
					"run":  "gh issue list",
					"env": map[string]any{
						"GH_TOKEN": "${{ secrets.GITHUB_TOKEN }}",
					},
				},
			},
			shouldError: false,
		},
		{
			name: "step with gh command but no env should error",
			steps: []any{
				map[string]any{
					"name": "Run gh command",
					"run":  "gh issue list",
				},
			},
			shouldError: true,
			errorMsg:    "step 0 uses 'gh' CLI commands but does not have GH_TOKEN set in env",
		},
		{
			name: "step with gh command but no GH_TOKEN in env should error",
			steps: []any{
				map[string]any{
					"name": "Run gh command",
					"run":  "gh pr list",
					"env": map[string]any{
						"OTHER_VAR": "value",
					},
				},
			},
			shouldError: true,
			errorMsg:    "step 0 uses 'gh' CLI commands but does not have GH_TOKEN set in env",
		},
		{
			name: "step without gh command does not require GH_TOKEN",
			steps: []any{
				map[string]any{
					"name": "Run other command",
					"run":  "echo hello",
				},
			},
			shouldError: false,
		},
		{
			name: "step with gh in string but not as command is valid",
			steps: []any{
				map[string]any{
					"name": "Echo gh",
					"run":  "echo 'this is a high priority task'",
				},
			},
			shouldError: false,
		},
		{
			name: "multiple steps with gh commands all having GH_TOKEN is valid",
			steps: []any{
				map[string]any{
					"name": "First gh command",
					"run":  "gh issue list",
					"env": map[string]any{
						"GH_TOKEN": "${{ secrets.GITHUB_TOKEN }}",
					},
				},
				map[string]any{
					"name": "Second gh command",
					"run":  "gh pr list",
					"env": map[string]any{
						"GH_TOKEN": "${{ secrets.GITHUB_TOKEN }}",
					},
				},
			},
			shouldError: false,
		},
		{
			name: "second step missing GH_TOKEN should error",
			steps: []any{
				map[string]any{
					"name": "First gh command",
					"run":  "gh issue list",
					"env": map[string]any{
						"GH_TOKEN": "${{ secrets.GITHUB_TOKEN }}",
					},
				},
				map[string]any{
					"name": "Second gh command",
					"run":  "gh pr list",
				},
			},
			shouldError: true,
			errorMsg:    "step 1 uses 'gh' CLI commands but does not have GH_TOKEN set in env",
		},
		{
			name: "step with multiline gh command and GH_TOKEN is valid",
			steps: []any{
				map[string]any{
					"name": "Run complex gh command",
					"run": `set -e
gh issue list
gh pr list`,
					"env": map[string]any{
						"GH_TOKEN": "${{ secrets.GITHUB_TOKEN }}",
					},
				},
			},
			shouldError: false,
		},
		{
			name: "step with gh in variable name but not command is valid",
			steps: []any{
				map[string]any{
					"name": "Use variable",
					"run":  "echo $GH_REPO",
				},
			},
			shouldError: false,
		},
		{
			name: "step with uses field (action) does not require GH_TOKEN",
			steps: []any{
				map[string]any{
					"name": "Checkout",
					"uses": "actions/checkout@v4",
				},
			},
			shouldError: false,
		},
		{
			name: "step with gh command in pipe and GH_TOKEN is valid",
			steps: []any{
				map[string]any{
					"name": "Pipe with gh",
					"run":  "gh issue list | grep bug",
					"env": map[string]any{
						"GH_TOKEN": "${{ secrets.GITHUB_TOKEN }}",
					},
				},
			},
			shouldError: false,
		},
		{
			name: "step with invalid env type should error",
			steps: []any{
				map[string]any{
					"name": "Run gh command",
					"run":  "gh issue list",
					"env":  "invalid",
				},
			},
			shouldError: true,
			errorMsg:    "step 0 uses 'gh' CLI commands but has invalid env field",
		},
		{
			name:        "empty steps should not error",
			steps:       []any{},
			shouldError: false,
		},
		{
			name:        "nil steps should not error",
			steps:       nil,
			shouldError: false,
		},
		{
			name: "step with gh --help should require GH_TOKEN",
			steps: []any{
				map[string]any{
					"name": "Get gh help",
					"run":  "gh --help",
				},
			},
			shouldError: true,
			errorMsg:    "step 0 uses 'gh' CLI commands but does not have GH_TOKEN set in env",
		},
		{
			name: "step with ./gh-aw (not gh CLI) should not require GH_TOKEN",
			steps: []any{
				map[string]any{
					"name": "Run gh-aw",
					"run":  "./gh-aw compile",
				},
			},
			shouldError: false,
		},
		{
			name: "step with gh as subcommand should require GH_TOKEN",
			steps: []any{
				map[string]any{
					"name": "Run gh in script",
					"run":  "bash -c 'gh issue list'",
				},
			},
			shouldError: true,
			errorMsg:    "step 0 uses 'gh' CLI commands but does not have GH_TOKEN set in env",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateStepsGHToken(tt.steps)

			if tt.shouldError {
				if err == nil {
					t.Errorf("Expected error but got nil")
					return
				}
				if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error to contain %q, got: %v", tt.errorMsg, err)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
			}
		})
	}
}

func TestTruncateForError(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "short string unchanged",
			input:    "short",
			expected: "short",
		},
		{
			name:     "exactly 100 chars unchanged",
			input:    strings.Repeat("a", 100),
			expected: strings.Repeat("a", 100),
		},
		{
			name:     "over 100 chars truncated",
			input:    strings.Repeat("a", 150),
			expected: strings.Repeat("a", 100) + "...",
		},
		{
			name:     "with leading/trailing spaces",
			input:    "  test  ",
			expected: "test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := truncateForError(tt.input)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}
