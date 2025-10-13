package workflow

import (
	_ "embed"
	"fmt"
	"strings"
)

//go:embed config/squid-tproxy.conf
var squidTPROXYConfigContent string

const copilotLogsFolder = "/tmp/gh-aw/.copilot/logs/"

// generateSquidTPROXYConfig generates Squid configuration with TPROXY support
// This configuration supports both HTTP (port 3128) and HTTPS (port 3129) proxying
func generateSquidTPROXYConfig() string {
	return squidTPROXYConfigContent
}

// needsEngineProxy determines if engine execution requires proxy setup
// Only enabled when NetworkPermissions is explicitly configured with allowed domains
func needsEngineProxy(workflowData *WorkflowData) (bool, []string) {
	// If no network permissions configured, don't use proxy
	// This is the case for tests and workflows without network restrictions
	if workflowData.NetworkPermissions == nil {
		return false, nil
	}

	// "defaults" mode means use default ecosystem domains via hooks (non-containerized)
	// Don't use containerized proxy for backward compatibility
	if workflowData.NetworkPermissions.Mode == "defaults" {
		return false, nil
	}

	// Get allowed domains from network permissions
	// This includes:
	// - explicit allowed list → those domains
	// - empty allowed list → deny-all (empty array)
	domains := GetAllowedDomains(workflowData.NetworkPermissions)

	// Enable proxy with the determined domains
	return true, domains
}

// generateInlineEngineProxyConfig generates proxy configuration files inline in the workflow
// This includes Squid TPROXY config, allowed domains file, and Docker Compose configuration
func (c *Compiler) generateInlineEngineProxyConfig(yaml *strings.Builder, workflowData *WorkflowData) {
	needsProxySetup, allowedDomains := needsEngineProxy(workflowData)
	if !needsProxySetup {
		return
	}

	if c.verbose {
		fmt.Printf("Generating inline engine proxy configuration with %d allowed domains\n", len(allowedDomains))
	}

	yaml.WriteString("      - name: Generate Engine Proxy Configuration\n")
	yaml.WriteString("        run: |\n")

	// Generate squid-tproxy.conf inline
	yaml.WriteString("          # Generate Squid TPROXY configuration for transparent proxy\n")
	yaml.WriteString("          cat > squid-tproxy.conf << 'EOF'\n")
	squidConfig := generateSquidTPROXYConfig()
	for _, line := range strings.Split(squidConfig, "\n") {
		fmt.Fprintf(yaml, "          %s\n", line)
	}
	yaml.WriteString("          EOF\n")
	yaml.WriteString("          \n")

	// Generate allowed_domains.txt inline (reuse existing function)
	yaml.WriteString("          # Generate allowed domains file for proxy ACL\n")
	yaml.WriteString("          cat > allowed_domains.txt << 'EOF'\n")
	allowedDomainsContent := generateAllowedDomainsFile(allowedDomains)
	for _, line := range strings.Split(allowedDomainsContent, "\n") {
		fmt.Fprintf(yaml, "          %s\n", line)
	}
	yaml.WriteString("          EOF\n")
	yaml.WriteString("          \n")

	// Generate docker-compose-engine.yml inline
	yaml.WriteString("          # Generate Docker Compose configuration for containerized engine\n")
	yaml.WriteString("          cat > docker-compose-engine.yml << 'EOF'\n")

	// Get engine configuration details
	engineID := "claude" // default
	engineVersion := ""
	envVars := make(map[string]string)

	if workflowData.EngineConfig != nil {
		if workflowData.EngineConfig.ID != "" {
			engineID = workflowData.EngineConfig.ID
		}
		if workflowData.EngineConfig.Version != "" {
			engineVersion = workflowData.EngineConfig.Version
		}
		// Copy engine-specific environment variables
		for k, v := range workflowData.EngineConfig.Env {
			envVars[k] = v
		}
	}

	// Build agent command
	agentCommand := buildAgentCommand(engineID, engineVersion, workflowData)

	// Generate Docker Compose content
	dockerComposeContent := generateEngineDockerCompose(engineID, engineVersion, envVars,
		allowedDomains, agentCommand, workflowData)
	for _, line := range strings.Split(dockerComposeContent, "\n") {
		fmt.Fprintf(yaml, "          %s\n", line)
	}
	yaml.WriteString("          EOF\n")
	yaml.WriteString("          \n")
}

