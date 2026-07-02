// This file provides Copilot engine installation logic.
//
// This file contains functions for generating GitHub Actions steps to install
// the GitHub Copilot CLI and related sandbox infrastructure (AWF or SRT).
//
// Installation order:
//  1. Secret validation (COPILOT_GITHUB_TOKEN) — runs in the activation job
//  2. Node.js setup
//  3. Sandbox installation (SRT or AWF, if needed)
//  4. Copilot CLI installation
//
// The installation strategy differs based on sandbox mode:
//   - Standard mode: Global installation using official installer script
//   - SRT mode: Local npm installation for offline compatibility
//   - AWF mode: Global installation + AWF binary

package workflow

import (
	"path/filepath"
	"strings"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
)

var copilotInstallLog = logger.New("workflow:copilot_engine_installation")

type copilotSDKInstallSpec struct {
	runtimeID string
	stepName  string
	command   string
}

const workspaceCommandPrefix = `cd "${GITHUB_WORKSPACE}" && `

// getWorkspaceCommandPrefixFor returns the shell cd prefix for engine command generation.
// When engine.cwd is configured it returns a prefix that changes to ${GH_AW_ENGINE_CWD}
// (set as an env var by applyEngineCwdEnv). When engine.cwd is not configured it falls
// back to the default workspace prefix.
func getWorkspaceCommandPrefixFor(config *EngineConfig) string {
	if config != nil && config.Cwd != "" {
		return `cd "${GH_AW_ENGINE_CWD}" && `
	}
	return workspaceCommandPrefix
}

// GetSecretValidationStep returns the secret validation step for the Copilot engine.
// Returns an empty step if:
//   - permissions.copilot-requests is set to write (uses GitHub Actions token instead), or
//   - COPILOT_PROVIDER_BASE_URL, COPILOT_PROVIDER_API_KEY, or COPILOT_PROVIDER_BEARER_TOKEN is set in engine.env
//     (BYOK mode — the external provider handles authentication, so COPILOT_GITHUB_TOKEN
//     is not required for model routing).
func (e *CopilotEngine) GetSecretValidationStep(workflowData *WorkflowData) GitHubActionStep {
	provider := e.ResolveLLMProvider(workflowData)
	if provider == LLMProviderGitHub && hasCopilotRequestsWritePermission(workflowData) {
		copilotInstallLog.Print("Skipping secret validation step: permissions.copilot-requests=write enabled, using GitHub Actions token")
		return GitHubActionStep{}
	}
	if engineEnvHasKey(workflowData, constants.CopilotProviderBaseURL) ||
		engineEnvHasKey(workflowData, constants.CopilotProviderAPIKey) ||
		engineEnvHasKey(workflowData, constants.CopilotProviderBearerToken) {
		copilotInstallLog.Print("Skipping COPILOT_GITHUB_TOKEN validation: BYOK provider credentials are configured")
		return GitHubActionStep{}
	}
	return BuildDefaultSecretValidationStep(
		workflowData,
		llmProviderSecretNames(provider),
		"GitHub Copilot CLI",
		llmProviderDocsURL(provider),
	)
}

// GetInstallationSteps generates the complete installation workflow for Copilot CLI.
// This includes Node.js setup, sandbox installation (SRT or AWF), and Copilot CLI installation.
// Secret validation is handled separately in the activation job via GetSecretValidationStep.
// The installation order is:
// 1. Node.js setup
// 2. Sandbox installation (AWF, if needed)
// 3. Copilot CLI installation
//
// If a custom command is specified in the engine configuration, this function skips
// standard Copilot CLI installation. When firewall is enabled, it still returns AWF
// runtime installation steps required for harness execution.
func (e *CopilotEngine) GetInstallationSteps(workflowData *WorkflowData) []GitHubActionStep {
	copilotInstallLog.Printf("Generating installation steps for Copilot engine: workflow=%s", workflowData.Name)
	sdkInstallStep := buildCopilotSDKInstallStep(workflowData)

	// Skip standard Copilot CLI installation if custom command is specified.
	if workflowData.EngineConfig != nil && workflowData.EngineConfig.Command != "" {
		// Keep firewall runtime installation when firewall is enabled, since the
		// custom engine command still runs inside the AWF harness.
		if isFirewallEnabled(workflowData) {
			copilotInstallLog.Printf("Skipping Copilot CLI installation: custom command specified (%s); keeping AWF runtime installation because firewall is enabled", workflowData.EngineConfig.Command)
			var steps []GitHubActionStep
			if len(sdkInstallStep) > 0 {
				steps = append(steps, sdkInstallStep)
			}
			return appendCopilotLSPInstallSteps(BuildNpmEngineInstallStepsWithAWF(steps, workflowData), workflowData)
		}
		if len(sdkInstallStep) > 0 {
			copilotInstallLog.Printf("Skipping Copilot CLI installation: custom command specified (%s); keeping Copilot SDK install step", workflowData.EngineConfig.Command)
			return appendCopilotLSPInstallSteps([]GitHubActionStep{sdkInstallStep}, workflowData)
		}
		copilotInstallLog.Printf("Skipping installation steps: custom command specified (%s)", workflowData.EngineConfig.Command)
		return appendCopilotLSPInstallSteps([]GitHubActionStep{}, workflowData)
	}

	// Copilot CLI is pinned to the default version constant.
	copilotVersion := string(constants.DefaultCopilotVersion)
	if workflowData.EngineConfig != nil {
		if workflowData.EngineConfig.Version != "" {
			copilotInstallLog.Printf("Ignoring pinned engine.version (%s): Copilot CLI install version is pinned to %s", workflowData.EngineConfig.Version, copilotVersion)
		}
		// Normalize engine config version to effective installed version so
		// downstream checks that consult EngineConfig.Version stay consistent.
		// This applies even when the original version was empty (unset), so all
		// downstream consumers observe the effective installed value.
		// This mutates workflowData by design because subsequent generation steps
		// in the same compile flow should observe the effective installed version.
		// Callers that reuse the same WorkflowData instance should expect this
		// field to be rewritten after installation-step generation.
		workflowData.EngineConfig.Version = copilotVersion
	}

	// Use the installer script for global installation
	copilotInstallLog.Print("Using new installer script for Copilot installation")
	npmSteps := GenerateCopilotInstallerSteps(copilotVersion, "Install GitHub Copilot CLI")
	if len(sdkInstallStep) > 0 {
		npmSteps = append(npmSteps, sdkInstallStep)
	}
	steps := BuildNpmEngineInstallStepsWithAWF(npmSteps, workflowData)

	return appendCopilotLSPInstallSteps(steps, workflowData)
}

