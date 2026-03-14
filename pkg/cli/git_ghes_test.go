//go:build !integration

package cli

import (
	"os"
	"os/exec"
	"testing"

	"github.com/github/gh-aw/pkg/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsGHESInstance(t *testing.T) {
	tests := []struct {
		name       string
		remoteURL  string
		wantIsGHES bool
	}{
		{
			name:       "public GitHub",
			remoteURL:  "https://github.com/org/repo.git",
			wantIsGHES: false,
		},
		{
			name:       "GHES instance",
			remoteURL:  "https://ghes.example.com/org/repo.git",
			wantIsGHES: true,
		},
		{
			name:       "GHES SSH format",
			remoteURL:  "git@ghes.example.com:org/repo.git",
			wantIsGHES: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := testutil.TempDir(t, "test-*")
			originalDir, err := os.Getwd()
			require.NoError(t, err, "Failed to get current directory")
			defer func() {
				_ = os.Chdir(originalDir)
			}()

			require.NoError(t, os.Chdir(tmpDir), "Failed to change to temp directory")

			// Initialize git repo
			require.NoError(t, exec.Command("git", "init").Run(), "Failed to init git repo")
			exec.Command("git", "config", "user.name", "Test User").Run()
			exec.Command("git", "config", "user.email", "test@example.com").Run()

			// Set remote URL
			require.NoError(t, exec.Command("git", "remote", "add", "origin", tt.remoteURL).Run(), "Failed to add remote")
			defer func() { _ = exec.Command("git", "remote", "remove", "origin").Run() }()

			got := isGHESInstance()
			assert.Equal(t, tt.wantIsGHES, got, "isGHESInstance() returned unexpected result")
		})
	}
}

func TestGetGHESAPIURL(t *testing.T) {
	tests := []struct {
		name       string
		remoteURL  string
		wantAPIURL string
	}{
		{
			name:       "public GitHub returns empty",
			remoteURL:  "https://github.com/org/repo.git",
			wantAPIURL: "",
		},
		{
			name:       "GHES instance returns API URL",
			remoteURL:  "https://ghes.example.com/org/repo.git",
			wantAPIURL: "https://ghes.example.com/api/v3",
		},
		{
			name:       "GHES SSH format returns API URL",
			remoteURL:  "git@contoso.ghe.com:org/repo.git",
			wantAPIURL: "https://contoso.ghe.com/api/v3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := testutil.TempDir(t, "test-*")
			originalDir, err := os.Getwd()
			require.NoError(t, err, "Failed to get current directory")
			defer func() {
				_ = os.Chdir(originalDir)
			}()

			require.NoError(t, os.Chdir(tmpDir), "Failed to change to temp directory")

			// Initialize git repo
			require.NoError(t, exec.Command("git", "init").Run(), "Failed to init git repo")
			exec.Command("git", "config", "user.name", "Test User").Run()
			exec.Command("git", "config", "user.email", "test@example.com").Run()

			// Set remote URL
			require.NoError(t, exec.Command("git", "remote", "add", "origin", tt.remoteURL).Run(), "Failed to add remote")
			defer func() { _ = exec.Command("git", "remote", "remove", "origin").Run() }()

			got := getGHESAPIURL()
			assert.Equal(t, tt.wantAPIURL, got, "getGHESAPIURL() returned unexpected result")
		})
	}
}

func TestGetGHESAllowedDomains(t *testing.T) {
	tests := []struct {
		name        string
		remoteURL   string
		wantDomains []string
	}{
		{
			name:        "public GitHub returns nil",
			remoteURL:   "https://github.com/org/repo.git",
			wantDomains: nil,
		},
		{
			name:      "GHES instance returns host and api subdomain",
			remoteURL: "https://ghes.example.com/org/repo.git",
			wantDomains: []string{
				"ghes.example.com",
				"api.ghes.example.com",
			},
		},
		{
			name:      "GHES SSH format returns domains",
			remoteURL: "git@contoso-aw.ghe.com:org/repo.git",
			wantDomains: []string{
				"contoso-aw.ghe.com",
				"api.contoso-aw.ghe.com",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := testutil.TempDir(t, "test-*")
			originalDir, err := os.Getwd()
			require.NoError(t, err, "Failed to get current directory")
			defer func() {
				_ = os.Chdir(originalDir)
			}()

			require.NoError(t, os.Chdir(tmpDir), "Failed to change to temp directory")

			// Initialize git repo
			require.NoError(t, exec.Command("git", "init").Run(), "Failed to init git repo")
			exec.Command("git", "config", "user.name", "Test User").Run()
			exec.Command("git", "config", "user.email", "test@example.com").Run()

			// Set remote URL
			require.NoError(t, exec.Command("git", "remote", "add", "origin", tt.remoteURL).Run(), "Failed to add remote")
			defer func() { _ = exec.Command("git", "remote", "remove", "origin").Run() }()

			got := getGHESAllowedDomains()
			assert.Equal(t, tt.wantDomains, got, "getGHESAllowedDomains() returned unexpected result")
		})
	}
}
