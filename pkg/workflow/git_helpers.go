package workflow

import (
	"os"
	"os/exec"
	"strings"

	"github.com/githubnext/gh-aw/pkg/logger"
)

var gitHelpersLog = logger.New("workflow:git_helpers")

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
