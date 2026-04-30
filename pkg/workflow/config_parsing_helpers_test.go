//go:build !integration

package workflow

import (
	"maps"
	"strings"
	"testing"
)

func TestExtractStringFromMap(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]any
		key      string
		expected string
	}{
		{
			name: "valid string value",
			input: map[string]any{
				"my-key": "my-value",
			},
			key:      "my-key",
			expected: "my-value",
		},
		{
			name: "empty string value",
			input: map[string]any{
				"my-key": "",
			},
			key:      "my-key",
			expected: "",
		},
		{
			name:     "missing key",
			input:    map[string]any{},
			key:      "my-key",
			expected: "",
		},
		{
			name: "non-string type",
			input: map[string]any{
				"my-key": 123,
			},
			key:      "my-key",
			expected: "",
		},
		{
			name: "string with special characters",
			input: map[string]any{
				"my-key": "[Special] 🎯 Value",
			},
			key:      "my-key",
			expected: "[Special] 🎯 Value",
		},
		{
			name: "different key returns different value",
			input: map[string]any{
				"key1": "value1",
				"key2": "value2",
			},
			key:      "key2",
			expected: "value2",
		},
		{
			name: "non-string value returns empty",
			input: map[string]any{
				"my-key": []string{"array", "value"},
			},
			key:      "my-key",
			expected: "",
		},
		{
			name: "nil value returns empty",
			input: map[string]any{
				"my-key": nil,
			},
			key:      "my-key",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractStringFromMap(tt.input, tt.key, nil)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestParseLabelsFromConfig(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]any
		expected []string
	}{
		{
			name: "valid labels array",
			input: map[string]any{
				"labels": []any{"bug", "enhancement", "documentation"},
			},
			expected: []string{"bug", "enhancement", "documentation"},
		},
		{
			name: "empty labels array",
			input: map[string]any{
				"labels": []any{},
			},
			expected: []string{},
		},
		{
			name:     "missing labels field",
			input:    map[string]any{},
			expected: nil,
		},
		{
			name: "labels with mixed types (filters non-strings)",
			input: map[string]any{
				"labels": []any{"bug", 123, "enhancement", true, "documentation"},
			},
			expected: []string{"bug", "enhancement", "documentation"},
		},
		{
			name: "labels as []string",
			input: map[string]any{
				"labels": []string{"bug", "enhancement"},
			},
			expected: []string{"bug", "enhancement"},
		},
		{
			name: "labels as non-array type",
			input: map[string]any{
				"labels": "not-an-array",
			},
			expected: nil,
		},
		{
			name: "labels with only non-string types",
			input: map[string]any{
				"labels": []any{123, true, 456},
			},
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseLabelsFromConfig(tt.input)

			// Handle nil vs empty slice comparison
			if tt.expected == nil {
				if result != nil {
					t.Errorf("expected nil, got %v", result)
				}
				return
			}

			if result == nil {
				t.Errorf("expected %v, got nil", tt.expected)
				return
			}

			if len(result) != len(tt.expected) {
				t.Errorf("expected length %d, got %d", len(tt.expected), len(result))
				return
			}

			for i, expected := range tt.expected {
				if result[i] != expected {
					t.Errorf("at index %d: expected %q, got %q", i, expected, result[i])
				}
			}
		})
	}
}

