package workflow

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/githubnext/gh-aw/pkg/console"
)

// validateDockerImage checks if a Docker image exists and is accessible
// Returns nil if docker is not available (with a warning printed)
func validateDockerImage(image string, verbose bool) error {
	// Check if docker is available
	_, err := exec.LookPath("docker")
	if err != nil {
		// Docker not available - print warning and skip validation
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Docker not available - skipping validation for container image '%s'", image)))
		}
		return nil
	}

	// Try to inspect the image (will succeed if image exists locally)
	cmd := exec.Command("docker", "image", "inspect", image)
	output, err := cmd.CombinedOutput()

	if err == nil {
		// Image exists locally
		_ = output // Suppress unused variable warning
		return nil
	}

	// Image doesn't exist locally, try to pull it
	pullCmd := exec.Command("docker", "pull", image)
	pullOutput, pullErr := pullCmd.CombinedOutput()

	if pullErr != nil {
		outputStr := strings.TrimSpace(string(pullOutput))

		// Check if the error is due to authentication issues for existing private repositories
		// We need to distinguish between:
		// 1. "repository does not exist" - should fail validation
		// 2. "authentication required" for existing repos - should pass (private repo)
		if (strings.Contains(outputStr, "denied") ||
			strings.Contains(outputStr, "unauthorized") ||
			strings.Contains(outputStr, "authentication required")) &&
			!strings.Contains(outputStr, "does not exist") &&
			!strings.Contains(outputStr, "not found") {
			// This is likely a private image that requires authentication
			// Don't fail validation for private/authenticated images
			return nil
		}

		// Other errors indicate the image truly doesn't exist or has issues
		return fmt.Errorf("container image '%s' not found and could not be pulled: %s", image, outputStr)
	}

	// Successfully pulled
	return nil
}
