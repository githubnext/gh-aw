package workflow

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/github/gh-aw/pkg/console"
)

// validateExpressions checks expression safety and runtime-import file references
// embedded in the workflow's markdown content. It is the first validator called in
// validateWorkflowData and guards against unsafe GitHub Actions expressions.
func (c *Compiler) validateExpressions(workflowData *WorkflowData, markdownPath string) error {
	// Validate expression safety - check that all GitHub Actions expressions are in the allowed list
	if strings.Contains(workflowData.MarkdownContent, "${{") {
		log.Printf("Validating expression safety")
		if err := validateExpressionSafety(workflowData.MarkdownContent); err != nil {
			return formatCompilerError(markdownPath, "error", err.Error(), err)
		}
	}

	// Validate expressions in runtime-import files at compile time
	if strings.Contains(workflowData.MarkdownContent, "{{#runtime-import") {
		log.Printf("Validating runtime-import files")
		// Go up from .github/workflows/file.md to repo root
		workflowDir := filepath.Dir(markdownPath) // .github/workflows
		githubDir := filepath.Dir(workflowDir)    // .github
		workspaceDir := filepath.Dir(githubDir)   // repo root
		if err := validateRuntimeImportFiles(workflowData.MarkdownContent, workspaceDir); err != nil {
			return formatCompilerError(markdownPath, "error", err.Error(), err)
		}
	}

	return nil
}

// validateFeatureConfig validates feature flags declared in the workflow frontmatter
// and applies any action-mode override specified via the "action-mode" feature flag.
func (c *Compiler) validateFeatureConfig(workflowData *WorkflowData, markdownPath string) error {
	// Validate feature flags
	log.Printf("Validating feature flags")
	if err := validateFeatures(workflowData); err != nil {
		return formatCompilerError(markdownPath, "error", err.Error(), err)
	}

	// Check for action-mode feature flag override
	if workflowData.Features != nil {
		if actionModeVal, exists := workflowData.Features["action-mode"]; exists {
			if actionModeStr, ok := actionModeVal.(string); ok && actionModeStr != "" {
				mode := ActionMode(actionModeStr)
				if !mode.IsValid() {
					return formatCompilerError(markdownPath, "error", fmt.Sprintf("invalid action-mode feature flag '%s'. Must be 'dev', 'release', or 'script'", actionModeStr), nil)
				}
				log.Printf("Overriding action mode from feature flag: %s", mode)
				c.SetActionMode(mode)
			}
		}
	}

	return nil
}

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

