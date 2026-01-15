package workflow

import (
	"fmt"
	"strings"

	"github.com/githubnext/gh-aw/pkg/constants"
	"github.com/githubnext/gh-aw/pkg/logger"
)

var opencodeLog = logger.New("workflow:opencode_engine")

// OpenCodeEngine represents the OpenCode agentic engine
type OpenCodeEngine struct {
	BaseEngine
}

func NewOpenCodeEngine() *OpenCodeEngine {
	return &OpenCodeEngine{
		BaseEngine: BaseEngine{
			id:                     "opencode",
			displayName:            "OpenCode",
			description:            "Uses OpenCode with MCP tool support",
			experimental:           true,
			supportsToolsAllowlist: true,
			supportsHTTPTransport:  true, // OpenCode supports both stdio and HTTP transport
			supportsMaxTurns:       false,
			supportsWebFetch:       false,
			supportsWebSearch:      false,
			supportsFirewall:       false, // OpenCode does not support network firewalling
		},
	}
}

// GetRequiredSecretNames returns the list of secrets required by the OpenCode engine
// This includes ANTHROPIC_API_KEY, OPENAI_API_KEY, and optionally MCP_GATEWAY_API_KEY
func (e *OpenCodeEngine) GetRequiredSecretNames(workflowData *WorkflowData) []string {
	secrets := []string{"ANTHROPIC_API_KEY", "OPENAI_API_KEY"}

	// Add MCP gateway API key if MCP servers are present (gateway is always started with MCP servers)
	if HasMCPServers(workflowData) {
		secrets = append(secrets, "MCP_GATEWAY_API_KEY")
	}

	// Add safe-inputs secret names
	if IsSafeInputsEnabled(workflowData.SafeInputs, workflowData) {
		safeInputsSecrets := collectSafeInputsSecrets(workflowData.SafeInputs)
		for varName := range safeInputsSecrets {
			secrets = append(secrets, varName)
		}
	}

	return secrets
}

func (e *OpenCodeEngine) GetInstallationSteps(workflowData *WorkflowData) []GitHubActionStep {
	opencodeLog.Printf("Generating installation steps for OpenCode engine: workflow=%s", workflowData.Name)

	// Skip installation if custom command is specified
	if workflowData.EngineConfig != nil && workflowData.EngineConfig.Command != "" {
		opencodeLog.Printf("Skipping installation steps: custom command specified (%s)", workflowData.EngineConfig.Command)
		return []GitHubActionStep{}
	}

	var steps []GitHubActionStep

	// Define engine configuration for shared validation
	config := EngineInstallConfig{
		Secrets:         []string{"ANTHROPIC_API_KEY", "OPENAI_API_KEY"},
		DocsURL:         "https://githubnext.github.io/gh-aw/reference/engines/#opencode",
		NpmPackage:      "opencode-ai",
		Version:         string(constants.DefaultOpenCodeVersion),
		Name:            "OpenCode",
		CliName:         "opencode",
		InstallStepName: "Install OpenCode CLI",
	}

	// Add secret validation step
	secretValidation := GenerateMultiSecretValidationStep(
		config.Secrets,
		config.Name,
		config.DocsURL,
	)
	steps = append(steps, secretValidation)

	// Determine OpenCode version
	opencodeVersion := config.Version
	if workflowData.EngineConfig != nil && workflowData.EngineConfig.Version != "" {
		opencodeVersion = workflowData.EngineConfig.Version
	}

	// Add Node.js setup step
	npmSteps := GenerateNpmInstallSteps(
		config.NpmPackage,
		opencodeVersion,
		config.InstallStepName,
		config.CliName,
		true, // Include Node.js setup
	)

	if len(npmSteps) > 0 {
		steps = append(steps, npmSteps...)
	}

	// Add jq installation step for MCP config transformation
	jqInstallStep := []string{
		"      - name: Install jq for MCP config transformation",
		"        run: |",
		"          sudo apt-get update && sudo apt-get install -y jq",
	}
	steps = append(steps, GitHubActionStep(jqInstallStep))

	return steps
}

