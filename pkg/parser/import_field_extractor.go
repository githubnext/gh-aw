// Package parser provides functions for parsing and processing workflow markdown files.
// import_field_extractor.go implements field extraction from imported workflow files.
// It defines the importAccumulator struct that centralizes all result-building state
// and provides the extractAllImportFields method for processing a single imported file.
package parser

import (
	"encoding/json"
	"fmt"
	"maps"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strings"
)

// importAccumulator centralizes the builder/slice/set variables used during
// BFS import traversal. It accumulates results from all imported files and provides
// a method to convert the accumulated state into the final ImportsResult.
type importAccumulator struct {
	toolsBuilder             strings.Builder
	mcpServersBuilder        strings.Builder
	markdownBuilder          strings.Builder // imports with substituted inputs or schema defaults (compile-time substitution)
	importPaths              []string        // Import paths for runtime-import macro generation
	promptImports            []PromptImportEntry
	stepsBuilder             strings.Builder
	copilotSetupStepsBuilder strings.Builder // Steps from copilot-setup-steps.yml (inserted at start)
	preStepsBuilder          strings.Builder
	preAgentStepsBuilder     strings.Builder
	runtimesBuilder          strings.Builder
	servicesBuilder          strings.Builder
	networkBuilder           strings.Builder
	permissionsBuilder       strings.Builder
	secretMaskingBuilder     strings.Builder
	postStepsBuilder         strings.Builder
	jobsBuilder              strings.Builder   // Jobs from imported YAML workflows
	envBuilder               strings.Builder   // env vars from imported workflows (JSON, one object per line)
	envSources               map[string]string // env var name → source import path (for conflict detection and header listing)
	observabilityConfigs     []string          // observability config JSON blobs from all imports (merged into endpoint array)
	engines                  []string
	safeOutputs              []string
	mcpScripts               []string
	bots                     []string
	botsSet                  map[string]bool
	labels                   []string
	labelsSet                map[string]bool
	skipRoles                []string
	skipRolesSet             map[string]bool
	skipBots                 []string
	skipBotsSet              map[string]bool
	skipIfMatch              string
	skipIfNoMatch            string
	caches                   []string
	features                 []map[string]any
	models                   []map[string][]string // model alias maps from each imported file (appended in import order)
	runInstallScripts        bool                  // true if any imported workflow sets runtimes.node.run-install-scripts: true
	agentFile                string
	agentImportSpec          string
	repositoryImports        []string
	importInputs             map[string]any
	// First on.github-token / on.github-app found across all imported files (first-wins strategy)
	activationGitHubToken string
	activationGitHubApp   string // JSON-encoded GitHubAppConfig
	// First top-level github-app found across all imported files (first-wins strategy)
	topLevelGitHubApp string // JSON-encoded GitHubAppConfig
	// Checkout configs from all imported files (append in order; main workflow's checkouts take precedence)
	checkouts []string // JSON-encoded checkout values, one per import
	// First engine.mcp.tool-timeout / engine.mcp.session-timeout found across all imported files (first-wins strategy)
	mergedEngineMCPToolTimeout    string // Go duration string (e.g. "10m", "30s")
	mergedEngineMCPSessionTimeout string // Go duration string (e.g. "4h", "30m")
	// First engine.model found in imports that have no engine.id (first-wins strategy).
	// These express a model preference without selecting a specific engine.
	mergedEngineModel string
	// First top-level max-turns / max-runs / max-effective-tokens /
	// max-daily-ai-credits (or deprecated max-daily-effective-tokens)
	// found across imports (first-wins).
	// Values are stored as JSON-encoded raw values so numeric literals and strings
	// round-trip consistently through import processing.
	mergedMaxTurns                string
	mergedMaxRuns                 string
	mergedMaxEffectiveTokens      string
	mergedMaxDailyEffectiveTokens string
	// Best-effort sub-agent frontmatter warnings collected during BFS traversal.
	warnings []string
}

// newImportAccumulator creates and initializes a new importAccumulator.
// Maps (botsSet, etc.) are explicitly initialized to prevent nil map panics
// during deduplication. Slices are left as nil, which is valid for append operations.
func newImportAccumulator() *importAccumulator {
	return &importAccumulator{
		botsSet:      make(map[string]bool),
		labelsSet:    make(map[string]bool),
		skipRolesSet: make(map[string]bool),
		skipBotsSet:  make(map[string]bool),
		importInputs: make(map[string]any),
		envSources:   make(map[string]string),
	}
}

// extractAllImportFields extracts all frontmatter fields from a single imported file
// and accumulates the results. Handles tools, engines, mcp-servers, safe-outputs,
// mcp-scripts, steps, runtimes, services, network, permissions, secret-masking, bots,
// skip-roles, skip-bots, pre-steps, pre-agent-steps, post-steps, labels, cache, and features.
// The work is delegated to focused helper methods, each handling one logical phase.
func (acc *importAccumulator) extractAllImportFields(content []byte, item importQueueItem, visited map[string]bool) error {
	parserLog.Printf("Extracting all import fields: path=%s, section=%s, inputs=%d, content_size=%d bytes", item.fullPath, item.sectionName, len(item.inputs), len(content))

	// Phase 1: Parse, apply defaults, substitute inputs, extract tools and markdown.
	origFm, fm, err := acc.prepareFrontmatter(content, item, visited)
	if err != nil {
		return err
	}

	// Phase 2: Validate 'with'/'inputs' values against the imported workflow's 'import-schema'.
	// Always use the ORIGINAL (unsubstituted) frontmatter for schema lookup so the import-schema
	// declaration itself is not affected by expression substitution.
	if _, hasSchema := origFm["import-schema"]; hasSchema {
		if err := validateWithImportSchema(item.inputs, origFm, item.importPath); err != nil {
			return err
		}
	}

	// Phase 3: Extract engine configuration (id, runtime, mcp timeouts, model preference).
	acc.extractEngineConfig(fm, item.fullPath)

	// Phase 4: Extract scalar and builder-based configuration fields.
	acc.extractConfigFields(fm, item.fullPath)

	// Phase 5: Extract activation, authentication, and access-control fields.
	acc.extractActivationFields(fm, item)

	// Phase 6: Extract step, job, and environment fields.
	if err := acc.extractStepAndJobFields(fm, item.importPath); err != nil {
		return err
	}

	// Phase 7: Extract feature flags, model aliases, run-install-scripts, and observability.
	acc.extractFeatureAndObservabilityFields(fm, item.fullPath)

	return nil
}