func TestParseTitlePrefixFromConfig(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]any
		expected string
	}{
		{
			name: "valid title-prefix",
			input: map[string]any{
				"title-prefix": "[bot] ",
			},
			expected: "[bot] ",
		},
		{
			name: "empty title-prefix",
			input: map[string]any{
				"title-prefix": "",
			},
			expected: "",
		},
		{
			name:     "missing title-prefix field",
			input:    map[string]any{},
			expected: "",
		},
		{
			name: "title-prefix as non-string type",
			input: map[string]any{
				"title-prefix": 123,
			},
			expected: "",
		},
		{
			name: "title-prefix with special characters",
			input: map[string]any{
				"title-prefix": "[AI-Generated] 🤖 ",
			},
			expected: "[AI-Generated] 🤖 ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseTitlePrefixFromConfig(tt.input)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestParseTargetRepoFromConfig(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]any
		expected string
	}{
		{
			name: "valid target-repo",
			input: map[string]any{
				"target-repo": "owner/repo",
			},
			expected: "owner/repo",
		},
		{
			name: "wildcard target-repo (returns * for caller to validate)",
			input: map[string]any{
				"target-repo": "*",
			},
			expected: "*",
		},
		{
			name:     "missing target-repo field",
			input:    map[string]any{},
			expected: "",
		},
		{
			name: "target-repo as non-string type",
			input: map[string]any{
				"target-repo": 123,
			},
			expected: "",
		},
		{
			name: "target-repo with organization and repo",
			input: map[string]any{
				"target-repo": "github/docs",
			},
			expected: "github/docs",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseTargetRepoFromConfig(tt.input)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

// Integration tests to verify the helpers work correctly in the parser functions

func TestParseIssuesConfigWithHelpers(t *testing.T) {
	compiler := &Compiler{}
	outputMap := map[string]any{
		"create-issue": map[string]any{
			"title-prefix": "[bot] ",
			"labels":       []any{"automation", "ai-generated"},
			"target-repo":  "owner/repo",
		},
	}

	result := compiler.parseIssuesConfig(outputMap)
	if result == nil {
		t.Fatal("expected non-nil result")
	}

	if result.TitlePrefix != "[bot] " {
		t.Errorf("expected title-prefix '[bot] ', got %q", result.TitlePrefix)
	}

	if len(result.Labels) != 2 {
		t.Errorf("expected 2 labels, got %d", len(result.Labels))
	}

	if result.TargetRepoSlug != "owner/repo" {
		t.Errorf("expected target-repo 'owner/repo', got %q", result.TargetRepoSlug)
	}
}

func TestParsePullRequestsConfigWithHelpers(t *testing.T) {
	compiler := &Compiler{}
	outputMap := map[string]any{
		"create-pull-request": map[string]any{
			"title-prefix":    "[auto] ",
			"labels":          []any{"automated", "needs-review"},
			"fallback-labels": []any{"failure", "automated"},
			"target-repo":     "org/project",
		},
	}

	result := compiler.parsePullRequestsConfig(outputMap)
	if result == nil {
		t.Fatal("expected non-nil result")
	}

	if result.TitlePrefix != "[auto] " {
		t.Errorf("expected title-prefix '[auto] ', got %q", result.TitlePrefix)
	}

	if len(result.Labels) != 2 {
		t.Errorf("expected 2 labels, got %d", len(result.Labels))
	}

	if len(result.FallbackLabels) != 2 {
		t.Errorf("expected 2 fallback labels, got %d", len(result.FallbackLabels))
	}

	if result.TargetRepoSlug != "org/project" {
		t.Errorf("expected target-repo 'org/project', got %q", result.TargetRepoSlug)
	}
}

func TestParsePullRequestsConfigExpires(t *testing.T) {
	tests := []struct {
		name          string
		expiresInput  any
		expectedHours int
	}{
		{
			name:          "integer days converted to hours",
			expiresInput:  14,
			expectedHours: 14 * 24,
		},
		{
			name:          "string duration converted to hours",
			expiresInput:  "7d",
			expectedHours: 7 * 24,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compiler := &Compiler{}
			outputMap := map[string]any{
				"create-pull-request": map[string]any{
					"expires": tt.expiresInput,
				},
			}

			result := compiler.parsePullRequestsConfig(outputMap)
			if result == nil {
				t.Fatal("expected non-nil result")
			}

			if result.Expires != tt.expectedHours {
				t.Errorf("expected expires %d hours, got %d", tt.expectedHours, result.Expires)
			}
		})
	}
}

func TestParseDiscussionsConfigWithHelpers(t *testing.T) {
	compiler := &Compiler{}
	outputMap := map[string]any{
		"create-discussion": map[string]any{
			"title-prefix": "[analysis] ",
			"target-repo":  "team/discussions",
		},
	}

	result := compiler.parseDiscussionsConfig(outputMap)
	if result == nil {
		t.Fatal("expected non-nil result")
	}

	if result.TitlePrefix != "[analysis] " {
		t.Errorf("expected title-prefix '[analysis] ', got %q", result.TitlePrefix)
	}

	if result.TargetRepoSlug != "team/discussions" {
		t.Errorf("expected target-repo 'team/discussions', got %q", result.TargetRepoSlug)
	}
}

func TestParseCommentsConfigWithHelpers(t *testing.T) {
	compiler := &Compiler{}
	outputMap := map[string]any{
		"add-comment": map[string]any{
			"target-repo": "upstream/project",
		},
	}

	result := compiler.parseCommentsConfig(outputMap)
	if result == nil {
		t.Fatal("expected non-nil result")
	}

	if result.TargetRepoSlug != "upstream/project" {
		t.Errorf("expected target-repo 'upstream/project', got %q", result.TargetRepoSlug)
	}
}

func TestParsePRReviewCommentsConfigWithHelpers(t *testing.T) {
	compiler := &Compiler{}
	outputMap := map[string]any{
		"create-pull-request-review-comment": map[string]any{
			"target-repo": "company/codebase",
		},
	}

	result := compiler.parsePullRequestReviewCommentsConfig(outputMap)
	if result == nil {
		t.Fatal("expected non-nil result")
	}

	if result.TargetRepoSlug != "company/codebase" {
		t.Errorf("expected target-repo 'company/codebase', got %q", result.TargetRepoSlug)
	}
}

// Test wildcard target-repo is now allowed for all create/close handlers

func TestParseIssuesConfigWithWildcardTargetRepo(t *testing.T) {
	compiler := &Compiler{}
	outputMap := map[string]any{
		"create-issue": map[string]any{
			"target-repo": "*",
		},
	}

	result := compiler.parseIssuesConfig(outputMap)
	if result == nil {
		t.Errorf("expected non-nil config for wildcard target-repo, got nil")
	} else if result.TargetRepoSlug != "*" {
		t.Errorf("expected TargetRepoSlug to be \"*\", got %q", result.TargetRepoSlug)
	}
}

func TestParsePullRequestsConfigWithWildcardTargetRepo(t *testing.T) {
	compiler := &Compiler{}
	outputMap := map[string]any{
		"create-pull-request": map[string]any{
			"target-repo": "*",
		},
	}

	result := compiler.parsePullRequestsConfig(outputMap)
	if result == nil {
		t.Errorf("expected non-nil config for wildcard target-repo, got nil")
	} else if result.TargetRepoSlug != "*" {
		t.Errorf("expected TargetRepoSlug to be \"*\", got %q", result.TargetRepoSlug)
	}
}

func TestParseDiscussionsConfigWithWildcardTargetRepo(t *testing.T) {
	compiler := &Compiler{}
	outputMap := map[string]any{
		"create-discussion": map[string]any{
			"target-repo": "*",
			"category":    "General",
		},
	}

	result := compiler.parseDiscussionsConfig(outputMap)
	if result == nil {
		t.Errorf("expected non-nil config for wildcard target-repo, got nil")
	} else if result.TargetRepoSlug != "*" {
		t.Errorf("expected TargetRepoSlug to be \"*\", got %q", result.TargetRepoSlug)
	}
}

func TestParseCommentsConfigWithWildcardTargetRepo(t *testing.T) {
	compiler := &Compiler{}
	outputMap := map[string]any{
		"add-comment": map[string]any{
			"target-repo": "*",
		},
	}

	result := compiler.parseCommentsConfig(outputMap)
	if result == nil {
		t.Errorf("expected non-nil config for wildcard target-repo, got nil")
	} else if result.TargetRepoSlug != "*" {
		t.Errorf("expected TargetRepoSlug to be \"*\", got %q", result.TargetRepoSlug)
	}
}

func TestParsePRReviewCommentsConfigWithWildcardTargetRepo(t *testing.T) {
	compiler := &Compiler{}
	outputMap := map[string]any{
		"create-pull-request-review-comment": map[string]any{
			"target-repo": "*",
		},
	}

	result := compiler.parsePullRequestReviewCommentsConfig(outputMap)
	if result != nil {
		t.Errorf("expected nil for wildcard target-repo, got %+v", result)
	}
}

func TestParseTargetRepoWithValidation(t *testing.T) {
	tests := []struct {
		name          string
		input         map[string]any
		expectedSlug  string
		expectedError bool
	}{
		{
			name: "valid target-repo",
			input: map[string]any{
				"target-repo": "owner/repo",
			},
			expectedSlug:  "owner/repo",
			expectedError: false,
		},
		{
			name: "empty target-repo",
			input: map[string]any{
				"target-repo": "",
			},
			expectedSlug:  "",
			expectedError: false,
		},
		{
			name:          "missing target-repo",
			input:         map[string]any{},
			expectedSlug:  "",
			expectedError: false,
		},
		{
			name: "wildcard target-repo (invalid)",
			input: map[string]any{
				"target-repo": "*",
			},
			expectedSlug:  "",
			expectedError: true,
		},
		{
			name: "target-repo with special characters",
			input: map[string]any{
				"target-repo": "github-next/gh-aw",
			},
			expectedSlug:  "github-next/gh-aw",
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			slug, isInvalid := parseTargetRepoWithValidation(tt.input)
			if slug != tt.expectedSlug {
				t.Errorf("expected slug %q, got %q", tt.expectedSlug, slug)
			}
			if isInvalid != tt.expectedError {
				t.Errorf("expected error %v, got %v", tt.expectedError, isInvalid)
			}
		})
	}
}

func TestParseBoolFromConfig(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]any
		key      string
		expected bool
	}{
		{
			name: "true value",
			input: map[string]any{
				"my-key": true,
			},
			key:      "my-key",
			expected: true,
		},
		{
			name: "false value",
			input: map[string]any{
				"my-key": false,
			},
			key:      "my-key",
			expected: false,
		},
		{
			name:     "missing key",
			input:    map[string]any{},
			key:      "my-key",
			expected: false,
		},
		{
			name: "non-bool type (string)",
			input: map[string]any{
				"my-key": "true",
			},
			key:      "my-key",
			expected: false,
		},
		{
			name: "non-bool type (int)",
			input: map[string]any{
				"my-key": 1,
			},
			key:      "my-key",
			expected: false,
		},
		{
			name: "non-bool type (int 0)",
			input: map[string]any{
				"my-key": 0,
			},
			key:      "my-key",
			expected: false,
		},
		{
			name: "non-bool type (array)",
			input: map[string]any{
				"my-key": []bool{true, false},
			},
			key:      "my-key",
			expected: false,
		},
		{
			name: "nil value",
			input: map[string]any{
				"my-key": nil,
			},
			key:      "my-key",
			expected: false,
		},
		{
			name: "different keys with different values",
			input: map[string]any{
				"key1": true,
				"key2": false,
			},
			key:      "key1",
			expected: true,
		},
		{
			name: "explicit false value should be preserved",
			input: map[string]any{
				"my-key": false,
			},
			key:      "my-key",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseBoolFromConfig(tt.input, tt.key, nil)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestPreprocessExpiresField(t *testing.T) {
	tests := []struct {
		name             string
		input            map[string]any
		expectedDisabled bool
		expectedValue    int
	}{
		{
			name: "valid integer days - converted to hours",
			input: map[string]any{
				"expires": 7,
			},
			expectedDisabled: false,
			expectedValue:    168, // 7 days * 24 hours
		},
		{
			name: "valid string format - 48h",
			input: map[string]any{
				"expires": "48h",
			},
			expectedDisabled: false,
			expectedValue:    48,
		},
		{
			name: "valid string format - 7d",
			input: map[string]any{
				"expires": "7d",
			},
			expectedDisabled: false,
			expectedValue:    168,
		},
		{
			name: "explicitly disabled with false",
			input: map[string]any{
				"expires": false,
			},
			expectedDisabled: true,
			expectedValue:    0,
		},
		{
			name: "invalid - true boolean",
			input: map[string]any{
				"expires": true,
			},
			expectedDisabled: false,
			expectedValue:    0,
		},
		{
			name: "invalid - 1 hour (below minimum)",
			input: map[string]any{
				"expires": "1h",
			},
			expectedDisabled: false,
			expectedValue:    0,
		},
		{
			name: "valid - 2 hours (at minimum)",
			input: map[string]any{
				"expires": "2h",
			},
			expectedDisabled: false,
			expectedValue:    2,
		},
		{
			name:             "no expires field",
			input:            map[string]any{},
			expectedDisabled: false,
			expectedValue:    0, // configData["expires"] not set when field missing
		},
		{
			name: "invalid string format",
			input: map[string]any{
				"expires": "invalid",
			},
			expectedDisabled: false,
			expectedValue:    0,
		},
		{
			name:             "nil configData",
			input:            nil,
			expectedDisabled: false,
			expectedValue:    0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Make a copy of input to check modification
			var configData map[string]any
			if tt.input != nil {
				configData = make(map[string]any)
				maps.Copy(configData, tt.input)
			}

			disabled := preprocessExpiresField(configData, nil)

			if disabled != tt.expectedDisabled {
				t.Errorf("expected disabled=%v, got %v", tt.expectedDisabled, disabled)
			}

			// Check that configData["expires"] was updated (if configData is not nil)
			if configData != nil && tt.input != nil {
				if _, exists := tt.input["expires"]; exists {
					expiresValue, ok := configData["expires"].(int)
					if !ok && configData["expires"] != nil {
						t.Errorf("expected expires to be int, got %T", configData["expires"])
					}
					if expiresValue != tt.expectedValue {
						t.Errorf("expected configData[\"expires\"]=%d, got %d", tt.expectedValue, expiresValue)
					}
				}
			}
		})
	}
}

func TestUnmarshalConfig(t *testing.T) {
	tests := []struct {
		name        string
		inputMap    map[string]any
		key         string
		expectError bool
		validate    func(*testing.T, *CreateIssuesConfig)
	}{
		{
			name: "valid config with all fields",
			inputMap: map[string]any{
				"create-issue": map[string]any{
					"title-prefix":   "[bot] ",
					"labels":         []any{"bug", "enhancement"},
					"allowed-labels": []any{"bug", "feature"},
					"assignees":      []any{"user1", "user2"},
					"target-repo":    "owner/repo",
					"allowed-repos":  []any{"owner/repo1", "owner/repo2"},
					"expires":        7,
					"max":            5,
					"github-token":   "${{ secrets.TOKEN }}",
				},
			},
			key:         "create-issue",
			expectError: false,
			validate: func(t *testing.T, config *CreateIssuesConfig) {
				if config.TitlePrefix != "[bot] " {
					t.Errorf("expected title-prefix '[bot] ', got %q", config.TitlePrefix)
				}
				if len(config.Labels) != 2 || config.Labels[0] != "bug" || config.Labels[1] != "enhancement" {
					t.Errorf("expected labels [bug, enhancement], got %v", config.Labels)
				}
				if len(config.AllowedLabels) != 2 {
					t.Errorf("expected 2 allowed-labels, got %d", len(config.AllowedLabels))
				}
				if len(config.Assignees) != 2 {
					t.Errorf("expected 2 assignees, got %d", len(config.Assignees))
				}
				if config.TargetRepoSlug != "owner/repo" {
					t.Errorf("expected target-repo 'owner/repo', got %q", config.TargetRepoSlug)
				}
				if len(config.AllowedRepos) != 2 {
					t.Errorf("expected 2 allowed-repos, got %d", len(config.AllowedRepos))
				}
				if config.Expires != 7 {
					t.Errorf("expected expires 7, got %d", config.Expires)
				}
				if templatableIntValue(config.Max) != 5 {
					t.Errorf("expected max 5, got %d", config.Max)
				}
				if config.GitHubToken != "${{ secrets.TOKEN }}" {
					t.Errorf("expected github-token, got %q", config.GitHubToken)
				}
			},
		},
		{
			name: "empty config (nil value)",
			inputMap: map[string]any{
				"create-issue": nil,
			},
			key:         "create-issue",
			expectError: false,
			validate: func(t *testing.T, config *CreateIssuesConfig) {
				// All fields should be zero values
				if config.TitlePrefix != "" {
					t.Errorf("expected empty title-prefix, got %q", config.TitlePrefix)
				}
				if len(config.Labels) != 0 {
					t.Errorf("expected no labels, got %v", config.Labels)
				}
			},
		},
		{
			name: "partial config",
			inputMap: map[string]any{
				"create-issue": map[string]any{
					"title-prefix": "[auto] ",
					"max":          3,
				},
			},
			key:         "create-issue",
			expectError: false,
			validate: func(t *testing.T, config *CreateIssuesConfig) {
				if config.TitlePrefix != "[auto] " {
					t.Errorf("expected title-prefix '[auto] ', got %q", config.TitlePrefix)
				}
				if templatableIntValue(config.Max) != 3 {
					t.Errorf("expected max 3, got %d", config.Max)
				}
				// Other fields should be zero values
				if len(config.Labels) != 0 {
					t.Errorf("expected no labels, got %v", config.Labels)
				}
			},
		},
		{
			name: "missing key",
			inputMap: map[string]any{
				"other-key": map[string]any{},
			},
			key:         "create-issue",
			expectError: true,
		},
		{
			name: "empty map",
			inputMap: map[string]any{
				"create-issue": map[string]any{},
			},
			key:         "create-issue",
			expectError: false,
			validate: func(t *testing.T, config *CreateIssuesConfig) {
				// All fields should be zero values
				if templatableIntValue(config.Max) != 0 {
					t.Errorf("expected max 0, got %d", config.Max)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var config CreateIssuesConfig
			err := unmarshalConfig(tt.inputMap, tt.key, &config, nil)

			if tt.expectError {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if tt.validate != nil {
				tt.validate(t, &config)
			}
		})
	}
}

// TestParseStringArrayOrExprFromConfig verifies that the expression-aware array helper
// accepts GitHub Actions expression strings in addition to the usual array types.
func TestParseStringArrayOrExprFromConfig(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]any
		key      string
		expected []string
	}{
		{
			name:     "nil map returns nil",
			input:    nil,
			key:      "labels",
			expected: nil,
		},
		{
			name:     "missing key returns nil",
			input:    map[string]any{},
			key:      "labels",
			expected: nil,
		},
		{
			name: "valid []string array",
			input: map[string]any{
				"labels": []string{"bug", "enhancement"},
			},
			key:      "labels",
			expected: []string{"bug", "enhancement"},
		},
		{
			name: "valid []any array",
			input: map[string]any{
				"labels": []any{"bug", "enhancement"},
			},
			key:      "labels",
			expected: []string{"bug", "enhancement"},
		},
		{
			name: "empty array returns empty slice",
			input: map[string]any{
				"labels": []string{},
			},
			key:      "labels",
			expected: []string{},
		},
		{
			name: "GitHub Actions expression string wrapped in single-element slice",
			input: map[string]any{
				"labels": "${{ inputs['required-labels'] }}",
			},
			key:      "labels",
			expected: []string{"${{ inputs['required-labels'] }}"},
		},
		{
			name: "expression with extra whitespace is still valid",
			input: map[string]any{
				"allowed-repos": "${{  inputs['allowed-repos']  }}",
			},
			key:      "allowed-repos",
			expected: []string{"${{  inputs['allowed-repos']  }}"},
		},
		{
			name: "non-expression bare string returns nil",
			input: map[string]any{
				"labels": "not-an-expression",
			},
			key:      "labels",
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseStringArrayOrExprFromConfig(tt.input, tt.key, nil)
			if tt.expected == nil {
				if result != nil {
					t.Errorf("expected nil, got %v", result)
				}
				return
			}
			if result == nil {
				t.Errorf("expected %v, got nil", tt.expected)
				return
			}
			if len(result) != len(tt.expected) {
				t.Errorf("expected length %d, got %d: %v", len(tt.expected), len(result), result)
				return
			}
			for i, v := range tt.expected {
				if result[i] != v {
					t.Errorf("element[%d]: expected %q, got %q", i, v, result[i])
				}
			}
		})
	}
}

