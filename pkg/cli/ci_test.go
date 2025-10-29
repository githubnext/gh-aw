package cli

import (
	"os"
	"testing"
)

func TestIsRunningInCI(t *testing.T) {
	tests := []struct {
		name     string
		envVars  map[string]string
		expected bool
	}{
		{
			name:     "No CI environment variables",
			envVars:  map[string]string{},
			expected: false,
		},
		{
			name: "CI variable set",
			envVars: map[string]string{
				"CI": "true",
			},
			expected: true,
		},
		{
			name: "CONTINUOUS_INTEGRATION variable set",
			envVars: map[string]string{
				"CONTINUOUS_INTEGRATION": "true",
			},
			expected: true,
		},
		{
			name: "GITHUB_ACTIONS variable set",
			envVars: map[string]string{
				"GITHUB_ACTIONS": "true",
			},
			expected: true,
		},
		{
			name: "Multiple CI variables set",
			envVars: map[string]string{
				"CI":             "true",
				"GITHUB_ACTIONS": "true",
			},
			expected: true,
		},
		{
			name: "CI variable set to empty string",
			envVars: map[string]string{
				"CI": "",
			},
			expected: false,
		},
		{
			name: "Other environment variables set (not CI)",
			envVars: map[string]string{
				"PATH":  "/usr/bin",
				"HOME":  "/home/user",
				"SHELL": "/bin/bash",
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save and clear existing CI environment variables
			origCI := os.Getenv("CI")
			origContinuous := os.Getenv("CONTINUOUS_INTEGRATION")
			origGitHubActions := os.Getenv("GITHUB_ACTIONS")

			os.Unsetenv("CI")
			os.Unsetenv("CONTINUOUS_INTEGRATION")
			os.Unsetenv("GITHUB_ACTIONS")

			// Restore at the end
			defer func() {
				if origCI != "" {
					os.Setenv("CI", origCI)
				} else {
					os.Unsetenv("CI")
				}
				if origContinuous != "" {
					os.Setenv("CONTINUOUS_INTEGRATION", origContinuous)
				} else {
					os.Unsetenv("CONTINUOUS_INTEGRATION")
				}
				if origGitHubActions != "" {
					os.Setenv("GITHUB_ACTIONS", origGitHubActions)
				} else {
					os.Unsetenv("GITHUB_ACTIONS")
				}
			}()

			// Set test environment variables
			for k, v := range tt.envVars {
				if v != "" {
					os.Setenv(k, v)
				}
			}

			got := IsRunningInCI()
			if got != tt.expected {
				t.Errorf("IsRunningInCI() = %v, want %v", got, tt.expected)
			}
		})
	}
}
