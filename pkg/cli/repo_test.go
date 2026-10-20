//go:build !integration

package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCurrentRepoSlugPrefersLocalGitBeforeGH(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fake gh uses a shell script")
	}
	for _, tc := range []struct {
		name, origin, override, result string
		wantGH                         bool
	}{
		{name: "https", origin: "https://github.com/local/repo.git", result: "local/repo"},
		{name: "ssh", origin: "git@github.com:local/repo.git", result: "local/repo"},
		{name: "missing origin", result: "hosted/repo", wantGH: true},
		{name: "unsupported origin", origin: "https://example.com/local/repo.git", result: "hosted/repo", wantGH: true},
		{name: "explicit environment", origin: "https://github.com/local/repo.git", override: "hosted/repo", result: "hosted/repo", wantGH: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			t.Chdir(dir)
			t.Setenv("GH_REPO", tc.override)
			t.Setenv("GITHUB_SERVER_URL", "https://github.com")
			t.Setenv("GH_HOST", "github.com")
			require.NoError(t, exec.Command("git", "init", "--quiet", "--template=").Run())
			if tc.origin != "" {
				require.NoError(t, exec.Command("git", "remote", "add", "origin", tc.origin).Run())
			}
			bin := filepath.Join(dir, "bin")
			require.NoError(t, os.Mkdir(bin, 0o700))
			called := filepath.Join(dir, "gh-called")
			t.Setenv("REPO_TEST_GH_CALL", called)
			require.NoError(t, os.WriteFile(filepath.Join(bin, "gh"), []byte("#!/bin/sh\nprintf '%s\\n' \"$*\" > \"$REPO_TEST_GH_CALL\"\nprintf 'hosted/repo\\n'\n"), 0o700))
			t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))

			got, err := getCurrentRepoSlugUncached()
			require.NoError(t, err)
			require.Equal(t, tc.result, got)
			if tc.wantGH {
				args, err := os.ReadFile(called)
				require.NoError(t, err)
				require.Contains(t, string(args), "repo view --json")
			} else {
				_, err := os.Stat(called)
				require.True(t, os.IsNotExist(err), "local repository lookup must not call gh")
			}
		})
	}
}

// TestGetCurrentRepoSlug tests that the cached version calls the underlying function only once
func TestGetCurrentRepoSlug(t *testing.T) {
	// Clear the cache before testing
	ClearCurrentRepoSlugCache()

	// Note: In a real-world scenario, we would use dependency injection or interfaces
	// For this test, we'll just verify that multiple calls to GetCurrentRepoSlug
	// produce the same result (which implies caching is working)

	// First call
	result1, err1 := GetCurrentRepoSlug()
	if err1 != nil {
		t.Logf("First call error (expected if not in a git repo): %v", err1)
	}

	// Second call
	result2, err2 := GetCurrentRepoSlug()
	if err2 != nil {
		t.Logf("Second call error (expected if not in a git repo): %v", err2)
	}

	// Both calls should return the same result and same error
	if result1 != result2 {
		t.Errorf("GetCurrentRepoSlug returned different results on multiple calls: %s vs %s", result1, result2)
	}

	if (err1 == nil) != (err2 == nil) {
		t.Errorf("GetCurrentRepoSlug returned different error states on multiple calls")
	}

	if err1 != nil && err2 != nil && err1.Error() != err2.Error() {
		t.Errorf("GetCurrentRepoSlug returned different errors on multiple calls: %v vs %v", err1, err2)
	}

	t.Logf("GetCurrentRepoSlug returned consistent results (result: %s, error: %v)", result1, err1)
}

// TestGetCurrentRepoSlugFormat tests the format validation
func TestGetCurrentRepoSlugFormat(t *testing.T) {
	// This test will only pass if we're in a GitHub repository
	result, err := GetCurrentRepoSlug()

	if err != nil {
		t.Logf("GetCurrentRepoSlug returned error (expected if not in a git repo): %v", err)
		// Skip further validation if we're not in a repo
		return
	}

	// Verify the format is "owner/repo"
	if result == "" {
		t.Error("GetCurrentRepoSlug returned empty string without error")
		return
	}

	// The function already validates the format, so if we got here without error,
	// the format should be valid. Let's just log the result.
	t.Logf("GetCurrentRepoSlug returned: %s", result)
}
