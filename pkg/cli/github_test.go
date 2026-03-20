//go:build !integration

package cli

import (
	"testing"
)

func TestGetGitHubHost(t *testing.T) {
	tests := []struct {
		name           string
		serverURL      string
		enterpriseHost string
		githubHost     string
		ghHost         string
		expectedHost   string
	}{
		{
			name:           "defaults to github.com",
			serverURL:      "",
			enterpriseHost: "",
			githubHost:     "",
			ghHost:         "",
			expectedHost:   "https://github.com",
		},
		{
			name:           "uses GITHUB_SERVER_URL when set",
			serverURL:      "https://github.enterprise.com",
			enterpriseHost: "",
			githubHost:     "",
			ghHost:         "",
			expectedHost:   "https://github.enterprise.com",
		},
		{
			name:           "uses GITHUB_ENTERPRISE_HOST when GITHUB_SERVER_URL not set",
			serverURL:      "",
			enterpriseHost: "github.enterprise.com",
			githubHost:     "",
			ghHost:         "",
			expectedHost:   "https://github.enterprise.com",
		},
		{
			name:           "uses GITHUB_HOST when GITHUB_SERVER_URL and GITHUB_ENTERPRISE_HOST not set",
			serverURL:      "",
			enterpriseHost: "",
			githubHost:     "github.company.com",
			ghHost:         "",
			expectedHost:   "https://github.company.com",
		},
		{
			name:           "uses GH_HOST when other vars not set",
			serverURL:      "",
			enterpriseHost: "",
			githubHost:     "",
			ghHost:         "https://github.company.com",
			expectedHost:   "https://github.company.com",
		},
		{
			name:           "GITHUB_SERVER_URL takes precedence over all others",
			serverURL:      "https://github.enterprise.com",
			enterpriseHost: "github.other.com",
			githubHost:     "github.another.com",
			ghHost:         "https://github.company.com",
			expectedHost:   "https://github.enterprise.com",
		},
		{
			name:           "GITHUB_ENTERPRISE_HOST takes precedence over GITHUB_HOST and GH_HOST",
			serverURL:      "",
			enterpriseHost: "github.enterprise.com",
			githubHost:     "github.company.com",
			ghHost:         "github.other.com",
			expectedHost:   "https://github.enterprise.com",
		},
		{
			name:           "GITHUB_HOST takes precedence over GH_HOST",
			serverURL:      "",
			enterpriseHost: "",
			githubHost:     "github.company.com",
			ghHost:         "github.other.com",
			expectedHost:   "https://github.company.com",
		},
		{
			name:           "removes trailing slash from GITHUB_SERVER_URL",
			serverURL:      "https://github.enterprise.com/",
			enterpriseHost: "",
			githubHost:     "",
			ghHost:         "",
			expectedHost:   "https://github.enterprise.com",
		},
		{
			name:           "removes trailing slash from GITHUB_ENTERPRISE_HOST",
			serverURL:      "",
			enterpriseHost: "github.enterprise.com/",
			githubHost:     "",
			ghHost:         "",
			expectedHost:   "https://github.enterprise.com",
		},
		{
			name:           "removes trailing slash from GH_HOST",
			serverURL:      "",
			enterpriseHost: "",
			githubHost:     "",
			ghHost:         "https://github.company.com/",
			expectedHost:   "https://github.company.com",
		},
		{
			name:           "adds https:// prefix to GITHUB_ENTERPRISE_HOST",
			serverURL:      "",
			enterpriseHost: "MYORG.ghe.com",
			githubHost:     "",
			ghHost:         "",
			expectedHost:   "https://MYORG.ghe.com",
		},
		{
			name:           "adds https:// prefix to GITHUB_HOST",
			serverURL:      "",
			enterpriseHost: "",
			githubHost:     "MYORG.ghe.com",
			ghHost:         "",
			expectedHost:   "https://MYORG.ghe.com",
		},
		{
			name:           "adds https:// prefix to GH_HOST",
			serverURL:      "",
			enterpriseHost: "",
			githubHost:     "",
			ghHost:         "MYORG.ghe.com",
			expectedHost:   "https://MYORG.ghe.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set test env vars (always set to ensure clean state)
			t.Setenv("GITHUB_SERVER_URL", tt.serverURL)
			t.Setenv("GITHUB_ENTERPRISE_HOST", tt.enterpriseHost)
			t.Setenv("GITHUB_HOST", tt.githubHost)
			t.Setenv("GH_HOST", tt.ghHost)

			// Test
			host := getGitHubHost()
			if host != tt.expectedHost {
				t.Errorf("Expected host '%s', got '%s'", tt.expectedHost, host)
			}
		})
	}
}

func TestGetGitHubHostForRepo(t *testing.T) {
	tests := []struct {
		name         string
		repo         string
		gheHost      string
		expectedHost string
	}{
		{
			name:         "non-allowlisted repo uses configured public host",
			repo:         "owner/repo",
			gheHost:      "",
			expectedHost: "https://github.com",
		},
		{
			name:         "non-allowlisted repo uses GHE host",
			repo:         "owner/repo",
			gheHost:      "myorg.ghe.com",
			expectedHost: "https://myorg.ghe.com",
		},
		{
			name:         "github/gh-aw always uses public GitHub",
			repo:         "github/gh-aw",
			gheHost:      "myorg.ghe.com",
			expectedHost: "https://github.com",
		},
		{
			name:         "github/gh-aw with path always uses public GitHub",
			repo:         "github/gh-aw/workflows/ci-doctor.md",
			gheHost:      "myorg.ghe.com",
			expectedHost: "https://github.com",
		},
		{
			name:         "githubnext/agentics always uses public GitHub",
			repo:         "githubnext/agentics",
			gheHost:      "myorg.ghe.com",
			expectedHost: "https://github.com",
		},
		{
			name:         "githubnext/agentics with path always uses public GitHub",
			repo:         "githubnext/agentics/workflows/daily-plan.md",
			gheHost:      "myorg.ghe.com",
			expectedHost: "https://github.com",
		},
		{
			name:         "githubnext/agentics without GHE host uses public GitHub",
			repo:         "githubnext/agentics",
			gheHost:      "",
			expectedHost: "https://github.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("GITHUB_SERVER_URL", "")
			t.Setenv("GITHUB_ENTERPRISE_HOST", tt.gheHost)
			t.Setenv("GITHUB_HOST", "")
			t.Setenv("GH_HOST", "")

			host := getGitHubHostForRepo(tt.repo)
			if host != tt.expectedHost {
				t.Errorf("getGitHubHostForRepo(%q) = %q, want %q", tt.repo, host, tt.expectedHost)
			}
		})
	}
}
