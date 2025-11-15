package workflow

import (
	"fmt"
	"sort"
	"strings"

	"github.com/githubnext/gh-aw/pkg/logger"
)

var engineHelpersLog = logger.New("workflow:engine_helpers")

// ExtractAgentIdentifier extracts the agent identifier (filename without extension) from an agent file path.
// This is used by the Copilot CLI which expects agent identifiers, not full paths.
//
// Parameters:
//   - agentFile: The relative path to the agent file (e.g., ".github/agents/test-agent.md")
//
// Returns:
//   - string: The agent identifier (e.g., "test-agent")
//
// Example:
//
//	identifier := ExtractAgentIdentifier(".github/agents/my-agent.md")
//	// Returns: "my-agent"
func ExtractAgentIdentifier(agentFile string) string {
	// Extract the base filename from the path
	lastSlash := strings.LastIndex(agentFile, "/")
	filename := agentFile
	if lastSlash >= 0 {
		filename = agentFile[lastSlash+1:]
	}

	// Remove the .md extension using TrimSuffix (unconditionally safe)
	filename = strings.TrimSuffix(filename, ".md")

	return filename
}

// ResolveAgentFilePath returns the properly quoted agent file path with GITHUB_WORKSPACE prefix.
// This helper extracts the common pattern shared by Copilot, Codex, and Claude engines.
//
// The agent file path is relative to the repository root, so we prefix it with ${GITHUB_WORKSPACE}
// and wrap the entire expression in double quotes to handle paths with spaces while allowing
// shell variable expansion.
//
// Parameters:
//   - agentFile: The relative path to the agent file (e.g., ".github/agents/test-agent.md")
//
// Returns:
//   - string: The double-quoted path with GITHUB_WORKSPACE prefix (e.g., "${GITHUB_WORKSPACE}/.github/agents/test-agent.md")
//
// Example:
//
//	agentPath := ResolveAgentFilePath(".github/agents/my-agent.md")
//	// Returns: "${GITHUB_WORKSPACE}/.github/agents/my-agent.md"
//
// Note: The entire path is wrapped in double quotes (not just the variable) to ensure:
//  1. The shellEscapeArg function recognizes it as already-quoted and doesn't add single quotes
//  2. Shell variable expansion works (${GITHUB_WORKSPACE} gets expanded inside double quotes)
//  3. Paths with spaces are properly handled
func ResolveAgentFilePath(agentFile string) string {
	return fmt.Sprintf("\"${GITHUB_WORKSPACE}/%s\"", agentFile)
}

// BuildStandardNpmEngineInstallSteps creates standard npm installation steps for engines
// This helper extracts the common pattern shared by Copilot, Codex, and Claude engines.
//
// Parameters:
//   - packageName: The npm package name (e.g., "@github/copilot")
//   - defaultVersion: The default version constant (e.g., constants.DefaultCopilotVersion)
//   - stepName: The display name for the install step (e.g., "Install GitHub Copilot CLI")
//   - cacheKeyPrefix: The cache key prefix (e.g., "copilot")
//   - workflowData: The workflow data containing engine configuration
//
// Returns:
//   - []GitHubActionStep: The installation steps including Node.js setup
func BuildStandardNpmEngineInstallSteps(
	packageName string,
	defaultVersion string,
	stepName string,
	cacheKeyPrefix string,
	workflowData *WorkflowData,
) []GitHubActionStep {
	engineHelpersLog.Printf("Building npm engine install steps: package=%s, version=%s", packageName, defaultVersion)

	// Use version from engine config if provided, otherwise default to pinned version
	version := defaultVersion
	if workflowData.EngineConfig != nil && workflowData.EngineConfig.Version != "" {
		version = workflowData.EngineConfig.Version
		engineHelpersLog.Printf("Using engine config version: %s", version)
	}

	// Add npm package installation steps (includes Node.js setup)
	return GenerateNpmInstallSteps(
		packageName,
		version,
		stepName,
		cacheKeyPrefix,
		true, // Include Node.js setup
	)
}