// prepareFrontmatter handles the parse → defaults → substitution → re-parse pipeline for
// a single imported file. It parses the original content, applies import-schema defaults,
// substitutes import-inputs expressions in the raw content, extracts tools and markdown
// (handling the substituted vs. unsubstituted cases), and re-parses the possibly-modified
// frontmatter for use in subsequent field extractions.
//
// Side effects: acc.toolsBuilder, acc.markdownBuilder, acc.importPaths, acc.warnings,
// acc.importInputs.
//
// Returns: origFm (parsed from unsubstituted content, used for schema validation),
// fm (parsed from possibly-substituted content, used for all field extraction), and
// any error that should abort processing for this import.
func (acc *importAccumulator) prepareFrontmatter(content []byte, item importQueueItem, visited map[string]bool) (origFm, fm map[string]any, err error) {
	origContent := string(content)
	origParsed, origParseErr := parseOriginalFrontmatter(content, item.fullPath, origContent)
	origFm = frontmatterMapOrEmpty(origParsed, origParseErr)
	rawContent, wasSubstituted := acc.applyImportDefaultsToContent(origContent, origFm, item.inputs)
	acc.collectInlineSubAgentWarnings(item.importPath, rawContent, wasSubstituted, origParsed, origParseErr)
	toolsContent, err := acc.extractToolsContent(rawContent, item, visited, wasSubstituted)
	if err != nil {
		return nil, nil, err
	}
	acc.toolsBuilder.WriteString(toolsContent + "\n")
	importRelPath := computeImportRelPath(item.fullPath, item.importPath)
	if err := acc.trackRuntimeOrInlineImport(item.fullPath, importRelPath, rawContent, wasSubstituted); err != nil {
		return nil, nil, err
	}

	fm = parseFrontmatterForExtraction(rawContent, wasSubstituted, origFm)
	return origFm, fm, nil
}

func parseOriginalFrontmatter(content []byte, fullPath, origContent string) (*FrontmatterResult, error) {
	if strings.HasPrefix(fullPath, BuiltinPathPrefix) {
		return ExtractFrontmatterFromBuiltinFile(fullPath, content)
	}
	return ExtractFrontmatterFromContent(origContent)
}

func frontmatterMapOrEmpty(result *FrontmatterResult, parseErr error) map[string]any {
	if parseErr != nil {
		return make(map[string]any)
	}
	return result.Frontmatter
}

func (acc *importAccumulator) applyImportDefaultsToContent(origContent string, origFm, inputs map[string]any) (string, bool) {
	inputsWithDefaults := applyImportSchemaDefaultsFromFrontmatter(origFm, inputs)
	if len(inputsWithDefaults) == 0 {
		return origContent, false
	}
	maps.Copy(acc.importInputs, inputsWithDefaults)
	rawContent := substituteImportInputsInContent(origContent, inputsWithDefaults)
	return rawContent, rawContent != origContent
}

func (acc *importAccumulator) collectInlineSubAgentWarnings(importPath, rawContent string, wasSubstituted bool, origParsed *FrontmatterResult, origParseErr error) {
	var bodyForValidation string
	if !wasSubstituted && origParseErr == nil {
		bodyForValidation = origParsed.Markdown
	}
	agentWarnings := validateSubAgentFrontmatterWarnings(bodyForValidation, rawContent)
	for _, w := range agentWarnings {
		msg := fmt.Sprintf("import '%s': %s", importPath, w)
		acc.warnings = append(acc.warnings, msg)
		parserLog.Printf("%s", msg)
	}
}

func validateSubAgentFrontmatterWarnings(bodyForValidation, rawContent string) []string {
	if bodyForValidation != "" {
		return ValidateInlineSubAgentsInBody(bodyForValidation)
	}
	return ValidateInlineSubAgentsFrontmatter(rawContent)
}

func (acc *importAccumulator) extractToolsContent(rawContent string, item importQueueItem, visited map[string]bool, wasSubstituted bool) (string, error) {
	if wasSubstituted {
		toolsContent, err := extractToolsFromContent(rawContent)
		if err != nil {
			return "", fmt.Errorf("failed to extract tools from '%s': %w", item.fullPath, err)
		}
		return toolsContent, nil
	}
	toolsContent, err := processIncludedFileWithVisited(item.fullPath, item.sectionName, true, visited)
	if err != nil {
		return "", fmt.Errorf("failed to process imported file '%s': %w", item.fullPath, err)
	}
	return toolsContent, nil
}

func (acc *importAccumulator) trackRuntimeOrInlineImport(fullPath, importRelPath, rawContent string, wasSubstituted bool) error {
	if !wasSubstituted && !strings.HasPrefix(importRelPath, BuiltinPathPrefix) {
		acc.importPaths = append(acc.importPaths, importRelPath)
		acc.promptImports = append(acc.promptImports, PromptImportEntry{ImportPath: importRelPath})
		parserLog.Printf("Added import path for runtime-import: %s", importRelPath)
		return nil
	}
	if !wasSubstituted {
		return nil
	}
	parserLog.Printf("Import %s has substituted inputs - will be inlined for compile-time substitution", importRelPath)
	markdownContent, err := ExtractMarkdownContent(rawContent)
	if err != nil {
		return fmt.Errorf("failed to extract markdown from imported file '%s': %w", fullPath, err)
	}
	appendMarkdownWithSeparator(&acc.markdownBuilder, markdownContent)
	acc.promptImports = append(acc.promptImports, PromptImportEntry{Markdown: markdownContent})
	return nil
}

func appendMarkdownWithSeparator(builder *strings.Builder, markdownContent string) {
	if markdownContent == "" {
		return
	}
	builder.WriteString(markdownContent)
	if strings.HasSuffix(markdownContent, "\n\n") {
		return
	}
	if strings.HasSuffix(markdownContent, "\n") {
		builder.WriteString("\n")
		return
	}
	builder.WriteString("\n\n")
}

func parseFrontmatterForExtraction(rawContent string, wasSubstituted bool, origFm map[string]any) map[string]any {
	if !wasSubstituted {
		return origFm
	}
	reparsed, err := ExtractFrontmatterFromContent(rawContent)
	if err != nil {
		return make(map[string]any)
	}
	return reparsed.Frontmatter
}

