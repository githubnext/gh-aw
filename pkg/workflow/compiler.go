package workflow

import (
	_ "embed"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/stringutil"
	"go.yaml.in/yaml/v3"
)

var log = logger.New("workflow:compiler")

const (
	// MaxLockFileSize is the maximum allowed size for generated lock workflow files (500KB)
	MaxLockFileSize = 512000 // 500KB in bytes

	// MaxExpressionSize is the maximum allowed size for GitHub Actions expression values (21KB)
	// This includes environment variable values, if conditions, and other expression contexts
	// See: https://docs.github.com/en/actions/learn-github-actions/usage-limits-billing-and-administration
	MaxExpressionSize = 21000 // 21KB in bytes

	// MaxPromptChunkSize is the maximum size for each chunk when splitting prompt text (20KB)
	// This limit ensures each heredoc block stays under GitHub Actions step size limits (21KB)
	MaxPromptChunkSize = 20000 // 20KB limit for each chunk

	// MaxPromptChunks is the maximum number of chunks allowed when splitting prompt text
	// This prevents excessive step generation for extremely large prompt texts
	MaxPromptChunks = 5 // Maximum number of chunks
)

//go:embed schemas/github-workflow.json
var githubWorkflowSchema string

// formatCompilerMessage creates a formatted compiler message string (for warnings printed to stderr)
// filePath: the file path to include in the message (typically markdownPath or lockFile)
// msgType: the message type ("error" or "warning")
// message: the message text
func formatCompilerMessage(filePath string, msgType string, message string) string {
	return console.FormatError(console.CompilerError{
		Position: console.ErrorPosition{
			File:   filePath,
			Line:   0,
			Column: 0,
		},
		Type:    msgType,
		Message: message,
	})
}

// CompileWorkflow compiles a workflow markdown file into a GitHub Actions YAML file.
// It reads the file from disk, parses frontmatter and markdown sections, and generates
// the corresponding workflow YAML. Returns the compiled workflow data or an error.
//
// The compilation process includes:
//   - Reading and parsing the markdown file
//   - Extracting frontmatter configuration
//   - Validating workflow configuration
//   - Generating GitHub Actions YAML
//   - Writing the compiled workflow to a .lock.yml file
//
// This is the main entry point for compiling workflows from disk. For compiling
// pre-parsed workflow data, use CompileWorkflowData instead.
func (c *Compiler) CompileWorkflow(markdownPath string) error {
	// Store markdownPath for use in dynamic tool generation
	c.markdownPath = markdownPath

	// Parse the markdown file
	log.Printf("Parsing workflow file")
	workflowData, err := c.ParseWorkflowFile(markdownPath)
	if err != nil {
		// ParseWorkflowFile already returns formatted compiler errors; pass them through.
		if isFormattedCompilerError(err) {
			return err
		}
		// Fallback for any unformatted error that slipped through.
		return formatCompilerError(markdownPath, "error", err.Error(), err)
	}

	return c.CompileWorkflowData(workflowData, markdownPath)
}

