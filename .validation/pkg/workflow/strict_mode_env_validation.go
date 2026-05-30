// This file contains strict mode environment secrets validation functions.
//
// It validates that secrets are not exposed through the env section of workflows
// compiled with the --strict flag, and emits warnings in non-strict mode.

package workflow

import (
	"fmt"
	"os"
	"strings"

	"github.com/github/gh-aw/pkg/console"
)

// validateEnvSecrets detects secrets in the top-level env section and the engine.env section,
// raising an error in strict mode or a warning in non-strict mode. Secrets in env will be
// leaked to the agent container.
//
// For engine.env, env vars whose key matches a known agentic engine env var (returned by the
// engine's GetRequiredSecretNames) are allowed to carry secrets – this enables users to
// override the engine's default secret with an org-specific one, e.g.
//
//	COPILOT_GITHUB_TOKEN: ${{ secrets.MY_ORG_COPILOT_TOKEN }}
//
// No other engine.env var is allowed to have secrets.
func (c *Compiler) validateEnvSecrets(frontmatter map[string]any) error {
	// Check top-level env section (no allowed overrides here)
	if err := c.validateEnvSecretsSection(frontmatter, "env", nil); err != nil {
		return err
	}

	// Check engine.env section when engine is in object format
	if engineValue, exists := frontmatter["engine"]; exists {
		if engineObj, ok := engineValue.(map[string]any); ok {
			// Determine which env var keys may carry secrets: those that the engine itself
			// requires (e.g. COPILOT_GITHUB_TOKEN for the copilot engine).
			// The second return value is *EngineConfig (not an error); we only need the engine ID.
			engineSetting, _ := c.ExtractEngineConfig(frontmatter)
			allowedEnvVarKeys := c.getEngineBaseEnvVarKeys(engineSetting)

			if err := c.validateEnvSecretsSection(engineObj, "engine.env", allowedEnvVarKeys); err != nil {
				return err
			}
		}
	}

	return nil
}

// getEngineBaseEnvVarKeys returns the set of env var key names that the named engine
// requires by default (using a minimal WorkflowData with no tools/MCP configured).
// These keys are allowed to carry secrets in engine.env overrides.
func (c *Compiler) getEngineBaseEnvVarKeys(engineID string) map[string]bool {
	if engineID == "" {
		return nil
	}
	engine, err := c.engineRegistry.GetEngine(engineID)
	if err != nil {
		strictModeValidationLog.Printf("Could not look up engine '%s' for env-key allowlist: %v", engineID, err)
		return nil
	}
	// Use a minimal WorkflowData so we get only the engine's unconditional secrets.
	// GetRequiredSecretNames only adds extra secrets when non-nil MCP tools (ParsedTools.GitHub,
	// ParsedTools.Playwright, etc.) are set, or when MCPScripts is populated. By passing empty
	// Tools/ParsedTools and no MCPScripts we get just the base engine secrets (e.g.
	// COPILOT_GITHUB_TOKEN, ANTHROPIC_API_KEY) without any optional/conditional ones.
	minimalData := &WorkflowData{
		Tools:       map[string]any{},
		ParsedTools: &ToolsConfig{},
	}
	keys := make(map[string]bool)
	for _, name := range engine.GetRequiredSecretNames(minimalData) {
		keys[name] = true
	}

	// Also include secrets declared in the AuthDefinition for inline engine definitions.
	// This allows workflows to pass auth secrets through engine.env without triggering the
	// "secret in env" strict-mode check.
	if def := c.engineCatalog.Get(engineID); def != nil && def.Provider.Auth != nil {
		for _, name := range def.Provider.Auth.RequiredSecretNames() {
			strictModeValidationLog.Printf("Adding auth-definition secret key to allowlist: %s", name)
			keys[name] = true
		}
	}

	return keys
}

// validateEnvSecretsSection checks a single config map's "env" key for secrets.
// sectionName is used in log and error messages (e.g. "env" or "engine.env").
// allowedEnvVarKeys is an optional set of env var key names whose secret values are
// permitted (used for engine.env to allow overriding engine env vars).
func (c *Compiler) validateEnvSecretsSection(config map[string]any, sectionName string, allowedEnvVarKeys map[string]bool) error {
	envValue, exists := config["env"]
	if !exists {
		strictModeValidationLog.Printf("No %s section found, validation passed", sectionName)
		return nil
	}

	// Check if env is a map[string]any
	envMap, ok := envValue.(map[string]any)
	if !ok {
		strictModeValidationLog.Printf("%s section is not a map, skipping validation", sectionName)
		return nil
	}

	// Convert to map[string]string for secret extraction, skipping keys whose secrets
	// are explicitly allowed (e.g. engine env var overrides in engine.env).
	envStrings := make(map[string]string)
	for key, value := range envMap {
		if allowedEnvVarKeys != nil && allowedEnvVarKeys[key] {
			strictModeValidationLog.Printf("Skipping allowed engine env var key in %s: %s", sectionName, key)
			continue
		}
		if strValue, ok := value.(string); ok {
			envStrings[key] = strValue
		}
	}

	// Extract secrets from env values
	secrets := ExtractSecretsFromMap(envStrings)
	if len(secrets) == 0 {
		strictModeValidationLog.Printf("No secrets found in %s section", sectionName)
		return nil
	}

	// Build list of secret references found
	var secretRefs []string
	for _, secretExpr := range secrets {
		secretRefs = append(secretRefs, secretExpr)
	}

	strictModeValidationLog.Printf("Found %d secret(s) in %s section: %v", len(secrets), sectionName, secretRefs)

	// In strict mode, this is an error
	if c.strictMode {
		return fmt.Errorf("strict mode: secrets detected in '%s' section will be leaked to the agent container. Found: %s. Use engine-specific secret configuration instead. See: https://github.github.com/gh-aw/reference/engines/", sectionName, strings.Join(secretRefs, ", "))
	}

	// In non-strict mode, emit a warning
	warningMsg := fmt.Sprintf("Warning: secrets detected in '%s' section will be leaked to the agent container. Found: %s. Consider using engine-specific secret configuration instead.", sectionName, strings.Join(secretRefs, ", "))
	fmt.Fprintln(os.Stderr, console.FormatWarningMessage(warningMsg))
	c.IncrementWarningCount()

	return nil
}
