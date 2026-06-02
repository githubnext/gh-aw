package workflow

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/parser"
	"github.com/github/gh-aw/pkg/workflow/compilerenv"
)

var orchestratorEngineLog = logger.New("workflow:compiler_orchestrator_engine")

// engineSetupResult holds the results of engine configuration and validation
type engineSetupResult struct {
	engineSetting      string
	engineConfig       *EngineConfig
	agenticEngine      CodingAgentEngine
	networkPermissions *NetworkPermissions
	sandboxConfig      *SandboxConfig
	importsResult      *parser.ImportsResult
	configSteps        []map[string]any // steps returned by RenderConfig (may be nil)
}

// setupEngineAndImports configures the AI engine, processes imports, and validates network/sandbox settings.
// This function handles:
// - Engine extraction and validation
// - Import processing and merging
// - Network permissions setup
// - Sandbox configuration
// - Strict mode validations
func (c *Compiler) setupEngineAndImports(result *parser.FrontmatterResult, cleanPath string, content []byte, markdownDir string) (*engineSetupResult, error) {
	orchestratorEngineLog.Printf("Setting up engine and processing imports")
	engineSetting, engineConfig := c.ExtractEngineConfig(result.Frontmatter)
	preservedMaxTurns, preservedMaxEffectiveTokens, preservedMaxRuns := extractEngineBudgetLimits(engineConfig)
	if err := c.validateAndRegisterInlineEngineConfig(engineConfig); err != nil {
		return nil, err
	}
	networkPermissions := defaultNetworkPermissions(c.extractNetworkPermissions(result.Frontmatter))
	sandboxConfig := c.extractSandboxConfig(result.Frontmatter)
	if err := c.runStrictFrontmatterValidations(result.Frontmatter, networkPermissions); err != nil {
		return nil, err
	}
	engineSetting, engineConfig = c.applyEngineOverride(engineSetting, engineConfig)
	engineSetting, engineConfig = c.injectBuiltinEngineImportIfNeeded(result.Frontmatter, engineSetting, engineConfig)
	importsResult, networkPermissions, err := c.processEngineImportsAndMerge(result, cleanPath, content, markdownDir, engineSetting, networkPermissions)
	if err != nil {
		return nil, err
	}
	engineSetting, engineConfig, err = c.resolveEngineFromIncludesAndImports(result, markdownDir, importsResult, engineSetting, engineConfig)
	if err != nil {
		return nil, err
	}
	engineConfig = c.applyEngineImportDefaults(engineConfig, engineSetting, importsResult, preservedMaxTurns, preservedMaxEffectiveTokens, preservedMaxRuns)
	agenticEngine, configSteps, err := c.resolveEngineRuntimeConfig(engineSetting, engineConfig)
	if err != nil {
		return nil, err
	}
	if err := c.runPostEngineValidations(result.Frontmatter, engineSetting, engineConfig, networkPermissions, sandboxConfig, agenticEngine, importsResult); err != nil {
		return nil, err
	}
	return &engineSetupResult{
		engineSetting:      engineSetting,
		engineConfig:       engineConfig,
		agenticEngine:      agenticEngine,
		networkPermissions: networkPermissions,
		sandboxConfig:      sandboxConfig,
		importsResult:      importsResult,
		configSteps:        configSteps,
	}, nil
}

func extractEngineBudgetLimits(engineConfig *EngineConfig) (string, int64, int) {
	if engineConfig == nil {
		return "", 0, 0
	}
	return engineConfig.MaxTurns, engineConfig.MaxEffectiveTokens, engineConfig.MaxRuns
}

func defaultNetworkPermissions(networkPermissions *NetworkPermissions) *NetworkPermissions {
	if networkPermissions != nil {
		return networkPermissions
	}
	return &NetworkPermissions{Allowed: []string{"defaults"}}
}

func (c *Compiler) validateAndRegisterInlineEngineConfig(engineConfig *EngineConfig) error {
	if engineConfig == nil || !engineConfig.IsInlineDefinition {
		return nil
	}
	if err := c.validateEngineInlineDefinition(engineConfig); err != nil {
		return err
	}
	if err := c.validateEngineAuthDefinition(engineConfig); err != nil {
		return err
	}
	c.registerInlineEngineDefinition(engineConfig)
	return nil
}

