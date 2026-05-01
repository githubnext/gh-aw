package workflow

import (
	"fmt"
	"maps"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
)

var crushLog = logger.New("workflow:crush_engine")

// crushNpmGlobalPrefix is the writable npm global prefix used when installing
// Crush CLI. GitHub-hosted runners mount the Node.js toolcache at
// /opt/hostedtoolcache with EROFS (read-only); npm install -g without a custom
// prefix fails. Pointing npm to $RUNNER_TEMP keeps the install writable while
// making the resulting binary findable via GetCrushNpmBinPathSetup.
const crushNpmGlobalPrefix = "${RUNNER_TEMP}/npm-global"

// CrushEngine represents the Crush CLI agentic engine.
// Crush is a provider-agnostic, open-source AI coding agent with broader BYOK
// (Bring Your Own Key) support, but gh-aw currently supports a subset of
// providers for engine.model validation: copilot, anthropic, openai, and codex.
type CrushEngine struct {
	UniversalLLMConsumerEngine
}

func NewCrushEngine() *CrushEngine {
	return &CrushEngine{
		UniversalLLMConsumerEngine: UniversalLLMConsumerEngine{
			BaseEngine: BaseEngine{
				id:                     "crush",
				displayName:            "Crush",
				description:            "Crush CLI with headless mode and multi-provider LLM support",
				experimental:           true,  // Start as experimental until smoke tests pass consistently
				supportsToolsAllowlist: false, // Crush manages its own tool permissions via .crush.json
				supportsMaxTurns:       false, // No --max-turns flag in crush run
				supportsWebSearch:      false, // Has built-in websearch but not exposed via gh-aw neutral tools yet
			},
		},
	}
}

// GetModelEnvVarName returns the native environment variable name that the Crush CLI uses
// for model selection. Setting CRUSH_MODEL is equivalent to passing --model to the CLI.
func (e *CrushEngine) GetModelEnvVarName() string {
	return constants.CrushCLIModelEnvVar
}

// GetRequiredSecretNames returns the list of secrets required by the Crush engine.
// By default, Crush routes through the Copilot API using COPILOT_GITHUB_TOKEN
// (or ${{ github.token }} when copilot-requests feature is enabled).
// Additional provider API keys can be added via engine.env overrides.
func (e *CrushEngine) GetRequiredSecretNames(workflowData *WorkflowData) []string {
	crushLog.Print("Collecting required secrets for Crush engine")
	return e.GetUniversalRequiredSecretNames(workflowData)
}

// GetInstallationSteps returns the GitHub Actions steps needed to install Crush CLI
func (e *CrushEngine) GetInstallationSteps(workflowData *WorkflowData) []GitHubActionStep {
	crushLog.Printf("Generating installation steps for Crush engine: workflow=%s", workflowData.Name)

	// Skip installation if custom command is specified
	if workflowData.EngineConfig != nil && workflowData.EngineConfig.Command != "" {
		crushLog.Printf("Skipping installation steps: custom command specified (%s)", workflowData.EngineConfig.Command)
		return []GitHubActionStep{}
	}

	// Determine version to install
	version := string(constants.DefaultCrushVersion)
	if workflowData.EngineConfig != nil && workflowData.EngineConfig.Version != "" {
		version = workflowData.EngineConfig.Version
	}

	// Use Node.js setup + custom install step that redirects npm's global prefix
	// to a writable directory. GitHub-hosted runners mount the Node.js toolcache
	// at /opt/hostedtoolcache with EROFS (read-only); npm install -g without a
	// custom prefix fails with EROFS.
	npmSteps := []GitHubActionStep{
		GenerateNodeJsSetupStep(),
		e.buildCrushInstallStep(version),
	}
	return BuildNpmEngineInstallStepsWithAWF(npmSteps, workflowData)
}

// buildCrushInstallStep generates an npm install step for the Crush CLI that
// redirects npm's global prefix to a writable directory under $RUNNER_TEMP.
// GitHub-hosted runners mount /opt/hostedtoolcache with EROFS (read-only
// filesystem) so a plain `npm install -g` fails. By setting NPM_CONFIG_PREFIX
// we avoid the read-only toolcache while keeping the binary accessible via
// GetCrushNpmBinPathSetup at execution time.
func (e *CrushEngine) buildCrushInstallStep(version string) GitHubActionStep {
	baseCmd := "export NPM_CONFIG_PREFIX=\"" + crushNpmGlobalPrefix + "\"\nmkdir -p \"" + crushNpmGlobalPrefix + "/bin\"\n"

	var command string
	var env map[string]string

	if ExpressionPattern.MatchString(version) {
		// Version is a GitHub Actions expression – pass through an env var to
		// prevent shell injection if the expression evaluates to an attacker-
		// controlled string.
		command = baseCmd + `npm install --ignore-scripts -g @charmland/crush@"${ENGINE_VERSION}"`
		env = map[string]string{"ENGINE_VERSION": version}
	} else {
		command = baseCmd + "npm install --ignore-scripts -g @charmland/crush@" + version
	}

	stepLines := []string{"      - name: Install Crush CLI"}
	stepLines = FormatStepWithCommandAndEnv(stepLines, command, env)
	return GitHubActionStep(stepLines)
}

