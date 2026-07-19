// This file contains MCP (Model Context Protocol) validation functions.
// This file consolidates validation logic for MCP server configurations.
// Binary path utilities (GetBinaryPath, logAndValidateBinaryPath) live in mcp_helpers.go.

package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"slices"
	"strings"
	"time"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/parser"
	"github.com/github/gh-aw/pkg/workflow"
)

var mcpValidationLog = logger.New("cli:mcp_validation")

// validateServerSecrets checks if required environment variables/secrets are available
func validateServerSecrets(config parser.RegistryMCPServerConfig, verbose bool, useActionsSecrets bool) error {
	mcpValidationLog.Printf("Validating server secrets: server=%s, type=%s, useActionsSecrets=%v", config.Name, config.Type, useActionsSecrets)

	// Extract secrets from the config
	requiredSecrets := extractSecretsFromConfig(config)

	// Special case: Check for GH_AW_GITHUB_TOKEN when GitHub tool is in remote mode
	requiredSecrets = validateServerSecretsGitHubRemote(config, requiredSecrets)

	if len(requiredSecrets) == 0 {
		mcpValidationLog.Printf("No required secrets found, validating %d environment variables", len(config.Env))
		return validateServerSecretsEnv(config)
	}

	// Check availability of required secrets
	mcpValidationLog.Printf("Checking availability of %d required secrets", len(requiredSecrets))
	secretsStatus := checkSecretsAvailability(requiredSecrets, useActionsSecrets)

	// Separate secrets by availability
	availableSecrets, missingSecrets := validateServerSecretsSplit(secretsStatus)

	// Display information about secrets
	validateServerSecretsDisplayAvailable(availableSecrets, verbose)

	// Warn about missing secrets
	validateServerSecretsDisplayMissing(missingSecrets)

	mcpValidationLog.Printf("Secret validation completed: available=%d, missing=%d", len(availableSecrets), len(missingSecrets))
	return nil
}

func validateServerSecretsGitHubRemote(config parser.RegistryMCPServerConfig, requiredSecrets []SecretInfo) []SecretInfo {
	if config.Name != "github" || config.Type != "http" {
		return requiredSecrets
	}

	mcpValidationLog.Print("GitHub remote mode detected, checking for GH_AW_GITHUB_TOKEN")
	hasCustomToken := false
	for _, value := range config.Env {
		if strings.Contains(value, "secrets.") && !strings.Contains(value, "GH_AW_GITHUB_TOKEN") {
			hasCustomToken = true
			break
		}
	}
	if hasCustomToken || slices.ContainsFunc(requiredSecrets, func(secret SecretInfo) bool {
		return secret.Name == "GH_AW_GITHUB_TOKEN"
	}) {
		return requiredSecrets
	}
	return append(requiredSecrets, SecretInfo{Name: "GH_AW_GITHUB_TOKEN", EnvKey: "GITHUB_TOKEN"})
}

func validateServerSecretsEnv(config parser.RegistryMCPServerConfig) error {
	// No secrets required, proceed with normal env var validation
	for key, value := range config.Env {
		if strings.Contains(value, "${") {
			if err := validateServerSecretsEnvReference(config, key, value); err != nil {
				return err
			}
		} else if err := validateServerSecretsEnvDirect(config, key, value); err != nil {
			return err
		}
	}
	return nil
}

func validateServerSecretsEnvReference(config parser.RegistryMCPServerConfig, key string, value string) error {
	if strings.Contains(value, "secrets.") {
		return nil
	}
	if strings.Contains(value, "GH_TOKEN") || strings.Contains(value, "GITHUB_TOKEN") || strings.Contains(value, "GITHUB_PERSONAL_ACCESS_TOKEN") {
		if token, err := parser.GetGitHubToken(); err != nil {
			return errors.New("GitHub token not found in environment (set GH_TOKEN or GITHUB_TOKEN)")
		} else {
			config.Env[key] = token
		}
	}
	if strings.Contains(value, "GITHUB_TOKEN_REQUIRED") {
		if token, err := parser.GetGitHubToken(); err != nil {
			return fmt.Errorf("GitHub token required but not available: %w", err)
		} else {
			config.Env[key] = token
		}
	}
	return nil
}

func validateServerSecretsEnvDirect(config parser.RegistryMCPServerConfig, key string, value string) error {
	if value == "" {
		return fmt.Errorf("environment variable '%s' has empty value", key)
	}
	if strings.Contains(value, "GITHUB_TOKEN_REQUIRED") {
		if token, err := parser.GetGitHubToken(); err != nil {
			return fmt.Errorf("GitHub token required but not available: %w", err)
		} else {
			config.Env[key] = token
		}
	} else if key == "GITHUB_PERSONAL_ACCESS_TOKEN" || key == "GITHUB_TOKEN" || key == "GH_TOKEN" {
		if actualValue := os.Getenv(key); actualValue == "" { //nolint:osgetenvlibrary
			if token, err := parser.GetGitHubToken(); err == nil {
				config.Env[key] = token
			} else {
				return fmt.Errorf("GitHub token required for '%s' but not available: %w", key, err)
			}
		}
	} else if actualValue := os.Getenv(key); actualValue == "" { //nolint:osgetenvlibrary
		return fmt.Errorf("environment variable '%s' not set", key)
	}
	return nil
}

func validateServerSecretsSplit(secretsStatus []SecretInfo) ([]SecretInfo, []SecretInfo) {
	var availableSecrets []SecretInfo
	var missingSecrets []SecretInfo
	for _, secret := range secretsStatus {
		if secret.Available {
			availableSecrets = append(availableSecrets, secret)
		} else {
			missingSecrets = append(missingSecrets, secret)
		}
	}
	return availableSecrets, missingSecrets
}

