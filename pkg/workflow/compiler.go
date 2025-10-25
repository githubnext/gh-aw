package workflow

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/githubnext/gh-aw/pkg/console"
	"github.com/githubnext/gh-aw/pkg/constants"
	"github.com/githubnext/gh-aw/pkg/logger"
	"github.com/githubnext/gh-aw/pkg/parser"
	"github.com/githubnext/gh-aw/pkg/workflow/pretty"
	"github.com/goccy/go-yaml"
)

var log = logger.New("workflow:compiler")

const (
	// MaxLockFileSize is the maximum allowed size for generated lock workflow files (1MB)
	MaxLockFileSize = 1048576 // 1MB in bytes

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

// FileTracker interface for tracking files created during compilation
type FileTracker interface {
	TrackCreated(filePath string)
}

// Compiler handles converting markdown workflows to GitHub Actions YAML
type Compiler struct {
	verbose              bool
	engineOverride       string
	customOutput         string            // If set, output will be written to this path instead of default location
	version              string            // Version of the extension
	skipValidation       bool              // If true, skip schema validation
	noEmit               bool              // If true, validate without generating lock files
	strictMode           bool              // If true, enforce strict validation requirements
	trialMode            bool              // If true, suppress safe outputs for trial mode execution
	trialLogicalRepoSlug string            // If set in trial mode, the logical repository to checkout
	jobManager           *JobManager       // Manages jobs and dependencies
	engineRegistry       *EngineRegistry   // Registry of available agentic engines
	fileTracker          FileTracker       // Optional file tracker for tracking created files
	warningCount         int               // Number of warnings encountered during compilation
	stepOrderTracker     *StepOrderTracker // Tracks step ordering for validation
}

// NewCompiler creates a new workflow compiler with optional configuration
func NewCompiler(verbose bool, engineOverride string, version string) *Compiler {
	c := &Compiler{
		verbose:          verbose,
		engineOverride:   engineOverride,
		version:          version,
		skipValidation:   true, // Skip validation by default for now since existing workflows don't fully comply
		jobManager:       NewJobManager(),
		engineRegistry:   GetGlobalEngineRegistry(),
		stepOrderTracker: NewStepOrderTracker(),
	}

	return c
}

// Configures whether to skip schema validation
func (c *Compiler) SetSkipValidation(skip bool) {
	c.skipValidation = skip
}

// Configures whether to validate without generating lock files
func (c *Compiler) SetNoEmit(noEmit bool) {
	c.noEmit = noEmit
}

// Sets the file tracker for tracking created files
func (c *Compiler) SetFileTracker(tracker FileTracker) {
	c.fileTracker = tracker
}

// Configures whether to run in trial mode (suppresses safe outputs)
func (c *Compiler) SetTrialMode(trialMode bool) {
	c.trialMode = trialMode
}

// Configures the target repository for trial mode
func (c *Compiler) SetTrialLogicalRepoSlug(repo string) {
	c.trialLogicalRepoSlug = repo
}

// Configures whether to enable strict validation mode
func (c *Compiler) SetStrictMode(strict bool) {
	c.strictMode = strict
}

// IncrementWarningCount increments the warning counter
func (c *Compiler) IncrementWarningCount() {
	c.warningCount++
}

// GetWarningCount returns the current warning count
func (c *Compiler) GetWarningCount() int {
	return c.warningCount
}

// ResetWarningCount resets the warning counter to zero
func (c *Compiler) ResetWarningCount() {
	c.warningCount = 0
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
	Name                string
	TrialMode           bool     // whether the workflow is running in trial mode
	TrialLogicalRepo    string   // target repository slug for trial mode (owner/repo)
	FrontmatterName     string   // name field from frontmatter (for code scanning alert driver default)
	Description         string   // optional description rendered as comment in lock file
	Source              string   // optional source field (owner/repo@ref/path) rendered as comment in lock file
	ImportedFiles       []string // list of files imported via imports field (rendered as comment in lock file)
	IncludedFiles       []string // list of files included via @include directives (rendered as comment in lock file)
	On                  string
	Permissions         string
	Network             string // top-level network permissions configuration
	Concurrency         string // workflow-level concurrency configuration
	RunName             string
	Env                 string
	If                  string
	TimeoutMinutes      string
	CustomSteps         string
	PostSteps           string // steps to run after AI execution
	RunsOn              string
	Environment         string // environment setting for the main job
	Container           string // container setting for the main job
	Services            string // services setting for the main job
	Tools               map[string]any
	ParsedTools         *Tools // Structured tools configuration (NEW: parsed from Tools map)
	MarkdownContent     string
	AI                  string        // "claude" or "codex" (for backwards compatibility)
	EngineConfig        *EngineConfig // Extended engine configuration
	StopTime            string
	Command             string              // for /command trigger support
	CommandEvents       []string            // events where command should be active (nil = all events)
	CommandOtherEvents  map[string]any      // for merging command with other events
	AIReaction          string              // AI reaction type like "eyes", "heart", etc.
	Jobs                map[string]any      // custom job configurations with dependencies
	Cache               string              // cache configuration
	NeedsTextOutput     bool                // whether the workflow uses ${{ needs.task.outputs.text }}
	NetworkPermissions  *NetworkPermissions // parsed network permissions
	SafeOutputs         *SafeOutputsConfig  // output configuration for automatic output routes
	Roles               []string            // permission levels required to trigger workflow
	CacheMemoryConfig   *CacheMemoryConfig  // parsed cache-memory configuration
	SafetyPrompt        bool                // whether to include XPIA safety prompt (default true)
	Runtimes            map[string]any      // runtime version overrides from frontmatter
	ToolsTimeout        int                 // timeout in seconds for tool/MCP operations (0 = use engine default)
	GitHubToken         string              // top-level github-token expression from frontmatter
	ToolsStartupTimeout int                 // timeout in seconds for MCP server startup (0 = use engine default)
	Features            map[string]bool     // feature flags from frontmatter
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
	CreateAgentTasks                *CreateAgentTaskConfig                 `yaml:"create-agent-task,omitempty"` // Create GitHub Copilot agent tasks
	MissingTool                     *MissingToolConfig                     `yaml:"missing-tool,omitempty"`      // Optional for reporting missing functionality
	ThreatDetection                 *ThreatDetectionConfig                 `yaml:"threat-detection,omitempty"`  // Threat detection configuration
	Jobs                            map[string]*SafeJobConfig              `yaml:"jobs,omitempty"`              // Safe-jobs configuration (moved from top-level)
	AllowedDomains                  []string                               `yaml:"allowed-domains,omitempty"`
	Staged                          bool                                   `yaml:"staged,omitempty"`         // If true, emit step summary messages instead of making GitHub API calls
	Env                             map[string]string                      `yaml:"env,omitempty"`            // Environment variables to pass to safe output jobs
	GitHubToken                     string                                 `yaml:"github-token,omitempty"`   // GitHub token for safe output jobs
	MaximumPatchSize                int                                    `yaml:"max-patch-size,omitempty"` // Maximum allowed patch size in KB (defaults to 1024)
	RunsOn                          string                                 `yaml:"runs-on,omitempty"`        // Runner configuration for safe-outputs jobs
}

// CompileWorkflow converts a markdown workflow to GitHub Actions YAML
func (c *Compiler) CompileWorkflow(markdownPath string) error {

	// Reset the step order tracker for this compilation
	c.stepOrderTracker = NewStepOrderTracker()

	// replace the .md extension by .lock.yml
	lockFile := strings.TrimSuffix(markdownPath, ".md") + ".lock.yml"

	log.Printf("Starting compilation: %s -> %s", markdownPath, lockFile)

	// Parse the markdown file
	log.Printf("Parsing workflow file")
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
	log.Printf("Validating expression safety")
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

	log.Printf("Workflow: %s, Tools: %d", workflowData.Name, len(workflowData.Tools))

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
		log.Print("Validating workflow against GitHub Actions schema")
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

		// Validate expression sizes
		log.Print("Validating expression sizes")
		if err := c.validateExpressionSizes(yamlContent); err != nil {
			formattedErr := console.FormatError(console.CompilerError{
				Position: console.ErrorPosition{
					File:   markdownPath,
					Line:   1,
					Column: 1,
				},
				Type:    "error",
				Message: fmt.Sprintf("expression size validation failed: %v", err),
			})
			// Write the invalid YAML to a .invalid.yml file for inspection
			invalidFile := strings.TrimSuffix(lockFile, ".lock.yml") + ".invalid.yml"
			if writeErr := os.WriteFile(invalidFile, []byte(yamlContent), 0644); writeErr == nil {
				fmt.Println(console.FormatWarningMessage(fmt.Sprintf("Invalid workflow YAML written to: %s", console.ToRelativePath(invalidFile))))
			}
			return errors.New(formattedErr)
		}

		// Validate container images used in MCP configurations
		log.Print("Validating container images")
		if err := c.validateContainerImages(workflowData); err != nil {
			// Treat container image validation failures as warnings, not errors
			// This is because validation may fail due to auth issues locally (e.g., private registries)
			formattedWarning := console.FormatError(console.CompilerError{
				Position: console.ErrorPosition{
					File:   markdownPath,
					Line:   1,
					Column: 1,
				},
				Type:    "warning",
				Message: fmt.Sprintf("container image validation failed: %v", err),
			})
			fmt.Fprintln(os.Stderr, formattedWarning)
			c.IncrementWarningCount()
		}

		// Validate runtime packages (npx, uv)
		log.Print("Validating runtime packages")
		if err := c.validateRuntimePackages(workflowData); err != nil {
			formattedErr := console.FormatError(console.CompilerError{
				Position: console.ErrorPosition{
					File:   markdownPath,
					Line:   1,
					Column: 1,
				},
				Type:    "error",
				Message: fmt.Sprintf("runtime package validation failed: %v", err),
			})
			return errors.New(formattedErr)
		}

		// Validate repository features (discussions, issues)
		log.Print("Validating repository features")
		if err := c.validateRepositoryFeatures(workflowData); err != nil {
			formattedErr := console.FormatError(console.CompilerError{
				Position: console.ErrorPosition{
					File:   markdownPath,
					Line:   1,
					Column: 1,
				},
				Type:    "error",
				Message: fmt.Sprintf("repository feature validation failed: %v", err),
			})
			return errors.New(formattedErr)
		}
	} else if c.verbose {
		fmt.Println(console.FormatWarningMessage("Schema validation available but skipped (use SetSkipValidation(false) to enable)"))
		c.IncrementWarningCount()
	}

	// Write to lock file (unless noEmit is enabled)
	if c.noEmit {
		log.Print("Validation completed - no lock file generated (--no-emit enabled)")
	} else {
		log.Printf("Writing output to: %s", lockFile)
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

// ParseWorkflowFile parses a markdown workflow file and extracts all necessary data
func (c *Compiler) ParseWorkflowFile(markdownPath string) (*WorkflowData, error) {
	log.Printf("Reading file: %s", markdownPath)

	// Read the file
	content, err := os.ReadFile(markdownPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	log.Printf("File size: %d bytes", len(content))

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

	log.Printf("Frontmatter: %d chars, Markdown: %d chars", len(result.Frontmatter), len(result.Markdown))

	markdownDir := filepath.Dir(markdownPath)

	// Extract AI engine setting from frontmatter
	engineSetting, engineConfig := c.ExtractEngineConfig(result.Frontmatter)

	// Extract network permissions from frontmatter
	networkPermissions := c.extractNetworkPermissions(result.Frontmatter)

	// Default to 'defaults' network access if no network permissions specified
	if networkPermissions == nil {
		networkPermissions = &NetworkPermissions{
			Mode: "defaults",
		}
	}

	// Save the initial strict mode state to restore it after this workflow is processed
	// This ensures that strict mode from one workflow doesn't affect other workflows
	initialStrictMode := c.strictMode

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
		// Restore strict mode before returning error
		c.strictMode = initialStrictMode
		return nil, err
	}

	// Restore the initial strict mode state after validation
	// This ensures strict mode doesn't leak to other workflows being compiled
	c.strictMode = initialStrictMode

	// Validate that @include/@import directives are not used inside template regions
	if err := validateNoIncludesInTemplateRegions(result.Markdown); err != nil {
		return nil, fmt.Errorf("template region validation failed: %w", err)
	}

	// Override with command line AI engine setting if provided
	if c.engineOverride != "" {
		originalEngineSetting := engineSetting
		if originalEngineSetting != "" && originalEngineSetting != c.engineOverride {
			fmt.Println(console.FormatWarningMessage(fmt.Sprintf("Command line --engine %s overrides markdown file engine: %s", c.engineOverride, originalEngineSetting)))
			c.IncrementWarningCount()
		}
		engineSetting = c.engineOverride
	}

	// Process imports from frontmatter first (before @include directives)
	importsResult, err := parser.ProcessImportsFromFrontmatterWithManifest(result.Frontmatter, markdownDir)
	if err != nil {
		return nil, fmt.Errorf("failed to process imports from frontmatter: %w", err)
	}

	// Merge network permissions from imports with top-level network permissions
	if importsResult.MergedNetwork != "" {
		networkPermissions, err = c.MergeNetworkPermissions(networkPermissions, importsResult.MergedNetwork)
		if err != nil {
			return nil, fmt.Errorf("failed to merge network permissions: %w", err)
		}
	}

	// Process @include directives to extract engine configurations and check for conflicts
	includedEngines, err := parser.ExpandIncludesForEngines(result.Markdown, markdownDir)
	if err != nil {
		return nil, fmt.Errorf("failed to expand includes for engines: %w", err)
	}

	// Combine imported engines with included engines
	allEngines := append(importsResult.MergedEngines, includedEngines...)

	// Validate that only one engine field exists across all files
	finalEngineSetting, err := c.validateSingleEngineSpecification(engineSetting, allEngines)
	if err != nil {
		return nil, err
	}
	if finalEngineSetting != "" {
		engineSetting = finalEngineSetting
	}

	// If engineConfig is nil (engine was in an included file), extract it from the included engine JSON
	if engineConfig == nil && len(allEngines) > 0 {
		extractedConfig, err := c.extractEngineConfigFromJSON(allEngines[0])
		if err != nil {
			return nil, fmt.Errorf("failed to extract engine config from included file: %w", err)
		}
		engineConfig = extractedConfig
	}

	// Apply the default AI engine setting if not specified
	if engineSetting == "" {
		defaultEngine := c.engineRegistry.GetDefaultEngine()
		engineSetting = defaultEngine.GetID()
		if c.verbose {
			fmt.Println(console.FormatInfoMessage(fmt.Sprintf("NOTE: No 'engine:' setting found, defaulting to: %s", engineSetting)))
		}
		// Create a default EngineConfig with the default engine ID if not already set
		if engineConfig == nil {
			engineConfig = &EngineConfig{ID: engineSetting}
		} else if engineConfig.ID == "" {
			engineConfig.ID = engineSetting
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

	log.Printf("AI engine: %s (%s)", agenticEngine.GetDisplayName(), engineSetting)
	if agenticEngine.IsExperimental() && c.verbose {
		fmt.Println(console.FormatWarningMessage(fmt.Sprintf("Using experimental engine: %s", agenticEngine.GetDisplayName())))
		c.IncrementWarningCount()
	}

	// Save the initial strict mode state again for network support check
	// (it was restored after validateStrictMode but we need it again)
	initialStrictModeForNetwork := c.strictMode
	if !c.strictMode {
		if strictValue, exists := result.Frontmatter["strict"]; exists {
			if strictBool, ok := strictValue.(bool); ok && strictBool {
				c.strictMode = true
			}
		}
	}

	// Check if the engine supports network restrictions when they are defined
	if err := c.checkNetworkSupport(agenticEngine, networkPermissions); err != nil {
		// Restore strict mode before returning error
		c.strictMode = initialStrictModeForNetwork
		return nil, err
	}

	// Restore the strict mode state after network check
	c.strictMode = initialStrictModeForNetwork

	log.Print("Processing tools and includes...")

	// Extract SafeOutputs configuration early so we can use it when applying default tools
	safeOutputs := c.extractSafeOutputsConfig(result.Frontmatter)

	var tools map[string]any

	// Extract tools from the main file
	topTools := extractToolsFromFrontmatter(result.Frontmatter)

	// Extract mcp-servers from the main file and merge them into tools
	mcpServers := extractMCPServersFromFrontmatter(result.Frontmatter)

	// Process @include directives to extract additional tools
	includedTools, includedToolFiles, err := parser.ExpandIncludesWithManifest(result.Markdown, markdownDir, true)
	if err != nil {
		return nil, fmt.Errorf("failed to expand includes for tools: %w", err)
	}

	// Combine imported tools with included tools
	var allIncludedTools string
	if importsResult.MergedTools != "" && includedTools != "" {
		allIncludedTools = importsResult.MergedTools + "\n" + includedTools
	} else if importsResult.MergedTools != "" {
		allIncludedTools = importsResult.MergedTools
	} else {
		allIncludedTools = includedTools
	}

	// Combine imported mcp-servers with top-level mcp-servers
	// Imported mcp-servers are in JSON format (newline-separated), need to merge them
	allMCPServers := mcpServers
	if importsResult.MergedMCPServers != "" {
		// Parse and merge imported MCP servers
		mergedMCPServers, err := c.MergeMCPServers(mcpServers, importsResult.MergedMCPServers)
		if err != nil {
			return nil, fmt.Errorf("failed to merge imported mcp-servers: %w", err)
		}
		allMCPServers = mergedMCPServers
	}

	// Merge tools including mcp-servers
	tools, err = c.mergeToolsAndMCPServers(topTools, allMCPServers, allIncludedTools)

	if err != nil {
		return nil, fmt.Errorf("failed to merge tools: %w", err)
	}

	// Extract safety-prompt setting from merged tools (defaults to true)
	safetyPrompt := c.extractSafetyPromptSetting(tools)

	// Extract timeout setting from merged tools (defaults to 0 which means use engine defaults)
	toolsTimeout := c.extractToolsTimeout(tools)

	// Extract startup-timeout setting from merged tools (defaults to 0 which means use engine defaults)
	toolsStartupTimeout := c.extractToolsStartupTimeout(tools)

	// Remove meta fields (safety-prompt, timeout, startup-timeout) from merged tools map
	// These are configuration fields, not actual tools
	delete(tools, "safety-prompt")
	delete(tools, "timeout")
	delete(tools, "startup-timeout")

	// Extract and merge runtimes from frontmatter and imports
	topRuntimes := extractRuntimesFromFrontmatter(result.Frontmatter)
	runtimes, err := mergeRuntimes(topRuntimes, importsResult.MergedRuntimes)
	if err != nil {
		return nil, fmt.Errorf("failed to merge runtimes: %w", err)
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
		c.IncrementWarningCount()
		if _, hasTools := result.Frontmatter["tools"]; hasTools {
			fmt.Println(console.FormatWarningMessage(fmt.Sprintf("'tools' section ignored when using engine: %s (%s doesn't support MCP tool allow-listing)", engineSetting, agenticEngine.GetDisplayName())))
			c.IncrementWarningCount()
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

	// Validate web-search support for the current engine (warning only)
	c.validateWebSearchSupport(tools, agenticEngine)

	// Process @include directives in markdown content
	markdownContent, includedMarkdownFiles, err := parser.ExpandIncludesWithManifest(result.Markdown, markdownDir, false)
	if err != nil {
		return nil, fmt.Errorf("failed to expand includes in markdown: %w", err)
	}

	// Prepend imported markdown from frontmatter imports field
	if importsResult.MergedMarkdown != "" {
		markdownContent = importsResult.MergedMarkdown + markdownContent
	}

	log.Print("Expanded includes in markdown content")

	// Combine all included files (from tools and markdown)
	// Use a map to deduplicate files
	allIncludedFilesMap := make(map[string]bool)
	for _, file := range includedToolFiles {
		allIncludedFilesMap[file] = true
	}
	for _, file := range includedMarkdownFiles {
		allIncludedFilesMap[file] = true
	}
	var allIncludedFiles []string
	for file := range allIncludedFilesMap {
		allIncludedFiles = append(allIncludedFiles, file)
	}
	// Sort files alphabetically to ensure consistent ordering in lock files
	sort.Strings(allIncludedFiles)

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

	log.Printf("Extracted workflow name: '%s'", workflowName)

	// Check if the markdown content uses the text output
	needsTextOutput := c.detectTextOutputUsage(markdownContent)

	// Build workflow data
	workflowData := &WorkflowData{
		Name:                workflowName,
		FrontmatterName:     frontmatterName,
		Description:         c.extractDescription(result.Frontmatter),
		Source:              c.extractSource(result.Frontmatter),
		ImportedFiles:       importsResult.ImportedFiles,
		IncludedFiles:       allIncludedFiles,
		Tools:               tools,
		ParsedTools:         NewTools(tools),
		Runtimes:            runtimes,
		MarkdownContent:     markdownContent,
		AI:                  engineSetting,
		EngineConfig:        engineConfig,
		NetworkPermissions:  networkPermissions,
		NeedsTextOutput:     needsTextOutput,
		SafetyPrompt:        safetyPrompt,
		ToolsTimeout:        toolsTimeout,
		ToolsStartupTimeout: toolsStartupTimeout,
		TrialMode:           c.trialMode,
		TrialLogicalRepo:    c.trialLogicalRepoSlug,
		GitHubToken:         extractStringValue(result.Frontmatter, "github-token"),
	}

	// Extract YAML sections from frontmatter - use direct frontmatter map extraction
	// to avoid issues with nested keys (e.g., tools.mcps.*.env being confused with top-level env)
	workflowData.On = c.extractTopLevelYAMLSection(result.Frontmatter, "on")
	workflowData.Permissions = c.extractPermissions(result.Frontmatter)
	workflowData.Network = c.extractTopLevelYAMLSection(result.Frontmatter, "network")
	workflowData.Concurrency = c.extractTopLevelYAMLSection(result.Frontmatter, "concurrency")
	workflowData.RunName = c.extractTopLevelYAMLSection(result.Frontmatter, "run-name")
	workflowData.Env = c.extractTopLevelYAMLSection(result.Frontmatter, "env")
	workflowData.Features = c.extractFeatures(result.Frontmatter)
	workflowData.If = c.extractIfCondition(result.Frontmatter)
	workflowData.TimeoutMinutes = c.extractTopLevelYAMLSection(result.Frontmatter, "timeout_minutes")
	workflowData.CustomSteps = c.extractTopLevelYAMLSection(result.Frontmatter, "steps")

	// Merge imported steps if any
	if importsResult.MergedSteps != "" {
		// Parse imported steps from YAML array
		var importedSteps []any
		if err := yaml.Unmarshal([]byte(importsResult.MergedSteps), &importedSteps); err == nil {
			// Apply action pinning to imported steps
			importedSteps = ApplyActionPinsToSteps(importedSteps)

			// If there are main workflow steps, parse and merge them
			if workflowData.CustomSteps != "" {
				// Parse main workflow steps (format: "steps:\n  - ...")
				var mainStepsWrapper map[string]any
				if err := yaml.Unmarshal([]byte(workflowData.CustomSteps), &mainStepsWrapper); err == nil {
					if mainStepsVal, hasSteps := mainStepsWrapper["steps"]; hasSteps {
						if mainSteps, ok := mainStepsVal.([]any); ok {
							// Apply action pinning to main steps
							mainSteps = ApplyActionPinsToSteps(mainSteps)

							// Prepend imported steps to main steps
							allSteps := append(importedSteps, mainSteps...)
							// Convert back to YAML with "steps:" wrapper
							stepsWrapper := map[string]any{"steps": allSteps}
							stepsYAML, err := yaml.Marshal(stepsWrapper)
							if err == nil {
								workflowData.CustomSteps = string(stepsYAML)
							}
						}
					}
				}
			} else {
				// Only imported steps exist, wrap in "steps:" format
				stepsWrapper := map[string]any{"steps": importedSteps}
				stepsYAML, err := yaml.Marshal(stepsWrapper)
				if err == nil {
					workflowData.CustomSteps = string(stepsYAML)
				}
			}
		}
	} else if workflowData.CustomSteps != "" {
		// No imported steps, but there are main steps - still apply pinning
		var mainStepsWrapper map[string]any
		if err := yaml.Unmarshal([]byte(workflowData.CustomSteps), &mainStepsWrapper); err == nil {
			if mainStepsVal, hasSteps := mainStepsWrapper["steps"]; hasSteps {
				if mainSteps, ok := mainStepsVal.([]any); ok {
					// Apply action pinning to main steps
					mainSteps = ApplyActionPinsToSteps(mainSteps)

					// Convert back to YAML with "steps:" wrapper
					stepsWrapper := map[string]any{"steps": mainSteps}
					stepsYAML, err := yaml.Marshal(stepsWrapper)
					if err == nil {
						workflowData.CustomSteps = string(stepsYAML)
					}
				}
			}
		}
	}

	workflowData.PostSteps = c.extractTopLevelYAMLSection(result.Frontmatter, "post-steps")

	// Apply action pinning to post-steps if any
	if workflowData.PostSteps != "" {
		var postStepsWrapper map[string]any
		if err := yaml.Unmarshal([]byte(workflowData.PostSteps), &postStepsWrapper); err == nil {
			if postStepsVal, hasPostSteps := postStepsWrapper["post-steps"]; hasPostSteps {
				if postSteps, ok := postStepsVal.([]any); ok {
					// Apply action pinning to post steps
					postSteps = ApplyActionPinsToSteps(postSteps)

					// Convert back to YAML with "post-steps:" wrapper
					stepsWrapper := map[string]any{"post-steps": postSteps}
					stepsYAML, err := yaml.Marshal(stepsWrapper)
					if err == nil {
						workflowData.PostSteps = string(stepsYAML)
					}
				}
			}
		}
	}

	workflowData.RunsOn = c.extractTopLevelYAMLSection(result.Frontmatter, "runs-on")
	workflowData.Environment = c.extractTopLevelYAMLSection(result.Frontmatter, "environment")
	workflowData.Container = c.extractTopLevelYAMLSection(result.Frontmatter, "container")
	workflowData.Services = c.extractTopLevelYAMLSection(result.Frontmatter, "services")

	// Merge imported services if any
	if importsResult.MergedServices != "" {
		// Parse imported services from YAML
		var importedServices map[string]any
		if err := yaml.Unmarshal([]byte(importsResult.MergedServices), &importedServices); err == nil {
			// If there are main workflow services, parse and merge them
			if workflowData.Services != "" {
				// Parse main workflow services
				var mainServicesWrapper map[string]any
				if err := yaml.Unmarshal([]byte(workflowData.Services), &mainServicesWrapper); err == nil {
					if mainServices, ok := mainServicesWrapper["services"].(map[string]any); ok {
						// Merge: main workflow services take precedence over imported
						for key, value := range importedServices {
							if _, exists := mainServices[key]; !exists {
								mainServices[key] = value
							}
						}
						// Convert back to YAML with "services:" wrapper
						servicesWrapper := map[string]any{"services": mainServices}
						servicesYAML, err := yaml.Marshal(servicesWrapper)
						if err == nil {
							workflowData.Services = string(servicesYAML)
						}
					}
				}
			} else {
				// Only imported services exist, wrap in "services:" format
				servicesWrapper := map[string]any{"services": importedServices}
				servicesYAML, err := yaml.Marshal(servicesWrapper)
				if err == nil {
					workflowData.Services = string(servicesYAML)
				}
			}
		}
	}

	workflowData.Cache = c.extractTopLevelYAMLSection(result.Frontmatter, "cache")

	// Extract cache-memory config and check for errors
	cacheMemoryConfig, err := c.extractCacheMemoryConfig(tools) // Use merged tools to support imports
	if err != nil {
		return nil, err
	}
	workflowData.CacheMemoryConfig = cacheMemoryConfig

	// Process stop-after configuration from the on: section
	err = c.processStopAfterConfiguration(result.Frontmatter, workflowData, markdownPath)
	if err != nil {
		return nil, err
	}

	workflowData.Command, workflowData.CommandEvents = c.extractCommandConfig(result.Frontmatter)
	workflowData.Jobs = c.extractJobsFromFrontmatter(result.Frontmatter)
	workflowData.Roles = c.extractRoles(result.Frontmatter)

	// Use the already extracted output configuration
	workflowData.SafeOutputs = safeOutputs

	// Extract safe-jobs from the new location (safe-outputs.jobs) or old location (safe-jobs) for backwards compatibility
	topSafeJobs := extractSafeJobsFromFrontmatter(result.Frontmatter)

	// Process @include directives to extract additional safe-outputs configurations
	includedSafeOutputsConfigs, err := parser.ExpandIncludesForSafeOutputs(result.Markdown, markdownDir)
	if err != nil {
		return nil, fmt.Errorf("failed to expand includes for safe-outputs: %w", err)
	}

	// Combine imported safe-outputs with included safe-outputs
	var allSafeOutputsConfigs []string
	if len(importsResult.MergedSafeOutputs) > 0 {
		allSafeOutputsConfigs = append(allSafeOutputsConfigs, importsResult.MergedSafeOutputs...)
	}
	if len(includedSafeOutputsConfigs) > 0 {
		allSafeOutputsConfigs = append(allSafeOutputsConfigs, includedSafeOutputsConfigs...)
	}

	// Merge safe-jobs from all safe-outputs configurations (imported and included)
	includedSafeJobs, err := c.mergeSafeJobsFromIncludedConfigs(topSafeJobs, allSafeOutputsConfigs)
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

	// Apply label filter if specified
	c.applyLabelFilter(workflowData, result.Frontmatter)

	return workflowData, nil
}

// extractTopLevelYAMLSection extracts a top-level YAML section from the frontmatter map
// This ensures we only extract keys at the root level, avoiding nested keys with the same name
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
					// Validate reaction value
					if !isValidReaction(reactionStr) {
						return fmt.Errorf("invalid reaction value '%s': must be one of %v", reactionStr, getValidReactions())
					}
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
			// Keep "on" quoted as it's a YAML boolean keyword
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

// mergeSafeJobsFromIncludedConfigs merges safe-jobs from included safe-outputs configurations
func (c *Compiler) mergeSafeJobsFromIncludedConfigs(topSafeJobs map[string]*SafeJobConfig, includedConfigs []string) (map[string]*SafeJobConfig, error) {
	result := topSafeJobs
	if result == nil {
		result = make(map[string]*SafeJobConfig)
	}

	for _, configJSON := range includedConfigs {
		if configJSON == "" || configJSON == "{}" {
			continue
		}

		// Parse the safe-outputs configuration
		var safeOutputsConfig map[string]any
		if err := json.Unmarshal([]byte(configJSON), &safeOutputsConfig); err != nil {
			continue // Skip invalid JSON
		}

		// Extract safe-jobs from the safe-outputs.jobs field
		includedSafeJobs := extractSafeJobsFromFrontmatter(map[string]any{
			"safe-outputs": safeOutputsConfig,
		})

		// Merge with conflict detection
		var err error
		result, err = mergeSafeJobs(result, includedSafeJobs)
		if err != nil {
			return nil, fmt.Errorf("failed to merge safe-jobs from includes: %w", err)
		}
	}

	return result, nil
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

	// Only set allowed tools if explicitly configured
	// Don't add default tools - let the MCP server use all available tools
	if len(existingAllowed) > 0 {
		githubConfig["allowed"] = existingAllowed
	}
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
			"git status",
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
	//   - bash: true → All commands allowed (converted to ["*"])
	//   - bash: false → Tool disabled (removed from tools)
	//   - bash: nil → Add default commands
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
			// bash is true - convert to wildcard (allow all commands)
			tools["bash"] = []any{"*"}
		} else if bashTool == false {
			// bash is false - disable the tool by removing it
			delete(tools, "bash")
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

				// Start with default commands (append handles capacity automatically)
				var mergedCommands []any
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
	log.Printf("Detected usage of activation.outputs.text: %v", hasUsage)
	return hasUsage
}

// generateYAML generates the complete GitHub Actions YAML content

// isActivationJobNeeded determines if the activation job is required
func (c *Compiler) isActivationJobNeeded() bool {
	// Activation job is always needed to perform the timestamp check
	// It also handles:
	// 1. Command is configured (for team member checking)
	// 2. Text output is needed (for compute-text action)
	// 3. If condition is specified (to handle runtime conditions)
	// 4. Permission checks are needed (consolidated team member validation)
	return true
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

	// Determine if permission checks or stop-time checks are needed
	needsPermissionCheck := c.needsRoleCheck(data, frontmatter)
	hasStopTime := data.StopTime != ""

	// Build pre-activation job if needed (combines membership checks, stop-time validation, and command position check)
	var preActivationJobCreated bool
	hasCommandTrigger := data.Command != ""
	if needsPermissionCheck || hasStopTime || hasCommandTrigger {
		preActivationJob, err := c.buildPreActivationJob(data, needsPermissionCheck)
		if err != nil {
			return fmt.Errorf("failed to build %s job: %w", constants.PreActivationJobName, err)
		}
		if err := c.jobManager.AddJob(preActivationJob); err != nil {
			return fmt.Errorf("failed to add %s job: %w", constants.PreActivationJobName, err)
		}
		preActivationJobCreated = true
	}

	// Build activation job if needed (preamble job that handles runtime conditions)
	// If pre-activation job exists, activation job depends on it and checks the "activated" output
	var activationJobCreated bool

	if c.isActivationJobNeeded() {
		activationJob, err := c.buildActivationJob(data, preActivationJobCreated)
		if err != nil {
			return fmt.Errorf("failed to build activation job: %w", err)
		}
		if err := c.jobManager.AddJob(activationJob); err != nil {
			return fmt.Errorf("failed to add activation job: %w", err)
		}
		activationJobCreated = true
	}

	// Build main workflow job
	mainJob, err := c.buildMainJob(data, activationJobCreated)
	if err != nil {
		return fmt.Errorf("failed to build main job: %w", err)
	}
	if err := c.jobManager.AddJob(mainJob); err != nil {
		return fmt.Errorf("failed to add main job: %w", err)
	}

	// Build safe outputs jobs if configured
	if err := c.buildSafeOutputsJobs(data, constants.AgentJobName, markdownPath); err != nil {
		return fmt.Errorf("failed to build safe outputs jobs: %w", err)
	}

	// Build safe-jobs if configured
	// Safe-jobs should depend on agent job (always) AND detection job (if threat detection is enabled)
	threatDetectionEnabledForSafeJobs := data.SafeOutputs != nil && data.SafeOutputs.ThreatDetection != nil
	if err := c.buildSafeJobs(data, threatDetectionEnabledForSafeJobs); err != nil {
		return fmt.Errorf("failed to build safe-jobs: %w", err)
	}

	// Build additional custom jobs from frontmatter jobs section
	if err := c.buildCustomJobs(data); err != nil {
		return fmt.Errorf("failed to build custom jobs: %w", err)
	}

	return nil
}

// buildSafeOutputsJobs creates all safe outputs jobs if configured
func (c *Compiler) buildSafeOutputsJobs(data *WorkflowData, jobName, markdownPath string) error {
	if data.SafeOutputs == nil {
		return nil
	}

	// Track whether threat detection job is enabled
	threatDetectionEnabled := false

	// Build threat detection job if enabled
	if data.SafeOutputs.ThreatDetection != nil {
		detectionJob, err := c.buildThreatDetectionJob(data, jobName)
		if err != nil {
			return fmt.Errorf("failed to build detection job: %w", err)
		}
		if err := c.jobManager.AddJob(detectionJob); err != nil {
			return fmt.Errorf("failed to add detection job: %w", err)
		}
		threatDetectionEnabled = true
	}

	// Track safe output job names to establish dependencies for update_reaction job
	var safeOutputJobNames []string

	// Track which jobs create_issue, create_discussion, and create_pull_request were created
	// so that add_comment can depend on them and reference their outputs
	var createIssueJobName string
	var createDiscussionJobName string
	var createPullRequestJobName string

	// Build create_issue job if output.create_issue is configured
	if data.SafeOutputs.CreateIssues != nil {
		createIssueJob, err := c.buildCreateOutputIssueJob(data, jobName)
		if err != nil {
			return fmt.Errorf("failed to build create_issue job: %w", err)
		}
		// Safe-output jobs should depend on agent job (always) AND detection job (if enabled)
		if threatDetectionEnabled {
			createIssueJob.Needs = append(createIssueJob.Needs, constants.DetectionJobName)
		}
		if err := c.jobManager.AddJob(createIssueJob); err != nil {
			return fmt.Errorf("failed to add create_issue job: %w", err)
		}
		safeOutputJobNames = append(safeOutputJobNames, createIssueJob.Name)
		createIssueJobName = createIssueJob.Name
	}

	// Build create_discussion job if output.create_discussion is configured
	if data.SafeOutputs.CreateDiscussions != nil {
		createDiscussionJob, err := c.buildCreateOutputDiscussionJob(data, jobName)
		if err != nil {
			return fmt.Errorf("failed to build create_discussion job: %w", err)
		}
		// Safe-output jobs should depend on agent job (always) AND detection job (if enabled)
		if threatDetectionEnabled {
			createDiscussionJob.Needs = append(createDiscussionJob.Needs, constants.DetectionJobName)
		}
		if err := c.jobManager.AddJob(createDiscussionJob); err != nil {
			return fmt.Errorf("failed to add create_discussion job: %w", err)
		}
		safeOutputJobNames = append(safeOutputJobNames, createDiscussionJob.Name)
		createDiscussionJobName = createDiscussionJob.Name
	}

	// Build create_pull_request job if output.create-pull-request is configured
	// NOTE: This is built BEFORE add_comment so that add_comment can depend on it
	if data.SafeOutputs.CreatePullRequests != nil {
		createPullRequestJob, err := c.buildCreateOutputPullRequestJob(data, jobName)
		if err != nil {
			return fmt.Errorf("failed to build create_pull_request job: %w", err)
		}
		// Safe-output jobs should depend on agent job (always) AND detection job (if enabled)
		if threatDetectionEnabled {
			createPullRequestJob.Needs = append(createPullRequestJob.Needs, constants.DetectionJobName)
		}
		if err := c.jobManager.AddJob(createPullRequestJob); err != nil {
			return fmt.Errorf("failed to add create_pull_request job: %w", err)
		}
		safeOutputJobNames = append(safeOutputJobNames, createPullRequestJob.Name)
		createPullRequestJobName = createPullRequestJob.Name
	}

	// Build add_comment job if output.add-comment is configured
	if data.SafeOutputs.AddComments != nil {
		createCommentJob, err := c.buildCreateOutputAddCommentJob(data, jobName, createIssueJobName, createDiscussionJobName, createPullRequestJobName)
		if err != nil {
			return fmt.Errorf("failed to build add_comment job: %w", err)
		}
		// Safe-output jobs should depend on agent job (always) AND detection job (if enabled)
		if threatDetectionEnabled {
			createCommentJob.Needs = append(createCommentJob.Needs, constants.DetectionJobName)
		}
		if err := c.jobManager.AddJob(createCommentJob); err != nil {
			return fmt.Errorf("failed to add add_comment job: %w", err)
		}
		safeOutputJobNames = append(safeOutputJobNames, createCommentJob.Name)
	}

	// Build create_pr_review_comment job if output.create-pull-request-review-comment is configured
	if data.SafeOutputs.CreatePullRequestReviewComments != nil {
		createPRReviewCommentJob, err := c.buildCreateOutputPullRequestReviewCommentJob(data, jobName)
		if err != nil {
			return fmt.Errorf("failed to build create_pr_review_comment job: %w", err)
		}
		// Safe-output jobs should depend on agent job (always) AND detection job (if enabled)
		if threatDetectionEnabled {
			createPRReviewCommentJob.Needs = append(createPRReviewCommentJob.Needs, constants.DetectionJobName)
		}
		if err := c.jobManager.AddJob(createPRReviewCommentJob); err != nil {
			return fmt.Errorf("failed to add create_pr_review_comment job: %w", err)
		}
		safeOutputJobNames = append(safeOutputJobNames, createPRReviewCommentJob.Name)
	}

	// Build create_code_scanning_alert job if output.create-code-scanning-alert is configured
	if data.SafeOutputs.CreateCodeScanningAlerts != nil {
		// Extract the workflow filename without extension for rule ID prefix
		workflowFilename := strings.TrimSuffix(filepath.Base(markdownPath), ".md")
		createCodeScanningAlertJob, err := c.buildCreateOutputCodeScanningAlertJob(data, jobName, workflowFilename)
		if err != nil {
			return fmt.Errorf("failed to build create_code_scanning_alert job: %w", err)
		}
		// Safe-output jobs should depend on agent job (always) AND detection job (if enabled)
		if threatDetectionEnabled {
			createCodeScanningAlertJob.Needs = append(createCodeScanningAlertJob.Needs, constants.DetectionJobName)
		}
		if err := c.jobManager.AddJob(createCodeScanningAlertJob); err != nil {
			return fmt.Errorf("failed to add create_code_scanning_alert job: %w", err)
		}
		safeOutputJobNames = append(safeOutputJobNames, createCodeScanningAlertJob.Name)
	}

	// Build add_labels job if output.add-labels is configured (including null/empty)
	if data.SafeOutputs.AddLabels != nil {
		addLabelsJob, err := c.buildAddLabelsJob(data, jobName)
		if err != nil {
			return fmt.Errorf("failed to build add_labels job: %w", err)
		}
		// Safe-output jobs should depend on agent job (always) AND detection job (if enabled)
		if threatDetectionEnabled {
			addLabelsJob.Needs = append(addLabelsJob.Needs, constants.DetectionJobName)
		}
		if err := c.jobManager.AddJob(addLabelsJob); err != nil {
			return fmt.Errorf("failed to add add_labels job: %w", err)
		}
		safeOutputJobNames = append(safeOutputJobNames, addLabelsJob.Name)
	}

	// Build update_issue job if output.update-issue is configured
	if data.SafeOutputs.UpdateIssues != nil {
		updateIssueJob, err := c.buildCreateOutputUpdateIssueJob(data, jobName)
		if err != nil {
			return fmt.Errorf("failed to build update_issue job: %w", err)
		}
		// Safe-output jobs should depend on agent job (always) AND detection job (if enabled)
		if threatDetectionEnabled {
			updateIssueJob.Needs = append(updateIssueJob.Needs, constants.DetectionJobName)
		}
		if err := c.jobManager.AddJob(updateIssueJob); err != nil {
			return fmt.Errorf("failed to add update_issue job: %w", err)
		}
		safeOutputJobNames = append(safeOutputJobNames, updateIssueJob.Name)
	}

	// Build push_to_pull_request_branch job if output.push-to-pull-request-branch is configured
	if data.SafeOutputs.PushToPullRequestBranch != nil {
		pushToBranchJob, err := c.buildCreateOutputPushToPullRequestBranchJob(data, jobName)
		if err != nil {
			return fmt.Errorf("failed to build push_to_pull_request_branch job: %w", err)
		}
		// Safe-output jobs should depend on agent job (always) AND detection job (if enabled)
		if threatDetectionEnabled {
			pushToBranchJob.Needs = append(pushToBranchJob.Needs, constants.DetectionJobName)
		}
		if err := c.jobManager.AddJob(pushToBranchJob); err != nil {
			return fmt.Errorf("failed to add push_to_pull_request_branch job: %w", err)
		}
		safeOutputJobNames = append(safeOutputJobNames, pushToBranchJob.Name)
	}

	// Build missing_tool job (always enabled when SafeOutputs exists)
	if data.SafeOutputs.MissingTool != nil {
		missingToolJob, err := c.buildCreateOutputMissingToolJob(data, jobName)
		if err != nil {
			return fmt.Errorf("failed to build missing_tool job: %w", err)
		}
		// Safe-output jobs should depend on agent job (always) AND detection job (if enabled)
		if threatDetectionEnabled {
			missingToolJob.Needs = append(missingToolJob.Needs, constants.DetectionJobName)
		}
		if err := c.jobManager.AddJob(missingToolJob); err != nil {
			return fmt.Errorf("failed to add missing_tool job: %w", err)
		}
		safeOutputJobNames = append(safeOutputJobNames, missingToolJob.Name)
	}

	// Build upload_assets job if output.upload-asset is configured
	if data.SafeOutputs.UploadAssets != nil {
		uploadAssetsJob, err := c.buildUploadAssetsJob(data, jobName)
		if err != nil {
			return fmt.Errorf("failed to build upload_assets job: %w", err)
		}
		// Safe-output jobs should depend on agent job (always) AND detection job (if enabled)
		if threatDetectionEnabled {
			uploadAssetsJob.Needs = append(uploadAssetsJob.Needs, constants.DetectionJobName)
		}
		if err := c.jobManager.AddJob(uploadAssetsJob); err != nil {
			return fmt.Errorf("failed to add upload_assets job: %w", err)
		}
		safeOutputJobNames = append(safeOutputJobNames, uploadAssetsJob.Name)
	}

	// Build create_agent_task job if output.create-agent-task is configured
	if data.SafeOutputs.CreateAgentTasks != nil {
		createAgentTaskJob, err := c.buildCreateOutputAgentTaskJob(data, jobName)
		if err != nil {
			return fmt.Errorf("failed to build create_agent_task job: %w", err)
		}
		// Safe-output jobs should depend on agent job (always) AND detection job (if enabled)
		if threatDetectionEnabled {
			createAgentTaskJob.Needs = append(createAgentTaskJob.Needs, constants.DetectionJobName)
		}
		if err := c.jobManager.AddJob(createAgentTaskJob); err != nil {
			return fmt.Errorf("failed to add create_agent_task job: %w", err)
		}
		safeOutputJobNames = append(safeOutputJobNames, createAgentTaskJob.Name)
	}

	// Build update_reaction job if add-comment is configured
	// This job runs last, after all safe output jobs, to update the activation comment on failure
	if data.SafeOutputs.AddComments != nil {
		updateReactionJob, err := c.buildUpdateReactionJob(data, jobName, safeOutputJobNames)
		if err != nil {
			return fmt.Errorf("failed to build update_reaction job: %w", err)
		}
		if updateReactionJob != nil {
			if err := c.jobManager.AddJob(updateReactionJob); err != nil {
				return fmt.Errorf("failed to add update_reaction job: %w", err)
			}
		}
	}

	return nil
}

// buildPreActivationJob creates a unified pre-activation job that combines membership checks and stop-time validation
// This job exposes a single "activated" output that indicates whether the workflow should proceed
func (c *Compiler) buildPreActivationJob(data *WorkflowData, needsPermissionCheck bool) (*Job, error) {
	var steps []string
	var permissions string

	// Add team member check if permission checks are needed
	if needsPermissionCheck {
		steps = c.generateMembershipCheck(data, steps)
	}

	// Add stop-time check if configured
	if data.StopTime != "" {
		// Extract workflow name for the stop-time check
		workflowName := data.Name

		steps = append(steps, "      - name: Check stop-time limit\n")
		steps = append(steps, fmt.Sprintf("        id: %s\n", constants.CheckStopTimeStepID))
		steps = append(steps, fmt.Sprintf("        uses: %s\n", GetActionPin("actions/github-script")))
		steps = append(steps, "        env:\n")
		steps = append(steps, fmt.Sprintf("          GH_AW_STOP_TIME: %s\n", data.StopTime))
		steps = append(steps, fmt.Sprintf("          GH_AW_WORKFLOW_NAME: %q\n", workflowName))
		steps = append(steps, "        with:\n")
		steps = append(steps, "          script: |\n")

		// Add the JavaScript script with proper indentation
		formattedScript := FormatJavaScriptForYAML(checkStopTimeScript)
		steps = append(steps, formattedScript...)
	}

	// Add command position check if this is a command workflow
	if data.Command != "" {
		steps = append(steps, "      - name: Check command position\n")
		steps = append(steps, fmt.Sprintf("        id: %s\n", constants.CheckCommandPositionStepID))
		steps = append(steps, fmt.Sprintf("        uses: %s\n", GetActionPin("actions/github-script")))
		steps = append(steps, "        env:\n")
		steps = append(steps, fmt.Sprintf("          GH_AW_COMMAND: %s\n", data.Command))
		steps = append(steps, "        with:\n")
		steps = append(steps, "          script: |\n")

		// Add the JavaScript script with proper indentation
		formattedScript := FormatJavaScriptForYAML(checkCommandPositionScript)
		steps = append(steps, formattedScript...)
	}

	// Generate the activated output expression using expression builders
	var activatedNode ConditionNode

	// Build condition nodes for each check
	var conditions []ConditionNode

	if needsPermissionCheck {
		// Add membership check condition
		membershipCheck := BuildComparison(
			BuildPropertyAccess(fmt.Sprintf("steps.%s.outputs.%s", constants.CheckMembershipStepID, constants.IsTeamMemberOutput)),
			"==",
			BuildStringLiteral("true"),
		)
		conditions = append(conditions, membershipCheck)
	}

	if data.StopTime != "" {
		// Add stop-time check condition
		stopTimeCheck := BuildComparison(
			BuildPropertyAccess(fmt.Sprintf("steps.%s.outputs.%s", constants.CheckStopTimeStepID, constants.StopTimeOkOutput)),
			"==",
			BuildStringLiteral("true"),
		)
		conditions = append(conditions, stopTimeCheck)
	}

	if data.Command != "" {
		// Add command position check condition
		commandPositionCheck := BuildComparison(
			BuildPropertyAccess(fmt.Sprintf("steps.%s.outputs.%s", constants.CheckCommandPositionStepID, constants.CommandPositionOkOutput)),
			"==",
			BuildStringLiteral("true"),
		)
		conditions = append(conditions, commandPositionCheck)
	}

	// Build the final expression
	if len(conditions) == 0 {
		// This should never happen - it means pre-activation job was created without any checks
		// If we reach this point, it's a developer error in the compiler logic
		return nil, fmt.Errorf("developer error: pre-activation job created without permission check or stop-time configuration")
	} else if len(conditions) == 1 {
		// Single condition
		activatedNode = conditions[0]
	} else {
		// Multiple conditions - combine with AND
		activatedNode = conditions[0]
		for i := 1; i < len(conditions); i++ {
			activatedNode = buildAnd(activatedNode, conditions[i])
		}
	}

	// Render the expression with ${{ }} wrapper
	activatedExpression := fmt.Sprintf("${{ %s }}", activatedNode.Render())

	outputs := map[string]string{
		"activated": activatedExpression,
	}

	job := &Job{
		Name:        constants.PreActivationJobName,
		If:          data.If, // Use the existing condition (which may include alias checks)
		RunsOn:      c.formatSafeOutputsRunsOn(data.SafeOutputs),
		Permissions: permissions,
		Steps:       steps,
		Outputs:     outputs,
	}

	return job, nil
}

// buildActivationJob creates the preamble activation job that acts as a barrier for runtime conditions
func (c *Compiler) buildActivationJob(data *WorkflowData, preActivationJobCreated bool) (*Job, error) {
	outputs := map[string]string{}
	var steps []string

	// Team member check is now handled by the separate check_membership job
	// No inline role checks needed in the task job anymore

	// Add timestamp check for lock file vs source file
	steps = append(steps, "      - name: Check workflow file timestamps\n")
	steps = append(steps, "        run: |\n")
	steps = append(steps, "          WORKFLOW_FILE=\"${GITHUB_WORKSPACE}/.github/workflows/$(basename \"$GITHUB_WORKFLOW\" .lock.yml).md\"\n")
	steps = append(steps, "          LOCK_FILE=\"${GITHUB_WORKSPACE}/.github/workflows/$GITHUB_WORKFLOW\"\n")
	steps = append(steps, "          \n")
	steps = append(steps, "          if [ -f \"$WORKFLOW_FILE\" ] && [ -f \"$LOCK_FILE\" ]; then\n")
	steps = append(steps, "            if [ \"$WORKFLOW_FILE\" -nt \"$LOCK_FILE\" ]; then\n")
	steps = append(steps, "              echo \"🔴🔴🔴 WARNING: Lock file '$LOCK_FILE' is outdated! The workflow file '$WORKFLOW_FILE' has been modified more recently. Run 'gh aw compile' to regenerate the lock file.\" >&2\n")
	steps = append(steps, "              echo \"## ⚠️ Workflow Lock File Warning\" >> $GITHUB_STEP_SUMMARY\n")
	steps = append(steps, "              echo \"🔴🔴🔴 **WARNING**: Lock file \\`$LOCK_FILE\\` is outdated!\" >> $GITHUB_STEP_SUMMARY\n")
	steps = append(steps, "              echo \"The workflow file \\`$WORKFLOW_FILE\\` has been modified more recently.\" >> $GITHUB_STEP_SUMMARY\n")
	steps = append(steps, "              echo \"Run \\`gh aw compile\\` to regenerate the lock file.\" >> $GITHUB_STEP_SUMMARY\n")
	steps = append(steps, "              echo \"\" >> $GITHUB_STEP_SUMMARY\n")
	steps = append(steps, "            fi\n")
	steps = append(steps, "          fi\n")

	// Use inlined compute-text script only if needed (no shared action)
	if data.NeedsTextOutput {
		steps = append(steps, "      - name: Compute current body text\n")
		steps = append(steps, "        id: compute-text\n")
		steps = append(steps, fmt.Sprintf("        uses: %s\n", GetActionPin("actions/github-script")))
		steps = append(steps, "        with:\n")
		steps = append(steps, "          script: |\n")

		// Inline the JavaScript directly instead of using shared action
		steps = append(steps, FormatJavaScriptForYAML(computeTextScript)...)

		// Set up outputs
		outputs["text"] = "${{ steps.compute-text.outputs.text }}"
	}

	// Add reaction step if ai-reaction is configured
	if data.AIReaction != "" {
		reactionCondition := buildReactionCondition()

		steps = append(steps, fmt.Sprintf("      - name: Add %s reaction to the triggering item\n", data.AIReaction))
		steps = append(steps, "        id: react\n")
		steps = append(steps, fmt.Sprintf("        if: %s\n", reactionCondition.Render()))
		steps = append(steps, fmt.Sprintf("        uses: %s\n", GetActionPin("actions/github-script")))

		// Add environment variables
		steps = append(steps, "        env:\n")
		steps = append(steps, fmt.Sprintf("          GH_AW_REACTION: %s\n", data.AIReaction))
		if data.Command != "" {
			steps = append(steps, fmt.Sprintf("          GH_AW_COMMAND: %s\n", data.Command))
		}
		steps = append(steps, fmt.Sprintf("          GH_AW_WORKFLOW_NAME: %q\n", data.Name))

		steps = append(steps, "        with:\n")
		steps = append(steps, "          script: |\n")

		// Add each line of the script with proper indentation
		formattedScript := FormatJavaScriptForYAML(addReactionAndEditCommentScript)
		steps = append(steps, formattedScript...)

		// Add reaction outputs
		outputs["reaction_id"] = "${{ steps.react.outputs.reaction-id }}"
		outputs["comment_id"] = "${{ steps.react.outputs.comment-id }}"
		outputs["comment_url"] = "${{ steps.react.outputs.comment-url }}"
		outputs["comment_repo"] = "${{ steps.react.outputs.comment-repo }}"
	}

	// If no steps have been added, add a dummy step to make the job valid
	// This can happen when the activation job is created only for an if condition
	if len(steps) == 0 {
		steps = append(steps, "      - run: echo \"Activation success\"\n")
	}

	// Build the conditional expression that validates activation status and other conditions
	var activationNeeds []string
	var activationCondition string

	if preActivationJobCreated {
		// Activation job depends on pre-activation job and checks the "activated" output
		activationNeeds = []string{constants.PreActivationJobName}
		activatedExpr := BuildEquals(
			BuildPropertyAccess(fmt.Sprintf("needs.%s.outputs.%s", constants.PreActivationJobName, constants.ActivatedOutput)),
			BuildStringLiteral("true"),
		)
		if data.If != "" {
			// Strip ${{ }} wrapper from data.If before combining
			unwrappedIf := stripExpressionWrapper(data.If)
			ifExpr := &ExpressionNode{Expression: unwrappedIf}
			combinedExpr := buildAnd(activatedExpr, ifExpr)
			activationCondition = combinedExpr.Render()
		} else {
			activationCondition = activatedExpr.Render()
		}
	} else {
		// No pre-activation check needed
		activationCondition = data.If
	}

	// Set permissions - add reaction permissions if reaction is configured
	var permissions string
	if data.AIReaction != "" {
		perms := NewPermissionsFromMap(map[PermissionScope]PermissionLevel{
			PermissionDiscussions:  PermissionWrite,
			PermissionIssues:       PermissionWrite,
			PermissionPullRequests: PermissionWrite,
		})
		permissions = perms.RenderToYAML()
	}

	job := &Job{
		Name:        constants.ActivationJobName,
		If:          activationCondition,
		RunsOn:      c.formatSafeOutputsRunsOn(data.SafeOutputs),
		Permissions: permissions,
		Steps:       steps,
		Outputs:     outputs,
		Needs:       activationNeeds, // Depend on pre-activation job if it exists
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
	// Permission checks are now handled by the separate check_membership job
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
		depends = []string{constants.ActivationJobName} // Depend on the activation job only if it exists
	}

	// Build outputs for all engines (GH_AW_SAFE_OUTPUTS functionality)
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

		// Set GH_AW_SAFE_OUTPUTS to fixed path
		env["GH_AW_SAFE_OUTPUTS"] = "/tmp/gh-aw/safeoutputs/outputs.jsonl"

		// Set GH_AW_SAFE_OUTPUTS_CONFIG with the safe outputs configuration
		safeOutputConfig := generateSafeOutputsConfig(data)
		if safeOutputConfig != "" {
			// The JSON string needs to be properly quoted for YAML
			env["GH_AW_SAFE_OUTPUTS_CONFIG"] = fmt.Sprintf("%q", safeOutputConfig)
		}

		// Add asset-related environment variables if upload-assets is configured
		if data.SafeOutputs.UploadAssets != nil {
			env["GH_AW_ASSETS_BRANCH"] = fmt.Sprintf("%q", data.SafeOutputs.UploadAssets.BranchName)
			env["GH_AW_ASSETS_MAX_SIZE_KB"] = fmt.Sprintf("%d", data.SafeOutputs.UploadAssets.MaxSizeKB)
			env["GH_AW_ASSETS_ALLOWED_EXTS"] = fmt.Sprintf("%q", strings.Join(data.SafeOutputs.UploadAssets.AllowedExts, ","))
		}
	}

	// Generate agent concurrency configuration
	agentConcurrency := GenerateJobConcurrencyConfig(data)

	job := &Job{
		Name:        constants.AgentJobName,
		If:          jobCondition,
		RunsOn:      c.indentYAMLLines(data.RunsOn, "    "),
		Environment: c.indentYAMLLines(data.Environment, "    "),
		Container:   c.indentYAMLLines(data.Container, "    "),
		Services:    c.indentYAMLLines(data.Services, "    "),
		Permissions: c.indentYAMLLines(data.Permissions, "    "),
		Concurrency: c.indentYAMLLines(agentConcurrency, "    "),
		Env:         env,
		Steps:       steps,
		Needs:       depends,
		Outputs:     outputs,
	}

	return job, nil
}

// generateMainJobSteps generates the steps section for the main job

// The original JavaScript code will use the pattern as-is with "g" flags

// validateMarkdownSizeForGitHubActions is no longer used - content is now split into multiple steps
// to handle GitHub Actions script size limits automatically
// func (c *Compiler) validateMarkdownSizeForGitHubActions(content string) error { ... }

// splitContentIntoChunks splits markdown content into chunks that fit within GitHub Actions script size limits

// generateCacheMemoryPromptStep generates a separate step for cache memory prompt section

// generateSafeOutputsPromptStep generates a separate step for safe outputs prompt section

// generatePostSteps generates the post-steps section that runs after AI execution
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
							// Apply action pinning before converting to YAML
							stepMap = ApplyActionPinToStep(stepMap)

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

// generateEngineExecutionSteps uses the new GetExecutionSteps interface method

// generateAgentVersionCapture generates a step that captures the agent version if the engine supports it

// generateCreateAwInfo generates a step that creates aw_info.json with agentic run metadata

// generateOutputCollectionStep generates a step that reads the output file and sets it as a GitHub Actions output
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

// computeAllowedDomainsForSanitization computes the allowed domains for sanitization
// based on the engine and network configuration, matching what's provided to the firewall
func (c *Compiler) computeAllowedDomainsForSanitization(data *WorkflowData) string {
	// Determine which engine is being used
	var engineID string
	if data.EngineConfig != nil {
		engineID = data.EngineConfig.ID
	} else if data.AI != "" {
		engineID = data.AI
	}

	// Compute domains based on engine type
	// For Copilot with firewall support, use GetCopilotAllowedDomains which merges
	// Copilot defaults with network permissions
	// For other engines, use GetAllowedDomains which uses network permissions only
	if engineID == "copilot" {
		return GetCopilotAllowedDomains(data.NetworkPermissions)
	}

	// For Claude, Codex, and other engines, use network permissions
	domains := GetAllowedDomains(data.NetworkPermissions)
	return strings.Join(domains, ",")
}
