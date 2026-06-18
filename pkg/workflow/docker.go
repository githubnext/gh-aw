package workflow

import (
	"fmt"
	"sort"
	"strings"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
)

var dockerLog = logger.New("workflow:docker")

// collectDockerImages collects all Docker images used in MCP configurations.
// When workflowData.ActionCache contains container pins, the returned slice uses
// the pinned references (image:tag@sha256:…) instead of the bare tags, ensuring
// deterministic and supply-chain-safe image pulls.
func collectDockerImages(tools map[string]any, workflowData *WorkflowData, actionMode ActionMode) []string {
	var images []string
	imageSet := make(map[string]struct{}) // Use a set to avoid duplicates

	// Check for GitHub tool (uses Docker image)
	if rawGithubTool, hasGitHub := tools["github"]; hasGitHub {
		// Only proceed when the value is an actual config map; a boolean false
		// means the tool is explicitly disabled.
		if githubTool, ok := rawGithubTool.(map[string]any); ok {
			githubType := getGitHubType(githubTool)
			// Only add if using local (Docker) mode
			if githubType == GitHubMCPModeLocal {
				githubDockerImageVersion := getGitHubDockerImageVersion(githubTool)
				image := "ghcr.io/github/github-mcp-server:" + githubDockerImageVersion
				if !hasStringKey(imageSet, image) {
					images = append(images, image)
					imageSet[image] = struct {
					}{}
				}
			}
		}
	}

	// Check for Playwright tool (uses Docker image - no version tag, only one image)
	// Only in MCP mode; CLI mode installs @playwright/cli via npm instead.
	if _, hasPlaywright := tools["playwright"]; hasPlaywright {
		if !isPlaywrightCLIMode(tools) {
			image := "mcr.microsoft.com/playwright/mcp"
			if !hasStringKey(imageSet, image) {
				images = append(images, image)
				imageSet[image] = struct {
				}{}
			}
		}
	}

	// Check for safe-outputs MCP server.
	// Safe outputs run in the published gh-aw node container and must be part of
	// the default predownload set and lock-file manifest whenever enabled.
	if workflowData != nil && HasSafeOutputsEnabled(workflowData.SafeOutputs) {
		image := constants.DefaultGhAwNodeImage
		if !hasStringKey(imageSet, image) {
			images = append(images, image)
			imageSet[image] = struct {
			}{}
			dockerLog.Printf("Added safe-outputs MCP server container: %s", image)
		}
	}

	// Check for agentic-workflows tool
	// In dev mode, the image is built locally in the workflow, so don't add to pull list
	// In release/script mode, use alpine:latest which needs to be pulled
	if _, hasAgenticWorkflows := tools["agentic-workflows"]; hasAgenticWorkflows {
		if !actionMode.IsDev() {
			// Release/script mode: Use alpine:latest (needs to be pulled)
			image := constants.DefaultAlpineImage
			if !hasStringKey(imageSet, image) {
				images = append(images, image)
				imageSet[image] = struct {
				}{}
				dockerLog.Printf("Added agentic-workflows MCP server container: %s", image)
			}
		}
		// Dev mode: localhost/gh-aw:dev is built locally, not pulled
	}

	// Collect AWF (firewall) container images when firewall is enabled
	// AWF uses three containers: squid (proxy), agent, and api-proxy (for engines with LLM gateway support)
	if isFirewallEnabled(workflowData) {
		// Get the firewall version for image tags
		firewallConfig := getFirewallConfig(workflowData)
		awfImageTag := getAWFImageTag(firewallConfig)

		// Add squid (proxy) container
		squidImage := constants.DefaultFirewallRegistry + "/squid:" + awfImageTag
		if !hasStringKey(imageSet, squidImage) {
			images = append(images, squidImage)
			imageSet[squidImage] = struct {
			}{}
			dockerLog.Printf("Added AWF squid (proxy) container: %s", squidImage)
		}

		// Add default agent container
		agentImage := constants.DefaultFirewallRegistry + "/agent:" + awfImageTag
		if !hasStringKey(imageSet, agentImage) {
			images = append(images, agentImage)
			imageSet[agentImage] = struct {
			}{}
			dockerLog.Printf("Added AWF agent container: %s", agentImage)
		}

		// Add api-proxy sidecar container (required for all engines — LLM gateway is mandatory)
		// The api-proxy holds LLM API keys securely and proxies requests through Squid
		// Each engine uses its own dedicated port for communication
		if workflowData != nil && workflowData.AI != "" {
			apiProxyImage := constants.DefaultFirewallRegistry + "/api-proxy:" + awfImageTag
			if !hasStringKey(imageSet, apiProxyImage) {
				images = append(images, apiProxyImage)
				imageSet[apiProxyImage] = struct {
				}{}
				dockerLog.Printf("Added AWF api-proxy sidecar container: %s", apiProxyImage)
			}
		}

		// Add cli-proxy sidecar container when the cli-proxy is needed.
		// Without this, --skip-pull causes AWF to fail because the cli-proxy image was never pulled.
		if isCliProxyNeeded(workflowData) {
			cliProxyImage := constants.DefaultFirewallRegistry + "/cli-proxy:" + awfImageTag
			if !hasStringKey(imageSet, cliProxyImage) {
				images = append(images, cliProxyImage)
				imageSet[cliProxyImage] = struct {
				}{}
				dockerLog.Printf("Added AWF cli-proxy sidecar container: %s", cliProxyImage)
			}
		}
	}

	// Collect sandbox.mcp container (MCP gateway)
	// Skip if sandbox is disabled (sandbox: false)
	if workflowData != nil && workflowData.SandboxConfig != nil {
		// Check if sandbox is disabled
		sandboxDisabled := workflowData.SandboxConfig.Agent != nil && workflowData.SandboxConfig.Agent.Disabled

		if !sandboxDisabled && workflowData.SandboxConfig.MCP != nil {
			mcpGateway := workflowData.SandboxConfig.MCP
			if mcpGateway.Container != "" {
				image := mcpGateway.Container
				if mcpGateway.Version != "" {
					image += ":" + mcpGateway.Version
				} else {
					// Use default version if not specified (consistent with mcp_servers.go)
					image += ":" + string(constants.DefaultMCPGatewayVersion)
				}
				if !hasStringKey(imageSet, image) {
					images = append(images, image)
					imageSet[image] = struct {
					}{}
					dockerLog.Printf("Added sandbox.mcp container: %s", image)
				}
			}
		} else if sandboxDisabled {
			dockerLog.Print("Sandbox disabled, skipping MCP gateway container image")
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
						if !hasStringKey(imageSet, image) {
							images = append(images, image)
							imageSet[image] = struct {
							}{}
						}
					} else if mcpConf.Command == "docker" && len(mcpConf.Args) > 0 {
						// Extract container image from docker args
						// Args format: ["run", "--rm", "-i", ... , "container-image"]
						// The container image is the last arg
						image := mcpConf.Args[len(mcpConf.Args)-1]
						// Skip if it's a docker flag (starts with -)
						if !strings.HasPrefix(image, "-") && !hasStringKey(imageSet, image) {
							images = append(images, image)
							imageSet[image] = struct {
							}{}
						}
					}
				}
			}
		}
	}

	// Sort for stable output
	sort.Strings(images)
	dockerLog.Printf("Collected %d Docker images from tools", len(images))

	// Apply digest pins from the action cache when available.
	// Each pinned ref replaces the bare tag with "tag@sha256:…" so that the pull
	// is bound to a specific immutable manifest and not just to a mutable tag.
	pinnedImages, imagePins := applyContainerPins(images, workflowData)

	// Store pinned image refs and full pin info in WorkflowData so they can be
	// included in the compiled lock file header and gh-aw-manifest for auditability.
	if workflowData != nil {
		workflowData.DockerImages = mergeDockerImages(workflowData.DockerImages, pinnedImages)
		workflowData.DockerImagePins = mergeDockerImagePins(workflowData.DockerImagePins, imagePins)
	}

	return pinnedImages
}

