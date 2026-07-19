package workflow

import (
	_ "embed"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/github/gh-aw/pkg/constants"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/gitutil"
	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/stringutil"
	"github.com/goccy/go-yaml"
)

var workflowLog = logger.New("workflow:compiler")

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

	// missingPermissionsDefaultToolsetWarning explains why strict mode was downgraded to warning.
	missingPermissionsDefaultToolsetWarning = "Some of the GitHub tools will not be available until the missing permissions are granted."
)

//go:embed schemas/github-workflow.json
var githubWorkflowSchema string

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
	workflowLog.Printf("Parsing workflow file")
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

// validateWorkflowData orchestrates all validation of workflow configuration by
// delegating to four focused validators. Each validator is independently testable
// and covers a distinct concern:
//
//   - validateExpressions: expression safety and runtime-import file checks
//   - validateFeatureConfig: feature flags and action-mode override
//   - validatePermissions: permissions parsing, MCP tool constraints, workflow_run security
//   - validateToolConfiguration: safe-outputs, GitHub tools, dispatches, and resources
func (c *Compiler) validateWorkflowData(workflowData *WorkflowData, markdownPath string) error {
	if err := validateRunnerConfig(workflowData.RunnerConfig); err != nil {
		return formatCompilerError(markdownPath, "error", err.Error(), err)
	}

	if err := validateArcDindRootless(workflowData); err != nil {
		return formatCompilerError(markdownPath, "error", err.Error(), err)
	}

	if err := c.validateExpressions(workflowData, markdownPath); err != nil {
		return err
	}

	if err := c.validateFeatureConfig(workflowData, markdownPath); err != nil {
		return err
	}

	workflowPermissions, err := c.validatePermissions(workflowData, markdownPath)
	if err != nil {
		return err
	}

	return c.validateToolConfiguration(workflowData, markdownPath, workflowPermissions)
}

// shouldDowngradeDefaultToolsetPermissionError returns true when strict-mode
// permission errors should be downgraded because the GitHub tool uses only the
// default toolset, either explicitly ([default]) or implicitly (no toolsets configured).
func shouldDowngradeDefaultToolsetPermissionError(githubTool *GitHubToolConfig) bool {
	if githubTool == nil {
		return false
	}

	if len(githubTool.Toolset) == 0 {
		return true
	}

	return len(githubTool.Toolset) == 1 && githubTool.Toolset[0] == GitHubToolset("default")
}

// generateAndValidateYAML generates GitHub Actions YAML and validates
// the output size and format.
func (c *Compiler) generateAndValidateYAML(workflowData *WorkflowData, markdownPath string, lockFile string) (string, []string, []string, error) {
	// Generate the YAML content along with the collected body secrets and action refs
	// (returned to avoid a second scan of the full YAML in the caller for safe update enforcement).
	yamlContent, bodySecrets, bodyActions, err := c.generateYAML(workflowData, markdownPath)
	if err != nil {
		return "", nil, nil, formatCompilerError(markdownPath, "error", fmt.Sprintf("failed to generate YAML: %v", err), err)
	}

	// Always validate expression sizes - this is a hard limit from GitHub Actions (21KB)
	// that cannot be bypassed, so we validate it unconditionally
	workflowLog.Print("Validating expression sizes")
	if err := c.validateExpressionSizes(yamlContent); err != nil {
		formattedErr := formatCompilerError(markdownPath, "error", fmt.Sprintf("expression size validation failed: %v", err), err)
		writeInvalidWorkflowYAML(lockFile, yamlContent)
		return "", nil, nil, formattedErr
	}

	// Template injection validation and GitHub Actions schema validation both require a
	// parsed representation of the compiled YAML.  Parse it once here and share the
	// result between the two validators to avoid redundant yaml.Unmarshal calls.
	//
	// Performance note: when schema validation is enabled (needsSchemaCheck=true) the
	// YAML is parsed regardless.  The text scan in validateTemplateInjection is only
	// used when schema validation is disabled (skipValidation=true), where targeted
	// fast-path checks avoid an unnecessary yaml.Unmarshal.
	needsSchemaCheck := !c.skipValidation

	parsedWorkflow := parseCompiledWorkflowForValidation(yamlContent, needsSchemaCheck)

	// Validate for template injection vulnerabilities (unsafe expression usage in run: commands).
	//
	// parsedWorkflow != nil means the YAML was already parsed for schema validation;
	// validateTemplateInjection reuses the pre-parsed tree (inspects only run: block values)
	// rather than re-scanning the full YAML string.  When parsedWorkflow is nil (schema
	// validation disabled), the validator uses targeted text scans to avoid an unnecessary
	// yaml.Unmarshal when all run-block expressions are in the compiler-owned allowed list.
	if err := c.validateTemplateInjection(yamlContent, lockFile, markdownPath, parsedWorkflow); err != nil {
		return "", nil, nil, err
	}

	// Validate against GitHub Actions schema (unless skipped)
	if needsSchemaCheck {
		if err := c.validateCompiledWorkflowSchemaAndResources(workflowData, markdownPath, lockFile, yamlContent, parsedWorkflow); err != nil {
			return "", nil, nil, err
		}
	} else if c.verbose {
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage("Schema validation available but skipped (use SetSkipValidation(false) to enable)"))
		c.IncrementWarningCount()
	}

	return yamlContent, bodySecrets, bodyActions, nil
}

