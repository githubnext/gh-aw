package workflow

import (
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"

	"github.com/githubnext/gh-aw/pkg/console"
)

// collectDockerImages collects all Docker images used in MCP configurations
func collectDockerImages(tools map[string]any) []string {
	var images []string
	imageSet := make(map[string]bool) // Use a set to avoid duplicates

	// Check for GitHub tool (uses Docker image)
	if githubTool, hasGitHub := tools["github"]; hasGitHub {
		githubType := getGitHubType(githubTool)
		// Only add if using local (Docker) mode
		if githubType == "local" {
			githubDockerImageVersion := getGitHubDockerImageVersion(githubTool)
			image := "ghcr.io/github/github-mcp-server:" + githubDockerImageVersion
			if !imageSet[image] {
				images = append(images, image)
				imageSet[image] = true
			}
		}
	}

	// Collect images from custom MCP tools with container configurations
	for toolName, toolValue := range tools {
		if mcpConfig, ok := toolValue.(map[string]any); ok {
			if hasMcp, _ := hasMCPConfig(mcpConfig); hasMcp {
				// Check if this tool uses a container
				if mcpConf, err := getMCPConfig(mcpConfig, toolName); err == nil {
					// Check for direct container field
					if mcpConf.Container != "" {
						image := mcpConf.Container
						if !imageSet[image] {
							images = append(images, image)
							imageSet[image] = true
						}
					} else if mcpConf.Command == "docker" && len(mcpConf.Args) > 0 {
						// Extract container image from docker args
						// Args format: ["run", "--rm", "-i", ... , "container-image"]
						// The container image is the last arg
						image := mcpConf.Args[len(mcpConf.Args)-1]
						// Skip if it's a docker flag (starts with -)
						if !strings.HasPrefix(image, "-") && !imageSet[image] {
							images = append(images, image)
							imageSet[image] = true
						}
					}
				}
			}
		}
	}

	// Sort for stable output
	sort.Strings(images)
	return images
}

// generateDownloadDockerImagesStep generates the step to download Docker images
func generateDownloadDockerImagesStep(yaml *strings.Builder, dockerImages []string) {
	if len(dockerImages) == 0 {
		return
	}

	yaml.WriteString("      - name: Downloading container images\n")
	yaml.WriteString("        run: |\n")
	yaml.WriteString("          set -e\n")
	for _, image := range dockerImages {
		fmt.Fprintf(yaml, "          docker pull %s\n", image)
	}
}

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
