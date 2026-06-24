//go:build !integration

package cli

import "testing"

func TestTrialRepositoryURLHelpers(t *testing.T) {
	tests := []struct {
		name               string
		serverURL          string
		enterpriseHost     string
		githubHost         string
		ghHost             string
		repoSlug           string
		expectedRepoURL    string
		expectedGitURL     string
		expectedActionsURL string
	}{
		{
			name:               "defaults to github.com",
			repoSlug:           "owner/repo",
			expectedRepoURL:    "https://github.com/owner/repo",
			expectedGitURL:     "https://github.com/owner/repo.git",
			expectedActionsURL: "https://github.com/owner/repo/settings/actions",
		},
		{
			name:               "uses GH_HOST for trial repository URLs",
			ghHost:             "example.ghe.com",
			repoSlug:           "owner/repo",
			expectedRepoURL:    "https://example.ghe.com/owner/repo",
			expectedGitURL:     "https://example.ghe.com/owner/repo.git",
			expectedActionsURL: "https://example.ghe.com/owner/repo/settings/actions",
		},
		{
			name:               "GITHUB_SERVER_URL takes precedence over GH_HOST",
			serverURL:          "https://server.ghe.com/",
			ghHost:             "example.ghe.com",
			repoSlug:           "owner/repo",
			expectedRepoURL:    "https://server.ghe.com/owner/repo",
			expectedGitURL:     "https://server.ghe.com/owner/repo.git",
			expectedActionsURL: "https://server.ghe.com/owner/repo/settings/actions",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("GITHUB_SERVER_URL", tt.serverURL)
			t.Setenv("GITHUB_ENTERPRISE_HOST", tt.enterpriseHost)
			t.Setenv("GITHUB_HOST", tt.githubHost)
			t.Setenv("GH_HOST", tt.ghHost)

			if got := trialRepositoryURL(tt.repoSlug); got != tt.expectedRepoURL {
				t.Fatalf("trialRepositoryURL() = %q, want %q", got, tt.expectedRepoURL)
			}
			if got := trialRepositoryGitURL(tt.repoSlug); got != tt.expectedGitURL {
				t.Fatalf("trialRepositoryGitURL() = %q, want %q", got, tt.expectedGitURL)
			}
			if got := trialRepositoryActionsSettingsURL(tt.repoSlug); got != tt.expectedActionsURL {
				t.Fatalf("trialRepositoryActionsSettingsURL() = %q, want %q", got, tt.expectedActionsURL)
			}
		})
	}
}
