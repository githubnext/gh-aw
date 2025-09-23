package workflow

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/githubnext/gh-aw/pkg/console"
	"github.com/githubnext/gh-aw/pkg/constants"
	"github.com/githubnext/gh-aw/pkg/parser"
	"github.com/githubnext/gh-aw/pkg/workflow/pretty"
	"github.com/goccy/go-yaml"
	"github.com/santhosh-tekuri/jsonschema/v6"
)

const (
	// OutputArtifactName is the standard name for GITHUB_AW_SAFE_OUTPUTS artifact
	OutputArtifactName = "safe_output.jsonl"
)

//go:embed schemas/github-workflow.json
var githubWorkflowSchema string

// FileTracker interface for tracking files created during compilation
type FileTracker interface {
	TrackCreated(filePath string)
}

// Compiler handles converting markdown workflows to GitHub Actions YAML
type Compiler struct {
	verbose        bool
	engineOverride string
	customOutput   string          // If set, output will be written to this path instead of default location
	version        string          // Version of the extension
	skipValidation bool            // If true, skip schema validation
	noEmit         bool            // If true, validate without generating lock files
	jobManager     *JobManager     // Manages jobs and dependencies
	engineRegistry *EngineRegistry // Registry of available agentic engines
	fileTracker    FileTracker     // Optional file tracker for tracking created files
}

// generateSafeFileName converts a workflow name to a safe filename for logs
func generateSafeFileName(name string) string {
	// Replace spaces and special characters with hyphens
	result := strings.ReplaceAll(name, " ", "-")
	result = strings.ReplaceAll(result, "/", "-")
	result = strings.ReplaceAll(result, "\\", "-")
	result = strings.ReplaceAll(result, ":", "-")
	result = strings.ReplaceAll(result, "*", "-")
	result = strings.ReplaceAll(result, "?", "-")
	result = strings.ReplaceAll(result, "\"", "-")
	result = strings.ReplaceAll(result, "<", "-")
	result = strings.ReplaceAll(result, ">", "-")
	result = strings.ReplaceAll(result, "|", "-")
	result = strings.ReplaceAll(result, "@", "-")
	result = strings.ToLower(result)

	// Remove multiple consecutive hyphens
	for strings.Contains(result, "--") {
		result = strings.ReplaceAll(result, "--", "-")
	}

	// Trim leading/trailing hyphens
	result = strings.Trim(result, "-")

	// Ensure it's not empty
	if result == "" {
		result = "workflow"
	}

	return result
}

// NewCompiler creates a new workflow compiler with optional configuration
func NewCompiler(verbose bool, engineOverride string, version string) *Compiler {
	c := &Compiler{
		verbose:        verbose,
		engineOverride: engineOverride,
		version:        version,
		skipValidation: true, // Skip validation by default for now since existing workflows don't fully comply
		jobManager:     NewJobManager(),
		engineRegistry: GetGlobalEngineRegistry(),
	}

	return c
}

// SetSkipValidation configures whether to skip schema validation
func (c *Compiler) SetSkipValidation(skip bool) {
	c.skipValidation = skip
}

// SetNoEmit configures whether to validate without generating lock files
func (c *Compiler) SetNoEmit(noEmit bool) {
	c.noEmit = noEmit
}

// SetFileTracker sets the file tracker for tracking created files
func (c *Compiler) SetFileTracker(tracker FileTracker) {
	c.fileTracker = tracker
}

// NewCompilerWithCustomOutput creates a new workflow compiler with custom output path
func NewCompilerWithCustomOutput(verbose bool, engineOverride string, customOutput string, version string) *Compiler {
	c := &Compiler{
		verbose:        verbose,
		engineOverride: engineOverride,
		customOutput:   customOutput,
		version:        version,
		skipValidation: true, // Skip validation by default for now since existing workflows don't fully comply
		jobManager:     NewJobManager(),
		engineRegistry: GetGlobalEngineRegistry(),
	}

	return c
}

// WorkflowData holds all the data needed to generate a GitHub Actions workflow
type WorkflowData struct {
	Name               string
	FrontmatterName    string // name field from frontmatter (for code scanning alert driver default)
	Description        string // optional description rendered as comment in lock file
	On                 string
	Permissions        string
	Network            string // top-level network permissions configuration
	Concurrency        string
	RunName            string
	Env                string
	If                 string
	TimeoutMinutes     string
	CustomSteps        string
	PostSteps          string // steps to run after AI execution
	RunsOn             string
	Tools              map[string]any
	MarkdownContent    string
	AI                 string        // "claude" or "codex" (for backwards compatibility)
	EngineConfig       *EngineConfig // Extended engine configuration
	StopTime           string
	Command            string              // for /command trigger support
	CommandOtherEvents map[string]any      // for merging command with other events
	AIReaction         string              // AI reaction type like "eyes", "heart", etc.
	Jobs               map[string]any      // custom job configurations with dependencies
	Cache              string              // cache configuration
	NeedsTextOutput    bool                // whether the workflow uses ${{ needs.task.outputs.text }}
	NetworkPermissions *NetworkPermissions // parsed network permissions
	SafeOutputs        *SafeOutputsConfig  // output configuration for automatic output routes
	Roles              []string            // permission levels required to trigger workflow
	CacheMemoryConfig  *CacheMemoryConfig  // parsed cache-memory configuration
}

// SafeOutputsConfig holds configuration for automatic output routes
type SafeOutputsConfig struct {
	CreateIssues                    *CreateIssuesConfig                    `yaml:"create-issues,omitempty"`
	CreateDiscussions               *CreateDiscussionsConfig               `yaml:"create-discussions,omitempty"`
	AddComments                     *AddCommentsConfig                     `yaml:"add-comments,omitempty"`
	CreatePullRequests              *CreatePullRequestsConfig              `yaml:"create-pull-requests,omitempty"`
	CreatePullRequestReviewComments *CreatePullRequestReviewCommentsConfig `yaml:"create-pull-request-review-comments,omitempty"`
	CreateCodeScanningAlerts        *CreateCodeScanningAlertsConfig        `yaml:"create-code-scanning-alerts,omitempty"`
	AddLabels                       *AddLabelsConfig                       `yaml:"add-labels,omitempty"`
	UpdateIssues                    *UpdateIssuesConfig                    `yaml:"update-issues,omitempty"`
	UpdateReleases                  *UpdateReleasesConfig                  `yaml:"update-releases,omitempty"`
	PushToPullRequestBranch         *PushToPullRequestBranchConfig         `yaml:"push-to-pull-request-branch,omitempty"`
	UploadAssets                    *UploadAssetsConfig                    `yaml:"upload-assets,omitempty"`
	MissingTool                     *MissingToolConfig                     `yaml:"missing-tool,omitempty"` // Optional for reporting missing functionality
	AllowedDomains                  []string                               `yaml:"allowed-domains,omitempty"`
	Staged                          *bool                                  `yaml:"staged,omitempty"`         // If true, emit step summary messages instead of making GitHub API calls
	Env                             map[string]string                      `yaml:"env,omitempty"`            // Environment variables to pass to safe output jobs
	GitHubToken                     string                                 `yaml:"github-token,omitempty"`   // GitHub token for safe output jobs
	MaximumPatchSize                int                                    `yaml:"max-patch-size,omitempty"` // Maximum allowed patch size in KB (defaults to 1024)
}

// CompileWorkflow converts a markdown workflow to GitHub Actions YAML
func (c *Compiler) CompileWorkflow(markdownPath string) error {

	// replace the .md extension by .lock.yml
	lockFile := strings.TrimSuffix(markdownPath, ".md") + ".lock.yml"

	if c.verbose {
		fmt.Println(console.FormatInfoMessage(fmt.Sprintf("Starting compilation of: %s", console.ToRelativePath(markdownPath))))
		fmt.Println(console.FormatInfoMessage(fmt.Sprintf("Output file: %s", console.ToRelativePath(lockFile))))
	}

	// Parse the markdown file
	if c.verbose {
		fmt.Println(console.FormatInfoMessage("Parsing workflow file..."))
	}
	workflowData, err := c.parseWorkflowFile(markdownPath)
	if err != nil {
		// Check if this is already a formatted console error
		if strings.Contains(err.Error(), ":") && (strings.Contains(err.Error(), "error:") || strings.Contains(err.Error(), "warning:")) {
			// Already formatted, return as-is
			return err
		}
		// Otherwise, create a basic formatted error
		formattedErr := console.FormatError(console.CompilerError{
			Position: console.ErrorPosition{
				File:   markdownPath,
				Line:   1,
				Column: 1,
			},
			Type:    "error",
			Message: err.Error(),
		})
		return errors.New(formattedErr)
	}

	// Validate expression safety - check that all GitHub Actions expressions are in the allowed list
	if c.verbose {
		fmt.Println(console.FormatInfoMessage("Validating expression safety..."))
	}
	if err := validateExpressionSafety(workflowData.MarkdownContent); err != nil {
		formattedErr := console.FormatError(console.CompilerError{
			Position: console.ErrorPosition{
				File:   markdownPath,
				Line:   1,
				Column: 1,
			},
			Type:    "error",
			Message: err.Error(),
		})
		return errors.New(formattedErr)
	}
	if c.verbose {
		fmt.Println(console.FormatSuccessMessage("Expression safety validation passed"))
	}

	if c.verbose {
		fmt.Println(console.FormatSuccessMessage("Successfully parsed frontmatter and markdown content"))
		fmt.Println(console.FormatInfoMessage(fmt.Sprintf("Workflow name: %s", workflowData.Name)))
		if len(workflowData.Tools) > 0 {
			fmt.Println(console.FormatInfoMessage(fmt.Sprintf("Tools configured: %d", len(workflowData.Tools))))
		}
		if workflowData.AIReaction != "" {
			fmt.Println(console.FormatInfoMessage(fmt.Sprintf("AI reaction configured: %s", workflowData.AIReaction)))
		}
	}

	// Note: compute-text functionality is now inlined directly in the task job
	// instead of using a shared action file

	// Generate the YAML content
	if c.verbose {
		fmt.Println(console.FormatInfoMessage("Generating GitHub Actions YAML..."))
	}
	yamlContent, err := c.generateYAML(workflowData, markdownPath)
	if err != nil {
		formattedErr := console.FormatError(console.CompilerError{
			Position: console.ErrorPosition{
				File:   markdownPath,
				Line:   1,
				Column: 1,
			},
			Type:    "error",
			Message: fmt.Sprintf("failed to generate YAML: %v", err),
		})
		return errors.New(formattedErr)
	}

	if c.verbose {
		fmt.Println(console.FormatSuccessMessage(fmt.Sprintf("Generated YAML content (%d bytes)", len(yamlContent))))
	}

	if c.verbose {
		fmt.Println(console.FormatSuccessMessage(fmt.Sprintf("Generated YAML content (%d bytes)", len(yamlContent))))
	}

	// Validate against GitHub Actions schema (unless skipped)
	if !c.skipValidation {
		if c.verbose {
			fmt.Println(console.FormatInfoMessage("Validating workflow against GitHub Actions schema..."))
		}
		if err := c.validateGitHubActionsSchema(yamlContent); err != nil {
			formattedErr := console.FormatError(console.CompilerError{
				Position: console.ErrorPosition{
					File:   markdownPath,
					Line:   1,
					Column: 1,
				},
				Type:    "error",
				Message: fmt.Sprintf("workflow schema validation failed: %v", err),
			})
			return errors.New(formattedErr)
		}

		if c.verbose {
			fmt.Println(console.FormatSuccessMessage("GitHub Actions schema validation passed"))
		}
	} else if c.verbose {
		fmt.Println(console.FormatWarningMessage("Schema validation available but skipped (use SetSkipValidation(false) to enable)"))
	}

	// Write to lock file (unless noEmit is enabled)
	if c.noEmit {
		if c.verbose {
			fmt.Println(console.FormatInfoMessage("Validation completed - no lock file generated (--no-emit enabled)"))
		}
	} else {
		if c.verbose {
			fmt.Println(console.FormatInfoMessage(fmt.Sprintf("Writing output to: %s", console.ToRelativePath(lockFile))))
		}
		if err := os.WriteFile(lockFile, []byte(yamlContent), 0644); err != nil {
			formattedErr := console.FormatError(console.CompilerError{
				Position: console.ErrorPosition{
					File:   lockFile,
					Line:   1,
					Column: 1,
				},
				Type:    "error",
				Message: fmt.Sprintf("failed to write lock file: %v", err),
			})
			return errors.New(formattedErr)
		}
	}

	// Display success message with file size if we generated a lock file
	if c.noEmit {
		fmt.Println(console.FormatSuccessMessage(console.ToRelativePath(markdownPath)))
	} else {
		// Get the size of the generated lock file for display
		if lockFileInfo, err := os.Stat(lockFile); err == nil {
			lockSize := pretty.FormatFileSize(lockFileInfo.Size())
			fmt.Println(console.FormatSuccessMessage(fmt.Sprintf("%s (%s)", console.ToRelativePath(markdownPath), lockSize)))
		} else {
			// Fallback to original display if we can't get file info
			fmt.Println(console.FormatSuccessMessage(console.ToRelativePath(markdownPath)))
		}
	}
	return nil
}

