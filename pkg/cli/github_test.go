package cli

import (
	"os"
	"testing"
)

func TestGetGitHubHost(t *testing.T) {
	tests := []struct {
		name         string
		serverURL    string
		ghHost       string
		expectedHost string
	}{
		{
			name:         "defaults to github.com",
			serverURL:    "",
			ghHost:       "",
			expectedHost: "https://github.com",
		},
		{
			name:         "uses GITHUB_SERVER_URL when set",
			serverURL:    "https://github.enterprise.com",
			ghHost:       "",
			expectedHost: "https://github.enterprise.com",
		},
		{
			name:         "uses GH_HOST when GITHUB_SERVER_URL not set",
			serverURL:    "",
			ghHost:       "https://github.company.com",
			expectedHost: "https://github.company.com",
		},
		{
			name:         "GITHUB_SERVER_URL takes precedence over GH_HOST",
			serverURL:    "https://github.enterprise.com",
			ghHost:       "https://github.company.com",
			expectedHost: "https://github.enterprise.com",
		},
		{
			name:         "removes trailing slash from GITHUB_SERVER_URL",
			serverURL:    "https://github.enterprise.com/",
			ghHost:       "",
			expectedHost: "https://github.enterprise.com",
		},
		{
			name:         "removes trailing slash from GH_HOST",
			serverURL:    "",
			ghHost:       "https://github.company.com/",
			expectedHost: "https://github.company.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original env vars
			originalServerURL := os.Getenv("GITHUB_SERVER_URL")
			originalGHHost := os.Getenv("GH_HOST")
			defer func() {
				if originalServerURL != "" {
					os.Setenv("GITHUB_SERVER_URL", originalServerURL)
				} else {
					os.Unsetenv("GITHUB_SERVER_URL")
				}
				if originalGHHost != "" {
					os.Setenv("GH_HOST", originalGHHost)
				} else {
					os.Unsetenv("GH_HOST")
				}
			}()

			// Set test env vars
			if tt.serverURL != "" {
				os.Setenv("GITHUB_SERVER_URL", tt.serverURL)
			} else {
				os.Unsetenv("GITHUB_SERVER_URL")
			}
			if tt.ghHost != "" {
				os.Setenv("GH_HOST", tt.ghHost)
			} else {
				os.Unsetenv("GH_HOST")
			}

			// Test
			host := getGitHubHost()
			if host != tt.expectedHost {
				t.Errorf("Expected host '%s', got '%s'", tt.expectedHost, host)
			}
		})
	}
}