func (c *Compiler) runStrictFrontmatterValidations(frontmatter map[string]any, networkPermissions *NetworkPermissions) error {
	return c.withEffectiveStrictMode(frontmatter, func() error {
		orchestratorEngineLog.Printf("Performing strict mode validation (strict=%v)", c.strictMode)
		if err := c.validateStrictMode(frontmatter, networkPermissions); err != nil {
			orchestratorEngineLog.Printf("Strict mode validation failed: %v", err)
			return err
		}
		validations := []struct {
			name string
			fn   func(map[string]any) error
		}{
			{name: "Env secrets", fn: c.validateEnvSecrets},
			{name: "Steps secrets", fn: c.validateStepsSecrets},
			{name: "Update check", fn: c.validateUpdateCheck},
		}
		for _, validation := range validations {
			if err := validation.fn(frontmatter); err != nil {
				orchestratorEngineLog.Printf("%s validation failed: %v", validation.name, err)
				return err
			}
		}
		return nil
	})
}

func (c *Compiler) withEffectiveStrictMode(frontmatter map[string]any, fn func() error) error {
	initialStrictMode := c.strictMode
	c.strictMode = c.effectiveStrictMode(frontmatter)
	defer func() {
		c.strictMode = initialStrictMode
	}()
	return fn()
}

func (c *Compiler) applyEngineOverride(engineSetting string, engineConfig *EngineConfig) (string, *EngineConfig) {
	if c.engineOverride == "" {
		return engineSetting, engineConfig
	}
	if engineSetting != "" && engineSetting != c.engineOverride {
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Command line --engine %s overrides markdown file engine: %s", c.engineOverride, engineSetting)))
		c.IncrementWarningCount()
	}
	if engineConfig != nil {
		engineConfig.ID = c.engineOverride
	}
	return c.engineOverride, engineConfig
}

func (c *Compiler) injectBuiltinEngineImportIfNeeded(frontmatter map[string]any, engineSetting string, engineConfig *EngineConfig) (string, *EngineConfig) {
	if c.engineOverride != "" || !isStringFormEngine(frontmatter) || engineSetting == "" {
		return engineSetting, engineConfig
	}
	builtinPath := builtinEnginePath(engineSetting)
	if !parser.BuiltinVirtualFileExists(builtinPath) {
		return engineSetting, engineConfig
	}
	orchestratorEngineLog.Printf("Injecting builtin engine import: %s", builtinPath)
	addImportToFrontmatter(frontmatter, builtinPath)
	delete(frontmatter, "engine")
	return "", nil
}

func (c *Compiler) processEngineImportsAndMerge(
	result *parser.FrontmatterResult,
	cleanPath string,
	content []byte,
	markdownDir string,
	engineSetting string,
	networkPermissions *NetworkPermissions,
) (*parser.ImportsResult, *NetworkPermissions, error) {
	orchestratorEngineLog.Printf("Processing imports from frontmatter")
	importCache := c.getSharedImportCache()
	importsResult, err := parser.ProcessImportsFromFrontmatterWithSource(result.Frontmatter, markdownDir, importCache, cleanPath, string(content))
	if err != nil {
		orchestratorEngineLog.Printf("Import processing failed: %v", err)
		var cycleErr *parser.ImportCycleError
		if errors.As(err, &cycleErr) {
			return nil, nil, parser.FormatImportCycleError(cycleErr)
		}
		return nil, nil, err
	}
	if err := scanImportedMarkdownFiles(importsResult.ImportedFiles, markdownDir, importCache); err != nil {
		return nil, nil, err
	}
	if importsResult.MergedNetwork != "" {
		orchestratorEngineLog.Printf("Merging network permissions from imports")
		networkPermissions, err = c.MergeNetworkPermissions(networkPermissions, importsResult.MergedNetwork)
		if err != nil {
			orchestratorEngineLog.Printf("Network permissions merge failed: %v", err)
			return nil, nil, fmt.Errorf("failed to merge network permissions: %w", err)
		}
	}
	if importsResult.MergedPermissions != "" {
		orchestratorEngineLog.Printf("Validating included permissions")
		topLevelPermissions := c.extractPermissions(result.Frontmatter)
		if err := c.ValidateIncludedPermissions(topLevelPermissions, importsResult.MergedPermissions); err != nil {
			orchestratorEngineLog.Printf("Included permissions validation failed: %v", err)
			return nil, nil, fmt.Errorf("permission validation failed: %w", err)
		}
	}
	return importsResult, networkPermissions, nil
}

