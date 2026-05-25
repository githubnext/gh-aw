//go:build !integration

package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

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
			assert.Equal(t, tt.expectedHost, host, "GetGitHubHostForRepo(%q, %q)", tt.owner, tt.repo)
		})
	}
}