// GetDeclaredOutputFiles returns the output files that OpenCode may produce
func (e *OpenCodeEngine) GetDeclaredOutputFiles() []string {
	return []string{}
}

// GetExecutionSteps returns the GitHub Actions steps for executing OpenCode
func (e *OpenCodeEngine) GetExecutionSteps(workflowData *WorkflowData, logFile string) []GitHubActionStep {
	opencodeLog.Printf("Generating execution steps for OpenCode engine: workflow=%s", workflowData.Name)

	var steps []GitHubActionStep

	// Add MCP configuration step if there are MCP servers
	if HasMCPServers(workflowData) {
		opencodeLog.Print("Adding MCP configuration step")
		mcpConfigStep := e.generateMCPConfigStep(workflowData)
		steps = append(steps, mcpConfigStep)
	}

	// Handle custom steps if they exist in engine config
	customSteps := InjectCustomEngineSteps(workflowData, e.convertStepToYAML)
	steps = append(steps, customSteps...)

	// Build opencode CLI arguments based on configuration
	var opencodeArgs []string

	// Add model if specified
	modelConfigured := workflowData.EngineConfig != nil && workflowData.EngineConfig.Model != ""
	if modelConfigured {
		opencodeLog.Printf("Using custom model: %s", workflowData.EngineConfig.Model)
		opencodeArgs = append(opencodeArgs, "--model", workflowData.EngineConfig.Model)
	}

	// Add print-logs flag for enhanced debugging output
	opencodeArgs = append(opencodeArgs, "--print-logs")

	// Add custom args from engine configuration before the prompt
	if workflowData.EngineConfig != nil && len(workflowData.EngineConfig.Args) > 0 {
		opencodeArgs = append(opencodeArgs, workflowData.EngineConfig.Args...)
	}

	// Build the agent command
	var promptCommand string
	if workflowData.AgentFile != "" {
		agentPath := ResolveAgentFilePath(workflowData.AgentFile)
		opencodeLog.Printf("Using custom agent file: %s", workflowData.AgentFile)
		// Extract markdown body from custom agent file and prepend to prompt
		promptCommand = fmt.Sprintf(`"$(awk 'BEGIN{skip=1} /^---$/{if(skip){skip=0;next}else{skip=1;next}} !skip' %s && cat /tmp/gh-aw/aw-prompts/prompt.txt)"`, agentPath)
	} else {
		promptCommand = "\"$(cat /tmp/gh-aw/aw-prompts/prompt.txt)\""
	}

	// Build the command string with proper argument formatting
	var commandName string
	if workflowData.EngineConfig != nil && workflowData.EngineConfig.Command != "" {
		commandName = workflowData.EngineConfig.Command
		opencodeLog.Printf("Using custom command: %s", commandName)
	} else {
		commandName = "opencode run"
	}

	commandParts := []string{commandName}
	commandParts = append(commandParts, promptCommand)
	commandParts = append(commandParts, opencodeArgs...)

	opencodeCommand := shellJoinArgs(commandParts)

	// Prepend PATH setup to find opencode in hostedtoolcache
	pathSetup := `NODE_BIN_PATH="$(find /opt/hostedtoolcache/node -mindepth 1 -maxdepth 1 -type d | head -1 | xargs basename)/x64/bin" && export PATH="/opt/hostedtoolcache/node/$NODE_BIN_PATH:$PATH"`
	opencodeCommand = fmt.Sprintf(`%s && %s`, pathSetup, opencodeCommand)

	// Add conditional model flag if not explicitly configured
	isDetectionJob := workflowData.SafeOutputs == nil
	var modelEnvVar string
	if isDetectionJob {
		modelEnvVar = constants.EnvVarModelDetectionOpenCode
	} else {
		modelEnvVar = constants.EnvVarModelAgentOpenCode
	}
	if !modelConfigured {
		opencodeCommand = fmt.Sprintf(`%s${%s:+ --model "$%s"}`, opencodeCommand, modelEnvVar, modelEnvVar)
	}

	// Build the full command
	command := fmt.Sprintf(`set -o pipefail
          # Execute OpenCode CLI with prompt from file
          %s 2>&1 | tee %s`, opencodeCommand, shellEscapeArg(logFile))

	// Build environment variables map
	env := map[string]string{
		"ANTHROPIC_API_KEY": "${{ secrets.ANTHROPIC_API_KEY }}",
		"OPENAI_API_KEY":    "${{ secrets.OPENAI_API_KEY }}",
		"GITHUB_TOKEN":      "${{ secrets.GITHUB_TOKEN }}",
		"GH_AW_PROMPT":      "/tmp/gh-aw/aw-prompts/prompt.txt",
		"GITHUB_WORKSPACE":  "${{ github.workspace }}",
	}

	// Add GH_AW_MCP_CONFIG for MCP server configuration only if there are MCP servers
	if HasMCPServers(workflowData) {
		env["GH_AW_MCP_CONFIG"] = "/tmp/gh-aw/mcp-config/mcp-servers.json"
	}

	// Add GH_AW_SAFE_OUTPUTS if output is needed
	applySafeOutputEnvToMap(env, workflowData)

	// Add model environment variable if model is not explicitly configured
	if !modelConfigured {
		if isDetectionJob {
			env[constants.EnvVarModelDetectionOpenCode] = fmt.Sprintf("${{ vars.%s || '' }}", constants.EnvVarModelDetectionOpenCode)
		} else {
			env[constants.EnvVarModelAgentOpenCode] = fmt.Sprintf("${{ vars.%s || '' }}", constants.EnvVarModelAgentOpenCode)
		}
	}

	// Add custom environment variables from engine config
	if workflowData.EngineConfig != nil && len(workflowData.EngineConfig.Env) > 0 {
		for key, value := range workflowData.EngineConfig.Env {
			env[key] = value
		}
	}

	// Add safe-inputs secrets to env for passthrough to MCP servers
	if IsSafeInputsEnabled(workflowData.SafeInputs, workflowData) {
		safeInputsSecrets := collectSafeInputsSecrets(workflowData.SafeInputs)
		for varName, secretExpr := range safeInputsSecrets {
			if _, exists := env[varName]; !exists {
				env[varName] = secretExpr
			}
		}
	}

	// Generate the step for OpenCode CLI execution
	stepName := "Execute OpenCode CLI"
	var stepLines []string

	stepLines = append(stepLines, fmt.Sprintf("      - name: %s", stepName))
	stepLines = append(stepLines, "        id: agentic_execution")

	// Add timeout at step level (GitHub Actions standard)
	if workflowData.TimeoutMinutes != "" {
		timeoutValue := strings.TrimPrefix(workflowData.TimeoutMinutes, "timeout_minutes: ")
		timeoutValue = strings.TrimPrefix(timeoutValue, "timeout-minutes: ")
		stepLines = append(stepLines, fmt.Sprintf("        timeout-minutes: %s", timeoutValue))
	} else {
		stepLines = append(stepLines, fmt.Sprintf("        timeout-minutes: %d", constants.DefaultAgenticWorkflowTimeoutMinutes))
	}

	// Filter environment variables to only include allowed secrets
	allowedSecrets := e.GetRequiredSecretNames(workflowData)
	filteredEnv := FilterEnvForSecrets(env, allowedSecrets)

	// Format step with command and filtered environment variables using shared helper
	stepLines = FormatStepWithCommandAndEnv(stepLines, command, filteredEnv)

	steps = append(steps, GitHubActionStep(stepLines))

	return steps
}