// TestPreprocessStringArrayFieldAsTemplatable verifies that the expression-aware
// preprocessor wraps expression strings and rejects non-expression strings.
func TestPreprocessStringArrayFieldAsTemplatable(t *testing.T) {
	tests := []struct {
		name            string
		configData      map[string]any
		fieldName       string
		wantErr         bool
		wantErrContains []string // substrings the error message must contain
		wantWrapped     bool     // true when the value should be wrapped in []string{expr}
	}{
		{
			name:        "nil configData is a no-op",
			configData:  nil,
			fieldName:   "labels",
			wantErr:     false,
			wantWrapped: false,
		},
		{
			name:        "missing field is a no-op",
			configData:  map[string]any{},
			fieldName:   "labels",
			wantErr:     false,
			wantWrapped: false,
		},
		{
			name: "array value is left unchanged",
			configData: map[string]any{
				"labels": []string{"bug"},
			},
			fieldName:   "labels",
			wantErr:     false,
			wantWrapped: false,
		},
		{
			name: "expression string is wrapped",
			configData: map[string]any{
				"labels": "${{ inputs.labels }}",
			},
			fieldName:   "labels",
			wantErr:     false,
			wantWrapped: true,
		},
		{
			name: "non-expression string returns error with actionable message",
			configData: map[string]any{
				"labels": "automation",
			},
			fieldName:       "labels",
			wantErr:         true,
			wantErrContains: []string{"labels", "array", "expression"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := preprocessStringArrayFieldAsTemplatable(tt.configData, tt.fieldName, nil)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
					return
				}
				for _, substr := range tt.wantErrContains {
					if !strings.Contains(err.Error(), substr) {
						t.Errorf("error message %q does not contain expected substring %q", err.Error(), substr)
					}
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if tt.wantWrapped {
				val, exists := tt.configData[tt.fieldName]
				if !exists {
					t.Errorf("field %q disappeared from configData", tt.fieldName)
					return
				}
				wrapped, ok := val.([]string)
				if !ok {
					t.Errorf("expected []string, got %T: %v", val, val)
					return
				}
				if len(wrapped) != 1 {
					t.Errorf("expected single-element slice, got len=%d: %v", len(wrapped), wrapped)
					return
				}
				if !isExpression(wrapped[0]) {
					t.Errorf("expected expression in wrapped slice, got %q", wrapped[0])
				}
			}
		})
	}
}

