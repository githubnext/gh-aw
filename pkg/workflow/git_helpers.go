package workflow

import (
	"os"
	"os/exec"
	"strings"
	"sync"

	"github.com/githubnext/gh-aw/pkg/logger"
)

var gitHelpersLog = logger.New("workflow:git_helpers")

// gitSHACache caches the current commit SHA to avoid repeated git calls
var (
	gitSHACache     string
	gitSHACacheMu   sync.RWMutex
	gitSHACacheOnce sync.Once
)

// GetCurrentCommitSHA returns the current commit SHA with caching.
// It tries the following sources in order:
// 1. GITHUB_SHA environment variable (when running in GitHub Actions)
// 2. git rev-parse HEAD (when running locally, cached)
// Returns empty string if SHA cannot be determined.
//
// Security note: The git command is only executed in controlled environments
// (local development or GitHub Actions). This is not exposed to user input.
func GetCurrentCommitSHA() string {
	// Try GITHUB_SHA environment variable first (set in GitHub Actions)
	if sha := os.Getenv("GITHUB_SHA"); sha != "" {
		gitHelpersLog.Printf("Using GITHUB_SHA: %s", sha)
		return sha
	}

	// Use cached value if available
	gitSHACacheMu.RLock()
	if gitSHACache != "" {
		sha := gitSHACache
		gitSHACacheMu.RUnlock()
		gitHelpersLog.Printf("Using cached git SHA: %s", sha)
		return sha
	}
	gitSHACacheMu.RUnlock()

	// Compute and cache the SHA
	gitSHACacheOnce.Do(func() {
		gitSHACacheMu.Lock()
		defer gitSHACacheMu.Unlock()

		// Fall back to git rev-parse HEAD
		// This command is safe because it runs in a controlled environment
		// and doesn't accept any user input
		cmd := exec.Command("git", "rev-parse", "HEAD")
		output, err := cmd.Output()
		if err != nil {
			gitHelpersLog.Printf("Failed to get commit SHA via git: %v", err)
			gitSHACache = ""
			return
		}

		gitSHACache = strings.TrimSpace(string(output))
		gitHelpersLog.Printf("Computed and cached git SHA: %s", gitSHACache)
	})

	gitSHACacheMu.RLock()
	defer gitSHACacheMu.RUnlock()
	return gitSHACache
}

// GetCurrentGitTag returns the current git tag if available.
// Returns empty string if not on a tag.
func GetCurrentGitTag() string {
	// Try GITHUB_REF for tags (refs/tags/v1.0.0)
	if ref := os.Getenv("GITHUB_REF"); strings.HasPrefix(ref, "refs/tags/") {
		tag := strings.TrimPrefix(ref, "refs/tags/")
		gitHelpersLog.Printf("Using tag from GITHUB_REF: %s", tag)
		return tag
	}

	// Try git describe --exact-match for local tag
	cmd := exec.Command("git", "describe", "--exact-match", "--tags", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		// Not on a tag, which is fine
		gitHelpersLog.Print("Not on a git tag")
		return ""
	}

	tag := strings.TrimSpace(string(output))
	gitHelpersLog.Printf("Using tag from git describe: %s", tag)
	return tag
}

// ResetGitSHACache resets the cached git SHA.
// This is primarily useful for testing.
func ResetGitSHACache() {
	gitSHACacheMu.Lock()
	defer gitSHACacheMu.Unlock()
	gitSHACache = ""
	gitSHACacheOnce = sync.Once{}
}