func writeInvalidWorkflowYAML(lockFile, yamlContent string) {
	invalidFile := strings.TrimSuffix(lockFile, ".lock.yml") + ".invalid.yml"
	if writeErr := os.WriteFile(invalidFile, []byte(yamlContent), constants.FilePermPublic); writeErr == nil {
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage("Invalid workflow YAML written to: "+console.ToRelativePath(invalidFile)))
	}
}

func parseCompiledWorkflowForValidation(yamlContent string, needsSchemaCheck bool) map[string]any {
	if !needsSchemaCheck {
		return nil
	}
	workflowLog.Print("Parsing compiled YAML for validation")
	var parsedWorkflow map[string]any
	if parseErr := yaml.Unmarshal([]byte(yamlContent), &parsedWorkflow); parseErr != nil {
		return nil
	}
	return parsedWorkflow
}

func (c *Compiler) validateCompiledWorkflowSchemaAndResources(workflowData *WorkflowData, markdownPath string, lockFile string, yamlContent string, parsedWorkflow map[string]any) error {
	if err := c.validateCompiledWorkflowSchema(workflowData, markdownPath, lockFile, yamlContent, parsedWorkflow); err != nil {
		return err
	}
	if err := c.validateCompiledWorkflowRuntimeResources(workflowData, markdownPath); err != nil {
		return err
	}
	return nil
}

func (c *Compiler) validateCompiledWorkflowSchema(workflowData *WorkflowData, markdownPath string, lockFile string, yamlContent string, parsedWorkflow map[string]any) error {
	workflowLog.Print("Validating workflow against GitHub Actions schema")
	var schemaErr error
	if parsedWorkflow != nil {
		schemaErr = c.validateGitHubActionsSchemaFromParsed(parsedWorkflow)
	} else {
		schemaErr = c.validateGitHubActionsSchema(yamlContent)
	}
	if schemaErr == nil {
		return nil
	}
	fieldLine := schemaErrorFrontmatterLine(workflowData, schemaErr)
	formattedErr := formatCompilerErrorWithPosition(markdownPath, fieldLine, 1, "error", fmt.Sprintf("invalid workflow: %v", schemaErr), schemaErr)
	writeInvalidWorkflowYAML(lockFile, yamlContent)
	return formattedErr
}

func schemaErrorFrontmatterLine(workflowData *WorkflowData, schemaErr error) int {
	fieldLine := 1
	if fieldName := extractSchemaErrorField(schemaErr); fieldName != "" {
		frontmatterLines := strings.Split(workflowData.FrontmatterYAML, "\n")
		if line := findFrontmatterFieldLine(frontmatterLines, 2, fieldName); line > 0 {
			fieldLine = line
		}
	}
	return fieldLine
}