// validateGitHubActionsSchema validates the generated YAML content against the GitHub Actions workflow schema
func (c *Compiler) validateGitHubActionsSchema(yamlContent string) error {
	// Convert YAML to any for JSON conversion
	var workflowData any
	if err := yaml.Unmarshal([]byte(yamlContent), &workflowData); err != nil {
		return fmt.Errorf("failed to parse YAML for schema validation: %w", err)
	}

	// Convert to JSON for schema validation
	jsonData, err := json.Marshal(workflowData)
	if err != nil {
		return fmt.Errorf("failed to convert YAML to JSON for validation: %w", err)
	}

	// Parse the embedded schema
	var schemaDoc any
	if err := json.Unmarshal([]byte(githubWorkflowSchema), &schemaDoc); err != nil {
		return fmt.Errorf("failed to parse embedded GitHub Actions schema: %w", err)
	}

	// Create compiler and add the schema as a resource
	loader := jsonschema.NewCompiler()
	schemaURL := "https://json.schemastore.org/github-workflow.json"
	if err := loader.AddResource(schemaURL, schemaDoc); err != nil {
		return fmt.Errorf("failed to add schema resource: %w", err)
	}

	// Compile the schema
	schema, err := loader.Compile(schemaURL)
	if err != nil {
		return fmt.Errorf("failed to compile GitHub Actions schema: %w", err)
	}

	// Validate the JSON data against the schema
	var jsonObj any
	if err := json.Unmarshal(jsonData, &jsonObj); err != nil {
		return fmt.Errorf("failed to unmarshal JSON for validation: %w", err)
	}

	if err := schema.Validate(jsonObj); err != nil {
		return fmt.Errorf("GitHub Actions schema validation failed: %w", err)
	}

	return nil
}