// TestAddTemplatableStringSliceBuilder verifies the handler config builder method.
func TestAddTemplatableStringSliceBuilder(t *testing.T) {
	tests := []struct {
		name        string
		value       []string
		wantKey     bool
		wantAsArray bool   // true if stored as []string; false if stored as string
		wantValue   string // expected string when stored as expression
	}{
		{
			name:    "nil slice omits key",
			value:   nil,
			wantKey: false,
		},
		{
			name:    "empty slice omits key",
			value:   []string{},
			wantKey: false,
		},
		{
			name:        "literal slice stored as array",
			value:       []string{"bug", "enhancement"},
			wantKey:     true,
			wantAsArray: true,
		},
		{
			name:        "expression single-element stored as string",
			value:       []string{"${{ inputs.labels }}"},
			wantKey:     true,
			wantAsArray: false,
			wantValue:   "${{ inputs.labels }}",
		},
		{
			name:        "multi-element slice with expression-like string stored as array",
			value:       []string{"${{ inputs.labels }}", "bug"},
			wantKey:     true,
			wantAsArray: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := newHandlerConfigBuilder()
			b.AddTemplatableStringSlice("field", tt.value)
			cfg := b.Build()

			if !tt.wantKey {
				if _, exists := cfg["field"]; exists {
					t.Errorf("expected key to be absent, but it was present: %v", cfg["field"])
				}
				return
			}

			val, exists := cfg["field"]
			if !exists {
				t.Error("expected key to be present, but it was absent")
				return
			}

			if tt.wantAsArray {
				arr, ok := val.([]string)
				if !ok {
					t.Errorf("expected []string, got %T: %v", val, val)
					return
				}
				if len(arr) != len(tt.value) {
					t.Errorf("array length mismatch: expected %d, got %d", len(tt.value), len(arr))
				}
			} else {
				s, ok := val.(string)
				if !ok {
					t.Errorf("expected string (expression), got %T: %v", val, val)
					return
				}
				if s != tt.wantValue {
					t.Errorf("expression value: expected %q, got %q", tt.wantValue, s)
				}
			}
		})
	}
}