func (c *Compiler) validateCompiledWorkflowRuntimeResources(workflowData *WorkflowData, markdownPath string) error {
	workflowLog.Print("Validating container images")
	if err := c.validateContainerImages(workflowData); err != nil {
		fmt.Fprintln(os.Stderr, formatCompilerMessage(markdownPath, "warning", fmt.Sprintf("container image validation failed: %v", err)))
		c.IncrementWarningCount()
	}
	if err := c.validateRuntimePackagesWithContext(workflowData, markdownPath); err != nil {
		return err
	}
	if err := c.validateFirewallConfigWithContext(workflowData, markdownPath); err != nil {
		return err
	}
	workflowLog.Print("Validating repository features")
	if err := c.validateRepositoryFeatures(workflowData); err != nil {
		return formatCompilerError(markdownPath, "error", fmt.Sprintf("repository feature validation failed: %v", err), err)
	}
	return nil
}

func (c *Compiler) validateRuntimePackagesWithContext(workflowData *WorkflowData, markdownPath string) error {
	workflowLog.Print("Validating runtime packages")
	if err := c.validateRuntimePackages(workflowData); err != nil {
		return formatCompilerError(markdownPath, "error", fmt.Sprintf("runtime package validation failed: %v", err), err)
	}
	return nil
}

func (c *Compiler) validateFirewallConfigWithContext(workflowData *WorkflowData, markdownPath string) error {
	workflowLog.Print("Validating firewall configuration")
	if err := c.validateFirewallConfig(workflowData); err != nil {
		return formatCompilerError(markdownPath, "error", fmt.Sprintf("firewall configuration validation failed: %v", err), err)
	}
	return nil
}