// GetCrushNpmBinPathSetup returns a shell command that prepends Crush's writable
// npm global bin directory to PATH, followed by the standard hostedtoolcache bin
// directories. Call this before executing the crush binary so it is found
// regardless of the runner's Node.js toolcache layout.
func GetCrushNpmBinPathSetup() string {
	return `export PATH="` + crushNpmGlobalPrefix + `/bin:$(find /opt/hostedtoolcache /home/runner/work/_tool -maxdepth 4 -type d -name bin 2>/dev/null | tr '\n' ':')$PATH"; [ -n "$GOROOT" ] && export PATH="$GOROOT/bin:$PATH" || true`
}

// GetSecretValidationStep returns the secret validation step for the Crush engine.
// Returns an empty step if copilot-requests feature is enabled (uses GitHub Actions token).
func (e *CrushEngine) GetSecretValidationStep(workflowData *WorkflowData) GitHubActionStep {
	return e.GetUniversalSecretValidationStep(
		workflowData,
		"Crush CLI",
		"https://github.github.com/gh-aw/reference/engines/#crush",
	)
}

// GetAgentManifestFiles returns Crush-specific instruction files that should be
// treated as security-sensitive manifests. Modifying these files can change the
// agent's instructions, permissions, or configuration on the next run.
// .crush.json is the primary Crush config file; AGENTS.md is the cross-engine
// convention that Crush also reads.
func (e *CrushEngine) GetAgentManifestFiles() []string {
	return []string{".crush.json", "AGENTS.md"}
}

// GetAgentManifestPathPrefixes returns Crush-specific config directory prefixes
// that must be protected from fork PR injection.
// The .crush/ directory contains agent configuration, instructions, and other
// settings that could alter agent behaviour.
func (e *CrushEngine) GetAgentManifestPathPrefixes() []string {
	return []string{".crush/"}
}

// GetDeclaredOutputFiles returns the output files that Crush may produce.
func (e *CrushEngine) GetDeclaredOutputFiles() []string {
	return []string{}
}

