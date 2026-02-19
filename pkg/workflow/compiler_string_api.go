package workflow

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/github/gh-aw/pkg/parser"
	"github.com/github/gh-aw/pkg/stringutil"
)

// CompileToYAML compiles workflow data and returns the YAML as a string
// without writing to disk. This is useful for Wasm builds and programmatic usage.
func (c *Compiler) CompileToYAML(workflowData *WorkflowData, markdownPath string) (string, error) {
	c.markdownPath = markdownPath
	c.skipHeader = true
	// Clear contentOverride after compilation (set by ParseWorkflowString)
	defer func() { c.contentOverride = "" }()

	startTime := time.Now()
	defer func() {
		log.Printf("CompileToYAML completed in %v", time.Since(startTime))
	}()

	c.stepOrderTracker = NewStepOrderTracker()
	c.scheduleFriendlyFormats = nil

	if c.artifactManager == nil {
		c.artifactManager = NewArtifactManager()
	} else {
		c.artifactManager.Reset()
	}

	lockFile := stringutil.MarkdownToLockFile(markdownPath)

	if err := c.validateWorkflowData(workflowData, markdownPath); err != nil {
		return "", err
	}

	yamlContent, err := c.generateAndValidateYAML(workflowData, markdownPath, lockFile)
	if err != nil {
		return "", err
	}

	return yamlContent, nil
}

// ParseWorkflowString parses workflow markdown content from a string rather than a file.
// This is the primary entry point for Wasm/browser usage where filesystem access is unavailable.
// The virtualPath is used for error messages and lock file naming (e.g., "workflow.md").
func (c *Compiler) ParseWorkflowString(content string, virtualPath string) (*WorkflowData, error) {
	log.Printf("ParseWorkflowString: parsing %d bytes with virtual path %s", len(content), virtualPath)

	cleanPath := filepath.Clean(virtualPath)

	// Store content so downstream code can use it instead of reading from disk.
	// Cleared in CompileToYAML after compilation completes.
	c.contentOverride = content

	// Enable inline prompt mode for string-based compilation (Wasm/browser)
	// since runtime-import macros cannot resolve without filesystem access
	c.inlinePrompt = true

	// Parse frontmatter directly from content string
	result, err := parser.ExtractFrontmatterFromContent(content)
	if err != nil {
		frontmatterStart := 2
		if result != nil && result.FrontmatterStart > 0 {
			frontmatterStart = result.FrontmatterStart
		}
		return nil, c.createFrontmatterError(cleanPath, content, err, frontmatterStart)
	}

	if len(result.Frontmatter) == 0 {
		return nil, fmt.Errorf("no frontmatter found")
	}

	// Preprocess schedule fields
	if err := c.preprocessScheduleFields(result.Frontmatter, cleanPath, content); err != nil {
		return nil, err
	}

	frontmatterForValidation := c.copyFrontmatterWithoutInternalMarkers(result.Frontmatter)

	// Check if shared workflow (no 'on' field)
	_, hasOnField := frontmatterForValidation["on"]
	if !hasOnField {
		return nil, &SharedWorkflowError{Path: cleanPath}
	}

	// Validate frontmatter against schema
	if err := parser.ValidateMainWorkflowFrontmatterWithSchemaAndLocation(frontmatterForValidation, cleanPath); err != nil {
		return nil, err
	}

	// Build parse result to reuse the rest of the orchestrator pipeline
	parseResult := &frontmatterParseResult{
		cleanPath:                cleanPath,
		content:                  []byte(content),
		frontmatterResult:        result,
		frontmatterForValidation: frontmatterForValidation,
		markdownDir:              filepath.Dir(cleanPath),
		isSharedWorkflow:         false,
	}

	// Setup engine and process imports
	engineSetup, err := c.setupEngineAndImports(parseResult.frontmatterResult, parseResult.cleanPath, parseResult.content, parseResult.markdownDir)
	if err != nil {
		return nil, err
	}

	// Process tools and markdown
	toolsResult, err := c.processToolsAndMarkdown(parseResult.frontmatterResult, parseResult.cleanPath, parseResult.markdownDir, engineSetup.agenticEngine, engineSetup.engineSetting, engineSetup.importsResult)
	if err != nil {
		return nil, err
	}

	// Build initial workflow data structure
	workflowData := c.buildInitialWorkflowData(parseResult.frontmatterResult, toolsResult, engineSetup, engineSetup.importsResult)
	workflowData.WorkflowID = GetWorkflowIDFromPath(cleanPath)

	// Validate bash tool configuration
	if err := validateBashToolConfig(workflowData.ParsedTools, workflowData.Name); err != nil {
		return nil, fmt.Errorf("%s: %w", cleanPath, err)
	}

	// Setup action cache and resolver
	actionCache, actionResolver := c.getSharedActionResolver()
	workflowData.ActionCache = actionCache
	workflowData.ActionResolver = actionResolver
	workflowData.ActionPinWarnings = c.actionPinWarnings

	// Extract YAML configuration sections
	c.extractYAMLSections(parseResult.frontmatterResult.Frontmatter, workflowData)

	// Merge features from imports
	if len(engineSetup.importsResult.MergedFeatures) > 0 {
		mergedFeatures, err := c.MergeFeatures(workflowData.Features, engineSetup.importsResult.MergedFeatures)
		if err != nil {
			return nil, fmt.Errorf("failed to merge features from imports: %w", err)
		}
		workflowData.Features = mergedFeatures
	}

	// Process and merge custom steps
	c.processAndMergeSteps(parseResult.frontmatterResult.Frontmatter, workflowData, engineSetup.importsResult)

	// Apply defaults
	if err := c.applyDefaults(workflowData, cleanPath); err != nil {
		return nil, err
	}

	return workflowData, nil
}