// validateWorkflowData performs comprehensive validation of workflow configuration
// including expressions, features, permissions, and configurations.
func (c *Compiler) validateWorkflowData(workflowData *WorkflowData, markdownPath string) error {
	// Validate expression safety - check that all GitHub Actions expressions are in the allowed list
	log.Printf("Validating expression safety")
	if err := validateExpressionSafety(workflowData.MarkdownContent); err != nil {
		return formatCompilerError(markdownPath, "error", err.Error(), err)
	}

	// Validate expressions in runtime-import files at compile time
	log.Printf("Validating runtime-import files")
	// Go up from .github/workflows/file.md to repo root
	workflowDir := filepath.Dir(markdownPath) // .github/workflows
	githubDir := filepath.Dir(workflowDir)    // .github
	workspaceDir := filepath.Dir(githubDir)   // repo root
	if err := validateRuntimeImportFiles(workflowData.MarkdownContent, workspaceDir); err != nil {
		return formatCompilerError(markdownPath, "error", err.Error(), err)
	}

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

	// Validate dangerous permissions
	log.Printf("Validating dangerous permissions")
	if err := validateDangerousPermissions(workflowData); err != nil {
		return formatCompilerError(markdownPath, "error", err.Error(), err)
	}

	// Validate GitHub App-only permissions require a GitHub App to be configured
	log.Printf("Validating GitHub App-only permissions")
	if err := validateGitHubAppOnlyPermissions(workflowData); err != nil {
		return formatCompilerError(markdownPath, "error", err.Error(), err)
	}

	// Validate agent file exists if specified in engine config
	log.Printf("Validating agent file if specified")
	if err := c.validateAgentFile(workflowData, markdownPath); err != nil {
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

	// Validate safe-outputs allowed-domains configuration
	log.Printf("Validating safe-outputs allowed-domains")
	if err := c.validateSafeOutputsAllowedDomains(workflowData.SafeOutputs); err != nil {
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

	// Validate labels configuration
	log.Printf("Validating labels")
	if err := validateLabels(workflowData); err != nil {
		return formatCompilerError(markdownPath, "error", err.Error(), err)
	}

	// Validate workflow-level concurrency group expression
	log.Printf("Validating workflow-level concurrency configuration")
	if workflowData.Concurrency != "" {
		// Extract the group expression from the concurrency YAML
		// The Concurrency field contains the full YAML (e.g., "concurrency:\n  group: \"...\"")
		// We need to extract just the group value
		groupExpr := extractConcurrencyGroupFromYAML(workflowData.Concurrency)
		if groupExpr != "" {
			if err := validateConcurrencyGroupExpression(groupExpr); err != nil {
				return formatCompilerError(markdownPath, "error", "workflow-level concurrency validation failed: "+err.Error(), err)
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
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage("⚠️  WARNING: Agent sandbox disabled (sandbox.agent: false). This removes firewall protection. The AI agent will have direct network access without firewall filtering. The MCP gateway remains enabled. Only use this for testing or in controlled environments where you trust the AI agent completely."))
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

	// Emit experimental warning for qmd documentation search feature
	if workflowData.QmdConfig != nil {
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage("Using experimental feature: qmd"))
		c.IncrementWarningCount()
	}

	// Emit experimental warning for dependencies (APM) feature
	if workflowData.APMDependencies != nil && len(workflowData.APMDependencies.Packages) > 0 {
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage("Using experimental feature: dependencies (APM)"))
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

	// Validate workflow_run triggers have branch restrictions
	log.Printf("Validating workflow_run triggers for branch restrictions")
	if err := c.validateWorkflowRunBranches(workflowData, markdownPath); err != nil {
		return err
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
			// Parse permissions from the workflow data
			// WorkflowData.Permissions contains the raw YAML string (including "permissions:" prefix)
			permissions := NewPermissionsParser(workflowData.Permissions).ToPermissions()

			// Validate permissions using the typed GitHub tool configuration
			validationResult := ValidatePermissions(permissions, workflowData.ParsedTools.GitHub)

			if validationResult.HasValidationIssues {
				// Format the validation message
				message := FormatValidationMessage(validationResult, c.strictMode)

				if len(validationResult.MissingPermissions) > 0 {
					if c.strictMode {
						// In strict mode, missing permissions are errors
						return formatCompilerError(markdownPath, "error", message, nil)
					} else {
						// In non-strict mode, missing permissions are warnings
						fmt.Fprintln(os.Stderr, formatCompilerMessage(markdownPath, "warning", message))
						c.IncrementWarningCount()
					}
				}
			}
		}
	}

	// Emit warning if id-token: write permission is detected
	log.Printf("Checking for id-token: write permission")
	if workflowData.Permissions != "" {
		permissions := NewPermissionsParser(workflowData.Permissions).ToPermissions()
		if permissions != nil {
			level, exists := permissions.Get(PermissionIdToken)
			if exists && level == PermissionWrite {
				warningMsg := `This workflow grants id-token: write permission
OIDC tokens can authenticate to cloud providers (AWS, Azure, GCP).
Ensure proper audience validation and trust policies are configured.`
				fmt.Fprintln(os.Stderr, formatCompilerMessage(markdownPath, "warning", warningMsg))
				c.IncrementWarningCount()
			}
		}
	}

	// Validate GitHub tools against enabled toolsets
	log.Printf("Validating GitHub tools against enabled toolsets")
	if workflowData.ParsedTools != nil && workflowData.ParsedTools.GitHub != nil {
		// Extract allowed tools and enabled toolsets from ParsedTools
		allowedTools := workflowData.ParsedTools.GitHub.Allowed.ToStringSlice()
		enabledToolsets := ParseGitHubToolsets(strings.Join(workflowData.ParsedTools.GitHub.Toolset.ToStringSlice(), ","))

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
		// Parse permissions from the workflow data
		permissions := NewPermissionsParser(workflowData.Permissions).ToPermissions()

		// Check if actions: read permission exists
		actionsLevel, hasActions := permissions.Get(PermissionActions)
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

// generateAndValidateYAML generates GitHub Actions YAML and validates
// the output size and format.
func (c *Compiler) generateAndValidateYAML(workflowData *WorkflowData, markdownPath string, lockFile string) (string, error) {
	// Generate the YAML content
	yamlContent, err := c.generateYAML(workflowData, markdownPath)
	if err != nil {
		return "", formatCompilerError(markdownPath, "error", fmt.Sprintf("failed to generate YAML: %v", err), err)
	}

	// Always validate expression sizes - this is a hard limit from GitHub Actions (21KB)
	// that cannot be bypassed, so we validate it unconditionally
	log.Print("Validating expression sizes")
	if err := c.validateExpressionSizes(yamlContent); err != nil {
		// Store error first so we can write invalid YAML before returning
		formattedErr := formatCompilerError(markdownPath, "error", fmt.Sprintf("expression size validation failed: %v", err), err)
		// Write the invalid YAML to a .invalid.yml file for inspection
		invalidFile := strings.TrimSuffix(lockFile, ".lock.yml") + ".invalid.yml"
		if writeErr := os.WriteFile(invalidFile, []byte(yamlContent), 0644); writeErr == nil {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage("Invalid workflow YAML written to: "+console.ToRelativePath(invalidFile)))
		}
		return "", formattedErr
	}

	// Template injection validation and GitHub Actions schema validation both require a
	// parsed representation of the compiled YAML.  Parse it once here and share the
	// result between the two validators to avoid redundant yaml.Unmarshal calls.
	//
	// Fast-path: if the YAML contains no unsafe context expressions we can skip the
	// parse (and template-injection check) entirely for the common case.
	needsTemplateCheck := unsafeContextRegex.MatchString(yamlContent)
	needsSchemaCheck := !c.skipValidation

	var parsedWorkflow map[string]any
	if needsTemplateCheck || needsSchemaCheck {
		log.Print("Parsing compiled YAML for validation")
		if parseErr := yaml.Unmarshal([]byte(yamlContent), &parsedWorkflow); parseErr != nil {
			// If parsing fails here the subsequent validators would also fail; keep going
			// so we surface the root error from the right validator.
			parsedWorkflow = nil
		}
	}

	// Validate for template injection vulnerabilities - detect unsafe expression usage in run: commands
	if needsTemplateCheck {
		log.Print("Validating for template injection vulnerabilities")
		var templateErr error
		if parsedWorkflow != nil {
			templateErr = validateNoTemplateInjectionFromParsed(parsedWorkflow)
		} else {
			templateErr = validateNoTemplateInjection(yamlContent)
		}
		if templateErr != nil {
			// Store error first so we can write invalid YAML before returning
			formattedErr := formatCompilerError(markdownPath, "error", templateErr.Error(), templateErr)
			// Write the invalid YAML to a .invalid.yml file for inspection
			invalidFile := strings.TrimSuffix(lockFile, ".lock.yml") + ".invalid.yml"
			if writeErr := os.WriteFile(invalidFile, []byte(yamlContent), 0644); writeErr == nil {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage("Workflow with template injection risks written to: "+console.ToRelativePath(invalidFile)))
			}
			return "", formattedErr
		}
	}

	// Validate against GitHub Actions schema (unless skipped)
	if needsSchemaCheck {
		log.Print("Validating workflow against GitHub Actions schema")
		var schemaErr error
		if parsedWorkflow != nil {
			schemaErr = c.validateGitHubActionsSchemaFromParsed(parsedWorkflow)
		} else {
			schemaErr = c.validateGitHubActionsSchema(yamlContent)
		}
		if schemaErr != nil {
			// Try to point at the exact line of the failing field in the source markdown.
			// extractSchemaErrorField unwraps the error chain to find the top-level field
			// name (e.g. "timeout-minutes"), which findFrontmatterFieldLine then locates in
			// the source frontmatter so the error is IDE-navigable.
			fieldLine := 1
			if fieldName := extractSchemaErrorField(schemaErr); fieldName != "" {
				frontmatterLines := strings.Split(workflowData.FrontmatterYAML, "\n")
				if line := findFrontmatterFieldLine(frontmatterLines, 2, fieldName); line > 0 {
					fieldLine = line
				}
			}
			// Store error first so we can write invalid YAML before returning
			formattedErr := formatCompilerErrorWithPosition(markdownPath, fieldLine, 1, "error",
				fmt.Sprintf("invalid workflow: %v", schemaErr), schemaErr)
			// Write the invalid YAML to a .invalid.yml file for inspection
			invalidFile := strings.TrimSuffix(lockFile, ".lock.yml") + ".invalid.yml"
			if writeErr := os.WriteFile(invalidFile, []byte(yamlContent), 0644); writeErr == nil {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage("Invalid workflow YAML written to: "+console.ToRelativePath(invalidFile)))
			}
			return "", formattedErr
		}

		// Validate container images used in MCP configurations
		log.Print("Validating container images")
		if err := c.validateContainerImages(workflowData); err != nil {
			// Treat container image validation failures as warnings, not errors
			// This is because validation may fail due to auth issues locally (e.g., private registries)
			fmt.Fprintln(os.Stderr, formatCompilerMessage(markdownPath, "warning", fmt.Sprintf("container image validation failed: %v", err)))
			c.IncrementWarningCount()
		}

		// Validate runtime packages (npx, uv)
		log.Print("Validating runtime packages")
		if err := c.validateRuntimePackages(workflowData); err != nil {
			return "", formatCompilerError(markdownPath, "error", fmt.Sprintf("runtime package validation failed: %v", err), err)
		}

		// Validate firewall configuration (log-level enum)
		log.Print("Validating firewall configuration")
		if err := c.validateFirewallConfig(workflowData); err != nil {
			return "", formatCompilerError(markdownPath, "error", fmt.Sprintf("firewall configuration validation failed: %v", err), err)
		}

		// Validate repository features (discussions, issues)
		log.Print("Validating repository features")
		if err := c.validateRepositoryFeatures(workflowData); err != nil {
			return "", formatCompilerError(markdownPath, "error", fmt.Sprintf("repository feature validation failed: %v", err), err)
		}
	} else if c.verbose {
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage("Schema validation available but skipped (use SetSkipValidation(false) to enable)"))
		c.IncrementWarningCount()
	}

	return yamlContent, nil
}

