package workflow

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/githubnext/gh-aw/pkg/console"
	"github.com/githubnext/gh-aw/pkg/constants"
	"github.com/githubnext/gh-aw/pkg/parser"
	"github.com/goccy/go-yaml"
	"github.com/santhosh-tekuri/jsonschema/v6"
)

// FileTracker interface for tracking files created during compilation
type FileTracker interface {
	TrackCreated(filePath string)
}

//go:embed templates/reaction_action.yaml
var reactionActionTemplate string

//go:embed templates/compute_text_action.yaml
var computeTextActionTemplate string

//go:embed templates/check_team_member.yaml
var checkTeamMemberTemplate string

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
	Name             string
	On               string
	Permissions      string
	Concurrency      string
	RunName          string
	Env              string
	If               string
	TimeoutMinutes   string
	CustomSteps      string
	PostSteps        string // steps to run after AI execution
	RunsOn           string
	Tools            map[string]any
	MarkdownContent  string
	AllowedTools     string
	AI               string        // "claude" or "codex" (for backwards compatibility)
	EngineConfig     *EngineConfig // Extended engine configuration
	MaxTurns         string
	StopTime         string
	Alias            string         // for @alias trigger support
	AliasOtherEvents map[string]any // for merging alias with other events
	AIReaction       string         // AI reaction type like "eyes", "heart", etc.
	Jobs             map[string]any // custom job configurations with dependencies
	Cache            string         // cache configuration
	NeedsTextOutput  bool           // whether the workflow uses ${{ needs.task.outputs.text }}
	Output           *OutputConfig  // output configuration for automatic output routes
}

// OutputConfig holds configuration for automatic output routes
type OutputConfig struct {
	Issue       *IssueConfig       `yaml:"issue,omitempty"`
	Comment     *CommentConfig     `yaml:"comment,omitempty"`
	PullRequest *PullRequestConfig `yaml:"pull-request,omitempty"`
}

// IssueConfig holds configuration for creating GitHub issues from agent output
type IssueConfig struct {
	TitlePrefix string   `yaml:"title-prefix,omitempty"`
	Labels      []string `yaml:"labels,omitempty"`
}

// CommentConfig holds configuration for creating GitHub issue/PR comments from agent output
type CommentConfig struct {
	// Empty struct for now, as per requirements, but structured for future expansion
}

// PullRequestConfig holds configuration for creating GitHub pull requests from agent output
type PullRequestConfig struct {
	TitlePrefix string   `yaml:"title-prefix,omitempty"`
	Labels      []string `yaml:"labels,omitempty"`
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

	// Write shared reaction action only if ai-reaction is configured
	if workflowData.AIReaction != "" {
		if err := c.writeReactionAction(markdownPath); err != nil {
			formattedErr := console.FormatError(console.CompilerError{
				Position: console.ErrorPosition{
					File:   markdownPath,
					Line:   1,
					Column: 1,
				},
				Type:    "error",
				Message: fmt.Sprintf("failed to write reaction action: %v", err),
			})
			return errors.New(formattedErr)
		}
	}

	// Write shared compute-text action (only if needed for task job)
	if workflowData.NeedsTextOutput {
		if err := c.writeComputeTextAction(markdownPath); err != nil {
			formattedErr := console.FormatError(console.CompilerError{
				Position: console.ErrorPosition{
					File:   markdownPath,
					Line:   1,
					Column: 1,
				},
				Type:    "error",
				Message: fmt.Sprintf("failed to write compute-text action: %v", err),
			})
			return errors.New(formattedErr)
		}
	}

	// Write shared check-team-member action (only for alias workflows)
	if workflowData.Alias != "" {
		if err := c.writeCheckTeamMemberAction(markdownPath); err != nil {
			formattedErr := console.FormatError(console.CompilerError{
				Position: console.ErrorPosition{
					File:   markdownPath,
					Line:   1,
					Column: 1,
				},
				Type:    "error",
				Message: fmt.Sprintf("failed to write check-team-member action: %v", err),
			})
			return errors.New(formattedErr)
		}
	}

	// Generate the YAML content
	if c.verbose {
		fmt.Println(console.FormatInfoMessage("Generating GitHub Actions YAML..."))
	}
	yamlContent, err := c.generateYAML(workflowData)
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
			tools = c.applyDefaultGitHubMCPTools(tools)
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
		Name:            workflowName,
		Tools:           tools,
		MarkdownContent: markdownContent,
		AI:              engineSetting,
		EngineConfig:    engineConfig,
		NeedsTextOutput: needsTextOutput,
	}

	// Extract YAML sections from frontmatter - use direct frontmatter map extraction
	// to avoid issues with nested keys (e.g., tools.mcps.*.env being confused with top-level env)
	workflowData.On = c.extractTopLevelYAMLSection(result.Frontmatter, "on")
	workflowData.Permissions = c.extractTopLevelYAMLSection(result.Frontmatter, "permissions")
	workflowData.Concurrency = c.extractTopLevelYAMLSection(result.Frontmatter, "concurrency")
	workflowData.RunName = c.extractTopLevelYAMLSection(result.Frontmatter, "run-name")
	workflowData.Env = c.extractTopLevelYAMLSection(result.Frontmatter, "env")
	workflowData.If = c.extractTopLevelYAMLSection(result.Frontmatter, "if")
	workflowData.TimeoutMinutes = c.extractTopLevelYAMLSection(result.Frontmatter, "timeout_minutes")
	workflowData.CustomSteps = c.extractTopLevelYAMLSection(result.Frontmatter, "steps")
	workflowData.PostSteps = c.extractTopLevelYAMLSection(result.Frontmatter, "post-steps")
	workflowData.RunsOn = c.extractTopLevelYAMLSection(result.Frontmatter, "runs-on")
	workflowData.Cache = c.extractTopLevelYAMLSection(result.Frontmatter, "cache")
	workflowData.MaxTurns = c.extractYAMLValue(result.Frontmatter, "max-turns")
	workflowData.StopTime = c.extractYAMLValue(result.Frontmatter, "stop-time")

	// Resolve relative stop-time to absolute time if needed
	if workflowData.StopTime != "" {
		resolvedStopTime, err := resolveStopTime(workflowData.StopTime, time.Now().UTC())
		if err != nil {
			return nil, fmt.Errorf("invalid stop-time format: %w", err)
		}
		originalStopTime := c.extractYAMLValue(result.Frontmatter, "stop-time")
		workflowData.StopTime = resolvedStopTime

		if c.verbose && isRelativeStopTime(originalStopTime) {
			fmt.Println(console.FormatInfoMessage(fmt.Sprintf("Resolved relative stop-time to: %s", resolvedStopTime)))
		} else if c.verbose && originalStopTime != resolvedStopTime {
			fmt.Println(console.FormatInfoMessage(fmt.Sprintf("Parsed absolute stop-time from '%s' to: %s", originalStopTime, resolvedStopTime)))
		}
	}

	workflowData.Alias = c.extractAliasName(result.Frontmatter)
	workflowData.AIReaction = c.extractYAMLValue(result.Frontmatter, "ai-reaction")
	workflowData.Jobs = c.extractJobsFromFrontmatter(result.Frontmatter)

	// Parse output configuration
	workflowData.Output = c.extractOutputConfig(result.Frontmatter)

	// Check if "alias" is used as a trigger in the "on" section
	var hasAlias bool
	var otherEvents map[string]any
	if onValue, exists := result.Frontmatter["on"]; exists {
		// Check for new format: on.alias
		if onMap, ok := onValue.(map[string]any); ok {
			if _, hasAliasKey := onMap["alias"]; hasAliasKey {
				hasAlias = true
				// Set default alias to filename if not specified in the alias section
				if workflowData.Alias == "" {
					baseName := strings.TrimSuffix(filepath.Base(markdownPath), ".md")
					workflowData.Alias = baseName
				}
				// Check for conflicting events
				conflictingEvents := []string{"issues", "issue_comment", "pull_request", "pull_request_review_comment"}
				for _, eventName := range conflictingEvents {
					if _, hasConflict := onMap[eventName]; hasConflict {
						return nil, fmt.Errorf("cannot use 'alias' with '%s' in the same workflow", eventName)
					}
				}

				// Extract other (non-conflicting) events
				otherEvents = make(map[string]any)
				for key, value := range onMap {
					if key != "alias" {
						otherEvents[key] = value
					}
				}

				// Clear the On field so applyDefaults will handle alias trigger generation
				workflowData.On = ""
			}
		}
	}

	// Clear alias field if no alias trigger was found
	if !hasAlias {
		workflowData.Alias = ""
	}

	// Store other events for merging in applyDefaults
	if hasAlias && len(otherEvents) > 0 {
		// We'll store this and handle it in applyDefaults
		workflowData.On = "" // This will trigger alias handling in applyDefaults
		workflowData.AliasOtherEvents = otherEvents
	}

	// Apply defaults
	c.applyDefaults(workflowData, markdownPath)

	// Apply pull request draft filter if specified
	c.applyPullRequestDraftFilter(workflowData, result.Frontmatter)

	// Compute allowed tools
	workflowData.AllowedTools = c.computeAllowedTools(tools)

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

	// Special handling for "on" section - comment out draft field from pull_request
	if key == "on" {
		yamlStr = c.commentOutDraftInOnSection(yamlStr)
	}

	return yamlStr
}

