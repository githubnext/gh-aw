package workflow

import (
	"fmt"
	"sort"
	"strings"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/parser"
	"github.com/github/gh-aw/pkg/setutil"
)

var dockerLog = logger.New("workflow:docker")

type dockerImageCollector struct {
	images []string
	seen   map[string]struct{}
}

// collectDockerImages collects all Docker images used in MCP configurations.
// When workflowData.ActionCache contains container pins, the returned slice uses
// the pinned references (image:tag@sha256:…) instead of the bare tags.
func collectDockerImages(tools map[string]any, workflowData *WorkflowData, actionMode ActionMode) []string {
	collector := dockerImageCollector{seen: make(map[string]struct{})}
	collector.collectBuiltInToolImages(tools, workflowData, actionMode)
	collector.collectFirewallImages(workflowData)
	collector.collectSandboxMCPImage(workflowData)
	collector.collectCustomMCPImages(tools)

	sort.Strings(collector.images)
	dockerLog.Printf("Collected %d Docker images from tools", len(collector.images))
	pinnedImages, imagePins := applyContainerPins(collector.images, workflowData)
	if workflowData != nil {
		workflowData.DockerImages = mergeDockerImages(workflowData.DockerImages, pinnedImages)
		workflowData.DockerImagePins = mergeDockerImagePins(workflowData.DockerImagePins, imagePins)
	}
	return pinnedImages
}

func (c *dockerImageCollector) add(image string) bool {
	if setutil.Contains(c.seen, image) {
		return false
	}
	c.images = append(c.images, image)
	c.seen[image] = struct{}{}
	return true
}

func (c *dockerImageCollector) collectBuiltInToolImages(tools map[string]any, workflowData *WorkflowData, actionMode ActionMode) {
	if rawGithubTool, hasGitHub := tools["github"]; hasGitHub {
		if githubTool, ok := rawGithubTool.(map[string]any); ok && getGitHubType(githubTool) == GitHubMCPModeLocal {
			c.add("ghcr.io/github/github-mcp-server:" + getGitHubDockerImageVersion(githubTool))
		}
	}
	if _, hasPlaywright := tools["playwright"]; hasPlaywright && !isPlaywrightCLIMode(tools) {
		c.add("mcr.microsoft.com/playwright/mcp")
	}
	if workflowData != nil && HasSafeOutputsEnabled(workflowData.SafeOutputs) {
		if c.add(constants.DefaultGhAwNodeImage) {
			dockerLog.Printf("Added safe-outputs MCP server container: %s", constants.DefaultGhAwNodeImage)
		}
	}
	if _, hasAgenticWorkflows := tools["agentic-workflows"]; hasAgenticWorkflows && !actionMode.IsDev() {
		if c.add(constants.DefaultAlpineImage) {
			dockerLog.Printf("Added agentic-workflows MCP server container: %s", constants.DefaultAlpineImage)
		}
	}
}

func (c *dockerImageCollector) collectFirewallImages(workflowData *WorkflowData) {
	if !isFirewallEnabled(workflowData) {
		return
	}
	firewallConfig := getFirewallConfig(workflowData)
	awfImageTag := getAWFImageTag(firewallConfig)
	c.addLoggedAWFImage("squid", awfImageTag, "Added AWF squid (proxy) container: %s")
	c.addLoggedAWFImage("agent", awfImageTag, "Added AWF agent container: %s")
	if workflowData != nil && workflowData.AI != "" {
		c.addLoggedAWFImage("api-proxy", awfImageTag, "Added AWF api-proxy sidecar container: %s")
	}
	if isCliProxyNeeded(workflowData) {
		c.addLoggedAWFImage("cli-proxy", awfImageTag, "Added AWF cli-proxy sidecar container: %s")
	}
	if isArcDindTopology(workflowData) {
		c.addLoggedAWFImage("build-tools", awfImageTag, "Added AWF build-tools sysroot container for arc-dind: %s")
	}
}

func (c *dockerImageCollector) addLoggedAWFImage(name, tag, logFormat string) {
	image := constants.DefaultFirewallRegistry + "/" + name + ":" + tag
	if c.add(image) {
		dockerLog.Printf(logFormat, image)
	}
}

func (c *dockerImageCollector) collectSandboxMCPImage(workflowData *WorkflowData) {
	if workflowData == nil || workflowData.SandboxConfig == nil {
		return
	}
	sandboxDisabled := workflowData.SandboxConfig.Agent != nil && workflowData.SandboxConfig.Agent.Disabled
	if sandboxDisabled {
		dockerLog.Print("Sandbox disabled, skipping MCP gateway container image")
		return
	}
	if workflowData.SandboxConfig.MCP == nil || workflowData.SandboxConfig.MCP.Container == "" {
		return
	}
	mcpGateway := workflowData.SandboxConfig.MCP
	image := mcpGateway.Container
	if mcpGateway.Version != "" {
		image += ":" + mcpGateway.Version
	} else {
		image += ":" + string(constants.DefaultMCPGatewayVersion)
	}
	if c.add(image) {
		dockerLog.Printf("Added sandbox.mcp container: %s", image)
	}
}

func (c *dockerImageCollector) collectCustomMCPImages(tools map[string]any) {
	for toolName, toolValue := range tools {
		mcpConfig, ok := toolValue.(map[string]any)
		if !ok {
			continue
		}
		if hasMcp, _ := hasMCPConfig(mcpConfig); !hasMcp {
			continue
		}
		if mcpConf, err := getMCPConfig(mcpConfig, toolName); err == nil {
			c.addCustomMCPImage(mcpConf)
		}
	}
}

func (c *dockerImageCollector) addCustomMCPImage(mcpConf *parser.RegistryMCPServerConfig) {
	if mcpConf.Container != "" {
		c.add(mcpConf.Container)
		return
	}
	if mcpConf.Command == "docker" && len(mcpConf.Args) > 0 {
		image := mcpConf.Args[len(mcpConf.Args)-1]
		if !strings.HasPrefix(image, "-") {
			c.add(image)
		}
	}
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
		if !setutil.Contains(seen, img) {
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
		if p.Image != "" && !setutil.Contains(seen, p.Image) {
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
