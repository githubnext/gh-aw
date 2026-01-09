package workflow

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/githubnext/gh-aw/pkg/console"
	"github.com/githubnext/gh-aw/pkg/logger"
	"github.com/githubnext/gh-aw/pkg/parser"
	"github.com/goccy/go-yaml"
)

var detectionLog = logger.New("workflow:detection")
var orchestratorLog = logger.New("workflow:compiler_orchestrator")

func (c *Compiler) ParseWorkflowFile(markdownPath string) (*WorkflowData, error) {
	orchestratorLog.Printf("Starting workflow file parsing: %s", markdownPath)
	log.Printf("Reading file: %s", markdownPath)

	// Clean the path to prevent path traversal issues (gosec G304)
	// filepath.Clean removes ".." and other problematic path elements
	cleanPath := filepath.Clean(markdownPath)

	// Read the file
	content, err := os.ReadFile(cleanPath)
	if err != nil {
		orchestratorLog.Printf("Failed to read file: %s, error: %v", cleanPath, err)
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	log.Printf("File size: %d bytes", len(content))

	// Parse frontmatter and markdown
	orchestratorLog.Printf("Parsing frontmatter from file: %s", cleanPath)
	result, err := parser.ExtractFrontmatterFromContent(string(content))
	if err != nil {
		orchestratorLog.Printf("Frontmatter extraction failed: %v", err)
		// Use FrontmatterStart from result if available, otherwise default to line 2 (after opening ---)
		frontmatterStart := 2
		if result != nil && result.FrontmatterStart > 0 {
			frontmatterStart = result.FrontmatterStart
		}
		return nil, c.createFrontmatterError(cleanPath, string(content), err, frontmatterStart)
	}

	if len(result.Frontmatter) == 0 {
		orchestratorLog.Print("No frontmatter found in file")
		return nil, fmt.Errorf("no frontmatter found")
	}

	// Preprocess schedule fields to convert human-friendly format to cron expressions
	if err := c.preprocessScheduleFields(result.Frontmatter, cleanPath, string(content)); err != nil {
		orchestratorLog.Printf("Schedule preprocessing failed: %v", err)
		return nil, err
	}

	// Create a copy of frontmatter without internal markers for schema validation
	// Keep the original frontmatter with markers for YAML generation
	frontmatterForValidation := c.copyFrontmatterWithoutInternalMarkers(result.Frontmatter)

	// Check if "on" field is missing - if so, treat as a shared/imported workflow
	_, hasOnField := frontmatterForValidation["on"]
	if !hasOnField {
		detectionLog.Printf("No 'on' field detected - treating as shared agentic workflow")

		// Validate as an included/shared workflow (uses included_file_schema)
		if err := parser.ValidateIncludedFileFrontmatterWithSchemaAndLocation(frontmatterForValidation, cleanPath); err != nil {
			orchestratorLog.Printf("Shared workflow validation failed: %v", err)
			return nil, err
		}

		// Return a special error to signal that this is a shared workflow
		// and compilation should be skipped with an info message
		// Note: Markdown content is optional for shared workflows (they may be just config)
		return nil, &SharedWorkflowError{
			Path: cleanPath,
		}
	}

	// For main workflows (with 'on' field), markdown content is required
	if result.Markdown == "" {
		orchestratorLog.Print("No markdown content found for main workflow")
		return nil, fmt.Errorf("no markdown content found")
	}

	// Validate main workflow frontmatter contains only expected entries
	orchestratorLog.Printf("Validating main workflow frontmatter schema")
	if err := parser.ValidateMainWorkflowFrontmatterWithSchemaAndLocation(frontmatterForValidation, cleanPath); err != nil {
		orchestratorLog.Printf("Main workflow frontmatter validation failed: %v", err)
		return nil, err
	}

	// Validate event filter mutual exclusivity (branches/branches-ignore, paths/paths-ignore)
	if err := ValidateEventFilters(frontmatterForValidation); err != nil {
		orchestratorLog.Printf("Event filter validation failed: %v", err)
		return nil, err
	}

	log.Printf("Frontmatter: %d chars, Markdown: %d chars", len(result.Frontmatter), len(result.Markdown))

	markdownDir := filepath.Dir(cleanPath)

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

	// Extract sandbox configuration from frontmatter
	sandboxConfig := c.extractSandboxConfig(result.Frontmatter)

	// Save the initial strict mode state to restore it after this workflow is processed
	// This ensures that strict mode from one workflow doesn't affect other workflows
	initialStrictMode := c.strictMode

	// Check strict mode in frontmatter
	// Priority: CLI flag > frontmatter > schema default (true)
	if !c.strictMode {
		// CLI flag not set, check frontmatter
		if strictValue, exists := result.Frontmatter["strict"]; exists {
			// Frontmatter explicitly sets strict mode
			if strictBool, ok := strictValue.(bool); ok {
				c.strictMode = strictBool
			}
		} else {
			// Neither CLI nor frontmatter set - use schema default (true)
			c.strictMode = true
		}
	}

	// Perform strict mode validations
	orchestratorLog.Printf("Performing strict mode validation (strict=%v)", c.strictMode)
	if err := c.validateStrictMode(result.Frontmatter, networkPermissions); err != nil {
		orchestratorLog.Printf("Strict mode validation failed: %v", err)
		// Restore strict mode before returning error
		c.strictMode = initialStrictMode
		return nil, err
	}

	// Restore the initial strict mode state after validation
	// This ensures strict mode doesn't leak to other workflows being compiled
	c.strictMode = initialStrictMode

	// Validate that @include/@import directives are not used inside template regions
	if err := validateNoIncludesInTemplateRegions(result.Markdown); err != nil {
		orchestratorLog.Printf("Template region validation failed: %v", err)
		return nil, fmt.Errorf("template region validation failed: %w", err)
	}

	// Override with command line AI engine setting if provided
	if c.engineOverride != "" {
		originalEngineSetting := engineSetting
		if originalEngineSetting != "" && originalEngineSetting != c.engineOverride {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Command line --engine %s overrides markdown file engine: %s", c.engineOverride, originalEngineSetting)))
			c.IncrementWarningCount()
		}
		engineSetting = c.engineOverride
	}

	// Process imports from frontmatter first (before @include directives)
	orchestratorLog.Printf("Processing imports from frontmatter")
	importCache := c.getSharedImportCache()
	// Pass the full file content for accurate line/column error reporting
	importsResult, err := parser.ProcessImportsFromFrontmatterWithSource(result.Frontmatter, markdownDir, importCache, cleanPath, string(content))
	if err != nil {
		orchestratorLog.Printf("Import processing failed: %v", err)
		return nil, err // Error is already formatted with source location
	}

	// Merge network permissions from imports with top-level network permissions
	if importsResult.MergedNetwork != "" {
		orchestratorLog.Printf("Merging network permissions from imports")
		networkPermissions, err = c.MergeNetworkPermissions(networkPermissions, importsResult.MergedNetwork)
		if err != nil {
			orchestratorLog.Printf("Network permissions merge failed: %v", err)
			return nil, fmt.Errorf("failed to merge network permissions: %w", err)
		}
	}

	// Validate permissions from imports against top-level permissions
	// Extract top-level permissions first
	topLevelPermissions := c.extractPermissions(result.Frontmatter)
	if importsResult.MergedPermissions != "" {
		orchestratorLog.Printf("Validating included permissions")
		if err := c.ValidateIncludedPermissions(topLevelPermissions, importsResult.MergedPermissions); err != nil {
			orchestratorLog.Printf("Included permissions validation failed: %v", err)
			return nil, fmt.Errorf("permission validation failed: %w", err)
		}
	}

	// Process @include directives to extract engine configurations and check for conflicts
	orchestratorLog.Printf("Expanding includes for engine configurations")
	includedEngines, err := parser.ExpandIncludesForEngines(result.Markdown, markdownDir)
	if err != nil {
		orchestratorLog.Printf("Failed to expand includes for engines: %v", err)
		return nil, fmt.Errorf("failed to expand includes for engines: %w", err)
	}

	// Combine imported engines with included engines
	allEngines := append(importsResult.MergedEngines, includedEngines...)

	// Validate that only one engine field exists across all files
	orchestratorLog.Printf("Validating single engine specification")
	finalEngineSetting, err := c.validateSingleEngineSpecification(engineSetting, allEngines)
	if err != nil {
		orchestratorLog.Printf("Engine specification validation failed: %v", err)
		return nil, err
	}
	if finalEngineSetting != "" {
		engineSetting = finalEngineSetting
	}

	// If engineConfig is nil (engine was in an included file), extract it from the included engine JSON
	if engineConfig == nil && len(allEngines) > 0 {
		orchestratorLog.Printf("Extracting engine config from included file")
		extractedConfig, err := c.extractEngineConfigFromJSON(allEngines[0])
		if err != nil {
			orchestratorLog.Printf("Failed to extract engine config: %v", err)
			return nil, fmt.Errorf("failed to extract engine config from included file: %w", err)
		}
		engineConfig = extractedConfig
	}

	// Apply the default AI engine setting if not specified
	if engineSetting == "" {
		defaultEngine := c.engineRegistry.GetDefaultEngine()
		engineSetting = defaultEngine.GetID()
		log.Printf("No 'engine:' setting found, defaulting to: %s", engineSetting)
		// Create a default EngineConfig with the default engine ID if not already set
		if engineConfig == nil {
			engineConfig = &EngineConfig{ID: engineSetting}
		} else if engineConfig.ID == "" {
			engineConfig.ID = engineSetting
		}
	}

	// Validate the engine setting
	orchestratorLog.Printf("Validating engine setting: %s", engineSetting)
	if err := c.validateEngine(engineSetting); err != nil {
		orchestratorLog.Printf("Engine validation failed: %v", err)
		return nil, err
	}

	// Get the agentic engine instance
	agenticEngine, err := c.getAgenticEngine(engineSetting)
	if err != nil {
		orchestratorLog.Printf("Failed to get agentic engine: %v", err)
		return nil, err
	}

	log.Printf("AI engine: %s (%s)", agenticEngine.GetDisplayName(), engineSetting)
	if agenticEngine.IsExperimental() && c.verbose {
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Using experimental engine: %s", agenticEngine.GetDisplayName())))
		c.IncrementWarningCount()
	}

	// Enable firewall by default for copilot engine when network restrictions are present
	// (unless SRT sandbox is configured, since AWF and SRT are mutually exclusive)
	enableFirewallByDefaultForCopilot(engineSetting, networkPermissions, sandboxConfig)

	// Enable firewall by default for claude engine when network restrictions are present
	enableFirewallByDefaultForClaude(engineSetting, networkPermissions, sandboxConfig)

	// Re-evaluate strict mode for firewall and network validation
	// (it was restored after validateStrictMode but we need it again)
	initialStrictModeForFirewall := c.strictMode
	if !c.strictMode {
		// CLI flag not set, check frontmatter
		if strictValue, exists := result.Frontmatter["strict"]; exists {
			// Frontmatter explicitly sets strict mode
			if strictBool, ok := strictValue.(bool); ok {
				c.strictMode = strictBool
			}
		} else {
			// Neither CLI nor frontmatter set - use schema default (true)
			c.strictMode = true
		}
	}

	// Validate firewall is enabled in strict mode for copilot with network restrictions
	orchestratorLog.Printf("Validating strict firewall (strict=%v)", c.strictMode)
	if err := c.validateStrictFirewall(engineSetting, networkPermissions, sandboxConfig); err != nil {
		orchestratorLog.Printf("Strict firewall validation failed: %v", err)
		c.strictMode = initialStrictModeForFirewall
		return nil, err
	}

	// Check if the engine supports network restrictions when they are defined
	if err := c.checkNetworkSupport(agenticEngine, networkPermissions); err != nil {
		orchestratorLog.Printf("Network support check failed: %v", err)
		// Restore strict mode before returning error
		c.strictMode = initialStrictModeForFirewall
		return nil, err
	}

	// Restore the strict mode state after network check
	c.strictMode = initialStrictModeForFirewall

	log.Print("Processing tools and includes...")

	// Extract SafeOutputs configuration early so we can use it when applying default tools
	safeOutputs := c.extractSafeOutputsConfig(result.Frontmatter)

	// Extract SecretMasking configuration
	secretMasking := c.extractSecretMaskingConfig(result.Frontmatter)

	// Merge secret-masking from imports with top-level secret-masking
	if importsResult.MergedSecretMasking != "" {
		orchestratorLog.Printf("Merging secret-masking from imports")
		secretMasking, err = c.MergeSecretMasking(secretMasking, importsResult.MergedSecretMasking)
		if err != nil {
			orchestratorLog.Printf("Secret-masking merge failed: %v", err)
			return nil, fmt.Errorf("failed to merge secret-masking: %w", err)
		}
	}

	var tools map[string]any

	// Extract tools from the main file
	topTools := extractToolsFromFrontmatter(result.Frontmatter)

	// Extract mcp-servers from the main file and merge them into tools
	mcpServers := extractMCPServersFromFrontmatter(result.Frontmatter)

	// Process @include directives to extract additional tools
	orchestratorLog.Printf("Expanding includes for tools")
	includedTools, includedToolFiles, err := parser.ExpandIncludesWithManifest(result.Markdown, markdownDir, true)
	if err != nil {
		orchestratorLog.Printf("Failed to expand includes for tools: %v", err)
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
		orchestratorLog.Printf("Merging imported mcp-servers")
		// Parse and merge imported MCP servers
		mergedMCPServers, err := c.MergeMCPServers(mcpServers, importsResult.MergedMCPServers)
		if err != nil {
			orchestratorLog.Printf("MCP servers merge failed: %v", err)
			return nil, fmt.Errorf("failed to merge imported mcp-servers: %w", err)
		}
		allMCPServers = mergedMCPServers
	}

	// Merge tools including mcp-servers
	orchestratorLog.Printf("Merging tools and MCP servers")
	tools, err = c.mergeToolsAndMCPServers(topTools, allMCPServers, allIncludedTools)

	if err != nil {
		orchestratorLog.Printf("Tools merge failed: %v", err)
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
	orchestratorLog.Printf("Merging runtimes")
	runtimes, err := mergeRuntimes(topRuntimes, importsResult.MergedRuntimes)
	if err != nil {
		orchestratorLog.Printf("Runtimes merge failed: %v", err)
		return nil, fmt.Errorf("failed to merge runtimes: %w", err)
	}

	// Add MCP fetch server if needed (when web-fetch is requested but engine doesn't support it)
	tools, _ = AddMCPFetchServerIfNeeded(tools, agenticEngine)

	// Validate MCP configurations
	orchestratorLog.Printf("Validating MCP configurations")
	if err := ValidateMCPConfigs(tools); err != nil {
		orchestratorLog.Printf("MCP configuration validation failed: %v", err)
		return nil, err
	}

	// Validate HTTP transport support for the current engine
	if err := c.validateHTTPTransportSupport(tools, agenticEngine); err != nil {
		orchestratorLog.Printf("HTTP transport validation failed: %v", err)
		return nil, err
	}

	if !agenticEngine.SupportsToolsAllowlist() {
		// For engines that don't support tool allowlists (like codex), ignore tools section and provide warnings
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Using experimental %s support (engine: %s)", agenticEngine.GetDisplayName(), engineSetting)))
		c.IncrementWarningCount()
		if _, hasTools := result.Frontmatter["tools"]; hasTools {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("'tools' section ignored when using engine: %s (%s doesn't support MCP tool allow-listing)", engineSetting, agenticEngine.GetDisplayName())))
			c.IncrementWarningCount()
		}
		tools = map[string]any{}
		// For now, we'll add a basic github tool (always uses docker MCP)
		githubConfig := map[string]any{}

		tools["github"] = githubConfig
	}

	// Validate max-turns support for the current engine
	if err := c.validateMaxTurnsSupport(result.Frontmatter, agenticEngine); err != nil {
		return nil, err
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
	workflowName, err := parser.ExtractWorkflowNameFromMarkdown(cleanPath)
	if err != nil {
		return nil, fmt.Errorf("failed to extract workflow name: %w", err)
	}

	// Check if frontmatter specifies a custom name and use it instead
	frontmatterName := extractStringFromMap(result.Frontmatter, "name", nil)
	if frontmatterName != "" {
		workflowName = frontmatterName
	}

	log.Printf("Extracted workflow name: '%s'", workflowName)

	// Check if the markdown content uses the text output
	needsTextOutput := c.detectTextOutputUsage(markdownContent)

	// Extract and validate tracker-id
	trackerID, err := c.extractTrackerID(result.Frontmatter)
	if err != nil {
		return nil, err
	}

	// Build workflow data
	workflowData := &WorkflowData{
		Name:                workflowName,
		FrontmatterName:     frontmatterName,
		FrontmatterYAML:     strings.Join(result.FrontmatterLines, "\n"),
		Description:         c.extractDescription(result.Frontmatter),
		Source:              c.extractSource(result.Frontmatter),
		TrackerID:           trackerID,
		ImportedFiles:       importsResult.ImportedFiles,
		IncludedFiles:       allIncludedFiles,
		ImportInputs:        importsResult.ImportInputs,
		Tools:               tools,
		ParsedTools:         NewTools(tools),
		Runtimes:            runtimes,
		MarkdownContent:     markdownContent,
		AI:                  engineSetting,
		EngineConfig:        engineConfig,
		AgentFile:           importsResult.AgentFile,
		NetworkPermissions:  networkPermissions,
		SandboxConfig:       sandboxConfig,
		NeedsTextOutput:     needsTextOutput,
		SafetyPrompt:        safetyPrompt,
		ToolsTimeout:        toolsTimeout,
		ToolsStartupTimeout: toolsStartupTimeout,
		TrialMode:           c.trialMode,
		TrialLogicalRepo:    c.trialLogicalRepoSlug,
		GitHubToken:         extractStringFromMap(result.Frontmatter, "github-token", nil),
		StrictMode:          c.strictMode,
		SecretMasking:       secretMasking,
	}

	// Apply defaults to sandbox config
	workflowData.SandboxConfig = applySandboxDefaults(workflowData.SandboxConfig, engineConfig)

	// Use shared action cache and resolver from the compiler
	// This ensures cache is shared across all workflows during compilation
	actionCache, actionResolver := c.getSharedActionResolver()
	workflowData.ActionCache = actionCache
	workflowData.ActionResolver = actionResolver

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
	// Prefer timeout-minutes (new) over timeout_minutes (deprecated)
	workflowData.TimeoutMinutes = c.extractTopLevelYAMLSection(result.Frontmatter, "timeout-minutes")
	if workflowData.TimeoutMinutes == "" {
		workflowData.TimeoutMinutes = c.extractTopLevelYAMLSection(result.Frontmatter, "timeout_minutes")
		if workflowData.TimeoutMinutes != "" {
			// Emit deprecation warning
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage("Field 'timeout_minutes' is deprecated. Please use 'timeout-minutes' instead to follow GitHub Actions naming convention."))
		}
	}
	workflowData.CustomSteps = c.extractTopLevelYAMLSection(result.Frontmatter, "steps")

	// Merge imported steps if any
	if importsResult.MergedSteps != "" {
		// Parse imported steps from YAML array
		var importedSteps []any
		if err := yaml.Unmarshal([]byte(importsResult.MergedSteps), &importedSteps); err == nil {
			// Apply action pinning to imported steps
			importedSteps = ApplyActionPinsToSteps(importedSteps, workflowData)

			// If there are main workflow steps, parse and merge them
			if workflowData.CustomSteps != "" {
				// Parse main workflow steps (format: "steps:\n  - ...")
				var mainStepsWrapper map[string]any
				if err := yaml.Unmarshal([]byte(workflowData.CustomSteps), &mainStepsWrapper); err == nil {
					if mainStepsVal, hasSteps := mainStepsWrapper["steps"]; hasSteps {
						if mainSteps, ok := mainStepsVal.([]any); ok {
							// Apply action pinning to main steps
							mainSteps = ApplyActionPinsToSteps(mainSteps, workflowData)

							// Prepend imported steps to main steps
							allSteps := append(importedSteps, mainSteps...)
							// Convert back to YAML with "steps:" wrapper
							stepsWrapper := map[string]any{"steps": allSteps}
							stepsYAML, err := yaml.Marshal(stepsWrapper)
							if err == nil {
								// Remove quotes from uses values with version comments
								workflowData.CustomSteps = unquoteUsesWithComments(string(stepsYAML))
							}
						}
					}
				}
			} else {
				// Only imported steps exist, wrap in "steps:" format
				stepsWrapper := map[string]any{"steps": importedSteps}
				stepsYAML, err := yaml.Marshal(stepsWrapper)
				if err == nil {
					// Remove quotes from uses values with version comments
					workflowData.CustomSteps = unquoteUsesWithComments(string(stepsYAML))
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
					mainSteps = ApplyActionPinsToSteps(mainSteps, workflowData)

					// Convert back to YAML with "steps:" wrapper
					stepsWrapper := map[string]any{"steps": mainSteps}
					stepsYAML, err := yaml.Marshal(stepsWrapper)
					if err == nil {
						// Remove quotes from uses values with version comments
						workflowData.CustomSteps = unquoteUsesWithComments(string(stepsYAML))
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
					postSteps = ApplyActionPinsToSteps(postSteps, workflowData)

					// Convert back to YAML with "post-steps:" wrapper
					stepsWrapper := map[string]any{"post-steps": postSteps}
					stepsYAML, err := yaml.Marshal(stepsWrapper)
					if err == nil {
						// Remove quotes from uses values with version comments
						workflowData.PostSteps = unquoteUsesWithComments(string(stepsYAML))
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
	// Use the backward compatibility wrapper to avoid changing all call sites at once
	cacheMemoryConfig, err := c.extractCacheMemoryConfigFromMap(tools) // Use merged tools to support imports
	if err != nil {
		return nil, err
	}
	workflowData.CacheMemoryConfig = cacheMemoryConfig

	// Extract repo-memory config and check for errors
	toolsConfig, err := ParseToolsConfig(tools)
	if err != nil {
		return nil, err
	}
	repoMemoryConfig, err := c.extractRepoMemoryConfig(toolsConfig)
	if err != nil {
		return nil, err
	}
	workflowData.RepoMemoryConfig = repoMemoryConfig

	// Process stop-after configuration from the on: section
	err = c.processStopAfterConfiguration(result.Frontmatter, workflowData, cleanPath)
	if err != nil {
		return nil, err
	}

	// Process skip-if-match configuration from the on: section
	err = c.processSkipIfMatchConfiguration(result.Frontmatter, workflowData)
	if err != nil {
		return nil, err
	}

	// Process skip-if-no-match configuration from the on: section
	err = c.processSkipIfNoMatchConfiguration(result.Frontmatter, workflowData)
	if err != nil {
		return nil, err
	}

	// Process manual-approval configuration from the on: section
	err = c.processManualApprovalConfiguration(result.Frontmatter, workflowData)
	if err != nil {
		return nil, err
	}

	workflowData.Command, workflowData.CommandEvents = c.extractCommandConfig(result.Frontmatter)
	workflowData.Jobs = c.extractJobsFromFrontmatter(result.Frontmatter)
	workflowData.Roles = c.extractRoles(result.Frontmatter)
	workflowData.Bots = c.extractBots(result.Frontmatter)

	// Use the already extracted output configuration
	workflowData.SafeOutputs = safeOutputs

	// Extract safe-inputs configuration
	workflowData.SafeInputs = c.extractSafeInputsConfig(result.Frontmatter)

	// Merge safe-inputs from imports
	if len(importsResult.MergedSafeInputs) > 0 {
		workflowData.SafeInputs = c.mergeSafeInputs(workflowData.SafeInputs, importsResult.MergedSafeInputs)
	}

	// Extract safe-jobs from safe-outputs.jobs location
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

	// Merge app configuration from included safe-outputs configurations
	includedApp, err := c.mergeAppFromIncludedConfigs(workflowData.SafeOutputs, allSafeOutputsConfigs)
	if err != nil {
		return nil, fmt.Errorf("failed to merge app from includes: %w", err)
	}

	// Ensure SafeOutputs exists and populate the Jobs field with merged jobs
	if workflowData.SafeOutputs == nil && len(includedSafeJobs) > 0 {
		workflowData.SafeOutputs = &SafeOutputsConfig{}
	}
	// Always use the merged includedSafeJobs as it contains both main and imported jobs
	// The mergeSafeJobsFromIncludedConfigs function already handles conflict detection
	if workflowData.SafeOutputs != nil && len(includedSafeJobs) > 0 {
		workflowData.SafeOutputs.Jobs = includedSafeJobs
	}

	// Populate the App field if it's not set in the top-level workflow but is in an included config
	if workflowData.SafeOutputs != nil && workflowData.SafeOutputs.App == nil && includedApp != nil {
		workflowData.SafeOutputs.App = includedApp
	}

	// Merge safe-outputs types from imports (create-issue, add-comment, etc.)
	mergedSafeOutputs, err := c.MergeSafeOutputs(workflowData.SafeOutputs, allSafeOutputsConfigs)
	if err != nil {
		return nil, fmt.Errorf("failed to merge safe-outputs from imports: %w", err)
	}
	workflowData.SafeOutputs = mergedSafeOutputs

	// Parse the "on" section for command triggers, reactions, and other events
	err = c.parseOnSection(result.Frontmatter, workflowData, cleanPath)
	if err != nil {
		return nil, err
	}

	// Apply defaults
	if err := c.applyDefaults(workflowData, cleanPath); err != nil {
		return nil, err
	}

	// Apply pull request draft filter if specified
	c.applyPullRequestDraftFilter(workflowData, result.Frontmatter)

	// Apply pull request fork filter if specified
	c.applyPullRequestForkFilter(workflowData, result.Frontmatter)

	// Apply label filter if specified
	c.applyLabelFilter(workflowData, result.Frontmatter)

	orchestratorLog.Printf("Workflow file parsing completed successfully: %s", markdownPath)
	return workflowData, nil
}

// copyFrontmatterWithoutInternalMarkers creates a deep copy of frontmatter without internal marker fields
// This is used for schema validation while preserving markers in the original for YAML generation
func (c *Compiler) copyFrontmatterWithoutInternalMarkers(frontmatter map[string]any) map[string]any {
	// Create a shallow copy of the top level
	copy := make(map[string]any)
	for k, v := range frontmatter {
		if k == "on" {
			// Special handling for "on" field - need to deep copy and remove markers
			if onMap, ok := v.(map[string]any); ok {
				onCopy := make(map[string]any)
				for onKey, onValue := range onMap {
					if onKey == "issues" || onKey == "pull_request" || onKey == "discussion" {
						// Deep copy the section and remove marker
						if sectionMap, ok := onValue.(map[string]any); ok {
							sectionCopy := make(map[string]any)
							for sectionKey, sectionValue := range sectionMap {
								if sectionKey != "__gh_aw_native_label_filter__" {
									sectionCopy[sectionKey] = sectionValue
								}
							}
							onCopy[onKey] = sectionCopy
						} else {
							onCopy[onKey] = onValue
						}
					} else {
						onCopy[onKey] = onValue
					}
				}
				copy[k] = onCopy
			} else {
				copy[k] = v
			}
		} else {
			copy[k] = v
		}
	}
	return copy
}

// detectTextOutputUsage checks if the markdown content uses ${{ needs.activation.outputs.text }}
func (c *Compiler) detectTextOutputUsage(markdownContent string) bool {
	// Check for the specific GitHub Actions expression
	hasUsage := strings.Contains(markdownContent, "${{ needs.activation.outputs.text }}")
	detectionLog.Printf("Detected usage of activation.outputs.text: %v", hasUsage)
	return hasUsage
}