func scanImportedMarkdownFiles(importedFiles []string, markdownDir string, importCache *parser.ImportCache) error {
	for _, importedFile := range importedFiles {
		importFilePath := importedFile
		if idx := strings.Index(importFilePath, "#"); idx >= 0 {
			importFilePath = importFilePath[:idx]
		}
		if !shouldScanImportedMarkdown(importFilePath) {
			continue
		}
		fullPath, resolveErr := parser.ResolveIncludePath(importFilePath, markdownDir, importCache)
		if resolveErr != nil {
			orchestratorEngineLog.Printf("Skipping security scan for unresolvable import: %s: %v", importedFile, resolveErr)
			fmt.Fprintf(os.Stderr, "WARNING: Skipping security scan for unresolvable import '%s': %v\n", importedFile, resolveErr)
			continue
		}
		importContent, readErr := parser.ReadFile(fullPath)
		if readErr != nil {
			orchestratorEngineLog.Printf("Skipping security scan for unreadable import: %s: %v", fullPath, readErr)
			fmt.Fprintf(os.Stderr, "WARNING: Skipping security scan for unreadable import '%s' (resolved path: %s): %v\n", importedFile, fullPath, readErr)
			continue
		}
		if findings := ScanMarkdownSecurity(string(importContent)); len(findings) > 0 {
			orchestratorEngineLog.Printf("Security scan failed for imported file: %s (%d findings)", importedFile, len(findings))
			return fmt.Errorf("imported workflow '%s' failed security scan: %s", importedFile, FormatSecurityFindings(findings, importedFile))
		}
	}
	return nil
}

func (c *Compiler) resolveEngineFromIncludesAndImports(
	result *parser.FrontmatterResult,
	markdownDir string,
	importsResult *parser.ImportsResult,
	engineSetting string,
	engineConfig *EngineConfig,
) (string, *EngineConfig, error) {
	orchestratorEngineLog.Printf("Expanding includes for engine configurations")
	includedEngines, err := parser.ExpandIncludesForEngines(result.Markdown, markdownDir)
	if err != nil {
		orchestratorEngineLog.Printf("Failed to expand includes for engines: %v", err)
		return "", nil, fmt.Errorf("failed to expand includes for engines: %w", err)
	}
	allEngines := append(importsResult.MergedEngines, includedEngines...)
	orchestratorEngineLog.Printf("Validating single engine specification")
	finalEngineSetting, err := c.validateSingleEngineSpecification(engineSetting, allEngines)
	if err != nil {
		orchestratorEngineLog.Printf("Engine specification validation failed: %v", err)
		return "", nil, err
	}
	if finalEngineSetting != "" {
		engineSetting = finalEngineSetting
	}
	if engineConfig == nil && len(allEngines) > 0 {
		orchestratorEngineLog.Printf("Extracting engine config from included file")
		engineConfig, err = c.extractEngineConfigFromJSON(allEngines[0])
		if err != nil {
			orchestratorEngineLog.Printf("Failed to extract engine config: %v", err)
			return "", nil, fmt.Errorf("failed to extract engine config from included file: %w", err)
		}
		if err := c.validateAndRegisterInlineEngineConfig(engineConfig); err != nil {
			return "", nil, err
		}
	}
	if engineSetting == "" {
		defaultEngine := c.engineRegistry.GetDefaultEngine()
		engineSetting = defaultEngine.GetID()
		workflowLog.Printf("No 'engine:' setting found, defaulting to: %s", engineSetting)
	}
	if engineConfig == nil {
		engineConfig = &EngineConfig{ID: engineSetting}
	} else if engineConfig.ID == "" && engineSetting != "" {
		engineConfig.ID = engineSetting
		orchestratorEngineLog.Printf("Normalized engineConfig.ID from engineSetting: %s", engineSetting)
	}
	return engineSetting, engineConfig, nil
}