// validateToolConfiguration validates safe-outputs settings, on.needs and safe-job
// declarations, network configuration, labels, concurrency expressions, sandbox
// security constraints, GitHub tool-to-toolset alignment, the agentic-workflows
// permission requirement, and dispatch/call-workflow configurations.
// workflowPermissions is the *Permissions value returned by validatePermissions.
func (c *Compiler) validateToolConfiguration(workflowData *WorkflowData, markdownPath string, workflowPermissions *Permissions) error {
	// Validate agent file exists if specified in engine config
	log.Printf("Validating agent file if specified")
	if err := c.validateAgentFile(workflowData, markdownPath); err != nil {
		// validateAgentFile always returns formatCompilerError results; pass through directly.
		return err
	}

	// Validate sandbox configuration
	log.Printf("Validating sandbox configuration")
	if err := validateSandboxConfig(workflowData); err != nil {
		return formatCompilerError(markdownPath, "error", err.Error(), err)
	}

	// Validate safe-outputs target configuration
	log.Printf("Validating safe-outputs target fields")
	if err := validateSafeOutputsTarget(workflowData.SafeOutputs); err != nil {
		return formatCompilerError(markdownPath, "error", err.Error(), err)
	}

	// Validate safe-outputs max configuration
	log.Printf("Validating safe-outputs max fields")
	if err := validateSafeOutputsMax(workflowData.SafeOutputs); err != nil {
		return formatCompilerError(markdownPath, "error", err.Error(), err)
	}

	// Validate safe-outputs allowed-domains configuration
	log.Printf("Validating safe-outputs allowed-domains")
	if err := c.validateSafeOutputsAllowedDomains(workflowData.SafeOutputs); err != nil {
		return formatCompilerError(markdownPath, "error", err.Error(), err)
	}

	// Validate safe-outputs merge-pull-request configuration
	log.Printf("Validating safe-outputs merge-pull-request")
	if err := validateSafeOutputsMergePullRequest(workflowData.SafeOutputs); err != nil {
		return formatCompilerError(markdownPath, "error", err.Error(), err)
	}

	// Validate safe-outputs needs declarations
	log.Printf("Validating safe-outputs needs declarations")
	if err := validateSafeOutputsNeeds(workflowData); err != nil {
		return formatCompilerError(markdownPath, "error", err.Error(), err)
	}

	// Validate on.needs declarations and on.github-app needs expressions
	log.Printf("Validating on.needs declarations")
	if err := c.validateOnNeeds(workflowData); err != nil {
		return formatCompilerError(markdownPath, "error", err.Error(), err)
	}

	// Validate safe-job needs: declarations against known generated job IDs
	log.Printf("Validating safe-job needs declarations")
	if err := validateSafeJobNeeds(workflowData); err != nil {
		return formatCompilerError(markdownPath, "error", err.Error(), err)
	}

	// Emit warnings for push-to-pull-request-branch misconfiguration
	log.Printf("Validating push-to-pull-request-branch configuration")
	c.validatePushToPullRequestBranchWarnings(workflowData.SafeOutputs, workflowData.CheckoutConfigs)

	// Validate network allowed domains configuration
	log.Printf("Validating network allowed domains")
	if err := c.validateNetworkAllowedDomains(workflowData.NetworkPermissions); err != nil {
		return formatCompilerError(markdownPath, "error", err.Error(), err)
	}

	// Validate network firewall configuration
	log.Printf("Validating network firewall configuration")
	if err := validateNetworkFirewallConfig(workflowData.NetworkPermissions); err != nil {
		return formatCompilerError(markdownPath, "error", err.Error(), err)
	}

	// Validate safe-outputs allow-workflows requires GitHub App
	log.Printf("Validating safe-outputs allow-workflows")
	if err := validateSafeOutputsAllowWorkflows(workflowData.SafeOutputs); err != nil {
		return formatCompilerError(markdownPath, "error", err.Error(), err)
	}

	// Validate labels configuration
	log.Printf("Validating labels")
	if err := validateLabels(workflowData); err != nil {
		return formatCompilerError(markdownPath, "error", err.Error(), err)
	}

	// Validate workflow-level concurrency group expression
	log.Printf("Validating workflow-level concurrency configuration")
	if workflowData.Concurrency != "" {
		// Use the cached validation result from applyDefaults to avoid re-running the
		// expensive ExpressionParser (regex + tokenize + parse) on every validateWorkflowData call.
		if workflowData.CachedConcurrencyGroupExprSet {
			if workflowData.CachedConcurrencyGroupExprErr != nil {
				return formatCompilerError(markdownPath, "error", "workflow-level concurrency validation failed: "+workflowData.CachedConcurrencyGroupExprErr.Error(), workflowData.CachedConcurrencyGroupExprErr)
			}
		} else {
			// Fallback: cache not populated (e.g. WorkflowData created without applyDefaults).
			// Extract the group expression directly from Concurrency YAML so validation is not
			// skipped for WorkflowData constructed outside of ParseWorkflowFile.
			groupExpr := extractConcurrencyGroupFromYAML(workflowData.Concurrency)
			if groupExpr != "" {
				if err := validateConcurrencyGroupExpression(groupExpr); err != nil {
					return formatCompilerError(markdownPath, "error", "workflow-level concurrency validation failed: "+err.Error(), err)
				}
			}
		}
	}

	// Validate concurrency.job-discriminator expression
	if workflowData.ConcurrencyJobDiscriminator != "" {
		if err := validateConcurrencyGroupExpression(workflowData.ConcurrencyJobDiscriminator); err != nil {
			return formatCompilerError(markdownPath, "error", "concurrency.job-discriminator validation failed: "+err.Error(), err)
		}
	}

	// Validate engine-level concurrency group expression
	log.Printf("Validating engine-level concurrency configuration")
	if workflowData.EngineConfig != nil && workflowData.EngineConfig.Concurrency != "" {
		// Extract the group expression from the engine concurrency YAML
		groupExpr := extractConcurrencyGroupFromYAML(workflowData.EngineConfig.Concurrency)
		if groupExpr != "" {
			if err := validateConcurrencyGroupExpression(groupExpr); err != nil {
				return formatCompilerError(markdownPath, "error", "engine.concurrency validation failed: "+err.Error(), err)
			}
		}
	}

	// Validate safe-outputs concurrency group expression
	if workflowData.SafeOutputs != nil && workflowData.SafeOutputs.ConcurrencyGroup != "" {
		if err := validateConcurrencyGroupExpression(workflowData.SafeOutputs.ConcurrencyGroup); err != nil {
			return formatCompilerError(markdownPath, "error", "safe-outputs.concurrency-group validation failed: "+err.Error(), err)
		}
	}

	// Warn when the user has specified custom workflow-level concurrency with cancel-in-progress: true
	// AND the workflow has the bot self-cancel risk combination (issue_comment triggers + GitHub App
	// safe-outputs). In this case the auto-generated bot-actor isolation cannot be applied because the
	// user's concurrency expression is preserved as-is. The user must add the bot-actor isolation
	// themselves (e.g. prepend `contains(github.actor, '[bot]') && github.run_id ||` to their group key).
	if workflowData.Concurrency != "" &&
		strings.Contains(workflowData.Concurrency, "cancel-in-progress: true") &&
		hasBotSelfCancelRisk(workflowData) {
		fmt.Fprintln(os.Stderr, formatCompilerMessage(markdownPath, "warning",
			"Custom workflow-level concurrency with cancel-in-progress: true may cause self-cancellation.\n"+
				"safe-outputs.github-app can post comments that re-trigger this workflow via issue_comment,\n"+
				"and those passive bot-authored runs can collide with the primary run's concurrency group.\n"+
				"Add `contains(github.actor, '[bot]') && github.run_id ||` at the start of your concurrency\n"+
				"group expression to route bot-triggered runs to a unique key and prevent self-cancellation.\n"+
				"See: https://gh.io/gh-aw/reference/concurrency for details."))
		c.IncrementWarningCount()
	}

	// Emit warning for sandbox.agent: false (disables agent sandbox firewall)
	if isAgentSandboxDisabled(workflowData) {
		fmt.Fprintln(os.Stderr, formatCompilerMessage(markdownPath, "warning",
			"Agent sandbox disabled (sandbox.agent: false). This removes firewall protection. "+
				"The AI agent will have direct network access without firewall filtering. "+
				"The MCP gateway remains enabled. Only use this for testing or in controlled "+
				"environments where you trust the AI agent completely."))
		c.IncrementWarningCount()
	}

	// Validate: threat detection requires sandbox.agent to be enabled (detection runs inside AWF)
	if workflowData.SafeOutputs != nil && workflowData.SafeOutputs.ThreatDetection != nil && isAgentSandboxDisabled(workflowData) {
		return formatCompilerError(markdownPath, "error", "threat detection requires sandbox.agent to be enabled. Threat detection runs inside the agent sandbox (AWF) with fully blocked network. Either enable sandbox.agent or use 'threat-detection: false' to disable the threat-detection configuration in safe-outputs.", errors.New("threat detection requires sandbox.agent"))
	}

	// Emit warning when assign-to-agent is used with github-app: but no explicit github-token:.
	// GitHub App tokens are rejected by the Copilot assignment API — a PAT is required.
	// The token fallback chain (GH_AW_AGENT_TOKEN || GH_AW_GITHUB_TOKEN || GITHUB_TOKEN) is used automatically.
	if workflowData.SafeOutputs != nil &&
		workflowData.SafeOutputs.AssignToAgent != nil &&
		workflowData.SafeOutputs.GitHubApp != nil &&
		workflowData.SafeOutputs.AssignToAgent.GitHubToken == "" {
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(
			"assign-to-agent does not support GitHub App tokens. "+
				"The Copilot assignment API requires a fine-grained PAT. "+
				"The token fallback chain (GH_AW_AGENT_TOKEN || GH_AW_GITHUB_TOKEN || GITHUB_TOKEN) will be used automatically. "+
				"Add github-token: to your assign-to-agent config to specify a different token."))
		c.IncrementWarningCount()
	}

	// Emit experimental warning for rate-limit feature
	if workflowData.RateLimit != nil {
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage("Using experimental feature: rate-limit"))
		c.IncrementWarningCount()
	}

	// Emit experimental warning for dispatch_repository feature
	if workflowData.SafeOutputs != nil && workflowData.SafeOutputs.DispatchRepository != nil {
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage("Using experimental feature: dispatch_repository"))
		c.IncrementWarningCount()
	}

	// Emit experimental warning for merge-pull-request feature
	if workflowData.SafeOutputs != nil && workflowData.SafeOutputs.MergePullRequest != nil {
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage("Using experimental feature: merge-pull-request"))
		c.IncrementWarningCount()
	}

	// Warn when slash_command and bots are both configured: if a bot listed in bots: posts
	// a comment that starts with the slash command text (e.g. /command-name), the
	// check_command_position check will pass and the bot will trigger the workflow —
	// occupying the concurrency slot and potentially blocking a simultaneous manual invocation.
	if len(workflowData.Command) > 0 && len(workflowData.Bots) > 0 {
		fmt.Fprintln(os.Stderr, formatCompilerMessage(markdownPath, "warning",
			"Both slash_command and bots triggers are configured. If a bot listed in bots: "+
				"posts a comment that starts with the slash command text (e.g., /command-name), "+
				"it will trigger the workflow and occupy the concurrency slot, potentially "+
				"blocking simultaneous manual invocations. To ensure the workflow only runs on "+
				"explicit user commands, remove the 'bots:' field."))
		c.IncrementWarningCount()
	}

	// Inform users when this workflow is a redirect stub for updates.
	if workflowData.Redirect != "" {
		fmt.Fprintln(os.Stderr, formatCompilerMessage(markdownPath, "info",
			"workflow redirect configured: updates move to "+workflowData.Redirect))
	}

	// Validate GitHub tools against enabled toolsets
	log.Printf("Validating GitHub tools against enabled toolsets")
	if workflowData.ParsedTools != nil && workflowData.ParsedTools.GitHub != nil {
		// Extract allowed tools and reuse the cached parsed toolsets from applyDefaults to
		// avoid a redundant ParseGitHubToolsets call on every validateWorkflowData iteration.
		allowedTools := workflowData.ParsedTools.GitHub.Allowed.ToStringSlice()
		var enabledToolsets []string
		if workflowData.CachedParsedToolsets != nil {
			enabledToolsets = workflowData.CachedParsedToolsets
		} else {
			enabledToolsets = ParseGitHubToolsets(strings.Join(workflowData.ParsedTools.GitHub.Toolset.ToStringSlice(), ","))
		}

		// Validate that all allowed tools have their toolsets enabled
		if err := ValidateGitHubToolsAgainstToolsets(allowedTools, enabledToolsets); err != nil {
			return formatCompilerError(markdownPath, "error", err.Error(), err)
		}

		// Print informational message if "projects" toolset is explicitly specified
		// (not when implied by "all", as users unlikely intend to use projects with "all")
		originalToolsets := workflowData.ParsedTools.GitHub.Toolset.ToStringSlice()
		if slices.Contains(originalToolsets, "projects") {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage("The 'projects' toolset requires additional authentication."))
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage("See: https://github.github.com/gh-aw/reference/auth-projects/"))
		}
	}

	// Validate permissions for agentic-workflows tool
	log.Printf("Validating permissions for agentic-workflows tool")
	if _, hasAgenticWorkflows := workflowData.Tools["agentic-workflows"]; hasAgenticWorkflows {
		// Check if actions: read permission exists
		actionsLevel, hasActions := workflowPermissions.Get(PermissionActions)
		if !hasActions || actionsLevel == PermissionNone {
			// Missing actions: read permission
			message := "ERROR: Missing required permission for agentic-workflows tool:\n"
			message += "  - actions: read\n\n"
			message += "The agentic-workflows tool requires actions: read permission to access GitHub Actions data.\n\n"
			message += "Suggested fix: Add the following to your workflow frontmatter:\n"
			message += "permissions:\n"
			message += "  actions: read"

			return formatCompilerError(markdownPath, "error", message, nil)
		}
	}

	// Validate resources field — GitHub Actions expression syntax is not allowed.
	log.Printf("Validating resources field")
	if workflowData.ParsedFrontmatter != nil {
		for _, r := range workflowData.ParsedFrontmatter.Resources {
			if strings.Contains(r, "${{") {
				return formatCompilerError(markdownPath, "error",
					fmt.Sprintf("resources entry %q contains GitHub Actions expression syntax (${{) which is not allowed; use static paths only", r), nil)
			}
		}
	}

	// Validate dispatch-workflow configuration (independent of agentic-workflows tool)
	log.Print("Validating dispatch-workflow configuration")
	if err := c.validateDispatchWorkflow(workflowData, markdownPath); err != nil {
		return formatCompilerError(markdownPath, "error", fmt.Sprintf("dispatch-workflow validation failed: %v", err), err)
	}

	// Validate dispatch_repository configuration (independent of agentic-workflows tool)
	log.Print("Validating dispatch_repository configuration")
	if err := c.validateDispatchRepository(workflowData, markdownPath); err != nil {
		return formatCompilerError(markdownPath, "error", fmt.Sprintf("dispatch_repository validation failed: %v", err), err)
	}

	// Validate call-workflow configuration (independent of agentic-workflows tool)
	log.Print("Validating call-workflow configuration")
	if err := c.validateCallWorkflow(workflowData, markdownPath); err != nil {
		return formatCompilerError(markdownPath, "error", fmt.Sprintf("call-workflow validation failed: %v", err), err)
	}

	return nil
}