// parseWorkflowFile parses a markdown workflow file and extracts all necessary data
func (c *Compiler) parseWorkflowFile(markdownPath string) (*WorkflowData, error) {
	if c.verbose {
		fmt.Println(console.FormatInfoMessage(fmt.Sprintf("Reading file: %s", console.ToRelativePath(markdownPath))))
	}

	// Read the file
	content, err := os.ReadFile(markdownPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	if c.verbose {
		fmt.Println(console.FormatInfoMessage(fmt.Sprintf("File size: %d bytes", len(content))))
		fmt.Println(console.FormatInfoMessage("Extracting frontmatter..."))
	}

	// Parse frontmatter and markdown
	result, err := parser.ExtractFrontmatterFromContent(string(content))
	if err != nil {
		// Use FrontmatterStart from result if available, otherwise default to line 2 (after opening ---)
		frontmatterStart := 2
		if result != nil && result.FrontmatterStart > 0 {
			frontmatterStart = result.FrontmatterStart
		}
		return nil, c.createFrontmatterError(markdownPath, string(content), err, frontmatterStart)
	}

	if len(result.Frontmatter) == 0 {
		return nil, fmt.Errorf("no frontmatter found")
	}

	if result.Markdown == "" {
		return nil, fmt.Errorf("no markdown content found")
	}

	// Check for deprecated stop-time usage at root level BEFORE schema validation
	if stopTimeValue := c.extractYAMLValue(result.Frontmatter, "stop-time"); stopTimeValue != "" {
		return nil, fmt.Errorf("'stop-time' is no longer supported at the root level. Please move it under the 'on:' section and rename to 'stop-after:'.\n\nExample:\n---\non:\n  schedule:\n    - cron: \"0 9 * * 1\"\n  stop-after: \"%s\"\n---", stopTimeValue)
	}

	// Validate main workflow frontmatter contains only expected entries
	if err := parser.ValidateMainWorkflowFrontmatterWithSchemaAndLocation(result.Frontmatter, markdownPath); err != nil {
		return nil, err
	}

	if c.verbose {
		fmt.Println(console.FormatInfoMessage(fmt.Sprintf("Frontmatter length: %d characters", len(result.Frontmatter))))
		fmt.Println(console.FormatInfoMessage(fmt.Sprintf("Markdown content length: %d characters", len(result.Markdown))))
	}

	markdownDir := filepath.Dir(markdownPath)

	// Extract AI engine setting from frontmatter
	engineSetting, engineConfig := c.extractEngineConfig(result.Frontmatter)

	// Extract network permissions from frontmatter
	networkPermissions := c.extractNetworkPermissions(result.Frontmatter)

	// Default to 'defaults' network access if no network permissions specified
	if networkPermissions == nil {
		networkPermissions = &NetworkPermissions{
			Mode: "defaults",
		}
	}

	// Override with command line AI engine setting if provided
	if c.engineOverride != "" {
		originalEngineSetting := engineSetting
		if originalEngineSetting != "" && originalEngineSetting != c.engineOverride {
			fmt.Println(console.FormatWarningMessage(fmt.Sprintf("Command line --engine %s overrides markdown file engine: %s", c.engineOverride, originalEngineSetting)))
		}
		engineSetting = c.engineOverride
	}

	// Process @include directives to extract engine configurations and check for conflicts
	includedEngines, err := parser.ExpandIncludesForEngines(result.Markdown, markdownDir)
	if err != nil {
		return nil, fmt.Errorf("failed to expand includes for engines: %w", err)
	}

	// Validate that only one engine field exists across all files
	finalEngineSetting, err := c.validateSingleEngineSpecification(engineSetting, includedEngines)
	if err != nil {
		return nil, err
	}
	if finalEngineSetting != "" {
		engineSetting = finalEngineSetting
	}

	// Apply the default AI engine setting if not specified
	if engineSetting == "" {
		defaultEngine := c.engineRegistry.GetDefaultEngine()
		engineSetting = defaultEngine.GetID()
		if c.verbose {
			fmt.Println(console.FormatInfoMessage(fmt.Sprintf("NOTE: No 'engine:' setting found, defaulting to: %s", engineSetting)))
		}
	}

	// Validate the engine setting
	if err := c.validateEngine(engineSetting); err != nil {
		return nil, fmt.Errorf("invalid engine setting '%s': %w", engineSetting, err)
	}

	// Get the agentic engine instance
	agenticEngine, err := c.getAgenticEngine(engineSetting)
	if err != nil {
		return nil, fmt.Errorf("failed to get agentic engine: %w", err)
	}

	if c.verbose {
		fmt.Println(console.FormatInfoMessage(fmt.Sprintf("AI engine: %s (%s)", agenticEngine.GetDisplayName(), engineSetting)))
		if agenticEngine.IsExperimental() {
			fmt.Println(console.FormatWarningMessage(fmt.Sprintf("Using experimental engine: %s", agenticEngine.GetDisplayName())))
		}
		fmt.Println(console.FormatInfoMessage("Processing tools and includes..."))
	}

	// Extract SafeOutputs configuration early so we can use it when applying default tools
	safeOutputs := c.extractSafeOutputsConfig(result.Frontmatter)

	var tools map[string]any

	// Extract tools from the main file
	topTools := extractToolsFromFrontmatter(result.Frontmatter)

	// Process @include directives to extract additional tools
	includedTools, err := parser.ExpandIncludes(result.Markdown, markdownDir, true)
	if err != nil {
		return nil, fmt.Errorf("failed to expand includes for tools: %w", err)
	}

	// Merge tools
	tools, err = c.mergeTools(topTools, includedTools)

	if err != nil {
		return nil, fmt.Errorf("failed to merge tools: %w", err)
	}

	// Validate MCP configurations
	if err := ValidateMCPConfigs(tools); err != nil {
		return nil, fmt.Errorf("invalid MCP configuration: %w", err)
	}

	// Validate HTTP transport support for the current engine
	if err := c.validateHTTPTransportSupport(tools, agenticEngine); err != nil {
		return nil, fmt.Errorf("HTTP transport not supported: %w", err)
	}

	if !agenticEngine.SupportsToolsAllowlist() {
		// For engines that don't support tool allowlists (like codex), ignore tools section and provide warnings
		fmt.Println(console.FormatWarningMessage(fmt.Sprintf("Using experimental %s support (engine: %s)", agenticEngine.GetDisplayName(), engineSetting)))
		if _, hasTools := result.Frontmatter["tools"]; hasTools {
			fmt.Println(console.FormatWarningMessage(fmt.Sprintf("'tools' section ignored when using engine: %s (%s doesn't support MCP tool allow-listing)", engineSetting, agenticEngine.GetDisplayName())))
		}
		tools = map[string]any{}
		// For now, we'll add a basic github tool (always uses docker MCP)
		githubConfig := map[string]any{}

		tools["github"] = githubConfig
	}

	// Validate max-turns support for the current engine
	if err := c.validateMaxTurnsSupport(result.Frontmatter, agenticEngine); err != nil {
		return nil, fmt.Errorf("max-turns not supported: %w", err)
	}

	// Process @include directives in markdown content
	markdownContent, err := parser.ExpandIncludes(result.Markdown, markdownDir, false)
	if err != nil {
		return nil, fmt.Errorf("failed to expand includes in markdown: %w", err)
	}

	if c.verbose {
		fmt.Println(console.FormatInfoMessage("Expanded includes in markdown content"))
	}

	// Extract workflow name
	workflowName, err := parser.ExtractWorkflowNameFromMarkdown(markdownPath)
	if err != nil {
		return nil, fmt.Errorf("failed to extract workflow name: %w", err)
	}

	// Check if frontmatter specifies a custom name and use it instead
	frontmatterName := extractStringValue(result.Frontmatter, "name")
	if frontmatterName != "" {
		workflowName = frontmatterName
	}

	if c.verbose {
		fmt.Println(console.FormatInfoMessage(fmt.Sprintf("Extracted workflow name: '%s'", workflowName)))
	}

	// Check if the markdown content uses the text output
	needsTextOutput := c.detectTextOutputUsage(markdownContent)

	// Build workflow data
	workflowData := &WorkflowData{
		Name:               workflowName,
		FrontmatterName:    frontmatterName,
		Description:        c.extractDescription(result.Frontmatter),
		Tools:              tools,
		MarkdownContent:    markdownContent,
		AI:                 engineSetting,
		EngineConfig:       engineConfig,
		NetworkPermissions: networkPermissions,
		NeedsTextOutput:    needsTextOutput,
	}

	// Extract YAML sections from frontmatter - use direct frontmatter map extraction
	// to avoid issues with nested keys (e.g., tools.mcps.*.env being confused with top-level env)
	workflowData.On = c.extractTopLevelYAMLSection(result.Frontmatter, "on")
	workflowData.Permissions = c.extractTopLevelYAMLSection(result.Frontmatter, "permissions")
	workflowData.Network = c.extractTopLevelYAMLSection(result.Frontmatter, "network")
	workflowData.Concurrency = c.extractTopLevelYAMLSection(result.Frontmatter, "concurrency")
	workflowData.RunName = c.extractTopLevelYAMLSection(result.Frontmatter, "run-name")
	workflowData.Env = c.extractTopLevelYAMLSection(result.Frontmatter, "env")
	workflowData.If = c.extractIfCondition(result.Frontmatter)
	workflowData.TimeoutMinutes = c.extractTopLevelYAMLSection(result.Frontmatter, "timeout_minutes")
	workflowData.CustomSteps = c.extractTopLevelYAMLSection(result.Frontmatter, "steps")
	workflowData.PostSteps = c.extractTopLevelYAMLSection(result.Frontmatter, "post-steps")
	workflowData.RunsOn = c.extractTopLevelYAMLSection(result.Frontmatter, "runs-on")
	workflowData.Cache = c.extractTopLevelYAMLSection(result.Frontmatter, "cache")
	workflowData.CacheMemoryConfig = c.extractCacheMemoryConfig(topTools)

	// Process stop-after configuration from the on: section
	err = c.processStopAfterConfiguration(result.Frontmatter, workflowData)
	if err != nil {
		return nil, err
	}

	workflowData.Command = c.extractCommandName(result.Frontmatter)
	workflowData.Jobs = c.extractJobsFromFrontmatter(result.Frontmatter)
	workflowData.Roles = c.extractRoles(result.Frontmatter)

	// Use the already extracted output configuration
	workflowData.SafeOutputs = safeOutputs

	// Parse the "on" section for command triggers, reactions, and other events
	err = c.parseOnSection(result.Frontmatter, workflowData, markdownPath)
	if err != nil {
		return nil, err
	}

	// Apply defaults
	c.applyDefaults(workflowData, markdownPath)

	// Apply pull request draft filter if specified
	c.applyPullRequestDraftFilter(workflowData, result.Frontmatter)

	// Apply pull request fork filter if specified
	c.applyPullRequestForkFilter(workflowData, result.Frontmatter)

	return workflowData, nil
}

// extractTopLevelYAMLSection extracts a top-level YAML section from the frontmatter map
// This ensures we only extract keys at the root level, avoiding nested keys with the same name
func (c *Compiler) extractTopLevelYAMLSection(frontmatter map[string]any, key string) string {
	value, exists := frontmatter[key]
	if !exists {
		return ""
	}

	// Convert the value back to YAML format
	yamlBytes, err := yaml.Marshal(map[string]any{key: value})
	if err != nil {
		return ""
	}

	yamlStr := string(yamlBytes)
	// Remove the trailing newline
	yamlStr = strings.TrimSuffix(yamlStr, "\n")

	// Clean up quoted keys - replace "key": with key:
	// This handles cases where YAML marshaling adds unnecessary quotes around reserved words like "on"
	quotedKeyPattern := `"` + key + `":`
	unquotedKey := key + ":"
	yamlStr = strings.Replace(yamlStr, quotedKeyPattern, unquotedKey, 1)

	// Special handling for "on" section - comment out draft and fork fields from pull_request
	if key == "on" {
		yamlStr = c.commentOutProcessedFieldsInOnSection(yamlStr)
	}

	return yamlStr
}

// extractIfCondition extracts the if condition from frontmatter, returning just the expression
// without the "if: " prefix
func (c *Compiler) extractIfCondition(frontmatter map[string]any) string {
	value, exists := frontmatter["if"]
	if !exists {
		return ""
	}

	// Convert the value to string - it should be just the expression
	if strValue, ok := value.(string); ok {
		return c.extractExpressionFromIfString(strValue)
	}

	return ""
}

// extractDescription extracts the description field from frontmatter
func (c *Compiler) extractDescription(frontmatter map[string]any) string {
	value, exists := frontmatter["description"]
	if !exists {
		return ""
	}

	// Convert the value to string
	if strValue, ok := value.(string); ok {
		return strings.TrimSpace(strValue)
	}

	return ""
}

// extractExpressionFromIfString extracts the expression part from a string that might
// contain "if: expression" or just "expression", returning just the expression
func (c *Compiler) extractExpressionFromIfString(ifString string) string {
	if ifString == "" {
		return ""
	}

	// Check if the string starts with "if: " and strip it
	if strings.HasPrefix(ifString, "if: ") {
		return strings.TrimSpace(ifString[4:]) // Remove "if: " prefix
	}

	// Return the string as-is (it's just the expression)
	return ifString
}

// commentOutProcessedFieldsInOnSection comments out draft, fork, and forks fields in pull_request sections within the YAML string
// These fields are processed separately by applyPullRequestDraftFilter and applyPullRequestForkFilter and should be commented for documentation
func (c *Compiler) commentOutProcessedFieldsInOnSection(yamlStr string) string {
	lines := strings.Split(yamlStr, "\n")
	var result []string
	inPullRequest := false
	inForksArray := false

	for _, line := range lines {
		// Check if we're entering a pull_request section
		if strings.Contains(line, "pull_request:") {
			inPullRequest = true
			result = append(result, line)
			continue
		}

		// Check if we're leaving the pull_request section (new top-level key or end of indent)
		if inPullRequest {
			// If line is not indented or is a new top-level key, we're out of pull_request
			if strings.TrimSpace(line) != "" && !strings.HasPrefix(line, "    ") && !strings.HasPrefix(line, "\t") {
				inPullRequest = false
				inForksArray = false
			}
		}

		trimmedLine := strings.TrimSpace(line)

		// Check if we're entering the forks array
		if inPullRequest && strings.HasPrefix(trimmedLine, "forks:") {
			inForksArray = true
		}

		// Check if we're leaving the forks array by encountering another top-level field at the same level
		if inForksArray && inPullRequest && strings.TrimSpace(line) != "" {
			// Get the indentation of the current line
			lineIndent := len(line) - len(strings.TrimLeft(line, " \t"))

			// If this is a non-dash line at the same level as the forks field (4 spaces), we're out of the array
			if lineIndent == 4 && !strings.HasPrefix(trimmedLine, "-") && !strings.HasPrefix(trimmedLine, "forks:") {
				inForksArray = false
			}
		}

		// Determine if we should comment out this line
		shouldComment := false
		var commentReason string

		if inPullRequest && strings.Contains(trimmedLine, "draft:") {
			shouldComment = true
			commentReason = " # Draft filtering applied via job conditions"
		} else if inPullRequest && strings.HasPrefix(trimmedLine, "forks:") {
			shouldComment = true
			commentReason = " # Fork filtering applied via job conditions"
		} else if inForksArray && strings.HasPrefix(trimmedLine, "-") {
			shouldComment = true
			commentReason = " # Fork filtering applied via job conditions"
		}

		if shouldComment {
			// Preserve the original indentation and comment out the line
			indentation := ""
			trimmed := strings.TrimLeft(line, " \t")
			if len(line) > len(trimmed) {
				indentation = line[:len(line)-len(trimmed)]
			}

			commentedLine := indentation + "# " + trimmed + commentReason
			result = append(result, commentedLine)
		} else {
			result = append(result, line)
		}
	}

	return strings.Join(result, "\n")
}

// parseOnSection parses the "on" section from frontmatter to extract command triggers, reactions, and other events
func (c *Compiler) parseOnSection(frontmatter map[string]any, workflowData *WorkflowData, markdownPath string) error {
	// Check if "command" is used as a trigger in the "on" section
	// Also extract "reaction" from the "on" section
	var hasCommand bool
	var hasReaction bool
	var hasStopAfter bool
	var otherEvents map[string]any

	if onValue, exists := frontmatter["on"]; exists {
		// Check for new format: on.command and on.reaction
		if onMap, ok := onValue.(map[string]any); ok {
			// Check for stop-after in the on section
			if _, hasStopAfterKey := onMap["stop-after"]; hasStopAfterKey {
				hasStopAfter = true
			}

			// Extract reaction from on section
			if reactionValue, hasReactionField := onMap["reaction"]; hasReactionField {
				hasReaction = true
				if reactionStr, ok := reactionValue.(string); ok {
					workflowData.AIReaction = reactionStr
				}
			}

			if _, hasCommandKey := onMap["command"]; hasCommandKey {
				hasCommand = true
				// Set default command to filename if not specified in the command section
				if workflowData.Command == "" {
					baseName := strings.TrimSuffix(filepath.Base(markdownPath), ".md")
					workflowData.Command = baseName
				}
				// Check for conflicting events
				conflictingEvents := []string{"issues", "issue_comment", "pull_request", "pull_request_review_comment"}
				for _, eventName := range conflictingEvents {
					if _, hasConflict := onMap[eventName]; hasConflict {
						return fmt.Errorf("cannot use 'command' with '%s' in the same workflow", eventName)
					}
				}

				// Clear the On field so applyDefaults will handle command trigger generation
				workflowData.On = ""
			}
			// Extract other (non-conflicting) events excluding command, reaction, and stop-after
			otherEvents = filterMapKeys(onMap, "command", "reaction", "stop-after")
		}
	}

	// Clear command field if no command trigger was found
	if !hasCommand {
		workflowData.Command = ""
	}

	// Store other events for merging in applyDefaults
	if hasCommand && len(otherEvents) > 0 {
		// We'll store this and handle it in applyDefaults
		workflowData.On = "" // This will trigger command handling in applyDefaults
		workflowData.CommandOtherEvents = otherEvents
	} else if (hasReaction || hasStopAfter) && len(otherEvents) > 0 {
		// Only re-marshal the "on" if we have to
		onEventsYAML, err := yaml.Marshal(map[string]any{"on": otherEvents})
		if err == nil {
			workflowData.On = strings.TrimSuffix(string(onEventsYAML), "\n")
		} else {
			// Fallback to extracting the original on field (this will include reaction but shouldn't matter for compilation)
			workflowData.On = c.extractTopLevelYAMLSection(frontmatter, "on")
		}
	}

	return nil
}

// generateJobName converts a workflow name to a valid YAML job identifier
func (c *Compiler) generateJobName(workflowName string) string {
	// Convert to lowercase and replace spaces and special characters with hyphens
	jobName := strings.ToLower(workflowName)

	// Replace spaces and common punctuation with hyphens
	jobName = strings.ReplaceAll(jobName, " ", "-")
	jobName = strings.ReplaceAll(jobName, ":", "-")
	jobName = strings.ReplaceAll(jobName, ".", "-")
	jobName = strings.ReplaceAll(jobName, ",", "-")
	jobName = strings.ReplaceAll(jobName, "(", "-")
	jobName = strings.ReplaceAll(jobName, ")", "-")
	jobName = strings.ReplaceAll(jobName, "/", "-")
	jobName = strings.ReplaceAll(jobName, "\\", "-")
	jobName = strings.ReplaceAll(jobName, "@", "-")
	jobName = strings.ReplaceAll(jobName, "'", "")
	jobName = strings.ReplaceAll(jobName, "\"", "")

	// Remove multiple consecutive hyphens
	for strings.Contains(jobName, "--") {
		jobName = strings.ReplaceAll(jobName, "--", "-")
	}

	// Remove leading/trailing hyphens
	jobName = strings.Trim(jobName, "-")

	// Ensure it's not empty and starts with a letter or underscore
	if jobName == "" || (!strings.ContainsAny(string(jobName[0]), "abcdefghijklmnopqrstuvwxyz_")) {
		jobName = "workflow-" + jobName
	}

	return jobName
} // extractCommandName extracts the command name from frontmatter using the new nested format
func (c *Compiler) extractCommandName(frontmatter map[string]any) string {
	// Check new format: on.command.name
	if onValue, exists := frontmatter["on"]; exists {
		if onMap, ok := onValue.(map[string]any); ok {
			if commandValue, hasCommand := onMap["command"]; hasCommand {
				if commandMap, ok := commandValue.(map[string]any); ok {
					if nameValue, hasName := commandMap["name"]; hasName {
						if nameStr, ok := nameValue.(string); ok {
							return nameStr
						}
					}
				}
			}
		}
	}

	return ""
}

// applyDefaults applies default values for missing workflow sections
func (c *Compiler) applyDefaults(data *WorkflowData, markdownPath string) {
	// Check if this is a command trigger workflow (by checking if user specified "on.command")
	isCommandTrigger := false
	if data.On == "" {
		// Check the original frontmatter for command trigger
		content, err := os.ReadFile(markdownPath)
		if err == nil {
			result, err := parser.ExtractFrontmatterFromContent(string(content))
			if err == nil {
				if onValue, exists := result.Frontmatter["on"]; exists {
					// Check for new format: on.command
					if onMap, ok := onValue.(map[string]any); ok {
						if _, hasCommand := onMap["command"]; hasCommand {
							isCommandTrigger = true
						}
					}
				}
			}
		}
	}

	if data.On == "" {
		if isCommandTrigger {
			// Generate command-specific GitHub Actions events (updated to include reopened and pull_request)
			commandEvents := `on:
  issues:
    types: [opened, edited, reopened]
  issue_comment:
    types: [created, edited]
  pull_request:
    types: [opened, edited, reopened]
  pull_request_review_comment:
    types: [created, edited]`

			// Check if there are other events to merge
			if len(data.CommandOtherEvents) > 0 {
				// Merge command events with other events
				commandEventsMap := map[string]any{
					"issues": map[string]any{
						"types": []string{"opened", "edited", "reopened"},
					},
					"issue_comment": map[string]any{
						"types": []string{"created", "edited"},
					},
					"pull_request": map[string]any{
						"types": []string{"opened", "edited", "reopened"},
					},
					"pull_request_review_comment": map[string]any{
						"types": []string{"created", "edited"},
					},
				}

				// Merge other events into command events
				for key, value := range data.CommandOtherEvents {
					commandEventsMap[key] = value
				}

				// Convert merged events to YAML
				mergedEventsYAML, err := yaml.Marshal(map[string]any{"on": commandEventsMap})
				if err == nil {
					data.On = strings.TrimSuffix(string(mergedEventsYAML), "\n")
				} else {
					// If conversion fails, just use command events
					data.On = commandEvents
				}
			} else {
				data.On = commandEvents
			}

			// Add conditional logic to check for command in issue content
			// Use event-aware condition that only applies command checks to comment-related events
			hasOtherEvents := len(data.CommandOtherEvents) > 0
			commandConditionTree := buildEventAwareCommandCondition(data.Command, hasOtherEvents)

			if data.If == "" {
				data.If = commandConditionTree.Render()
			}
		} else {
			data.On = `on:
  # Start either every 10 minutes, or when some kind of human event occurs.
  # Because of the implicit "concurrency" section, only one instance of this
  # workflow will run at a time.
  schedule:
    - cron: "0/10 * * * *"
  issues:
    types: [opened, edited, closed]
  issue_comment:
    types: [created, edited]
  pull_request:
    types: [opened, edited, closed]
  push:
    branches:
      - main
  workflow_dispatch:`
		}
	}

	if data.Permissions == "" {
		// Default behavior: use read-all permissions
		data.Permissions = `permissions: read-all`
	}

	// Generate concurrency configuration using the dedicated concurrency module
	data.Concurrency = GenerateConcurrencyConfig(data, isCommandTrigger)

	if data.RunName == "" {
		data.RunName = fmt.Sprintf(`run-name: "%s"`, data.Name)
	}

	if data.TimeoutMinutes == "" {
		data.TimeoutMinutes = `timeout_minutes: 5`
	}

	if data.RunsOn == "" {
		data.RunsOn = "runs-on: ubuntu-latest"
	}
	// Apply default tools
	data.Tools = c.applyDefaultTools(data.Tools, data.SafeOutputs)
}

// applyPullRequestDraftFilter applies draft filter conditions for pull_request triggers
func (c *Compiler) applyPullRequestDraftFilter(data *WorkflowData, frontmatter map[string]any) {
	// Check if there's an "on" section in the frontmatter
	onValue, hasOn := frontmatter["on"]
	if !hasOn {
		return
	}

	// Check if "on" is an object (not a string)
	onMap, isOnMap := onValue.(map[string]any)
	if !isOnMap {
		return
	}

	// Check if there's a pull_request section
	prValue, hasPR := onMap["pull_request"]
	if !hasPR {
		return
	}

	// Check if pull_request is an object with draft settings
	prMap, isPRMap := prValue.(map[string]any)
	if !isPRMap {
		return
	}

	// Check if draft is specified
	draftValue, hasDraft := prMap["draft"]
	if !hasDraft {
		return
	}

	// Check if draft is a boolean
	draftBool, isDraftBool := draftValue.(bool)
	if !isDraftBool {
		// If draft is not a boolean, don't add filter
		return
	}

	// Generate conditional logic based on draft value using expression nodes
	var draftCondition ConditionNode
	if draftBool {
		// draft: true - include only draft PRs
		// The condition should be true for non-pull_request events or for draft pull_requests
		notPullRequestEvent := BuildNotEquals(
			BuildPropertyAccess("github.event_name"),
			BuildStringLiteral("pull_request"),
		)
		isDraftPR := BuildEquals(
			BuildPropertyAccess("github.event.pull_request.draft"),
			BuildBooleanLiteral(true),
		)
		draftCondition = &OrNode{
			Left:  notPullRequestEvent,
			Right: isDraftPR,
		}
	} else {
		// draft: false - exclude draft PRs
		// The condition should be true for non-pull_request events or for non-draft pull_requests
		notPullRequestEvent := BuildNotEquals(
			BuildPropertyAccess("github.event_name"),
			BuildStringLiteral("pull_request"),
		)
		isNotDraftPR := BuildEquals(
			BuildPropertyAccess("github.event.pull_request.draft"),
			BuildBooleanLiteral(false),
		)
		draftCondition = &OrNode{
			Left:  notPullRequestEvent,
			Right: isNotDraftPR,
		}
	}

	// Build condition tree and render
	existingCondition := data.If
	conditionTree := buildConditionTree(existingCondition, draftCondition.Render())
	data.If = conditionTree.Render()
}

// applyPullRequestForkFilter applies fork filter conditions for pull_request triggers
// Supports "forks: []string" with glob patterns
func (c *Compiler) applyPullRequestForkFilter(data *WorkflowData, frontmatter map[string]any) {
	// Check if there's an "on" section in the frontmatter
	onValue, hasOn := frontmatter["on"]
	if !hasOn {
		return
	}

	// Check if "on" is an object (not a string)
	onMap, isOnMap := onValue.(map[string]any)
	if !isOnMap {
		return
	}

	// Check if there's a pull_request section
	prValue, hasPR := onMap["pull_request"]
	if !hasPR {
		return
	}

	// Check if pull_request is an object with fork settings
	prMap, isPRMap := prValue.(map[string]any)
	if !isPRMap {
		return
	}

	// Check for "forks" field (string or array)
	forksValue, hasForks := prMap["forks"]

	if !hasForks {
		return
	}

	// Convert forks value to []string, handling both string and array formats
	var allowedForks []string

	// Handle string format (e.g., forks: "*" or forks: "org/*")
	if forksStr, isForksStr := forksValue.(string); isForksStr {
		allowedForks = []string{forksStr}
	} else if forksArray, isForksArray := forksValue.([]any); isForksArray {
		// Handle array format (e.g., forks: ["*", "org/repo"])
		for _, fork := range forksArray {
			if forkStr, isForkStr := fork.(string); isForkStr {
				allowedForks = append(allowedForks, forkStr)
			}
		}
	} else {
		// Invalid forks format, skip
		return
	}

	// If "*" wildcard is present, skip fork filtering (allow all forks)
	for _, pattern := range allowedForks {
		if pattern == "*" {
			return // No fork filtering needed
		}
	}

	// Build condition for allowed forks with glob support
	notPullRequestEvent := BuildNotEquals(
		BuildPropertyAccess("github.event_name"),
		BuildStringLiteral("pull_request"),
	)
	allowedForksCondition := BuildFromAllowedForks(allowedForks)

	forkCondition := &OrNode{
		Left:  notPullRequestEvent,
		Right: allowedForksCondition,
	}

	// Build condition tree and render
	existingCondition := data.If
	conditionTree := buildConditionTree(existingCondition, forkCondition.Render())
	data.If = conditionTree.Render()
}

// extractToolsFromFrontmatter extracts tools section from frontmatter map
func extractToolsFromFrontmatter(frontmatter map[string]any) map[string]any {
	tools, exists := frontmatter["tools"]
	if !exists {
		return make(map[string]any)
	}

	if toolsMap, ok := tools.(map[string]any); ok {
		return toolsMap
	}

	return make(map[string]any)
}

// mergeTools merges two tools maps, combining allowed arrays when keys coincide
func (c *Compiler) mergeTools(topTools map[string]any, includedToolsJSON string) (map[string]any, error) {
	if includedToolsJSON == "" || includedToolsJSON == "{}" {
		return topTools, nil
	}

	var includedTools map[string]any
	if err := json.Unmarshal([]byte(includedToolsJSON), &includedTools); err != nil {
		return topTools, nil // Return original tools if parsing fails
	}

	// Use the merge logic from the parser package
	mergedTools, err := parser.MergeTools(topTools, includedTools)
	if err != nil {
		return nil, fmt.Errorf("failed to merge tools: %w", err)
	}
	return mergedTools, nil
}

// applyDefaultTools adds default read-only GitHub MCP tools, creating github tool if not present
func (c *Compiler) applyDefaultTools(tools map[string]any, safeOutputs *SafeOutputsConfig) map[string]any {
	// Always apply default GitHub tools (create github section if it doesn't exist)

	// Define the default read-only GitHub MCP tools
	defaultGitHubTools := []string{
		// actions
		"download_workflow_run_artifact",
		"get_job_logs",
		"get_workflow_run",
		"get_workflow_run_logs",
		"get_workflow_run_usage",
		"list_workflow_jobs",
		"list_workflow_run_artifacts",
		"list_workflow_runs",
		"list_workflows",
		// code security
		"get_code_scanning_alert",
		"list_code_scanning_alerts",
		// context
		"get_me",
		// dependabot
		"get_dependabot_alert",
		"list_dependabot_alerts",
		// discussions
		"get_discussion",
		"get_discussion_comments",
		"list_discussion_categories",
		"list_discussions",
		// issues
		"get_issue",
		"get_issue_comments",
		"list_issues",
		"search_issues",
		// notifications
		"get_notification_details",
		"list_notifications",
		// organizations
		"search_orgs",
		// prs
		"get_pull_request",
		"get_pull_request_comments",
		"get_pull_request_diff",
		"get_pull_request_files",
		"get_pull_request_reviews",
		"get_pull_request_status",
		"list_pull_requests",
		"search_pull_requests",
		// repos
		"get_commit",
		"get_file_contents",
		"get_tag",
		"list_branches",
		"list_commits",
		"list_tags",
		"search_code",
		"search_repositories",
		// secret protection
		"get_secret_scanning_alert",
		"list_secret_scanning_alerts",
		// users
		"search_users",
	}

	if tools == nil {
		tools = make(map[string]any)
	}

	// Get existing github tool configuration
	githubTool := tools["github"]
	var githubConfig map[string]any

	if toolConfig, ok := githubTool.(map[string]any); ok {
		githubConfig = make(map[string]any)
		for k, v := range toolConfig {
			githubConfig[k] = v
		}
	} else {
		githubConfig = make(map[string]any)
	}

	// Get existing allowed tools
	var existingAllowed []any
	if allowed, hasAllowed := githubConfig["allowed"]; hasAllowed {
		if allowedSlice, ok := allowed.([]any); ok {
			existingAllowed = allowedSlice
		}
	}

	// Create a set of existing tools for efficient lookup
	existingToolsSet := make(map[string]bool)
	for _, tool := range existingAllowed {
		if toolStr, ok := tool.(string); ok {
			existingToolsSet[toolStr] = true
		}
	}

	// Add default tools that aren't already present
	newAllowed := make([]any, len(existingAllowed))
	copy(newAllowed, existingAllowed)

	for _, defaultTool := range defaultGitHubTools {
		if !existingToolsSet[defaultTool] {
			newAllowed = append(newAllowed, defaultTool)
		}
	}

	// Update the github tool configuration
	githubConfig["allowed"] = newAllowed
	tools["github"] = githubConfig

	// Add Git commands and file editing tools when safe-outputs includes create-pull-request or push-to-pull-request-branch
	if safeOutputs != nil && needsGitCommands(safeOutputs) {

		// Add edit tool with null value
		if _, exists := tools["edit"]; !exists {
			tools["edit"] = nil
		}
		gitCommands := []any{
			"git checkout:*",
			"git branch:*",
			"git switch:*",
			"git add:*",
			"git rm:*",
			"git commit:*",
			"git merge:*",
		}

		// Add bash tool with Git commands if not already present
		if _, exists := tools["bash"]; !exists {
			// bash tool doesn't exist, add it with Git commands
			tools["bash"] = gitCommands
		} else {
			// bash tool exists, merge Git commands with existing commands
			existingBash := tools["bash"]
			if existingCommands, ok := existingBash.([]any); ok {
				// Convert existing commands to strings for comparison
				existingSet := make(map[string]bool)
				for _, cmd := range existingCommands {
					if cmdStr, ok := cmd.(string); ok {
						existingSet[cmdStr] = true
						// If we see :* or *, all bash commands are already allowed
						if cmdStr == ":*" || cmdStr == "*" {
							// Don't add specific Git commands since all are already allowed
							goto bashComplete
						}
					}
				}

				// Add Git commands that aren't already present
				newCommands := make([]any, len(existingCommands))
				copy(newCommands, existingCommands)
				for _, gitCmd := range gitCommands {
					if gitCmdStr, ok := gitCmd.(string); ok {
						if !existingSet[gitCmdStr] {
							newCommands = append(newCommands, gitCmd)
						}
					}
				}
				tools["bash"] = newCommands
			} else if existingBash == nil {
				_ = existingBash // Keep the nil value as-is
			}
		}
	bashComplete:
	}
	return tools
}

// needsGitCommands checks if safe outputs configuration requires Git commands
func needsGitCommands(safeOutputs *SafeOutputsConfig) bool {
	if safeOutputs == nil {
		return false
	}
	return safeOutputs.CreatePullRequests != nil || safeOutputs.PushToPullRequestBranch != nil
}

// detectTextOutputUsage checks if the markdown content uses ${{ needs.task.outputs.text }}
func (c *Compiler) detectTextOutputUsage(markdownContent string) bool {
	// Check for the specific GitHub Actions expression
	hasUsage := strings.Contains(markdownContent, "${{ needs.task.outputs.text }}")
	if c.verbose {
		if hasUsage {
			fmt.Println(console.FormatInfoMessage("Detected usage of task.outputs.text - compute-text step will be included"))
		} else {
			fmt.Println(console.FormatInfoMessage("No usage of task.outputs.text found - compute-text step will be skipped"))
		}
	}
	return hasUsage
}

// generateYAML generates the complete GitHub Actions YAML content
func (c *Compiler) generateYAML(data *WorkflowData, markdownPath string) (string, error) {
	// Reset job manager for this compilation
	c.jobManager = NewJobManager()

	// Build all jobs
	if err := c.buildJobs(data, markdownPath); err != nil {
		return "", fmt.Errorf("failed to build jobs: %w", err)
	}

	// Validate job dependencies
	if err := c.jobManager.ValidateDependencies(); err != nil {
		return "", fmt.Errorf("job dependency validation failed: %w", err)
	}

	var yaml strings.Builder

	// Add auto-generated disclaimer
	yaml.WriteString("# This file was automatically generated by gh-aw. DO NOT EDIT.\n")
	yaml.WriteString("# To update this file, edit the corresponding .md file and run:\n")
	yaml.WriteString("#   " + constants.CLIExtensionPrefix + " compile\n")
	yaml.WriteString("# For more information: https://github.com/githubnext/gh-aw/blob/main/.github/instructions/github-agentic-workflows.instructions.md\n")

	// Add description comment if provided
	if data.Description != "" {
		yaml.WriteString("#\n")
		// Split description into lines and prefix each with "# "
		descriptionLines := strings.Split(strings.TrimSpace(data.Description), "\n")
		for _, line := range descriptionLines {
			yaml.WriteString(fmt.Sprintf("# %s\n", strings.TrimSpace(line)))
		}
	}

	// Add stop-time comment if configured
	if data.StopTime != "" {
		yaml.WriteString("#\n")
		yaml.WriteString(fmt.Sprintf("# Effective stop-time: %s\n", data.StopTime))
	}

	yaml.WriteString("\n")

	// Write basic workflow structure
	yaml.WriteString(fmt.Sprintf("name: \"%s\"\n", data.Name))
	yaml.WriteString(data.On + "\n\n")
	yaml.WriteString("permissions: {}\n\n")
	yaml.WriteString(data.Concurrency + "\n\n")
	yaml.WriteString(data.RunName + "\n\n")

	// Add env section if present
	if data.Env != "" {
		yaml.WriteString(data.Env + "\n\n")
	}

	// Add cache comment if cache configuration was provided
	if data.Cache != "" {
		yaml.WriteString("# Cache configuration from frontmatter was processed and added to the main job steps\n\n")
	}

	// Generate jobs section using JobManager
	yaml.WriteString(c.jobManager.RenderToYAML())

	return yaml.String(), nil
}

// isTaskJobNeeded determines if the task job is required
func (c *Compiler) isTaskJobNeeded(data *WorkflowData, needsPermissionCheck bool) bool {
	// Task job is needed if:
	// 1. Command is configured (for team member checking)
	// 2. Text output is needed (for compute-text action)
	// 3. If condition is specified (to handle runtime conditions)
	// 4. Permission checks are needed (consolidated team member validation)
	return data.Command != "" || data.NeedsTextOutput || data.If != "" || needsPermissionCheck
}

// buildJobs creates all jobs for the workflow and adds them to the job manager
func (c *Compiler) buildJobs(data *WorkflowData, markdownPath string) error {
	// Try to read frontmatter to determine event types for safe events check
	// This is used for the enhanced permission checking logic
	var frontmatter map[string]any
	if content, err := os.ReadFile(markdownPath); err == nil {
		if result, err := parser.ExtractFrontmatterFromContent(string(content)); err == nil {
			frontmatter = result.Frontmatter
		}
	}
	// If frontmatter cannot be read, we'll fall back to the basic permission check logic

	// Generate job name from workflow name
	jobName := c.generateJobName(data.Name)

	// Build task job if needed (preamble job that handles runtime conditions and permission checks)
	var taskJobCreated bool
	needsPermissionCheck := c.needsRoleCheck(data, frontmatter)

	if c.isTaskJobNeeded(data, needsPermissionCheck) {
		taskJob, err := c.buildTaskJob(data, frontmatter)
		if err != nil {
			return fmt.Errorf("failed to build task job: %w", err)
		}
		if err := c.jobManager.AddJob(taskJob); err != nil {
			return fmt.Errorf("failed to add task job: %w", err)
		}
		taskJobCreated = true
	}

	// Build add_reaction job only if ai-reaction is configured
	if data.AIReaction != "" {
		addReactionJob, err := c.buildAddReactionJob(data, taskJobCreated, frontmatter)
		if err != nil {
			return fmt.Errorf("failed to build add_reaction job: %w", err)
		}
		if err := c.jobManager.AddJob(addReactionJob); err != nil {
			return fmt.Errorf("failed to add add_reaction job: %w", err)
		}
	}

	// Build main workflow job
	mainJob, err := c.buildMainJob(data, jobName, taskJobCreated, frontmatter)
	if err != nil {
		return fmt.Errorf("failed to build main job: %w", err)
	}
	if err := c.jobManager.AddJob(mainJob); err != nil {
		return fmt.Errorf("failed to add main job: %w", err)
	}

	// Build safe outputs jobs if configured
	if err := c.buildSafeOutputsJobs(data, jobName, taskJobCreated, frontmatter, markdownPath); err != nil {
		return fmt.Errorf("failed to build safe outputs jobs: %w", err)
	}
	// Build additional custom jobs from frontmatter jobs section
	if err := c.buildCustomJobs(data); err != nil {
		return fmt.Errorf("failed to build custom jobs: %w", err)
	}

	return nil
}

// buildSafeOutputsJobs creates all safe outputs jobs if configured
func (c *Compiler) buildSafeOutputsJobs(data *WorkflowData, jobName string, taskJobCreated bool, frontmatter map[string]any, markdownPath string) error {
	if data.SafeOutputs == nil {
		return nil
	}

	// Build create_issue job if output.create_issue is configured
	if data.SafeOutputs.CreateIssues != nil {
		createIssueJob, err := c.buildCreateOutputIssueJob(data, jobName, taskJobCreated, frontmatter)
		if err != nil {
			return fmt.Errorf("failed to build create_issue job: %w", err)
		}
		if err := c.jobManager.AddJob(createIssueJob); err != nil {
			return fmt.Errorf("failed to add create_issue job: %w", err)
		}
	}

	// Build create_discussion job if output.create_discussion is configured
	if data.SafeOutputs.CreateDiscussions != nil {
		createDiscussionJob, err := c.buildCreateOutputDiscussionJob(data, jobName)
		if err != nil {
			return fmt.Errorf("failed to build create_discussion job: %w", err)
		}
		if err := c.jobManager.AddJob(createDiscussionJob); err != nil {
			return fmt.Errorf("failed to add create_discussion job: %w", err)
		}
	}

	// Build create_issue_comment job if output.add-comment is configured
	if data.SafeOutputs.AddComments != nil {
		createCommentJob, err := c.buildCreateOutputAddCommentJob(data, jobName)
		if err != nil {
			return fmt.Errorf("failed to build create_issue_comment job: %w", err)
		}
		if err := c.jobManager.AddJob(createCommentJob); err != nil {
			return fmt.Errorf("failed to add create_issue_comment job: %w", err)
		}
	}

	// Build create_pr_review_comment job if output.create-pull-request-review-comment is configured
	if data.SafeOutputs.CreatePullRequestReviewComments != nil {
		createPRReviewCommentJob, err := c.buildCreateOutputPullRequestReviewCommentJob(data, jobName)
		if err != nil {
			return fmt.Errorf("failed to build create_pr_review_comment job: %w", err)
		}
		if err := c.jobManager.AddJob(createPRReviewCommentJob); err != nil {
			return fmt.Errorf("failed to add create_pr_review_comment job: %w", err)
		}
	}

	// Build create_code_scanning_alert job if output.create-code-scanning-alert is configured
	if data.SafeOutputs.CreateCodeScanningAlerts != nil {
		// Extract the workflow filename without extension for rule ID prefix
		workflowFilename := strings.TrimSuffix(filepath.Base(markdownPath), ".md")
		createCodeScanningAlertJob, err := c.buildCreateOutputCodeScanningAlertJob(data, jobName, workflowFilename)
		if err != nil {
			return fmt.Errorf("failed to build create_code_scanning_alert job: %w", err)
		}
		if err := c.jobManager.AddJob(createCodeScanningAlertJob); err != nil {
			return fmt.Errorf("failed to add create_code_scanning_alert job: %w", err)
		}
	}

	// Build create_pull_request job if output.create-pull-request is configured
	if data.SafeOutputs.CreatePullRequests != nil {
		createPullRequestJob, err := c.buildCreateOutputPullRequestJob(data, jobName)
		if err != nil {
			return fmt.Errorf("failed to build create_pull_request job: %w", err)
		}
		if err := c.jobManager.AddJob(createPullRequestJob); err != nil {
			return fmt.Errorf("failed to add create_pull_request job: %w", err)
		}
	}

	// Build add_labels job if output.add-labels is configured (including null/empty)
	if data.SafeOutputs.AddLabels != nil {
		addLabelsJob, err := c.buildCreateOutputLabelJob(data, jobName)
		if err != nil {
			return fmt.Errorf("failed to build add_labels job: %w", err)
		}
		if err := c.jobManager.AddJob(addLabelsJob); err != nil {
			return fmt.Errorf("failed to add add_labels job: %w", err)
		}
	}

	// Build update_issue job if output.update-issue is configured
	if data.SafeOutputs.UpdateIssues != nil {
		updateIssueJob, err := c.buildCreateOutputUpdateIssueJob(data, jobName)
		if err != nil {
			return fmt.Errorf("failed to build update_issue job: %w", err)
		}
		if err := c.jobManager.AddJob(updateIssueJob); err != nil {
			return fmt.Errorf("failed to add update_issue job: %w", err)
		}
	}

	// Build update_release job if output.update-release is configured
	if data.SafeOutputs.UpdateReleases != nil {
		updateReleaseJob, err := c.buildCreateOutputUpdateReleaseJob(data, jobName)
		if err != nil {
			return fmt.Errorf("failed to build update_release job: %w", err)
		}
		if err := c.jobManager.AddJob(updateReleaseJob); err != nil {
			return fmt.Errorf("failed to add update_release job: %w", err)
		}
	}

	// Build push_to_pull_request_branch job if output.push-to-pull-request-branch is configured
	if data.SafeOutputs.PushToPullRequestBranch != nil {
		pushToBranchJob, err := c.buildCreateOutputPushToPullRequestBranchJob(data, jobName)
		if err != nil {
			return fmt.Errorf("failed to build push_to_pull_request_branch job: %w", err)
		}
		if err := c.jobManager.AddJob(pushToBranchJob); err != nil {
			return fmt.Errorf("failed to add push_to_pull_request_branch job: %w", err)
		}
	}

	// Build missing_tool job (always enabled when SafeOutputs exists)
	if data.SafeOutputs.MissingTool != nil {
		missingToolJob, err := c.buildCreateOutputMissingToolJob(data, jobName)
		if err != nil {
			return fmt.Errorf("failed to build missing_tool job: %w", err)
		}
		if err := c.jobManager.AddJob(missingToolJob); err != nil {
			return fmt.Errorf("failed to add missing_tool job: %w", err)
		}
	}

	// Build upload_assets job if output.upload-asset is configured
	if data.SafeOutputs.UploadAssets != nil {
		uploadAssetsJob, err := c.buildUploadAssetsJob(data, jobName, taskJobCreated, frontmatter)
		if err != nil {
			return fmt.Errorf("failed to build upload_assets job: %w", err)
		}
		if err := c.jobManager.AddJob(uploadAssetsJob); err != nil {
			return fmt.Errorf("failed to add upload_assets job: %w", err)
		}
	}

	return nil
}

// buildTaskJob creates the preamble task job that acts as a barrier for runtime conditions
func (c *Compiler) buildTaskJob(data *WorkflowData, frontmatter map[string]any) (*Job, error) {
	outputs := map[string]string{}
	var steps []string

	// Add team member check based on new permission requirements (issue #567)
	needsRoleCheck := c.needsRoleCheck(data, frontmatter)

	if needsRoleCheck || data.Command != "" {
		steps = c.generateRoleCheck(data, steps)
	}

	// Use inlined compute-text script only if needed (no shared action)
	if data.NeedsTextOutput {
		steps = append(steps, "      - name: Compute current body text\n")
		steps = append(steps, "        id: compute-text\n")
		steps = append(steps, "        uses: actions/github-script@v8\n")
		steps = append(steps, "        with:\n")
		steps = append(steps, "          script: |\n")

		// Inline the JavaScript directly instead of using shared action
		steps = append(steps, FormatJavaScriptForYAML(computeTextScript)...)

		// Set up outputs
		outputs["text"] = "${{ steps.compute-text.outputs.text }}"
	}

	// If no steps have been added, add a dummy step to make the job valid
	// This can happen when the task job is created only for an if condition
	if len(steps) == 0 {
		steps = append(steps, "      - name: Task job condition barrier\n")
		steps = append(steps, "        run: echo \"Task job executed - conditions satisfied\"\n")
	}

	job := &Job{
		Name:        "task",
		If:          data.If, // Use the existing condition (which may include alias checks)
		RunsOn:      "runs-on: ubuntu-latest",
		Permissions: "", // Will be set below based on permission check needs
		Steps:       steps,
		Outputs:     outputs,
	}

	// Add actions: write permission if team member checks are present
	// Any workflow that needs permission checks will use setCancelled() which requires actions: write
	requiresWorkflowCancellation := data.Command != "" || needsRoleCheck

	if requiresWorkflowCancellation {
		job.Permissions = "permissions:\n      actions: write  # Required for github.rest.actions.cancelWorkflowRun()"
	}

	return job, nil
}

// buildMainJob creates the main workflow job
func (c *Compiler) buildMainJob(data *WorkflowData, jobName string, taskJobCreated bool, frontmatter map[string]any) (*Job, error) {
	var steps []string

	// Add permission checks if no task job was created but permission checks are needed
	if !taskJobCreated {
		needsRoleCheck := c.needsRoleCheck(data, frontmatter)

		if needsRoleCheck {
			steps = c.generateRoleCheck(data, steps)
		}
	}

	// Build step content using the generateMainJobSteps helper method
	// but capture it into a string instead of writing directly
	var stepBuilder strings.Builder
	c.generateMainJobSteps(&stepBuilder, data)

	// Split the steps content into individual step entries
	stepsContent := stepBuilder.String()
	if stepsContent != "" {
		steps = append(steps, stepsContent)
	}

	var depends []string
	if taskJobCreated {
		depends = []string{"task"} // Depend on the task job only if it exists
	}

	// Build outputs for all engines (GITHUB_AW_SAFE_OUTPUTS functionality)
	// Only include output if the workflow actually uses the safe-outputs feature
	var outputs map[string]string
	if data.SafeOutputs != nil {
		outputs = map[string]string{
			"output": "${{ steps.collect_output.outputs.output }}",
		}
	}

	// Determine the job condition for command workflows
	var jobCondition string
	if data.Command != "" {
		// Build the command trigger condition
		commandCondition := buildCommandOnlyCondition(data.Command)
		commandConditionStr := commandCondition.Render()
		jobCondition = commandConditionStr
	} else {
		jobCondition = data.If // Use the original If condition from the workflow data
	}

	job := &Job{
		Name:        jobName,
		If:          jobCondition,
		RunsOn:      c.indentYAMLLines(data.RunsOn, "    "),
		Permissions: c.indentYAMLLines(data.Permissions, "    "),
		Steps:       steps,
		Needs:       depends,
		Outputs:     outputs,
	}

	return job, nil
}

// generateMainJobSteps generates the steps section for the main job
func (c *Compiler) generateMainJobSteps(yaml *strings.Builder, data *WorkflowData) {
	// Add custom steps or default checkout step
	if data.CustomSteps != "" {
		// Remove "steps:" line and adjust indentation
		lines := strings.Split(data.CustomSteps, "\n")
		if len(lines) > 1 {
			for _, line := range lines[1:] {
				// Skip empty lines
				if strings.TrimSpace(line) == "" {
					yaml.WriteString("\n")
					continue
				}

				// Simply add 6 spaces for job context indentation
				yaml.WriteString("      " + line + "\n")
			}
		}
	} else {
		yaml.WriteString("      - name: Checkout repository\n")
		yaml.WriteString("        uses: actions/checkout@v5\n")
	}

	// Add cache steps if cache configuration is present
	generateCacheSteps(yaml, data, c.verbose)

	// Add cache-memory steps if cache-memory configuration is present
	generateCacheMemorySteps(yaml, data, c.verbose)

	// Configure git credentials if git operations will be needed
	if needsGitCommands(data.SafeOutputs) {
		c.generateGitConfiguration(yaml, data)
	}

	// Add Node.js setup if the engine requires it
	engine, err := c.getAgenticEngine(data.AI)

	if err != nil {
		return
	}

	// Add engine-specific installation steps
	installSteps := engine.GetInstallationSteps(data)
	for _, step := range installSteps {
		for _, line := range step {
			yaml.WriteString(line + "\n")
		}
	}

	// Generate output file setup step only if safe-outputs feature is used (GITHUB_AW_SAFE_OUTPUTS functionality)
	if data.SafeOutputs != nil {
		c.generateOutputFileSetup(yaml)
	}

	// Add MCP setup
	c.generateMCPSetup(yaml, data.Tools, engine, data)

	// Add safety checks before executing agentic tools
	c.generateStopTimeChecks(yaml, data)

	// Add prompt creation step
	c.generatePrompt(yaml, data)

	logFile := generateSafeFileName(data.Name)
	logFileFull := fmt.Sprintf("/tmp/%s.log", logFile)

	// Generate aw_info.json with agentic run metadata
	c.generateCreateAwInfo(yaml, data, engine)

	// Upload info to artifact
	c.generateUploadAwInfo(yaml)

	// Add AI execution step using the agentic engine
	c.generateEngineExecutionSteps(yaml, data, engine, logFileFull)

	// Add output collection step only if safe-outputs feature is used (GITHUB_AW_SAFE_OUTPUTS functionality)
	if data.SafeOutputs != nil {
		c.generateOutputCollectionStep(yaml, data)
	}

	// Add engine-declared output files collection (if any)
	if len(engine.GetDeclaredOutputFiles()) > 0 {
		c.generateEngineOutputCollection(yaml, engine)
	}

	// Extract and upload squid access logs (if any proxy tools were used)
	c.generateExtractAccessLogs(yaml, data.Tools)
	c.generateUploadAccessLogs(yaml, data.Tools)

	// upload MCP logs (if any MCP tools were used)
	c.generateUploadMCPLogs(yaml, data.Tools)

	// parse agent logs for GITHUB_STEP_SUMMARY
	c.generateLogParsing(yaml, engine, logFileFull)

	// upload agent logs
	c.generateUploadAgentLogs(yaml, logFile, logFileFull)

	// upload assets if upload-asset is configured
	if data.SafeOutputs != nil && data.SafeOutputs.UploadAssets != nil {
		c.generateUploadAssets(yaml)
	}

	// Add error validation for AI execution logs
	c.generateErrorValidation(yaml, engine, logFileFull, data)

	// Add git patch generation step only if safe-outputs create-pull-request feature is used
	if data.SafeOutputs != nil && (data.SafeOutputs.CreatePullRequests != nil || data.SafeOutputs.PushToPullRequestBranch != nil) {
		c.generateGitPatchStep(yaml)
	}

	// Add post-steps (if any) after AI execution
	c.generatePostSteps(yaml, data)
}

func (c *Compiler) generateUploadAgentLogs(yaml *strings.Builder, logFile string, logFileFull string) {
	yaml.WriteString("      - name: Upload agent logs\n")
	yaml.WriteString("        if: always()\n")
	yaml.WriteString("        uses: actions/upload-artifact@v4\n")
	yaml.WriteString("        with:\n")
	fmt.Fprintf(yaml, "          name: %s.log\n", logFile)
	fmt.Fprintf(yaml, "          path: %s\n", logFileFull)
	yaml.WriteString("          if-no-files-found: warn\n")
}

func (c *Compiler) generateUploadAssets(yaml *strings.Builder) {
	yaml.WriteString("      - name: Upload safe outputs assets\n")
	yaml.WriteString("        if: always()\n")
	yaml.WriteString("        uses: actions/upload-artifact@v4\n")
	yaml.WriteString("        with:\n")
	yaml.WriteString("          name: safe-outputs-assets\n")
	yaml.WriteString("          path: /tmp/safe-outputs/assets/\n")
	yaml.WriteString("          if-no-files-found: ignore\n")
}

func (c *Compiler) generateLogParsing(yaml *strings.Builder, engine CodingAgentEngine, logFileFull string) {
	parserScriptName := engine.GetLogParserScriptId()
	if parserScriptName == "" {
		// Skip log parsing if engine doesn't provide a parser
		return
	}

	logParserScript := GetLogParserScript(parserScriptName)
	if logParserScript == "" {
		// Skip if parser script not found
		return
	}

	yaml.WriteString("      - name: Parse agent logs for step summary\n")
	yaml.WriteString("        if: always()\n")
	yaml.WriteString("        uses: actions/github-script@v8\n")
	yaml.WriteString("        env:\n")
	fmt.Fprintf(yaml, "          GITHUB_AW_AGENT_OUTPUT: %s\n", logFileFull)
	yaml.WriteString("        with:\n")
	yaml.WriteString("          script: |\n")

	// Inline the JavaScript code with proper indentation
	steps := FormatJavaScriptForYAML(logParserScript)
	for _, step := range steps {
		yaml.WriteString(step)
	}
}

func (c *Compiler) generateErrorValidation(yaml *strings.Builder, engine CodingAgentEngine, logFileFull string, data *WorkflowData) {
	// Concatenate engine error patterns and configured error patterns
	var errorPatterns []ErrorPattern

	// Add engine-defined patterns
	enginePatterns := engine.GetErrorPatterns()
	errorPatterns = append(errorPatterns, enginePatterns...)

	// Add user-configured patterns from engine config
	if data.EngineConfig != nil && len(data.EngineConfig.ErrorPatterns) > 0 {
		errorPatterns = append(errorPatterns, data.EngineConfig.ErrorPatterns...)
	}

	// Skip if no error patterns are available
	if len(errorPatterns) == 0 {
		return
	}

	errorValidationScript := validateErrorsScript
	if errorValidationScript == "" {
		// Skip if validation script not found
		return
	}

	yaml.WriteString("      - name: Validate agent logs for errors\n")
	yaml.WriteString("        if: always()\n")
	yaml.WriteString("        uses: actions/github-script@v8\n")
	yaml.WriteString("        env:\n")
	fmt.Fprintf(yaml, "          GITHUB_AW_AGENT_OUTPUT: %s\n", logFileFull)

	// Add error patterns as a single JSON array
	patternsJSON, err := json.Marshal(errorPatterns)
	if err != nil {
		// Skip if patterns can't be marshaled
		return
	}
	fmt.Fprintf(yaml, "          GITHUB_AW_ERROR_PATTERNS: %q\n", string(patternsJSON))

	yaml.WriteString("        with:\n")
	yaml.WriteString("          script: |\n")

	// Inline the JavaScript code with proper indentation
	steps := FormatJavaScriptForYAML(errorValidationScript)
	for _, step := range steps {
		yaml.WriteString(step)
	}
}

func (c *Compiler) generateUploadAwInfo(yaml *strings.Builder) {
	yaml.WriteString("      - name: Upload agentic run info\n")
	yaml.WriteString("        if: always()\n")
	yaml.WriteString("        uses: actions/upload-artifact@v4\n")
	yaml.WriteString("        with:\n")
	yaml.WriteString("          name: aw_info.json\n")
	yaml.WriteString("          path: /tmp/aw_info.json\n")
	yaml.WriteString("          if-no-files-found: warn\n")
}

func (c *Compiler) generateExtractAccessLogs(yaml *strings.Builder, tools map[string]any) {
	// Check if any tools require proxy setup
	var proxyTools []string
	for toolName, toolConfig := range tools {
		if toolConfigMap, ok := toolConfig.(map[string]any); ok {
			needsProxySetup, _ := needsProxy(toolConfigMap)
			if needsProxySetup {
				proxyTools = append(proxyTools, toolName)
			}
		}
	}

	// If no proxy tools, no access logs to extract
	if len(proxyTools) == 0 {
		return
	}

	yaml.WriteString("      - name: Extract squid access logs\n")
	yaml.WriteString("        if: always()\n")
	yaml.WriteString("        run: |\n")
	yaml.WriteString("          mkdir -p /tmp/access-logs\n")

	for _, toolName := range proxyTools {
		fmt.Fprintf(yaml, "          echo 'Extracting access.log from squid-proxy-%s container'\n", toolName)
		fmt.Fprintf(yaml, "          if docker ps -a --format '{{.Names}}' | grep -q '^squid-proxy-%s$'; then\n", toolName)
		fmt.Fprintf(yaml, "            docker cp squid-proxy-%s:/var/log/squid/access.log /tmp/access-logs/access-%s.log 2>/dev/null || echo 'No access.log found for %s'\n", toolName, toolName, toolName)
		yaml.WriteString("          else\n")
		fmt.Fprintf(yaml, "            echo 'Container squid-proxy-%s not found'\n", toolName)
		yaml.WriteString("          fi\n")
	}
}

func (c *Compiler) generateUploadAccessLogs(yaml *strings.Builder, tools map[string]any) {
	// Check if any tools require proxy setup
	var proxyTools []string
	for toolName, toolConfig := range tools {
		if toolConfigMap, ok := toolConfig.(map[string]any); ok {
			needsProxySetup, _ := needsProxy(toolConfigMap)
			if needsProxySetup {
				proxyTools = append(proxyTools, toolName)
			}
		}
	}

	// If no proxy tools, no access logs to upload
	if len(proxyTools) == 0 {
		return
	}

	yaml.WriteString("      - name: Upload squid access logs\n")
	yaml.WriteString("        if: always()\n")
	yaml.WriteString("        uses: actions/upload-artifact@v4\n")
	yaml.WriteString("        with:\n")
	yaml.WriteString("          name: access.log\n")
	yaml.WriteString("          path: /tmp/access-logs/\n")
	yaml.WriteString("          if-no-files-found: warn\n")
}

func (c *Compiler) generateUploadMCPLogs(yaml *strings.Builder, tools map[string]any) {
	yaml.WriteString("      - name: Upload MCP logs\n")
	yaml.WriteString("        if: always()\n")
	yaml.WriteString("        uses: actions/upload-artifact@v4\n")
	yaml.WriteString("        with:\n")
	yaml.WriteString("          name: mcp-logs\n")
	yaml.WriteString("          path: /tmp/mcp-logs/\n")
	yaml.WriteString("          if-no-files-found: ignore\n")
}

func (c *Compiler) generatePrompt(yaml *strings.Builder, data *WorkflowData) {
	yaml.WriteString("      - name: Create prompt\n")

	// Add environment variables section - always include GITHUB_AW_PROMPT
	yaml.WriteString("        env:\n")
	yaml.WriteString("          GITHUB_AW_PROMPT: /tmp/aw-prompts/prompt.txt\n")

	// Only add GITHUB_AW_SAFE_OUTPUTS environment variable if safe-outputs feature is used
	if data.SafeOutputs != nil {
		yaml.WriteString("          GITHUB_AW_SAFE_OUTPUTS: ${{ env.GITHUB_AW_SAFE_OUTPUTS }}\n")
	}

	yaml.WriteString("        run: |\n")
	yaml.WriteString("          mkdir -p /tmp/aw-prompts\n")
	yaml.WriteString("          cat > $GITHUB_AW_PROMPT << 'EOF'\n")

	// Add markdown content with proper indentation (removing XML comments)
	cleanedMarkdownContent := removeXMLComments(data.MarkdownContent)
	for _, line := range strings.Split(cleanedMarkdownContent, "\n") {
		yaml.WriteString("          " + line + "\n")
	}

	// Add cache folder notification if cache-memory is enabled
	generateCacheMemoryPromptSection(yaml, data.CacheMemoryConfig)

	generateSafeOutputsPromptSection(yaml, data.SafeOutputs)

	yaml.WriteString("          EOF\n")

	// Add step to print prompt to GitHub step summary for debugging
	yaml.WriteString("      - name: Print prompt to step summary\n")
	yaml.WriteString("        run: |\n")
	yaml.WriteString("          echo \"## Generated Prompt\" >> $GITHUB_STEP_SUMMARY\n")
	yaml.WriteString("          echo \"\" >> $GITHUB_STEP_SUMMARY\n")
	yaml.WriteString("          echo '``````markdown' >> $GITHUB_STEP_SUMMARY\n")
	yaml.WriteString("          cat $GITHUB_AW_PROMPT >> $GITHUB_STEP_SUMMARY\n")
	yaml.WriteString("          echo '``````' >> $GITHUB_STEP_SUMMARY\n")
	yaml.WriteString("        env:\n")
	yaml.WriteString("          GITHUB_AW_PROMPT: /tmp/aw-prompts/prompt.txt\n")
}

// generatePostSteps generates the post-steps section that runs after AI execution
func (c *Compiler) generatePostSteps(yaml *strings.Builder, data *WorkflowData) {
	if data.PostSteps != "" {
		// Remove "post-steps:" line and adjust indentation, similar to CustomSteps processing
		lines := strings.Split(data.PostSteps, "\n")
		if len(lines) > 1 {
			for _, line := range lines[1:] {
				// Remove 2 existing spaces, add 6
				if strings.HasPrefix(line, "  ") {
					yaml.WriteString("    " + line[2:] + "\n")
				} else {
					yaml.WriteString("    " + line + "\n")
				}
			}
		}
	}
}

// extractJobsFromFrontmatter extracts job configuration from frontmatter
func (c *Compiler) extractJobsFromFrontmatter(frontmatter map[string]any) map[string]any {
	if jobs, exists := frontmatter["jobs"]; exists {
		if jobsMap, ok := jobs.(map[string]any); ok {
			return jobsMap
		}
	}
	return make(map[string]any)
}

// buildCustomJobs creates custom jobs defined in the frontmatter jobs section
func (c *Compiler) buildCustomJobs(data *WorkflowData) error {
	for jobName, jobConfig := range data.Jobs {
		if configMap, ok := jobConfig.(map[string]any); ok {
			job := &Job{
				Name: jobName,
			}

			// Extract job dependencies
			if needs, hasNeeds := configMap["needs"]; hasNeeds {
				if needsList, ok := needs.([]any); ok {
					for _, need := range needsList {
						if needStr, ok := need.(string); ok {
							job.Needs = append(job.Needs, needStr)
						}
					}
				} else if needStr, ok := needs.(string); ok {
					// Single dependency as string
					job.Needs = append(job.Needs, needStr)
				}
			}

			// Extract other job properties
			if runsOn, hasRunsOn := configMap["runs-on"]; hasRunsOn {
				if runsOnStr, ok := runsOn.(string); ok {
					job.RunsOn = fmt.Sprintf("runs-on: %s", runsOnStr)
				}
			}

			if ifCond, hasIf := configMap["if"]; hasIf {
				if ifStr, ok := ifCond.(string); ok {
					job.If = c.extractExpressionFromIfString(ifStr)
				}
			}

			// Add basic steps if specified
			if steps, hasSteps := configMap["steps"]; hasSteps {
				if stepsList, ok := steps.([]any); ok {
					for _, step := range stepsList {
						if stepMap, ok := step.(map[string]any); ok {
							stepYAML, err := c.convertStepToYAML(stepMap)
							if err != nil {
								return fmt.Errorf("failed to convert step to YAML for job '%s': %w", jobName, err)
							}
							job.Steps = append(job.Steps, stepYAML)
						}
					}
				}
			}

			if err := c.jobManager.AddJob(job); err != nil {
				return fmt.Errorf("failed to add custom job '%s': %w", jobName, err)
			}
		}
	}

	return nil
}

// convertStepToYAML converts a step map to YAML string with proper indentation
func (c *Compiler) convertStepToYAML(stepMap map[string]any) (string, error) {
	return ConvertStepToYAML(stepMap)
}

// generateEngineExecutionSteps uses the new GetExecutionSteps interface method
func (c *Compiler) generateEngineExecutionSteps(yaml *strings.Builder, data *WorkflowData, engine CodingAgentEngine, logFile string) {
	steps := engine.GetExecutionSteps(data, logFile)

	for _, step := range steps {
		for _, line := range step {
			yaml.WriteString(line + "\n")
		}
	}
}

// generateCreateAwInfo generates a step that creates aw_info.json with agentic run metadata
func (c *Compiler) generateCreateAwInfo(yaml *strings.Builder, data *WorkflowData, engine CodingAgentEngine) {
	yaml.WriteString("      - name: Generate agentic run info\n")
	yaml.WriteString("        uses: actions/github-script@v8\n")
	yaml.WriteString("        with:\n")
	yaml.WriteString("          script: |\n")
	yaml.WriteString("            const fs = require('fs');\n")
	yaml.WriteString("            \n")
	yaml.WriteString("            const awInfo = {\n")

	// Engine ID (prefer EngineConfig.ID, fallback to AI field for backwards compatibility)
	engineID := engine.GetID()
	if data.EngineConfig != nil && data.EngineConfig.ID != "" {
		engineID = data.EngineConfig.ID
	} else if data.AI != "" {
		engineID = data.AI
	}
	fmt.Fprintf(yaml, "              engine_id: \"%s\",\n", engineID)

	// Engine display name
	fmt.Fprintf(yaml, "              engine_name: \"%s\",\n", engine.GetDisplayName())

	// Model information
	model := ""
	if data.EngineConfig != nil && data.EngineConfig.Model != "" {
		model = data.EngineConfig.Model
	}
	fmt.Fprintf(yaml, "              model: \"%s\",\n", model)

	// Version information
	version := ""
	if data.EngineConfig != nil && data.EngineConfig.Version != "" {
		version = data.EngineConfig.Version
	}
	fmt.Fprintf(yaml, "              version: \"%s\",\n", version)

	// Workflow information
	fmt.Fprintf(yaml, "              workflow_name: \"%s\",\n", data.Name)
	fmt.Fprintf(yaml, "              experimental: %t,\n", engine.IsExperimental())
	fmt.Fprintf(yaml, "              supports_tools_allowlist: %t,\n", engine.SupportsToolsAllowlist())
	fmt.Fprintf(yaml, "              supports_http_transport: %t,\n", engine.SupportsHTTPTransport())

	// Run metadata
	yaml.WriteString("              run_id: context.runId,\n")
	yaml.WriteString("              run_number: context.runNumber,\n")
	yaml.WriteString("              run_attempt: process.env.GITHUB_RUN_ATTEMPT,\n")
	yaml.WriteString("              repository: context.repo.owner + '/' + context.repo.repo,\n")
	yaml.WriteString("              ref: context.ref,\n")
	yaml.WriteString("              sha: context.sha,\n")
	yaml.WriteString("              actor: context.actor,\n")
	yaml.WriteString("              event_name: context.eventName,\n")

	// Add staged value from safe-outputs configuration
	stagedValue := "false"
	if data.SafeOutputs != nil && data.SafeOutputs.Staged != nil && *data.SafeOutputs.Staged {
		stagedValue = "true"
	}
	fmt.Fprintf(yaml, "              staged: %s,\n", stagedValue)

	yaml.WriteString("              created_at: new Date().toISOString()\n")

	yaml.WriteString("            };\n")
	yaml.WriteString("            \n")
	yaml.WriteString("            // Write to /tmp directory to avoid inclusion in PR\n")
	yaml.WriteString("            const tmpPath = '/tmp/aw_info.json';\n")
	yaml.WriteString("            fs.writeFileSync(tmpPath, JSON.stringify(awInfo, null, 2));\n")
	yaml.WriteString("            console.log('Generated aw_info.json at:', tmpPath);\n")
	yaml.WriteString("            console.log(JSON.stringify(awInfo, null, 2));\n")
	yaml.WriteString("            \n")
	yaml.WriteString("            // Add agentic workflow run information to step summary\n")
	yaml.WriteString("            core.summary\n")
	yaml.WriteString("              .addRaw('## Agentic Run Information\\n\\n')\n")
	yaml.WriteString("              .addRaw('```json\\n')\n")
	yaml.WriteString("              .addRaw(JSON.stringify(awInfo, null, 2))\n")
	yaml.WriteString("              .addRaw('\\n```\\n')\n")
	yaml.WriteString("              .write();\n")
}

// generateOutputFileSetup generates a step that sets up the GITHUB_AW_SAFE_OUTPUTS environment variable
func (c *Compiler) generateOutputFileSetup(yaml *strings.Builder) {
	yaml.WriteString("      - name: Setup agent output\n")
	yaml.WriteString("        id: setup_agent_output\n")
	yaml.WriteString("        uses: actions/github-script@v8\n")
	yaml.WriteString("        with:\n")
	yaml.WriteString("          script: |\n")

	// Use the embedded setup agent output script
	WriteJavaScriptToYAML(yaml, setupAgentOutputScript)
}

func (c *Compiler) generateSafeOutputsConfig(data *WorkflowData) string {
	// Pass the safe-outputs configuration for validation
	if data.SafeOutputs == nil {
		return ""
	}
	// Create a simplified config object for validation
	safeOutputsConfig := make(map[string]any)
	if data.SafeOutputs.CreateIssues != nil {
		safeOutputsConfig["create-issue"] = map[string]any{}
	}
	if data.SafeOutputs.AddComments != nil {
		commentConfig := map[string]any{}
		if data.SafeOutputs.AddComments.Target != "" {
			commentConfig["target"] = data.SafeOutputs.AddComments.Target
		}
		safeOutputsConfig["add-comment"] = commentConfig
	}
	if data.SafeOutputs.CreateDiscussions != nil {
		discussionConfig := map[string]any{}
		if data.SafeOutputs.CreateDiscussions.Max > 0 {
			discussionConfig["max"] = data.SafeOutputs.CreateDiscussions.Max
		}
		safeOutputsConfig["create-discussion"] = discussionConfig
	}
	if data.SafeOutputs.CreatePullRequests != nil {
		safeOutputsConfig["create-pull-request"] = map[string]any{}
	}
	if data.SafeOutputs.CreatePullRequestReviewComments != nil {
		prReviewCommentConfig := map[string]any{}
		if data.SafeOutputs.CreatePullRequestReviewComments.Max > 0 {
			prReviewCommentConfig["max"] = data.SafeOutputs.CreatePullRequestReviewComments.Max
		}
		safeOutputsConfig["create-pull-request-review-comment"] = prReviewCommentConfig
	}
	if data.SafeOutputs.CreateCodeScanningAlerts != nil {
		// Security reports typically have unlimited max, but check if configured
		securityReportConfig := map[string]any{}
		if data.SafeOutputs.CreateCodeScanningAlerts.Max > 0 {
			securityReportConfig["max"] = data.SafeOutputs.CreateCodeScanningAlerts.Max
		}
		safeOutputsConfig["create-code-scanning-alert"] = securityReportConfig
	}
	if data.SafeOutputs.AddLabels != nil {
		safeOutputsConfig["add-labels"] = map[string]any{}
	}
	if data.SafeOutputs.UpdateIssues != nil {
		safeOutputsConfig["update-issue"] = map[string]any{}
	}
	if data.SafeOutputs.PushToPullRequestBranch != nil {
		pushToBranchConfig := map[string]any{}
		if data.SafeOutputs.PushToPullRequestBranch.Target != "" {
			pushToBranchConfig["target"] = data.SafeOutputs.PushToPullRequestBranch.Target
		}
		safeOutputsConfig["push-to-pull-request-branch"] = pushToBranchConfig
	}
	if data.SafeOutputs.UploadAssets != nil {
		safeOutputsConfig["upload-asset"] = map[string]any{}
	}
	if data.SafeOutputs.MissingTool != nil {
		missingToolConfig := map[string]any{}
		if data.SafeOutputs.MissingTool.Max > 0 {
			missingToolConfig["max"] = data.SafeOutputs.MissingTool.Max
		}
		safeOutputsConfig["missing-tool"] = missingToolConfig
	}
	configJSON, _ := json.Marshal(safeOutputsConfig)
	return string(configJSON)
}

// generateOutputCollectionStep generates a step that reads the output file and sets it as a GitHub Actions output
func (c *Compiler) generateOutputCollectionStep(yaml *strings.Builder, data *WorkflowData) {
	yaml.WriteString("      - name: Print Agent output\n")
	yaml.WriteString("        env:\n")
	yaml.WriteString("          GITHUB_AW_SAFE_OUTPUTS: ${{ env.GITHUB_AW_SAFE_OUTPUTS }}\n")
	yaml.WriteString("        run: |\n")
	yaml.WriteString("          echo \"## Agent Output (JSONL)\" >> $GITHUB_STEP_SUMMARY\n")
	yaml.WriteString("          echo \"\" >> $GITHUB_STEP_SUMMARY\n")
	yaml.WriteString("          echo '``````json' >> $GITHUB_STEP_SUMMARY\n")
	yaml.WriteString("          if [ -f ${{ env.GITHUB_AW_SAFE_OUTPUTS }} ]; then\n")
	yaml.WriteString("            cat ${{ env.GITHUB_AW_SAFE_OUTPUTS }} >> $GITHUB_STEP_SUMMARY\n")
	yaml.WriteString("            # Ensure there's a newline after the file content if it doesn't end with one\n")
	yaml.WriteString("            if [ -s ${{ env.GITHUB_AW_SAFE_OUTPUTS }} ] && [ \"$(tail -c1 ${{ env.GITHUB_AW_SAFE_OUTPUTS }})\" != \"\" ]; then\n")
	yaml.WriteString("              echo \"\" >> $GITHUB_STEP_SUMMARY\n")
	yaml.WriteString("            fi\n")
	yaml.WriteString("          else\n")
	yaml.WriteString("            echo \"No agent output file found\" >> $GITHUB_STEP_SUMMARY\n")
	yaml.WriteString("          fi\n")
	yaml.WriteString("          echo '``````' >> $GITHUB_STEP_SUMMARY\n")
	yaml.WriteString("          echo \"\" >> $GITHUB_STEP_SUMMARY\n")

	yaml.WriteString("      - name: Upload agentic output file\n")
	yaml.WriteString("        if: always()\n")
	yaml.WriteString("        uses: actions/upload-artifact@v4\n")
	yaml.WriteString("        with:\n")
	fmt.Fprintf(yaml, "          name: %s\n", OutputArtifactName)
	yaml.WriteString("          path: ${{ env.GITHUB_AW_SAFE_OUTPUTS }}\n")
	yaml.WriteString("          if-no-files-found: warn\n")

	yaml.WriteString("      - name: Ingest agent output\n")
	yaml.WriteString("        id: collect_output\n")
	yaml.WriteString("        uses: actions/github-script@v8\n")

	// Add environment variables for JSONL validation
	yaml.WriteString("        env:\n")
	yaml.WriteString("          GITHUB_AW_SAFE_OUTPUTS: ${{ env.GITHUB_AW_SAFE_OUTPUTS }}\n")

	// Pass the safe-outputs configuration for validation
	safeOutputConfig := c.generateSafeOutputsConfig(data)
	if safeOutputConfig != "" {
		fmt.Fprintf(yaml, "          GITHUB_AW_SAFE_OUTPUTS_CONFIG: %q\n", safeOutputConfig)
	}

	// Add allowed domains configuration for sanitization
	if data.SafeOutputs != nil && len(data.SafeOutputs.AllowedDomains) > 0 {
		domainsStr := strings.Join(data.SafeOutputs.AllowedDomains, ",")
		fmt.Fprintf(yaml, "          GITHUB_AW_ALLOWED_DOMAINS: %q\n", domainsStr)
	}

	yaml.WriteString("        with:\n")
	yaml.WriteString("          script: |\n")

	// Add each line of the script with proper indentation
	WriteJavaScriptToYAML(yaml, collectJSONLOutputScript)

	yaml.WriteString("      - name: Upload sanitized agent output\n")
	yaml.WriteString("        if: always() && env.GITHUB_AW_AGENT_OUTPUT\n")
	yaml.WriteString("        uses: actions/upload-artifact@v4\n")
	yaml.WriteString("        with:\n")
	yaml.WriteString("          name: agent_output.json\n")
	yaml.WriteString("          path: ${{ env.GITHUB_AW_AGENT_OUTPUT }}\n")
	yaml.WriteString("          if-no-files-found: warn\n")

}

// validateHTTPTransportSupport validates that HTTP MCP servers are only used with engines that support HTTP transport
func (c *Compiler) validateHTTPTransportSupport(tools map[string]any, engine CodingAgentEngine) error {
	if engine.SupportsHTTPTransport() {
		// Engine supports HTTP transport, no validation needed
		return nil
	}

	// Engine doesn't support HTTP transport, check for HTTP MCP servers
	for toolName, toolConfig := range tools {
		if config, ok := toolConfig.(map[string]any); ok {
			if hasMcp, mcpType := hasMCPConfig(config); hasMcp && mcpType == "http" {
				return fmt.Errorf("tool '%s' uses HTTP transport which is not supported by engine '%s' (only stdio transport is supported)", toolName, engine.GetID())
			}
		}
	}

	return nil
}

// validateMaxTurnsSupport validates that max-turns is only used with engines that support this feature
func (c *Compiler) validateMaxTurnsSupport(frontmatter map[string]any, engine CodingAgentEngine) error {
	// Check if max-turns is specified in the engine config
	engineSetting, engineConfig := c.extractEngineConfig(frontmatter)
	_ = engineSetting // Suppress unused variable warning

	hasMaxTurns := engineConfig != nil && engineConfig.MaxTurns != ""

	if !hasMaxTurns {
		// No max-turns specified, no validation needed
		return nil
	}

	// max-turns is specified, check if the engine supports it
	if !engine.SupportsMaxTurns() {
		return fmt.Errorf("max-turns not supported: engine '%s' does not support the max-turns feature", engine.GetID())
	}

	// Engine supports max-turns - additional validation could be added here if needed
	// For now, we rely on JSON schema validation for format checking

	return nil
}
