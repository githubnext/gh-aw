package workflow

import (
	"fmt"
	"regexp"
	"strings"
)

// CollectSecretsPattern generates a regex pattern that matches all secrets used in the workflow
// This includes engine-specific secrets and MCP server secrets
func CollectSecretsPattern(workflowData *WorkflowData, engine CodingAgentEngine) string {
	secretPatterns := make([]string, 0)

	// Add engine-specific secrets based on engine type
	engineID := engine.GetID()
	switch engineID {
	case "claude":
		// Match Anthropic API keys
		// Format: sk-ant-... (actual format from Anthropic)
		secretPatterns = append(secretPatterns, `sk-ant-[a-zA-Z0-9_-]{95,}`)
	case "copilot":
		// Match GitHub PATs (Personal Access Tokens)
		// Formats: ghp_... (classic), github_pat_... (fine-grained)
		secretPatterns = append(secretPatterns, `ghp_[a-zA-Z0-9]{36,}`)
		secretPatterns = append(secretPatterns, `github_pat_[a-zA-Z0-9_]{82,}`)
	case "codex":
		// Match OpenAI API keys
		// Format: sk-... or sk-proj-...
		secretPatterns = append(secretPatterns, `sk-[a-zA-Z0-9]{32,}`)
		secretPatterns = append(secretPatterns, `sk-proj-[a-zA-Z0-9]{20,}`)
	}

	// Add generic GitHub token patterns that might be in MCP configs
	// These are catch-all patterns for any GitHub tokens
	secretPatterns = append(secretPatterns, `ghs_[a-zA-Z0-9]{36,}`) // GitHub App installation token
	secretPatterns = append(secretPatterns, `gho_[a-zA-Z0-9]{36,}`) // GitHub OAuth token
	secretPatterns = append(secretPatterns, `ghu_[a-zA-Z0-9]{36,}`) // GitHub user-to-server token

	// TODO: Add MCP server-specific secret patterns if they're documented
	// This would require parsing MCP configurations to find environment variable patterns

	// Combine all patterns into a single alternation regex
	if len(secretPatterns) == 0 {
		return ""
	}

	// Create a single regex pattern with alternation
	// We escape any special regex characters in the patterns (though our patterns don't have them)
	combinedPattern := strings.Join(secretPatterns, "|")

	return combinedPattern
}

// generateSecretRedactionStep generates a workflow step that redacts secrets from files in /tmp
func (c *Compiler) generateSecretRedactionStep(yaml *strings.Builder, workflowData *WorkflowData, engine CodingAgentEngine) {
	// Get the combined secrets pattern
	secretsPattern := CollectSecretsPattern(workflowData, engine)

	// If no secrets pattern, skip the redaction step
	if secretsPattern == "" {
		return
	}

	yaml.WriteString("      - name: Redact secrets from files in /tmp\n")
	yaml.WriteString("        if: always()\n")
	yaml.WriteString("        uses: actions/github-script@v8\n")
	yaml.WriteString("        with:\n")
	yaml.WriteString("          script: |\n")

	// Use the embedded JavaScript code
	jsCode := redactSecretsScript

	// Indent the JavaScript code properly for YAML
	lines := strings.Split(jsCode, "\n")
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			yaml.WriteString("            " + line + "\n")
		} else {
			yaml.WriteString("\n")
		}
	}

	// Add environment variable with the secrets pattern
	yaml.WriteString("        env:\n")

	// Escape the pattern for YAML (single quotes to avoid interpretation)
	// We need to be careful with special characters in regex patterns
	escapedPattern := strings.ReplaceAll(secretsPattern, "'", "''")
	yaml.WriteString(fmt.Sprintf("          GITHUB_AW_SECRETS_PATTERN: '%s'\n", escapedPattern))
}

// validateSecretsPattern validates that the generated secrets pattern is a valid regex
func validateSecretsPattern(pattern string) error {
	if pattern == "" {
		return nil
	}

	_, err := regexp.Compile(pattern)
	if err != nil {
		return fmt.Errorf("invalid secrets pattern regex: %w", err)
	}

	return nil
}
