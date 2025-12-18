package workflow

import (
	"fmt"
	"sort"
	"strings"

	"github.com/githubnext/gh-aw/pkg/logger"
)

var dockerLog = logger.New("workflow:docker")

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

	// Check for Playwright tool (uses Docker image - no version tag, only one image)
	if _, hasPlaywright := tools["playwright"]; hasPlaywright {
		image := "mcr.microsoft.com/playwright/mcp"
		if !imageSet[image] {
			images = append(images, image)
			imageSet[image] = true
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
	dockerLog.Printf("Collected %d Docker images from tools", len(images))
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
	yaml.WriteString("          # Helper function to pull Docker images with retry logic\n")
	yaml.WriteString("          docker_pull_with_retry() {\n")
	yaml.WriteString("            local image=\"$1\"\n")
	yaml.WriteString("            local max_attempts=3\n")
	yaml.WriteString("            local attempt=1\n")
	yaml.WriteString("            local wait_time=5\n")
	yaml.WriteString("            \n")
	yaml.WriteString("            while [ $attempt -le $max_attempts ]; do\n")
	yaml.WriteString("              echo \"Attempt $attempt of $max_attempts: Pulling $image...\"\n")
	yaml.WriteString("              if docker pull \"$image\"; then\n")
	yaml.WriteString("                echo \"Successfully pulled $image\"\n")
	yaml.WriteString("                return 0\n")
	yaml.WriteString("              fi\n")
	yaml.WriteString("              \n")
	yaml.WriteString("              if [ $attempt -lt $max_attempts ]; then\n")
	yaml.WriteString("                echo \"Failed to pull $image. Retrying in ${wait_time}s...\"\n")
	yaml.WriteString("                sleep $wait_time\n")
	yaml.WriteString("                wait_time=$((wait_time * 2))  # Exponential backoff\n")
	yaml.WriteString("              else\n")
	yaml.WriteString("                echo \"Failed to pull $image after $max_attempts attempts\"\n")
	yaml.WriteString("                return 1\n")
	yaml.WriteString("              fi\n")
	yaml.WriteString("              attempt=$((attempt + 1))\n")
	yaml.WriteString("            done\n")
	yaml.WriteString("          }\n")
	yaml.WriteString("          \n")
	for _, image := range dockerImages {
		fmt.Fprintf(yaml, "          docker_pull_with_retry %s\n", image)
	}
}