// extractEngineConfig extracts engine-related settings from the imported frontmatter map
// and accumulates them. Engine configs with only `mcp` sub-keys (no `id` or `runtime`)
// are not counted as engine specifications — they carry MCP gateway settings only.
//
// Side effects: acc.engines, acc.mergedEngineMCPToolTimeout,
// acc.mergedEngineMCPSessionTimeout, acc.mergedEngineModel.
func (acc *importAccumulator) extractEngineConfig(fm map[string]any, fullPath string) {
	engineVal, hasEngine := fm["engine"]
	if !hasEngine {
		return
	}
	parserLog.Printf("Found engine config in import: %s", fullPath)

	switch v := engineVal.(type) {
	case string:
		// String engine (e.g. "copilot") — always counts as an engine spec.
		if engineJSON, merr := json.Marshal(v); merr == nil {
			acc.engines = append(acc.engines, string(engineJSON))
		}
	case map[string]any:
		// Object engine — extract engine.mcp.* settings first, then decide
		// whether to add to engines based on whether an engine ID is present.
		if mcpVal, hasMCP := v["mcp"]; hasMCP {
			if mcpMap, ok := mcpVal.(map[string]any); ok {
				// Extract tool-timeout (first-wins across all imports)
				if acc.mergedEngineMCPToolTimeout == "" {
					if ttStr, ok := mcpMap["tool-timeout"].(string); ok && ttStr != "" {
						acc.mergedEngineMCPToolTimeout = ttStr
						parserLog.Printf("Extracted engine.mcp.tool-timeout from import %s: %s", fullPath, ttStr)
					}
				}
				// Extract session-timeout (first-wins across all imports)
				if acc.mergedEngineMCPSessionTimeout == "" {
					if stStr, ok := mcpMap["session-timeout"].(string); ok && stStr != "" {
						acc.mergedEngineMCPSessionTimeout = stStr
						parserLog.Printf("Extracted engine.mcp.session-timeout from import %s: %s", fullPath, stStr)
					}
				}
			}
		}
		// Only add to engines list if this config specifies an actual engine
		// (i.e. it carries an 'id' or 'runtime' field). Configs with only
		// 'model' or 'mcp' settings are preferences, not engine selections,
		// and must not trigger the "multiple engine fields" validation error.
		_, hasID := v["id"]
		_, hasRuntime := v["runtime"]
		if hasID || hasRuntime {
			if engineJSON, merr := json.Marshal(v); merr == nil {
				acc.engines = append(acc.engines, string(engineJSON))
			}
		} else {
			// No engine ID or runtime — this is a model/MCP-only preference.
			// Extract the model hint (first-wins) so it can be applied to the
			// resolved engine after all imports are processed.
			if modelStr, ok := v["model"].(string); ok && modelStr != "" {
				if acc.mergedEngineModel == "" {
					acc.mergedEngineModel = modelStr
					parserLog.Printf("Extracted engine.model preference from import %s: %s", fullPath, modelStr)
				}
			}
		}
	default:
		// Unexpected type — marshal and add to preserve existing behavior.
		if engineJSON, merr := json.Marshal(engineVal); merr == nil {
			acc.engines = append(acc.engines, string(engineJSON))
		}
	}
}

// extractConfigFields extracts scalar and builder-based configuration fields from the
// frontmatter map and writes them into the appropriate accumulator builders and slices.
//
// Side effects: acc.mergedMaxTurns, acc.mergedMaxRuns, acc.mergedMaxEffectiveTokens,
// acc.mergedMaxDailyEffectiveTokens, acc.mcpServersBuilder,
// acc.safeOutputs, acc.mcpScripts, acc.stepsBuilder, acc.runtimesBuilder,
// acc.servicesBuilder, acc.networkBuilder, acc.permissionsBuilder,
// acc.secretMaskingBuilder.
func (acc *importAccumulator) extractConfigFields(fm map[string]any, fullPath string) {
	acc.extractFirstWinsJSONField(fm, fullPath, "max-turns", &acc.mergedMaxTurns)
	acc.extractFirstWinsJSONField(fm, fullPath, "max-runs", &acc.mergedMaxRuns)
	acc.extractFirstWinsJSONField(fm, fullPath, "max-effective-tokens", &acc.mergedMaxEffectiveTokens)
	acc.extractFirstWinsJSONField(fm, fullPath, "max-daily-ai-credits", &acc.mergedMaxDailyEffectiveTokens)
	acc.extractFirstWinsJSONField(fm, fullPath, "max-daily-effective-tokens", &acc.mergedMaxDailyEffectiveTokens)

	acc.appendJSONBuilderField(fm, "mcp-servers", "{}", &acc.mcpServersBuilder)
	acc.appendJSONSliceField(fm, "safe-outputs", "{}", &acc.safeOutputs)
	acc.appendJSONSliceField(fm, "mcp-scripts", "{}", &acc.mcpScripts)
	acc.appendYAMLBuilderField(fm, "steps", &acc.stepsBuilder)
	acc.appendJSONBuilderField(fm, "runtimes", "{}", &acc.runtimesBuilder)
	acc.appendYAMLBuilderField(fm, "services", &acc.servicesBuilder)
	acc.appendJSONBuilderField(fm, "network", "{}", &acc.networkBuilder)
	acc.appendJSONBuilderField(fm, "permissions", "{}", &acc.permissionsBuilder)
	acc.appendJSONBuilderField(fm, "secret-masking", "{}", &acc.secretMaskingBuilder)
}

func (acc *importAccumulator) extractFirstWinsJSONField(fm map[string]any, fullPath, field string, target *string) {
	if *target != "" {
		return
	}
	fieldJSON, err := extractFieldJSONFromMap(fm, field, "")
	if err != nil || fieldJSON == "" || fieldJSON == "null" {
		return
	}
	*target = fieldJSON
	parserLog.Printf("Extracted %s from import: %s", field, fullPath)
}

func (acc *importAccumulator) appendJSONBuilderField(fm map[string]any, field, emptyValue string, builder *strings.Builder) {
	content, err := extractFieldJSONFromMap(fm, field, emptyValue)
	if err != nil || content == "" || content == emptyValue {
		return
	}
	builder.WriteString(content + "\n")
}

func (acc *importAccumulator) appendJSONSliceField(fm map[string]any, field, emptyValue string, target *[]string) {
	content, err := extractFieldJSONFromMap(fm, field, emptyValue)
	if err != nil || content == "" || content == emptyValue {
		return
	}
	*target = append(*target, content)
}

func (acc *importAccumulator) appendYAMLBuilderField(fm map[string]any, field string, builder *strings.Builder) {
	content, err := extractYAMLFieldFromMap(fm, field)
	if err != nil || content == "" {
		return
	}
	builder.WriteString(content + "\n")
}

// extractActivationFields extracts activation and authentication-related fields from
// the frontmatter map: bots, skip-roles, skip-bots, skip-if-match, skip-if-no-match,
// on.github-token, on.github-app, top-level github-app, and checkout.
//
// Side effects: acc.bots, acc.botsSet, acc.skipRoles, acc.skipRolesSet, acc.skipBots,
// acc.skipBotsSet, acc.skipIfMatch, acc.skipIfNoMatch, acc.activationGitHubToken,
// acc.activationGitHubApp, acc.topLevelGitHubApp, acc.checkouts.
func (acc *importAccumulator) extractActivationFields(fm map[string]any, item importQueueItem) {
	acc.mergeBots(fm)
	acc.mergeSkipRoles(fm)
	acc.mergeSkipBots(fm)
	acc.extractActivationSkipMatchFields(fm, item.fullPath)
	acc.extractActivationGitHubToken(fm, item.fullPath)
	acc.extractActivationGitHubAppFields(fm, item.fullPath)
	acc.extractCheckoutField(fm, item.fullPath)
}