// InjectCustomEngineSteps processes custom steps from engine config and converts them to GitHubActionSteps.
// This shared function extracts the common pattern used by Copilot, Codex, and Claude engines.
//
// Parameters:
//   - workflowData: The workflow data containing engine configuration
//   - convertStepFunc: A function that converts a step map to YAML string (engine-specific)
//
// Returns:
//   - []GitHubActionStep: Array of custom steps ready to be included in the execution pipeline
func InjectCustomEngineSteps(
	workflowData *WorkflowData,
	convertStepFunc func(map[string]any) (string, error),
) []GitHubActionStep {
	var steps []GitHubActionStep

	// Handle custom steps if they exist in engine config
	if workflowData.EngineConfig != nil && len(workflowData.EngineConfig.Steps) > 0 {
		engineHelpersLog.Printf("Injecting %d custom engine steps", len(workflowData.EngineConfig.Steps))
		for _, step := range workflowData.EngineConfig.Steps {
			stepYAML, err := convertStepFunc(step)
			if err != nil {
				engineHelpersLog.Printf("Failed to convert custom step: %v", err)
				// Log error but continue with other steps
				continue
			}
			steps = append(steps, GitHubActionStep{stepYAML})
		}
		engineHelpersLog.Printf("Successfully injected %d custom engine steps", len(steps))
	}

	return steps
}

// RenderCustomMCPToolConfigHandler is a function type that engines must provide to render their specific MCP config
type RenderCustomMCPToolConfigHandler func(yaml *strings.Builder, toolName string, toolConfig map[string]any, isLast bool) error

// HandleCustomMCPToolInSwitch processes custom MCP tools in the default case of a switch statement.
// This shared function extracts the common pattern used across all workflow engines.
//
// Parameters:
//   - yaml: The string builder for YAML output
//   - toolName: The name of the tool being processed
//   - tools: The tools map containing tool configurations (supports both expanded and non-expanded tools)
//   - isLast: Whether this is the last tool in the list
//   - renderFunc: Engine-specific function to render the MCP configuration
//
// Returns:
//   - bool: true if a custom MCP tool was handled, false otherwise
func HandleCustomMCPToolInSwitch(
	yaml *strings.Builder,
	toolName string,
	tools map[string]any,
	isLast bool,
	renderFunc RenderCustomMCPToolConfigHandler,
) bool {
	// Handle custom MCP tools (those with MCP-compatible type)
	if toolConfig, ok := tools[toolName].(map[string]any); ok {
		if hasMcp, _ := hasMCPConfig(toolConfig); hasMcp {
			if err := renderFunc(yaml, toolName, toolConfig, isLast); err != nil {
				fmt.Printf("Error generating custom MCP configuration for %s: %v\n", toolName, err)
			}
			return true
		}
	}
	return false
}

// FormatStepWithCommandAndEnv formats a GitHub Actions step with command and environment variables.
// This shared function extracts the common pattern used by Copilot and Codex engines.
//
// Parameters:
//   - stepLines: Existing step lines to append to (e.g., name, id, comments, timeout)
//   - command: The command to execute (may contain multiple lines)
//   - env: Map of environment variables to include in the step
//
// Returns:
//   - []string: Complete step lines including run command and env section
func FormatStepWithCommandAndEnv(stepLines []string, command string, env map[string]string) []string {
	// Add the run section
	stepLines = append(stepLines, "        run: |")

	// Split command into lines and indent them properly
	commandLines := strings.Split(command, "\n")
	for _, line := range commandLines {
		// Don't add indentation to empty lines
		if line == "" {
			stepLines = append(stepLines, "")
		} else {
			stepLines = append(stepLines, "          "+line)
		}
	}

	// Add environment variables
	if len(env) > 0 {
		stepLines = append(stepLines, "        env:")
		// Sort environment keys for consistent output
		envKeys := make([]string, 0, len(env))
		for key := range env {
			envKeys = append(envKeys, key)
		}
		sort.Strings(envKeys)

		for _, key := range envKeys {
			value := env[key]
			stepLines = append(stepLines, fmt.Sprintf("          %s: %s", key, value))
		}
	}

	return stepLines
}