// commentOutDraftInOnSection comments out draft fields in pull_request sections within the YAML string
// The draft field is processed separately by applyPullRequestDraftFilter and should be commented for documentation
func (c *Compiler) commentOutDraftInOnSection(yamlStr string) string {
	lines := strings.Split(yamlStr, "\n")
	var result []string
	inPullRequest := false

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
			}
		}

		// If we're in pull_request section and this line contains draft:, comment it out
		if inPullRequest && strings.Contains(strings.TrimSpace(line), "draft:") {
			// Preserve the original indentation and comment out the line
			indentation := ""
			trimmed := strings.TrimLeft(line, " \t")
			if len(line) > len(trimmed) {
				indentation = line[:len(line)-len(trimmed)]
			}

			commentedLine := indentation + "# " + trimmed + " # Draft filtering applied via job conditions"
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
} // extractAliasName extracts the alias name from frontmatter using the new nested format
func (c *Compiler) extractAliasName(frontmatter map[string]any) string {
	// Check new format: on.alias.name
	if onValue, exists := frontmatter["on"]; exists {
		if onMap, ok := onValue.(map[string]any); ok {
			if aliasValue, hasAlias := onMap["alias"]; hasAlias {
				if aliasMap, ok := aliasValue.(map[string]any); ok {
					if nameValue, hasName := aliasMap["name"]; hasName {
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
	// Check if this is an alias trigger workflow (by checking if user specified "on.alias")
	isAliasTrigger := false
	if data.On == "" {
		// Check the original frontmatter for alias trigger
		content, err := os.ReadFile(markdownPath)
		if err == nil {
			result, err := parser.ExtractFrontmatterFromContent(string(content))
			if err == nil {
				if onValue, exists := result.Frontmatter["on"]; exists {
					// Check for new format: on.alias
					if onMap, ok := onValue.(map[string]any); ok {
						if _, hasAlias := onMap["alias"]; hasAlias {
							isAliasTrigger = true
						}
					}
				}
			}
		}
	}

	if data.On == "" {
		if isAliasTrigger {
			// Generate alias-specific GitHub Actions events (updated to include reopened and pull_request)
			aliasEvents := `on:
  issues:
    types: [opened, edited, reopened]
  issue_comment:
    types: [created, edited]
  pull_request:
    types: [opened, edited, reopened]
  pull_request_review_comment:
    types: [created, edited]`

			// Check if there are other events to merge
			if len(data.AliasOtherEvents) > 0 {
				// Merge alias events with other events
				aliasEventsMap := map[string]any{
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

				// Merge other events into alias events
				for key, value := range data.AliasOtherEvents {
					aliasEventsMap[key] = value
				}

				// Convert merged events to YAML
				mergedEventsYAML, err := yaml.Marshal(map[string]any{"on": aliasEventsMap})
				if err == nil {
					data.On = strings.TrimSuffix(string(mergedEventsYAML), "\n")
				} else {
					// If conversion fails, just use alias events
					data.On = aliasEvents
				}
			} else {
				data.On = aliasEvents
			}

			// Add conditional logic to check for alias in issue content
			// Use event-aware condition that only applies alias checks to comment-related events
			hasOtherEvents := len(data.AliasOtherEvents) > 0
			aliasConditionTree := buildEventAwareAliasCondition(data.Alias, hasOtherEvents)

			if data.If == "" {
				data.If = fmt.Sprintf("if: %s", aliasConditionTree.Render())
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
		data.Permissions = `permissions:
  contents: read
  issues: read
  pull-requests: read
  discussions: read
  deployments: read
  models: read`
	}

	// Generate concurrency configuration using the dedicated concurrency module
	data.Concurrency = GenerateConcurrencyConfig(data, isAliasTrigger)

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

// applyDefaultGitHubMCPTools adds default read-only GitHub MCP tools, creating github tool if not present
func (c *Compiler) applyDefaultGitHubMCPTools(tools map[string]any) map[string]any {
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

	// Update the claude section with the new format
	claudeSection["allowed"] = claudeExistingAllowed
	tools["claude"] = claudeSection

	return tools
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
func (c *Compiler) computeAllowedTools(tools map[string]any) string {
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
func (c *Compiler) generateYAML(data *WorkflowData) (string, error) {
	// Reset job manager for this compilation
	c.jobManager = NewJobManager()

	// Build all jobs
	if err := c.buildJobs(data); err != nil {
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

// buildJobs creates all jobs for the workflow and adds them to the job manager
func (c *Compiler) buildJobs(data *WorkflowData) error {
	// Generate job name from workflow name
	jobName := c.generateJobName(data.Name)

	// Build task job first (preamble job that handles runtime conditions)
	taskJob, err := c.buildTaskJob(data)
	if err != nil {
		return fmt.Errorf("failed to build task job: %w", err)
	}
	if err := c.jobManager.AddJob(taskJob); err != nil {
		return fmt.Errorf("failed to add task job: %w", err)
	}

	// Build add_reaction job only if ai-reaction is configured
	if data.AIReaction != "" {
		addReactionJob, err := c.buildAddReactionJob(data)
		if err != nil {
			return fmt.Errorf("failed to build add_reaction job: %w", err)
		}
		if err := c.jobManager.AddJob(addReactionJob); err != nil {
			return fmt.Errorf("failed to add add_reaction job: %w", err)
		}
	}

	// Build main workflow job
	mainJob, err := c.buildMainJob(data, jobName)
	if err != nil {
		return fmt.Errorf("failed to build main job: %w", err)
	}
	if err := c.jobManager.AddJob(mainJob); err != nil {
		return fmt.Errorf("failed to add main job: %w", err)
	}

	// Build create_issue job if output.issue is configured
	if data.Output != nil && data.Output.Issue != nil {
		createIssueJob, err := c.buildCreateOutputIssueJob(data, jobName)
		if err != nil {
			return fmt.Errorf("failed to build create_issue job: %w", err)
		}
		if err := c.jobManager.AddJob(createIssueJob); err != nil {
			return fmt.Errorf("failed to add create_issue job: %w", err)
		}
	}

	// Build create_issue_comment job if output.comment is configured
	if data.Output != nil && data.Output.Comment != nil {
		createCommentJob, err := c.buildCreateOutputCommentJob(data, jobName)
		if err != nil {
			return fmt.Errorf("failed to build create_issue_comment job: %w", err)
		}
		if err := c.jobManager.AddJob(createCommentJob); err != nil {
			return fmt.Errorf("failed to add create_issue_comment job: %w", err)
		}
	}

	// Build create_pull_request job if output.pull-request is configured
	if data.Output != nil && data.Output.PullRequest != nil {
		createPullRequestJob, err := c.buildCreateOutputPullRequestJob(data, jobName)
		if err != nil {
			return fmt.Errorf("failed to build create_pull_request job: %w", err)
		}
		if err := c.jobManager.AddJob(createPullRequestJob); err != nil {
			return fmt.Errorf("failed to add create_pull_request job: %w", err)
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
	var steps []string

	// Add shallow checkout step to access shared actions
	steps = append(steps, "      - uses: actions/checkout@v4\n")
	steps = append(steps, "        with:\n")
	steps = append(steps, "          sparse-checkout: .github\n")
	steps = append(steps, "          fetch-depth: 1\n")

	// Add team member check for alias workflows, but only when triggered by alias mention
	if data.Alias != "" {
		// Build condition that only applies to alias mentions in comment-related events
		aliasCondition := buildAliasOnlyCondition(data.Alias)
		aliasConditionStr := aliasCondition.Render()

		// Build the validation condition using expression nodes
		// Since the check-team-member step is gated by alias condition, we check if it ran and returned 'false'
		// This avoids running validation when the step didn't run at all (non-alias triggers)
		validationCondition := BuildEquals(
			BuildPropertyAccess("steps.check-team-member.outputs.is_team_member"),
			BuildStringLiteral("false"),
		)
		validationConditionStr := validationCondition.Render()

		steps = append(steps, "      - name: Check team membership for alias workflow\n")
		steps = append(steps, "        id: check-team-member\n")
		steps = append(steps, "        uses: ./.github/actions/check-team-member\n")
		steps = append(steps, fmt.Sprintf("        if: %s\n", aliasConditionStr))
		steps = append(steps, "      - name: Validate team membership\n")
		steps = append(steps, fmt.Sprintf("        if: %s\n", validationConditionStr))
		steps = append(steps, "        run: |\n")
		steps = append(steps, "          echo \"❌ Access denied: Only team members can trigger alias workflows\"\n")
		steps = append(steps, "          echo \"User ${{ github.actor }} is not a team member\"\n")
		steps = append(steps, "          exit 1\n")
	}

	// Use shared compute-text action only if needed
	if data.NeedsTextOutput {
		steps = append(steps, "      - name: Compute current body text\n")
		steps = append(steps, "        id: compute-text\n")
		steps = append(steps, "        uses: ./.github/actions/compute-text\n")
	}

	// Set up outputs
	outputs := map[string]string{}
	if data.NeedsTextOutput {
		outputs["text"] = "${{ steps.compute-text.outputs.text }}"
	}

	job := &Job{
		Name:        "task",
		If:          data.If, // Use the existing condition (which may include alias checks)
		RunsOn:      "runs-on: ubuntu-latest",
		Permissions: "permissions:\n      contents: read", // Read permission for checkout
		Steps:       steps,
		Outputs:     outputs,
	}

	return job, nil
}

// buildAddReactionJob creates the add_reaction job
func (c *Compiler) buildAddReactionJob(data *WorkflowData) (*Job, error) {
	reactionCondition := buildReactionCondition()

	var steps []string
	steps = append(steps, "      - uses: actions/checkout@v4\n")
	steps = append(steps, "        with:\n")
	steps = append(steps, "          sparse-checkout: .github\n")
	steps = append(steps, fmt.Sprintf("      - name: Add %s reaction to the triggering item\n", data.AIReaction))
	steps = append(steps, "        id: react\n")
	steps = append(steps, "        uses: ./.github/actions/reaction\n")
	steps = append(steps, "        with:\n")
	steps = append(steps, "          github-token: ${{ secrets.GITHUB_TOKEN }}\n")
	steps = append(steps, "          mode: add\n")
	steps = append(steps, fmt.Sprintf("          reaction: %s\n", data.AIReaction))

	outputs := map[string]string{
		"reaction_id": "${{ steps.react.outputs.reaction-id }}",
	}

	job := &Job{
		Name:        "add_reaction",
		If:          fmt.Sprintf("if: %s", reactionCondition.Render()),
		RunsOn:      "runs-on: ubuntu-latest",
		Permissions: "permissions:\n      contents: write # Read .github\n      issues: write\n      pull-requests: write",
		Steps:       steps,
		Outputs:     outputs,
		Depends:     []string{"task"}, // Depend on the task job
	}

	return job, nil
}

// buildCreateOutputIssueJob creates the create_issue job
func (c *Compiler) buildCreateOutputIssueJob(data *WorkflowData, mainJobName string) (*Job, error) {
	if data.Output == nil || data.Output.Issue == nil {
		return nil, fmt.Errorf("output.issue configuration is required")
	}

	var steps []string
	steps = append(steps, "      - name: Create Output Issue\n")
	steps = append(steps, "        id: create_issue\n")
	steps = append(steps, "        uses: actions/github-script@v7\n")

	// Add environment variables
	steps = append(steps, "        env:\n")
	// Pass the agent output content from the main job
	steps = append(steps, fmt.Sprintf("          GITHUB_AW_AGENT_OUTPUT: ${{ needs.%s.outputs.output }}\n", mainJobName))
	if data.Output.Issue.TitlePrefix != "" {
		steps = append(steps, fmt.Sprintf("          GITHUB_AW_ISSUE_TITLE_PREFIX: %q\n", data.Output.Issue.TitlePrefix))
	}
	if len(data.Output.Issue.Labels) > 0 {
		labelsStr := strings.Join(data.Output.Issue.Labels, ",")
		steps = append(steps, fmt.Sprintf("          GITHUB_AW_ISSUE_LABELS: %q\n", labelsStr))
	}

	steps = append(steps, "        with:\n")
	steps = append(steps, "          script: |\n")

	// Add each line of the script with proper indentation
	scriptLines := strings.Split(createIssueScript, "\n")
	for _, line := range scriptLines {
		if strings.TrimSpace(line) == "" {
			steps = append(steps, "\n")
		} else {
			steps = append(steps, fmt.Sprintf("            %s\n", line))
		}
	}

	// Create outputs for the job
	outputs := map[string]string{
		"issue_number": "${{ steps.create_issue.outputs.issue_number }}",
		"issue_url":    "${{ steps.create_issue.outputs.issue_url }}",
	}

	job := &Job{
		Name:           "create_issue",
		If:             "", // No conditional execution
		RunsOn:         "runs-on: ubuntu-latest",
		Permissions:    "permissions:\n      contents: read\n      issues: write",
		TimeoutMinutes: 10, // 10-minute timeout as required
		Steps:          steps,
		Outputs:        outputs,
		Depends:        []string{mainJobName}, // Depend on the main workflow job
	}

	return job, nil
}

// buildCreateOutputCommentJob creates the create_issue_comment job
func (c *Compiler) buildCreateOutputCommentJob(data *WorkflowData, mainJobName string) (*Job, error) {
	if data.Output == nil || data.Output.Comment == nil {
		return nil, fmt.Errorf("output.comment configuration is required")
	}

	var steps []string
	steps = append(steps, "      - name: Create Output Comment\n")
	steps = append(steps, "        id: create_comment\n")
	steps = append(steps, "        uses: actions/github-script@v7\n")

	// Add environment variables
	steps = append(steps, "        env:\n")
	// Pass the agent output content from the main job
	steps = append(steps, fmt.Sprintf("          GITHUB_AW_AGENT_OUTPUT: ${{ needs.%s.outputs.output }}\n", mainJobName))

	steps = append(steps, "        with:\n")
	steps = append(steps, "          script: |\n")

	// Add each line of the script with proper indentation
	scriptLines := strings.Split(createCommentScript, "\n")
	for _, line := range scriptLines {
		if strings.TrimSpace(line) == "" {
			steps = append(steps, "\n")
		} else {
			steps = append(steps, fmt.Sprintf("            %s\n", line))
		}
	}

	// Create outputs for the job
	outputs := map[string]string{
		"comment_id":  "${{ steps.create_comment.outputs.comment_id }}",
		"comment_url": "${{ steps.create_comment.outputs.comment_url }}",
	}

	job := &Job{
		Name:           "create_issue_comment",
		If:             "if: github.event.issue.number || github.event.pull_request.number", // Only run in issue or PR context
		RunsOn:         "runs-on: ubuntu-latest",
		Permissions:    "permissions:\n      contents: read\n      issues: write\n      pull-requests: write",
		TimeoutMinutes: 10, // 10-minute timeout as required
		Steps:          steps,
		Outputs:        outputs,
		Depends:        []string{mainJobName}, // Depend on the main workflow job
	}

	return job, nil
}

// buildCreateOutputPullRequestJob creates the create_pull_request job
func (c *Compiler) buildCreateOutputPullRequestJob(data *WorkflowData, mainJobName string) (*Job, error) {
	if data.Output == nil || data.Output.PullRequest == nil {
		return nil, fmt.Errorf("output.pull-request configuration is required")
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
	steps = append(steps, "        uses: actions/checkout@v4\n")
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
	if data.Output.PullRequest.TitlePrefix != "" {
		steps = append(steps, fmt.Sprintf("          GITHUB_AW_PR_TITLE_PREFIX: %q\n", data.Output.PullRequest.TitlePrefix))
	}
	if len(data.Output.PullRequest.Labels) > 0 {
		labelsStr := strings.Join(data.Output.PullRequest.Labels, ",")
		steps = append(steps, fmt.Sprintf("          GITHUB_AW_PR_LABELS: %q\n", labelsStr))
	}

	steps = append(steps, "        with:\n")
	steps = append(steps, "          script: |\n")

	// Add each line of the script with proper indentation
	scriptLines := strings.Split(createPullRequestScript, "\n")
	for _, line := range scriptLines {
		if strings.TrimSpace(line) == "" {
			steps = append(steps, "\n")
		} else {
			steps = append(steps, fmt.Sprintf("            %s\n", line))
		}
	}

	// Create outputs for the job
	outputs := map[string]string{
		"pull_request_number": "${{ steps.create_pull_request.outputs.pull_request_number }}",
		"pull_request_url":    "${{ steps.create_pull_request.outputs.pull_request_url }}",
		"branch_name":         "${{ steps.create_pull_request.outputs.branch_name }}",
	}

	job := &Job{
		Name:           "create_pull_request",
		If:             "", // No conditional execution
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
func (c *Compiler) buildMainJob(data *WorkflowData, jobName string) (*Job, error) {
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

	job := &Job{
		Name:        jobName,
		If:          "", // Remove the If condition since task job handles alias checks
		RunsOn:      c.indentYAMLLines(data.RunsOn, "    "),
		Permissions: c.indentYAMLLines(data.Permissions, "    "),
		Steps:       steps,
		Depends:     []string{"task"}, // Depend on the task job
		Outputs: map[string]string{
			"output": "${{ steps.collect_output.outputs.output }}",
		},
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
	yaml.WriteString(fmt.Sprintf("          WORKFLOW_NAME=\"%s\"\n", workflowName))

	// Add stop-time check
	if data.StopTime != "" {
		yaml.WriteString("          \n")
		yaml.WriteString("          # Check stop-time limit\n")
		yaml.WriteString(fmt.Sprintf("          STOP_TIME=\"%s\"\n", data.StopTime))
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
	for toolName, toolValue := range tools {
		// Standard MCP tools
		if toolName == "github" {
			mcpTools = append(mcpTools, toolName)
		} else if mcpConfig, ok := toolValue.(map[string]any); ok {
			// Check if it's explicitly marked as MCP type in the new format
			if hasMcp, _ := hasMCPConfig(mcpConfig); hasMcp {
				mcpTools = append(mcpTools, toolName)
			}
		}
	}

	// Sort MCP tools to ensure stable code generation
	sort.Strings(mcpTools)

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
	githubDockerImageVersion := "sha-45e90ae" // Default Docker image version
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

// writeSharedAction writes a shared action file, creating directories as needed and updating only if content differs
func (c *Compiler) writeSharedAction(markdownPath string, actionPath string, content string, actionName string) error {
	// Get git root to write action relative to it, fallback to markdown directory for tests
	gitRoot := filepath.Dir(markdownPath)
	for {
		if _, err := os.Stat(filepath.Join(gitRoot, ".git")); err == nil {
			break
		}
		parent := filepath.Dir(gitRoot)
		if parent == gitRoot {
			// Reached filesystem root without finding .git - use markdown directory as fallback
			gitRoot = filepath.Dir(markdownPath)
			break
		}
		gitRoot = parent
	}

	actionsDir := filepath.Join(gitRoot, ".github", "actions", actionPath)
	actionFile := filepath.Join(actionsDir, "action.yml")

	// Create directory if it doesn't exist
	if err := os.MkdirAll(actionsDir, 0755); err != nil {
		return fmt.Errorf("failed to create actions directory: %w", err)
	}

	// Write the action file if it doesn't exist or is different
	if _, err := os.Stat(actionFile); os.IsNotExist(err) {
		if c.verbose {
			fmt.Println(console.FormatInfoMessage(fmt.Sprintf("Creating shared %s action: %s", actionName, actionFile)))
		}
		if err := os.WriteFile(actionFile, []byte(content), 0644); err != nil {
			return fmt.Errorf("failed to write %s action: %w", actionName, err)
		}
		// Track the created file
		if c.fileTracker != nil {
			c.fileTracker.TrackCreated(actionFile)
		}
	} else {
		// Check if the content is different and update if needed
		existing, err := os.ReadFile(actionFile)
		if err != nil {
			return fmt.Errorf("failed to read existing action file: %w", err)
		}
		if string(existing) != content {
			if c.verbose {
				fmt.Println(console.FormatInfoMessage(fmt.Sprintf("Updating shared %s action: %s", actionName, actionFile)))
			}
			if err := os.WriteFile(actionFile, []byte(content), 0644); err != nil {
				return fmt.Errorf("failed to update %s action: %w", actionName, err)
			}
			// Track the updated file
			if c.fileTracker != nil {
				c.fileTracker.TrackCreated(actionFile)
			}
		}
	}

	return nil
}

// writeReactionAction writes the shared reaction action if ai-reaction is used
func (c *Compiler) writeReactionAction(markdownPath string) error {
	return c.writeSharedAction(markdownPath, "reaction", reactionActionTemplate, "reaction")
}

// writeComputeTextAction writes the shared compute-text action
func (c *Compiler) writeComputeTextAction(markdownPath string) error {
	return c.writeSharedAction(markdownPath, "compute-text", computeTextActionTemplate, "compute-text")
}

// writeCheckTeamMemberAction writes the shared check-team-member action
func (c *Compiler) writeCheckTeamMemberAction(markdownPath string) error {
	return c.writeSharedAction(markdownPath, "check-team-member", checkTeamMemberTemplate, "check-team-member")
}

// generateMainJobSteps generates the steps section for the main job
func (c *Compiler) generateMainJobSteps(yaml *strings.Builder, data *WorkflowData) {
	// Add custom steps or default checkout step
	if data.CustomSteps != "" {
		// Remove "steps:" line and adjust indentation
		lines := strings.Split(data.CustomSteps, "\n")
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
	} else {
		yaml.WriteString("      - name: Checkout repository\n")
		yaml.WriteString("        uses: actions/checkout@v4\n")
	}

	// Add cache steps if cache configuration is present
	generateCacheSteps(yaml, data, c.verbose)

	// Add Node.js setup if the engine requires it
	engine, err := c.getAgenticEngine(data.AI)

	if err != nil {
		return
	}

	// Add engine-specific installation steps
	installSteps := engine.GetInstallationSteps(data.EngineConfig)
	for _, step := range installSteps {
		for _, line := range step {
			yaml.WriteString(line + "\n")
		}
	}

	// Generate output file setup step
	c.generateOutputFileSetup(yaml, data)

	// Add MCP setup
	c.generateMCPSetup(yaml, data.Tools, engine)

	// Add safety checks before executing agentic tools
	c.generateSafetyChecks(yaml, data)

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

	// add workflow_complete.txt
	c.generateWorkflowComplete(yaml)

	// Add output collection step
	c.generateOutputCollectionStep(yaml, data)

	// upload agent logs
	c.generateUploadAgentLogs(yaml, logFile, logFileFull)

	// Add git patch generation step after agentic execution
	c.generateGitPatchStep(yaml)

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
	yaml.WriteString(fmt.Sprintf("          name: %s.log\n", logFile))
	yaml.WriteString(fmt.Sprintf("          path: %s\n", logFileFull))
	yaml.WriteString("          if-no-files-found: warn\n")
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

func (c *Compiler) generatePrompt(yaml *strings.Builder, data *WorkflowData) {
	yaml.WriteString("      - name: Create prompt\n")
	yaml.WriteString("        env:\n")
	yaml.WriteString("          GITHUB_AW_OUTPUT: ${{ env.GITHUB_AW_OUTPUT }}\n")
	yaml.WriteString("        run: |\n")
	yaml.WriteString("          mkdir -p /tmp/aw-prompts\n")
	yaml.WriteString("          cat > /tmp/aw-prompts/prompt.txt << 'EOF'\n")

	// Add markdown content with proper indentation
	for _, line := range strings.Split(data.MarkdownContent, "\n") {
		yaml.WriteString("          " + line + "\n")
	}

	// Add output instructions for the agent
	yaml.WriteString("          \n")
	yaml.WriteString("          ---\n")
	yaml.WriteString("          \n")
	yaml.WriteString("          **IMPORTANT**: If you need to provide output that should be captured as a workflow output variable, write it to the file \"${{ env.GITHUB_AW_OUTPUT }}\". This file is available for you to write any output that should be exposed from this workflow. The content of this file will be made available as the 'output' workflow output.\n")
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

// extractOutputConfig extracts output configuration from frontmatter
func (c *Compiler) extractOutputConfig(frontmatter map[string]any) *OutputConfig {
	if output, exists := frontmatter["output"]; exists {
		if outputMap, ok := output.(map[string]any); ok {
			config := &OutputConfig{}

			// Parse issue configuration
			if issue, exists := outputMap["issue"]; exists {
				if issueMap, ok := issue.(map[string]any); ok {
					issueConfig := &IssueConfig{}

					// Parse title-prefix
					if titlePrefix, exists := issueMap["title-prefix"]; exists {
						if titlePrefixStr, ok := titlePrefix.(string); ok {
							issueConfig.TitlePrefix = titlePrefixStr
						}
					}

					// Parse labels
					if labels, exists := issueMap["labels"]; exists {
						if labelsArray, ok := labels.([]any); ok {
							var labelStrings []string
							for _, label := range labelsArray {
								if labelStr, ok := label.(string); ok {
									labelStrings = append(labelStrings, labelStr)
								}
							}
							issueConfig.Labels = labelStrings
						}
					}

					config.Issue = issueConfig
				}
			}

			// Parse comment configuration
			if comment, exists := outputMap["comment"]; exists {
				if _, ok := comment.(map[string]any); ok {
					// For now, CommentConfig is an empty struct
					config.Comment = &CommentConfig{}
				}
			}

			// Parse pull-request configuration
			if pullRequest, exists := outputMap["pull-request"]; exists {
				if pullRequestMap, ok := pullRequest.(map[string]any); ok {
					pullRequestConfig := &PullRequestConfig{}

					// Parse title-prefix
					if titlePrefix, exists := pullRequestMap["title-prefix"]; exists {
						if titlePrefixStr, ok := titlePrefix.(string); ok {
							pullRequestConfig.TitlePrefix = titlePrefixStr
						}
					}

					// Parse labels
					if labels, exists := pullRequestMap["labels"]; exists {
						if labelsArray, ok := labels.([]any); ok {
							var labelStrings []string
							for _, label := range labelsArray {
								if labelStr, ok := label.(string); ok {
									labelStrings = append(labelStrings, labelStr)
								}
							}
							pullRequestConfig.Labels = labelStrings
						}
					}

					config.PullRequest = pullRequestConfig
				}
			}

			return config
		}
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
			stepYAML.WriteString(fmt.Sprintf("        run: %s\n", runStr))
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

	executionConfig := engine.GetExecutionConfig(data.Name, logFile, data.EngineConfig)

	if executionConfig.Command != "" {
		// Command-based execution (e.g., Codex)
		yaml.WriteString(fmt.Sprintf("      - name: %s\n", executionConfig.StepName))
		yaml.WriteString("        run: |\n")

		// Split command into lines and indent them properly
		commandLines := strings.Split(executionConfig.Command, "\n")
		for _, line := range commandLines {
			yaml.WriteString("          " + line + "\n")
		}

		// Add environment variables
		if len(executionConfig.Environment) > 0 {
			yaml.WriteString("        env:\n")
			// Sort environment keys for consistent output
			envKeys := make([]string, 0, len(executionConfig.Environment))
			for key := range executionConfig.Environment {
				envKeys = append(envKeys, key)
			}
			sort.Strings(envKeys)

			for _, key := range envKeys {
				value := executionConfig.Environment[key]
				yaml.WriteString(fmt.Sprintf("          %s: %s\n", key, value))
			}
			// Add GITHUB_AW_OUTPUT environment variable
			yaml.WriteString("          GITHUB_AW_OUTPUT: ${{ env.GITHUB_AW_OUTPUT }}\n")
		} else {
			// Add GITHUB_AW_OUTPUT environment variable even if no other env vars
			yaml.WriteString("        env:\n")
			yaml.WriteString("          GITHUB_AW_OUTPUT: ${{ env.GITHUB_AW_OUTPUT }}\n")
		}
	} else if executionConfig.Action != "" {

		// Add the main action step
		yaml.WriteString(fmt.Sprintf("      - name: %s\n", executionConfig.StepName))
		yaml.WriteString("        id: agentic_execution\n")
		yaml.WriteString(fmt.Sprintf("        uses: %s\n", executionConfig.Action))
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
					yaml.WriteString(fmt.Sprintf("          %s: \"%s\"\n", key, data.AllowedTools))
				}
			} else if key == "timeout_minutes" {
				if data.TimeoutMinutes != "" {
					yaml.WriteString("          " + data.TimeoutMinutes + "\n")
				}
			} else if key == "max_turns" {
				if data.MaxTurns != "" {
					yaml.WriteString(fmt.Sprintf("          max_turns: %s\n", data.MaxTurns))
				}
			} else if value != "" {
				yaml.WriteString(fmt.Sprintf("          %s: %s\n", key, value))
			}
		}
		// Add environment section to pass GITHUB_AW_OUTPUT to the action
		yaml.WriteString("        env:\n")
		yaml.WriteString("          GITHUB_AW_OUTPUT: ${{ env.GITHUB_AW_OUTPUT }}\n")
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
	yaml.WriteString(fmt.Sprintf("              engine_id: \"%s\",\n", engineID))

	// Engine display name
	yaml.WriteString(fmt.Sprintf("              engine_name: \"%s\",\n", engine.GetDisplayName()))

	// Model information
	model := ""
	if data.EngineConfig != nil && data.EngineConfig.Model != "" {
		model = data.EngineConfig.Model
	}
	yaml.WriteString(fmt.Sprintf("              model: \"%s\",\n", model))

	// Version information
	version := ""
	if data.EngineConfig != nil && data.EngineConfig.Version != "" {
		version = data.EngineConfig.Version
	}
	yaml.WriteString(fmt.Sprintf("              version: \"%s\",\n", version))

	// Workflow information
	yaml.WriteString(fmt.Sprintf("              workflow_name: \"%s\",\n", data.Name))
	yaml.WriteString(fmt.Sprintf("              experimental: %t,\n", engine.IsExperimental()))
	yaml.WriteString(fmt.Sprintf("              supports_tools_whitelist: %t,\n", engine.SupportsToolsWhitelist()))
	yaml.WriteString(fmt.Sprintf("              supports_http_transport: %t,\n", engine.SupportsHTTPTransport()))

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

// generateOutputFileSetup generates a step that sets up the GITHUB_AW_OUTPUT environment variable
func (c *Compiler) generateOutputFileSetup(yaml *strings.Builder, data *WorkflowData) {
	yaml.WriteString("      - name: Setup agent output\n")
	yaml.WriteString("        id: setup_agent_output\n")
	yaml.WriteString("        uses: actions/github-script@v7\n")
	yaml.WriteString("        with:\n")
	yaml.WriteString("          script: |\n")
	yaml.WriteString("            const fs = require('fs');\n")
	yaml.WriteString("            const crypto = require('crypto');\n")
	yaml.WriteString("            // Generate a random filename for the output file\n")
	yaml.WriteString("            const randomId = crypto.randomBytes(8).toString('hex');\n")
	yaml.WriteString("            const outputFile = `/tmp/aw_output_${randomId}.txt`;\n")
	yaml.WriteString("            // Ensure the /tmp directory exists and create empty output file\n")
	yaml.WriteString("            fs.mkdirSync('/tmp', { recursive: true });\n")
	yaml.WriteString("            fs.writeFileSync(outputFile, '', { mode: 0o644 });\n")
	yaml.WriteString("            // Verify the file was created and is writable\n")
	yaml.WriteString("            if (!fs.existsSync(outputFile)) {\n")
	yaml.WriteString("              throw new Error(`Failed to create output file: ${outputFile}`);\n")
	yaml.WriteString("            }\n")
	yaml.WriteString("            // Set the environment variable for subsequent steps\n")
	yaml.WriteString("            core.exportVariable('GITHUB_AW_OUTPUT', outputFile);\n")
	yaml.WriteString("            console.log('Created agentic output file:', outputFile);\n")
	yaml.WriteString("            // Also set as step output for reference\n")
	yaml.WriteString("            core.setOutput('output_file', outputFile);\n")
}

// generateOutputCollectionStep generates a step that reads the output file and sets it as a GitHub Actions output
func (c *Compiler) generateOutputCollectionStep(yaml *strings.Builder, data *WorkflowData) {
	yaml.WriteString("      - name: Collect agent output\n")
	yaml.WriteString("        id: collect_output\n")
	yaml.WriteString("        uses: actions/github-script@v7\n")
	yaml.WriteString("        with:\n")
	yaml.WriteString("          script: |\n")
	yaml.WriteString("            const fs = require('fs');\n")
	yaml.WriteString("            \n")
	yaml.WriteString("            // Sanitization function for adversarial LLM outputs\n")
	yaml.WriteString("            function sanitizeContent(content) {\n")
	yaml.WriteString("              if (!content || typeof content !== 'string') {\n")
	yaml.WriteString("                return '';\n")
	yaml.WriteString("              }\n")
	yaml.WriteString("              \n")
	yaml.WriteString("              // Remove control characters (except newlines and tabs)\n")
	yaml.WriteString("              let sanitized = content.replace(/[\\x00-\\x08\\x0B\\x0C\\x0E-\\x1F\\x7F]/g, '');\n")
	yaml.WriteString("              \n")
	yaml.WriteString("              // Limit total length to prevent DoS (0.5MB max)\n")
	yaml.WriteString("              const maxLength = 524288;\n")
	yaml.WriteString("              if (sanitized.length > maxLength) {\n")
	yaml.WriteString("                sanitized = sanitized.substring(0, maxLength) + '\\n[Content truncated due to length]';\n")
	yaml.WriteString("              }\n")
	yaml.WriteString("              \n")
	yaml.WriteString("              // Limit number of lines to prevent log flooding (65k max)\n")
	yaml.WriteString("              const lines = sanitized.split('\\n');\n")
	yaml.WriteString("              const maxLines = 65000;\n")
	yaml.WriteString("              if (lines.length > maxLines) {\n")
	yaml.WriteString("                sanitized = lines.slice(0, maxLines).join('\\n') + '\\n[Content truncated due to line count]';\n")
	yaml.WriteString("              }\n")
	yaml.WriteString("              \n")
	yaml.WriteString("              \n")
	yaml.WriteString("              // Remove ANSI escape sequences\n")
	yaml.WriteString("              sanitized = sanitized.replace(/\\x1b\\[[0-9;]*[mGKH]/g, '');\n")
	yaml.WriteString("              \n")
	yaml.WriteString("              // Trim excessive whitespace\n")
	yaml.WriteString("              return sanitized.trim();\n")
	yaml.WriteString("            }\n")
	yaml.WriteString("            \n")
	yaml.WriteString("            const outputFile = process.env.GITHUB_AW_OUTPUT;\n")
	yaml.WriteString("            if (!outputFile) {\n")
	yaml.WriteString("              console.log('GITHUB_AW_OUTPUT not set, no output to collect');\n")
	yaml.WriteString("              core.setOutput('output', '');\n")
	yaml.WriteString("              return;\n")
	yaml.WriteString("            }\n")
	yaml.WriteString("            if (!fs.existsSync(outputFile)) {\n")
	yaml.WriteString("              console.log('Output file does not exist:', outputFile);\n")
	yaml.WriteString("              core.setOutput('output', '');\n")
	yaml.WriteString("              return;\n")
	yaml.WriteString("            }\n")
	yaml.WriteString("            const outputContent = fs.readFileSync(outputFile, 'utf8');\n")
	yaml.WriteString("            if (outputContent.trim() === '') {\n")
	yaml.WriteString("              console.log('Output file is empty');\n")
	yaml.WriteString("              core.setOutput('output', '');\n")
	yaml.WriteString("            } else {\n")
	yaml.WriteString("              const sanitizedContent = sanitizeContent(outputContent);\n")
	yaml.WriteString("              console.log('Collected agentic output (sanitized):', sanitizedContent.substring(0, 200) + (sanitizedContent.length > 200 ? '...' : ''));\n")
	yaml.WriteString("              core.setOutput('output', sanitizedContent);\n")
	yaml.WriteString("            }\n")

	yaml.WriteString("      - name: Print agent output to step summary\n")
	yaml.WriteString("        env:\n")
	yaml.WriteString("          GITHUB_AW_OUTPUT: ${{ env.GITHUB_AW_OUTPUT }}\n")
	yaml.WriteString("        run: |\n")
	yaml.WriteString("          echo \"## Agent Output\" >> $GITHUB_STEP_SUMMARY\n")
	yaml.WriteString("          echo \"\" >> $GITHUB_STEP_SUMMARY\n")
	yaml.WriteString("          echo '``````markdown' >> $GITHUB_STEP_SUMMARY\n")
	yaml.WriteString("          cat ${{ env.GITHUB_AW_OUTPUT }} >> $GITHUB_STEP_SUMMARY\n")
	yaml.WriteString("          echo '``````' >> $GITHUB_STEP_SUMMARY\n")
	yaml.WriteString("      - name: Upload agentic output file\n")
	yaml.WriteString("        if: always() && steps.collect_output.outputs.output != ''\n")
	yaml.WriteString("        uses: actions/upload-artifact@v4\n")
	yaml.WriteString("        with:\n")
	yaml.WriteString("          name: aw_output.txt\n")
	yaml.WriteString("          path: ${{ env.GITHUB_AW_OUTPUT }}\n")
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
	// Check if max-turns is specified in the frontmatter
	_, hasMaxTurns := frontmatter["max-turns"]
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