// writeWorkflowOutput writes the compiled workflow to the lock file
// and handles console output formatting.
func (c *Compiler) writeWorkflowOutput(lockFile, yamlContent string, markdownPath string) error {
	// Write to lock file (unless noEmit is enabled)
	if c.noEmit {
		log.Print("Validation completed - no lock file generated (--no-emit enabled)")
	} else {
		log.Printf("Writing output to: %s", lockFile)

		// Check if content has actually changed
		contentUnchanged := false
		if existingContent, err := os.ReadFile(lockFile); err == nil {
			if string(existingContent) == yamlContent {
				// Content is identical - skip write to preserve timestamp
				contentUnchanged = true
				log.Print("Lock file content unchanged - skipping write to preserve timestamp")
			}
		}

		// Only write if content has changed
		if !contentUnchanged {
			if err := os.WriteFile(lockFile, []byte(yamlContent), 0644); err != nil {
				return formatCompilerError(lockFile, "error", fmt.Sprintf("failed to write lock file: %v", err), err)
			}
			log.Print("Lock file written successfully")
		}

		// Validate file size after writing
		if lockFileInfo, err := os.Stat(lockFile); err == nil {
			if lockFileInfo.Size() > MaxLockFileSize {
				lockSize := console.FormatFileSize(lockFileInfo.Size())
				maxSize := console.FormatFileSize(MaxLockFileSize)
				warningMsg := fmt.Sprintf("Generated lock file size (%s) exceeds recommended maximum size (%s)", lockSize, maxSize)
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(warningMsg))
			}
		}
	}

	// Display success message with file size if we generated a lock file (unless quiet mode)
	if !c.quiet {
		if c.noEmit {
			fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(console.ToRelativePath(markdownPath)))
		} else {
			// Get the size of the generated lock file for display
			if lockFileInfo, err := os.Stat(lockFile); err == nil {
				lockSize := console.FormatFileSize(lockFileInfo.Size())
				fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("%s (%s)", console.ToRelativePath(markdownPath), lockSize)))
			} else {
				// Fallback to original display if we can't get file info
				fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(console.ToRelativePath(markdownPath)))
			}
		}
	}
	return nil
}

