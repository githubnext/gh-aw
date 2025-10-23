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

			result := IsFeatureEnabled(tt.flag)
			if result != tt.expected {
				t.Errorf("IsFeatureEnabled(%q) with env=%q = %v, want %v",
					tt.flag, tt.envValue, result, tt.expected)
			}
		})
	}
}

func TestIsFeatureEnabledNoEnv(t *testing.T) {
	// Ensure environment variable is not set
	os.Unsetenv("GH_AW_FEATURES")

	result := IsFeatureEnabled("firewall")
	if result != false {
		t.Errorf("IsFeatureEnabled(\"firewall\") with no env = %v, want false", result)
	}
}
