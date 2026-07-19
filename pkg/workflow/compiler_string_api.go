package workflow

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/parser"
	"github.com/github/gh-aw/pkg/stringutil"
)

var compilerStringAPILog = logger.New("workflow:compiler_string_api")

// CompileToYAML compiles workflow data and returns the YAML as a string
// without writing to disk. This is useful for Wasm builds and programmatic usage.
func (c *Compiler) CompileToYAML(workflowData *WorkflowData, markdownPath string) (string, error) {
	compilerStringAPILog.Printf("CompileToYAML: markdownPath=%s", markdownPath)
	c.markdownPath = markdownPath
	c.skipHeader = true
	// Clear contentOverride after compilation (set by ParseWorkflowString)
	defer func() { c.contentOverride = "" }()

	startTime := time.Now()
	defer func() {
		workflowLog.Printf("CompileToYAML completed in %v", time.Since(startTime))
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

	yamlContent, _, _, err := c.generateAndValidateYAML(workflowData, markdownPath, lockFile)
	if err != nil {
		return "", err
	}

	return yamlContent, nil
}

// ParseWorkflowString parses workflow markdown content from a string rather than a file.
// This is the primary entry point for Wasm/browser usage where filesystem access is unavailable.
// The virtualPath is used for error messages and lock file naming (e.g., "workflow.md").
func (c *Compiler) ParseWorkflowString(content string, virtualPath string) (*WorkflowData, error) {
	workflowLog.Printf("ParseWorkflowString: parsing %d bytes with virtual path %s", len(content), virtualPath)

	cleanPath := filepath.Clean(virtualPath)
	c.contentOverride = content
	c.inlinePrompt = true

	parseResult, err := c.parseAndValidateWorkflowStringFrontmatter(content, cleanPath)
	if err != nil {
		return nil, err
	}
	workflowData, engineSetup, err := c.buildWorkflowDataFromStringParseResult(parseResult)
	if err != nil {
		return nil, err
	}
	if err := c.validateStringWorkflowData(cleanPath, workflowData); err != nil {
		return nil, err
	}
	if err := c.finalizeStringWorkflowData(parseResult, workflowData, engineSetup); err != nil {
		return nil, err
	}

	return workflowData, nil
}

func (c *Compiler) parseAndValidateWorkflowStringFrontmatter(content, cleanPath string) (*frontmatterParseResult, error) {
	contentBytes := []byte(content)
	result, err := parser.ExtractFrontmatterFromContent(content)
	if err != nil {
		frontmatterStart := 2
		if result != nil && result.FrontmatterStart > 0 {
			frontmatterStart = result.FrontmatterStart
		}
		return nil, c.createFrontmatterError(cleanPath, content, err, frontmatterStart)
	}
	if len(result.Frontmatter) == 0 {
		return nil, errors.New("no frontmatter found")
	}
	compilerStringAPILog.Printf("ParseWorkflowString: extracted frontmatter with %d fields", len(result.Frontmatter))
	if err := c.preprocessScheduleFields(result.Frontmatter, cleanPath, content); err != nil {
		return nil, err
	}
	frontmatterForValidation := c.copyFrontmatterWithoutInternalMarkers(result.Frontmatter)
	if err := validateStringWorkflowOnField(cleanPath, frontmatterForValidation); err != nil {
		return nil, err
	}
	if err := c.validateEngineBeforeSchema(cleanPath, contentBytes, result, frontmatterForValidation); err != nil {
		compilerStringAPILog.Printf("ParseWorkflowString: string engine pre-validation failed for %s", cleanPath)
		return nil, err
	}
	if err := parser.ValidateMainWorkflowFrontmatterWithSchemaAndLocation(frontmatterForValidation, cleanPath); err != nil {
		compilerStringAPILog.Printf("ParseWorkflowString: schema validation failed for %s", cleanPath)
		return nil, err
	}
	compilerStringAPILog.Printf("ParseWorkflowString: frontmatter validated, frontmatter_fields=%d", len(frontmatterForValidation))
	return newStringFrontmatterParseResult(cleanPath, contentBytes, result, frontmatterForValidation), nil
}

func validateStringWorkflowOnField(cleanPath string, frontmatterForValidation map[string]any) error {
	if _, hasOnField := frontmatterForValidation["on"]; hasOnField {
		return nil
	}
	if redirectVal, hasRedirect := frontmatterForValidation["redirect"]; hasRedirect {
		if redirectStr, ok := redirectVal.(string); ok {
			if redirectTarget := strings.TrimSpace(redirectStr); redirectTarget != "" {
				compilerStringAPILog.Printf("ParseWorkflowString: redirect-only workflow detected: redirect=%s", redirectTarget)
				return &RedirectOnlyWorkflowError{Path: cleanPath, Target: redirectTarget}
			}
		}
	}
	compilerStringAPILog.Printf("ParseWorkflowString: no 'on' field, treating as shared workflow: %s", cleanPath)
	return &SharedWorkflowError{Path: cleanPath}
}

func newStringFrontmatterParseResult(cleanPath string, contentBytes []byte, result *parser.FrontmatterResult, frontmatterForValidation map[string]any) *frontmatterParseResult {
	return &frontmatterParseResult{
		cleanPath:                cleanPath,
		content:                  contentBytes,
		frontmatterResult:        result,
		frontmatterForValidation: frontmatterForValidation,
		markdownDir:              filepath.Dir(cleanPath),
		isSharedWorkflow:         false,
	}
}

func (c *Compiler) buildWorkflowDataFromStringParseResult(parseResult *frontmatterParseResult) (*WorkflowData, *engineSetupResult, error) {
	engineSetup, err := c.setupEngineAndImports(parseResult.frontmatterResult, parseResult.cleanPath, parseResult.content, parseResult.markdownDir)
	if err != nil {
		return nil, nil, err
	}
	toolsResult, err := c.processToolsAndMarkdown(parseResult.frontmatterResult, parseResult.cleanPath, parseResult.markdownDir, engineSetup.agenticEngine, engineSetup.engineSetting, engineSetup.importsResult)
	if err != nil {
		return nil, nil, err
	}
	workflowData := c.buildInitialWorkflowData(parseResult.frontmatterResult, toolsResult, engineSetup, engineSetup.importsResult)
	workflowData.WorkflowID = GetWorkflowIDFromPath(parseResult.cleanPath)
	return workflowData, engineSetup, nil
}

func (c *Compiler) validateStringWorkflowData(cleanPath string, workflowData *WorkflowData) error {
	if err := validateBashToolConfig(workflowData.ParsedTools, workflowData.Name); err != nil {
		return fmt.Errorf("%s: %w", cleanPath, err)
	}
	if err := c.validateEngineMCPSessionTimeout(workflowData); err != nil {
		return fmt.Errorf("%s: %w", cleanPath, err)
	}
	if err := c.validateEngineMCPToolTimeout(workflowData); err != nil {
		return fmt.Errorf("%s: %w", cleanPath, err)
	}
	if err := validateGitHubToolConfig(workflowData.ParsedTools, workflowData.Name); err != nil {
		return fmt.Errorf("%s: %w", cleanPath, err)
	}
	if err := validateGitHubReadOnly(workflowData.ParsedTools, workflowData.Name); err != nil {
		return fmt.Errorf("%s: %w", cleanPath, err)
	}
	if err := validateGitHubGuardPolicy(workflowData.ParsedTools, workflowData.Name); err != nil {
		return fmt.Errorf("%s: %w", cleanPath, err)
	}
	emitGitHubLockdownGuardPolicyWarning(c, workflowData.ParsedTools, cleanPath)
	var gatewayConfig *MCPGatewayRuntimeConfig
	if workflowData.SandboxConfig != nil {
		gatewayConfig = workflowData.SandboxConfig.MCP
	}
	if err := validateIntegrityReactions(workflowData.ParsedTools, workflowData.Name, workflowData, gatewayConfig); err != nil {
		return fmt.Errorf("%s: %w", cleanPath, err)
	}
	return nil
}

func (c *Compiler) finalizeStringWorkflowData(parseResult *frontmatterParseResult, workflowData *WorkflowData, engineSetup *engineSetupResult) error {
	actionCache, actionResolver := c.getSharedActionResolver()
	workflowData.Ctx = c.ctx
	workflowData.ActionCache = actionCache
	workflowData.ActionResolver = actionResolver
	workflowData.ActionPinWarnings = c.actionPinWarnings
	workflowData.ActionPinMappings = c.getActionPinMappings()

	// Extract YAML configuration sections
	if err := c.extractYAMLSections(parseResult.frontmatterResult.Frontmatter, workflowData); err != nil {
		return fmt.Errorf("failed to extract YAML sections: %w", err)
	}

	// Merge features from imports
	if len(engineSetup.importsResult.MergedFeatures) > 0 {
		compilerStringAPILog.Printf("ParseWorkflowString: merging %d features from imports", len(engineSetup.importsResult.MergedFeatures))
		mergedFeatures, err := c.MergeFeatures(workflowData.Features, engineSetup.importsResult.MergedFeatures)
		if err != nil {
			return fmt.Errorf("failed to merge features from imports: %w", err)
		}
		workflowData.Features = mergedFeatures
	}

	// Process and merge custom steps
	if err := c.processAndMergeSteps(parseResult.frontmatterResult.Frontmatter, workflowData, engineSetup.importsResult); err != nil {
		return err
	}

	// Apply defaults
	if err := c.applyDefaults(workflowData, parseResult.cleanPath); err != nil {
		return err
	}
	return nil
}