// TestParsePullRequestsConfigExpressionFields verifies that create-pull-request
// accepts GitHub Actions expression strings for labels, allowed-repos, and
// allowed-base-branches.
func TestParsePullRequestsConfigExpressionFields(t *testing.T) {
	tests := []struct {
		name     string
		field    string
		expr     string
		getField func(*CreatePullRequestsConfig) []string
	}{
		{
			name:     "labels as expression",
			field:    "labels",
			expr:     "${{ inputs.labels }}",
			getField: func(c *CreatePullRequestsConfig) []string { return c.Labels },
		},
		{
			name:     "allowed-repos as expression",
			field:    "allowed-repos",
			expr:     "${{ inputs['allowed-repos'] }}",
			getField: func(c *CreatePullRequestsConfig) []string { return c.AllowedRepos },
		},
		{
			name:     "allowed-base-branches as expression",
			field:    "allowed-base-branches",
			expr:     "${{ inputs['allowed-base-branches'] }}",
			getField: func(c *CreatePullRequestsConfig) []string { return c.AllowedBaseBranches },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compiler := &Compiler{}
			outputMap := map[string]any{
				"create-pull-request": map[string]any{
					tt.field: tt.expr,
				},
			}

			result := compiler.parsePullRequestsConfig(outputMap)
			if result == nil {
				t.Fatal("expected non-nil result")
			}

			got := tt.getField(result)
			if len(got) != 1 {
				t.Fatalf("expected single-element slice, got %v", got)
			}
			if got[0] != tt.expr {
				t.Errorf("expected expression %q, got %q", tt.expr, got[0])
			}
		})
	}
}

