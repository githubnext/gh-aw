package workflow

import (
	"os"
	"testing"
)

func TestIsFeatureEnabled(t *testing.T) {
	tests := []struct {
		name     string
		envValue string
		flag     string
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
			os.Setenv("GH_AW_FEATURES", tt.envValue)
			defer os.Unsetenv("GH_AW_FEATURES")

			result := isFeatureEnabled(tt.flag, nil)
			if result != tt.expected {
				t.Errorf("isFeatureEnabled(%q, nil) with env=%q = %v, want %v",
					tt.flag, tt.envValue, result, tt.expected)
			}
		})
	}
}

func TestIsFeatureEnabledNoEnv(t *testing.T) {
	// Ensure environment variable is not set
	os.Unsetenv("GH_AW_FEATURES")

	result := isFeatureEnabled("firewall", nil)
	if result != false {
		t.Errorf("isFeatureEnabled(\"firewall\", nil) with no env = %v, want false", result)
	}
}

func TestIsFeatureEnabledWithData(t *testing.T) {
	tests := []struct {
		name        string
		envValue    string
		frontmatter map[string]bool
		flag        string
		expected    bool
		description string
	}{
		{
			name:        "frontmatter takes precedence - enabled in frontmatter, disabled in env",
			envValue:    "",
			frontmatter: map[string]bool{"firewall": true},
			flag:        "firewall",
			expected:    true,
			description: "When feature is in frontmatter, it should be enabled regardless of env",
		},
		{
			name:        "frontmatter takes precedence - disabled in frontmatter, enabled in env",
			envValue:    "firewall",
			frontmatter: map[string]bool{"firewall": false},
			flag:        "firewall",
			expected:    false,
			description: "When feature is explicitly disabled in frontmatter, env should be ignored",
		},
		{
			name:        "fallback to env when not in frontmatter",
			envValue:    "firewall",
			frontmatter: map[string]bool{"other-feature": true},
			flag:        "firewall",
			expected:    true,
			description: "When feature is not in frontmatter, should check env",
		},
		{
			name:        "disabled when not in frontmatter or env",
			envValue:    "",
			frontmatter: map[string]bool{"other-feature": true},
			flag:        "firewall",
			expected:    false,
			description: "When feature is in neither frontmatter nor env, should be disabled",
		},
		{
			name:        "case insensitive frontmatter check",
			envValue:    "",
			frontmatter: map[string]bool{"FIREWALL": true},
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
			frontmatter: map[string]bool{},
			flag:        "firewall",
			expected:    true,
			description: "When frontmatter is empty, should check env",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variable
			if tt.envValue != "" {
				os.Setenv("GH_AW_FEATURES", tt.envValue)
				defer os.Unsetenv("GH_AW_FEATURES")
			} else {
				os.Unsetenv("GH_AW_FEATURES")
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
	// Ensure environment variable is set
	os.Setenv("GH_AW_FEATURES", "firewall")
	defer os.Unsetenv("GH_AW_FEATURES")

	// When workflowData is nil, should fall back to env
	result := isFeatureEnabled("firewall", nil)
	if result != true {
		t.Errorf("isFeatureEnabled(\"firewall\", nil) with env=firewall = %v, want true", result)
	}
}
