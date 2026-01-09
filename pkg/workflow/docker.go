package workflow

import (
	"fmt"
	"sort"
	"strings"

	"github.com/githubnext/gh-aw/pkg/constants"
	"github.com/githubnext/gh-aw/pkg/logger"
)

var dockerLog = logger.New("workflow:docker")

// collectDockerImages collects all Docker images used in MCP configurations
func collectDockerImages(tools map[string]any, workflowData *WorkflowData) []string {
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

	// Check for safe-outputs MCP server (uses node:lts-alpine container)
	if workflowData != nil && workflowData.SafeOutputs != nil && HasSafeOutputsEnabled(workflowData.SafeOutputs) {
		image := constants.DefaultNodeAlpineLTSImage
		if !imageSet[image] {
			images = append(images, image)
			imageSet[image] = true
			dockerLog.Printf("Added safe-outputs MCP server container: %s", image)
		}
	}

	// Collect sandbox.mcp container (MCP gateway)
	if workflowData != nil && workflowData.SandboxConfig != nil && workflowData.SandboxConfig.MCP != nil {
		mcpGateway := workflowData.SandboxConfig.MCP
		if mcpGateway.Container != "" {
			image := mcpGateway.Container
			if mcpGateway.Version != "" {
				image += ":" + mcpGateway.Version
			}
			if !imageSet[image] {
				images = append(images, image)
				imageSet[image] = true
				dockerLog.Printf("Added sandbox.mcp container: %s", image)
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
	dockerLog.Printf("Collected %d Docker images from tools", len(images))
	return images
}

// generateDownloadDockerImagesStep generates the step to download Docker images
func generateDownloadDockerImagesStep(yaml *strings.Builder, dockerImages []string) {
	if len(dockerImages) == 0 {
		return
	}

	yaml.WriteString("      - name: Downloading container images\n")
	yaml.WriteString("        run: bash /opt/gh-aw/actions/download_docker_images.sh")
	for _, image := range dockerImages {
		fmt.Fprintf(yaml, " %s", image)
	}
	yaml.WriteString("\n")
}