// generateMCPConfigStep creates a GitHub Actions step that transforms MCP config for OpenCode
func (e *OpenCodeEngine) generateMCPConfigStep(workflowData *WorkflowData) GitHubActionStep {
	stepLines := []string{
		"      - name: Configure OpenCode MCP servers",
		"        run: |",
		"          set -e",
		"          ",
		"          # Create OpenCode config directory",
		"          mkdir -p ~/.config/opencode",
		"          ",
		"          # Check if MCP config exists",
		"          if [ -n \"$GH_AW_MCP_CONFIG\" ] && [ -f \"$GH_AW_MCP_CONFIG\" ]; then",
		"            echo \"Found MCP configuration at: $GH_AW_MCP_CONFIG\"",
		"            ",
		"            # Create base OpenCode config with proper schema",
		"            echo '{\"$schema\": \"https://opencode.sh/schema.json\", \"mcp\": {}}' > ~/.config/opencode/opencode.json",
		"            ",
		"            # Transform Copilot-style MCP config to OpenCode format",
		"            jq -r '.mcpServers | to_entries[] | ",
		"              if .value.type == \"local\" or (.value.command and .value.args) then",
		"                {",
		"                  key: .key,",
		"                  value: {",
		"                    type: \"local\",",
		"                    command: (",
		"                      if .value.command then",
		"                        [.value.command] + (.value.args // [])",
		"                      else",
		"                        []",
		"                      end",
		"                    ),",
		"                    enabled: true,",
		"                    environment: (.value.env // {})",
		"                  }",
		"                }",
		"              elif .value.type == \"http\" then",
		"                {",
		"                  key: .key,",
		"                  value: {",
		"                    type: \"remote\",",
		"                    url: .value.url,",
		"                    enabled: true,",
		"                    headers: (.value.headers // {})",
		"                  }",
		"                }",
		"              else empty end' \"$GH_AW_MCP_CONFIG\" | \\",
		"              jq -s 'reduce .[] as $item ({}; .[$item.key] = $item.value)' > /tmp/mcp-servers.json",
		"            ",
		"            # Merge MCP servers into config",
		"            jq --slurpfile servers /tmp/mcp-servers.json '.mcp = $servers[0]' \\",
		"              ~/.config/opencode/opencode.json > /tmp/opencode-final.json",
		"            mv /tmp/opencode-final.json ~/.config/opencode/opencode.json",
		"            ",
		"            echo \"✅ OpenCode MCP configuration created successfully\"",
		"            echo \"Configuration contents:\"",
		"            cat ~/.config/opencode/opencode.json | jq .",
		"          else",
		"            echo \"⚠️  No MCP config found - OpenCode will run without MCP tools\"",
		"          fi",
		"        env:",
		"          GH_AW_MCP_CONFIG: ${{ env.GH_AW_MCP_CONFIG }}",
	}
	return GitHubActionStep(stepLines)
}

// GetLogParserScriptId returns the JavaScript script name for parsing OpenCode logs
func (e *OpenCodeEngine) GetLogParserScriptId() string {
	// OpenCode doesn't have a specific log parser yet, use a generic one
	return "parse_generic_log"
}

// RenderMCPConfig renders the MCP configuration for OpenCode
// OpenCode uses Claude-style MCP configuration
func (e *OpenCodeEngine) RenderMCPConfig(yaml *strings.Builder, tools map[string]any, mcpTools []string, workflowData *WorkflowData) {
	// Use Claude's MCP config rendering since OpenCode uses the same format
	claudeEngine := NewClaudeEngine()
	claudeEngine.RenderMCPConfig(yaml, tools, mcpTools, workflowData)
}

// ParseLogMetrics extracts metrics from engine-specific log content
func (e *OpenCodeEngine) ParseLogMetrics(logContent string, verbose bool) LogMetrics {
	// For now, return empty metrics. Can be enhanced later with OpenCode-specific parsing
	return LogMetrics{}
}