// MCPToolRenderers holds engine-specific rendering functions for each MCP tool type
type MCPToolRenderers struct {
	RenderGitHub           func(yaml *strings.Builder, githubTool any, isLast bool, workflowData *WorkflowData)
	RenderPlaywright       func(yaml *strings.Builder, playwrightTool any, isLast bool)
	RenderCacheMemory      func(yaml *strings.Builder, isLast bool, workflowData *WorkflowData)
	RenderAgenticWorkflows func(yaml *strings.Builder, isLast bool)
	RenderSafeOutputs      func(yaml *strings.Builder, isLast bool)
	RenderWebFetch         func(yaml *strings.Builder, isLast bool)
	RenderCustomMCPConfig  RenderCustomMCPToolConfigHandler
}

// JSONMCPConfigOptions defines configuration for JSON-based MCP config rendering
type JSONMCPConfigOptions struct {
	// ConfigPath is the file path for the MCP config (e.g., "/tmp/gh-aw/mcp-config/mcp-servers.json")
	ConfigPath string
	// Renderers contains engine-specific rendering functions for each tool
	Renderers MCPToolRenderers
	// FilterTool is an optional function to filter out tools before processing
	// Returns true if the tool should be included, false to skip it
	FilterTool func(toolName string) bool
	// PostEOFCommands is an optional function to add commands after the EOF (e.g., debug output)
	PostEOFCommands func(yaml *strings.Builder)
}

// GitHubMCPDockerOptions defines configuration for GitHub MCP Docker rendering
type GitHubMCPDockerOptions struct {
	// ReadOnly enables read-only mode for GitHub API operations
	ReadOnly bool
	// Toolsets specifies the GitHub toolsets to enable
	Toolsets string
	// DockerImageVersion specifies the GitHub MCP server Docker image version
	DockerImageVersion string
	// CustomArgs are additional arguments to append to the Docker command
	CustomArgs []string
	// IncludeTypeField indicates whether to include the "type": "local" field (Copilot needs it, Claude doesn't)
	IncludeTypeField bool
	// AllowedTools specifies the list of allowed tools (Copilot uses this, Claude doesn't)
	AllowedTools []string
	// EffectiveToken is the GitHub token to use (Claude uses this, Copilot uses env passthrough)
	EffectiveToken string
}

// RenderGitHubMCPDockerConfig renders the GitHub MCP server configuration for Docker (local mode).
// This shared function extracts the duplicate pattern from Claude and Copilot engines.
//
// Parameters:
//   - yaml: The string builder for YAML output
//   - options: GitHub MCP Docker rendering options
func RenderGitHubMCPDockerConfig(yaml *strings.Builder, options GitHubMCPDockerOptions) {
	// Add type field if needed (Copilot requires this, Claude doesn't)
	if options.IncludeTypeField {
		yaml.WriteString("                \"type\": \"local\",\n")
	}

	yaml.WriteString("                \"command\": \"docker\",\n")
	yaml.WriteString("                \"args\": [\n")
	yaml.WriteString("                  \"run\",\n")
	yaml.WriteString("                  \"-i\",\n")
	yaml.WriteString("                  \"--rm\",\n")
	yaml.WriteString("                  \"-e\",\n")
	yaml.WriteString("                  \"GITHUB_PERSONAL_ACCESS_TOKEN\",\n")

	if options.ReadOnly {
		yaml.WriteString("                  \"-e\",\n")
		yaml.WriteString("                  \"GITHUB_READ_ONLY=1\",\n")
	}

	// Add GITHUB_TOOLSETS environment variable (always configured, defaults to "default")
	yaml.WriteString("                  \"-e\",\n")
	yaml.WriteString(fmt.Sprintf("                  \"GITHUB_TOOLSETS=%s\",\n", options.Toolsets))

	yaml.WriteString("                  \"ghcr.io/github/github-mcp-server:" + options.DockerImageVersion + "\"")

	// Append custom args if present
	writeArgsToYAML(yaml, options.CustomArgs, "                  ")

	yaml.WriteString("\n")
	yaml.WriteString("                ],\n")

	// Add tools field if provided (Copilot uses this, Claude doesn't)
	if len(options.AllowedTools) > 0 {
		yaml.WriteString("                \"tools\": [\n")
		for i, tool := range options.AllowedTools {
			comma := ","
			if i == len(options.AllowedTools)-1 {
				comma = ""
			}
			fmt.Fprintf(yaml, "                  \"%s\"%s\n", tool, comma)
		}
		yaml.WriteString("                ],\n")
	} else if options.IncludeTypeField {
		// Copilot always includes tools field, even if empty (uses wildcard)
		yaml.WriteString("                \"tools\": [\"*\"],\n")
	}

	// Add env section
	yaml.WriteString("                \"env\": {\n")
	// Use shell environment variable instead of GitHub Actions expression to prevent template injection
	// The actual GitHub expression is set in the step's env: block
	// Copilot uses escaped variables (\${VAR}), others use plain variables ($VAR)
	if options.IncludeTypeField {
		// Copilot engine: use escaped variable for Copilot CLI to interpolate
		yaml.WriteString("                  \"GITHUB_PERSONAL_ACCESS_TOKEN\": \"\\${GITHUB_MCP_SERVER_TOKEN}\"")
	} else {
		// Non-Copilot engines (Claude/Custom): use plain shell variable
		yaml.WriteString("                  \"GITHUB_PERSONAL_ACCESS_TOKEN\": \"$GITHUB_MCP_SERVER_TOKEN\"")
	}
	yaml.WriteString("\n")
	yaml.WriteString("                }\n")
}