func validateServerSecretsDisplayAvailable(availableSecrets []SecretInfo, verbose bool) {
	if !verbose || len(availableSecrets) == 0 {
		return
	}
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Found %d available secret(s):", len(availableSecrets))))
	for _, secret := range availableSecrets {
		source := "environment"
		if secret.Source == "actions" {
			source = "GitHub Actions"
		}
		fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("  ✓ %s (from %s)", secret.Name, source)))
	}
}

func validateServerSecretsDisplayMissing(missingSecrets []SecretInfo) {
	if len(missingSecrets) == 0 {
		return
	}
	mcpValidationLog.Printf("Found %d missing secrets", len(missingSecrets))
	fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("⚠️  %d required secret(s) not found:", len(missingSecrets))))
	for _, secret := range missingSecrets {
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage("  ✗ "+secret.Name))
	}
}

// validateMCPServerConfiguration validates that the CLI is properly configured
// by running the status command as a test.
// Diagnostics are emitted through the debug logger only.
func validateMCPServerConfiguration(ctx context.Context, cmdPath string, env []string) error {
	mcpValidationLog.Printf("Validating MCP server configuration: cmdPath=%s", cmdPath)

	// Determine, log, and validate the binary path only if --cmd flag is not provided
	// When --cmd is provided, the user explicitly specified the binary path to use
	if cmdPath == "" {
		// Attempt to detect the binary path and assign it to cmdPath
		// This ensures the validation uses the actual binary path instead of falling back to "gh aw"
		detectedPath, err := logAndValidateBinaryPath()
		if err == nil && detectedPath != "" {
			cmdPath = detectedPath
		}
	}

	// Try to run the status command to verify CLI is working
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var cmd *exec.Cmd
	if cmdPath != "" {
		mcpValidationLog.Printf("Using custom command path: %s", cmdPath)
		// Use custom command path
		cmd = exec.CommandContext(ctx, cmdPath, "status")
	} else {
		mcpValidationLog.Print("Using default gh aw command with proper token handling")
		// Use default gh aw command with proper token handling
		cmd = workflow.ExecGHContext(ctx, "aw", "status")
	}
	if env != nil {
		cmd.Env = append([]string(nil), env...)
	}
	output, err := runMCPSubprocessCombinedOutput(ctx, cmd)

	if err != nil {
		// Check for common error cases
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			mcpValidationLog.Print("Status command timed out")
			return errors.New("status command timed out - this may indicate a configuration issue")
		}

		mcpValidationLog.Printf("Status command failed: %v", err)

		// If the command failed, provide helpful error message
		if cmdPath != "" {
			return fmt.Errorf("failed to run status command with custom command '%s': %w\nOutput: %s\n\nPlease ensure:\n  - The command path is correct and executable\n  - You are in a git repository with .github/workflows directory", cmdPath, err, string(output))
		}
		return fmt.Errorf("failed to run status command: %w\nOutput: %s\n\nPlease ensure:\n  - gh CLI is installed and in PATH\n  - gh aw extension is installed (run: gh extension install github/gh-aw)\n  - You are in a git repository with .github/workflows directory", err, string(output))
	}

	// Status command succeeded - configuration is valid
	mcpValidationLog.Print("MCP server configuration validated successfully")
	return nil
}

// validateMCPWorkflowName validates that a workflow name exists in the repository.
// Returns nil if the workflow exists, or an error with suggestions if not.
// Empty workflow names are considered valid (means "all workflows").
//
// Note: Unlike ValidateWorkflowName in validators.go (which enforces strict format
// rules and rejects empty names), this MCP-specific function accepts empty names
// because in the MCP context an empty workflow name is a valid wildcard meaning
// "apply to all workflows". It also performs existence checks rather than format
// checks, delegating to workflow.ResolveWorkflowName and the live workflow list.
func validateMCPWorkflowName(workflowName string) error {
	// Empty workflow name means "all workflows" - this is valid in the MCP context
	if workflowName == "" {
		return nil
	}

	mcpLog.Printf("Validating workflow name: %s", workflowName)

	// Try to resolve as workflow ID first
	resolvedName, err := workflow.ResolveWorkflowName(workflowName)
	if err == nil {
		mcpLog.Printf("Workflow name resolved successfully: %s -> %s", workflowName, resolvedName)
		return nil
	}

	// Check if it's a valid GitHub Actions workflow name
	agenticWorkflowNames, nameErr := getAgenticWorkflowNames(false)
	if nameErr == nil && slices.Contains(agenticWorkflowNames, workflowName) {
		mcpLog.Printf("Workflow name is valid GitHub Actions workflow name: %s", workflowName)
		return nil
	}

	// Workflow not found - build error with suggestions
	mcpLog.Printf("Workflow name not found: %s", workflowName)

	suggestions := []string{
		"Use the 'status' tool to see all available workflows",
		"Check for typos in the workflow name",
		"Use the workflow ID (e.g., 'test-claude') or GitHub Actions workflow name (e.g., 'Test Claude')",
	}

	// Add fuzzy match suggestions
	similarNames := suggestWorkflowNames(workflowName)
	if len(similarNames) > 0 {
		suggestions = append([]string{fmt.Sprintf("Did you mean: %s?", strings.Join(similarNames, ", "))}, suggestions...)
	}

	return fmt.Errorf("workflow '%s' not found. %s", workflowName, strings.Join(suggestions, " "))
}
