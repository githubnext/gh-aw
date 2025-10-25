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

func TestFirewallDefaultEnabledForCopilot(t *testing.T) {
	// Clear environment to ensure we're testing default behavior
	os.Unsetenv("GH_AW_FEATURES")
	defer os.Unsetenv("GH_AW_FEATURES")

	tests := []struct {
		name         string
		engineID     string
		features     map[string]bool
		expectedFlag bool
		description  string
	}{
		{
			name:         "copilot engine - firewall enabled by default",
			engineID:     "copilot",
			features:     nil,
			expectedFlag: true,
			description:  "Firewall should be enabled by default for copilot engine when no features are set",
		},
		{
			name:         "copilot engine - firewall enabled by default with empty features",
			engineID:     "copilot",
			features:     map[string]bool{},
			expectedFlag: true,
			description:  "Firewall should be enabled by default for copilot engine with empty features map",
		},
		{
			name:         "copilot engine - firewall explicitly enabled",
			engineID:     "copilot",
			features:     map[string]bool{"firewall": true},
			expectedFlag: true,
			description:  "Firewall should be enabled when explicitly set to true",
		},
		{
			name:         "copilot engine - firewall explicitly disabled",
			engineID:     "copilot",
			features:     map[string]bool{"firewall": false},
			expectedFlag: false,
			description:  "Firewall should respect explicit disable even for copilot engine",
		},
		{
			name:         "copilot engine - other features don't affect default",
			engineID:     "copilot",
			features:     map[string]bool{"some-other-feature": true},
			expectedFlag: true,
			description:  "Firewall should still be enabled by default when other features are set",
		},
		{
			name:         "claude engine - firewall NOT enabled by default",
			engineID:     "claude",
			features:     nil,
			expectedFlag: false,
			description:  "Firewall should NOT be enabled by default for claude engine",
		},
		{
			name:         "claude engine - firewall explicitly enabled",
			engineID:     "claude",
			features:     map[string]bool{"firewall": true},
			expectedFlag: true,
			description:  "Firewall can be explicitly enabled for claude engine",
		},
		{
			name:         "codex engine - firewall NOT enabled by default",
			engineID:     "codex",
			features:     nil,
			expectedFlag: false,
			description:  "Firewall should NOT be enabled by default for codex engine",
		},
		{
			name:         "custom engine - firewall NOT enabled by default",
			engineID:     "custom",
			features:     nil,
			expectedFlag: false,
			description:  "Firewall should NOT be enabled by default for custom engine",
		},
		{
			name:         "nil engine config - firewall not enabled",
			engineID:     "",
			features:     nil,
			expectedFlag: false,
			description:  "When no engine is configured, firewall should not be enabled",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var workflowData *WorkflowData
			if tt.engineID != "" {
				workflowData = &WorkflowData{
					EngineConfig: &EngineConfig{
						ID: tt.engineID,
					},
					Features: tt.features,
				}
			} else {
				// Test with nil engine config
				workflowData = &WorkflowData{
					Features: tt.features,
				}
			}

			result := isFeatureEnabled("firewall", workflowData)
			if result != tt.expectedFlag {
				t.Errorf("%s: expected firewall=%v, got %v", tt.description, tt.expectedFlag, result)
			}
		})
	}
}

func TestFirewallDefaultWithEnvironment(t *testing.T) {
	tests := []struct {
		name         string
		engineID     string
		features     map[string]bool
		envValue     string
		expectedFlag bool
		description  string
	}{
		{
			name:         "copilot with env disabled - defaults to enabled",
			engineID:     "copilot",
			features:     nil,
			envValue:     "",
			expectedFlag: true,
			description:  "Copilot defaults to firewall enabled even without env",
		},
		{
			name:         "copilot with env enabled - stays enabled",
			engineID:     "copilot",
			features:     nil,
			envValue:     "firewall",
			expectedFlag: true,
			description:  "Copilot stays enabled when env also enables it",
		},
		{
			name:         "copilot explicitly disabled - respects frontmatter over default",
			engineID:     "copilot",
			features:     map[string]bool{"firewall": false},
			envValue:     "firewall",
			expectedFlag: false,
			description:  "Explicit frontmatter disable takes precedence over env and default",
		},
		{
			name:         "claude with env enabled - uses env",
			engineID:     "claude",
			features:     nil,
			envValue:     "firewall",
			expectedFlag: true,
			description:  "Non-copilot engines can enable via env",
		},
		{
			name:         "claude with env disabled - stays disabled",
			engineID:     "claude",
			features:     nil,
			envValue:     "",
			expectedFlag: false,
			description:  "Non-copilot engines default to disabled",
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

			workflowData := &WorkflowData{
				EngineConfig: &EngineConfig{
					ID: tt.engineID,
				},
				Features: tt.features,
			}

			result := isFeatureEnabled("firewall", workflowData)
			if result != tt.expectedFlag {
				t.Errorf("%s: expected firewall=%v, got %v (env=%q)", tt.description, tt.expectedFlag, result, tt.envValue)
			}
		})
	}
}

func TestOtherFeaturesNotAffectedByCopilotDefault(t *testing.T) {
	// Clear environment to ensure we're testing default behavior
	os.Unsetenv("GH_AW_FEATURES")
	defer os.Unsetenv("GH_AW_FEATURES")

	workflowData := &WorkflowData{
		EngineConfig: &EngineConfig{
			ID: "copilot",
		},
		Features: nil,
	}

	// Test that other features (not firewall) are not affected by copilot's default
	result := isFeatureEnabled("some-other-feature", workflowData)
	if result != false {
		t.Errorf("Non-firewall features should not be enabled by default for copilot, got %v", result)
	}

	// Test that firewall is enabled for copilot
	firewallResult := isFeatureEnabled("firewall", workflowData)
	if firewallResult != true {
		t.Errorf("Firewall should be enabled by default for copilot, got %v", firewallResult)
	}
}