// GitHubMCPRemoteOptions defines configuration for GitHub MCP remote mode rendering
type GitHubMCPRemoteOptions struct {
	// ReadOnly enables read-only mode for GitHub API operations
	ReadOnly bool
	// Toolsets specifies the GitHub toolsets to enable
	Toolsets string
	// AuthorizationValue is the value for the Authorization header
	// For Claude: "Bearer {effectiveToken}"
	// For Copilot: "Bearer \\${GITHUB_PERSONAL_ACCESS_TOKEN}"
	AuthorizationValue string
	// IncludeToolsField indicates whether to include the "tools" field (Copilot needs it, Claude doesn't)
	IncludeToolsField bool
	// AllowedTools specifies the list of allowed tools (Copilot uses this, Claude doesn't)
	AllowedTools []string
	// IncludeEnvSection indicates whether to include the env section (Copilot needs it, Claude doesn't)
	IncludeEnvSection bool
}

// RenderGitHubMCPRemoteConfig renders the GitHub MCP server configuration for remote (hosted) mode.
// This shared function extracts the duplicate pattern from Claude and Copilot engines.
//
// Parameters:
//   - yaml: The string builder for YAML output
//   - options: GitHub MCP remote rendering options
func RenderGitHubMCPRemoteConfig(yaml *strings.Builder, options GitHubMCPRemoteOptions) {
	// Remote mode - use hosted GitHub MCP server
	yaml.WriteString("                \"type\": \"http\",\n")
	yaml.WriteString("                \"url\": \"https://api.githubcopilot.com/mcp/\",\n")
	yaml.WriteString("                \"headers\": {\n")

	// Collect headers in a map
	headers := make(map[string]string)
	headers["Authorization"] = options.AuthorizationValue

	// Add X-MCP-Readonly header if read-only mode is enabled
	if options.ReadOnly {
		headers["X-MCP-Readonly"] = "true"
	}

	// Add X-MCP-Toolsets header if toolsets are configured
	if options.Toolsets != "" {
		headers["X-MCP-Toolsets"] = options.Toolsets
	}

	// Write headers using helper
	writeHeadersToYAML(yaml, headers, "                  ")

	// Close headers section
	if options.IncludeToolsField || options.IncludeEnvSection {
		yaml.WriteString("                },\n")
	} else {
		yaml.WriteString("                }\n")
	}

	// Add tools field if needed (Copilot uses this, Claude doesn't)
	if options.IncludeToolsField {
		if len(options.AllowedTools) > 0 {
			yaml.WriteString("                \"tools\": [\n")
			for i, tool := range options.AllowedTools {
				comma := ","
				if i == len(options.AllowedTools)-1 {
					comma = ""
				}
				fmt.Fprintf(yaml, "                  \"%s\"%s\n", tool, comma)
			}
			yaml.WriteString("                ],\n")
		} else {
			yaml.WriteString("                \"tools\": [\"*\"],\n")
		}
	}

	// Add env section if needed (Copilot uses this, Claude doesn't)
	if options.IncludeEnvSection {
		yaml.WriteString("                \"env\": {\n")
		yaml.WriteString("                  \"GITHUB_PERSONAL_ACCESS_TOKEN\": \"\\${GITHUB_MCP_SERVER_TOKEN}\"\n")
		yaml.WriteString("                }\n")
	}
}