func (acc *importAccumulator) mergeBots(fm map[string]any) {
	mergeJSONStringListField(fm, "bots", "[]", acc.botsSet, &acc.bots, func(m map[string]any, field string) (string, error) {
		return extractFieldJSONFromMap(m, field, "[]")
	})
}

func (acc *importAccumulator) mergeSkipRoles(fm map[string]any) {
	mergeJSONStringListField(fm, "skip-roles", "[]", acc.skipRolesSet, &acc.skipRoles, extractOnSectionFieldFromMap)
}

func (acc *importAccumulator) mergeSkipBots(fm map[string]any) {
	mergeJSONStringListField(fm, "skip-bots", "[]", acc.skipBotsSet, &acc.skipBots, extractOnSectionFieldFromMap)
}

func mergeJSONStringListField(
	fm map[string]any,
	field, emptyValue string,
	seen map[string]bool,
	merged *[]string,
	extractor func(map[string]any, string) (string, error),
) {
	content, err := extractor(fm, field)
	if err != nil || content == "" || content == emptyValue {
		return
	}
	var imported []string
	if jsonErr := json.Unmarshal([]byte(content), &imported); jsonErr != nil {
		return
	}
	for _, value := range imported {
		if !seen[value] {
			seen[value] = true
			*merged = append(*merged, value)
		}
	}
}

func (acc *importAccumulator) extractActivationSkipMatchFields(fm map[string]any, fullPath string) {
	if acc.skipIfMatch == "" {
		if skipJSON, skipErr := extractOnSectionAnyFieldFromMap(fm, "skip-if-match"); skipErr == nil && skipJSON != "" && skipJSON != "null" {
			acc.skipIfMatch = skipJSON
			parserLog.Printf("Extracted on.skip-if-match from import: %s", fullPath)
		}
	}
	if acc.skipIfNoMatch == "" {
		if skipJSON, skipErr := extractOnSectionAnyFieldFromMap(fm, "skip-if-no-match"); skipErr == nil && skipJSON != "" && skipJSON != "null" {
			acc.skipIfNoMatch = skipJSON
			parserLog.Printf("Extracted on.skip-if-no-match from import: %s", fullPath)
		}
	}
}

func (acc *importAccumulator) extractActivationGitHubToken(fm map[string]any, fullPath string) {
	if acc.activationGitHubToken != "" {
		return
	}
	tokenJSON, tokenErr := extractOnSectionAnyFieldFromMap(fm, "github-token")
	if tokenErr != nil || tokenJSON == "" || tokenJSON == "null" {
		return
	}
	var token string
	if jsonErr := json.Unmarshal([]byte(tokenJSON), &token); jsonErr == nil && token != "" {
		acc.activationGitHubToken = token
		parserLog.Printf("Extracted on.github-token from import: %s", fullPath)
	}
}

func (acc *importAccumulator) extractActivationGitHubAppFields(fm map[string]any, fullPath string) {
	if acc.activationGitHubApp == "" {
		if appJSON, appErr := extractOnSectionAnyFieldFromMap(fm, "github-app"); appErr == nil {
			if validated := validateGitHubAppJSON(appJSON); validated != "" {
				acc.activationGitHubApp = validated
				parserLog.Printf("Extracted on.github-app from import: %s", fullPath)
			}
		}
	}
	if acc.topLevelGitHubApp == "" {
		if appJSON, appErr := extractFieldJSONFromMap(fm, "github-app", ""); appErr == nil {
			if validated := validateGitHubAppJSON(appJSON); validated != "" {
				acc.topLevelGitHubApp = validated
				parserLog.Printf("Extracted top-level github-app from import: %s", fullPath)
			}
		}
	}
}

func (acc *importAccumulator) extractCheckoutField(fm map[string]any, fullPath string) {
	checkoutJSON, checkoutErr := extractFieldJSONFromMap(fm, "checkout", "")
	if checkoutErr != nil || checkoutJSON == "" || checkoutJSON == "null" || checkoutJSON == "false" {
		return
	}
	acc.checkouts = append(acc.checkouts, checkoutJSON)
	parserLog.Printf("Extracted checkout from import: %s", fullPath)
}

// extractStepAndJobFields extracts step and job configuration fields from the frontmatter
// map. Environment variable conflict detection is performed: if the same env var is
// defined in two different imports, an error is returned.
//
// Side effects: acc.preStepsBuilder, acc.preAgentStepsBuilder, acc.postStepsBuilder,
// acc.jobsBuilder, acc.envBuilder, acc.envSources.
func (acc *importAccumulator) extractStepAndJobFields(fm map[string]any, importPath string) error {
	// Extract pre-steps (prepend in order).
	if preStepsContent, err := extractYAMLFieldFromMap(fm, "pre-steps"); err == nil && preStepsContent != "" {
		acc.preStepsBuilder.WriteString(preStepsContent + "\n")
	}

	// Extract pre-agent-steps (prepend in order).
	if preAgentStepsContent, err := extractYAMLFieldFromMap(fm, "pre-agent-steps"); err == nil && preAgentStepsContent != "" {
		acc.preAgentStepsBuilder.WriteString(preAgentStepsContent + "\n")
	}

	// Extract post-steps (append in order).
	if postStepsContent, err := extractYAMLFieldFromMap(fm, "post-steps"); err == nil && postStepsContent != "" {
		acc.postStepsBuilder.WriteString(postStepsContent + "\n")
	}

	// Extract jobs (append in order; merged into custom jobs map).
	if jobsContent, err := extractFieldJSONFromMap(fm, "jobs", "{}"); err == nil && jobsContent != "" && jobsContent != "{}" {
		acc.jobsBuilder.WriteString(jobsContent + "\n")
	}

	// Extract env (append in order; main workflow env takes precedence).
	// Conflicts between two imports are disallowed — only the main workflow may override imported vars.
	envContent, err := extractFieldJSONFromMap(fm, "env", "{}")
	if err == nil && envContent != "" && envContent != "{}" {
		var envMap map[string]any
		if jsonErr := json.Unmarshal([]byte(envContent), &envMap); jsonErr == nil {
			for key := range envMap {
				if existingSource, exists := acc.envSources[key]; exists {
					return fmt.Errorf("env variable %q is defined in multiple imports: %q and %q; remove the duplicate definition from one of the imports, or move it to the main workflow to override imported values", key, existingSource, importPath)
				}
				acc.envSources[key] = importPath
			}
			acc.envBuilder.WriteString(envContent + "\n")
		}
	}

	return nil
}

