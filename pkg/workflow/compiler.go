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
	// MaxLockFileSize is the maximum allowed size for generated lock workflow files (1MB)
	MaxLockFileSize = 1048576 // 1MB in bytes
)

//go:embed schemas/github-workflow.json
var githubWorkflowSchema string

// FileTracker interface for tracking files created during compilation
type FileTracker interface {
	TrackCreated(filePath string)
}

// Compiler handles converting markdown workflows to GitHub Actions YAML
type Compiler struct {
	verbose         bool
	engineOverride  string
	customOutput    string          // If set, output will be written to this path instead of default location
	version         string          // Version of the extension
	skipValidation  bool            // If true, skip schema validation
	noEmit          bool            // If true, validate without generating lock files
	strictMode      bool            // If true, enforce strict validation requirements
	trialMode       bool            // If true, suppress safe outputs for trial mode execution
	trialTargetRepo string          // If set in trial mode, the target repository to checkout
	jobManager      *JobManager     // Manages jobs and dependencies
	engineRegistry  *EngineRegistry // Registry of available agentic engines
	fileTracker     FileTracker     // Optional file tracker for tracking created files
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

// SetTrialMode configures whether to run in trial mode (suppresses safe outputs)
func (c *Compiler) SetTrialMode(trialMode bool) {
	c.trialMode = trialMode
}

// SetTrialTargetRepo configures the target repository for trial mode
func (c *Compiler) SetTrialTargetRepo(targetRepo string) {
	c.trialTargetRepo = targetRepo
}

// SetStrictMode configures whether to enable strict validation mode
func (c *Compiler) SetStrictMode(strict bool) {
	c.strictMode = strict
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

// BaseSafeOutputConfig holds common configuration fields for all safe output types
type BaseSafeOutputConfig struct {
	Max         int    `yaml:"max,omitempty"`          // Maximum number of items to create
	Min         int    `yaml:"min,omitempty"`          // Minimum number of items to create
	GitHubToken string `yaml:"github-token,omitempty"` // GitHub token for this specific output type
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
	PushToPullRequestBranch         *PushToPullRequestBranchConfig         `yaml:"push-to-pull-request-branch,omitempty"`
	UploadAssets                    *UploadAssetsConfig                    `yaml:"upload-assets,omitempty"`
	MissingTool                     *MissingToolConfig                     `yaml:"missing-tool,omitempty"` // Optional for reporting missing functionality
	Jobs                            map[string]*SafeJobConfig              `yaml:"jobs,omitempty"`         // Safe-jobs configuration (moved from top-level)
	AllowedDomains                  []string                               `yaml:"allowed-domains,omitempty"`
	Staged                          *bool                                  `yaml:"staged,omitempty"`         // If true, emit step summary messages instead of making GitHub API calls
	Env                             map[string]string                      `yaml:"env,omitempty"`            // Environment variables to pass to safe output jobs
	GitHubToken                     string                                 `yaml:"github-token,omitempty"`   // GitHub token for safe output jobs
	MaximumPatchSize                int                                    `yaml:"max-patch-size,omitempty"` // Maximum allowed patch size in KB (defaults to 1024)
	RunsOn                          string                                 `yaml:"runs-on,omitempty"`        // Runner configuration for safe-outputs jobs
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
	workflowData, err := c.ParseWorkflowFile(markdownPath)
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

	// Note: Markdown content size is now handled by splitting into multiple steps in generatePrompt

	if c.verbose {
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
			// Write the invalid YAML to a .invalid.yml file for inspection
			invalidFile := strings.TrimSuffix(lockFile, ".lock.yml") + ".invalid.yml"
			if writeErr := os.WriteFile(invalidFile, []byte(yamlContent), 0644); writeErr == nil {
				fmt.Println(console.FormatWarningMessage(fmt.Sprintf("Invalid workflow YAML written to: %s", console.ToRelativePath(invalidFile))))
			}
			return errors.New(formattedErr)
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

		// Validate file size after writing
		if lockFileInfo, err := os.Stat(lockFile); err == nil {
			if lockFileInfo.Size() > MaxLockFileSize {
				lockSize := pretty.FormatFileSize(lockFileInfo.Size())
				maxSize := pretty.FormatFileSize(MaxLockFileSize)
				formattedErr := console.FormatError(console.CompilerError{
					Position: console.ErrorPosition{
						File:   lockFile,
						Line:   1,
						Column: 1,
					},
					Type:    "error",
					Message: fmt.Sprintf("generated lock file size (%s) exceeds maximum allowed size (%s)", lockSize, maxSize),
				})
				return errors.New(formattedErr)
			}
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

// ParseWorkflowFile parses a markdown workflow file and extracts all necessary data
func (c *Compiler) ParseWorkflowFile(markdownPath string) (*WorkflowData, error) {
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
		fmt.Println(console.FormatInfoMessage(fmt.Sprintf("Frontmatter: %d characters, Markdown content length: %d characters",
			len(result.Frontmatter), len(result.Markdown))))
	}

	markdownDir := filepath.Dir(markdownPath)

	// Process imports from frontmatter if present
	if _, hasImports := result.Frontmatter["imports"]; hasImports {
		if c.verbose {
			fmt.Println(console.FormatInfoMessage("Processing imports..."))
		}

		processedMarkdown, processedFrontmatter, err := parser.ProcessImportsInFrontmatter(
			result.Markdown,
			result.Frontmatter,
			markdownDir,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to process imports: %w", err)
		}

		// Update result with processed content
		result.Markdown = processedMarkdown
		result.Frontmatter = processedFrontmatter

		if c.verbose {
			fmt.Println(console.FormatSuccessMessage("Successfully processed imports"))
		}
	}

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

	// Check if strict mode is enabled in frontmatter
	// If strict is true in frontmatter, enable strict mode for this workflow
	// This allows declarative strict mode control per workflow
	// Note: CLI --strict flag is already set in c.strictMode and takes precedence
	// Frontmatter can enable strict mode, but cannot disable it if CLI flag is set
	if !c.strictMode {
		if strictValue, exists := result.Frontmatter["strict"]; exists {
			if strictBool, ok := strictValue.(bool); ok && strictBool {
				c.strictMode = true
			}
		}
	}

	// Perform strict mode validations
	if err := c.validateStrictMode(result.Frontmatter, networkPermissions); err != nil {
		return nil, err
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

	// Extract mcp-servers from the main file and merge them into tools
	mcpServers := extractMCPServersFromFrontmatter(result.Frontmatter)

	// Process @include directives to extract additional tools
	includedTools, err := parser.ExpandIncludes(result.Markdown, markdownDir, true)
	if err != nil {
		return nil, fmt.Errorf("failed to expand includes for tools: %w", err)
	}

	// Merge tools including mcp-servers
	tools, err = c.mergeToolsAndMCPServers(topTools, mcpServers, includedTools)

	if err != nil {
		return nil, fmt.Errorf("failed to merge tools: %w", err)
	}

	// Add MCP fetch server if needed (when web-fetch is requested but engine doesn't support it)
	tools, _ = AddMCPFetchServerIfNeeded(tools, agenticEngine)

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

	// Validate web-search support for the current engine
	if err := c.validateWebSearchSupport(tools, agenticEngine); err != nil {
		return nil, fmt.Errorf("web-search not supported: %w", err)
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

	// Extract safe-jobs from the new location (safe-outputs.jobs) or old location (safe-jobs) for backwards compatibility
	topSafeJobs := extractSafeJobsFromFrontmatter(result.Frontmatter)

	// Process @include directives to extract additional safe-jobs (reuse the same includedTools JSON)
	// Since ExpandIncludes extracts all frontmatter as JSON, we can use the same result
	includedSafeJobs, err := c.mergeSafeJobsFromIncludes(topSafeJobs, includedTools)
	if err != nil {
		return nil, fmt.Errorf("failed to merge safe-jobs from includes: %w", err)
	}

	// Ensure SafeOutputs exists and populate the Jobs field
	if workflowData.SafeOutputs == nil && len(includedSafeJobs) > 0 {
		workflowData.SafeOutputs = &SafeOutputsConfig{}
	}
	if workflowData.SafeOutputs != nil && len(workflowData.SafeOutputs.Jobs) == 0 && len(includedSafeJobs) > 0 {
		workflowData.SafeOutputs.Jobs = includedSafeJobs
	}

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

	// Clean up quoted keys - replace "key": with key: at the start of a line
	// This handles cases where YAML marshaling adds unnecessary quotes around reserved words like "on"
	yamlStr = unquoteYAMLKey(yamlStr, key)

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

	// Auto-enable "eyes" reaction for command triggers if no explicit reaction was specified
	if hasCommand && !hasReaction && workflowData.AIReaction == "" {
		workflowData.AIReaction = "eyes"
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
			yamlStr := strings.TrimSuffix(string(onEventsYAML), "\n")

			// Clean up quoted keys - replace "on": with on: at the start of a line
			// This handles cases where YAML marshaling adds unnecessary quotes around reserved words like "on"
			yamlStr = unquoteYAMLKey(yamlStr, "on")

			workflowData.On = yamlStr
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
	// Check new format: on.command or on.command.name
	if onValue, exists := frontmatter["on"]; exists {
		if onMap, ok := onValue.(map[string]any); ok {
			if commandValue, hasCommand := onMap["command"]; hasCommand {
				// Check if command is a string (shorthand format)
				if commandStr, ok := commandValue.(string); ok {
					return commandStr
				}
				// Check if command is a map with a name key (object format)
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

// mergeSafeJobsFromIncludes merges safe-jobs from included files and detects conflicts
func (c *Compiler) mergeSafeJobsFromIncludes(topSafeJobs map[string]*SafeJobConfig, includedContentJSON string) (map[string]*SafeJobConfig, error) {
	if includedContentJSON == "" || includedContentJSON == "{}" {
		return topSafeJobs, nil
	}

	// Parse the included content as frontmatter to extract safe-jobs
	var includedContent map[string]any
	if err := json.Unmarshal([]byte(includedContentJSON), &includedContent); err != nil {
		return topSafeJobs, nil // Return original safe-jobs if parsing fails
	}

	// Extract safe-jobs from the included content
	includedSafeJobs := extractSafeJobsFromFrontmatter(includedContent)

	// Merge with conflict detection
	mergedSafeJobs, err := mergeSafeJobs(topSafeJobs, includedSafeJobs)
	if err != nil {
		return nil, fmt.Errorf("failed to merge safe-jobs: %w", err)
	}

	return mergedSafeJobs, nil
}

// applyDefaultTools adds default read-only GitHub MCP tools, creating github tool if not present
func (c *Compiler) applyDefaultTools(tools map[string]any, safeOutputs *SafeOutputsConfig) map[string]any {
	// Always apply default GitHub tools (create github section if it doesn't exist)

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

	// Add default GitHub tools that aren't already present
	newAllowed := make([]any, len(existingAllowed))
	copy(newAllowed, existingAllowed)

	for _, defaultTool := range constants.DefaultGitHubTools {
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

	// Add default bash commands when bash is enabled but no specific commands are provided
	// This runs after git commands logic, so it only applies when git commands weren't added
	// Behavior:
	//   - bash: true or bash: nil → Add default commands
	//   - bash: [] → No commands (empty array means no tools allowed)
	//   - bash: ["cmd1", "cmd2"] → Add default commands + specific commands
	if bashTool, exists := tools["bash"]; exists {
		// Check if bash was left as nil or true after git processing
		if bashTool == nil {
			// bash is nil - only add defaults if this wasn't processed by git commands
			// If git commands were needed, bash would have been set to git commands or left as nil intentionally
			if !(safeOutputs != nil && needsGitCommands(safeOutputs)) {
				defaultCommands := make([]any, len(constants.DefaultBashTools))
				for i, cmd := range constants.DefaultBashTools {
					defaultCommands[i] = cmd
				}
				tools["bash"] = defaultCommands
			}
		} else if bashTool == true {
			// bash is true - always add default commands
			defaultCommands := make([]any, len(constants.DefaultBashTools))
			for i, cmd := range constants.DefaultBashTools {
				defaultCommands[i] = cmd
			}
			tools["bash"] = defaultCommands
		} else if bashArray, ok := bashTool.([]any); ok {
			// bash is an array - merge default commands with custom commands
			if len(bashArray) > 0 {
				// Create a set to track existing commands to avoid duplicates
				existingCommands := make(map[string]bool)
				for _, cmd := range bashArray {
					if cmdStr, ok := cmd.(string); ok {
						existingCommands[cmdStr] = true
					}
				}

				// Start with default commands
				mergedCommands := make([]any, 0, len(constants.DefaultBashTools)+len(bashArray))
				for _, cmd := range constants.DefaultBashTools {
					if !existingCommands[cmd] {
						mergedCommands = append(mergedCommands, cmd)
					}
				}

				// Add the custom commands
				mergedCommands = append(mergedCommands, bashArray...)
				tools["bash"] = mergedCommands
			}
			// Note: bash with empty array (bash: []) means "no bash tools allowed" and is left as-is
		}
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

// detectTextOutputUsage checks if the markdown content uses ${{ needs.activation.outputs.text }}
func (c *Compiler) detectTextOutputUsage(markdownContent string) bool {
	// Check for the specific GitHub Actions expression
	hasUsage := strings.Contains(markdownContent, "${{ needs.activation.outputs.text }}")
	if c.verbose {
		if hasUsage {
			fmt.Println(console.FormatInfoMessage("Detected usage of activation.outputs.text - compute-text step will be included"))
		} else {
			fmt.Println(console.FormatInfoMessage("No usage of activation.outputs.text found - compute-text step will be skipped"))
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

// isActivationJobNeeded determines if the activation job is required
func (c *Compiler) isActivationJobNeeded(data *WorkflowData, needsPermissionCheck bool) bool {
	// Activation job is needed if:
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

	// Main job ID is always constants.AgentJobName

	// Build check-membership job if needed (validates team membership levels)
	// Team membership checks are specifically for command workflows
	// Non-command workflows use general role checks instead
	needsPermissionCheck := c.needsRoleCheck(data, frontmatter)

	if needsPermissionCheck {
		checkMembershipJob, err := c.buildCheckMembershipJob(data, frontmatter)
		if err != nil {
			return fmt.Errorf("failed to build check-membership job: %w", err)
		}
		if err := c.jobManager.AddJob(checkMembershipJob); err != nil {
			return fmt.Errorf("failed to add check-membership job: %w", err)
		}
	}

	// Build activation job if needed (preamble job that handles runtime conditions)
	// If check-membership job exists, activation job is ALWAYS created and depends on it
	var activationJobCreated bool

	if c.isActivationJobNeeded(data, needsPermissionCheck) {
		activationJob, err := c.buildActivationJob(data, needsPermissionCheck)
		if err != nil {
			return fmt.Errorf("failed to build activation job: %w", err)
		}
		if err := c.jobManager.AddJob(activationJob); err != nil {
			return fmt.Errorf("failed to add activation job: %w", err)
		}
		activationJobCreated = true
	}

	// Build add_reaction job only if ai-reaction is configured
	if data.AIReaction != "" {
		addReactionJob, err := c.buildAddReactionJob(data, activationJobCreated, frontmatter)
		if err != nil {
			return fmt.Errorf("failed to build add_reaction job: %w", err)
		}
		if err := c.jobManager.AddJob(addReactionJob); err != nil {
			return fmt.Errorf("failed to add add_reaction job: %w", err)
		}
	}

	// Build stop-time check job if stop-time is configured
	if data.StopTime != "" {
		stopTimeCheckJob, err := c.buildStopTimeCheckJob(data, activationJobCreated)
		if err != nil {
			return fmt.Errorf("failed to build stop_time_check job: %w", err)
		}
		if err := c.jobManager.AddJob(stopTimeCheckJob); err != nil {
			return fmt.Errorf("failed to add stop_time_check job: %w", err)
		}
	}

	// Build main workflow job
	mainJob, err := c.buildMainJob(data, activationJobCreated)
	if err != nil {
		return fmt.Errorf("failed to build main job: %w", err)
	}
	if err := c.jobManager.AddJob(mainJob); err != nil {
		return fmt.Errorf("failed to add main job: %w", err)
	}

	// Build safe outputs jobs if configured (skip in trial mode)
	if !c.trialMode {
		if err := c.buildSafeOutputsJobs(data, constants.AgentJobName, activationJobCreated, frontmatter, markdownPath); err != nil {
			return fmt.Errorf("failed to build safe outputs jobs: %w", err)
		}
	}

	// Build safe-jobs if configured (skip in trial mode)
	if !c.trialMode {
		if err := c.buildSafeJobs(data); err != nil {
			return fmt.Errorf("failed to build safe-jobs: %w", err)
		}
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

	// Build add_comment job if output.add-comment is configured
	if data.SafeOutputs.AddComments != nil {
		createCommentJob, err := c.buildCreateOutputAddCommentJob(data, jobName)
		if err != nil {
			return fmt.Errorf("failed to build add_comment job: %w", err)
		}
		if err := c.jobManager.AddJob(createCommentJob); err != nil {
			return fmt.Errorf("failed to add add_comment job: %w", err)
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

// buildCheckMembershipJob creates the check-membership job that validates team membership levels
func (c *Compiler) buildCheckMembershipJob(data *WorkflowData, frontmatter map[string]any) (*Job, error) {
	outputs := map[string]string{
		"is_team_member":  "${{ steps.check-membership.outputs.is_team_member }}",
		"result":          "${{ steps.check-membership.outputs.result }}",
		"user_permission": "${{ steps.check-membership.outputs.user_permission }}",
		"error_message":   "${{ steps.check-membership.outputs.error_message }}",
	}
	var steps []string

	// Add team member check that only sets outputs
	steps = c.generateMembershipCheck(data, steps)

	job := &Job{
		Name:        "check-membership",
		If:          data.If, // Use the existing condition (which may include alias checks)
		RunsOn:      c.formatSafeOutputsRunsOn(data.SafeOutputs),
		Permissions: "", // No special permissions needed - just reading repo permissions
		Steps:       steps,
		Outputs:     outputs,
	}

	return job, nil
}

// buildActivationJob creates the preamble activation job that acts as a barrier for runtime conditions
func (c *Compiler) buildActivationJob(data *WorkflowData, checkMembershipJobCreated bool) (*Job, error) {
	outputs := map[string]string{}
	var steps []string

	// Team member check is now handled by the separate check-membership job
	// No inline role checks needed in the task job anymore

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
	// This can happen when the activation job is created only for an if condition
	if len(steps) == 0 {
		steps = append(steps, "      - run: echo \"Activation success\"\n")
	}

	// Build the conditional expression that validates membership and other conditions
	var activationNeeds []string
	var activationCondition string

	if checkMembershipJobCreated {
		// Activation job is the only job that can rely on check-membership
		activationNeeds = []string{"check-membership"}
		membershipExpr := BuildEquals(
			BuildPropertyAccess("needs.check-membership.outputs.is_team_member"),
			BuildStringLiteral("true"),
		)
		if data.If != "" {
			ifExpr := &ExpressionNode{Expression: data.If}
			combinedExpr := &AndNode{Left: membershipExpr, Right: ifExpr}
			activationCondition = combinedExpr.Render()
		} else {
			activationCondition = membershipExpr.Render()
		}
	} else {
		// No membership check needed
		activationCondition = data.If
	}

	// No special permissions needed since role checks are handled by separate job
	var permissions string

	job := &Job{
		Name:        "activation",
		If:          activationCondition,
		RunsOn:      c.formatSafeOutputsRunsOn(data.SafeOutputs),
		Permissions: permissions,
		Steps:       steps,
		Outputs:     outputs,
		Needs:       activationNeeds, // Depend on check-membership job if it exists
	}

	return job, nil
}

// buildMainJob creates the main workflow job
func (c *Compiler) buildMainJob(data *WorkflowData, activationJobCreated bool) (*Job, error) {
	var steps []string

	var jobCondition = data.If
	if activationJobCreated {
		jobCondition = "" // Main job depends on activation job, so no need for inline condition
	}
	// Permission checks are now handled by the separate check-membership job
	// No role checks needed in the main job

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
	if activationJobCreated {
		depends = []string{"activation"} // Depend on the activation job only if it exists
	}

	// Build outputs for all engines (GITHUB_AW_SAFE_OUTPUTS functionality)
	// Only include output if the workflow actually uses the safe-outputs feature
	var outputs map[string]string
	if data.SafeOutputs != nil {
		outputs = map[string]string{
			"output":       "${{ steps.collect_output.outputs.output }}",
			"output_types": "${{ steps.collect_output.outputs.output_types }}",
		}
	}

	// Build job-level environment variables for safe outputs
	var env map[string]string
	if data.SafeOutputs != nil {
		env = make(map[string]string)

		// Set GITHUB_AW_SAFE_OUTPUTS to fixed path
		env["GITHUB_AW_SAFE_OUTPUTS"] = "/tmp/safe-outputs/outputs.jsonl"

		// Set GITHUB_AW_SAFE_OUTPUTS_CONFIG with the safe outputs configuration
		safeOutputConfig := c.generateSafeOutputsConfig(data)
		if safeOutputConfig != "" {
			// The JSON string needs to be properly quoted for YAML
			env["GITHUB_AW_SAFE_OUTPUTS_CONFIG"] = fmt.Sprintf("%q", safeOutputConfig)
		}
	}

	job := &Job{
		Name:        constants.AgentJobName,
		If:          jobCondition,
		RunsOn:      c.indentYAMLLines(data.RunsOn, "    "),
		Permissions: c.indentYAMLLines(data.Permissions, "    "),
		Env:         env,
		Steps:       steps,
		Needs:       depends,
		Outputs:     outputs,
	}

	return job, nil
}

// generateMainJobSteps generates the steps section for the main job
func (c *Compiler) generateMainJobSteps(yaml *strings.Builder, data *WorkflowData) {
	// Determine if we need to add a checkout step
	needsCheckout := c.shouldAddCheckoutStep(data)

	// Add checkout step first if needed
	if needsCheckout {
		yaml.WriteString("      - name: Checkout repository\n")
		yaml.WriteString("        uses: actions/checkout@v5\n")
		if c.trialMode {
			yaml.WriteString("        with:\n")
			if c.trialTargetRepo != "" {
				yaml.WriteString(fmt.Sprintf("          repository: %s\n", c.trialTargetRepo))
			}
			yaml.WriteString("          github-token: ${{ secrets.GH_AW_GITHUB_TOKEN || secrets.GITHUB_TOKEN }}\n")
		}

		// Add step to checkout PR branch if the event is a comment on a PR
		c.generatePRBranchCheckout(yaml, data)
	}

	// Add custom steps if present
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
	}

	// Add cache steps if cache configuration is present
	generateCacheSteps(yaml, data, c.verbose)

	// Add cache-memory steps if cache-memory configuration is present
	generateCacheMemorySteps(yaml, data, c.verbose)

	// Configure git credentials if git operations will be needed
	// Note: Git configuration is handled by token in checkout step when in trial mode

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

	// GITHUB_AW_SAFE_OUTPUTS is now set at job level, no setup step needed

	// Add MCP setup
	c.generateMCPSetup(yaml, data.Tools, engine, data)

	// Stop-time safety checks are now handled by a dedicated job (stop_time_check)
	// No longer generated in the main job steps

	// Add prompt creation step
	c.generatePrompt(yaml, data)

	logFile := "agent-stdio"
	logFileFull := "/tmp/agent-stdio.log"

	// Capture agent version if engine supports it
	c.generateAgentVersionCapture(yaml, engine)

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
	yaml.WriteString("      - name: Upload Agent Stdio\n")
	yaml.WriteString("        if: always()\n")
	yaml.WriteString("        uses: actions/upload-artifact@v4\n")
	yaml.WriteString("        with:\n")
	yaml.WriteString("          name: agent-stdio.log\n")
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

// convertGoPatternToJavaScript converts a Go regex pattern to JavaScript-compatible format
// This removes Go's (?i) inline case-insensitive flag since JavaScript doesn't support it
// The original JavaScript code will use the pattern as-is with "g" flags
func (c *Compiler) convertGoPatternToJavaScript(goPattern string) string {
	// Convert (?i) inline case-insensitive flag by removing it
	// JavaScript RegExp will be created with "gi" flags to handle case insensitivity
	if strings.HasPrefix(goPattern, "(?i)") {
		return goPattern[4:] // Remove (?i) prefix
	}
	return goPattern
}

// convertErrorPatternsToJavaScript converts a slice of Go error patterns to JavaScript-compatible patterns
func (c *Compiler) convertErrorPatternsToJavaScript(goPatterns []ErrorPattern) []ErrorPattern {
	jsPatterns := make([]ErrorPattern, len(goPatterns))
	for i, pattern := range goPatterns {
		jsPatterns[i] = ErrorPattern{
			Pattern:      c.convertGoPatternToJavaScript(pattern.Pattern),
			LevelGroup:   pattern.LevelGroup,
			MessageGroup: pattern.MessageGroup,
			Description:  pattern.Description,
		}
	}
	return jsPatterns
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

	// Convert Go regex patterns to JavaScript-compatible patterns
	jsCompatiblePatterns := c.convertErrorPatternsToJavaScript(errorPatterns)

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

	// Add JavaScript-compatible error patterns as a single JSON array
	patternsJSON, err := json.Marshal(jsCompatiblePatterns)
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

// validateMarkdownSizeForGitHubActions is no longer used - content is now split into multiple steps
// to handle GitHub Actions script size limits automatically
// func (c *Compiler) validateMarkdownSizeForGitHubActions(content string) error { ... }

// splitContentIntoChunks splits markdown content into chunks that fit within GitHub Actions script size limits
func splitContentIntoChunks(content string) []string {
	const maxChunkSize = 20900        // 21000 - 100 character buffer
	const indentSpaces = "          " // 10 spaces added to each line

	lines := strings.Split(content, "\n")
	var chunks []string
	var currentChunk []string
	currentSize := 0

	for _, line := range lines {
		lineSize := len(indentSpaces) + len(line) + 1 // +1 for newline

		// If adding this line would exceed the limit, start a new chunk
		if currentSize+lineSize > maxChunkSize && len(currentChunk) > 0 {
			chunks = append(chunks, strings.Join(currentChunk, "\n"))
			currentChunk = []string{line}
			currentSize = lineSize
		} else {
			currentChunk = append(currentChunk, line)
			currentSize += lineSize
		}
	}

	// Add the last chunk if there's content
	if len(currentChunk) > 0 {
		chunks = append(chunks, strings.Join(currentChunk, "\n"))
	}

	return chunks
}

func (c *Compiler) generatePrompt(yaml *strings.Builder, data *WorkflowData) {
	// Clean the markdown content
	cleanedMarkdownContent := removeXMLComments(data.MarkdownContent)

	// Split content into manageable chunks
	chunks := splitContentIntoChunks(cleanedMarkdownContent)

	// Create the initial prompt file step
	yaml.WriteString("      - name: Create prompt\n")
	yaml.WriteString("        env:\n")
	yaml.WriteString("          GITHUB_AW_PROMPT: /tmp/aw-prompts/prompt.txt\n")
	if data.SafeOutputs != nil {
		yaml.WriteString("          GITHUB_AW_SAFE_OUTPUTS: ${{ env.GITHUB_AW_SAFE_OUTPUTS }}\n")
	}
	yaml.WriteString("        run: |\n")
	yaml.WriteString("          mkdir -p $(dirname \"$GITHUB_AW_PROMPT\")\n")

	if len(chunks) > 0 {
		yaml.WriteString("          cat > $GITHUB_AW_PROMPT << 'EOF'\n")
		for _, line := range strings.Split(chunks[0], "\n") {
			yaml.WriteString("          " + line + "\n")
		}
		yaml.WriteString("          EOF\n")
	} else {
		yaml.WriteString("          touch $GITHUB_AW_PROMPT\n")
	}

	// Create additional steps for remaining chunks
	for i, chunk := range chunks[1:] {
		stepNum := i + 2
		yaml.WriteString(fmt.Sprintf("      - name: Append prompt (part %d)\n", stepNum))
		yaml.WriteString("        env:\n")
		yaml.WriteString("          GITHUB_AW_PROMPT: /tmp/aw-prompts/prompt.txt\n")
		yaml.WriteString("        run: |\n")
		yaml.WriteString("          cat >> $GITHUB_AW_PROMPT << 'EOF'\n")
		for _, line := range strings.Split(chunk, "\n") {
			yaml.WriteString("          " + line + "\n")
		}
		yaml.WriteString("          EOF\n")
	}

	// Add cache memory prompt as separate step if enabled
	c.generateCacheMemoryPromptStep(yaml, data.CacheMemoryConfig)

	// Add safe outputs prompt as separate step if enabled
	c.generateSafeOutputsPromptStep(yaml, data.SafeOutputs)

	// Add PR context prompt as separate step if enabled
	c.generatePRContextPromptStep(yaml, data)

	// Add step to print prompt to GitHub step summary for debugging
	yaml.WriteString("      - name: Print prompt to step summary\n")
	yaml.WriteString("        env:\n")
	yaml.WriteString("          GITHUB_AW_PROMPT: /tmp/aw-prompts/prompt.txt\n")
	yaml.WriteString("        run: |\n")
	yaml.WriteString("          echo \"## Generated Prompt\" >> $GITHUB_STEP_SUMMARY\n")
	yaml.WriteString("          echo \"\" >> $GITHUB_STEP_SUMMARY\n")
	yaml.WriteString("          echo '``````markdown' >> $GITHUB_STEP_SUMMARY\n")
	yaml.WriteString("          cat $GITHUB_AW_PROMPT >> $GITHUB_STEP_SUMMARY\n")
	yaml.WriteString("          echo '``````' >> $GITHUB_STEP_SUMMARY\n")
}

// generateCacheMemoryPromptStep generates a separate step for cache memory prompt section
func (c *Compiler) generateCacheMemoryPromptStep(yaml *strings.Builder, config *CacheMemoryConfig) {
	if config == nil || !config.Enabled {
		return
	}

	yaml.WriteString("      - name: Append cache memory instructions to prompt\n")
	yaml.WriteString("        env:\n")
	yaml.WriteString("          GITHUB_AW_PROMPT: /tmp/aw-prompts/prompt.txt\n")
	yaml.WriteString("        run: |\n")
	yaml.WriteString("          cat >> $GITHUB_AW_PROMPT << 'EOF'\n")
	generateCacheMemoryPromptSection(yaml, config)
	yaml.WriteString("          EOF\n")
}

// generateSafeOutputsPromptStep generates a separate step for safe outputs prompt section
func (c *Compiler) generateSafeOutputsPromptStep(yaml *strings.Builder, safeOutputs *SafeOutputsConfig) {
	if safeOutputs == nil {
		return
	}

	yaml.WriteString("      - name: Append safe outputs instructions to prompt\n")
	yaml.WriteString("        env:\n")
	yaml.WriteString("          GITHUB_AW_PROMPT: /tmp/aw-prompts/prompt.txt\n")
	yaml.WriteString("        run: |\n")
	yaml.WriteString("          cat >> $GITHUB_AW_PROMPT << 'EOF'\n")
	generateSafeOutputsPromptSection(yaml, safeOutputs)
	yaml.WriteString("          EOF\n")
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

// shouldAddCheckoutStep determines if the checkout step should be added based on permissions and custom steps
func (c *Compiler) shouldAddCheckoutStep(data *WorkflowData) bool {
	// Check condition 1: If custom steps already contain checkout, don't add another one
	if data.CustomSteps != "" && ContainsCheckout(data.CustomSteps) {
		return false // Custom steps already have checkout
	}

	// Check condition 2: If permissions don't grant contents access, don't add checkout
	permParser := NewPermissionsParser(data.Permissions)
	if !permParser.HasContentsReadAccess() {
		return false // No contents read access, so checkout is not needed
	}

	// If we get here, permissions allow contents access and custom steps (if any) don't contain checkout
	return true // Add checkout because it's needed and not already present
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

// generateAgentVersionCapture generates a step that captures the agent version if the engine supports it
func (c *Compiler) generateAgentVersionCapture(yaml *strings.Builder, engine CodingAgentEngine) {
	versionCmd := engine.GetVersionCommand()
	if versionCmd == "" {
		// Engine doesn't support version reporting, set empty env var
		yaml.WriteString("      - name: Set agent version (not available)\n")
		yaml.WriteString("        run: echo \"AGENT_VERSION=\" >> $GITHUB_ENV\n")
		return
	}

	yaml.WriteString("      - name: Capture agent version\n")
	yaml.WriteString("        run: |\n")
	fmt.Fprintf(yaml, "          VERSION_OUTPUT=$(%s 2>&1 || echo \"unknown\")\n", versionCmd)
	fmt.Fprintf(yaml, "          # Extract semantic version pattern (e.g., 1.2.3, v1.2.3-beta)\n")
	fmt.Fprintf(yaml, "          CLEAN_VERSION=$(echo \"$VERSION_OUTPUT\" | grep -oE 'v?[0-9]+\\.[0-9]+\\.[0-9]+(-[a-zA-Z0-9]+)?' | head -n1 || echo \"unknown\")\n")
	yaml.WriteString("          echo \"AGENT_VERSION=$CLEAN_VERSION\" >> $GITHUB_ENV\n")
	yaml.WriteString("          echo \"Agent version: $VERSION_OUTPUT\"\n")
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

	// Agent version captured from running version command
	yaml.WriteString("              agent_version: process.env.AGENT_VERSION || \"\",\n")

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

func (c *Compiler) generateSafeOutputsConfig(data *WorkflowData) string {
	// Pass the safe-outputs configuration for validation
	if data.SafeOutputs == nil {
		return ""
	}
	// Create a simplified config object for validation
	safeOutputsConfig := make(map[string]any)

	// Handle safe-outputs configuration if present
	if data.SafeOutputs != nil {
		if data.SafeOutputs.CreateIssues != nil {
			issueConfig := map[string]any{}
			if data.SafeOutputs.CreateIssues.Max > 0 {
				issueConfig["max"] = data.SafeOutputs.CreateIssues.Max
			}
			if data.SafeOutputs.CreateIssues.Min > 0 {
				issueConfig["min"] = data.SafeOutputs.CreateIssues.Min
			}
			safeOutputsConfig["create-issue"] = issueConfig
		}
		if data.SafeOutputs.AddComments != nil {
			commentConfig := map[string]any{}
			if data.SafeOutputs.AddComments.Target != "" {
				commentConfig["target"] = data.SafeOutputs.AddComments.Target
			}
			if data.SafeOutputs.AddComments.Max > 0 {
				commentConfig["max"] = data.SafeOutputs.AddComments.Max
			}
			if data.SafeOutputs.AddComments.Min > 0 {
				commentConfig["min"] = data.SafeOutputs.AddComments.Min
			}
			safeOutputsConfig["add-comment"] = commentConfig
		}
		if data.SafeOutputs.CreateDiscussions != nil {
			discussionConfig := map[string]any{}
			if data.SafeOutputs.CreateDiscussions.Max > 0 {
				discussionConfig["max"] = data.SafeOutputs.CreateDiscussions.Max
			}
			if data.SafeOutputs.CreateDiscussions.Min > 0 {
				discussionConfig["min"] = data.SafeOutputs.CreateDiscussions.Min
			}
			safeOutputsConfig["create-discussion"] = discussionConfig
		}
		if data.SafeOutputs.CreatePullRequests != nil {
			prConfig := map[string]any{}
			// Note: max is always 1 for pull requests, not configurable
			if data.SafeOutputs.CreatePullRequests.Min > 0 {
				prConfig["min"] = data.SafeOutputs.CreatePullRequests.Min
			}
			safeOutputsConfig["create-pull-request"] = prConfig
		}
		if data.SafeOutputs.CreatePullRequestReviewComments != nil {
			prReviewCommentConfig := map[string]any{}
			if data.SafeOutputs.CreatePullRequestReviewComments.Max > 0 {
				prReviewCommentConfig["max"] = data.SafeOutputs.CreatePullRequestReviewComments.Max
			}
			if data.SafeOutputs.CreatePullRequestReviewComments.Min > 0 {
				prReviewCommentConfig["min"] = data.SafeOutputs.CreatePullRequestReviewComments.Min
			}
			safeOutputsConfig["create-pull-request-review-comment"] = prReviewCommentConfig
		}
		if data.SafeOutputs.CreateCodeScanningAlerts != nil {
			// Security reports typically have unlimited max, but check if configured
			securityReportConfig := map[string]any{}
			if data.SafeOutputs.CreateCodeScanningAlerts.Max > 0 {
				securityReportConfig["max"] = data.SafeOutputs.CreateCodeScanningAlerts.Max
			}
			if data.SafeOutputs.CreateCodeScanningAlerts.Min > 0 {
				securityReportConfig["min"] = data.SafeOutputs.CreateCodeScanningAlerts.Min
			}
			safeOutputsConfig["create-code-scanning-alert"] = securityReportConfig
		}
		if data.SafeOutputs.AddLabels != nil {
			labelConfig := map[string]any{}
			if data.SafeOutputs.AddLabels.Max > 0 {
				labelConfig["max"] = data.SafeOutputs.AddLabels.Max
			}
			if data.SafeOutputs.AddLabels.Min > 0 {
				labelConfig["min"] = data.SafeOutputs.AddLabels.Min
			}
			safeOutputsConfig["add-labels"] = labelConfig
		}
		if data.SafeOutputs.UpdateIssues != nil {
			updateConfig := map[string]any{}
			if data.SafeOutputs.UpdateIssues.Max > 0 {
				updateConfig["max"] = data.SafeOutputs.UpdateIssues.Max
			}
			if data.SafeOutputs.UpdateIssues.Min > 0 {
				updateConfig["min"] = data.SafeOutputs.UpdateIssues.Min
			}
			safeOutputsConfig["update-issue"] = updateConfig
		}
		if data.SafeOutputs.PushToPullRequestBranch != nil {
			pushToBranchConfig := map[string]any{}
			if data.SafeOutputs.PushToPullRequestBranch.Target != "" {
				pushToBranchConfig["target"] = data.SafeOutputs.PushToPullRequestBranch.Target
			}
			if data.SafeOutputs.PushToPullRequestBranch.Max > 0 {
				pushToBranchConfig["max"] = data.SafeOutputs.PushToPullRequestBranch.Max
			}
			if data.SafeOutputs.PushToPullRequestBranch.Min > 0 {
				pushToBranchConfig["min"] = data.SafeOutputs.PushToPullRequestBranch.Min
			}
			safeOutputsConfig["push-to-pull-request-branch"] = pushToBranchConfig
		}
		if data.SafeOutputs.UploadAssets != nil {
			uploadConfig := map[string]any{}
			if data.SafeOutputs.UploadAssets.Max > 0 {
				uploadConfig["max"] = data.SafeOutputs.UploadAssets.Max
			}
			if data.SafeOutputs.UploadAssets.Min > 0 {
				uploadConfig["min"] = data.SafeOutputs.UploadAssets.Min
			}
			safeOutputsConfig["upload-asset"] = uploadConfig
		}
		if data.SafeOutputs.MissingTool != nil {
			missingToolConfig := map[string]any{}
			if data.SafeOutputs.MissingTool.Max > 0 {
				missingToolConfig["max"] = data.SafeOutputs.MissingTool.Max
			}
			if data.SafeOutputs.MissingTool.Min > 0 {
				missingToolConfig["min"] = data.SafeOutputs.MissingTool.Min
			}
			safeOutputsConfig["missing-tool"] = missingToolConfig
		}
	}

	// Add safe-jobs configuration from SafeOutputs.Jobs
	if len(data.SafeOutputs.Jobs) > 0 {
		for jobName, jobConfig := range data.SafeOutputs.Jobs {
			safeJobConfig := map[string]any{}

			// Add inputs information
			if len(jobConfig.Inputs) > 0 {
				inputsConfig := make(map[string]any)
				for inputName, inputDef := range jobConfig.Inputs {
					inputConfig := map[string]any{
						"type":        inputDef.Type,
						"description": inputDef.Description,
						"required":    inputDef.Required,
					}
					if inputDef.Default != "" {
						inputConfig["default"] = inputDef.Default
					}
					if len(inputDef.Options) > 0 {
						inputConfig["options"] = inputDef.Options
					}
					inputsConfig[inputName] = inputConfig
				}
				safeJobConfig["inputs"] = inputsConfig
			}

			safeOutputsConfig[jobName] = safeJobConfig
		}
	}

	configJSON, _ := json.Marshal(safeOutputsConfig)
	return string(configJSON)
}

// generateOutputCollectionStep generates a step that reads the output file and sets it as a GitHub Actions output
func (c *Compiler) generateOutputCollectionStep(yaml *strings.Builder, data *WorkflowData) {
	yaml.WriteString("      - name: Print Safe Outputs\n")
	yaml.WriteString("        env:\n")
	yaml.WriteString("          GITHUB_AW_SAFE_OUTPUTS: ${{ env.GITHUB_AW_SAFE_OUTPUTS }}\n")
	yaml.WriteString("        run: |\n")
	yaml.WriteString("          echo \"## Safe Outputs (JSONL)\" >> $GITHUB_STEP_SUMMARY\n")
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

	yaml.WriteString("      - name: Upload Safe Outputs\n")
	yaml.WriteString("        if: always()\n")
	yaml.WriteString("        uses: actions/upload-artifact@v4\n")
	yaml.WriteString("        with:\n")
	fmt.Fprintf(yaml, "          name: %s\n", constants.SafeOutputArtifactName)
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

// validateWebSearchSupport validates that web-search tool is only used with engines that support this feature
func (c *Compiler) validateWebSearchSupport(tools map[string]any, engine CodingAgentEngine) error {
	// Check if web-search tool is requested
	_, hasWebSearch := tools["web-search"]

	if !hasWebSearch {
		// No web-search specified, no validation needed
		return nil
	}

	// web-search is specified, check if the engine supports it
	if !engine.SupportsWebSearch() {
		return fmt.Errorf("engine '%s' does not support the web-search tool", engine.GetID())
	}

	return nil
}

// parseBaseSafeOutputConfig parses common fields (max, min, github-token) from a config map
func (c *Compiler) parseBaseSafeOutputConfig(configMap map[string]any, config *BaseSafeOutputConfig) {
	// Parse max
	if max, exists := configMap["max"]; exists {
		if maxInt, ok := parseIntValue(max); ok {
			config.Max = maxInt
		}
	}

	// Parse min
	if min, exists := configMap["min"]; exists {
		if minInt, ok := parseIntValue(min); ok {
			config.Min = minInt
		}
	}

	// Parse github-token
	if githubToken, exists := configMap["github-token"]; exists {
		if githubTokenStr, ok := githubToken.(string); ok {
			config.GitHubToken = githubTokenStr
		}
	}
}