func appendCopilotLSPInstallSteps(steps []GitHubActionStep, workflowData *WorkflowData) []GitHubActionStep {
	if workflowData == nil {
		return steps
	}
	manager := NewLSPManager(workflowData.LSP)
	lspSteps := manager.GenerateInstallSteps(workflowData)
	if len(lspSteps) == 0 {
		return steps
	}
	copilotInstallLog.Printf("Adding %d LSP dependency installation step(s)", len(lspSteps))
	return append(steps, lspSteps...)
}

func buildCopilotSDKInstallStep(workflowData *WorkflowData) GitHubActionStep {
	if workflowData == nil || workflowData.EngineConfig == nil || !workflowData.EngineConfig.CopilotSDK {
		return GitHubActionStep{}
	}
	// When a custom SDK driver is configured without a custom engine command, use the driver's
	// file extension to determine which language SDK to install. This ensures the correct SDK
	// package manager command is generated (e.g., pip for .py drivers, ruby/gem for .rb drivers).
	command := workflowData.EngineConfig.Command
	if command == "" && workflowData.EngineConfig.Driver != "" {
		command = sdkDriverInstallCommand(workflowData.EngineConfig.Driver)
	}
	spec := getCopilotSDKInstallSpec(command)
	copilotInstallLog.Printf("copilot-sdk enabled; runtime=%s; install command=%s", spec.runtimeID, spec.command)
	return GitHubActionStep{
		"      - name: " + spec.stepName,
		"        run: " + spec.command,
	}
}

// sdkDriverInstallCommand returns a synthetic command string for the given driver filename
// that can be passed to getCopilotSDKInstallSpec/detectRuntimeFromCopilotCommand to select
// the correct SDK package manager. Only non-JS language extensions need special handling;
// JS drivers and arbitrary commands (no extension) fall back to the Node.js default.
func sdkDriverInstallCommand(driverName string) string {
	ext := strings.ToLower(filepath.Ext(driverName))
	switch ext {
	case ".py":
		return "python3 " + driverName
	case ".rb":
		return "ruby " + driverName
	case ".ts", ".mts":
		return "ts-node " + driverName
	default:
		// .js/.cjs/.mjs and no-extension (arbitrary commands) default to Node.js.
		return ""
	}
}

func getCopilotSDKInstallSpec(command string) copilotSDKInstallSpec {
	runtimeID := detectRuntimeFromCopilotCommand(command)
	version := string(constants.DefaultCopilotSDKVersion)

	spec := copilotSDKInstallSpec{
		runtimeID: runtimeID,
		stepName:  "Install GitHub Copilot SDK (Node.js)",
		command:   workspaceCommandPrefix + "npm install --ignore-scripts --no-save @github/copilot-sdk@" + version,
	}

	switch runtimeID {
	case "python":
		spec.stepName = "Install GitHub Copilot SDK (Python)"
		spec.command = workspaceCommandPrefix + "python3 -m pip install --disable-pip-version-check github-copilot-sdk==" + version
	case "typescript":
		spec.stepName = "Install GitHub Copilot SDK (TypeScript)"
		spec.command = workspaceCommandPrefix + "npm install --ignore-scripts --no-save @github/copilot-sdk@" + version + " ts-node typescript"
	case "go":
		spec.stepName = "Install GitHub Copilot SDK (Go)"
		spec.command = workspaceCommandPrefix + "go get github.com/github/copilot-sdk/go@v" + version
	case "rust":
		spec.stepName = "Install GitHub Copilot SDK (Rust)"
		spec.command = workspaceCommandPrefix + "cargo add github-copilot-sdk@" + version
	case "dotnet":
		spec.stepName = "Install GitHub Copilot SDK (.NET)"
		spec.command = workspaceCommandPrefix + "dotnet add package GitHub.Copilot.SDK --version " + version
	case "java":
		spec.stepName = "Install GitHub Copilot SDK (Java)"
		spec.command = workspaceCommandPrefix + "mvn -q org.apache.maven.plugins:maven-dependency-plugin:3.8.1:get -Dartifact=com.github:copilot-sdk-java:" + version
	}

	return spec
}