// extractFeatureAndObservabilityFields extracts labels, cache, feature flags, model
// aliases, the run-install-scripts flag, and observability configuration from the
// frontmatter map.
//
// Side effects: acc.labels, acc.labelsSet, acc.caches, acc.features, acc.models,
// acc.runInstallScripts, acc.observabilityConfigs.
func (acc *importAccumulator) extractFeatureAndObservabilityFields(fm map[string]any, fullPath string) {
	acc.mergeLabels(fm)
	acc.appendCacheField(fm)
	acc.appendFeaturesField(fm)
	acc.appendModelsField(fm)
	acc.extractRunInstallScripts(fm, fullPath)
	acc.appendObservabilityField(fm, fullPath)
}

func (acc *importAccumulator) mergeLabels(fm map[string]any) {
	mergeJSONStringListField(fm, "labels", "[]", acc.labelsSet, &acc.labels, func(m map[string]any, field string) (string, error) {
		return extractFieldJSONFromMap(m, field, "[]")
	})
}

func (acc *importAccumulator) appendCacheField(fm map[string]any) {
	if cacheContent, err := extractFieldJSONFromMap(fm, "cache", "{}"); err == nil && cacheContent != "" && cacheContent != "{}" {
		acc.caches = append(acc.caches, cacheContent)
	}
}

func (acc *importAccumulator) appendFeaturesField(fm map[string]any) {
	featuresContent, err := extractFieldJSONFromMap(fm, "features", "{}")
	if err != nil || featuresContent == "" || featuresContent == "{}" {
		return
	}
	var featuresMap map[string]any
	if jsonErr := json.Unmarshal([]byte(featuresContent), &featuresMap); jsonErr == nil {
		acc.features = append(acc.features, featuresMap)
		parserLog.Printf("Extracted features from import: %d entries", len(featuresMap))
	}
}

func (acc *importAccumulator) appendModelsField(fm map[string]any) {
	modelsContent, err := extractFieldJSONFromMap(fm, "models", "{}")
	if err != nil || modelsContent == "" || modelsContent == "{}" {
		return
	}
	var rawModels map[string]any
	if jsonErr := json.Unmarshal([]byte(modelsContent), &rawModels); jsonErr != nil {
		return
	}
	modelsMap := normalizeModelAliases(rawModels)
	if len(modelsMap) > 0 {
		acc.models = append(acc.models, modelsMap)
		parserLog.Printf("Extracted model aliases from import: %d entries", len(modelsMap))
	}
}

func normalizeModelAliases(rawModels map[string]any) map[string][]string {
	modelsMap := make(map[string][]string, len(rawModels))
	for k, v := range rawModels {
		patterns, ok := v.([]any)
		if !ok {
			continue
		}
		strs := make([]string, 0, len(patterns))
		for _, p := range patterns {
			if s, ok := p.(string); ok {
				strs = append(strs, s)
			}
		}
		modelsMap[k] = strs
	}
	return modelsMap
}

func (acc *importAccumulator) extractRunInstallScripts(fm map[string]any, fullPath string) {
	if acc.runInstallScripts {
		return
	}
	if hasNodeRuntimeRunInstallScripts(fm) {
		acc.runInstallScripts = true
		parserLog.Printf("Extracted runtimes.node.run-install-scripts: true from import: %s", fullPath)
	}
}

func hasNodeRuntimeRunInstallScripts(fm map[string]any) bool {
	runtimesAny, hasRuntimes := fm["runtimes"]
	if !hasRuntimes {
		return false
	}
	runtimesMap, ok := runtimesAny.(map[string]any)
	if !ok {
		return false
	}
	nodeAny, hasNode := runtimesMap["node"]
	if !hasNode {
		return false
	}
	nodeMap, ok := nodeAny.(map[string]any)
	if !ok {
		return false
	}
	rsAny, hasRS := nodeMap["run-install-scripts"]
	if !hasRS {
		return false
	}
	rsBool, ok := rsAny.(bool)
	return ok && rsBool
}

func (acc *importAccumulator) appendObservabilityField(fm map[string]any, fullPath string) {
	obsContent, obsErr := extractFieldJSONFromMap(fm, "observability", "{}")
	if obsErr != nil || obsContent == "" || obsContent == "{}" {
		return
	}
	acc.observabilityConfigs = append(acc.observabilityConfigs, obsContent)
	parserLog.Printf("Extracted observability from import: %s", fullPath)
}

// toImportsResult converts the accumulated state to a final ImportsResult.
// topologicalOrder is the result from topologicalSortImports.
func (acc *importAccumulator) toImportsResult(topologicalOrder []string) *ImportsResult {
	parserLog.Printf("Building ImportsResult: importedFiles=%d, importPaths=%d, engines=%d, bots=%d, labels=%d",
		len(topologicalOrder), len(acc.importPaths), len(acc.engines), len(acc.bots), len(acc.labels))
	return &ImportsResult{
		MergedTools:                   acc.toolsBuilder.String(),
		MergedMCPServers:              acc.mcpServersBuilder.String(),
		MergedEngines:                 acc.engines,
		MergedSafeOutputs:             acc.safeOutputs,
		MergedMCPScripts:              acc.mcpScripts,
		MergedMarkdown:                acc.markdownBuilder.String(),
		ImportPaths:                   acc.importPaths,
		PromptImports:                 acc.promptImports,
		MergedSteps:                   acc.stepsBuilder.String(),
		CopilotSetupSteps:             acc.copilotSetupStepsBuilder.String(),
		MergedPreSteps:                acc.preStepsBuilder.String(),
		MergedPreAgentSteps:           acc.preAgentStepsBuilder.String(),
		MergedRuntimes:                acc.runtimesBuilder.String(),
		MergedRunInstallScripts:       acc.runInstallScripts,
		MergedServices:                acc.servicesBuilder.String(),
		MergedNetwork:                 acc.networkBuilder.String(),
		MergedPermissions:             acc.permissionsBuilder.String(),
		MergedSecretMasking:           acc.secretMaskingBuilder.String(),
		MergedBots:                    acc.bots,
		MergedSkipRoles:               acc.skipRoles,
		MergedSkipBots:                acc.skipBots,
		MergedSkipIfMatch:             acc.skipIfMatch,
		MergedSkipIfNoMatch:           acc.skipIfNoMatch,
		MergedPostSteps:               acc.postStepsBuilder.String(),
		MergedLabels:                  acc.labels,
		MergedCaches:                  acc.caches,
		MergedJobs:                    acc.jobsBuilder.String(),
		MergedEnv:                     acc.envBuilder.String(),
		MergedEnvSources:              acc.envSources,
		MergedFeatures:                acc.features,
		MergedModels:                  acc.models,
		MergedObservability:           mergeObservabilityConfigs(acc.observabilityConfigs),
		ImportedFiles:                 topologicalOrder,
		AgentFile:                     acc.agentFile,
		AgentImportSpec:               acc.agentImportSpec,
		RepositoryImports:             acc.repositoryImports,
		ImportInputs:                  acc.importInputs,
		MergedActivationGitHubToken:   acc.activationGitHubToken,
		MergedActivationGitHubApp:     acc.activationGitHubApp,
		MergedTopLevelGitHubApp:       acc.topLevelGitHubApp,
		MergedCheckout:                strings.Join(acc.checkouts, "\n"),
		MergedEngineMCPToolTimeout:    acc.mergedEngineMCPToolTimeout,
		MergedEngineMCPSessionTimeout: acc.mergedEngineMCPSessionTimeout,
		MergedEngineModel:             acc.mergedEngineModel,
		MergedMaxTurns:                acc.mergedMaxTurns,
		MergedMaxRuns:                 acc.mergedMaxRuns,
		MergedMaxEffectiveTokens:      acc.mergedMaxEffectiveTokens,
		MergedMaxDailyEffectiveTokens: acc.mergedMaxDailyEffectiveTokens,
		Warnings:                      acc.warnings,
	}
}