// applyEngineImportDefaults merges import-derived engine defaults into engineConfig.
// It mutates the provided config when non-nil and returns the effective pointer.
// Callers must always use the returned value because a new config may be allocated
// when the input engineConfig is nil.
func (c *Compiler) applyEngineImportDefaults(
	engineConfig *EngineConfig,
	engineSetting string,
	importsResult *parser.ImportsResult,
	preservedMaxTurns string,
	preservedMaxEffectiveTokens int64,
	preservedMaxRuns int,
) *EngineConfig {
	if engineConfig == nil {
		engineConfig = &EngineConfig{ID: engineSetting}
	}
	if preservedMaxTurns != "" {
		engineConfig.MaxTurns = preservedMaxTurns
	}
	if preservedMaxEffectiveTokens != 0 {
		engineConfig.MaxEffectiveTokens = preservedMaxEffectiveTokens
	}
	if preservedMaxRuns > 0 {
		engineConfig.MaxRuns = preservedMaxRuns
	}
	if engineConfig.MaxTurns == "" && importsResult.MergedMaxTurns != "" {
		var importedMaxTurns any
		if err := json.Unmarshal([]byte(importsResult.MergedMaxTurns), &importedMaxTurns); err == nil {
			if parsed := parseMaxTurnsValue(importedMaxTurns); parsed != "" {
				engineConfig.MaxTurns = parsed
				orchestratorEngineLog.Printf("Applied max-turns from import")
			}
		}
	}
	if engineConfig.MaxRuns <= 0 && importsResult.MergedMaxRuns != "" {
		var importedMaxRuns any
		if err := json.Unmarshal([]byte(importsResult.MergedMaxRuns), &importedMaxRuns); err == nil {
			if parsed := parseMaxRunsValue(importedMaxRuns); parsed > 0 {
				engineConfig.MaxRuns = parsed
				orchestratorEngineLog.Printf("Applied max-runs from import")
			}
		}
	}
	if engineConfig.MaxEffectiveTokens == 0 && importsResult.MergedMaxEffectiveTokens != "" {
		var importedMaxTokens any
		if err := json.Unmarshal([]byte(importsResult.MergedMaxEffectiveTokens), &importedMaxTokens); err == nil {
			if parsed := parseMaxEffectiveTokensValue(importedMaxTokens); parsed != 0 {
				engineConfig.MaxEffectiveTokens = parsed
				orchestratorEngineLog.Printf("Applied max-effective-tokens from import")
			}
		}
	}
	if engineConfig.MCPToolTimeout == "" && importsResult.MergedEngineMCPToolTimeout != "" {
		engineConfig.MCPToolTimeout = importsResult.MergedEngineMCPToolTimeout
		orchestratorEngineLog.Printf("Applied engine.mcp.tool-timeout from import: %s", engineConfig.MCPToolTimeout)
	}
	if engineConfig.MCPSessionTimeout == "" && importsResult.MergedEngineMCPSessionTimeout != "" {
		engineConfig.MCPSessionTimeout = importsResult.MergedEngineMCPSessionTimeout
		orchestratorEngineLog.Printf("Applied engine.mcp.session-timeout from import: %s", engineConfig.MCPSessionTimeout)
	}
	if engineConfig.Model == "" && importsResult.MergedEngineModel != "" {
		engineConfig.Model = importsResult.MergedEngineModel
		orchestratorEngineLog.Printf("Applied engine.model preference from import: %s", engineConfig.Model)
	}
	return engineConfig
}

func (c *Compiler) resolveEngineRuntimeConfig(engineSetting string, engineConfig *EngineConfig) (CodingAgentEngine, []map[string]any, error) {
	orchestratorEngineLog.Printf("Resolving engine setting: %s", engineSetting)
	resolvedEngine, err := c.engineCatalog.Resolve(engineSetting, engineConfig)
	if err != nil {
		orchestratorEngineLog.Printf("Engine resolution failed: %v", err)
		return nil, nil, err
	}
	agenticEngine := resolvedEngine.Runtime
	const noDefaultMaxTurns = ""
	if engineConfig != nil && engineConfig.MaxTurns == "" && agenticEngine.GetCapabilities().MaxTurns {
		engineConfig.MaxTurns = compilerenv.ResolveDefaultMaxTurns(noDefaultMaxTurns)
	}
	orchestratorEngineLog.Printf("Calling RenderConfig for engine: %s", engineSetting)
	configSteps, err := agenticEngine.RenderConfig(resolvedEngine)
	if err != nil {
		orchestratorEngineLog.Printf("RenderConfig failed for engine %s: %v", engineSetting, err)
		return nil, nil, fmt.Errorf("engine %s RenderConfig failed: %w", engineSetting, err)
	}
	workflowLog.Printf("AI engine: %s (%s)", agenticEngine.GetDisplayName(), engineSetting)
	if agenticEngine.IsExperimental() && c.verbose {
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage("Using experimental engine: "+agenticEngine.GetDisplayName()))
		c.IncrementWarningCount()
	}
	return agenticEngine, configSteps, nil
}

