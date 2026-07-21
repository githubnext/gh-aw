//go:build !integration

package parser

import (
	"testing"

	"github.com/stretchr/testify/require"
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
			name:           "GITHUB_SERVER_URL wins over others",
			serverURL:      "acme.ghe.com/redacted",
			enterpriseHost: "enterprise.ghe.com",
			githubHost:     "github-host.ghe.com",
			ghHost:         "gh-host.ghe.com",
			expectedHost:   "https://acme.ghe.com/redacted",
		},
		{
			name:           "GITHUB_ENTERPRISE_HOST wins over GITHUB_HOST and GH_HOST",
			serverURL:      "",
			enterpriseHost: "acme.ghe.com",
			githubHost:     "github-host.ghe.com",
			ghHost:         "gh-host.ghe.com",
			expectedHost:   "https://acme.ghe.com",
		},
		{
			name:           "GITHUB_HOST wins over GH_HOST",
			serverURL:      "",
			enterpriseHost: "",
			githubHost:     "acme.ghe.com/",
			ghHost:         "gh-host.ghe.com",
			expectedHost:   "https://acme.ghe.com",
		},
		{
			name:           "GH_HOST used when others are empty",
			serverURL:      "",
			enterpriseHost: "",
			githubHost:     "",
			ghHost:         "acme.ghe.com",
			expectedHost:   "https://acme.ghe.com",
		},
		{
			name:           "all vars empty falls back to github.com",
			serverURL:      "",
			enterpriseHost: "",
			githubHost:     "",
			ghHost:         "",
			expectedHost:   "https://github.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("GITHUB_SERVER_URL", tt.serverURL)
			t.Setenv("GITHUB_ENTERPRISE_HOST", tt.enterpriseHost)
			t.Setenv("GITHUB_HOST", tt.githubHost)
			t.Setenv("GH_HOST", tt.ghHost)

			host := GetGitHubHost()
			require.Equal(t, tt.expectedHost, host)
		})
	}
}

func TestGetGitHubHostForRepo_PublicOrgFallback(t *testing.T) {
	tests := []struct {
		name         string
		owner        string
		repo         string
		gheHost      string
		expectedHost string
	}{
		{
			name:         "non-fallback owner uses configured host",
			owner:        "acme",
			repo:         "repo",
			gheHost:      "myorg.ghe.com",
			expectedHost: "https://myorg.ghe.com",
		},
		{
			name:         "empty gheHost falls back to public for non-fallback owner",
			owner:        "acme",
			repo:         "repo",
			gheHost:      "",
			expectedHost: "https://github.com",
		},
		{
			name:         "github owner uses public host",
			owner:        "github",
			repo:         "copilot",
			gheHost:      "myorg.ghe.com",
			expectedHost: "https://github.com",
		},
		{
			name:         "githubnext owner uses public host",
			owner:        "githubnext",
			repo:         "agentics",
			gheHost:      "myorg.ghe.com",
			expectedHost: "https://github.com",
		},
		{
			name:         "microsoft owner uses public host",
			owner:        "microsoft",
			repo:         "vscode",
			gheHost:      "myorg.ghe.com",
			expectedHost: "https://github.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("GITHUB_SERVER_URL", "")
			t.Setenv("GITHUB_ENTERPRISE_HOST", tt.gheHost)
			t.Setenv("GITHUB_HOST", "")
			t.Setenv("GH_HOST", "")

			host := GetGitHubHostForRepo(tt.owner, tt.repo)
			require.Equal(t, tt.expectedHost, host, "GetGitHubHostForRepo(%q, %q)", tt.owner, tt.repo)
		})
	}
}