// observabilityImportEndpoint is an endpoint entry used during import merging.
// Headers are kept as any (original format: string or map) so that the workflow
// package can later normalise both supported forms correctly.
type observabilityImportEndpoint struct {
	URL     string `json:"url"`
	Headers any    `json:"headers,omitempty"`
}

// extractOTLPEndpointsFromObsMap reads the `otlp.endpoint` field from a raw
// observability map and returns all endpoint entries as observabilityImportEndpoints.
// Supports string, object, and array forms of the endpoint field.
// Top-level `headers` is only applied to the backward-compat string endpoint form.
func extractOTLPEndpointsFromObsMap(obs map[string]any) []observabilityImportEndpoint {
	otlpAny, ok := obs["otlp"]
	if !ok {
		return nil
	}
	otlpMap, ok := otlpAny.(map[string]any)
	if !ok {
		return nil
	}

	endpointRaw := otlpMap["endpoint"]
	headersRaw := otlpMap["headers"] // only applies to the backward-compat string form

	var result []observabilityImportEndpoint
	switch ep := endpointRaw.(type) {
	case string:
		if ep != "" {
			entry := observabilityImportEndpoint{URL: ep}
			if headersRaw != nil {
				entry.Headers = headersRaw
			}
			result = append(result, entry)
		}
	case map[string]any:
		if url, _ := ep["url"].(string); url != "" {
			entry := observabilityImportEndpoint{URL: url}
			if h, hasH := ep["headers"]; hasH {
				entry.Headers = h
			}
			result = append(result, entry)
		}
	case []any:
		for _, item := range ep {
			itemMap, ok := item.(map[string]any)
			if !ok {
				continue
			}
			url, _ := itemMap["url"].(string)
			if url == "" {
				continue
			}
			entry := observabilityImportEndpoint{URL: url}
			if h, hasH := itemMap["headers"]; hasH {
				entry.Headers = h
			}
			result = append(result, entry)
		}
	}
	return result
}

// mergeObservabilityConfigs takes a slice of observability config JSON strings (one per
// import), extracts all OTLP endpoint entries from each (supporting string, object, and
// array forms), deduplicates by URL (first occurrence wins), and returns a single merged
// observability JSON string with all endpoints expressed as an array.  Custom OTLP
// attributes are also merged across imports (first occurrence wins per key).
// Returns "" when no valid endpoints or attributes are found.
func mergeObservabilityConfigs(configs []string) string {
	seen := make(map[string]bool)
	var allEndpoints []observabilityImportEndpoint
	mergedAttrs := make(map[string]string)
	var mergedGitHubApp map[string]any

	for i, cfgJSON := range configs {
		if cfgJSON == "" {
			continue
		}
		var obs map[string]any
		if err := json.Unmarshal([]byte(cfgJSON), &obs); err != nil {
			parserLog.Printf("Failed to unmarshal observability config from import %d during merge: %v", i, err)
			continue
		}
		for _, e := range extractOTLPEndpointsFromObsMap(obs) {
			if !seen[e.URL] {
				seen[e.URL] = true
				allEndpoints = append(allEndpoints, e)
			}
		}
		for k, v := range extractOTLPAttributesFromObsMap(obs) {
			if _, exists := mergedAttrs[k]; !exists {
				mergedAttrs[k] = v
			}
		}
		if mergedGitHubApp == nil {
			mergedGitHubApp = extractOTLPGitHubAppFromObsMap(obs)
		}
	}

	if len(allEndpoints) == 0 && len(mergedAttrs) == 0 && mergedGitHubApp == nil {
		return ""
	}

	// Produce a merged config with the endpoint field as an array so that the
	// workflow package's collectAllOTLPEndpoints handles it uniformly.  Include
	// any merged custom attributes so the orchestrator can propagate them.
	otlpMap := map[string]any{}
	if len(allEndpoints) > 0 {
		otlpMap["endpoint"] = allEndpoints
	}
	if len(mergedAttrs) > 0 {
		otlpMap["attributes"] = mergedAttrs
	}
	if mergedGitHubApp != nil {
		otlpMap["github-app"] = mergedGitHubApp
	}
	merged := map[string]any{"otlp": otlpMap}
	b, err := json.Marshal(merged)
	if err != nil {
		parserLog.Printf("Failed to marshal %d merged OTLP endpoints: %v", len(allEndpoints), err)
		return ""
	}
	return string(b)
}

func extractOTLPGitHubAppFromObsMap(obs map[string]any) map[string]any {
	if obs == nil {
		return nil
	}
	otlpAny, ok := obs["otlp"]
	if !ok {
		return nil
	}
	otlpMap, ok := otlpAny.(map[string]any)
	if !ok {
		return nil
	}
	githubAppAny, ok := otlpMap["github-app"]
	if !ok {
		return nil
	}
	githubAppMap, ok := githubAppAny.(map[string]any)
	if !ok {
		return nil
	}
	copyMap := make(map[string]any, len(githubAppMap))
	maps.Copy(copyMap, githubAppMap)
	return copyMap
}

// extractOTLPAttributesFromObsMap reads the custom OTLP attributes map from a
// raw observability section (as parsed from an import's frontmatter).  Only
// string values are accepted; non-string values are silently ignored.
// Returns nil when the field is absent or empty.
//
// Note: this intentionally duplicates the logic of
// workflow.extractOTLPCustomAttributesFromObsMap.  The parser package must not
// import the workflow package (circular-dependency risk), so the helper lives
// here as a local copy.  Both implementations must stay in sync.
func extractOTLPAttributesFromObsMap(obs map[string]any) map[string]string {
	if obs == nil {
		return nil
	}
	otlpAny, ok := obs["otlp"]
	if !ok {
		return nil
	}
	otlpMap, ok := otlpAny.(map[string]any)
	if !ok {
		return nil
	}
	attrsAny, ok := otlpMap["attributes"]
	if !ok {
		return nil
	}
	attrsMap, ok := attrsAny.(map[string]any)
	if !ok {
		return nil
	}
	result := make(map[string]string, len(attrsMap))
	for k, v := range attrsMap {
		if s, ok := v.(string); ok && k != "" {
			result[k] = s
		}
	}
	return result
}