// CompileWorkflowData compiles pre-parsed workflow content into GitHub Actions YAML.
// Unlike CompileWorkflow, this accepts already-parsed frontmatter and markdown content
// rather than reading from disk. This is useful for testing and programmatic workflow generation.
//
// The compilation process includes:
//   - Validating workflow configuration and features
//   - Checking permissions and tool configurations
//   - Generating GitHub Actions YAML structure
//   - Writing the compiled workflow to a .lock.yml file
//
// This function avoids re-parsing when workflow data has already been extracted,
// making it efficient for scenarios where the same workflow is compiled multiple times
// or when workflow data comes from a non-file source.
func (c *Compiler) CompileWorkflowData(workflowData *WorkflowData, markdownPath string) error {
	// Store markdownPath for use in dynamic tool generation and prompt generation
	c.markdownPath = markdownPath

	// Track compilation time for performance monitoring
	startTime := time.Now()
	defer func() {
		log.Printf("Compilation completed in %v", time.Since(startTime))
	}()

	// Reset the step order tracker for this compilation
	c.stepOrderTracker = NewStepOrderTracker()

	// Reset schedule friendly formats for this compilation
	c.scheduleFriendlyFormats = nil

	// Reset the artifact manager for this compilation
	if c.artifactManager == nil {
		c.artifactManager = NewArtifactManager()
	} else {
		c.artifactManager.Reset()
	}

	// Generate lock file name
	lockFile := stringutil.MarkdownToLockFile(markdownPath)

	// Sanitize the lock file path to prevent path traversal attacks
	lockFile = filepath.Clean(lockFile)

	log.Printf("Starting compilation: %s -> %s", markdownPath, lockFile)

	// Validate workflow data
	if err := c.validateWorkflowData(workflowData, markdownPath); err != nil {
		return err
	}

	// Note: Markdown content size is now handled by splitting into multiple steps in generatePrompt
	log.Printf("Workflow: %s, Tools: %d", workflowData.Name, len(workflowData.Tools))

	// Note: compute-text functionality is now inlined directly in the task job
	// instead of using a shared action file

	// Generate and validate YAML
	yamlContent, err := c.generateAndValidateYAML(workflowData, markdownPath, lockFile)
	if err != nil {
		return err
	}

	// Write output
	return c.writeWorkflowOutput(lockFile, yamlContent, markdownPath)
}