// TestParseAddCommentConfigExpressionFields verifies that add-comment accepts a
// GitHub Actions expression string for allowed-repos.
func TestParseAddCommentConfigExpressionFields(t *testing.T) {
	compiler := &Compiler{}
	expr := "${{ inputs['allowed-repos'] }}"
	outputMap := map[string]any{
		"add-comment": map[string]any{
			"allowed-repos": expr,
		},
	}

	result := compiler.parseCommentsConfig(outputMap)
	if result == nil {
		t.Fatal("expected non-nil result")
	}

	if len(result.AllowedRepos) != 1 {
		t.Fatalf("expected single-element AllowedRepos, got %v", result.AllowedRepos)
	}
	if result.AllowedRepos[0] != expr {
		t.Errorf("expected expression %q, got %q", expr, result.AllowedRepos[0])
	}
}

// TestParsePushToPullRequestBranchExpressionFields verifies that
// push-to-pull-request-branch accepts GitHub Actions expression strings for
// labels and allowed-repos.
func TestParsePushToPullRequestBranchExpressionFields(t *testing.T) {
	tests := []struct {
		name     string
		field    string
		expr     string
		getField func(*PushToPullRequestBranchConfig) []string
	}{
		{
			name:     "labels as expression",
			field:    "labels",
			expr:     "${{ inputs['required-labels'] }}",
			getField: func(c *PushToPullRequestBranchConfig) []string { return c.Labels },
		},
		{
			name:     "allowed-repos as expression",
			field:    "allowed-repos",
			expr:     "${{ inputs['allowed-repos'] }}",
			getField: func(c *PushToPullRequestBranchConfig) []string { return c.AllowedRepos },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compiler := &Compiler{}
			outputMap := map[string]any{
				"push-to-pull-request-branch": map[string]any{
					tt.field: tt.expr,
				},
			}

			result := compiler.parsePushToPullRequestBranchConfig(outputMap)
			if result == nil {
				t.Fatal("expected non-nil result")
			}

			got := tt.getField(result)
			if len(got) != 1 {
				t.Fatalf("expected single-element slice, got %v", got)
			}
			if got[0] != tt.expr {
				t.Errorf("expected expression %q, got %q", tt.expr, got[0])
			}
		})
	}
}

// TestHandlerConfigExpressionFields verifies that the handler config builder
// emits expression strings as JSON strings (not arrays) when a single-element
// expression slice is provided.
func TestHandlerConfigExpressionFields(t *testing.T) {
	tests := []struct {
		name      string
		safeOuts  *SafeOutputsConfig
		handler   string
		configKey string
		wantValue string // expected string in config; empty means wantArray=true
		wantArray bool
	}{
		{
			name: "create_pull_request labels expression stored as string",
			safeOuts: &SafeOutputsConfig{
				CreatePullRequests: &CreatePullRequestsConfig{
					Labels: []string{"${{ inputs.labels }}"},
				},
			},
			handler:   "create_pull_request",
			configKey: "labels",
			wantValue: "${{ inputs.labels }}",
		},
		{
			name: "create_pull_request labels literal stored as array",
			safeOuts: &SafeOutputsConfig{
				CreatePullRequests: &CreatePullRequestsConfig{
					Labels: []string{"bug", "enhancement"},
				},
			},
			handler:   "create_pull_request",
			configKey: "labels",
			wantArray: true,
		},
		{
			name: "create_pull_request allowed_repos expression stored as string",
			safeOuts: &SafeOutputsConfig{
				CreatePullRequests: &CreatePullRequestsConfig{
					AllowedRepos: []string{"${{ inputs['allowed-repos'] }}"},
				},
			},
			handler:   "create_pull_request",
			configKey: "allowed_repos",
			wantValue: "${{ inputs['allowed-repos'] }}",
		},
		{
			name: "create_pull_request allowed_base_branches expression stored as string",
			safeOuts: &SafeOutputsConfig{
				CreatePullRequests: &CreatePullRequestsConfig{
					AllowedBaseBranches: []string{"${{ inputs['allowed-base-branches'] }}"},
				},
			},
			handler:   "create_pull_request",
			configKey: "allowed_base_branches",
			wantValue: "${{ inputs['allowed-base-branches'] }}",
		},
		{
			name: "add_comment allowed_repos expression stored as string",
			safeOuts: &SafeOutputsConfig{
				AddComments: &AddCommentsConfig{
					AllowedRepos: []string{"${{ inputs['allowed-repos'] }}"},
				},
			},
			handler:   "add_comment",
			configKey: "allowed_repos",
			wantValue: "${{ inputs['allowed-repos'] }}",
		},
		{
			name: "push_to_pull_request_branch labels expression stored as string",
			safeOuts: &SafeOutputsConfig{
				PushToPullRequestBranch: &PushToPullRequestBranchConfig{
					Labels: []string{"${{ inputs['required-labels'] }}"},
				},
			},
			handler:   "push_to_pull_request_branch",
			configKey: "labels",
			wantValue: "${{ inputs['required-labels'] }}",
		},
		{
			name: "push_to_pull_request_branch allowed_repos expression stored as string",
			safeOuts: &SafeOutputsConfig{
				PushToPullRequestBranch: &PushToPullRequestBranchConfig{
					AllowedRepos: []string{"${{ inputs['allowed-repos'] }}"},
				},
			},
			handler:   "push_to_pull_request_branch",
			configKey: "allowed_repos",
			wantValue: "${{ inputs['allowed-repos'] }}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder, exists := handlerRegistry[tt.handler]
			if !exists {
				t.Fatalf("handler %q not found in registry", tt.handler)
			}

			cfg := builder(tt.safeOuts)
			if cfg == nil {
				t.Fatal("handler returned nil config")
			}

			val, exists := cfg[tt.configKey]
			if !exists {
				t.Fatalf("key %q not found in handler config; config: %v", tt.configKey, cfg)
			}

			if tt.wantArray {
				_, ok := val.([]string)
				if !ok {
					t.Errorf("expected []string, got %T: %v", val, val)
				}
			} else {
				s, ok := val.(string)
				if !ok {
					t.Errorf("expected string (expression), got %T: %v", val, val)
					return
				}
				if s != tt.wantValue {
					t.Errorf("expected %q, got %q", tt.wantValue, s)
				}
			}
		})
	}
}