// suitable for use in a {{#runtime-import ...}} macro.
//
// The rules are:
//  1. If fullPath contains "/.github/" (as a path component), trim everything before
//     and including the leading slash so the result starts with ".github/".
//     LastIndex is used so that repos named ".github" (e.g. path
//     "/root/.github/.github/workflows/file.md") resolve to the correct
//     ".github/workflows/…" segment rather than the first occurrence.
//  2. If fullPath already starts with ".github/" (a relative path) use it as-is.
//  3. Otherwise fall back to importPath (the original import spec).
func computeImportRelPath(fullPath, importPath string) string {
	normalizedFullPath := filepath.ToSlash(fullPath)
	if idx := strings.LastIndex(normalizedFullPath, "/.github/"); idx >= 0 {
		return normalizedFullPath[idx+1:] // +1 to skip the leading slash
	}
	if strings.HasPrefix(normalizedFullPath, ".github/") {
		return normalizedFullPath
	}
	return importPath
}

// validateGitHubAppJSON validates that a JSON-encoded GitHub App configuration has the required
// fields ((client-id or app-id) and private-key). Returns the input JSON if valid, or "" otherwise.
func validateGitHubAppJSON(appJSON string) string {
	if appJSON == "" || appJSON == "null" {
		return ""
	}
	var appMap map[string]any
	if err := json.Unmarshal([]byte(appJSON), &appMap); err != nil {
		return ""
	}
	_, hasClientID := appMap["client-id"]
	_, hasAppID := appMap["app-id"]
	if !hasClientID && !hasAppID {
		return ""
	}
	if _, hasKey := appMap["private-key"]; !hasKey {
		return ""
	}
	return appJSON
}

// validateWithImportSchema validates the provided 'with'/'inputs' values against
// the 'import-schema' declared in the imported workflow's frontmatter.
// It checks that:
//   - all required parameters declared in import-schema are present in 'with'
//   - no unknown parameters are provided (i.e., not declared in import-schema)
//   - provided values match the declared type (string, number, boolean, choice)
//   - choice values are within the allowed options list
//
// If the imported workflow has no 'import-schema', all provided 'with' values are
// accepted without validation (backward compatibility with 'inputs' form).
func validateWithImportSchema(inputs map[string]any, fm map[string]any, importPath string) error {
	rawSchema, hasSchema := fm["import-schema"]
	if !hasSchema {
		return nil
	}
	schemaMap, ok := rawSchema.(map[string]any)
	if !ok {
		return nil
	}
	if len(schemaMap) == 0 {
		return nil
	}

	// Check for unknown keys not declared in import-schema
	for key := range inputs {
		if _, declared := schemaMap[key]; !declared {
			return fmt.Errorf("import '%s': unknown 'with' input %q is not declared in the import-schema", importPath, key)
		}
	}

	// Check each declared schema field
	for paramName, paramDefRaw := range schemaMap {
		paramDef, _ := paramDefRaw.(map[string]any)

		// Check required parameters
		if req, _ := paramDef["required"].(bool); req {
			if _, provided := inputs[paramName]; !provided {
				return fmt.Errorf("import '%s': required 'with' input %q is missing (declared in import-schema)", importPath, paramName)
			}
		}

		value, provided := inputs[paramName]
		if !provided {
			continue
		}

		// Skip type validation when type is not specified
		declaredType, _ := paramDef["type"].(string)
		if declaredType == "" {
			continue
		}

		// Validate type
		if err := validateImportInputType(paramName, value, declaredType, paramDef, importPath); err != nil {
			return err
		}
	}
	return nil
}

// validateObjectInput validates a 'with' value of type object against the
// one-level deep 'properties' declared in the import-schema.
func validateObjectInput(name string, value any, paramDef map[string]any, importPath string) error {
	objMap, ok := value.(map[string]any)
	if !ok {
		return fmt.Errorf("import '%s': 'with' input %q must be an object (got %T)", importPath, name, value)
	}
	propsAny, hasProps := paramDef["properties"]
	if !hasProps {
		return nil // no schema for properties - accept any object
	}
	propsMap, ok := propsAny.(map[string]any)
	if !ok {
		return nil
	}
	// Check for unknown sub-keys
	for subKey := range objMap {
		if _, declared := propsMap[subKey]; !declared {
			return fmt.Errorf("import '%s': 'with' input %q has unknown property %q (not in import-schema)", importPath, name, subKey)
		}
	}
	// Validate each declared property
	for propName, propDefRaw := range propsMap {
		propDef, _ := propDefRaw.(map[string]any)
		// Check required sub-fields
		if req, _ := propDef["required"].(bool); req {
			if _, provided := objMap[propName]; !provided {
				return fmt.Errorf("import '%s': required property %q of 'with' input %q is missing", importPath, propName, name)
			}
		}
		subValue, provided := objMap[propName]
		if !provided {
			continue
		}
		propType, _ := propDef["type"].(string)
		if propType == "" {
			continue
		}
		qualifiedName := name + "." + propName
		if err := validateImportInputType(qualifiedName, subValue, propType, propDef, importPath); err != nil {
			return err
		}
	}
	return nil
}

// validateImportInputType checks that a single 'with' value matches the declared type.
func validateImportInputType(name string, value any, declaredType string, paramDef map[string]any, importPath string) error {
	switch declaredType {
	case "string":
		if _, ok := value.(string); !ok {
			return fmt.Errorf("import '%s': 'with' input %q must be a string (got %T)", importPath, name, value)
		}
	case "number":
		// Accept all numeric types that YAML parsers may produce
		switch value.(type) {
		case int, int8, int16, int32, int64,
			uint, uint8, uint16, uint32, uint64,
			float32, float64:
			// OK
		default:
			return fmt.Errorf("import '%s': 'with' input %q must be a number (got %T)", importPath, name, value)
		}
	case "boolean":
		if _, ok := value.(bool); !ok {
			return fmt.Errorf("import '%s': 'with' input %q must be a boolean (got %T)", importPath, name, value)
		}
	case "choice":
		strVal, ok := value.(string)
		if !ok {
			return fmt.Errorf("import '%s': 'with' input %q must be a string for choice type (got %T)", importPath, name, value)
		}
		if opts, hasOpts := paramDef["options"]; hasOpts {
			if optsList, ok := opts.([]any); ok {
				for _, opt := range optsList {
					if optStr, ok := opt.(string); ok && optStr == strVal {
						return nil
					}
				}
				return fmt.Errorf("import '%s': 'with' input %q value %q is not in the allowed options", importPath, name, strVal)
			}
		}
	case "array":
		arr, ok := value.([]any)
		if !ok {
			return fmt.Errorf("import '%s': 'with' input %q must be an array (got %T)", importPath, name, value)
		}
		// Validate item types if an 'items' schema is declared
		itemsDefRaw, hasItems := paramDef["items"]
		if !hasItems {
			return nil
		}
		itemsDef, _ := itemsDefRaw.(map[string]any)
		itemType, _ := itemsDef["type"].(string)
		if itemType == "" {
			return nil
		}
		for i, item := range arr {
			itemName := fmt.Sprintf("%s[%d]", name, i)
			if err := validateImportInputType(itemName, item, itemType, itemsDef, importPath); err != nil {
				return err
			}
		}
	case "object":
		return validateObjectInput(name, value, paramDef, importPath)
	}
	return nil
}