// RenderJSONMCPConfig renders MCP configuration in JSON format with the common mcpServers structure.
// This shared function extracts the duplicate pattern from Claude, Copilot, and Custom engines.
//
// Parameters:
//   - yaml: The string builder for YAML output
//   - tools: Map of tool configurations
//   - mcpTools: Ordered list of MCP tool names to render
//   - workflowData: Workflow configuration data
//   - options: JSON MCP config rendering options
func RenderJSONMCPConfig(
	yaml *strings.Builder,
	tools map[string]any,
	mcpTools []string,
	workflowData *WorkflowData,
	options JSONMCPConfigOptions,
) {
	engineHelpersLog.Printf("Rendering JSON MCP config: %d tools, path=%s", len(mcpTools), options.ConfigPath)

	// Write config file header
	yaml.WriteString(fmt.Sprintf("          cat > %s << EOF\n", options.ConfigPath))
	yaml.WriteString("          {\n")
	yaml.WriteString("            \"mcpServers\": {\n")

	// Filter tools if needed (e.g., Copilot filters out cache-memory)
	var filteredTools []string
	for _, toolName := range mcpTools {
		if options.FilterTool != nil && !options.FilterTool(toolName) {
			engineHelpersLog.Printf("Filtering out MCP tool: %s", toolName)
			continue
		}
		filteredTools = append(filteredTools, toolName)
	}

	engineHelpersLog.Printf("Rendering %d MCP tools after filtering", len(filteredTools))

	// Process each MCP tool
	totalServers := len(filteredTools)
	serverCount := 0

	for _, toolName := range filteredTools {
		serverCount++
		isLast := serverCount == totalServers

		switch toolName {
		case "github":
			githubTool := tools["github"]
			options.Renderers.RenderGitHub(yaml, githubTool, isLast, workflowData)
		case "playwright":
			playwrightTool := tools["playwright"]
			options.Renderers.RenderPlaywright(yaml, playwrightTool, isLast)
		case "cache-memory":
			options.Renderers.RenderCacheMemory(yaml, isLast, workflowData)
		case "agentic-workflows":
			options.Renderers.RenderAgenticWorkflows(yaml, isLast)
		case "safe-outputs":
			options.Renderers.RenderSafeOutputs(yaml, isLast)
		case "web-fetch":
			options.Renderers.RenderWebFetch(yaml, isLast)
		default:
			// Handle custom MCP tools using shared helper
			HandleCustomMCPToolInSwitch(yaml, toolName, tools, isLast, options.Renderers.RenderCustomMCPConfig)
		}
	}

	// Write config file footer
	yaml.WriteString("            }\n")
	yaml.WriteString("          }\n")
	yaml.WriteString("          EOF\n")

	// Add any post-EOF commands (e.g., debug output for Copilot)
	if options.PostEOFCommands != nil {
		options.PostEOFCommands(yaml)
	}
}

