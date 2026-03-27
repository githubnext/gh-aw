package workflow

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/parser"
)

var orchestratorToolsLog = logger.New("workflow:compiler_orchestrator_tools")

// toolsProcessingResult holds the results of tools and markdown processing
type toolsProcessingResult struct {
	tools                 map[string]any
	runtimes              map[string]any
	apmDependencies       *APMDependenciesInfo // APM (Agent Package Manager) dependencies
	toolsTimeout          int
	toolsStartupTimeout   int
	markdownContent       string
	importedMarkdown      string   // Only imports WITH inputs (for compile-time substitution)
	importPaths           []string // Import paths for runtime-import macro generation (imports without inputs)
	mainWorkflowMarkdown  string   // main workflow markdown without imports (for runtime-import)
	allIncludedFiles      []string
	workflowName          string
	frontmatterName       string
	needsTextOutput       bool
	trackerID             string
	safeOutputs           *SafeOutputsConfig
	secretMasking         *SecretMaskingConfig
	parsedFrontmatter     *FrontmatterConfig
	hasExplicitGitHubTool bool // true if tools.github was explicitly configured in frontmatter
}

// processToolsAndMarkdown processes tools configuration, runtimes, and markdown content.
// This function handles:
// - Safe outputs and secret masking configuration
// - Tools and MCP servers merging
// - Runtimes merging
// - MCP validations
// - Markdown content expansion
// - Workflow name extraction
func (c *Compiler) processToolsAndMarkdown(result *parser.FrontmatterResult, cleanPath string, markdownDir string,
	agenticEngine CodingAgentEngine, engineSetting string, importsResult *parser.ImportsResult) (*toolsProcessingResult, error) {

	orchestratorToolsLog.Printf("Processing tools and markdown")
	log.Print("Processing tools and includes...")

	// Extract SafeOutputs configuration early so we can use it when applying default tools
	safeOutputs := c.extractSafeOutputsConfig(result.Frontmatter)

	// Extract SecretMasking configuration
	secretMasking := c.extractSecretMaskingConfig(result.Frontmatter)

	// Merge secret-masking from imports with top-level secret-masking
	if importsResult.MergedSecretMasking != "" {
		orchestratorToolsLog.Printf("Merging secret-masking from imports")
		var err error
		secretMasking, err = c.MergeSecretMasking(secretMasking, importsResult.MergedSecretMasking)
		if err != nil {
			orchestratorToolsLog.Printf("Secret-masking merge failed: %v", err)
			return nil, fmt.Errorf("failed to merge secret-masking: %w", err)
		}
	}

	var tools map[string]any

	// Extract tools from the main file
	topTools := extractToolsFromFrontmatter(result.Frontmatter)

	// Emit deprecation warning if tools.serena is used directly in frontmatter.
	// The check is done here, before merging with mcp-servers, so that the warning
	// fires only for explicit tools.serena usage and not when serena is configured
	// via the recommended mcp-servers.serena approach (shared/mcp/serena.md import).
	if _, exists := topTools["serena"]; exists {
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(
			"'tools.serena' is deprecated. Use 'mcp-servers.serena' with the shared/mcp/serena.md workflow instead:\n"+
				"  imports:\n"+
				"    - uses: shared/mcp/serena.md\n"+
				"      with:\n"+
				"        languages: [\"go\", \"typescript\"]",
		))
	}

	// Extract mcp-servers from the main file and merge them into tools
	mcpServers := extractMCPServersFromFrontmatter(result.Frontmatter)

	// Process @include directives to extract additional tools
	orchestratorToolsLog.Printf("Expanding includes for tools")
	includedTools, includedToolFiles, err := parser.ExpandIncludesWithManifest(result.Markdown, markdownDir, true)
	if err != nil {
		orchestratorToolsLog.Printf("Failed to expand includes for tools: %v", err)
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
		orchestratorToolsLog.Printf("Merging imported mcp-servers")
		// Parse and merge imported MCP servers
		mergedMCPServers, err := c.MergeMCPServers(mcpServers, importsResult.MergedMCPServers)
		if err != nil {
			orchestratorToolsLog.Printf("MCP servers merge failed: %v", err)
			return nil, fmt.Errorf("failed to merge imported mcp-servers: %w", err)
		}
		allMCPServers = mergedMCPServers
	}

	// Merge tools including mcp-servers
	orchestratorToolsLog.Printf("Merging tools and MCP servers")
	tools, err = c.mergeToolsAndMCPServers(topTools, allMCPServers, allIncludedTools)
	if err != nil {
		orchestratorToolsLog.Printf("Tools merge failed: %v", err)
		return nil, fmt.Errorf("failed to merge tools: %w", err)
	}

	// Check if GitHub tool was explicitly configured in the original frontmatter
	// This is needed to determine if permissions validation should be skipped
	hasExplicitGitHubTool := false
	if tools != nil {
		if _, exists := tools["github"]; exists {
			// GitHub tool exists in merged tools - check if it was explicitly configured
			// by looking at the original frontmatter before any merging
			if topTools != nil {
				if _, existsInTop := topTools["github"]; existsInTop {
					hasExplicitGitHubTool = true
					orchestratorToolsLog.Print("GitHub tool was explicitly configured in frontmatter")
				}
			}
		}
	}
	orchestratorToolsLog.Printf("hasExplicitGitHubTool: %v", hasExplicitGitHubTool)

	// Extract and validate tools timeout settings
	toolsTimeout, err := c.extractToolsTimeout(tools)
	if err != nil {
		return nil, fmt.Errorf("invalid tools timeout configuration: %w", err)
	}

	toolsStartupTimeout, err := c.extractToolsStartupTimeout(tools)
	if err != nil {
		return nil, fmt.Errorf("invalid tools startup timeout configuration: %w", err)
	}

	// Remove meta fields (timeout, startup-timeout) from merged tools map
	// These are configuration fields, not actual tools
	delete(tools, "timeout")
	delete(tools, "startup-timeout")

	// Extract and merge runtimes from frontmatter and imports
	topRuntimes := extractRuntimesFromFrontmatter(result.Frontmatter)
	orchestratorToolsLog.Printf("Merging runtimes")
	runtimes, err := mergeRuntimes(topRuntimes, importsResult.MergedRuntimes)
	if err != nil {
		orchestratorToolsLog.Printf("Runtimes merge failed: %v", err)
		return nil, fmt.Errorf("failed to merge runtimes: %w", err)
	}

	// Extract APM dependencies from frontmatter
	apmDependencies, err := extractAPMDependenciesFromFrontmatter(result.Frontmatter)
	if err != nil {
		return nil, err
	}
	if apmDependencies != nil {
		orchestratorToolsLog.Printf("Extracted %d APM dependencies from frontmatter", len(apmDependencies.Packages))
	}

	// Add MCP fetch server if needed (when web-fetch is requested but engine doesn't support it)
	tools, _ = AddMCPFetchServerIfNeeded(tools, agenticEngine)

	// Validate MCP configurations
	orchestratorToolsLog.Printf("Validating MCP configurations")
	if err := ValidateMCPConfigs(tools); err != nil {
		orchestratorToolsLog.Printf("MCP configuration validation failed: %v", err)
		return nil, err
	}

	if !agenticEngine.SupportsToolsAllowlist() {
		// For engines that don't support tool allowlists (like custom engine), ignore tools section and provide warnings
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Using experimental %s support (engine: %s)", agenticEngine.GetDisplayName(), agenticEngine.GetID())))
		c.IncrementWarningCount()
		if _, hasTools := result.Frontmatter["tools"]; hasTools {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("'tools' section ignored when using engine: %s (%s doesn't support MCP tool allow-listing)", agenticEngine.GetID(), agenticEngine.GetDisplayName())))
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

	// Validate max-continuations support for the current engine
	if err := c.validateMaxContinuationsSupport(result.Frontmatter, agenticEngine); err != nil {
		return nil, err
	}

	// Validate web-search support for the current engine (warning only)
	c.validateWebSearchSupport(tools, agenticEngine)

	// Process @include directives in markdown content
	markdownContent, includedMarkdownFiles, err := parser.ExpandIncludesWithManifest(result.Markdown, markdownDir, false)
	if err != nil {
		return nil, fmt.Errorf("failed to expand includes in markdown: %w", err)
	}

	// Store the main workflow markdown (before prepending imports)
	mainWorkflowMarkdown := markdownContent
	orchestratorToolsLog.Printf("Main workflow markdown: %d bytes", len(mainWorkflowMarkdown))

	// Get import paths for runtime-import macro generation
	var importPaths []string
	if len(importsResult.ImportPaths) > 0 {
		importPaths = importsResult.ImportPaths
		orchestratorToolsLog.Printf("Found %d import paths for runtime-import macros", len(importPaths))
	}

	// Handle imported markdown from frontmatter imports field
	// Only imports WITH inputs will have markdown content (for compile-time substitution)
	var importedMarkdown string
	if importsResult.MergedMarkdown != "" {
		importedMarkdown = importsResult.MergedMarkdown
		markdownContent = importsResult.MergedMarkdown + markdownContent
		orchestratorToolsLog.Printf("Stored imported markdown with inputs: %d bytes, combined markdown: %d bytes", len(importedMarkdown), len(markdownContent))
	} else {
		orchestratorToolsLog.Print("No imported markdown with inputs")
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

	// Extract workflow name — use content-based extraction when content is pre-loaded (Wasm)
	var workflowName string
	if c.contentOverride != "" {
		workflowName, err = parser.ExtractWorkflowNameFromContent(c.contentOverride, cleanPath)
	} else {
		// Use the already-parsed markdown body to avoid a redundant file read and YAML parse.
		workflowName, err = parser.ExtractWorkflowNameFromMarkdownBody(result.Markdown, cleanPath)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to extract workflow name: %w", err)
	}

	// Check if frontmatter specifies a custom name and use it instead
	frontmatterName := extractStringFromMap(result.Frontmatter, "name", nil)
	if frontmatterName != "" {
		workflowName = frontmatterName
	}

	log.Printf("Extracted workflow name: '%s'", workflowName)

	// Check if the markdown content uses the text output OR if the workflow is triggered by
	// events that have content (issues, discussions, PRs, comments). The sanitized step should
	// be added in either case to make text/title/body outputs available.
	explicitUsage := c.detectTextOutputUsage(markdownContent)
	hasContext := c.hasContentContext(result.Frontmatter)
	needsTextOutput := explicitUsage || hasContext

	orchestratorToolsLog.Printf("Text output needed: explicit=%v, context=%v, final=%v",
		explicitUsage, hasContext, needsTextOutput)

	// Extract and validate tracker-id
	trackerID, err := c.extractTrackerID(result.Frontmatter)
	if err != nil {
		return nil, err
	}

	// Parse frontmatter config once for performance optimization
	parsedFrontmatter, err := ParseFrontmatterConfig(result.Frontmatter)
	if err != nil {
		orchestratorToolsLog.Printf("Failed to parse frontmatter config: %v", err)
		// Non-fatal error - continue with nil ParsedFrontmatter
		parsedFrontmatter = nil
	}

	return &toolsProcessingResult{
		tools:                 tools,
		runtimes:              runtimes,
		apmDependencies:       apmDependencies,
		toolsTimeout:          toolsTimeout,
		toolsStartupTimeout:   toolsStartupTimeout,
		markdownContent:       markdownContent,
		importedMarkdown:      importedMarkdown, // Only imports WITH inputs
		importPaths:           importPaths,      // Import paths for runtime-import macros (imports without inputs)
		mainWorkflowMarkdown:  mainWorkflowMarkdown,
		allIncludedFiles:      allIncludedFiles,
		workflowName:          workflowName,
		frontmatterName:       frontmatterName,
		needsTextOutput:       needsTextOutput,
		trackerID:             trackerID,
		safeOutputs:           safeOutputs,
		secretMasking:         secretMasking,
		parsedFrontmatter:     parsedFrontmatter,
		hasExplicitGitHubTool: hasExplicitGitHubTool,
	}, nil
}