// writeWorkflowOutput writes the compiled workflow to the lock file
// and handles console output formatting.
func (c *Compiler) writeWorkflowOutput(lockFile, yamlContent string, markdownPath string) error {
	// Write to lock file (unless noEmit is enabled)
	if c.noEmit {
		workflowLog.Print("Validation completed - no lock file generated (--no-emit enabled)")
	} else {
		workflowLog.Printf("Writing output to: %s", lockFile)

		// Check if content has actually changed
		contentUnchanged := false
		if existingContent, err := os.ReadFile(lockFile); err == nil {
			if normalizeHeredocDelimiters(string(existingContent)) == normalizeHeredocDelimiters(yamlContent) {
				// Content is identical (modulo random heredoc tokens) - skip write to preserve timestamp
				contentUnchanged = true
				workflowLog.Print("Lock file content unchanged - skipping write to preserve timestamp")
			}
		}

		// Only write if content has changed
		if !contentUnchanged {
			if err := os.WriteFile(lockFile, []byte(yamlContent), constants.FilePermPublic); err != nil {
				return formatCompilerError(lockFile, "error", fmt.Sprintf("failed to write lock file: %v", err), err)
			}
			workflowLog.Print("Lock file written successfully")
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

// validateTemplateInjection checks compiled YAML for template injection vulnerabilities
// (unsafe GitHub Actions expressions used directly in run: blocks).
//
// When parsedWorkflow is non-nil the YAML was already parsed for schema validation;
// this function reuses it directly by walking the run: block values in the pre-parsed
// tree, which is faster than re-scanning the full YAML string with a regex.
//
// When parsedWorkflow is nil (schema validation disabled via skipValidation), this
// function uses two targeted text scans instead of a broad any-expression check:
//
//  1. hasExpressionInRunContent(UnsafeContextPattern) — detects user-controlled
//     contexts (github.event.*, steps.*.outputs.*, inputs.*) that represent genuine
//     template-injection risks.
//
//  2. hasNonAllowedExpressionInRunContent — detects compiler-regression cases where
//     a ${{ ... }} expression that is NOT in the compiler-owned allow-list slipped
//     into a run: block.
//
// For well-formed compiled workflows (the common case) both scans return false, so
// no yaml.Unmarshal is needed at all.  A yaml.Unmarshal is only performed when at
// least one scan detects a potential issue.
func (c *Compiler) validateTemplateInjection(yamlContent, lockFile, markdownPath string, parsedWorkflow map[string]any) error {
	var templateErr error

	if parsedWorkflow != nil {
		// Path A: YAML was already parsed for schema validation; reuse it.
		// Walking the pre-parsed tree (run: block values only) is faster than
		// scanning the full YAML string.
		workflowLog.Print("Validating for template injection vulnerabilities")
		templateErr = validateNoTemplateInjectionFromParsed(parsedWorkflow)
		if templateErr == nil {
			templateErr = validateNoGitHubExpressionsInRunScriptsFromParsed(parsedWorkflow)
		}
	} else {
		// Path B: schema validation is disabled (parsedWorkflow is nil).
		//
		// Use two targeted text scans with a shared run-block walker so we can skip the
		// expensive yaml.Unmarshal when all run-block expressions are compiler-owned safe
		// references (e.g. ${{ runner.temp }}, ${{ env.FOO }}).
		//
		// needsUnsafeCheck:    true when user-controlled contexts appear in run: blocks
		//                      → triggers validateNoTemplateInjectionFromParsed
		// needsDisallowedCheck: true when any expression outside the compiler allow-list
		//                      appears in run: blocks
		//                      → triggers validateNoGitHubExpressionsInRunScriptsFromParsed
		scan := scanRunContentExpressions(yamlContent)
		needsUnsafeCheck := scan.hasUnsafe
		needsDisallowedCheck := scan.hasDisallowed

		if needsUnsafeCheck || needsDisallowedCheck {
			workflowLog.Print("Validating for template injection vulnerabilities")
			var reparsed map[string]any
			if err := yaml.Unmarshal([]byte(yamlContent), &reparsed); err != nil {
				// Malformed YAML: skip validation (compilation would have surfaced this elsewhere).
				templateInjectionValidationLog.Printf("Failed to parse YAML for template injection check: %v", err)
				reparsed = nil
			}
			if reparsed != nil {
				// validateNoTemplateInjectionFromParsed only catches user-controlled contexts,
				// so it is intentionally skipped when the UnsafeContextPattern scan found none.
				if needsUnsafeCheck {
					templateErr = validateNoTemplateInjectionFromParsed(reparsed)
				}
				if templateErr == nil && needsDisallowedCheck {
					templateErr = validateNoGitHubExpressionsInRunScriptsFromParsed(reparsed)
				}
			}
		}
	}

	if templateErr != nil {
		// Store error first so we can write invalid YAML before returning
		formattedErr := formatCompilerError(markdownPath, "error", templateErr.Error(), templateErr)
		// Write the invalid YAML to a .invalid.yml file for inspection
		invalidFile := strings.TrimSuffix(lockFile, ".lock.yml") + ".invalid.yml"
		if writeErr := os.WriteFile(invalidFile, []byte(yamlContent), constants.FilePermPublic); writeErr == nil {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage("Workflow with template injection risks written to: "+console.ToRelativePath(invalidFile)))
		}
		return formattedErr
	}
	return nil
}

// readLockFileFromHEAD reads a lock file from git HEAD using the compiler's cached
// git root directory, avoiding the overhead of spawning a subprocess to re-discover
// the repository root on every call.
func (c *Compiler) readLockFileFromHEAD(lockFile string) (string, error) {
	if c.gitRoot == "" {
		return "", errors.New("git root not available (not in a git repository or git not installed)")
	}
	return gitutil.ReadFileFromHEAD(lockFile, c.gitRoot)
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
		workflowLog.Printf("Compilation completed in %v", time.Since(startTime))
	}()

	c.resetCompilationState()
	c.configureGHESArtifactCompatibility()

	lockFile := cleanedWorkflowLockFile(markdownPath)
	workflowLog.Printf("Starting compilation: %s -> %s", markdownPath, lockFile)

	safeUpdateEnabled := c.effectiveSafeUpdate(workflowData)
	oldManifest, oldHasPR, oldHasPRTarget := c.safeUpdateBaselineIfEnabled(lockFile, safeUpdateEnabled)

	if err := c.validateWorkflowDataForCompile(workflowData, markdownPath); err != nil {
		return err
	}

	workflowLog.Printf("Workflow: %s, Tools: %d", workflowData.Name, len(workflowData.Tools))

	yamlContent, bodySecrets, bodyActions, err := c.generateYAMLForCompile(workflowData, markdownPath, lockFile)
	if err != nil {
		return err
	}

	if safeUpdateEnabled {
		c.enforceSafeUpdateWarning(workflowData, markdownPath, oldManifest, oldHasPR, oldHasPRTarget, bodySecrets, bodyActions)
	}

	if resolver := c.GetSharedActionResolver(); resolver != nil {
		resolver.MarkCompilerGeneratedActionsAsUsed()
	}

	// Write output
	if err := c.writeWorkflowOutput(lockFile, yamlContent, markdownPath); err != nil {
		return err
	}

	return nil
}

func cleanedWorkflowLockFile(markdownPath string) string {
	return filepath.Clean(stringutil.MarkdownToLockFile(markdownPath))
}

func (c *Compiler) safeUpdateBaselineIfEnabled(lockFile string, enabled bool) (*GHAWManifest, bool, bool) {
	if !enabled {
		return nil, false, false
	}
	return c.safeUpdateBaseline(lockFile)
}

func (c *Compiler) validateWorkflowDataForCompile(workflowData *WorkflowData, markdownPath string) error {
	if err := c.validateWorkflowData(workflowData, markdownPath); err != nil {
		if isFormattedCompilerError(err) {
			return err
		}
		return formatCompilerError(markdownPath, "error", "workflow validation: "+err.Error(), err)
	}
	return nil
}

func (c *Compiler) generateYAMLForCompile(workflowData *WorkflowData, markdownPath string, lockFile string) (string, []string, []string, error) {
	yamlContent, bodySecrets, bodyActions, err := c.generateAndValidateYAML(workflowData, markdownPath, lockFile)
	if err != nil {
		if isFormattedCompilerError(err) {
			return "", nil, nil, err
		}
		return "", nil, nil, formatCompilerError(markdownPath, "error", "YAML generation: "+err.Error(), err)
	}
	return yamlContent, bodySecrets, bodyActions, nil
}

func (c *Compiler) resetCompilationState() {
	c.stepOrderTracker = NewStepOrderTracker()
	c.scheduleFriendlyFormats = nil
	if c.artifactManager == nil {
		c.artifactManager = NewArtifactManager()
	} else {
		c.artifactManager.Reset()
	}
}

func (c *Compiler) configureGHESArtifactCompatibility() {
	c.ghesArtifactCompat = c.ghesCompatFromCLI
	if !c.ghesArtifactCompat {
		if repoConfig, err := c.loadRepoConfig(); err == nil && repoConfig != nil {
			c.ghesArtifactCompat = repoConfig.GHES
		}
	}
	if c.ghesArtifactCompat {
		actionPinsLog.Print("GHES compatibility mode enabled: artifact actions continue using latest non-v3 pins")
	}
}

func (c *Compiler) safeUpdateBaseline(lockFile string) (*GHAWManifest, bool, bool) {
	if cached, ok := c.priorManifests[lockFile]; ok {
		return c.safeUpdateBaselineFromCache(lockFile, cached)
	}
	if committedContent, readErr := c.readLockFileFromHEAD(lockFile); readErr == nil {
		return c.safeUpdateBaselineFromCommitted(lockFile, committedContent)
	} else {
		return c.safeUpdateBaselineFromFilesystem(lockFile, readErr)
	}
}

func (c *Compiler) safeUpdateBaselineFromCache(lockFile string, cached *GHAWManifest) (*GHAWManifest, bool, bool) {
	var oldHasPR, oldHasPRTarget bool
	if committedContent, readErr := c.readLockFileFromHEAD(lockFile); readErr == nil {
		oldHasPR, oldHasPRTarget = extractPullRequestEventPresenceFromCompiledWorkflow(committedContent)
	}
	secretCount := 0
	if cached != nil {
		secretCount = len(cached.Secrets)
	}
	workflowLog.Printf("Using pre-cached gh-aw-manifest for %s: %d secret(s)", lockFile, secretCount)
	return cached, oldHasPR, oldHasPRTarget
}

func (c *Compiler) safeUpdateBaselineFromCommitted(lockFile string, committedContent string) (*GHAWManifest, bool, bool) {
	oldHasPR, oldHasPRTarget := extractPullRequestEventPresenceFromCompiledWorkflow(committedContent)
	oldManifest, _ := parseSafeUpdateManifest("committed", committedContent)
	c.cacheSafeUpdateBaseline(lockFile, oldManifest)
	return oldManifest, oldHasPR, oldHasPRTarget
}

func (c *Compiler) safeUpdateBaselineFromFilesystem(lockFile string, readErr error) (*GHAWManifest, bool, bool) {
	workflowLog.Printf("Lock file %s not found in HEAD commit (%v); falling back to filesystem read.", lockFile, readErr)
	if existingContent, fsErr := os.ReadFile(lockFile); fsErr == nil {
		oldHasPR, oldHasPRTarget := extractPullRequestEventPresenceFromCompiledWorkflow(string(existingContent))
		oldManifest, _ := parseSafeUpdateManifest("filesystem", string(existingContent))
		c.cacheSafeUpdateBaseline(lockFile, oldManifest)
		return oldManifest, oldHasPR, oldHasPRTarget
	}
	workflowLog.Printf("Lock file %s not found (new workflow). Safe update enforcement will use an empty baseline.", lockFile)
	oldManifest := &GHAWManifest{Version: currentGHAWManifestVersion}
	c.cacheSafeUpdateBaseline(lockFile, oldManifest)
	return oldManifest, false, false
}

func parseSafeUpdateManifest(source string, content string) (*GHAWManifest, bool) {
	manifest, parseErr := ExtractGHAWManifestFromLockFile(content)
	if parseErr != nil {
		if source == "committed" {
			workflowLog.Printf("Failed to parse committed gh-aw-manifest: %v. Safe update enforcement will proceed without baseline comparison (all secrets will be considered new).", parseErr)
		} else {
			workflowLog.Printf("Failed to parse filesystem gh-aw-manifest: %v. Safe update enforcement will treat as empty manifest.", parseErr)
		}
		return nil, false
	}
	if manifest != nil {
		if source == "committed" {
			workflowLog.Printf("Loaded committed gh-aw-manifest from HEAD: %d secret(s)", len(manifest.Secrets))
		} else {
			workflowLog.Printf("Loaded gh-aw-manifest from filesystem: %d secret(s)", len(manifest.Secrets))
		}
	}
	return manifest, true
}

func (c *Compiler) cacheSafeUpdateBaseline(lockFile string, oldManifest *GHAWManifest) {
	if oldManifest == nil {
		return
	}
	if _, ok := c.priorManifests[lockFile]; !ok {
		c.priorManifests[lockFile] = oldManifest
	}
}

func (c *Compiler) enforceSafeUpdateWarning(workflowData *WorkflowData, markdownPath string, oldManifest *GHAWManifest, oldHasPR bool, oldHasPRTarget bool, bodySecrets []string, bodyActions []string) {
	currentHasPR, currentHasPRTarget := extractPullRequestEventPresenceFromOnField(workflowData.RawFrontmatter["on"])
	enforceErr := EnforceSafeUpdate(oldManifest, bodySecrets, bodyActions, workflowData.Redirect, oldHasPR, oldHasPRTarget, currentHasPR, currentHasPRTarget)
	if enforceErr == nil {
		return
	}
	warningMsg := buildSafeUpdateWarningPrompt(enforceErr.Error())
	c.AddSafeUpdateWarning(warningMsg)
	fmt.Fprintln(os.Stderr, formatCompilerMessage(markdownPath, "warning", enforceErr.Error()))
	c.IncrementWarningCount()
}