// BuildStandardPipInstallSteps creates standard Python package installation steps
// This helper extracts the common pattern for installing Python packages via pip or uv.
//
// Parameters:
//   - packages: List of package names to install (e.g., ["requests", "numpy"])
//   - useUv: If true, uses "uv pip install" instead of "pip install"
//
// Returns:
//   - []GitHubActionStep: The installation steps including Python setup
func BuildStandardPipInstallSteps(packages []string, useUv bool) []GitHubActionStep {
	engineHelpersLog.Printf("Building pip install steps: packages=%v, useUv=%v", packages, useUv)

	if len(packages) == 0 {
		engineHelpersLog.Print("No packages to install, returning empty steps")
		return []GitHubActionStep{}
	}

	var steps []GitHubActionStep

	// Add Python setup step
	pythonSetupStep := GitHubActionStep{
		"      - name: Setup Python",
		fmt.Sprintf("        uses: %s", GetActionPin("actions/setup-python")),
		"        with:",
		"          python-version: '3.12'",
	}
	steps = append(steps, pythonSetupStep)

	// Add uv setup if needed
	if useUv {
		uvSetupStep := GitHubActionStep{
			"      - name: Setup uv",
			fmt.Sprintf("        uses: %s", GetActionPin("astral-sh/setup-uv")),
		}
		steps = append(steps, uvSetupStep)
	}

	// Build install command
	var installCmd string
	if useUv {
		installCmd = "uv pip install --system " + strings.Join(packages, " ")
	} else {
		installCmd = "pip install " + strings.Join(packages, " ")
	}

	// Add package installation step
	installStep := GitHubActionStep{
		"      - name: Install Python packages",
		fmt.Sprintf("        run: %s", installCmd),
	}
	steps = append(steps, installStep)

	return steps
}

// BuildStandardDockerSetupSteps creates standard Docker image pre-download steps
// This helper extracts the common pattern for pre-downloading Docker images used by engines.
//
// Parameters:
//   - images: List of Docker image names to pre-download (e.g., ["ghcr.io/github/github-mcp-server:v1.0.0"])
//
// Returns:
//   - []GitHubActionStep: The Docker image download step (empty if no images)
func BuildStandardDockerSetupSteps(images []string) []GitHubActionStep {
	engineHelpersLog.Printf("Building Docker setup steps: %d images", len(images))

	if len(images) == 0 {
		engineHelpersLog.Print("No Docker images to download, returning empty steps")
		return []GitHubActionStep{}
	}

	// Sort images for consistent output
	sortedImages := make([]string, len(images))
	copy(sortedImages, images)
	sort.Strings(sortedImages)

	// Build the docker pull commands
	var stepLines []string
	stepLines = append(stepLines, "      - name: Download Docker images")
	stepLines = append(stepLines, "        run: |")
	stepLines = append(stepLines, "          set -e")
	for _, image := range sortedImages {
		stepLines = append(stepLines, fmt.Sprintf("          docker pull %s", image))
	}

	return []GitHubActionStep{stepLines}
}

// BuildStandardEngineCleanupSteps creates standard cleanup steps for engines
// This helper provides a common pattern for cleanup operations after engine execution.
//
// Parameters:
//   - cleanupPaths: List of file/directory paths to remove (e.g., ["/tmp/gh-aw/.copilot/"])
//
// Returns:
//   - []GitHubActionStep: The cleanup step (empty if no paths)
func BuildStandardEngineCleanupSteps(cleanupPaths []string) []GitHubActionStep {
	engineHelpersLog.Printf("Building engine cleanup steps: paths=%v", cleanupPaths)

	if len(cleanupPaths) == 0 {
		engineHelpersLog.Print("No cleanup paths specified, returning empty steps")
		return []GitHubActionStep{}
	}

	// Sort paths for consistent output
	sortedPaths := make([]string, len(cleanupPaths))
	copy(sortedPaths, cleanupPaths)
	sort.Strings(sortedPaths)

	// Build the cleanup commands
	var stepLines []string
	stepLines = append(stepLines, "      - name: Cleanup temporary files")
	stepLines = append(stepLines, "        if: always()")
	stepLines = append(stepLines, "        run: |")
	for _, path := range sortedPaths {
		stepLines = append(stepLines, fmt.Sprintf("          rm -rf %s", path))
	}

	return []GitHubActionStep{stepLines}
}
