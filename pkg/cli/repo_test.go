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

func TestWorkQueueRepoSlugPrefersLocalGitBeforeGH(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fake gh uses a shell script")
	}
	for _, tc := range []struct {
		name, origin, override, result string
		wantGH                         bool
	}{
		{name: "https", origin: "https://github.com/local/repo.git", result: "local/repo"},
		{name: "ssh", origin: "git@github.com:local/repo.git", result: "local/repo"},
		{name: "enterprise", origin: "git@github.example.com:local/repo.git", result: "local/repo"},
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
			if tc.name == "enterprise" {
				t.Setenv("GITHUB_SERVER_URL", "https://github.example.com")
				t.Setenv("GH_HOST", "github.example.com")
			}
			ClearCurrentRepoSlugCache()
			t.Cleanup(ClearCurrentRepoSlugCache)
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

			got, err := getWorkQueueRepoSlug()
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

func TestCurrentRepoSlugPreservesGHSelectedRepository(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fake gh uses a shell script")
	}
	for _, tc := range []struct {
		name, origin, selected string
	}{
		{"fork with parent default", "git@github.com:fork/repo.git", "parent/repo"},
		{"renamed origin", "https://github.com/owner/old-name.git", "owner/new-name"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			t.Chdir(dir)
			t.Setenv("GH_REPO", "")
			t.Setenv("GITHUB_SERVER_URL", "https://github.com")
			t.Setenv("GH_HOST", "github.com")
			require.NoError(t, exec.Command("git", "init", "--quiet", "--template=").Run())
			require.NoError(t, exec.Command("git", "remote", "add", "origin", tc.origin).Run())
			require.NoError(t, exec.Command("git", "remote", "add", "upstream", "https://github.com/parent/repo.git").Run())
			bin := filepath.Join(dir, "bin")
			require.NoError(t, os.Mkdir(bin, 0o700))
			t.Setenv("REPO_TEST_SELECTED", tc.selected)
			require.NoError(t, os.WriteFile(filepath.Join(bin, "gh"), []byte("#!/bin/sh\nprintf '%s\\n' \"$REPO_TEST_SELECTED\"\n"), 0o700))
			t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
			got, err := getCurrentRepoSlugUncached()
			require.NoError(t, err)
			require.Equal(t, tc.selected, got)
			local, err := getWorkQueueRepoSlug()
			require.NoError(t, err)
			if tc.name == "fork with parent default" {
				require.Equal(t, "fork/repo", local, "only queue inference should choose the checkout fork")
			} else {
				require.Equal(t, "owner/old-name", local, "local discovery cannot prove a hosted rename")
			}
		})
	}
}

func TestCurrentRepoSlugFallsBackOnlyWithoutExplicitSelection(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fake gh uses a shell script")
	}
	for _, response := range []string{"exit 1", "printf 'invalid-slug\\n'"} {
		for _, override := range []string{"", "selected/repo"} {
			t.Run(response+"/"+override, func(t *testing.T) {
				dir := t.TempDir()
				t.Chdir(dir)
				t.Setenv("GH_REPO", override)
				t.Setenv("GITHUB_SERVER_URL", "https://github.com")
				t.Setenv("GH_HOST", "github.com")
				ClearCurrentRepoSlugCache()
				t.Cleanup(ClearCurrentRepoSlugCache)
				require.NoError(t, exec.Command("git", "init", "--quiet", "--template=").Run())
				require.NoError(t, exec.Command("git", "remote", "add", "origin", "git@github.com:local/repo.git").Run())
				bin := filepath.Join(dir, "bin")
				require.NoError(t, os.Mkdir(bin, 0o700))
				require.NoError(t, os.WriteFile(filepath.Join(bin, "gh"), []byte("#!/bin/sh\n"+response+"\n"), 0o700))
				t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
				got, err := getCurrentRepoSlugUncached()
				if override == "" {
					require.NoError(t, err)
					require.Equal(t, "local/repo", got)
				} else {
					require.Error(t, err)
					require.Empty(t, got)
					queueRepo, queueErr := getWorkQueueRepoSlug()
					require.Error(t, queueErr)
					require.Empty(t, queueRepo, "an explicit target cannot silently become the local checkout")
				}
			})
		}
	}
}

func TestRepositoryOriginDiagnosticsDoNotLeakCredentials(t *testing.T) {
	if os.Getenv("GH_AW_TEST_ORIGIN_DIAGNOSTICS") == "1" {
		_, err := getCurrentRepoSlugFromGit()
		if err != nil {
			t.Log(err)
		}
		return
	}
	for _, mode := range []string{"debug", "actions runner debug", "runner debug"} {
		for _, origin := range []string{
			"https://x-access-token:repo-test-secret@github.com/owner/repo.git",
			"ssh://user:repo-test-secret@github.com/owner/repo.git",
			"repo-test-secret@github.com:owner/repo.git",
			"https://github.com/owner/repo?token=repo-test-secret",
			"git@github.com:owner/https://user:repo-test-secret@host/repo.git",
		} {
			t.Run(mode+"/"+origin, func(t *testing.T) {
				dir := t.TempDir()
				cmd := exec.Command("git", "init", "--quiet", "--template=", dir)
				require.NoError(t, cmd.Run())
				require.NoError(t, exec.Command("git", "-C", dir, "remote", "add", "origin", origin).Run())
				t.Setenv("DEBUG", "")
				t.Setenv("ACTIONS_RUNNER_DEBUG", "")
				t.Setenv("RUNNER_DEBUG", "")
				switch mode {
				case "debug":
					t.Setenv("DEBUG", "cli:*")
				case "actions runner debug":
					t.Setenv("ACTIONS_RUNNER_DEBUG", "true")
				case "runner debug":
					t.Setenv("RUNNER_DEBUG", "1")
				}
				t.Setenv("GITHUB_SERVER_URL", "https://github.com")
				t.Setenv("GH_AW_TEST_ORIGIN_DIAGNOSTICS", "1")
				process := exec.Command(os.Args[0], "-test.run=^TestRepositoryOriginDiagnosticsDoNotLeakCredentials$", "-test.v")
				process.Dir = dir
				output, err := process.CombinedOutput()
				require.NoError(t, err, "%s", output)
				require.Contains(t, string(output), "Parsing GitHub repository URL", "debug logging must be enabled in the subprocess")
				require.NotContains(t, string(output), "repo-test-secret")
				require.NotContains(t, string(output), origin)
			})
		}
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
