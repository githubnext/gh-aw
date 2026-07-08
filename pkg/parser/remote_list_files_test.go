//go:build !integration

package parser

import (
	"strings"
	"testing"
)

func TestResolveListRepoCloneConfig(t *testing.T) {
	tests := []struct {
		name    string
		owner   string
		repo    string
		ref     string
		host    string
		wantErr bool
		checkFn func(t *testing.T, cfg listRepoCloneConfig)
	}{
		{
			name:    "empty ref returns error",
			owner:   "o",
			repo:    "r",
			ref:     "",
			wantErr: true,
		},
		{
			name:    "whitespace-only ref returns error",
			owner:   "o",
			repo:    "r",
			ref:     "   ",
			wantErr: true,
		},
		{
			name:  "host override used in repoURL and cacheKey",
			owner: "o",
			repo:  "r",
			ref:   "main",
			host:  "ghes.example.com",
			checkFn: func(t *testing.T, cfg listRepoCloneConfig) {
				t.Helper()
				if !strings.Contains(cfg.repoURL, "ghes.example.com") {
					t.Errorf("expected ghes.example.com in repoURL, got %q", cfg.repoURL)
				}
				if !strings.Contains(cfg.cacheKey, "ghes.example.com") {
					t.Errorf("expected ghes.example.com in cacheKey, got %q", cfg.cacheKey)
				}
			},
		},
		{
			name:  "cache key contains all identity fields",
			owner: "myowner",
			repo:  "myrepo",
			ref:   "v1.2.3",
			host:  "ghes.example.com",
			checkFn: func(t *testing.T, cfg listRepoCloneConfig) {
				t.Helper()
				for _, part := range []string{"myowner", "myrepo", "v1.2.3"} {
					if !strings.Contains(cfg.cacheKey, part) {
						t.Errorf("missing %q in cacheKey %q", part, cfg.cacheKey)
					}
				}
			},
		},
		{
			name:  "owner and repo are set on config",
			owner: "myowner",
			repo:  "myrepo",
			ref:   "main",
			host:  "ghes.example.com",
			checkFn: func(t *testing.T, cfg listRepoCloneConfig) {
				t.Helper()
				if cfg.owner != "myowner" {
					t.Errorf("owner = %q, want %q", cfg.owner, "myowner")
				}
				if cfg.repo != "myrepo" {
					t.Errorf("repo = %q, want %q", cfg.repo, "myrepo")
				}
			},
		},
		{
			name:  "ref is trimmed of surrounding whitespace",
			owner: "o",
			repo:  "r",
			ref:   "  main  ",
			host:  "ghes.example.com",
			checkFn: func(t *testing.T, cfg listRepoCloneConfig) {
				t.Helper()
				if cfg.ref != "main" {
					t.Errorf("ref = %q, want %q", cfg.ref, "main")
				}
			},
		},
		{
			name:  "public github org uses github.com host",
			owner: "github",
			repo:  "myrepo",
			ref:   "main",
			host:  "",
			checkFn: func(t *testing.T, cfg listRepoCloneConfig) {
				t.Helper()
				if !strings.Contains(cfg.repoURL, "github.com") {
					t.Errorf("expected github.com in repoURL for public org, got %q", cfg.repoURL)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Isolate environment so GetGitHubHostForRepo uses a predictable host.
			t.Setenv("GITHUB_SERVER_URL", "")
			t.Setenv("GITHUB_ENTERPRISE_HOST", "")
			t.Setenv("GITHUB_HOST", "")
			t.Setenv("GH_HOST", "")

			cfg, err := resolveListRepoCloneConfig(tt.owner, tt.repo, tt.ref, tt.host)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("resolveListRepoCloneConfig() error = nil, want error")
				}
				return
			}
			if err != nil {
				t.Fatalf("resolveListRepoCloneConfig() unexpected error: %v", err)
			}
			if tt.checkFn != nil {
				tt.checkFn(t, cfg)
			}
		})
	}
}