// applyImportSchemaDefaultsFromFrontmatter applies import-schema defaults from an
// already-parsed frontmatter map, avoiding a redundant YAML parse when the caller
// has already extracted the frontmatter. Returns a copy of inputs augmented with
// default values for any schema parameters declared with a "default" field but not
// present in the provided inputs map. Parameters already in inputs are left unchanged.
func applyImportSchemaDefaultsFromFrontmatter(frontmatter map[string]any, inputs map[string]any) map[string]any {
	rawSchema, ok := frontmatter["import-schema"]
	if !ok {
		return inputs
	}
	schemaMap, ok := rawSchema.(map[string]any)
	if !ok || len(schemaMap) == 0 {
		return inputs
	}

	// Check if there are any defaults to apply - avoid copying if not needed.
	hasDefaults := false
	for paramName, paramDefRaw := range schemaMap {
		if _, provided := inputs[paramName]; provided {
			continue
		}
		if paramDef, ok := paramDefRaw.(map[string]any); ok {
			if _, hasDefault := paramDef["default"]; hasDefault {
				hasDefaults = true
				break
			}
		}
	}
	if !hasDefaults {
		return inputs
	}

	// Copy the inputs map and add defaults for unprovided parameters.
	augmented := make(map[string]any, len(inputs))
	maps.Copy(augmented, inputs)
	for paramName, paramDefRaw := range schemaMap {
		if _, provided := augmented[paramName]; provided {
			continue
		}
		paramDef, ok := paramDefRaw.(map[string]any)
		if !ok {
			continue
		}
		if defaultVal, hasDefault := paramDef["default"]; hasDefault {
			augmented[paramName] = defaultVal
		}
	}
	return augmented
}

// importInputsExprRegex matches ${{ github.aw.import-inputs.<key> }} and
// ${{ github.aw.import-inputs.<key>.<subkey> }} expressions in raw content.
var importInputsExprRegex = regexp.MustCompile(`\$\{\{\s*github\.aw\.import-inputs\.([a-zA-Z0-9_-]+(?:\.[a-zA-Z0-9_-]+)?)\s*\}\}`)

// legacyInputsExprRegex matches ${{ github.aw.inputs.<key> }} (legacy form) in raw content.
var legacyInputsExprRegex = regexp.MustCompile(`\$\{\{\s*github\.aw\.inputs\.([a-zA-Z0-9_-]+)\s*\}\}`)

// substituteImportInputsInContent performs text-level substitution of
// ${{ github.aw.import-inputs.* }} and ${{ github.aw.inputs.* }} expressions
// in raw file content (including YAML frontmatter). This is called before YAML
// parsing so that array/object values serialised as JSON produce valid YAML.
func substituteImportInputsInContent(content string, inputs map[string]any) string {
	if len(inputs) == 0 {
		return content
	}

	result := legacyInputsExprRegex.ReplaceAllStringFunc(content, buildImportInputReplaceFunc(legacyInputsExprRegex, inputs))
	result = importInputsExprRegex.ReplaceAllStringFunc(result, buildImportInputReplaceFunc(importInputsExprRegex, inputs))
	return result
}

func buildImportInputReplaceFunc(regex *regexp.Regexp, inputs map[string]any) func(string) string {
	return func(match string) string {
		m := regex.FindStringSubmatch(match)
		if len(m) < 2 {
			return match
		}
		strVal, found := resolveImportInputPath(inputs, m[1])
		if found {
			return strVal
		}
		return match
	}
}

func resolveImportInputPath(inputs map[string]any, inputPath string) (string, bool) {
	value, ok := resolveImportInputValue(inputs, inputPath)
	if !ok {
		return "", false
	}
	return formatResolvedImportInputValue(value)
}

func resolveImportInputValue(inputs map[string]any, inputPath string) (any, bool) {
	top, sub, hasDot := strings.Cut(inputPath, ".")
	if !hasDot {
		value, ok := inputs[top]
		return value, ok
	}
	topVal, topOK := inputs[top]
	if !topOK {
		return nil, false
	}
	obj, isMap := topVal.(map[string]any)
	if !isMap {
		return nil, false
	}
	value, ok := obj[sub]
	return value, ok
}

func formatResolvedImportInputValue(value any) (string, bool) {
	switch v := value.(type) {
	case []any:
		return marshalImportInputValue(v)
	case map[string]any:
		return marshalImportInputValue(v)
	case nil:
		return "", false
	default:
		return formatReflectiveImportInputValue(v)
	}
}

func formatReflectiveImportInputValue(value any) (string, bool) {
	rv := reflect.ValueOf(value)
	switch rv.Kind() {
	case reflect.Slice:
		return marshalImportInputValue(normalizeSliceForImportInput(rv))
	case reflect.Map:
		return marshalImportInputValue(normalizeMapForImportInput(rv))
	default:
		return fmt.Sprintf("%v", value), true
	}
}

func marshalImportInputValue(value any) (string, bool) {
	b, err := json.Marshal(value)
	if err != nil {
		return "", false
	}
	return string(b), true
}

func normalizeSliceForImportInput(rv reflect.Value) []any {
	normalized := make([]any, rv.Len())
	for i := range rv.Len() {
		normalized[i] = rv.Index(i).Interface()
	}
	return normalized
}

func normalizeMapForImportInput(rv reflect.Value) map[string]any {
	keys := make([]string, 0, rv.Len())
	for _, key := range rv.MapKeys() {
		keys = append(keys, key.String())
	}
	sort.Strings(keys)
	normalized := make(map[string]any, rv.Len())
	for _, k := range keys {
		normalized[k] = rv.MapIndex(reflect.ValueOf(k)).Interface()
	}
	return normalized
}