func detectRuntimeFromCopilotCommand(command string) string {
	token := firstCommandToken(command)
	if token == "" {
		return "node"
	}

	runtime, found := commandToRuntime[token]
	if found && runtime.ID != "" {
		return runtime.ID
	}

	switch token {
	case "ts-node":
		return "typescript"
	case "cargo", "rustc":
		return "rust"
	case "mvnw":
		return "java"
	}
	return "node"
}

func firstCommandToken(command string) string {
	fields := strings.Fields(command)
	if len(fields) == 0 {
		return ""
	}
	token := normalizeCommandToken(fields[0])
	if token != "env" {
		return token
	}
	// Shell-form commands sometimes start with `env` wrappers:
	//   env FOO=bar python app.py
	// Skip env assignments/flags and return the first executable token.
	for _, field := range fields[1:] {
		if strings.Contains(field, "=") || strings.HasPrefix(field, "-") {
			continue
		}
		return normalizeCommandToken(field)
	}
	return ""
}

func normalizeCommandToken(token string) string {
	trimmed := strings.Trim(token, `"'`)
	if trimmed == "" {
		return ""
	}
	return strings.ToLower(filepath.Base(trimmed))
}

// generateAWFInstallationStep creates a GitHub Actions step to install the AWF binary
// with SHA256 checksum verification to protect against supply chain attacks.
//
// The installation logic is implemented in a separate shell script (install_awf_binary.sh)
// which downloads the binary directly from GitHub releases, verifies its checksum against
// the official checksums.txt file, and installs it. This approach:
// - Eliminates trust in the installer script itself
// - Provides full transparency of the installation process
// - Protects against tampered or compromised installer scripts
// - Verifies the binary integrity before execution
//
// If a custom command is specified in the agent config, the installation is skipped
// as the custom command replaces the AWF binary.
func generateAWFInstallationStep(version string, agentConfig *AgentSandboxConfig) GitHubActionStep {
	// If custom command is specified, skip installation (command replaces binary)
	if agentConfig != nil && agentConfig.Command != "" {
		copilotInstallLog.Print("Skipping AWF binary installation (custom command specified)")
		// Return empty step - custom command will be used in execution
		return GitHubActionStep([]string{})
	}

	// Use default version for logging when not specified
	if version == "" {
		version = string(constants.DefaultFirewallVersion)
	}

	installCmd := "bash \"${RUNNER_TEMP}/gh-aw/actions/install_awf_binary.sh\" " + version
	// When sudo is false (network isolation mode), AWF runs rootless: pass --rootless
	// so the install script installs into $HOME/.local/{bin,lib/awf} (always writable,
	// even on standard GitHub-hosted runners where /usr/local is root-owned) and exports
	// $GITHUB_PATH so the bare awf invocation in later steps resolves correctly.
	// Also check Disabled to match isAWFNetworkIsolationEnabled() behavior.
	if agentConfig != nil && agentConfig.NetworkIsolation && !agentConfig.Disabled {
		installCmd += " --rootless"
	}

	stepLines := []string{
		"      - name: Install AWF binary",
		"        run: " + installCmd,
	}

	return GitHubActionStep(stepLines)
}

// generateDockerComposeInstallStep creates a step that installs the Docker Compose
// CLI plugin. ARC/DinD runners may not have Docker Compose pre-installed, but AWF
// requires it to orchestrate the squid-proxy, agent, and api-proxy containers.
func generateDockerComposeInstallStep() GitHubActionStep {
	return GitHubActionStep([]string{
		"      - name: Install Docker Compose plugin",
		"        run: |",
		`          export DOCKER_CONFIG="${DOCKER_CONFIG:-$HOME/.docker}"`,
		`          mkdir -p "$DOCKER_CONFIG/cli-plugins"`,
		`          arch="$(uname -m)"`,
		`          case "$arch" in x86_64|amd64) arch="x86_64" ;; aarch64|arm64) arch="aarch64" ;; *) echo "Unsupported architecture for docker compose plugin: $arch" >&2; exit 1 ;; esac`,
		`          curl -fsSL "https://github.com/docker/compose/releases/download/v2.36.2/docker-compose-linux-$arch" -o "$DOCKER_CONFIG/cli-plugins/docker-compose"`,
		`          chmod +x "$DOCKER_CONFIG/cli-plugins/docker-compose"`,
		`          docker compose version`,
	})
}