func (c *Compiler) runPostEngineValidations(
	frontmatter map[string]any,
	engineSetting string,
	engineConfig *EngineConfig,
	networkPermissions *NetworkPermissions,
	sandboxConfig *SandboxConfig,
	agenticEngine CodingAgentEngine,
	importsResult *parser.ImportsResult,
) error {
	enableFirewallByDefaultForCopilot(engineSetting, networkPermissions, sandboxConfig)
	enableFirewallByDefaultForClaude(engineSetting, networkPermissions, sandboxConfig)
	return c.withEffectiveStrictMode(frontmatter, func() error {
		orchestratorEngineLog.Printf("Validating strict firewall (strict=%v)", c.strictMode)
		if err := c.validateStrictFirewall(engineSetting, networkPermissions, sandboxConfig); err != nil {
			orchestratorEngineLog.Printf("Strict firewall validation failed: %v", err)
			return err
		}
		orchestratorEngineLog.Printf("Validating strict sandbox customization (strict=%v)", c.strictMode)
		if err := c.validateStrictSandboxCustomization(sandboxConfig); err != nil {
			orchestratorEngineLog.Printf("Strict sandbox customization validation failed: %v", err)
			return err
		}
		if err := c.checkNetworkSupport(agenticEngine, networkPermissions); err != nil {
			orchestratorEngineLog.Printf("Network support check failed: %v", err)
			return err
		}
		orchestratorEngineLog.Printf("Validating imported steps for agentic secrets (strict=%v)", c.strictMode)
		if err := c.validateImportedStepsNoAgenticSecrets(engineConfig, engineSetting); err != nil {
			orchestratorEngineLog.Printf("Imported steps validation failed: %v", err)
			return err
		}
		orchestratorEngineLog.Printf("Validating checkout persist-credentials (strict=%v)", c.strictMode)
		if err := c.validateCheckoutPersistCredentials(frontmatter, importsResult.MergedSteps); err != nil {
			orchestratorEngineLog.Printf("Checkout persist-credentials validation failed: %v", err)
			return err
		}
		return nil
	})
}

// shouldScanImportedMarkdown reports whether an import path should be processed by
// markdown security scanning.
func shouldScanImportedMarkdown(importFilePath string) bool {
	if !strings.HasSuffix(importFilePath, ".md") {
		return false
	}
	return !strings.HasPrefix(importFilePath, parser.BuiltinPathPrefix)
}

// isStringFormEngine reports whether the "engine" field in the given frontmatter is a
// plain string (e.g. "engine: copilot"), as opposed to an object with an "id" or
// "runtime" sub-key.
func isStringFormEngine(frontmatter map[string]any) bool {
	engine, exists := frontmatter["engine"]
	if !exists {
		return false
	}
	_, isString := engine.(string)
	return isString
}

// addImportToFrontmatter appends importPath to the "imports" slice in frontmatter.
// It handles the case where "imports" may be absent, a []any, a []string, or a
// single string (which is converted to a two-element slice preserving the original value).
// When "imports" is an object (map) with an "aw" subfield, the path is appended to "aw".
// Any other unexpected type is left unchanged and importPath is not injected.
func addImportToFrontmatter(frontmatter map[string]any, importPath string) {
	existing, hasImports := frontmatter["imports"]
	if !hasImports {
		frontmatter["imports"] = []any{importPath}
		return
	}
	switch v := existing.(type) {
	case []any:
		frontmatter["imports"] = append(v, importPath)
	case []string:
		newSlice := make([]any, len(v)+1)
		for i, s := range v {
			newSlice[i] = s
		}
		newSlice[len(v)] = importPath
		frontmatter["imports"] = newSlice
	case string:
		// Single string import — preserve it and append the new one.
		frontmatter["imports"] = []any{v, importPath}
	case map[string]any:
		// Object form — append to the "aw" subfield.
		if awAny, hasAW := v["aw"]; hasAW {
			switch aw := awAny.(type) {
			case []any:
				v["aw"] = append(aw, importPath)
			case []string:
				newSlice := make([]any, len(aw)+1)
				for i, s := range aw {
					newSlice[i] = s
				}
				newSlice[len(aw)] = importPath
				v["aw"] = newSlice
			}
		} else {
			// No "aw" subfield yet — create it.
			v["aw"] = []any{importPath}
		}
		// For any other unexpected type, leave the field untouched so the
		// downstream parser can still report its own error for the invalid value.
	}
}
