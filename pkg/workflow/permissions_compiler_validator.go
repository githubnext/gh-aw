// This file contains the validatePermissions compiler validator, extracted from
// compiler_validators.go for single-responsibility maintenance per AGENTS.md.
//
// # Permission Validation
//
// validatePermissions orchestrates all permission-related checks in the compilation
// pipeline. It is called from validateWorkflowData (compiler_orchestrator.go) and
// returns the parsed *Permissions object for reuse in later validation steps.
//
// # Validation Sequence
//
// The following checks are performed in order:
//
//  1. Dangerous permissions — rejects globally-dangerous permission combinations.
//  2. GitHub App-only permissions — ensures a GitHub App is configured when
//     App-only permission scopes are requested.
//  3. GitHub MCP App write restriction — rejects "write" in
//     tools.github.github-app.permissions.
//  4. Unsupported github-app.permissions contexts — emits a warning when the
//     github-app.permissions field is used in a context that does not support it.
//  5. workflow_run branch restrictions — validates that workflow_run triggers carry
//     explicit branch filters to prevent untrusted-code execution.
//  6. GitHub MCP toolset permission alignment — validates that the workflow's
//     declared permissions cover the read/write requirements of all enabled toolsets.
//  7. id-token: write warning — emits a security reminder when OIDC tokens are
//     requested, because they can be used to authenticate to cloud providers.
//
// # Strict Mode
//
// When the Compiler is running in strict mode (c.strictMode == true), missing
// permissions for GitHub MCP toolsets are promoted to hard errors, except when all
// enabled toolsets are default-only toolsets (which are downgraded back to warnings
// to avoid blocking legacy workflows that relied on automatic permission injection).
package workflow

import (
	"fmt"
	"os"
)

// validatePermissions validates all permission-related configuration: dangerous
// permissions, GitHub App-only constraints, MCP app write restrictions, workflow_run
// branch security, GitHub MCP toolset permissions, and the id-token write warning.
// It returns the parsed *Permissions for reuse in subsequent validation steps.
func (c *Compiler) validatePermissions(workflowData *WorkflowData, markdownPath string) (*Permissions, error) {
	// Use the cached *Permissions object when available to avoid repeated YAML parsing.
	// CachedPermissions is populated by applyDefaults after all permission mutations are applied.
	// Fall back to parsing from the raw string for code paths that bypass applyDefaults
	// (e.g., tests that construct WorkflowData directly).
	var workflowPermissions *Permissions
	if workflowData.CachedPermissions != nil {
		workflowPermissions = workflowData.CachedPermissions
	} else {
		workflowPermissions = NewPermissionsParser(workflowData.Permissions).ToPermissions()
	}

	// Validate dangerous permissions
	log.Printf("Validating dangerous permissions")
	if err := validateDangerousPermissions(workflowData, workflowPermissions); err != nil {
		return nil, formatCompilerError(markdownPath, "error", err.Error(), err)
	}

	// Validate GitHub App-only permissions require a GitHub App to be configured
	log.Printf("Validating GitHub App-only permissions")
	if err := validateGitHubAppOnlyPermissions(workflowData, workflowPermissions); err != nil {
		return nil, formatCompilerError(markdownPath, "error", err.Error(), err)
	}

	// Validate tools.github.github-app.permissions does not use "write"
	log.Printf("Validating GitHub MCP app permissions (no write)")
	if err := validateGitHubMCPAppPermissionsNoWrite(workflowData); err != nil {
		return nil, formatCompilerError(markdownPath, "error", err.Error(), err)
	}

	// Warn when github-app.permissions is set in contexts that don't support it
	warnGitHubAppPermissionsUnsupportedContexts(workflowData)

	// Validate workflow_run triggers have branch restrictions
	log.Printf("Validating workflow_run triggers for branch restrictions")
	if err := c.validateWorkflowRunBranches(workflowData, markdownPath); err != nil {
		return nil, err
	}

	// Validate permissions against GitHub MCP toolsets
	log.Printf("Validating permissions for GitHub MCP toolsets")
	if workflowData.ParsedTools != nil && workflowData.ParsedTools.GitHub != nil {
		// Check if GitHub tool was explicitly configured in frontmatter
		// If permissions exist but tools.github was NOT explicitly configured,
		// skip validation and let the GitHub MCP server handle permission issues
		hasPermissions := workflowData.Permissions != ""

		log.Printf("Permission validation check: hasExplicitGitHubTool=%v, hasPermissions=%v",
			workflowData.HasExplicitGitHubTool, hasPermissions)

		// Skip validation if permissions exist but GitHub tool was auto-added (not explicit)
		if hasPermissions && !workflowData.HasExplicitGitHubTool {
			log.Printf("Skipping permission validation: permissions exist but tools.github not explicitly configured")
		} else {
			// Validate permissions using the typed GitHub tool configuration.
			// Pass the cached parsed toolsets from applyDefaults to avoid a redundant
			// ParseGitHubToolsets call inside ValidatePermissions.
			validationResult := ValidatePermissions(workflowPermissions, workflowData.ParsedTools.GitHub, workflowData.CachedParsedToolsets)

			if validationResult.HasValidationIssues {
				// Format the validation message
				message := FormatValidationMessage(validationResult, c.strictMode)

				if len(validationResult.MissingPermissions) > 0 {
					downgradeToWarning := c.strictMode && shouldDowngradeDefaultToolsetPermissionError(workflowData.ParsedTools.GitHub)
					if c.strictMode && !downgradeToWarning {
						// In strict mode, missing permissions are errors
						return nil, formatCompilerError(markdownPath, "error", message, nil)
					}

					if downgradeToWarning {
						message += "\n\n" + missingPermissionsDefaultToolsetWarning
					}

					// In non-strict mode, missing permissions are warnings.
					// In strict mode with default-only toolsets, this is intentionally downgraded to warning.
					fmt.Fprintln(os.Stderr, formatCompilerMessage(markdownPath, "warning", message))
					c.IncrementWarningCount()
				}
			}
		}
	}

	// Emit warning if id-token: write permission is detected
	log.Printf("Checking for id-token: write permission")
	if level, exists := workflowPermissions.Get(PermissionIdToken); exists && level == PermissionWrite {
		warningMsg := `This workflow grants id-token: write permission
OIDC tokens can authenticate to cloud providers (AWS, Azure, GCP).
Ensure proper audience validation and trust policies are configured.`
		fmt.Fprintln(os.Stderr, formatCompilerMessage(markdownPath, "warning", warningMsg))
		c.IncrementWarningCount()
	}

	return workflowPermissions, nil
}