// buildAgentCommand builds the command to run inside the agent container
// This installs the appropriate CLI tool and executes it with the right arguments
func buildAgentCommand(engineID string, engineVersion string, workflowData *WorkflowData) []string {
	var command []string

	switch engineID {
	case "claude":
		// For Claude, we'll use sh -c to install and run in one command
		command = append(command, "sh", "-c")

		// Build install and run command
		installCmd := fmt.Sprintf("npm install -g @anthropic-ai/claude-code@%s", engineVersion)

		// Build claude CLI command
		claudeCmd := "claude --print"

		// Add model if specified
		if workflowData.EngineConfig != nil && workflowData.EngineConfig.Model != "" {
			claudeCmd += fmt.Sprintf(" --model %s", workflowData.EngineConfig.Model)
		}

		// Add max-turns if specified
		if workflowData.EngineConfig != nil && workflowData.EngineConfig.MaxTurns != "" {
			claudeCmd += fmt.Sprintf(" --max-turns %s", workflowData.EngineConfig.MaxTurns)
		}

		// Add MCP config if there are MCP servers
		if HasMCPServers(workflowData) {
			claudeCmd += " --mcp-config /tmp/gh-aw/mcp-config/mcp-servers.json"
		}

		// Add debug and verbose flags
		claudeCmd += " --debug --verbose"

		// Add permission mode for non-interactive execution
		claudeCmd += " --permission-mode bypassPermissions"

		// Add output format
		claudeCmd += " --output-format stream-json"

		// Add prompt from file
		claudeCmd += " \"$(cat /tmp/gh-aw/aw-prompts/prompt.txt)\""

		// Redirect output to log file
		claudeCmd += " 2>&1 | tee /tmp/gh-aw/logs/agent-execution.log"

		// Combine install and run
		fullCommand := installCmd + " && " + claudeCmd

		command = append(command, fullCommand)

	case "copilot":
		// For Copilot, we'll use sh -c to install and run in one command
		command = append(command, "sh", "-c")

		// Build install and run command
		installCmd := fmt.Sprintf("npm install -g @github/copilot@%s", engineVersion)

		// Build copilot CLI command with environment variable for instruction
		copilotCmd := "COPILOT_CLI_INSTRUCTION=$(cat /tmp/gh-aw/aw-prompts/prompt.txt)"

		// Build command arguments
		copilotCmd += " && copilot --add-dir /tmp/gh-aw/ --log-level all --log-dir " + copilotLogsFolder

		// Add model if specified
		if workflowData.EngineConfig != nil && workflowData.EngineConfig.Model != "" {
			copilotCmd += fmt.Sprintf(" --model %s", workflowData.EngineConfig.Model)
		}

		// Add tool permission arguments
		if workflowData.Tools != nil {
			// Build tool args similar to non-containerized mode
			// For simplicity, we'll allow shell by default in containerized mode
			copilotCmd += " --allow-tool shell"

			// Add edit tool if configured
			if _, hasEdit := workflowData.Tools["edit"]; hasEdit {
				copilotCmd += " --allow-tool write"
			}

			// Add github tool if configured
			if githubTool, hasGithub := workflowData.Tools["github"]; hasGithub {
				if githubConfig, ok := githubTool.(map[string]any); ok {
					if allowed, hasAllowed := githubConfig["allowed"]; hasAllowed {
						if allowedList, ok := allowed.([]any); ok {
							hasWildcard := false
							for _, item := range allowedList {
								if str, ok := item.(string); ok && str == "*" {
									hasWildcard = true
									break
								}
							}
							if hasWildcard {
								copilotCmd += " --allow-tool github"
							}
						}
					}
				}
			}
		}

		// Add cache-memory directory if configured
		if workflowData.CacheMemoryConfig != nil {
			copilotCmd += " --add-dir /tmp/gh-aw/cache-memory/"
		}

		// Add prompt
		copilotCmd += " --prompt \"$COPILOT_CLI_INSTRUCTION\""

		// Redirect output to log file
		copilotCmd += " 2>&1 | tee /tmp/gh-aw/logs/agent-execution.log"

		// Combine install and run
		fullCommand := installCmd + " && " + copilotCmd

		command = append(command, fullCommand)

	case "codex":
		// For Codex, we'll use sh -c to install and run in one command
		command = append(command, "sh", "-c")

		// Build install and run command
		installCmd := fmt.Sprintf("npm install -g @openai/codex@%s", engineVersion)

		// Build model parameter only if specified in engineConfig
		var modelParam string
		if workflowData.EngineConfig != nil && workflowData.EngineConfig.Model != "" {
			modelParam = fmt.Sprintf("-c model=%s ", workflowData.EngineConfig.Model)
		}

		// Build search parameter if web-search tool is present
		webSearchParam := ""
		if workflowData.Tools != nil {
			if _, hasWebSearch := workflowData.Tools["web-search"]; hasWebSearch {
				webSearchParam = "--search "
			}
		}

		// Full auto mode for non-interactive execution
		fullAutoParam := " --full-auto --skip-git-repo-check "

		// Build codex CLI command
		codexCmd := "mkdir -p /tmp/gh-aw/mcp-config/logs && INSTRUCTION=$(cat /tmp/gh-aw/aw-prompts/prompt.txt)"
		codexCmd += fmt.Sprintf(" && codex %sexec%s%s\"$INSTRUCTION\"", modelParam, webSearchParam, fullAutoParam)

		// Redirect output to log file
		codexCmd += " 2>&1 | tee /tmp/gh-aw/logs/agent-execution.log"

		// Combine install and run
		fullCommand := installCmd + " && " + codexCmd

		command = append(command, fullCommand)

	default:
		command = append(command, "sh", "-c", "echo 'Unknown engine' && exit 1")
	}

	return command
}