// detectTextOutputUsage checks if the markdown content uses ${{ steps.sanitized.outputs.text }},
// ${{ steps.sanitized.outputs.title }}, or ${{ steps.sanitized.outputs.body }}
func (c *Compiler) detectTextOutputUsage(markdownContent string) bool {
	// Check for any of the text-related output expressions
	hasTextUsage := strings.Contains(markdownContent, "${{ steps.sanitized.outputs.text }}")
	hasTitleUsage := strings.Contains(markdownContent, "${{ steps.sanitized.outputs.title }}")
	hasBodyUsage := strings.Contains(markdownContent, "${{ steps.sanitized.outputs.body }}")

	hasUsage := hasTextUsage || hasTitleUsage || hasBodyUsage
	detectionLog.Printf("Detected usage of sanitized outputs - text: %v, title: %v, body: %v, any: %v",
		hasTextUsage, hasTitleUsage, hasBodyUsage, hasUsage)
	return hasUsage
}

// hasContentContext checks if the workflow is triggered by events that have text content
// (issues, discussions, pull requests, or comments). These events can provide sanitized
// text/title/body outputs via the sanitized step, even if not explicitly referenced.
func (c *Compiler) hasContentContext(frontmatter map[string]any) bool {
	// Check if "on" field exists
	onField, exists := frontmatter["on"]
	if !exists || onField == nil {
		return false
	}

	// Only the map form of the "on" field contains individually-keyed event triggers.
	// String ("on: issues") and array ("on: [issues]") forms are not inspected because
	// GitHub Actions treats them as default-activity-type triggers and the original
	// implementation only detected events that appeared as YAML map keys (i.e. "event:").
	onMap, ok := onField.(map[string]any)
	if !ok {
		orchestratorToolsLog.Printf("No content context detected: 'on' is not a map")
		return false
	}

	// Content-related event types that provide text/title/body outputs via the sanitized step.
	// These are the same events supported by compute_text.cjs.
	// Note: "issues", "pull_request", and "discussion" are included here, which also covers
	// workflows using "labeled"/"unlabeled" activity types on those events — any trigger that
	// declares one of these events as a map key is treated as having content context.
	contentEventKeys := map[string]bool{
		"issues":                      true,
		"pull_request":                true,
		"pull_request_target":         true,
		"issue_comment":               true,
		"pull_request_review_comment": true,
		"pull_request_review":         true,
		"discussion":                  true,
		"discussion_comment":          true,
		"slash_command":               true,
	}

	for eventName := range onMap {
		if contentEventKeys[eventName] {
			orchestratorToolsLog.Printf("Detected content context: workflow triggered by %s", eventName)
			return true
		}
	}

	orchestratorToolsLog.Printf("No content context detected in trigger events")
	return false
}