// GetExecutionSteps returns the GitHub Actions steps for executing Crush
func (e *CrushEngine) GetExecutionSteps(workflowData *WorkflowData, logFile string) []GitHubActionStep {
	crushLog.Printf("Generating execution steps for Crush engine: workflow=%s, firewall=%v",
		workflowData.Name, isFirewallEnabled(workflowData))

	var steps []GitHubActionStep

	// Step 1: Write .crush.json config (permissions)
	configStep := e.generateCrushConfigStep(workflowData)
	steps = append(steps, configStep)

	// Step 2: Build CLI arguments
	var crushArgs []string

	modelConfigured := workflowData.EngineConfig != nil && workflowData.EngineConfig.Model != ""

	// Enable verbose logging for debugging in CI
	crushArgs = append(crushArgs, "--verbose")

	// Prompt from file (positional argument to `crush run`).
	// Keep this outside shellJoinArgs so command substitution expands at runtime.
	promptArg := "\"$(cat /tmp/gh-aw/aw-prompts/prompt.txt)\""

	// Build command name
	commandName := "crush"
	if workflowData.EngineConfig != nil && workflowData.EngineConfig.Command != "" {
		commandName = workflowData.EngineConfig.Command
	}
	crushCommand := fmt.Sprintf("%s run %s %s", commandName, shellJoinArgs(crushArgs), promptArg)

	// AWF wrapping
	firewallEnabled := isFirewallEnabled(workflowData)
	var command string
	if firewallEnabled {
		// Resolve model for provider-specific domain allowlisting
		model := ""
		if modelConfigured {
			model = workflowData.EngineConfig.Model
		}
		// The model was validated by validateUniversalLLMConsumerModel before reaching here,
		// so a malformed model (e.g. leading slash) must never occur. Panic is the correct
		// response to an internal invariant violation.
		allowedDomains, err := GetCrushAllowedDomainsWithToolsAndRuntimes(
			model,
			workflowData.NetworkPermissions,
			workflowData.Tools,
			workflowData.Runtimes,
		)
		if err != nil {
			panic(fmt.Sprintf("BUG: invalid model %q reached domain computation (should have been caught by validation): %v", model, err))
		}

		npmPathSetup := GetCrushNpmBinPathSetup()
		crushCommandWithPath := fmt.Sprintf("%s && %s", npmPathSetup, crushCommand)
		if mcpCLIPath := GetMCPCLIPathSetup(workflowData); mcpCLIPath != "" {
			crushCommandWithPath = fmt.Sprintf("%s && %s", mcpCLIPath, crushCommandWithPath)
		}

		command = BuildAWFCommand(AWFCommandConfig{
			EngineName:     "crush",
			EngineCommand:  crushCommandWithPath,
			LogFile:        logFile,
			WorkflowData:   workflowData,
			UsesTTY:        false,
			AllowedDomains: allowedDomains,
		})
	} else {
		// Add PATH setup so crush is found from its writable npm global prefix.
		pathSetup := GetCrushNpmBinPathSetup()
		command = fmt.Sprintf("set -o pipefail\n%s\n%s 2>&1 | tee -a %s", pathSetup, crushCommand, logFile)
	}

	env := map[string]string{
		"GH_AW_PROMPT":     "/tmp/gh-aw/aw-prompts/prompt.txt",
		"GITHUB_WORKSPACE": "${{ github.workspace }}",
		"NO_PROXY":         "localhost,127.0.0.1",
	}
	e.ApplyUniversalProviderEnv(env, workflowData, firewallEnabled)

	// MCP config path
	if HasMCPServers(workflowData) {
		env["GH_AW_MCP_CONFIG"] = "${{ github.workspace }}/.crush.json"
	}

	// Safe outputs env
	applySafeOutputEnvToMap(env, workflowData)

	// Model env var (only when explicitly configured)
	if modelConfigured {
		crushLog.Printf("Setting %s env var for model: %s",
			constants.CrushCLIModelEnvVar, workflowData.EngineConfig.Model)
		env[constants.CrushCLIModelEnvVar] = workflowData.EngineConfig.Model
	}

	// Custom env from engine config (allows provider override)
	if workflowData.EngineConfig != nil && len(workflowData.EngineConfig.Env) > 0 {
		maps.Copy(env, workflowData.EngineConfig.Env)
	}

	// Agent config env
	agentConfig := getAgentConfig(workflowData)
	if agentConfig != nil && len(agentConfig.Env) > 0 {
		maps.Copy(env, agentConfig.Env)
	}

	// Build execution step
	stepLines := []string{
		"      - name: Execute Crush CLI",
		"        id: agentic_execution",
	}
	allowedSecrets := e.GetRequiredSecretNames(workflowData)
	filteredEnv := FilterEnvForSecrets(env, allowedSecrets)
	stepLines = FormatStepWithCommandAndEnv(stepLines, command, filteredEnv)

	steps = append(steps, GitHubActionStep(stepLines))
	return steps
}

// generateCrushConfigStep writes .crush.json with all permissions set to allow
// to prevent CI hanging on permission prompts.
func (e *CrushEngine) generateCrushConfigStep(_ *WorkflowData) GitHubActionStep {
	// Build the config JSON with all permissions set to allow
	// OpenCode/Crush uses "permission" (singular) — "permissions" (plural) is silently ignored.
	// "external_directory" must be "allow" in non-interactive CI mode (defaults to "ask" → implicit deny).
	configJSON := `{"agent":{"build":{"permission":{"bash":"allow","edit":"allow","read":"allow","glob":"allow","grep":"allow","write":"allow","webfetch":"allow","websearch":"allow","external_directory":"allow"}}}}`

	// Shell command to write or merge the config with restrictive permissions
	command := fmt.Sprintf(`umask 077
mkdir -p "$GITHUB_WORKSPACE"
CONFIG="$GITHUB_WORKSPACE/.crush.json"
BASE_CONFIG='%s'
if [ -f "$CONFIG" ]; then
  MERGED=$(jq -n --argjson base "$BASE_CONFIG" --argjson existing "$(cat "$CONFIG")" '$existing * $base')
  echo "$MERGED" > "$CONFIG"
else
  echo "$BASE_CONFIG" > "$CONFIG"
fi
chmod 600 "$CONFIG"`, configJSON)

	stepLines := []string{"      - name: Write Crush Config"}
	stepLines = FormatStepWithCommandAndEnv(stepLines, command, nil)
	return GitHubActionStep(stepLines)
}