// TestExpressionFieldsRejectedForInvalidStrings verifies that non-expression bare strings
// are rejected (return nil / return error) rather than silently accepted.
func TestExpressionFieldsRejectedForInvalidStrings(t *testing.T) {
	// create-pull-request: non-expression bare string returns nil config
	compiler := &Compiler{}

	for _, field := range []string{"labels", "allowed-repos", "allowed-base-branches"} {
		t.Run("create-pull-request/"+field+"/non-expression", func(t *testing.T) {
			outputMap := map[string]any{
				"create-pull-request": map[string]any{
					field: "not-an-expression",
				},
			}
			result := compiler.parsePullRequestsConfig(outputMap)
			// Must return nil (invalid input is rejected)
			if result != nil {
				t.Errorf("expected nil result for invalid bare string in %q, got %+v", field, result)
			}
		})
	}

	// add-comment: non-expression bare string returns nil config
	t.Run("add-comment/allowed-repos/non-expression", func(t *testing.T) {
		outputMap := map[string]any{
			"add-comment": map[string]any{
				"allowed-repos": "not-an-expression",
			},
		}
		result := compiler.parseCommentsConfig(outputMap)
		if result != nil {
			t.Errorf("expected nil result for invalid bare string in allowed-repos, got %+v", result)
		}
	})
}

