package workflow

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/githubnext/gh-aw/pkg/console"
	"github.com/githubnext/gh-aw/pkg/constants"
	"github.com/githubnext/gh-aw/pkg/parser"
	"github.com/goccy/go-yaml"
	"github.com/santhosh-tekuri/jsonschema/v6"
)

const (
	// OutputArtifactName is the standard name for GITHUB_AW_SAFE_OUTPUTS artifact
	OutputArtifactName = "aw_output.txt"
)

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
	FrontmatterName    string // name field from frontmatter (for security report driver default)
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
	AllowedTools       string
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
}

// SafeOutputsConfig holds configuration for automatic output routes
type SafeOutputsConfig struct {
	CreateIssues                    *CreateIssuesConfig                    `yaml:"create-issue,omitempty"`
	CreateDiscussions               *CreateDiscussionsConfig               `yaml:"create-discussion,omitempty"`
	AddIssueComments                *AddIssueCommentsConfig                `yaml:"add-issue-comment,omitempty"`
	CreatePullRequests              *CreatePullRequestsConfig              `yaml:"create-pull-request,omitempty"`
	CreatePullRequestReviewComments *CreatePullRequestReviewCommentsConfig `yaml:"create-pull-request-review-comment,omitempty"`
	CreateSecurityReports           *CreateSecurityReportsConfig           `yaml:"create-security-report,omitempty"`
	AddIssueLabels                  *AddIssueLabelsConfig                  `yaml:"add-issue-label,omitempty"`
	UpdateIssues                    *UpdateIssuesConfig                    `yaml:"update-issue,omitempty"`
	PushToBranch                    *PushToBranchConfig                    `yaml:"push-to-branch,omitempty"`
	MissingTool                     *MissingToolConfig                     `yaml:"missing-tool,omitempty"` // Optional for reporting missing functionality
	AllowedDomains                  []string                               `yaml:"allowed-domains,omitempty"`
}

// CreateIssuesConfig holds configuration for creating GitHub issues from agent output
type CreateIssuesConfig struct {
	TitlePrefix string   `yaml:"title-prefix,omitempty"`
	Labels      []string `yaml:"labels,omitempty"`
	Max         int      `yaml:"max,omitempty"` // Maximum number of issues to create
}

// CreateDiscussionsConfig holds configuration for creating GitHub discussions from agent output
type CreateDiscussionsConfig struct {
	TitlePrefix string `yaml:"title-prefix,omitempty"`
	CategoryId  string `yaml:"category-id,omitempty"` // Discussion category ID
	Max         int    `yaml:"max,omitempty"`         // Maximum number of discussions to create
}

// AddIssueCommentConfig holds configuration for creating GitHub issue/PR comments from agent output (deprecated, use AddIssueCommentsConfig)
type AddIssueCommentConfig struct {
	// Empty struct for now, as per requirements, but structured for future expansion
}

// AddIssueCommentsConfig holds configuration for creating GitHub issue/PR comments from agent output
type AddIssueCommentsConfig struct {
	Max    int    `yaml:"max,omitempty"`    // Maximum number of comments to create
	Target string `yaml:"target,omitempty"` // Target for comments: "triggering" (default), "*" (any issue), or explicit issue number
}

// CreatePullRequestsConfig holds configuration for creating GitHub pull requests from agent output
type CreatePullRequestsConfig struct {
	TitlePrefix string   `yaml:"title-prefix,omitempty"`
	Labels      []string `yaml:"labels,omitempty"`
	Draft       *bool    `yaml:"draft,omitempty"` // Pointer to distinguish between unset (nil) and explicitly false
	Max         int      `yaml:"max,omitempty"`   // Maximum number of pull requests to create
}

// CreatePullRequestReviewCommentsConfig holds configuration for creating GitHub pull request review comments from agent output
type CreatePullRequestReviewCommentsConfig struct {
	Max  int    `yaml:"max,omitempty"`  // Maximum number of review comments to create (default: 1)
	Side string `yaml:"side,omitempty"` // Side of the diff: "LEFT" or "RIGHT" (default: "RIGHT")
}

// CreateSecurityReportsConfig holds configuration for creating security reports (SARIF format) from agent output
type CreateSecurityReportsConfig struct {
	Max    int    `yaml:"max,omitempty"`    // Maximum number of security findings to include (default: unlimited)
	Driver string `yaml:"driver,omitempty"` // Driver name for SARIF tool.driver.name field (default: "GitHub Agentic Workflows Security Scanner")
}

// AddIssueLabelsConfig holds configuration for adding labels to issues/PRs from agent output
type AddIssueLabelsConfig struct {
	Allowed  []string `yaml:"allowed,omitempty"` // Optional list of allowed labels. If omitted, any labels are allowed (including creating new ones).
	MaxCount *int     `yaml:"max,omitempty"`     // Optional maximum number of labels to add (default: 3)
}

// UpdateIssuesConfig holds configuration for updating GitHub issues from agent output
type UpdateIssuesConfig struct {
	Status *bool  `yaml:"status,omitempty"` // Allow updating issue status (open/closed) - presence indicates field can be updated
	Target string `yaml:"target,omitempty"` // Target for updates: "triggering" (default), "*" (any issue), or explicit issue number
	Title  *bool  `yaml:"title,omitempty"`  // Allow updating issue title - presence indicates field can be updated
	Body   *bool  `yaml:"body,omitempty"`   // Allow updating issue body - presence indicates field can be updated
	Max    int    `yaml:"max,omitempty"`    // Maximum number of issues to update (default: 1)
}

// PushToBranchConfig holds configuration for pushing changes to a specific branch from agent output
type PushToBranchConfig struct {
	Branch string `yaml:"branch"`           // The branch to push changes to (defaults to "triggering")
	Target string `yaml:"target,omitempty"` // Target for push-to-branch: like add-issue-comment but for pull requests
}

// MissingToolConfig holds configuration for reporting missing tools or functionality
type MissingToolConfig struct {
	Max int `yaml:"max,omitempty"` // Maximum number of missing tool reports (default: unlimited)
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

	// Validate generated YAML against GitHub Actions schema (unless skipped)
	if !c.skipValidation {
		if c.verbose {
			fmt.Println(console.FormatInfoMessage("Validating workflow against GitHub Actions schema..."))
		}
		if err := c.validateWorkflowSchema(yamlContent); err != nil {
			formattedErr := console.FormatError(console.CompilerError{
				Position: console.ErrorPosition{
					File:   markdownPath,
					Line:   1,
					Column: 1,
				},
				Type:    "error",
				Message: fmt.Sprintf("workflow validation failed: %v", err),
			})
			return errors.New(formattedErr)
		}

		if c.verbose {
			fmt.Println(console.FormatSuccessMessage("Workflow validation passed"))
		}
	} else if c.verbose {
		fmt.Println(console.FormatWarningMessage("Schema validation available but skipped (use SetSkipValidation(false) to enable)"))
	}

	// Write to lock file
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

	fmt.Println(console.FormatSuccessMessage(console.ToRelativePath(markdownPath)))
	return nil
}

// httpURLLoader implements URLLoader for HTTP(S) URLs
type httpURLLoader struct {
	client *http.Client
}

// Load implements URLLoader interface for HTTP URLs
func (h *httpURLLoader) Load(url string) (any, error) {
	resp, err := h.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch URL %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch URL %s: HTTP %d", url, resp.StatusCode)
	}

	var result interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode JSON from %s: %w", url, err)
	}

	return result, nil
}

