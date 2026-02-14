//go:build !integration

package workflow

import (
	"testing"

	"github.com/github/gh-aw/pkg/constants"
)

func TestIsFeatureEnabled(t *testing.T) {
	tests := []struct {
		name     string
		envValue string
		flag     constants.FeatureFlag
		expected bool
	}{
		{
			name:     "feature enabled - single flag",
			envValue: "firewall",
			flag:     "firewall",
			expected: true,
		},
		{
			name:     "feature enabled - case insensitive",
			envValue: "FIREWALL",
			flag:     "firewall",
			expected: true,
		},
		{
			name:     "feature enabled - mixed case",
			envValue: "Firewall",
			flag:     "FIREWALL",
			expected: true,
		},
		{
			name:     "feature enabled - multiple flags",
			envValue: "feature1,firewall,feature2",
			flag:     "firewall",
			expected: true,
		},
		{
			name:     "feature enabled - with spaces",
			envValue: "feature1, firewall , feature2",
			flag:     "firewall",
			expected: true,
		},
		{
			name:     "feature disabled - empty env",
			envValue: "",
			flag:     "firewall",
			expected: false,
		},
		{
			name:     "feature disabled - not in list",
			envValue: "feature1,feature2",
			flag:     "firewall",
			expected: false,
		},
		{
			name:     "feature disabled - partial match",
			envValue: "firewall-extra",
			flag:     "firewall",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variable
			t.Setenv("GH_AW_FEATURES", tt.envValue)

			result := isFeatureEnabled(tt.flag, nil)
			if result != tt.expected {
				t.Errorf("isFeatureEnabled(%q, nil) with env=%q = %v, want %v",
					tt.flag, tt.envValue, result, tt.expected)
			}
		})
	}
}

func TestIsFeatureEnabledNoEnv(t *testing.T) {
	result := isFeatureEnabled(constants.FeatureFlag("firewall"), nil)
	if result != false {
		t.Errorf("isFeatureEnabled(\"firewall\", nil) with no env = %v, want false", result)
	}
}

func TestIsFeatureEnabledWithData(t *testing.T) {
	tests := []struct {
		name        string
		envValue    string
		frontmatter map[string]any
		flag        constants.FeatureFlag
		expected    bool
		description string
	}{
		{
			name:        "frontmatter takes precedence - enabled in frontmatter, disabled in env",
			envValue:    "",
			frontmatter: map[string]any{"firewall": true},
			flag:        "firewall",
			expected:    true,
			description: "When feature is in frontmatter, it should be enabled regardless of env",
		},
		{
			name:        "frontmatter takes precedence - disabled in frontmatter, enabled in env",
			envValue:    "firewall",
			frontmatter: map[string]any{"firewall": false},
			flag:        "firewall",
			expected:    false,
			description: "When feature is explicitly disabled in frontmatter, env should be ignored",
		},
		{
			name:        "fallback to env when not in frontmatter",
			envValue:    "firewall",
			frontmatter: map[string]any{"other-feature": true},
			flag:        "firewall",
			expected:    true,
			description: "When feature is not in frontmatter, should check env",
		},
		{
			name:        "disabled when not in frontmatter or env",
			envValue:    "",
			frontmatter: map[string]any{"other-feature": true},
			flag:        "firewall",
			expected:    false,
			description: "When feature is in neither frontmatter nor env, should be disabled",
		},
		{
			name:        "case insensitive frontmatter check",
			envValue:    "",
			frontmatter: map[string]any{"FIREWALL": true},
			flag:        "firewall",
			expected:    true,
			description: "Frontmatter feature check should be case insensitive",
		},
		{
			name:        "nil frontmatter falls back to env",
			envValue:    "firewall",
			frontmatter: nil,
			flag:        "firewall",
			expected:    true,
			description: "When frontmatter is nil, should check env",
		},
		{
			name:        "empty frontmatter falls back to env",
			envValue:    "firewall",
			frontmatter: map[string]any{},
			flag:        "firewall",
			expected:    true,
			description: "When frontmatter is empty, should check env",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variable
			if tt.envValue != "" {
				t.Setenv("GH_AW_FEATURES", tt.envValue)
			}

			// Create WorkflowData with features
			var workflowData *WorkflowData
			if tt.frontmatter != nil {
				workflowData = &WorkflowData{
					Features: tt.frontmatter,
				}
			}

			result := isFeatureEnabled(tt.flag, workflowData)
			if result != tt.expected {
				t.Errorf("%s: isFeatureEnabled(%q, %+v) with env=%q = %v, want %v",
					tt.description, tt.flag, tt.frontmatter, tt.envValue, result, tt.expected)
			}
		})
	}
}

func TestIsFeatureEnabledWithDataNilWorkflow(t *testing.T) {
	// Set environment variable
	t.Setenv("GH_AW_FEATURES", "firewall")

	// When workflowData is nil, should fall back to env
	result := isFeatureEnabled(constants.FeatureFlag("firewall"), nil)
	if result != true {
		t.Errorf("isFeatureEnabled(\"firewall\", nil) with env=firewall = %v, want true", result)
	}
}

// TestMergedFeaturesAreUsedByIsFeatureEnabled verifies that features merged from imports
// are accessible via isFeatureEnabled function
func TestMergedFeaturesAreUsedByIsFeatureEnabled(t *testing.T) {
	// Create workflow data with merged features (simulating the result of merging imports)
	workflowData := &WorkflowData{
		Features: map[string]any{
			"imported-feature":  true,
			"another-feature":   false,
			"string-feature":    "enabled",
			"top-level-feature": true,
		},
	}

	// Test that imported features are accessible via isFeatureEnabled
	tests := []struct {
		name     string
		flag     constants.FeatureFlag
		expected bool
	}{
		{
			name:     "imported feature enabled",
			flag:     "imported-feature",
			expected: true,
		},
		{
			name:     "imported feature disabled",
			flag:     "another-feature",
			expected: false,
		},
		{
			name:     "string feature treated as enabled",
			flag:     "string-feature",
			expected: true,
		},
		{
			name:     "top-level feature enabled",
			flag:     "top-level-feature",
			expected: true,
		},
		{
			name:     "non-existent feature",
			flag:     "non-existent",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isFeatureEnabled(tt.flag, workflowData)
			if result != tt.expected {
				t.Errorf("isFeatureEnabled(%q) = %v, want %v", tt.flag, result, tt.expected)
			}
		})
	}
}

// TestMergedFeaturesTopLevelPrecedence verifies that top-level features take precedence
// over imported features in the merged features map
func TestMergedFeaturesTopLevelPrecedence(t *testing.T) {
	// This test verifies that when features are merged, top-level features override imports
	// The actual merging happens in MergeFeatures function, but we test the end result here

	// Simulate a workflow where top-level feature overrides an imported one
	workflowData := &WorkflowData{
		Features: map[string]any{
			"override-feature": false, // Top-level value (overriding import that had true)
			"import-only":      true,  // Only from import
		},
	}

	// Verify that the overridden value is what isFeatureEnabled sees
	overrideResult := isFeatureEnabled(constants.FeatureFlag("override-feature"), workflowData)
	if overrideResult != false {
		t.Errorf("isFeatureEnabled(\"override-feature\") = %v, want false (top-level override)", overrideResult)
	}

	// Verify that import-only feature is still accessible
	importOnlyResult := isFeatureEnabled(constants.FeatureFlag("import-only"), workflowData)
	if importOnlyResult != true {
		t.Errorf("isFeatureEnabled(\"import-only\") = %v, want true (from import)", importOnlyResult)
	}
}