// ParseWorkflowFile parses a markdown workflow file and extracts all necessary data

// extractTopLevelYAMLSection extracts a top-level YAML section from the frontmatter map
// This ensures we only extract keys at the root level, avoiding nested keys with the same name
// parseOnSection parses the "on" section from frontmatter to extract command triggers, reactions, and other events

// generateYAML generates the complete GitHub Actions YAML content

// isActivationJobNeeded determines if the activation job is required
// generateMainJobSteps generates the steps section for the main job

// The original JavaScript code will use the pattern as-is with "g" flags

// validateMarkdownSizeForGitHubActions is no longer used - content is now split into multiple steps
// to handle GitHub Actions script size limits automatically
// func (c *Compiler) validateMarkdownSizeForGitHubActions(content string) error { ... }

// splitContentIntoChunks splits markdown content into chunks that fit within GitHub Actions script size limits

// generatePostSteps generates the post-steps section that runs after AI execution

// convertStepToYAML converts a step map to YAML string with proper indentation

// generateEngineExecutionSteps uses the new GetExecutionSteps interface method

// generateAgentVersionCapture generates a step that captures the agent version if the engine supports it

// generateCreateAwInfo generates a step that creates aw_info.json with agentic run metadata

// generateOutputCollectionStep generates a step that reads the output file and sets it as a GitHub Actions output