// validateWorkflowSchema validates the generated YAML content against the GitHub Actions workflow schema
func (c *Compiler) validateWorkflowSchema(yamlContent string) error {
	// Convert YAML to JSON for validation
	var workflowData interface{}
	if err := yaml.Unmarshal([]byte(yamlContent), &workflowData); err != nil {
		return fmt.Errorf("failed to parse generated YAML: %w", err)
	}

	// Convert to JSON
	jsonData, err := json.Marshal(workflowData)
	if err != nil {
		return fmt.Errorf("failed to convert YAML to JSON: %w", err)
	}

	// Load GitHub Actions workflow schema from SchemaStore
	schemaURL := "https://raw.githubusercontent.com/SchemaStore/schemastore/master/src/schemas/json/github-workflow.json"

	// Create compiler with HTTP loader
	loader := jsonschema.NewCompiler()
	httpLoader := &httpURLLoader{
		client: &http.Client{Timeout: 30 * time.Second},
	}

	// Configure the compiler to use HTTP loader for https and http schemes
	schemeLoader := jsonschema.SchemeURLLoader{
		"https": httpLoader,
		"http":  httpLoader,
	}
	loader.UseLoader(schemeLoader)

	schema, err := loader.Compile(schemaURL)
	if err != nil {
		return fmt.Errorf("failed to load GitHub Actions schema from %s: %w", schemaURL, err)
	}

	// Validate the JSON data against the schema
	var jsonObj interface{}
	if err := json.Unmarshal(jsonData, &jsonObj); err != nil {
		return fmt.Errorf("failed to unmarshal JSON for validation: %w", err)
	}

	if err := schema.Validate(jsonObj); err != nil {
		return fmt.Errorf("workflow schema validation failed: %w", err)
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

	// Default to full network access if no network permissions specified
	if networkPermissions == nil && engineConfig != nil && engineConfig.ID == "claude" {
		// Default to "defaults" mode (full network access for now)
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

	if !agenticEngine.SupportsToolsWhitelist() {
		// For engines that don't support tool whitelists (like codex), ignore tools section and provide warnings
		fmt.Println(console.FormatWarningMessage(fmt.Sprintf("Using experimental %s support (engine: %s)", agenticEngine.GetDisplayName(), engineSetting)))
		tools = make(map[string]any)
		if _, hasTools := result.Frontmatter["tools"]; hasTools {
			fmt.Println(console.FormatWarningMessage(fmt.Sprintf("'tools' section ignored when using engine: %s (%s doesn't support MCP tool allow-listing)", engineSetting, agenticEngine.GetDisplayName())))
		}
		// Force docker version of GitHub MCP if github tool would be needed
		// For now, we'll add a basic github tool (always uses docker MCP)
		githubConfig := map[string]any{}

		tools["github"] = githubConfig
	} else {
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

		// Apply default GitHub MCP tools (only for engines that support MCP)
		if agenticEngine.SupportsToolsWhitelist() {
			tools = c.applyDefaultGitHubMCPAndClaudeTools(tools, safeOutputs)
		}

		if c.verbose && len(tools) > 0 {
			fmt.Println(console.FormatInfoMessage(fmt.Sprintf("Merged tools: %d total tools configured", len(tools))))
		}
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

	if c.verbose {
		fmt.Println(console.FormatInfoMessage(fmt.Sprintf("Extracted workflow name: '%s'", workflowName)))
	}

	// Check if the markdown content uses the text output
	needsTextOutput := c.detectTextOutputUsage(markdownContent)

	// Build workflow data
	workflowData := &WorkflowData{
		Name:               workflowName,
		FrontmatterName:    c.extractStringValue(result.Frontmatter, "name"),
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
	workflowData.If = c.extractTopLevelYAMLSection(result.Frontmatter, "if")
	workflowData.TimeoutMinutes = c.extractTopLevelYAMLSection(result.Frontmatter, "timeout_minutes")
	workflowData.CustomSteps = c.extractTopLevelYAMLSection(result.Frontmatter, "steps")
	workflowData.PostSteps = c.extractTopLevelYAMLSection(result.Frontmatter, "post-steps")
	workflowData.RunsOn = c.extractTopLevelYAMLSection(result.Frontmatter, "runs-on")
	workflowData.Cache = c.extractTopLevelYAMLSection(result.Frontmatter, "cache")

	// Extract stop-after from the on: section
	stopAfter, err := c.extractStopAfterFromOn(result.Frontmatter)
	if err != nil {
		return nil, err
	}
	workflowData.StopTime = stopAfter

	// Resolve relative stop-after to absolute time if needed
	if workflowData.StopTime != "" {
		resolvedStopTime, err := resolveStopTime(workflowData.StopTime, time.Now().UTC())
		if err != nil {
			return nil, fmt.Errorf("invalid stop-after format: %w", err)
		}
		originalStopTime := stopAfter
		workflowData.StopTime = resolvedStopTime

		if c.verbose && isRelativeStopTime(originalStopTime) {
			fmt.Println(console.FormatInfoMessage(fmt.Sprintf("Resolved relative stop-after to: %s", resolvedStopTime)))
		} else if c.verbose && originalStopTime != resolvedStopTime {
			fmt.Println(console.FormatInfoMessage(fmt.Sprintf("Parsed absolute stop-after from '%s' to: %s", originalStopTime, resolvedStopTime)))
		}
	}

	workflowData.Command = c.extractCommandName(result.Frontmatter)
	workflowData.Jobs = c.extractJobsFromFrontmatter(result.Frontmatter)

	// Use the already extracted output configuration
	workflowData.SafeOutputs = safeOutputs

	// Check if "command" is used as a trigger in the "on" section
	// Also extract "reaction" from the "on" section
	var hasCommand bool
	var hasReaction bool
	var hasStopAfter bool
	var otherEvents map[string]any
	if onValue, exists := result.Frontmatter["on"]; exists {
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
						return nil, fmt.Errorf("cannot use 'command' with '%s' in the same workflow", eventName)
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
			workflowData.On = c.extractTopLevelYAMLSection(result.Frontmatter, "on")
		}
	}

	// Apply defaults
	c.applyDefaults(workflowData, markdownPath)

	// Apply pull request draft filter if specified
	c.applyPullRequestDraftFilter(workflowData, result.Frontmatter)

	// Apply pull request fork filter if specified
	c.applyPullRequestForkFilter(workflowData, result.Frontmatter)

	// Compute allowed tools
	workflowData.AllowedTools = c.computeAllowedTools(tools, workflowData.SafeOutputs)

	return workflowData, nil
}

// extractNetworkPermissions extracts network permissions from frontmatter
func (c *Compiler) extractNetworkPermissions(frontmatter map[string]any) *NetworkPermissions {
	if network, exists := frontmatter["network"]; exists {
		// Handle string format: "defaults"
		if networkStr, ok := network.(string); ok {
			if networkStr == "defaults" {
				return &NetworkPermissions{
					Mode: "defaults",
				}
			}
			// Unknown string format, return nil
			return nil
		}

		// Handle object format: { allowed: [...] } or {}
		if networkObj, ok := network.(map[string]any); ok {
			permissions := &NetworkPermissions{}

			// Extract allowed domains if present
			if allowed, hasAllowed := networkObj["allowed"]; hasAllowed {
				if allowedSlice, ok := allowed.([]any); ok {
					for _, domain := range allowedSlice {
						if domainStr, ok := domain.(string); ok {
							permissions.Allowed = append(permissions.Allowed, domainStr)
						}
					}
				}
			}
			// Empty object {} means no network access (empty allowed list)
			return permissions
		}
	}
	return nil
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

// extractStringValue extracts a string value from the frontmatter map
func (c *Compiler) extractStringValue(frontmatter map[string]any, key string) string {
	value, exists := frontmatter[key]
	if !exists {
		return ""
	}

	if strValue, ok := value.(string); ok {
		return strValue
	}

	return ""
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

// extractYAMLValue extracts a scalar value from the frontmatter map
func (c *Compiler) extractYAMLValue(frontmatter map[string]any, key string) string {
	if value, exists := frontmatter[key]; exists {
		if str, ok := value.(string); ok {
			return str
		}
		if num, ok := value.(int); ok {
			return fmt.Sprintf("%d", num)
		}
		if num, ok := value.(int64); ok {
			return fmt.Sprintf("%d", num)
		}
		if num, ok := value.(uint64); ok {
			return fmt.Sprintf("%d", num)
		}
		if float, ok := value.(float64); ok {
			return fmt.Sprintf("%.0f", float)
		}
	}
	return ""
}

// extractStopAfterFromOn extracts the stop-after value from the on: section
func (c *Compiler) extractStopAfterFromOn(frontmatter map[string]any) (string, error) {
	onSection, exists := frontmatter["on"]
	if !exists {
		return "", nil
	}

	// Handle different formats of the on: section
	switch on := onSection.(type) {
	case string:
		// Simple string format like "on: push" - no stop-after possible
		return "", nil
	case map[string]any:
		// Complex object format - look for stop-after
		if stopAfter, exists := on["stop-after"]; exists {
			if str, ok := stopAfter.(string); ok {
				return str, nil
			}
			return "", fmt.Errorf("stop-after value must be a string")
		}
		return "", nil
	default:
		return "", fmt.Errorf("invalid on: section format")
	}
}

// filterMapKeys creates a new map excluding the specified keys
func filterMapKeys(original map[string]any, excludeKeys ...string) map[string]any {
	excludeSet := make(map[string]bool)
	for _, key := range excludeKeys {
		excludeSet[key] = true
	}

	result := make(map[string]any)
	for key, value := range original {
		if !excludeSet[key] {
			result[key] = value
		}
	}
	return result
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
				data.If = fmt.Sprintf("if: %s", commandConditionTree.Render())
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
	existingCondition := strings.TrimPrefix(data.If, "if: ")
	conditionTree := buildConditionTree(existingCondition, draftCondition.Render())
	data.If = fmt.Sprintf("if: %s", conditionTree.Render())
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
	existingCondition := strings.TrimPrefix(data.If, "if: ")
	conditionTree := buildConditionTree(existingCondition, forkCondition.Render())
	data.If = fmt.Sprintf("if: %s", conditionTree.Render())
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

// applyDefaultGitHubMCPAndClaudeTools adds default read-only GitHub MCP tools, creating github tool if not present
func (c *Compiler) applyDefaultGitHubMCPAndClaudeTools(tools map[string]any, safeOutputs *SafeOutputsConfig) map[string]any {
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

	defaultClaudeTools := []string{
		"Task",
		"Glob",
		"Grep",
		"ExitPlanMode",
		"TodoWrite",
		"LS",
		"Read",
		"NotebookRead",
	}

	// Ensure claude section exists with the new format
	var claudeSection map[string]any
	if existing, hasClaudeSection := tools["claude"]; hasClaudeSection {
		if claudeMap, ok := existing.(map[string]any); ok {
			claudeSection = claudeMap
		} else {
			claudeSection = make(map[string]any)
		}
	} else {
		claudeSection = make(map[string]any)
	}

	// Get existing allowed tools from the new format (map structure)
	var claudeExistingAllowed map[string]any
	if allowed, hasAllowed := claudeSection["allowed"]; hasAllowed {
		if allowedMap, ok := allowed.(map[string]any); ok {
			claudeExistingAllowed = allowedMap
		} else {
			claudeExistingAllowed = make(map[string]any)
		}
	} else {
		claudeExistingAllowed = make(map[string]any)
	}

	// Add default tools that aren't already present
	for _, defaultTool := range defaultClaudeTools {
		if _, exists := claudeExistingAllowed[defaultTool]; !exists {
			claudeExistingAllowed[defaultTool] = nil // Add tool with null value
		}
	}

	// Add Git commands and file editing tools when safe-outputs includes create-pull-request or push-to-branch
	if safeOutputs != nil && needsGitCommands(safeOutputs) {
		gitCommands := []any{
			"git checkout:*",
			"git branch:*",
			"git switch:*",
			"git add:*",
			"git rm:*",
			"git commit:*",
			"git merge:*",
		}

		// Add additional Claude tools needed for file editing and pull request creation
		additionalTools := []string{
			"Edit",
			"MultiEdit",
			"Write",
			"NotebookEdit",
		}

		// Add file editing tools that aren't already present
		for _, tool := range additionalTools {
			if _, exists := claudeExistingAllowed[tool]; !exists {
				claudeExistingAllowed[tool] = nil // Add tool with null value
			}
		}

		// Add Bash tool with Git commands if not already present
		if _, exists := claudeExistingAllowed["Bash"]; !exists {
			// Bash tool doesn't exist, add it with Git commands
			claudeExistingAllowed["Bash"] = gitCommands
		} else {
			// Bash tool exists, merge Git commands with existing commands
			existingBash := claudeExistingAllowed["Bash"]
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
				claudeExistingAllowed["Bash"] = newCommands
			} else if existingBash == nil {
				// Bash tool exists but with nil value (allows all commands)
				// Keep it as nil since that's more permissive than specific commands
				// No action needed - nil value already permits all commands
				_ = existingBash // Keep the nil value as-is
			}
		}
	bashComplete:
	}

	// Check if Bash tools are present and add implicit KillBash and BashOutput
	if _, hasBash := claudeExistingAllowed["Bash"]; hasBash {
		// Implicitly add KillBash and BashOutput when any Bash tools are allowed
		if _, exists := claudeExistingAllowed["KillBash"]; !exists {
			claudeExistingAllowed["KillBash"] = nil
		}
		if _, exists := claudeExistingAllowed["BashOutput"]; !exists {
			claudeExistingAllowed["BashOutput"] = nil
		}
	}

	// Update the claude section with the new format
	claudeSection["allowed"] = claudeExistingAllowed
	tools["claude"] = claudeSection

	return tools
}

// needsGitCommands checks if safe outputs configuration requires Git commands
func needsGitCommands(safeOutputs *SafeOutputsConfig) bool {
	if safeOutputs == nil {
		return false
	}
	return safeOutputs.CreatePullRequests != nil || safeOutputs.PushToBranch != nil
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

// computeAllowedTools computes the comma-separated list of allowed tools for Claude
func (c *Compiler) computeAllowedTools(tools map[string]any, safeOutputs *SafeOutputsConfig) string {
	var allowedTools []string

	// Process claude-specific tools from the claude section (new format only)
	if claudeSection, hasClaudeSection := tools["claude"]; hasClaudeSection {
		if claudeConfig, ok := claudeSection.(map[string]any); ok {
			if allowed, hasAllowed := claudeConfig["allowed"]; hasAllowed {
				// In the new format, allowed is a map where keys are tool names
				if allowedMap, ok := allowed.(map[string]any); ok {
					for toolName, toolValue := range allowedMap {
						if toolName == "Bash" {
							// Handle Bash tool with specific commands
							if bashCommands, ok := toolValue.([]any); ok {
								// Check for :* wildcard first - if present, ignore all other bash commands
								for _, cmd := range bashCommands {
									if cmdStr, ok := cmd.(string); ok {
										if cmdStr == ":*" {
											// :* means allow all bash and ignore other commands
											allowedTools = append(allowedTools, "Bash")
											goto nextClaudeTool
										}
									}
								}
								// Process the allowed bash commands (no :* found)
								for _, cmd := range bashCommands {
									if cmdStr, ok := cmd.(string); ok {
										if cmdStr == "*" {
											// Wildcard means allow all bash
											allowedTools = append(allowedTools, "Bash")
											goto nextClaudeTool
										}
									}
								}
								// Add individual bash commands with Bash() prefix
								for _, cmd := range bashCommands {
									if cmdStr, ok := cmd.(string); ok {
										allowedTools = append(allowedTools, fmt.Sprintf("Bash(%s)", cmdStr))
									}
								}
							} else {
								// Bash with no specific commands or null value - allow all bash
								allowedTools = append(allowedTools, "Bash")
							}
						} else if strings.HasPrefix(toolName, strings.ToUpper(toolName[:1])) {
							// Tool name starts with uppercase letter - regular Claude tool
							allowedTools = append(allowedTools, toolName)
						}
					nextClaudeTool:
					}
				}
			}
		}
	}

	// Process top-level tools (MCP tools and claude)
	for toolName, toolValue := range tools {
		if toolName == "claude" {
			// Skip the claude section as we've already processed it
			continue
		} else {
			// Check if this is an MCP tool (has MCP-compatible type) or standard MCP tool (github)
			if mcpConfig, ok := toolValue.(map[string]any); ok {
				// Check if it's explicitly marked as MCP type
				isCustomMCP := false
				if hasMcp, _ := hasMCPConfig(mcpConfig); hasMcp {
					isCustomMCP = true
				}

				// Handle standard MCP tools (github) or tools with MCP-compatible type
				if toolName == "github" || isCustomMCP {
					if allowed, hasAllowed := mcpConfig["allowed"]; hasAllowed {
						if allowedSlice, ok := allowed.([]any); ok {
							// Check for wildcard access first
							hasWildcard := false
							for _, item := range allowedSlice {
								if str, ok := item.(string); ok && str == "*" {
									hasWildcard = true
									break
								}
							}

							if hasWildcard {
								// For wildcard access, just add the server name with mcp__ prefix
								allowedTools = append(allowedTools, fmt.Sprintf("mcp__%s", toolName))
							} else {
								// For specific tools, add each one individually
								for _, item := range allowedSlice {
									if str, ok := item.(string); ok {
										allowedTools = append(allowedTools, fmt.Sprintf("mcp__%s__%s", toolName, str))
									}
								}
							}
						}
					}
				}
			}
		}
	}

	// Handle SafeOutputs requirement for file write access
	if safeOutputs != nil {
		// Check if a general "Write" permission is already granted
		hasGeneralWrite := slices.Contains(allowedTools, "Write")

		// If no general Write permission and SafeOutputs is configured,
		// add specific write permission for GITHUB_AW_SAFE_OUTPUTS
		if !hasGeneralWrite {
			allowedTools = append(allowedTools, "Write")
			// Ideally we would only give permission to the exact file, but that doesn't seem
			// to be working with Claude. See https://github.com/githubnext/gh-aw/issues/244#issuecomment-3240319103
			//allowedTools = append(allowedTools, "Write(${{ env.GITHUB_AW_SAFE_OUTPUTS }})")
		}
	}

	// Sort the allowed tools alphabetically for consistent output
	sort.Strings(allowedTools)

	return strings.Join(allowedTools, ",")
}

// generateAllowedToolsComment generates a multi-line comment showing each allowed tool
func (c *Compiler) generateAllowedToolsComment(allowedToolsStr string, indent string) string {
	if allowedToolsStr == "" {
		return ""
	}

	tools := strings.Split(allowedToolsStr, ",")
	if len(tools) == 0 {
		return ""
	}

	var comment strings.Builder
	comment.WriteString(indent + "# Allowed tools (sorted):\n")
	for _, tool := range tools {
		comment.WriteString(fmt.Sprintf("%s# - %s\n", indent, tool))
	}

	return comment.String()
}

// indentYAMLLines adds indentation to all lines of a multi-line YAML string except the first
func (c *Compiler) indentYAMLLines(yamlContent, indent string) string {
	if yamlContent == "" {
		return yamlContent
	}

	lines := strings.Split(yamlContent, "\n")
	if len(lines) <= 1 {
		return yamlContent
	}

	// First line doesn't get additional indentation
	result := lines[0]
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) != "" {
			result += "\n" + indent + lines[i]
		} else {
			result += "\n" + lines[i]
		}
	}

	return result
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
func (c *Compiler) isTaskJobNeeded(data *WorkflowData) bool {
	// Task job is needed if:
	// 1. Command is configured (for team member checking)
	// 2. Text output is needed (for compute-text action)
	// 3. If condition is specified (to handle runtime conditions)
	return data.Command != "" || data.NeedsTextOutput || data.If != ""
}

// buildJobs creates all jobs for the workflow and adds them to the job manager
func (c *Compiler) buildJobs(data *WorkflowData, markdownPath string) error {
	// Generate job name from workflow name
	jobName := c.generateJobName(data.Name)

	// Build task job only if actually needed (preamble job that handles runtime conditions)
	var taskJobCreated bool
	if c.isTaskJobNeeded(data) {
		taskJob, err := c.buildTaskJob(data)
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
		addReactionJob, err := c.buildAddReactionJob(data, taskJobCreated)
		if err != nil {
			return fmt.Errorf("failed to build add_reaction job: %w", err)
		}
		if err := c.jobManager.AddJob(addReactionJob); err != nil {
			return fmt.Errorf("failed to add add_reaction job: %w", err)
		}
	}

	// Build main workflow job
	mainJob, err := c.buildMainJob(data, jobName, taskJobCreated)
	if err != nil {
		return fmt.Errorf("failed to build main job: %w", err)
	}
	if err := c.jobManager.AddJob(mainJob); err != nil {
		return fmt.Errorf("failed to add main job: %w", err)
	}

	if data.SafeOutputs != nil {
		// Build create_issue job if output.create_issue is configured
		if data.SafeOutputs.CreateIssues != nil {
			createIssueJob, err := c.buildCreateOutputIssueJob(data, jobName)
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

		// Build create_issue_comment job if output.add-issue-comment is configured
		if data.SafeOutputs.AddIssueComments != nil {
			createCommentJob, err := c.buildCreateOutputAddIssueCommentJob(data, jobName)
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

		// Build create_security_report job if output.create-security-report is configured
		if data.SafeOutputs.CreateSecurityReports != nil {
			// Extract the workflow filename without extension for rule ID prefix
			workflowFilename := strings.TrimSuffix(filepath.Base(markdownPath), ".md")
			createSecurityReportJob, err := c.buildCreateOutputSecurityReportJob(data, jobName, workflowFilename)
			if err != nil {
				return fmt.Errorf("failed to build create_security_report job: %w", err)
			}
			if err := c.jobManager.AddJob(createSecurityReportJob); err != nil {
				return fmt.Errorf("failed to add create_security_report job: %w", err)
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

		// Build add_labels job if output.add-issue-label is configured (including null/empty)
		if data.SafeOutputs.AddIssueLabels != nil {
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

		// Build push_to_branch job if output.push-to-branch is configured
		if data.SafeOutputs.PushToBranch != nil {
			pushToBranchJob, err := c.buildCreateOutputPushToBranchJob(data, jobName)
			if err != nil {
				return fmt.Errorf("failed to build push_to_branch job: %w", err)
			}
			if err := c.jobManager.AddJob(pushToBranchJob); err != nil {
				return fmt.Errorf("failed to add push_to_branch job: %w", err)
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
	}
	// Build additional custom jobs from frontmatter jobs section
	if err := c.buildCustomJobs(data); err != nil {
		return fmt.Errorf("failed to build custom jobs: %w", err)
	}

	return nil
}

// buildTaskJob creates the preamble task job that acts as a barrier for runtime conditions
func (c *Compiler) buildTaskJob(data *WorkflowData) (*Job, error) {
	outputs := map[string]string{}
	var steps []string

	// Add team member check for command workflows, but only when triggered by command mention
	if data.Command != "" {
		// Build condition that only applies to command mentions in comment-related events
		commandCondition := buildCommandOnlyCondition(data.Command)
		commandConditionStr := commandCondition.Render()

		// Build the validation condition using expression nodes
		// Since the check-team-member step is gated by command condition, we check if it ran and returned 'false'
		// This avoids running validation when the step didn't run at all (non-command triggers)
		validationCondition := BuildEquals(
			BuildPropertyAccess("steps.check-team-member.outputs.is_team_member"),
			BuildStringLiteral("false"),
		)
		validationConditionStr := validationCondition.Render()

		steps = append(steps, "      - name: Check team membership for command workflow\n")
		steps = append(steps, "        id: check-team-member\n")
		steps = append(steps, fmt.Sprintf("        if: %s\n", commandConditionStr))
		steps = append(steps, "        uses: actions/github-script@v7\n")
		steps = append(steps, "        with:\n")
		steps = append(steps, "          script: |\n")

		// Inline the JavaScript code with proper indentation
		scriptLines := strings.Split(checkTeamMemberScript, "\n")
		for _, line := range scriptLines {
			if strings.TrimSpace(line) != "" {
				steps = append(steps, fmt.Sprintf("            %s\n", line))
			}
		}
		steps = append(steps, "      - name: Validate team membership\n")
		steps = append(steps, fmt.Sprintf("        if: %s\n", validationConditionStr))
		steps = append(steps, "        run: |\n")
		steps = append(steps, "          echo \"❌ Access denied: Only team members can trigger command workflows\"\n")
		steps = append(steps, "          echo \"User ${{ github.actor }} is not a team member\"\n")
		steps = append(steps, "          exit 1\n")
	}

	// Use inlined compute-text script only if needed (no shared action)
	if data.NeedsTextOutput {
		steps = append(steps, "      - name: Compute current body text\n")
		steps = append(steps, "        id: compute-text\n")
		steps = append(steps, "        uses: actions/github-script@v7\n")
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
		Permissions: "", // No permissions needed - task job does not require content access
		Steps:       steps,
		Outputs:     outputs,
	}

	return job, nil
}

// buildAddReactionJob creates the add_reaction job
func (c *Compiler) buildAddReactionJob(data *WorkflowData, taskJobCreated bool) (*Job, error) {
	reactionCondition := buildReactionCondition()

	var steps []string
	steps = append(steps, fmt.Sprintf("      - name: Add %s reaction to the triggering item\n", data.AIReaction))
	steps = append(steps, "        id: react\n")
	steps = append(steps, "        uses: actions/github-script@v7\n")

	// Add environment variables
	steps = append(steps, "        env:\n")
	steps = append(steps, fmt.Sprintf("          GITHUB_AW_REACTION: %s\n", data.AIReaction))
	if data.Command != "" {
		steps = append(steps, fmt.Sprintf("          GITHUB_AW_COMMAND: %s\n", data.Command))
	}

	steps = append(steps, "        with:\n")
	steps = append(steps, "          script: |\n")

	// Add each line of the script with proper indentation
	formattedScript := FormatJavaScriptForYAML(addReactionAndEditCommentScript)
	steps = append(steps, formattedScript...)

	outputs := map[string]string{
		"reaction_id": "${{ steps.react.outputs.reaction-id }}",
	}

	var depends []string
	if taskJobCreated {
		depends = []string{"task"} // Depend on the task job only if it exists
	}

	job := &Job{
		Name:        "add_reaction",
		If:          fmt.Sprintf("if: %s", reactionCondition.Render()),
		RunsOn:      "runs-on: ubuntu-latest",
		Permissions: "permissions:\n      issues: write\n      pull-requests: write",
		Steps:       steps,
		Outputs:     outputs,
		Depends:     depends,
	}

	return job, nil
}

// buildCreateOutputIssueJob creates the create_issue job
func (c *Compiler) buildCreateOutputIssueJob(data *WorkflowData, mainJobName string) (*Job, error) {
	if data.SafeOutputs == nil || data.SafeOutputs.CreateIssues == nil {
		return nil, fmt.Errorf("safe-outputs.create-issue configuration is required")
	}

	var steps []string
	steps = append(steps, "      - name: Create Output Issue\n")
	steps = append(steps, "        id: create_issue\n")
	steps = append(steps, "        uses: actions/github-script@v7\n")

	// Add environment variables
	steps = append(steps, "        env:\n")
	// Pass the agent output content from the main job
	steps = append(steps, fmt.Sprintf("          GITHUB_AW_AGENT_OUTPUT: ${{ needs.%s.outputs.output }}\n", mainJobName))
	if data.SafeOutputs.CreateIssues.TitlePrefix != "" {
		steps = append(steps, fmt.Sprintf("          GITHUB_AW_ISSUE_TITLE_PREFIX: %q\n", data.SafeOutputs.CreateIssues.TitlePrefix))
	}
	if len(data.SafeOutputs.CreateIssues.Labels) > 0 {
		labelsStr := strings.Join(data.SafeOutputs.CreateIssues.Labels, ",")
		steps = append(steps, fmt.Sprintf("          GITHUB_AW_ISSUE_LABELS: %q\n", labelsStr))
	}

	steps = append(steps, "        with:\n")
	steps = append(steps, "          script: |\n")

	// Add each line of the script with proper indentation
	formattedScript := FormatJavaScriptForYAML(createIssueScript)
	steps = append(steps, formattedScript...)

	// Create outputs for the job
	outputs := map[string]string{
		"issue_number": "${{ steps.create_issue.outputs.issue_number }}",
		"issue_url":    "${{ steps.create_issue.outputs.issue_url }}",
	}

	// Determine the job condition for command workflows
	var jobCondition string
	if data.Command != "" {
		// Build the command trigger condition
		commandCondition := buildCommandOnlyCondition(data.Command)
		commandConditionStr := commandCondition.Render()
		jobCondition = fmt.Sprintf("if: %s", commandConditionStr)
	} else {
		jobCondition = "" // No conditional execution
	}

	job := &Job{
		Name:           "create_issue",
		If:             jobCondition,
		RunsOn:         "runs-on: ubuntu-latest",
		Permissions:    "permissions:\n      contents: read\n      issues: write",
		TimeoutMinutes: 10, // 10-minute timeout as required
		Steps:          steps,
		Outputs:        outputs,
		Depends:        []string{mainJobName}, // Depend on the main workflow job
	}

	return job, nil
}

// buildCreateOutputDiscussionJob creates the create_discussion job
func (c *Compiler) buildCreateOutputDiscussionJob(data *WorkflowData, mainJobName string) (*Job, error) {
	if data.SafeOutputs == nil || data.SafeOutputs.CreateDiscussions == nil {
		return nil, fmt.Errorf("safe-outputs.create-discussion configuration is required")
	}

	var steps []string
	steps = append(steps, "      - name: Create Output Discussion\n")
	steps = append(steps, "        id: create_discussion\n")
	steps = append(steps, "        uses: actions/github-script@v7\n")

	// Add environment variables
	steps = append(steps, "        env:\n")
	// Pass the agent output content from the main job
	steps = append(steps, fmt.Sprintf("          GITHUB_AW_AGENT_OUTPUT: ${{ needs.%s.outputs.output }}\n", mainJobName))
	if data.SafeOutputs.CreateDiscussions.TitlePrefix != "" {
		steps = append(steps, fmt.Sprintf("          GITHUB_AW_DISCUSSION_TITLE_PREFIX: %q\n", data.SafeOutputs.CreateDiscussions.TitlePrefix))
	}
	if data.SafeOutputs.CreateDiscussions.CategoryId != "" {
		steps = append(steps, fmt.Sprintf("          GITHUB_AW_DISCUSSION_CATEGORY_ID: %q\n", data.SafeOutputs.CreateDiscussions.CategoryId))
	}

	steps = append(steps, "        with:\n")
	steps = append(steps, "          script: |\n")

	// Add each line of the script with proper indentation
	formattedScript := FormatJavaScriptForYAML(createDiscussionScript)
	steps = append(steps, formattedScript...)

	outputs := map[string]string{
		"discussion_number": "${{ steps.create_discussion.outputs.discussion_number }}",
		"discussion_url":    "${{ steps.create_discussion.outputs.discussion_url }}",
	}

	// Determine the job condition based on command configuration
	var jobCondition string
	if data.Command != "" {
		// Build the command trigger condition
		commandCondition := buildCommandOnlyCondition(data.Command)
		commandConditionStr := commandCondition.Render()
		jobCondition = fmt.Sprintf("if: %s", commandConditionStr)
	} else {
		jobCondition = "" // No conditional execution
	}

	job := &Job{
		Name:           "create_discussion",
		If:             jobCondition,
		RunsOn:         "runs-on: ubuntu-latest",
		Permissions:    "permissions:\n      contents: read\n      discussions: write",
		TimeoutMinutes: 10, // 10-minute timeout as required
		Steps:          steps,
		Outputs:        outputs,
		Depends:        []string{mainJobName}, // Depend on the main workflow job
	}

	return job, nil
}

// buildCreateOutputAddIssueCommentJob creates the create_issue_comment job
func (c *Compiler) buildCreateOutputAddIssueCommentJob(data *WorkflowData, mainJobName string) (*Job, error) {
	if data.SafeOutputs == nil || data.SafeOutputs.AddIssueComments == nil {
		return nil, fmt.Errorf("safe-outputs.add-issue-comment configuration is required")
	}

	var steps []string
	steps = append(steps, "      - name: Add Issue Comment\n")
	steps = append(steps, "        id: create_comment\n")
	steps = append(steps, "        uses: actions/github-script@v7\n")

	// Add environment variables
	steps = append(steps, "        env:\n")
	// Pass the agent output content from the main job
	steps = append(steps, fmt.Sprintf("          GITHUB_AW_AGENT_OUTPUT: ${{ needs.%s.outputs.output }}\n", mainJobName))
	// Pass the comment target configuration
	if data.SafeOutputs.AddIssueComments.Target != "" {
		steps = append(steps, fmt.Sprintf("          GITHUB_AW_COMMENT_TARGET: %q\n", data.SafeOutputs.AddIssueComments.Target))
	}

	steps = append(steps, "        with:\n")
	steps = append(steps, "          script: |\n")

	// Add each line of the script with proper indentation
	formattedScript := FormatJavaScriptForYAML(createCommentScript)
	steps = append(steps, formattedScript...)

	// Create outputs for the job
	outputs := map[string]string{
		"comment_id":  "${{ steps.create_comment.outputs.comment_id }}",
		"comment_url": "${{ steps.create_comment.outputs.comment_url }}",
	}

	// Determine the job condition based on target configuration
	var baseCondition string
	if data.SafeOutputs.AddIssueComments.Target == "*" {
		// Allow the job to run in any context when target is "*"
		baseCondition = "always()" // This allows the job to run even without triggering issue/PR
	} else {
		// Default behavior: only run in issue or PR context
		baseCondition = "github.event.issue.number || github.event.pull_request.number"
	}

	// If this is a command workflow, combine the command trigger condition with the base condition
	var jobCondition string
	if data.Command != "" {
		// Build the command trigger condition
		commandCondition := buildCommandOnlyCondition(data.Command)
		commandConditionStr := commandCondition.Render()

		// Combine command condition with base condition using AND
		if baseCondition == "always()" {
			// If base condition is always(), just use the command condition
			jobCondition = fmt.Sprintf("if: %s", commandConditionStr)
		} else {
			// Combine both conditions with AND
			jobCondition = fmt.Sprintf("if: (%s) && (%s)", commandConditionStr, baseCondition)
		}
	} else {
		// No command trigger, just use the base condition
		jobCondition = fmt.Sprintf("if: %s", baseCondition)
	}

	job := &Job{
		Name:           "create_issue_comment",
		If:             jobCondition,
		RunsOn:         "runs-on: ubuntu-latest",
		Permissions:    "permissions:\n      contents: read\n      issues: write\n      pull-requests: write",
		TimeoutMinutes: 10, // 10-minute timeout as required
		Steps:          steps,
		Outputs:        outputs,
		Depends:        []string{mainJobName}, // Depend on the main workflow job
	}

	return job, nil
}

// buildCreateOutputPullRequestReviewCommentJob creates the create_pr_review_comment job
func (c *Compiler) buildCreateOutputPullRequestReviewCommentJob(data *WorkflowData, mainJobName string) (*Job, error) {
	if data.SafeOutputs == nil || data.SafeOutputs.CreatePullRequestReviewComments == nil {
		return nil, fmt.Errorf("safe-outputs.create-pull-request-review-comment configuration is required")
	}

	var steps []string
	steps = append(steps, "      - name: Create PR Review Comment\n")
	steps = append(steps, "        id: create_pr_review_comment\n")
	steps = append(steps, "        uses: actions/github-script@v7\n")

	// Add environment variables
	steps = append(steps, "        env:\n")
	// Pass the agent output content from the main job
	steps = append(steps, fmt.Sprintf("          GITHUB_AW_AGENT_OUTPUT: ${{ needs.%s.outputs.output }}\n", mainJobName))
	// Pass the side configuration
	if data.SafeOutputs.CreatePullRequestReviewComments.Side != "" {
		steps = append(steps, fmt.Sprintf("          GITHUB_AW_PR_REVIEW_COMMENT_SIDE: %q\n", data.SafeOutputs.CreatePullRequestReviewComments.Side))
	}

	steps = append(steps, "        with:\n")
	steps = append(steps, "          script: |\n")

	// Add each line of the script with proper indentation
	formattedScript := FormatJavaScriptForYAML(createPRReviewCommentScript)
	steps = append(steps, formattedScript...)

	// Create outputs for the job
	outputs := map[string]string{
		"review_comment_id":  "${{ steps.create_pr_review_comment.outputs.review_comment_id }}",
		"review_comment_url": "${{ steps.create_pr_review_comment.outputs.review_comment_url }}",
	}

	// Only run in pull request context
	baseCondition := "github.event.pull_request.number"

	// If this is a command workflow, combine the command trigger condition with the base condition
	var jobCondition string
	if data.Command != "" {
		// Build the command trigger condition
		commandCondition := buildCommandOnlyCondition(data.Command)
		commandConditionStr := commandCondition.Render()

		// Combine command condition with base condition using AND
		jobCondition = fmt.Sprintf("if: (%s) && (%s)", commandConditionStr, baseCondition)
	} else {
		// No command trigger, just use the base condition
		jobCondition = fmt.Sprintf("if: %s", baseCondition)
	}

	job := &Job{
		Name:           "create_pr_review_comment",
		If:             jobCondition,
		RunsOn:         "runs-on: ubuntu-latest",
		Permissions:    "permissions:\n      contents: read\n      pull-requests: write",
		TimeoutMinutes: 10, // 10-minute timeout as required
		Steps:          steps,
		Outputs:        outputs,
		Depends:        []string{mainJobName}, // Depend on the main workflow job
	}

	return job, nil
}

// buildCreateOutputSecurityReportJob creates the create_security_report job
func (c *Compiler) buildCreateOutputSecurityReportJob(data *WorkflowData, mainJobName string, workflowFilename string) (*Job, error) {
	if data.SafeOutputs == nil || data.SafeOutputs.CreateSecurityReports == nil {
		return nil, fmt.Errorf("safe-outputs.create-security-report configuration is required")
	}

	var steps []string
	steps = append(steps, "      - name: Create Security Report\n")
	steps = append(steps, "        id: create_security_report\n")
	steps = append(steps, "        uses: actions/github-script@v7\n")

	// Add environment variables
	steps = append(steps, "        env:\n")
	// Pass the agent output content from the main job
	steps = append(steps, fmt.Sprintf("          GITHUB_AW_AGENT_OUTPUT: ${{ needs.%s.outputs.output }}\n", mainJobName))
	// Pass the max configuration
	if data.SafeOutputs.CreateSecurityReports.Max > 0 {
		steps = append(steps, fmt.Sprintf("          GITHUB_AW_SECURITY_REPORT_MAX: %d\n", data.SafeOutputs.CreateSecurityReports.Max))
	}
	// Pass the driver configuration, defaulting to frontmatter name
	driverName := data.SafeOutputs.CreateSecurityReports.Driver
	if driverName == "" {
		if data.FrontmatterName != "" {
			driverName = data.FrontmatterName
		} else {
			driverName = data.Name // fallback to H1 header name
		}
	}
	steps = append(steps, fmt.Sprintf("          GITHUB_AW_SECURITY_REPORT_DRIVER: %s\n", driverName))
	// Pass the workflow filename for rule ID prefix
	steps = append(steps, fmt.Sprintf("          GITHUB_AW_WORKFLOW_FILENAME: %s\n", workflowFilename))

	steps = append(steps, "        with:\n")
	steps = append(steps, "          script: |\n")

	// Add each line of the script with proper indentation
	formattedScript := FormatJavaScriptForYAML(createSecurityReportScript)
	steps = append(steps, formattedScript...)

	// Add step to upload SARIF artifact
	steps = append(steps, "      - name: Upload SARIF artifact\n")
	steps = append(steps, "        if: steps.create_security_report.outputs.sarif_file\n")
	steps = append(steps, "        uses: actions/upload-artifact@v4\n")
	steps = append(steps, "        with:\n")
	steps = append(steps, "          name: security-report.sarif\n")
	steps = append(steps, "          path: ${{ steps.create_security_report.outputs.sarif_file }}\n")

	// Add step to upload SARIF to GitHub Code Scanning
	steps = append(steps, "      - name: Upload SARIF to GitHub Security\n")
	steps = append(steps, "        if: steps.create_security_report.outputs.sarif_file\n")
	steps = append(steps, "        uses: github/codeql-action/upload-sarif@v3\n")
	steps = append(steps, "        with:\n")
	steps = append(steps, "          sarif_file: ${{ steps.create_security_report.outputs.sarif_file }}\n")

	// Create outputs for the job
	outputs := map[string]string{
		"sarif_file":        "${{ steps.create_security_report.outputs.sarif_file }}",
		"findings_count":    "${{ steps.create_security_report.outputs.findings_count }}",
		"artifact_uploaded": "${{ steps.create_security_report.outputs.artifact_uploaded }}",
		"codeql_uploaded":   "${{ steps.create_security_report.outputs.codeql_uploaded }}",
	}

	// Build job condition - security reports can run in any context unlike PR review comments
	var jobCondition string
	if data.Command != "" {
		// Build the command trigger condition
		commandCondition := buildCommandOnlyCondition(data.Command)
		commandConditionStr := commandCondition.Render()
		jobCondition = fmt.Sprintf("if: %s", commandConditionStr)
	} else {
		// No specific condition needed - security reports can run anytime
		jobCondition = ""
	}

	job := &Job{
		Name:           "create_security_report",
		If:             jobCondition,
		RunsOn:         "runs-on: ubuntu-latest",
		Permissions:    "permissions:\n      contents: read\n      security-events: write\n      actions: read", // Need security-events:write for SARIF upload
		TimeoutMinutes: 10,                                                                                      // 10-minute timeout
		Steps:          steps,
		Outputs:        outputs,
		Depends:        []string{mainJobName}, // Depend on the main workflow job
	}

	return job, nil
}

// buildCreateOutputPullRequestJob creates the create_pull_request job
func (c *Compiler) buildCreateOutputPullRequestJob(data *WorkflowData, mainJobName string) (*Job, error) {
	if data.SafeOutputs == nil || data.SafeOutputs.CreatePullRequests == nil {
		return nil, fmt.Errorf("safe-outputs.create-pull-request configuration is required")
	}

	var steps []string

	// Step 1: Download patch artifact
	steps = append(steps, "      - name: Download patch artifact\n")
	steps = append(steps, "        uses: actions/download-artifact@v4\n")
	steps = append(steps, "        with:\n")
	steps = append(steps, "          name: aw.patch\n")
	steps = append(steps, "          path: /tmp/\n")

	// Step 2: Checkout repository
	steps = append(steps, "      - name: Checkout repository\n")
	steps = append(steps, "        uses: actions/checkout@v5\n")
	steps = append(steps, "        with:\n")
	steps = append(steps, "          fetch-depth: 0\n")

	// Step 3: Create pull request
	steps = append(steps, "      - name: Create Pull Request\n")
	steps = append(steps, "        id: create_pull_request\n")
	steps = append(steps, "        uses: actions/github-script@v7\n")

	// Add environment variables
	steps = append(steps, "        env:\n")
	// Pass the agent output content from the main job
	steps = append(steps, fmt.Sprintf("          GITHUB_AW_AGENT_OUTPUT: ${{ needs.%s.outputs.output }}\n", mainJobName))
	// Pass the workflow ID for branch naming
	steps = append(steps, fmt.Sprintf("          GITHUB_AW_WORKFLOW_ID: %q\n", mainJobName))
	// Pass the base branch from GitHub context
	steps = append(steps, "          GITHUB_AW_BASE_BRANCH: ${{ github.ref_name }}\n")
	if data.SafeOutputs.CreatePullRequests.TitlePrefix != "" {
		steps = append(steps, fmt.Sprintf("          GITHUB_AW_PR_TITLE_PREFIX: %q\n", data.SafeOutputs.CreatePullRequests.TitlePrefix))
	}
	if len(data.SafeOutputs.CreatePullRequests.Labels) > 0 {
		labelsStr := strings.Join(data.SafeOutputs.CreatePullRequests.Labels, ",")
		steps = append(steps, fmt.Sprintf("          GITHUB_AW_PR_LABELS: %q\n", labelsStr))
	}
	// Pass draft setting - default to true for backwards compatibility
	draftValue := true // Default value
	if data.SafeOutputs.CreatePullRequests.Draft != nil {
		draftValue = *data.SafeOutputs.CreatePullRequests.Draft
	}
	steps = append(steps, fmt.Sprintf("          GITHUB_AW_PR_DRAFT: %q\n", fmt.Sprintf("%t", draftValue)))

	steps = append(steps, "        with:\n")
	steps = append(steps, "          script: |\n")

	// Add each line of the script with proper indentation
	formattedScript := FormatJavaScriptForYAML(createPullRequestScript)
	steps = append(steps, formattedScript...)

	// Create outputs for the job
	outputs := map[string]string{
		"pull_request_number": "${{ steps.create_pull_request.outputs.pull_request_number }}",
		"pull_request_url":    "${{ steps.create_pull_request.outputs.pull_request_url }}",
		"branch_name":         "${{ steps.create_pull_request.outputs.branch_name }}",
	}

	// Determine the job condition for command workflows
	var jobCondition string
	if data.Command != "" {
		// Build the command trigger condition
		commandCondition := buildCommandOnlyCondition(data.Command)
		commandConditionStr := commandCondition.Render()
		jobCondition = fmt.Sprintf("if: %s", commandConditionStr)
	} else {
		jobCondition = "" // No conditional execution
	}

	job := &Job{
		Name:           "create_pull_request",
		If:             jobCondition,
		RunsOn:         "runs-on: ubuntu-latest",
		Permissions:    "permissions:\n      contents: write\n      issues: write\n      pull-requests: write",
		TimeoutMinutes: 10, // 10-minute timeout as required
		Steps:          steps,
		Outputs:        outputs,
		Depends:        []string{mainJobName}, // Depend on the main workflow job
	}

	return job, nil
}

// buildMainJob creates the main workflow job
func (c *Compiler) buildMainJob(data *WorkflowData, jobName string, taskJobCreated bool) (*Job, error) {
	var steps []string

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

	job := &Job{
		Name:        jobName,
		If:          "", // Remove the If condition since task job handles alias checks
		RunsOn:      c.indentYAMLLines(data.RunsOn, "    "),
		Permissions: c.indentYAMLLines(data.Permissions, "    "),
		Steps:       steps,
		Depends:     depends,
		Outputs:     outputs,
	}

	return job, nil
}

// generateSafetyChecks generates safety checks for stop-time before executing agentic tools
func (c *Compiler) generateSafetyChecks(yaml *strings.Builder, data *WorkflowData) {
	// If no safety settings, skip generating safety checks
	if data.StopTime == "" {
		return
	}

	yaml.WriteString("      - name: Safety checks\n")
	yaml.WriteString("        run: |\n")
	yaml.WriteString("          set -e\n")
	yaml.WriteString("          echo \"Performing safety checks before executing agentic tools...\"\n")

	// Extract workflow name for gh workflow commands
	workflowName := data.Name
	fmt.Fprintf(yaml, "          WORKFLOW_NAME=\"%s\"\n", workflowName)

	// Add stop-time check
	if data.StopTime != "" {
		yaml.WriteString("          \n")
		yaml.WriteString("          # Check stop-time limit\n")
		fmt.Fprintf(yaml, "          STOP_TIME=\"%s\"\n", data.StopTime)
		yaml.WriteString("          echo \"Checking stop-time limit: $STOP_TIME\"\n")
		yaml.WriteString("          \n")
		yaml.WriteString("          # Convert stop time to epoch seconds\n")
		yaml.WriteString("          STOP_EPOCH=$(date -d \"$STOP_TIME\" +%s 2>/dev/null || echo \"invalid\")\n")
		yaml.WriteString("          if [ \"$STOP_EPOCH\" = \"invalid\" ]; then\n")
		yaml.WriteString("            echo \"Warning: Invalid stop-time format: $STOP_TIME. Expected format: YYYY-MM-DD HH:MM:SS\"\n")
		yaml.WriteString("          else\n")
		yaml.WriteString("            CURRENT_EPOCH=$(date +%s)\n")
		yaml.WriteString("            echo \"Current time: $(date)\"\n")
		yaml.WriteString("            echo \"Stop time: $STOP_TIME\"\n")
		yaml.WriteString("            \n")
		yaml.WriteString("            if [ \"$CURRENT_EPOCH\" -ge \"$STOP_EPOCH\" ]; then\n")
		yaml.WriteString("              echo \"Stop time reached. Attempting to disable workflow to prevent cost overrun, then exiting.\"\n")
		yaml.WriteString("              gh workflow disable \"$WORKFLOW_NAME\"\n")
		yaml.WriteString("              echo \"Workflow disabled. No future runs will be triggered.\"\n")
		yaml.WriteString("              exit 1\n")
		yaml.WriteString("            fi\n")
		yaml.WriteString("          fi\n")
	}

	yaml.WriteString("          echo \"All safety checks passed. Proceeding with agentic tool execution.\"\n")
	yaml.WriteString("        env:\n")
	yaml.WriteString("          GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}\n")
}

// generateMCPSetup generates the MCP server configuration setup
func (c *Compiler) generateMCPSetup(yaml *strings.Builder, tools map[string]any, engine AgenticEngine) {
	// Collect tools that need MCP server configuration
	var mcpTools []string
	var proxyTools []string

	for toolName, toolValue := range tools {
		// Standard MCP tools
		if toolName == "github" {
			mcpTools = append(mcpTools, toolName)
		} else if mcpConfig, ok := toolValue.(map[string]any); ok {
			// Check if it's explicitly marked as MCP type in the new format
			if hasMcp, _ := hasMCPConfig(mcpConfig); hasMcp {
				mcpTools = append(mcpTools, toolName)

				// Check if this tool needs proxy
				if needsProxySetup, _ := needsProxy(mcpConfig); needsProxySetup {
					proxyTools = append(proxyTools, toolName)
				}
			}
		}
	}

	// Sort tools to ensure stable code generation
	sort.Strings(mcpTools)
	sort.Strings(proxyTools)

	// Generate proxy configuration files inline for proxy-enabled tools
	// These files will be used automatically by docker compose when MCP tools run
	if len(proxyTools) > 0 {
		yaml.WriteString("      - name: Setup Proxy Configuration for MCP Network Restrictions\n")
		yaml.WriteString("        run: |\n")
		yaml.WriteString("          echo \"Generating proxy configuration files for MCP tools with network restrictions...\"\n")
		yaml.WriteString("          \n")

		// Generate proxy configurations inline for each proxy-enabled tool
		for _, toolName := range proxyTools {
			if toolConfig, ok := tools[toolName].(map[string]any); ok {
				c.generateInlineProxyConfig(yaml, toolName, toolConfig)
			}
		}

		yaml.WriteString("          echo \"Proxy configuration files generated.\"\n")

		// Pre-pull images and start squid proxy ahead of time to avoid timeouts
		yaml.WriteString("      - name: Pre-pull images and start Squid proxy\n")
		yaml.WriteString("        run: |\n")
		yaml.WriteString("          set -e\n")
		yaml.WriteString("          echo 'Pre-pulling Docker images for proxy-enabled MCP tools...'\n")
		yaml.WriteString("          docker pull ubuntu/squid:latest\n")

		// Pull each tool's container image if specified, and bring up squid service
		for _, toolName := range proxyTools {
			if toolConfig, ok := tools[toolName].(map[string]any); ok {
				if mcpConf, err := getMCPConfig(toolConfig, toolName); err == nil {
					if containerVal, hasContainer := mcpConf["container"]; hasContainer {
						if containerStr, ok := containerVal.(string); ok && containerStr != "" {
							fmt.Fprintf(yaml, "          echo 'Pulling %s for tool %s'\n", containerStr, toolName)
							fmt.Fprintf(yaml, "          docker pull %s\n", containerStr)
						}
					}
				}
				fmt.Fprintf(yaml, "          echo 'Starting squid-proxy service for %s'\n", toolName)
				fmt.Fprintf(yaml, "          docker compose -f docker-compose-%s.yml up -d squid-proxy\n", toolName)

				// Enforce that egress from this tool's network can only reach the Squid proxy
				subnetCIDR, squidIP, _ := computeProxyNetworkParams(toolName)
				fmt.Fprintf(yaml, "          echo 'Enforcing egress to proxy for %s (subnet %s, squid %s)'\n", toolName, subnetCIDR, squidIP)
				yaml.WriteString("          if command -v sudo >/dev/null 2>&1; then SUDO=sudo; else SUDO=; fi\n")
				// Accept established/related connections first (position 1)
				yaml.WriteString("          $SUDO iptables -C DOCKER-USER -m conntrack --ctstate ESTABLISHED,RELATED -j ACCEPT 2>/dev/null || $SUDO iptables -I DOCKER-USER 1 -m conntrack --ctstate ESTABLISHED,RELATED -j ACCEPT\n")
				// Accept all egress from Squid IP (position 2)
				fmt.Fprintf(yaml, "          $SUDO iptables -C DOCKER-USER -s %s -j ACCEPT 2>/dev/null || $SUDO iptables -I DOCKER-USER 2 -s %s -j ACCEPT\n", squidIP, squidIP)
				// Allow traffic to squid:3128 from the subnet (position 3)
				fmt.Fprintf(yaml, "          $SUDO iptables -C DOCKER-USER -s %s -d %s -p tcp --dport 3128 -j ACCEPT 2>/dev/null || $SUDO iptables -I DOCKER-USER 3 -s %s -d %s -p tcp --dport 3128 -j ACCEPT\n", subnetCIDR, squidIP, subnetCIDR, squidIP)
				// Then reject all other egress from that subnet (append to end)
				fmt.Fprintf(yaml, "          $SUDO iptables -C DOCKER-USER -s %s -j REJECT 2>/dev/null || $SUDO iptables -A DOCKER-USER -s %s -j REJECT\n", subnetCIDR, subnetCIDR)
			}
		}
	}

	// If no MCP tools, no configuration needed
	if len(mcpTools) == 0 {
		return
	}

	yaml.WriteString("      - name: Setup MCPs\n")
	yaml.WriteString("        run: |\n")
	yaml.WriteString("          mkdir -p /tmp/mcp-config\n")

	// Use the engine's RenderMCPConfig method
	engine.RenderMCPConfig(yaml, tools, mcpTools)
}

func getGitHubDockerImageVersion(githubTool any) string {
	githubDockerImageVersion := "sha-09deac4" // Default Docker image version
	// Extract docker_image_version setting from tool properties
	if toolConfig, ok := githubTool.(map[string]any); ok {
		if versionSetting, exists := toolConfig["docker_image_version"]; exists {
			if stringValue, ok := versionSetting.(string); ok {
				githubDockerImageVersion = stringValue
			}
		}
	}
	return githubDockerImageVersion
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

	// Add Node.js setup if the engine requires it
	engine, err := c.getAgenticEngine(data.AI)

	if err != nil {
		return
	}

	// Add engine-specific installation steps
	installSteps := engine.GetInstallationSteps(data.EngineConfig, data.NetworkPermissions)
	for _, step := range installSteps {
		for _, line := range step {
			yaml.WriteString(line + "\n")
		}
	}

	// Generate output file setup step only if safe-outputs feature is used (GITHUB_AW_SAFE_OUTPUTS functionality)
	if data.SafeOutputs != nil {
		c.generateOutputFileSetup(yaml, data)
	}

	// Add MCP setup
	c.generateMCPSetup(yaml, data.Tools, engine)

	// Add safety checks before executing agentic tools
	c.generateSafetyChecks(yaml, data)

	// Add prompt creation step
	c.generatePrompt(yaml, data, engine)

	logFile := generateSafeFileName(data.Name)
	logFileFull := fmt.Sprintf("/tmp/%s.log", logFile)

	// Generate aw_info.json with agentic run metadata
	c.generateCreateAwInfo(yaml, data, engine)

	// Upload info to artifact
	c.generateUploadAwInfo(yaml)

	// Add AI execution step using the agentic engine
	c.generateEngineExecutionSteps(yaml, data, engine, logFileFull)

	// add workflow_complete.txt
	c.generateWorkflowComplete(yaml)

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

	// parse agent logs for GITHUB_STEP_SUMMARY
	c.generateLogParsing(yaml, engine, logFileFull)

	// upload agent logs
	c.generateUploadAgentLogs(yaml, logFile, logFileFull)

	// Add git patch generation step only if safe-outputs create-pull-request feature is used
	if data.SafeOutputs != nil && (data.SafeOutputs.CreatePullRequests != nil || data.SafeOutputs.PushToBranch != nil) {
		c.generateGitPatchStep(yaml, data)
	}

	// Add post-steps (if any) after AI execution
	c.generatePostSteps(yaml, data)
}

func (c *Compiler) generateWorkflowComplete(yaml *strings.Builder) {
	yaml.WriteString("      - name: Check if workflow-complete.txt exists, if so upload it\n")
	yaml.WriteString("        id: check_file\n")
	yaml.WriteString("        run: |\n")
	yaml.WriteString("          if [ -f workflow-complete.txt ]; then\n")
	yaml.WriteString("            echo \"File exists\"\n")
	yaml.WriteString("            echo \"upload=true\" >> $GITHUB_OUTPUT\n")
	yaml.WriteString("          else\n")
	yaml.WriteString("            echo \"File does not exist\"\n")
	yaml.WriteString("            echo \"upload=false\" >> $GITHUB_OUTPUT\n")
	yaml.WriteString("          fi\n")
	yaml.WriteString("      - name: Upload workflow-complete.txt\n")
	yaml.WriteString("        if: steps.check_file.outputs.upload == 'true'\n")
	yaml.WriteString("        uses: actions/upload-artifact@v4\n")
	yaml.WriteString("        with:\n")
	yaml.WriteString("          name: workflow-complete\n")
	yaml.WriteString("          path: workflow-complete.txt\n")
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

func (c *Compiler) generateLogParsing(yaml *strings.Builder, engine AgenticEngine, logFileFull string) {
	parserScriptName := engine.GetLogParserScript()
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
	yaml.WriteString("        uses: actions/github-script@v7\n")
	yaml.WriteString("        env:\n")
	fmt.Fprintf(yaml, "          AGENT_LOG_FILE: %s\n", logFileFull)
	yaml.WriteString("        with:\n")
	yaml.WriteString("          script: |\n")

	// Inline the JavaScript code with proper indentation
	steps := FormatJavaScriptForYAML(logParserScript)
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

func (c *Compiler) generatePrompt(yaml *strings.Builder, data *WorkflowData, engine AgenticEngine) {
	yaml.WriteString("      - name: Create prompt\n")

	// Only add GITHUB_AW_SAFE_OUTPUTS environment variable if safe-outputs feature is used
	if data.SafeOutputs != nil {
		yaml.WriteString("        env:\n")
		yaml.WriteString("          GITHUB_AW_SAFE_OUTPUTS: ${{ env.GITHUB_AW_SAFE_OUTPUTS }}\n")
	}

	yaml.WriteString("        run: |\n")
	yaml.WriteString("          mkdir -p /tmp/aw-prompts\n")
	yaml.WriteString("          cat > /tmp/aw-prompts/prompt.txt << 'EOF'\n")

	// Add markdown content with proper indentation
	for _, line := range strings.Split(data.MarkdownContent, "\n") {
		yaml.WriteString("          " + line + "\n")
	}

	if data.SafeOutputs != nil {
		// Add output instructions for all engines (GITHUB_AW_SAFE_OUTPUTS functionality)
		yaml.WriteString("          \n")
		yaml.WriteString("          ---\n")
		yaml.WriteString("          \n")
		yaml.WriteString("          ## ")
		written := false
		if data.SafeOutputs.AddIssueComments != nil {
			yaml.WriteString("Adding a Comment to an Issue or Pull Request")
			written = true
		}
		if data.SafeOutputs.CreateIssues != nil {
			if written {
				yaml.WriteString(", ")
			}
			yaml.WriteString("Creating an Issue")
		}
		if data.SafeOutputs.CreatePullRequests != nil {
			if written {
				yaml.WriteString(", ")
			}
			yaml.WriteString("Creating a Pull Request")
		}

		if data.SafeOutputs.AddIssueLabels != nil {
			if written {
				yaml.WriteString(", ")
			}
			yaml.WriteString("Adding Labels to Issues or Pull Requests")
			written = true
		}

		if data.SafeOutputs.UpdateIssues != nil {
			if written {
				yaml.WriteString(", ")
			}
			yaml.WriteString("Updating Issues")
			written = true
		}

		if data.SafeOutputs.PushToBranch != nil {
			if written {
				yaml.WriteString(", ")
			}
			yaml.WriteString("Pushing Changes to Branch")
			written = true
		}

		// Missing-tool is always available
		if written {
			yaml.WriteString(", ")
		}
		yaml.WriteString("Reporting Missing Tools or Functionality")

		yaml.WriteString("\n")
		yaml.WriteString("          \n")
		yaml.WriteString("          **IMPORTANT**: To do the actions mentioned in the header of this section, do NOT attempt to use MCP tools, do NOT attempt to use `gh`, do NOT attempt to use the GitHub API. You don't have write access to the GitHub repo. Instead write JSON objects to the file \"${{ env.GITHUB_AW_SAFE_OUTPUTS }}\". Each line should contain a single JSON object (JSONL format). You can write them one by one as you do them.\n")
		yaml.WriteString("          \n")
		yaml.WriteString("          **Format**: Write one JSON object per line. Each object must have a `type` field specifying the action type.\n")
		yaml.WriteString("          \n")
		yaml.WriteString("          ### Available Output Types:\n")
		yaml.WriteString("          \n")

		if data.SafeOutputs.AddIssueComments != nil {
			yaml.WriteString("          **Adding a Comment to an Issue or Pull Request**\n")
			yaml.WriteString("          \n")
			yaml.WriteString("          To add a comment to an issue or pull request:\n")
			yaml.WriteString("          1. Write an entry to \"${{ env.GITHUB_AW_SAFE_OUTPUTS }}\":\n")
			yaml.WriteString("          ```json\n")
			yaml.WriteString("          {\"type\": \"add-issue-comment\", \"body\": \"Your comment content in markdown\"}\n")
			yaml.WriteString("          ```\n")
			yaml.WriteString("          2. After you write to that file, read it as JSONL and check it is valid. If it isn't, make any necessary corrections to it to fix it up\n")
			yaml.WriteString("          \n")
		}

		if data.SafeOutputs.CreateIssues != nil {
			yaml.WriteString("          **Creating an Issue**\n")
			yaml.WriteString("          \n")
			yaml.WriteString("          To create an issue:\n")
			yaml.WriteString("          1. Write an entry to \"${{ env.GITHUB_AW_SAFE_OUTPUTS }}\":\n")
			yaml.WriteString("          ```json\n")
			yaml.WriteString("          {\"type\": \"create-issue\", \"title\": \"Issue title\", \"body\": \"Issue body in markdown\", \"labels\": [\"optional\", \"labels\"]}\n")
			yaml.WriteString("          ```\n")
			yaml.WriteString("          2. After you write to that file, read it as JSONL and check it is valid. If it isn't, make any necessary corrections to it to fix it up\n")
			yaml.WriteString("          \n")
		}

		if data.SafeOutputs.CreatePullRequests != nil {
			yaml.WriteString("          **Creating a Pull Request**\n")
			yaml.WriteString("          \n")
			yaml.WriteString("          To create a pull request:\n")
			yaml.WriteString("          1. Make any file changes directly in the working directory\n")
			yaml.WriteString("          2. If you haven't done so already, create a local branch using an appropriate unique name\n")
			yaml.WriteString("          3. Add and commit your changes to the branch. Be careful to add exactly the files you intend, and check there are no extra files left un-added. Check you haven't deleted or changed any files you didn't intend to.\n")
			yaml.WriteString("          4. Do not push your changes. That will be done later. Instead append the PR specification to the file \"${{ env.GITHUB_AW_SAFE_OUTPUTS }}\":\n")
			yaml.WriteString("          ```json\n")
			yaml.WriteString("          {\"type\": \"create-pull-request\", \"branch\": \"branch-name\", \"title\": \"PR title\", \"body\": \"PR body in markdown\", \"labels\": [\"optional\", \"labels\"]}\n")
			yaml.WriteString("          ```\n")
			yaml.WriteString("          5. After you write to that file, read it as JSONL and check it is valid. If it isn't, make any necessary corrections to it to fix it up\n")
			yaml.WriteString("          \n")
		}

		if data.SafeOutputs.AddIssueLabels != nil {
			yaml.WriteString("          **Adding Labels to Issues or Pull Requests**\n")
			yaml.WriteString("          \n")
			yaml.WriteString("          To add labels to a pull request:\n")
			yaml.WriteString("          1. Write an entry to \"${{ env.GITHUB_AW_SAFE_OUTPUTS }}\":\n")
			yaml.WriteString("          ```json\n")
			yaml.WriteString("          {\"type\": \"add-issue-label\", \"labels\": [\"label1\", \"label2\", \"label3\"]}\n")
			yaml.WriteString("          ```\n")
			yaml.WriteString("          2. After you write to that file, read it as JSONL and check it is valid. If it isn't, make any necessary corrections to it to fix it up\n")
			yaml.WriteString("          \n")
		}

		if data.SafeOutputs.UpdateIssues != nil {
			yaml.WriteString("          **Updating an Issue**\n")
			yaml.WriteString("          \n")
			yaml.WriteString("          To udpate an issue:\n")
			yaml.WriteString("          ```json\n")

			// Build example based on allowed fields
			var fields []string
			if data.SafeOutputs.UpdateIssues.Status != nil {
				fields = append(fields, "\"status\": \"open\" // or \"closed\"")
			}
			if data.SafeOutputs.UpdateIssues.Title != nil {
				fields = append(fields, "\"title\": \"New issue title\"")
			}
			if data.SafeOutputs.UpdateIssues.Body != nil {
				fields = append(fields, "\"body\": \"Updated issue body in markdown\"")
			}

			if len(fields) > 0 {
				yaml.WriteString("          {\"type\": \"update-issue\"")
				for _, field := range fields {
					yaml.WriteString(", " + field)
				}
				yaml.WriteString("}\n")
			} else {
				yaml.WriteString("          {\"type\": \"update-issue\", \"title\": \"New issue title\", \"body\": \"Updated issue body\", \"status\": \"open\"}\n")
			}

			yaml.WriteString("          ```\n")
			yaml.WriteString("          2. After you write to that file, read it as JSONL and check it is valid. If it isn't, make any necessary corrections to it to fix it up\n")
			yaml.WriteString("          \n")
		}

		if data.SafeOutputs.PushToBranch != nil {
			yaml.WriteString("          **Pushing Changes to Branch**\n")
			yaml.WriteString("          \n")
			yaml.WriteString("          To push changes to a branch, for example to add code to a pull request:\n")
			yaml.WriteString("          1. Make any file changes directly in the working directory\n")
			yaml.WriteString("          2. Add and commit your changes to the branch. Be careful to add exactly the files you intend, and check there are no extra files left un-added. Check you haven't deleted or changed any files you didn't intend to.\n")
			yaml.WriteString("          3. Indicate your intention to push to the branch by writing to the file \"${{ env.GITHUB_AW_SAFE_OUTPUTS }}\":\n")
			yaml.WriteString("          ```json\n")
			yaml.WriteString("          {\"type\": \"push-to-branch\", \"message\": \"Commit message describing the changes\"}\n")
			yaml.WriteString("          ```\n")
			yaml.WriteString("          4. After you write to that file, read it as JSONL and check it is valid. If it isn't, make any necessary corrections to it to fix it up\n")
			yaml.WriteString("          \n")
		}

		// Missing-tool instructions are only included when configured
		if data.SafeOutputs.MissingTool != nil {
			yaml.WriteString("          **Reporting Missing Tools or Functionality**\n")
			yaml.WriteString("          \n")
			yaml.WriteString("          If you need to use a tool or functionality that is not available to complete your task:\n")
			yaml.WriteString("          1. Write an entry to \"${{ env.GITHUB_AW_SAFE_OUTPUTS }}\":\n")
			yaml.WriteString("          ```json\n")
			yaml.WriteString("          {\"type\": \"missing-tool\", \"tool\": \"tool-name\", \"reason\": \"Why this tool is needed\", \"alternatives\": \"Suggested alternatives or workarounds\"}\n")
			yaml.WriteString("          ```\n")
			yaml.WriteString("          2. The `tool` field should specify the name or type of missing functionality\n")
			yaml.WriteString("          3. The `reason` field should explain why this tool/functionality is required to complete the task\n")
			yaml.WriteString("          4. The `alternatives` field is optional but can suggest workarounds or alternative approaches\n")
			yaml.WriteString("          5. After you write to that file, read it as JSONL and check it is valid. If it isn't, make any necessary corrections to it to fix it up\n")
			yaml.WriteString("          \n")
		}

		yaml.WriteString("          **Example JSONL file content:**\n")
		yaml.WriteString("          ```\n")

		// Generate conditional examples based on enabled SafeOutputs
		exampleCount := 0
		if data.SafeOutputs.CreateIssues != nil {
			yaml.WriteString("          {\"type\": \"create-issue\", \"title\": \"Bug Report\", \"body\": \"Found an issue with...\"}\n")
			exampleCount++
		}
		if data.SafeOutputs.AddIssueComments != nil {
			yaml.WriteString("          {\"type\": \"add-issue-comment\", \"body\": \"This is related to the issue above.\"}\n")
			exampleCount++
		}
		if data.SafeOutputs.CreatePullRequests != nil {
			yaml.WriteString("          {\"type\": \"create-pull-request\", \"title\": \"Fix typo\", \"body\": \"Corrected spelling mistake in documentation\"}\n")
			exampleCount++
		}
		if data.SafeOutputs.AddIssueLabels != nil {
			yaml.WriteString("          {\"type\": \"add-issue-label\", \"labels\": [\"bug\", \"priority-high\"]}\n")
			exampleCount++
		}
		if data.SafeOutputs.PushToBranch != nil {
			yaml.WriteString("          {\"type\": \"push-to-branch\", \"message\": \"Update documentation with latest changes\"}\n")
			exampleCount++
		}

		// Include missing-tool example only when configured
		if data.SafeOutputs.MissingTool != nil {
			yaml.WriteString("          {\"type\": \"missing-tool\", \"tool\": \"docker\", \"reason\": \"Need Docker to build container images\", \"alternatives\": \"Could use GitHub Actions build instead\"}\n")
			exampleCount++
		}

		// If no SafeOutputs are enabled, show a generic example
		if exampleCount == 0 {
			yaml.WriteString("          # No safe outputs configured for this workflow\n")
		}

		yaml.WriteString("          ```\n")
		yaml.WriteString("          \n")
		yaml.WriteString("          **Important Notes:**\n")
		yaml.WriteString("          - Do NOT attempt to use MCP tools, `gh`, or the GitHub API for these actions\n")
		yaml.WriteString("          - Each JSON object must be on its own line\n")
		yaml.WriteString("          - Only include output types that are configured for this workflow\n")
		yaml.WriteString("          - The content of this file will be automatically processed and executed\n")
		yaml.WriteString("          \n")
	}

	yaml.WriteString("          EOF\n")

	// Add step to print prompt to GitHub step summary for debugging
	yaml.WriteString("      - name: Print prompt to step summary\n")
	yaml.WriteString("        run: |\n")
	yaml.WriteString("          echo \"## Generated Prompt\" >> $GITHUB_STEP_SUMMARY\n")
	yaml.WriteString("          echo \"\" >> $GITHUB_STEP_SUMMARY\n")
	yaml.WriteString("          echo '``````markdown' >> $GITHUB_STEP_SUMMARY\n")
	yaml.WriteString("          cat /tmp/aw-prompts/prompt.txt >> $GITHUB_STEP_SUMMARY\n")
	yaml.WriteString("          echo '``````' >> $GITHUB_STEP_SUMMARY\n")
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

// extractSafeOutputsConfig extracts output configuration from frontmatter
func (c *Compiler) extractSafeOutputsConfig(frontmatter map[string]any) *SafeOutputsConfig {
	var config *SafeOutputsConfig

	if output, exists := frontmatter["safe-outputs"]; exists {
		if outputMap, ok := output.(map[string]any); ok {
			config = &SafeOutputsConfig{}

			// Handle create-issue
			issuesConfig := c.parseIssuesConfig(outputMap)
			if issuesConfig != nil {
				config.CreateIssues = issuesConfig
			}

			// Handle create-discussion
			discussionsConfig := c.parseDiscussionsConfig(outputMap)
			if discussionsConfig != nil {
				config.CreateDiscussions = discussionsConfig
			}

			// Handle add-issue-comment
			commentsConfig := c.parseCommentsConfig(outputMap)
			if commentsConfig != nil {
				config.AddIssueComments = commentsConfig
			}

			// Handle create-pull-request
			pullRequestsConfig := c.parsePullRequestsConfig(outputMap)
			if pullRequestsConfig != nil {
				config.CreatePullRequests = pullRequestsConfig
			}

			// Handle create-pull-request-review-comment
			prReviewCommentsConfig := c.parsePullRequestReviewCommentsConfig(outputMap)
			if prReviewCommentsConfig != nil {
				config.CreatePullRequestReviewComments = prReviewCommentsConfig
			}

			// Handle create-security-report
			securityReportsConfig := c.parseSecurityReportsConfig(outputMap)
			if securityReportsConfig != nil {
				config.CreateSecurityReports = securityReportsConfig
			}

			// Parse allowed-domains configuration
			if allowedDomains, exists := outputMap["allowed-domains"]; exists {
				if domainsArray, ok := allowedDomains.([]any); ok {
					var domainStrings []string
					for _, domain := range domainsArray {
						if domainStr, ok := domain.(string); ok {
							domainStrings = append(domainStrings, domainStr)
						}
					}
					config.AllowedDomains = domainStrings
				}
			}

			// Parse add-issue-label configuration
			if labels, exists := outputMap["add-issue-label"]; exists {
				if labelsMap, ok := labels.(map[string]any); ok {
					labelConfig := &AddIssueLabelsConfig{}

					// Parse allowed labels (optional)
					if allowed, exists := labelsMap["allowed"]; exists {
						if allowedArray, ok := allowed.([]any); ok {
							var allowedStrings []string
							for _, label := range allowedArray {
								if labelStr, ok := label.(string); ok {
									allowedStrings = append(allowedStrings, labelStr)
								}
							}
							labelConfig.Allowed = allowedStrings
						}
					}

					// Parse max (optional)
					if maxCount, exists := labelsMap["max"]; exists {
						// Handle different numeric types that YAML parsers might return
						var maxCountInt int
						var validMaxCount bool
						switch v := maxCount.(type) {
						case int:
							maxCountInt = v
							validMaxCount = true
						case int64:
							maxCountInt = int(v)
							validMaxCount = true
						case uint64:
							maxCountInt = int(v)
							validMaxCount = true
						case float64:
							maxCountInt = int(v)
							validMaxCount = true
						}
						if validMaxCount {
							labelConfig.MaxCount = &maxCountInt
						}
					}

					config.AddIssueLabels = labelConfig
				} else if labels == nil {
					// Handle null case: create empty config (allows any labels)
					config.AddIssueLabels = &AddIssueLabelsConfig{}
				}
			}

			// Handle update-issue
			updateIssuesConfig := c.parseUpdateIssuesConfig(outputMap)
			if updateIssuesConfig != nil {
				config.UpdateIssues = updateIssuesConfig
			}

			// Handle push-to-branch
			pushToBranchConfig := c.parsePushToBranchConfig(outputMap)
			if pushToBranchConfig != nil {
				config.PushToBranch = pushToBranchConfig
			}

			// Handle missing-tool (parse configuration if present)
			missingToolConfig := c.parseMissingToolConfig(outputMap)
			if missingToolConfig != nil {
				config.MissingTool = missingToolConfig
			}
		}
	}

	return config
}

// parseIssuesConfig handles create-issue configuration
func (c *Compiler) parseIssuesConfig(outputMap map[string]any) *CreateIssuesConfig {
	if configData, exists := outputMap["create-issue"]; exists {
		issuesConfig := &CreateIssuesConfig{Max: 1} // Default max is 1

		if configMap, ok := configData.(map[string]any); ok {
			// Parse title-prefix
			if titlePrefix, exists := configMap["title-prefix"]; exists {
				if titlePrefixStr, ok := titlePrefix.(string); ok {
					issuesConfig.TitlePrefix = titlePrefixStr
				}
			}

			// Parse labels
			if labels, exists := configMap["labels"]; exists {
				if labelsArray, ok := labels.([]any); ok {
					var labelStrings []string
					for _, label := range labelsArray {
						if labelStr, ok := label.(string); ok {
							labelStrings = append(labelStrings, labelStr)
						}
					}
					issuesConfig.Labels = labelStrings
				}
			}

			// Parse max
			if max, exists := configMap["max"]; exists {
				if maxInt, ok := c.parseIntValue(max); ok {
					issuesConfig.Max = maxInt
				}
			}
		}

		return issuesConfig
	}

	return nil
}

// parseDiscussionsConfig handles create-discussion configuration
func (c *Compiler) parseDiscussionsConfig(outputMap map[string]any) *CreateDiscussionsConfig {
	if configData, exists := outputMap["create-discussion"]; exists {
		discussionsConfig := &CreateDiscussionsConfig{Max: 1} // Default max is 1

		if configMap, ok := configData.(map[string]any); ok {
			// Parse title-prefix
			if titlePrefix, exists := configMap["title-prefix"]; exists {
				if titlePrefixStr, ok := titlePrefix.(string); ok {
					discussionsConfig.TitlePrefix = titlePrefixStr
				}
			}

			// Parse category-id
			if categoryId, exists := configMap["category-id"]; exists {
				if categoryIdStr, ok := categoryId.(string); ok {
					discussionsConfig.CategoryId = categoryIdStr
				}
			}

			// Parse max
			if max, exists := configMap["max"]; exists {
				if maxInt, ok := c.parseIntValue(max); ok {
					discussionsConfig.Max = maxInt
				}
			}
		}

		return discussionsConfig
	}

	return nil
}

// parseCommentsConfig handles add-issue-comment configuration
func (c *Compiler) parseCommentsConfig(outputMap map[string]any) *AddIssueCommentsConfig {
	if configData, exists := outputMap["add-issue-comment"]; exists {
		commentsConfig := &AddIssueCommentsConfig{Max: 1} // Default max is 1

		if configMap, ok := configData.(map[string]any); ok {
			// Parse max
			if max, exists := configMap["max"]; exists {
				if maxInt, ok := c.parseIntValue(max); ok {
					commentsConfig.Max = maxInt
				}
			}

			// Parse target
			if target, exists := configMap["target"]; exists {
				if targetStr, ok := target.(string); ok {
					commentsConfig.Target = targetStr
				}
			}
		}

		return commentsConfig
	}

	return nil
}

// parsePullRequestsConfig handles only create-pull-request (singular) configuration
func (c *Compiler) parsePullRequestsConfig(outputMap map[string]any) *CreatePullRequestsConfig {
	// Check for singular form only
	if _, exists := outputMap["create-pull-request"]; !exists {
		return nil
	}

	configData := outputMap["create-pull-request"]
	pullRequestsConfig := &CreatePullRequestsConfig{Max: 1} // Always max 1 for pull requests

	if configMap, ok := configData.(map[string]any); ok {
		// Parse title-prefix
		if titlePrefix, exists := configMap["title-prefix"]; exists {
			if titlePrefixStr, ok := titlePrefix.(string); ok {
				pullRequestsConfig.TitlePrefix = titlePrefixStr
			}
		}

		// Parse labels
		if labels, exists := configMap["labels"]; exists {
			if labelsArray, ok := labels.([]any); ok {
				var labelStrings []string
				for _, label := range labelsArray {
					if labelStr, ok := label.(string); ok {
						labelStrings = append(labelStrings, labelStr)
					}
				}
				pullRequestsConfig.Labels = labelStrings
			}
		}

		// Parse draft
		if draft, exists := configMap["draft"]; exists {
			if draftBool, ok := draft.(bool); ok {
				pullRequestsConfig.Draft = &draftBool
			}
		}

		// Note: max parameter is not supported for pull requests (always limited to 1)
		// If max is specified, it will be ignored as pull requests are singular only
	}

	return pullRequestsConfig
}

// parsePullRequestReviewCommentsConfig handles create-pull-request-review-comment configuration
func (c *Compiler) parsePullRequestReviewCommentsConfig(outputMap map[string]any) *CreatePullRequestReviewCommentsConfig {
	if _, exists := outputMap["create-pull-request-review-comment"]; !exists {
		return nil
	}

	configData := outputMap["create-pull-request-review-comment"]
	prReviewCommentsConfig := &CreatePullRequestReviewCommentsConfig{Max: 10, Side: "RIGHT"} // Default max is 10, side is RIGHT

	if configMap, ok := configData.(map[string]any); ok {
		// Parse max
		if max, exists := configMap["max"]; exists {
			if maxInt, ok := c.parseIntValue(max); ok {
				prReviewCommentsConfig.Max = maxInt
			}
		}

		// Parse side
		if side, exists := configMap["side"]; exists {
			if sideStr, ok := side.(string); ok {
				// Validate side value
				if sideStr == "LEFT" || sideStr == "RIGHT" {
					prReviewCommentsConfig.Side = sideStr
				}
			}
		}
	}

	return prReviewCommentsConfig
}

// parseSecurityReportsConfig handles create-security-report configuration
func (c *Compiler) parseSecurityReportsConfig(outputMap map[string]any) *CreateSecurityReportsConfig {
	if _, exists := outputMap["create-security-report"]; !exists {
		return nil
	}

	configData := outputMap["create-security-report"]
	securityReportsConfig := &CreateSecurityReportsConfig{Max: 0} // Default max is 0 (unlimited)

	if configMap, ok := configData.(map[string]any); ok {
		// Parse max
		if max, exists := configMap["max"]; exists {
			if maxInt, ok := c.parseIntValue(max); ok {
				securityReportsConfig.Max = maxInt
			}
		}

		// Parse driver
		if driver, exists := configMap["driver"]; exists {
			if driverStr, ok := driver.(string); ok {
				securityReportsConfig.Driver = driverStr
			}
		}
	}

	return securityReportsConfig
}

// parseIntValue safely parses various numeric types to int
func (c *Compiler) parseIntValue(value any) (int, bool) {
	switch v := value.(type) {
	case int:
		return v, true
	case int64:
		return int(v), true
	case uint64:
		return int(v), true
	case float64:
		return int(v), true
	default:
		return 0, false
	}
}

// parseUpdateIssuesConfig handles update-issue configuration
func (c *Compiler) parseUpdateIssuesConfig(outputMap map[string]any) *UpdateIssuesConfig {
	if configData, exists := outputMap["update-issue"]; exists {
		updateIssuesConfig := &UpdateIssuesConfig{Max: 1} // Default max is 1

		if configMap, ok := configData.(map[string]any); ok {
			// Parse max
			if max, exists := configMap["max"]; exists {
				if maxInt, ok := c.parseIntValue(max); ok {
					updateIssuesConfig.Max = maxInt
				}
			}

			// Parse target
			if target, exists := configMap["target"]; exists {
				if targetStr, ok := target.(string); ok {
					updateIssuesConfig.Target = targetStr
				}
			}

			// Parse status - presence of the key (even if nil/empty) indicates field can be updated
			if _, exists := configMap["status"]; exists {
				// If the key exists, it means we can update the status
				// We don't care about the value - just that the key is present
				updateIssuesConfig.Status = new(bool) // Allocate a new bool pointer (defaults to false)
			}

			// Parse title - presence of the key (even if nil/empty) indicates field can be updated
			if _, exists := configMap["title"]; exists {
				updateIssuesConfig.Title = new(bool)
			}

			// Parse body - presence of the key (even if nil/empty) indicates field can be updated
			if _, exists := configMap["body"]; exists {
				updateIssuesConfig.Body = new(bool)
			}
		}

		return updateIssuesConfig
	}

	return nil
}

// parsePushToBranchConfig handles push-to-branch configuration
func (c *Compiler) parsePushToBranchConfig(outputMap map[string]any) *PushToBranchConfig {
	if configData, exists := outputMap["push-to-branch"]; exists {
		pushToBranchConfig := &PushToBranchConfig{
			Branch: "triggering", // Default branch value
		}

		// Handle the case where configData is nil (push-to-branch: with no value)
		if configData == nil {
			return pushToBranchConfig
		}

		if configMap, ok := configData.(map[string]any); ok {
			// Parse branch (optional, defaults to "triggering")
			if branch, exists := configMap["branch"]; exists {
				if branchStr, ok := branch.(string); ok {
					pushToBranchConfig.Branch = branchStr
				}
			}

			// Parse target (optional, similar to add-issue-comment)
			if target, exists := configMap["target"]; exists {
				if targetStr, ok := target.(string); ok {
					pushToBranchConfig.Target = targetStr
				}
			}
		}

		return pushToBranchConfig
	}

	return nil
}

// parseMissingToolConfig handles missing-tool configuration
func (c *Compiler) parseMissingToolConfig(outputMap map[string]any) *MissingToolConfig {
	if configData, exists := outputMap["missing-tool"]; exists {
		missingToolConfig := &MissingToolConfig{} // Default: no max limit

		// Handle the case where configData is nil (missing-tool: with no value)
		if configData == nil {
			return missingToolConfig
		}

		if configMap, ok := configData.(map[string]any); ok {
			// Parse max (optional)
			if max, exists := configMap["max"]; exists {
				// Handle different numeric types that YAML parsers might return
				var maxInt int
				var validMax bool
				switch v := max.(type) {
				case int:
					maxInt = v
					validMax = true
				case int64:
					maxInt = int(v)
					validMax = true
				case uint64:
					maxInt = int(v)
					validMax = true
				case float64:
					maxInt = int(v)
					validMax = true
				}
				if validMax {
					missingToolConfig.Max = maxInt
				}
			}
		}

		return missingToolConfig
	}

	return nil
}

// buildCustomJobs creates custom jobs defined in the frontmatter jobs section
func (c *Compiler) buildCustomJobs(data *WorkflowData) error {
	for jobName, jobConfig := range data.Jobs {
		if configMap, ok := jobConfig.(map[string]any); ok {
			job := &Job{
				Name: jobName,
			}

			// Extract job dependencies
			if depends, hasDeps := configMap["depends"]; hasDeps {
				if depsList, ok := depends.([]any); ok {
					for _, dep := range depsList {
						if depStr, ok := dep.(string); ok {
							job.Depends = append(job.Depends, depStr)
						}
					}
				} else if depStr, ok := depends.(string); ok {
					// Single dependency as string
					job.Depends = append(job.Depends, depStr)
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
					job.If = fmt.Sprintf("if: %s", ifStr)
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
	// Simple YAML generation for steps
	var stepYAML strings.Builder

	// Add step name
	if name, hasName := stepMap["name"]; hasName {
		if nameStr, ok := name.(string); ok {
			stepYAML.WriteString(fmt.Sprintf("      - name: %s\n", nameStr))
		}
	}

	// Add run command
	if run, hasRun := stepMap["run"]; hasRun {
		if runStr, ok := run.(string); ok {
			if strings.Contains(runStr, "\n") {
				// Multi-line run command - use literal block scalar
				stepYAML.WriteString("        run: |\n")
				for _, line := range strings.Split(runStr, "\n") {
					stepYAML.WriteString("          " + line + "\n")
				}
			} else {
				// Single-line run command
				stepYAML.WriteString(fmt.Sprintf("        run: %s\n", runStr))
			}
		}
	}

	// Add uses action
	if uses, hasUses := stepMap["uses"]; hasUses {
		if usesStr, ok := uses.(string); ok {
			stepYAML.WriteString(fmt.Sprintf("        uses: %s\n", usesStr))
		}
	}

	// Add with parameters
	if with, hasWith := stepMap["with"]; hasWith {
		if withMap, ok := with.(map[string]any); ok {
			stepYAML.WriteString("        with:\n")
			for key, value := range withMap {
				stepYAML.WriteString(fmt.Sprintf("          %s: %v\n", key, value))
			}
		}
	}

	return stepYAML.String(), nil
}

// generateEngineExecutionSteps generates the execution steps for the specified agentic engine
func (c *Compiler) generateEngineExecutionSteps(yaml *strings.Builder, data *WorkflowData, engine AgenticEngine, logFile string) {

	// Handle custom engine (with or without user-defined steps)
	if engine.GetID() == "custom" {
		c.generateCustomEngineSteps(yaml, data, logFile)
		return
	}

	executionConfig := engine.GetExecutionConfig(data.Name, logFile, data.EngineConfig, data.NetworkPermissions, data.SafeOutputs != nil)

	// If the execution config contains custom steps, inject them before the main command/action
	if len(executionConfig.Steps) > 0 {
		for i, step := range executionConfig.Steps {
			stepYAML, err := c.convertStepToYAML(step)
			if err != nil {
				// Log error but continue with other steps
				fmt.Printf("Error converting step %d to YAML: %v\n", i+1, err)
				continue
			}

			// The convertStepToYAML already includes proper indentation, just add it directly
			yaml.WriteString(stepYAML)
		}
	}

	if executionConfig.Command != "" {
		// Command-based execution (e.g., Codex)
		fmt.Fprintf(yaml, "      - name: %s\n", executionConfig.StepName)
		yaml.WriteString("        run: |\n")

		// Split command into lines and indent them properly
		commandLines := strings.Split(executionConfig.Command, "\n")
		for _, line := range commandLines {
			yaml.WriteString("          " + line + "\n")
		}
		env := executionConfig.Environment

		if data.SafeOutputs != nil {
			env["GITHUB_AW_SAFE_OUTPUTS"] = "${{ env.GITHUB_AW_SAFE_OUTPUTS }}"
		}
		// Add environment variables
		if len(env) > 0 {
			yaml.WriteString("        env:\n")
			// Sort environment keys for consistent output
			envKeys := make([]string, 0, len(env))
			for key := range env {
				envKeys = append(envKeys, key)
			}
			sort.Strings(envKeys)

			for _, key := range envKeys {
				value := env[key]
				fmt.Fprintf(yaml, "          %s: %s\n", key, value)
			}
		}
	} else if executionConfig.Action != "" {

		// Add the main action step
		fmt.Fprintf(yaml, "      - name: %s\n", executionConfig.StepName)
		yaml.WriteString("        id: agentic_execution\n")
		fmt.Fprintf(yaml, "        uses: %s\n", executionConfig.Action)
		yaml.WriteString("        with:\n")

		// Add inputs in alphabetical order by key
		keys := make([]string, 0, len(executionConfig.Inputs))
		for key := range executionConfig.Inputs {
			keys = append(keys, key)
		}
		sort.Strings(keys)

		for _, key := range keys {
			value := executionConfig.Inputs[key]
			if key == "allowed_tools" {
				if data.AllowedTools != "" {
					// Add comment listing all allowed tools for readability
					comment := c.generateAllowedToolsComment(data.AllowedTools, "          ")
					yaml.WriteString(comment)
					fmt.Fprintf(yaml, "          %s: \"%s\"\n", key, data.AllowedTools)
				}
			} else if key == "timeout_minutes" {
				if data.TimeoutMinutes != "" {
					yaml.WriteString("          " + data.TimeoutMinutes + "\n")
				}
			} else if key == "max_turns" {
				if data.EngineConfig != nil && data.EngineConfig.MaxTurns != "" {
					fmt.Fprintf(yaml, "          max_turns: %s\n", data.EngineConfig.MaxTurns)
				}
			} else if value != "" {
				if strings.HasPrefix(value, "|") {
					// For YAML literal block scalars, add proper newline after the content
					fmt.Fprintf(yaml, "          %s: %s\n", key, value)
				} else {
					fmt.Fprintf(yaml, "          %s: %s\n", key, value)
				}
			}
		}
		// Add environment section for safe-outputs, max-turns, and custom env vars
		hasEnvSection := data.SafeOutputs != nil || (data.EngineConfig != nil && len(data.EngineConfig.Env) > 0) || (data.EngineConfig != nil && data.EngineConfig.MaxTurns != "")
		if hasEnvSection {
			yaml.WriteString("        env:\n")

			// Add GITHUB_AW_SAFE_OUTPUTS if safe-outputs feature is used
			if data.SafeOutputs != nil {
				yaml.WriteString("          GITHUB_AW_SAFE_OUTPUTS: ${{ env.GITHUB_AW_SAFE_OUTPUTS }}\n")
			}

			// Add GITHUB_AW_MAX_TURNS if max-turns is configured
			if data.EngineConfig != nil && data.EngineConfig.MaxTurns != "" {
				fmt.Fprintf(yaml, "          GITHUB_AW_MAX_TURNS: %s\n", data.EngineConfig.MaxTurns)
			}

			// Add custom environment variables from engine config
			if data.EngineConfig != nil && len(data.EngineConfig.Env) > 0 {
				for _, envVar := range data.EngineConfig.Env {
					// Parse environment variable in format "KEY=value" or "KEY: value"
					parts := strings.SplitN(envVar, "=", 2)
					if len(parts) == 2 {
						key := strings.TrimSpace(parts[0])
						value := strings.TrimSpace(parts[1])
						fmt.Fprintf(yaml, "          %s: %s\n", key, value)
					} else {
						// Try "KEY: value" format
						parts = strings.SplitN(envVar, ":", 2)
						if len(parts) == 2 {
							key := strings.TrimSpace(parts[0])
							value := strings.TrimSpace(parts[1])
							fmt.Fprintf(yaml, "          %s: %s\n", key, value)
						}
					}
				}
			}
		}
		yaml.WriteString("      - name: Capture Agentic Action logs\n")
		yaml.WriteString("        if: always()\n")
		yaml.WriteString("        run: |\n")
		yaml.WriteString("          # Copy the detailed execution file from Agentic Action if available\n")
		yaml.WriteString("          if [ -n \"${{ steps.agentic_execution.outputs.execution_file }}\" ] && [ -f \"${{ steps.agentic_execution.outputs.execution_file }}\" ]; then\n")
		yaml.WriteString("            cp ${{ steps.agentic_execution.outputs.execution_file }} " + logFile + "\n")
		yaml.WriteString("          else\n")
		yaml.WriteString("            echo \"No execution file output found from Agentic Action\" >> " + logFile + "\n")
		yaml.WriteString("          fi\n")
		yaml.WriteString("          \n")
		yaml.WriteString("          # Ensure log file exists\n")
		yaml.WriteString("          touch " + logFile + "\n")
	}
}

// generateCustomEngineSteps generates the custom steps defined in the engine configuration
func (c *Compiler) generateCustomEngineSteps(yaml *strings.Builder, data *WorkflowData, logFile string) {
	// Generate each custom step if they exist, with environment variables
	if data.EngineConfig != nil && len(data.EngineConfig.Steps) > 0 {
		// Check if we need environment section for any step
		hasEnvSection := data.SafeOutputs != nil || (data.EngineConfig != nil && data.EngineConfig.MaxTurns != "") || (data.EngineConfig != nil && len(data.EngineConfig.Env) > 0)

		for i, step := range data.EngineConfig.Steps {
			stepYAML, err := c.convertStepToYAML(step)
			if err != nil {
				// Log error but continue with other steps
				fmt.Printf("Error converting step %d to YAML: %v\n", i+1, err)
				continue
			}

			// Check if this step needs environment variables injected
			stepStr := stepYAML
			if hasEnvSection && strings.Contains(stepYAML, "run:") {
				// Add environment variables to run steps after the entire run block
				// Find the end of the run block and add env section at step level
				stepStr = strings.TrimRight(stepYAML, "\n")
				stepStr += "\n        env:\n"

				// Add GITHUB_AW_SAFE_OUTPUTS if safe-outputs feature is used
				if data.SafeOutputs != nil {
					stepStr += "          GITHUB_AW_SAFE_OUTPUTS: ${{ env.GITHUB_AW_SAFE_OUTPUTS }}\n"
				}

				// Add GITHUB_AW_MAX_TURNS if max-turns is configured
				if data.EngineConfig != nil && data.EngineConfig.MaxTurns != "" {
					stepStr += fmt.Sprintf("          GITHUB_AW_MAX_TURNS: %s\n", data.EngineConfig.MaxTurns)
				}

				// Add custom environment variables from engine config
				if data.EngineConfig != nil && len(data.EngineConfig.Env) > 0 {
					for _, envVar := range data.EngineConfig.Env {
						// Parse environment variable in format "KEY=value" or "KEY: value"
						parts := strings.SplitN(envVar, "=", 2)
						if len(parts) == 2 {
							key := strings.TrimSpace(parts[0])
							value := strings.TrimSpace(parts[1])
							stepStr += fmt.Sprintf("          %s: %s\n", key, value)
						} else {
							// Try "KEY: value" format
							parts = strings.SplitN(envVar, ":", 2)
							if len(parts) == 2 {
								key := strings.TrimSpace(parts[0])
								value := strings.TrimSpace(parts[1])
								stepStr += fmt.Sprintf("          %s: %s\n", key, value)
							}
						}
					}
				}
			}

			// The convertStepToYAML already includes proper indentation, just add it directly
			yaml.WriteString(stepStr)
		}
	}

	// Add a step to ensure the log file exists for consistency with other engines
	yaml.WriteString("      - name: Ensure log file exists\n")
	yaml.WriteString("        run: |\n")
	yaml.WriteString("          echo \"Custom steps execution completed\" >> " + logFile + "\n")
	yaml.WriteString("          touch " + logFile + "\n")
}

// generateCreateAwInfo generates a step that creates aw_info.json with agentic run metadata
func (c *Compiler) generateCreateAwInfo(yaml *strings.Builder, data *WorkflowData, engine AgenticEngine) {
	yaml.WriteString("      - name: Generate agentic run info\n")
	yaml.WriteString("        uses: actions/github-script@v7\n")
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
	fmt.Fprintf(yaml, "              supports_tools_whitelist: %t,\n", engine.SupportsToolsWhitelist())
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
	yaml.WriteString("              created_at: new Date().toISOString()\n")

	yaml.WriteString("            };\n")
	yaml.WriteString("            \n")
	yaml.WriteString("            // Write to /tmp directory to avoid inclusion in PR\n")
	yaml.WriteString("            const tmpPath = '/tmp/aw_info.json';\n")
	yaml.WriteString("            fs.writeFileSync(tmpPath, JSON.stringify(awInfo, null, 2));\n")
	yaml.WriteString("            console.log('Generated aw_info.json at:', tmpPath);\n")
	yaml.WriteString("            console.log(JSON.stringify(awInfo, null, 2));\n")
}

// generateOutputFileSetup generates a step that sets up the GITHUB_AW_SAFE_OUTPUTS environment variable
func (c *Compiler) generateOutputFileSetup(yaml *strings.Builder, data *WorkflowData) {
	yaml.WriteString("      - name: Setup agent output\n")
	yaml.WriteString("        id: setup_agent_output\n")
	yaml.WriteString("        uses: actions/github-script@v7\n")
	yaml.WriteString("        with:\n")
	yaml.WriteString("          script: |\n")

	// Use the embedded setup agent output script
	WriteJavaScriptToYAML(yaml, setupAgentOutputScript)
}

// generateOutputCollectionStep generates a step that reads the output file and sets it as a GitHub Actions output
func (c *Compiler) generateOutputCollectionStep(yaml *strings.Builder, data *WorkflowData) {
	yaml.WriteString("      - name: Collect agent output\n")
	yaml.WriteString("        id: collect_output\n")
	yaml.WriteString("        uses: actions/github-script@v7\n")

	// Add environment variables for JSONL validation
	yaml.WriteString("        env:\n")
	yaml.WriteString("          GITHUB_AW_SAFE_OUTPUTS: ${{ env.GITHUB_AW_SAFE_OUTPUTS }}\n")

	// Pass the safe-outputs configuration for validation
	if data.SafeOutputs != nil {
		// Create a simplified config object for validation
		safeOutputsConfig := make(map[string]interface{})
		if data.SafeOutputs.CreateIssues != nil {
			safeOutputsConfig["create-issue"] = true
		}
		if data.SafeOutputs.AddIssueComments != nil {
			// Pass the full comment configuration including target
			commentConfig := map[string]interface{}{
				"enabled": true,
			}
			if data.SafeOutputs.AddIssueComments.Target != "" {
				commentConfig["target"] = data.SafeOutputs.AddIssueComments.Target
			}
			safeOutputsConfig["add-issue-comment"] = commentConfig
		}
		if data.SafeOutputs.CreatePullRequests != nil {
			safeOutputsConfig["create-pull-request"] = true
		}
		if data.SafeOutputs.AddIssueLabels != nil {
			safeOutputsConfig["add-issue-label"] = true
		}
		if data.SafeOutputs.UpdateIssues != nil {
			safeOutputsConfig["update-issue"] = true
		}
		if data.SafeOutputs.PushToBranch != nil {
			pushToBranchConfig := map[string]interface{}{
				"enabled": true,
				"branch":  data.SafeOutputs.PushToBranch.Branch,
			}
			if data.SafeOutputs.PushToBranch.Target != "" {
				pushToBranchConfig["target"] = data.SafeOutputs.PushToBranch.Target
			}
			safeOutputsConfig["push-to-branch"] = pushToBranchConfig
		}
		if data.SafeOutputs.MissingTool != nil {
			missingToolConfig := map[string]interface{}{
				"enabled": true,
			}
			if data.SafeOutputs.MissingTool.Max > 0 {
				missingToolConfig["max"] = data.SafeOutputs.MissingTool.Max
			}
			safeOutputsConfig["missing-tool"] = missingToolConfig
		}

		// Convert to JSON string for environment variable
		configJSON, _ := json.Marshal(safeOutputsConfig)
		fmt.Fprintf(yaml, "          GITHUB_AW_SAFE_OUTPUTS_CONFIG: %q\n", string(configJSON))
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

	yaml.WriteString("      - name: Print agent output to step summary\n")
	yaml.WriteString("        env:\n")
	yaml.WriteString("          GITHUB_AW_SAFE_OUTPUTS: ${{ env.GITHUB_AW_SAFE_OUTPUTS }}\n")
	yaml.WriteString("        run: |\n")
	yaml.WriteString("          echo \"## Agent Output (JSONL)\" >> $GITHUB_STEP_SUMMARY\n")
	yaml.WriteString("          echo \"\" >> $GITHUB_STEP_SUMMARY\n")
	yaml.WriteString("          echo '``````json' >> $GITHUB_STEP_SUMMARY\n")
	yaml.WriteString("          cat ${{ env.GITHUB_AW_SAFE_OUTPUTS }} >> $GITHUB_STEP_SUMMARY\n")
	yaml.WriteString("          # Ensure there's a newline after the file content if it doesn't end with one\n")
	yaml.WriteString("          if [ -s ${{ env.GITHUB_AW_SAFE_OUTPUTS }} ] && [ \"$(tail -c1 ${{ env.GITHUB_AW_SAFE_OUTPUTS }})\" != \"\" ]; then\n")
	yaml.WriteString("            echo \"\" >> $GITHUB_STEP_SUMMARY\n")
	yaml.WriteString("          fi\n")
	yaml.WriteString("          echo '``````' >> $GITHUB_STEP_SUMMARY\n")
	yaml.WriteString("      - name: Upload agentic output file\n")
	yaml.WriteString("        if: always() && steps.collect_output.outputs.output != ''\n")
	yaml.WriteString("        uses: actions/upload-artifact@v4\n")
	yaml.WriteString("        with:\n")
	fmt.Fprintf(yaml, "          name: %s\n", OutputArtifactName)
	yaml.WriteString("          path: ${{ env.GITHUB_AW_SAFE_OUTPUTS }}\n")
	yaml.WriteString("          if-no-files-found: warn\n")

}

// validateHTTPTransportSupport validates that HTTP MCP servers are only used with engines that support HTTP transport
func (c *Compiler) validateHTTPTransportSupport(tools map[string]any, engine AgenticEngine) error {
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
func (c *Compiler) validateMaxTurnsSupport(frontmatter map[string]any, engine AgenticEngine) error {
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