// TestAllListEncodingForms verifies all supported list encodings for the templatable
// string-array safe-output fields, covering both compile-time (literal arrays) and
// runtime (expression strings) forms.
//
// Compile-time forms (supported by the Go parser):
//   - Literal array with multiple items
//   - Literal array with a single item
//   - Empty literal array (→ omitted from handler config)
//   - GitHub Actions expression string (bracket notation required for names with hyphens)
//
// Runtime forms (resolved by GitHub Actions before config.json is written, then parsed
// by JS handlers using comma-splitting):
//   - The expression resolves to a comma-separated string: "bug,enhancement"
//   - The expression resolves to a single value: "bug"
//   - The expression resolves to a value with spaces: "bug, enhancement"
//
// This test covers the Go compile-time pipeline only.  The JS runtime forms are
// covered by the existing JS handler tests (parseAllowedRepos, parseStringListConfig,
// parseAllowedBaseBranches).
func TestAllListEncodingForms(t *testing.T) {
	type listFieldCase struct {
		value     any      // raw value as it would appear in the config map
		wantSlice []string // expected []string in the parsed config struct; nil = field absent
	}

	// Helper: run a single field case through parsePullRequestsConfig.labels and
	// verify the Labels field.
	runPRLabelsCase := func(t *testing.T, tc listFieldCase) {
		t.Helper()
		compiler := &Compiler{}
		outputMap := map[string]any{
			"create-pull-request": map[string]any{
				"labels": tc.value,
			},
		}
		result := compiler.parsePullRequestsConfig(outputMap)
		if tc.wantSlice == nil {
			// Invalid input — parser should reject (nil config)
			if result != nil && len(result.Labels) > 0 {
				t.Errorf("expected empty/absent Labels, got %v", result.Labels)
			}
			return
		}
		if result == nil {
			t.Fatal("expected non-nil config")
		}
		if len(result.Labels) != len(tc.wantSlice) {
			t.Fatalf("Labels length: expected %d, got %d (%v)", len(tc.wantSlice), len(result.Labels), result.Labels)
		}
		for i, v := range tc.wantSlice {
			if result.Labels[i] != v {
				t.Errorf("Labels[%d]: expected %q, got %q", i, v, result.Labels[i])
			}
		}
	}

	// Helper: run a case through the handler builder and check the JSON output.
	runHandlerLabelsCase := func(t *testing.T, tc listFieldCase, wantKey bool, wantArray bool, wantExprValue string) {
		t.Helper()
		labels, ok := tc.value.([]string)
		if !ok {
			t.Skip("handler test requires []string input")
		}
		builder, exists := handlerRegistry["create_pull_request"]
		if !exists {
			t.Fatal("create_pull_request handler not found")
		}
		cfg := builder(&SafeOutputsConfig{
			CreatePullRequests: &CreatePullRequestsConfig{Labels: labels},
		})
		if cfg == nil {
			t.Fatal("handler returned nil config")
		}
		val, exists := cfg["labels"]
		if !wantKey {
			if exists {
				t.Errorf("expected 'labels' key absent, got %v", val)
			}
			return
		}
		if !exists {
			t.Errorf("expected 'labels' key present, but absent")
			return
		}
		if wantArray {
			if _, ok := val.([]string); !ok {
				t.Errorf("expected []string, got %T: %v", val, val)
			}
		} else {
			s, ok := val.(string)
			if !ok {
				t.Errorf("expected string (expression), got %T: %v", val, val)
				return
			}
			if s != wantExprValue {
				t.Errorf("expected expression %q, got %q", wantExprValue, s)
			}
		}
	}

	t.Run("parser/literal_multi_item_array", func(t *testing.T) {
		runPRLabelsCase(t, listFieldCase{
			value:     []any{"bug", "enhancement"},
			wantSlice: []string{"bug", "enhancement"},
		})
	})

	t.Run("parser/literal_single_item_array", func(t *testing.T) {
		runPRLabelsCase(t, listFieldCase{
			value:     []any{"automation"},
			wantSlice: []string{"automation"},
		})
	})

	t.Run("parser/literal_empty_array", func(t *testing.T) {
		runPRLabelsCase(t, listFieldCase{
			value:     []any{},
			wantSlice: []string{},
		})
	})

	t.Run("parser/expression_with_bracket_notation", func(t *testing.T) {
		expr := "${{ inputs['required-labels'] }}"
		runPRLabelsCase(t, listFieldCase{
			value:     expr,
			wantSlice: []string{expr},
		})
	})

	t.Run("parser/expression_vars_reference", func(t *testing.T) {
		expr := "${{ vars.PR_LABELS }}"
		runPRLabelsCase(t, listFieldCase{
			value:     expr,
			wantSlice: []string{expr},
		})
	})

	t.Run("parser/expression_with_fallback_operator", func(t *testing.T) {
		expr := "${{ inputs.labels || 'automation' }}"
		runPRLabelsCase(t, listFieldCase{
			value:     expr,
			wantSlice: []string{expr},
		})
	})

	t.Run("parser/expression_env_reference", func(t *testing.T) {
		expr := "${{ env.DEFAULT_LABELS }}"
		runPRLabelsCase(t, listFieldCase{
			value:     expr,
			wantSlice: []string{expr},
		})
	})

	// Handler builder forms — verify JSON config output.
	t.Run("builder/multi_item_array_stored_as_array", func(t *testing.T) {
		runHandlerLabelsCase(t, listFieldCase{value: []string{"bug", "enhancement"}}, true, true, "")
	})

	t.Run("builder/single_item_literal_stored_as_array", func(t *testing.T) {
		runHandlerLabelsCase(t, listFieldCase{value: []string{"automation"}}, true, true, "")
	})

	t.Run("builder/empty_array_omits_key", func(t *testing.T) {
		runHandlerLabelsCase(t, listFieldCase{value: []string{}}, false, false, "")
	})

	t.Run("builder/expression_string_stored_as_string", func(t *testing.T) {
		expr := "${{ inputs['required-labels'] }}"
		runHandlerLabelsCase(t, listFieldCase{value: []string{expr}}, true, false, expr)
	})

	t.Run("builder/vars_expression_stored_as_string", func(t *testing.T) {
		expr := "${{ vars.PR_LABELS }}"
		runHandlerLabelsCase(t, listFieldCase{value: []string{expr}}, true, false, expr)
	})

	// ParseStringArrayOrExprFromConfig — all resolved string forms.
	// These represent what the expression might resolve to at runtime (passed through
	// the Go helper when config is read from a pre-expanded config.json at startup time).
	t.Run("parseHelper/comma_separated_string_not_accepted_at_compile_time", func(t *testing.T) {
		// A comma-separated bare string is rejected at compile time (not an expression).
		// At runtime the expression resolves before Go code runs, so this case never
		// occurs in practice — but we document it explicitly.
		result := ParseStringArrayOrExprFromConfig(map[string]any{
			"labels": "automation,bot",
		}, "labels", nil)
		if result != nil {
			t.Errorf("expected nil for non-expression bare string, got %v", result)
		}
	})

	t.Run("parseHelper/expression_string_accepted", func(t *testing.T) {
		expr := "${{ inputs.labels }}"
		result := ParseStringArrayOrExprFromConfig(map[string]any{
			"labels": expr,
		}, "labels", nil)
		if len(result) != 1 || result[0] != expr {
			t.Errorf("expected [%q], got %v", expr, result)
		}
	})

	t.Run("parseHelper/string_array_all_forms", func(t *testing.T) {
		cases := []struct {
			name     string
			value    any
			wantLen  int
			wantVals []string
		}{
			{"single-string-array", []string{"automation"}, 1, []string{"automation"}},
			{"multi-string-array", []string{"automation", "bot"}, 2, []string{"automation", "bot"}},
			{"any-array", []any{"automation", "bot"}, 2, []string{"automation", "bot"}},
			{"empty-array", []string{}, 0, []string{}},
		}
		for _, c := range cases {
			t.Run(c.name, func(t *testing.T) {
				result := ParseStringArrayOrExprFromConfig(map[string]any{"labels": c.value}, "labels", nil)
				if len(result) != c.wantLen {
					t.Fatalf("expected len %d, got %d: %v", c.wantLen, len(result), result)
				}
				for i, v := range c.wantVals {
					if result[i] != v {
						t.Errorf("[%d]: expected %q, got %q", i, v, result[i])
					}
				}
			})
		}
	})

	// Error message content for hyphenated vs non-hyphenated field names.
	t.Run("errorMessage/hyphenated_field_uses_bracket_notation", func(t *testing.T) {
		err := preprocessStringArrayFieldAsTemplatable(
			map[string]any{"allowed-repos": "not-an-expr"},
			"allowed-repos", nil,
		)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		msg := err.Error()
		if !strings.Contains(msg, "inputs['allowed-repos']") {
			t.Errorf("error message should use bracket notation, got: %q", msg)
		}
	})

	t.Run("errorMessage/non_hyphenated_field_uses_dot_notation", func(t *testing.T) {
		err := preprocessStringArrayFieldAsTemplatable(
			map[string]any{"labels": "not-an-expr"},
			"labels", nil,
		)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		msg := err.Error()
		if !strings.Contains(msg, "inputs.labels") {
			t.Errorf("error message should use dot notation, got: %q", msg)
		}
	})
}
