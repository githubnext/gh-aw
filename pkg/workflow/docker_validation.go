// Package workflow provides Docker image validation for agentic workflows.
//
// # Docker Image Validation
//
// This file validates Docker container images used in MCP configurations.
// Validation ensures that Docker images specified in workflows exist and are accessible,
// preventing runtime failures due to typos or non-existent images.
//
// # Validation Functions
//
//   - validateDockerImage() - Validates a single Docker image exists and is accessible
//
// # Validation Pattern: Warning vs Error
//
// Docker image validation uses a flexible approach:
//   - If Docker is not available, a warning is emitted but validation is skipped
//   - If an image cannot be pulled due to authentication (private repo), validation passes
//   - If an image truly doesn't exist, validation fails with an error
//   - Verbose mode provides detailed validation feedback
//
// # When to Add Validation Here
//
// Add validation to this file when:
//   - It validates Docker images
//   - It checks container image accessibility
//   - It validates Docker-specific configurations
//
// For Docker image collection functions, see docker.go.
// For general validation, see validation.go.
// For detailed documentation, see specs/validation-architecture.md
package workflow

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/githubnext/gh-aw/pkg/console"
	"github.com/githubnext/gh-aw/pkg/logger"
)

var dockerValidationLog = logger.New("workflow:docker_validation")

// validateDockerImage checks if a Docker image exists and is accessible
// Returns nil if docker is not available (with a warning printed)
func validateDockerImage(image string, verbose bool) error {
	dockerValidationLog.Printf("Validating Docker image: %s", image)

	// Check if docker is available
	_, err := exec.LookPath("docker")
	if err != nil {
		dockerValidationLog.Print("Docker not available, skipping image validation")
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
		dockerValidationLog.Printf("Docker image found locally: %s", image)
		_ = output // Suppress unused variable warning
		return nil
	}

	dockerValidationLog.Printf("Docker image not found locally, attempting to pull: %s", image)

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