// applyContainerPins substitutes cached digest-pinned references for any image
// tags that have an entry in workflowData.ActionCache.ContainerPins.
// Images without a cached pin are returned unchanged.
// Returns both the resolved image strings (for script args) and full GHAWManifestContainer
// entries (for the manifest).
func applyContainerPins(images []string, workflowData *WorkflowData) ([]string, []GHAWManifestContainer) {
	result := make([]string, len(images))
	pins := make([]GHAWManifestContainer, len(images))

	var cache *ActionCache
	if workflowData != nil {
		cache = workflowData.ActionCache
	}

	for i, img := range images {
		if pin, ok := lookupContainerPin(img, cache); ok && pin.PinnedImage != "" {
			result[i] = pin.PinnedImage
			pins[i] = GHAWManifestContainer(pin)
			dockerLog.Printf("Pinned container image: %s -> %s", img, pin.PinnedImage)
			continue
		}
		result[i] = img
		pins[i] = GHAWManifestContainer{Image: img}
	}
	return result, pins
}

// mergeDockerImages appends any images from newImages that are not already present
// in existing, preserving order for stability.
func mergeDockerImages(existing, newImages []string) []string {
	seen := make(map[string]struct {
	}, len(existing))
	for _, img := range existing {
		seen[img] = struct {
		}{}
	}
	result := existing
	for _, img := range newImages {
		if !hasStringKey(seen, img) {
			result = append(result, img)
			seen[img] = struct {
			}{}
		}
	}
	return result
}

// mergeDockerImagePins appends any pin entries from newPins that are not already present
// in existing (keyed by Image), preserving order for stability.
func mergeDockerImagePins(existing, newPins []GHAWManifestContainer) []GHAWManifestContainer {
	seen := make(map[string]struct {
	}, len(existing))
	for _, p := range existing {
		seen[p.Image] = struct {
		}{}
	}
	result := existing
	for _, p := range newPins {
		if p.Image != "" && !hasStringKey(seen, p.Image) {
			result = append(result, p)
			seen[p.Image] = struct {
			}{}
		}
	}
	return result
}

// generateDownloadDockerImagesStep generates the step to download Docker images
func generateDownloadDockerImagesStep(yaml *strings.Builder, dockerImages []string) {
	if len(dockerImages) == 0 {
		return
	}

	yaml.WriteString("      - name: Download container images\n")
	yaml.WriteString("        run: bash \"${RUNNER_TEMP}/gh-aw/actions/download_docker_images.sh\"")
	for _, image := range dockerImages {
		fmt.Fprintf(yaml, " %s", image)
	}
	yaml.WriteString("\n")
}
