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
	runInstallScripts        bool                  // true if any imported workflow sets run-install-scripts: true (global or node-level)
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
	// First top-level max-runs / max-effective-tokens found across imports (first-wins).
	// Values are stored as JSON-encoded raw values so numeric literals and strings
	// round-trip consistently through import processing.
	mergedMaxRuns            string
	mergedMaxEffectiveTokens string
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
	// Parse frontmatter once from the original content. This parse is reused for
	// import-schema default extraction and schema validation, avoiding redundant YAML parsing.
	// For builtin files we use the process-level cache.
	origContent := string(content)
	var origParsed *FrontmatterResult
	var origParseErr error
	if strings.HasPrefix(item.fullPath, BuiltinPathPrefix) {
		origParsed, origParseErr = ExtractFrontmatterFromBuiltinFile(item.fullPath, content)
	} else {
		origParsed, origParseErr = ExtractFrontmatterFromContent(origContent)
	}
	if origParseErr == nil {
		origFm = origParsed.Frontmatter
	} else {
		origFm = make(map[string]any)
	}

	// Apply import-schema defaults before any YAML or markdown processing, even when no
	// explicit 'with:' inputs were provided by the importing workflow. This enables
	// ${{ github.aw.import-inputs.* }} expressions in the imported workflow's frontmatter
	// fields (engine, safe-outputs, tools, runtimes, etc.) and markdown body to resolve
	// to their declared default values rather than remaining as literal strings.
	// Array and map values are serialized as JSON so they produce valid YAML inline syntax.
	// We reuse the already-parsed frontmatter to avoid a second YAML parse.
	inputsWithDefaults := applyImportSchemaDefaultsFromFrontmatter(origFm, item.inputs)
	rawContent := origContent
	if len(inputsWithDefaults) > 0 {
		rawContent = substituteImportInputsInContent(origContent, inputsWithDefaults)
		// Add resolved defaults to acc.importInputs so the compile-time markdown
		// substitution pass (generatePrompt) also has access to them.
		maps.Copy(acc.importInputs, inputsWithDefaults)
	}
	wasSubstituted := rawContent != origContent

	// Best-effort: detect and validate inline sub-agent frontmatter in the imported file.
	// Unknown fields in sub-agent frontmatter blocks are surfaced as advisory warnings.
	// Validation failures never abort the import — they are accumulated for later display.
	//
	// When content was NOT substituted we reuse origParsed.Markdown (already parsed above)
	// to avoid a redundant YAML parse. When content was substituted we pass the full
	// substituted content so ValidateInlineSubAgentsFrontmatter can extract the body itself.
	var bodyForValidation string
	if !wasSubstituted && origParseErr == nil {
		bodyForValidation = origParsed.Markdown
	}
	var agentWarnings []string
	if bodyForValidation != "" {
		agentWarnings = ValidateInlineSubAgentsInBody(bodyForValidation)
	} else {
		agentWarnings = ValidateInlineSubAgentsFrontmatter(rawContent)
	}
	for _, w := range agentWarnings {
		msg := fmt.Sprintf("import '%s': %s", item.importPath, w)
		acc.warnings = append(acc.warnings, msg)
		parserLog.Printf("%s", msg)
	}

	// Extract tools from imported file.
	// When content was modified by substitution (either explicit inputs or schema defaults),
	// we use the already-substituted content (to pick up any ${{ github.aw.import-inputs.* }}
	// expressions in the tools/mcp-servers frontmatter) rather than re-reading the original file.
	var toolsContent string
	if wasSubstituted {
		toolsContent, err = extractToolsFromContent(rawContent)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to extract tools from '%s': %w", item.fullPath, err)
		}
	} else {
		toolsContent, err = processIncludedFileWithVisited(item.fullPath, item.sectionName, true, visited)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to process imported file '%s': %w", item.fullPath, err)
		}
	}
	acc.toolsBuilder.WriteString(toolsContent + "\n")

	// Track import path for runtime-import macro generation (only if no substitution happened).
	// Imports with substituted inputs (explicit or via schema defaults) must be inlined for
	// compile-time substitution so that ${{ github.aw.import-inputs.* }} expressions are resolved.
	// Builtin paths (@builtin:…) are pure configuration — they carry no user-visible
	// prompt content and must not generate runtime-import macros.
	importRelPath := computeImportRelPath(item.fullPath, item.importPath)
	if !wasSubstituted && !strings.HasPrefix(importRelPath, BuiltinPathPrefix) {
		// No substitution happened and not a builtin - use runtime-import macro
		acc.importPaths = append(acc.importPaths, importRelPath)
		parserLog.Printf("Added import path for runtime-import: %s", importRelPath)
	} else if wasSubstituted {
		// Content was modified by substitution - inline for compile-time substitution.
		// Extract markdown from the already-substituted content so that import-inputs
		// expressions embedded in the markdown body are resolved here.
		parserLog.Printf("Import %s has substituted inputs - will be inlined for compile-time substitution", importRelPath)
		markdownContent, merr := ExtractMarkdownContent(rawContent)
		if merr != nil {
			return nil, nil, fmt.Errorf("failed to extract markdown from imported file '%s': %w", item.fullPath, merr)
		}
		if markdownContent != "" {
			acc.markdownBuilder.WriteString(markdownContent)
			// Add blank line separator between imported files
			if !strings.HasSuffix(markdownContent, "\n\n") {
				if strings.HasSuffix(markdownContent, "\n") {
					acc.markdownBuilder.WriteString("\n")
				} else {
					acc.markdownBuilder.WriteString("\n\n")
				}
			}
		}
	}

	// Parse frontmatter from the (possibly substituted) content for field extraction.
	// All subsequent field extractions use this pre-parsed result.
	// When substitution changed the content, reparse from rawContent so that all
	// frontmatter fields (runtimes, mcp-servers, engine, safe-outputs, etc.) reflect
	// the resolved values. When content is unchanged we reuse origFm, which was already
	// parsed above — for builtin files the cache also applies.
	if wasSubstituted {
		if reparsed, rerr := ExtractFrontmatterFromContent(rawContent); rerr == nil {
			fm = reparsed.Frontmatter
		} else {
			fm = make(map[string]any)
		}
	} else {
		fm = origFm
	}
	return origFm, fm, nil
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
// Side effects: acc.mergedMaxRuns, acc.mergedMaxEffectiveTokens, acc.mcpServersBuilder,
// acc.safeOutputs, acc.mcpScripts, acc.stepsBuilder, acc.runtimesBuilder,
// acc.servicesBuilder, acc.networkBuilder, acc.permissionsBuilder,
// acc.secretMaskingBuilder.
func (acc *importAccumulator) extractConfigFields(fm map[string]any, fullPath string) {
	// Extract max-runs (first-wins across imports).
	if acc.mergedMaxRuns == "" {
		if maxRunsJSON, merr := extractFieldJSONFromMap(fm, "max-runs", ""); merr == nil &&
			maxRunsJSON != "" && maxRunsJSON != "null" {
			acc.mergedMaxRuns = maxRunsJSON
			parserLog.Printf("Extracted max-runs from import: %s", fullPath)
		}
	}

	// Extract max-effective-tokens (first-wins across imports).
	if acc.mergedMaxEffectiveTokens == "" {
		if maxTokensJSON, merr := extractFieldJSONFromMap(fm, "max-effective-tokens", ""); merr == nil &&
			maxTokensJSON != "" && maxTokensJSON != "null" {
			acc.mergedMaxEffectiveTokens = maxTokensJSON
			parserLog.Printf("Extracted max-effective-tokens from import: %s", fullPath)
		}
	}

	if mcpServersContent, err := extractFieldJSONFromMap(fm, "mcp-servers", "{}"); err == nil && mcpServersContent != "" && mcpServersContent != "{}" {
		acc.mcpServersBuilder.WriteString(mcpServersContent + "\n")
	}

	if safeOutputsContent, err := extractFieldJSONFromMap(fm, "safe-outputs", "{}"); err == nil && safeOutputsContent != "" && safeOutputsContent != "{}" {
		acc.safeOutputs = append(acc.safeOutputs, safeOutputsContent)
	}

	if mcpScriptsContent, err := extractFieldJSONFromMap(fm, "mcp-scripts", "{}"); err == nil && mcpScriptsContent != "" && mcpScriptsContent != "{}" {
		acc.mcpScripts = append(acc.mcpScripts, mcpScriptsContent)
	}

	if stepsContent, err := extractYAMLFieldFromMap(fm, "steps"); err == nil && stepsContent != "" {
		acc.stepsBuilder.WriteString(stepsContent + "\n")
	}

	if runtimesContent, err := extractFieldJSONFromMap(fm, "runtimes", "{}"); err == nil && runtimesContent != "" && runtimesContent != "{}" {
		acc.runtimesBuilder.WriteString(runtimesContent + "\n")
	}

	if servicesContent, err := extractYAMLFieldFromMap(fm, "services"); err == nil && servicesContent != "" {
		acc.servicesBuilder.WriteString(servicesContent + "\n")
	}

	if networkContent, err := extractFieldJSONFromMap(fm, "network", "{}"); err == nil && networkContent != "" && networkContent != "{}" {
		acc.networkBuilder.WriteString(networkContent + "\n")
	}

	if permissionsContent, err := extractFieldJSONFromMap(fm, "permissions", "{}"); err == nil && permissionsContent != "" && permissionsContent != "{}" {
		acc.permissionsBuilder.WriteString(permissionsContent + "\n")
	}

	if secretMaskingContent, err := extractFieldJSONFromMap(fm, "secret-masking", "{}"); err == nil && secretMaskingContent != "" && secretMaskingContent != "{}" {
		acc.secretMaskingBuilder.WriteString(secretMaskingContent + "\n")
	}
}

// extractActivationFields extracts activation and authentication-related fields from
// the frontmatter map: bots, skip-roles, skip-bots, skip-if-match, skip-if-no-match,
// on.github-token, on.github-app, top-level github-app, and checkout.
//
// Side effects: acc.bots, acc.botsSet, acc.skipRoles, acc.skipRolesSet, acc.skipBots,
// acc.skipBotsSet, acc.skipIfMatch, acc.skipIfNoMatch, acc.activationGitHubToken,
// acc.activationGitHubApp, acc.topLevelGitHubApp, acc.checkouts.
func (acc *importAccumulator) extractActivationFields(fm map[string]any, item importQueueItem) {
	// Extract and merge bots (merge into set to avoid duplicates).
	if botsContent, err := extractFieldJSONFromMap(fm, "bots", "[]"); err == nil && botsContent != "" && botsContent != "[]" {
		var importedBots []string
		if jsonErr := json.Unmarshal([]byte(botsContent), &importedBots); jsonErr == nil {
			for _, bot := range importedBots {
				if !acc.botsSet[bot] {
					acc.botsSet[bot] = true
					acc.bots = append(acc.bots, bot)
				}
			}
		}
	}

	// Extract and merge skip-roles (merge into set to avoid duplicates).
	if skipRolesContent, err := extractOnSectionFieldFromMap(fm, "skip-roles"); err == nil && skipRolesContent != "" && skipRolesContent != "[]" {
		var importedSkipRoles []string
		if jsonErr := json.Unmarshal([]byte(skipRolesContent), &importedSkipRoles); jsonErr == nil {
			for _, role := range importedSkipRoles {
				if !acc.skipRolesSet[role] {
					acc.skipRolesSet[role] = true
					acc.skipRoles = append(acc.skipRoles, role)
				}
			}
		}
	}

	// Extract and merge skip-bots (merge into set to avoid duplicates).
	if skipBotsContent, err := extractOnSectionFieldFromMap(fm, "skip-bots"); err == nil && skipBotsContent != "" && skipBotsContent != "[]" {
		var importedSkipBots []string
		if jsonErr := json.Unmarshal([]byte(skipBotsContent), &importedSkipBots); jsonErr == nil {
			for _, user := range importedSkipBots {
				if !acc.skipBotsSet[user] {
					acc.skipBotsSet[user] = true
					acc.skipBots = append(acc.skipBots, user)
				}
			}
		}
	}

	// Extract on.skip-if-match (first-wins).
	if acc.skipIfMatch == "" {
		if skipJSON, skipErr := extractOnSectionAnyFieldFromMap(fm, "skip-if-match"); skipErr == nil && skipJSON != "" && skipJSON != "null" {
			acc.skipIfMatch = skipJSON
			parserLog.Printf("Extracted on.skip-if-match from import: %s", item.fullPath)
		}
	}

	// Extract on.skip-if-no-match (first-wins).
	if acc.skipIfNoMatch == "" {
		if skipJSON, skipErr := extractOnSectionAnyFieldFromMap(fm, "skip-if-no-match"); skipErr == nil && skipJSON != "" && skipJSON != "null" {
			acc.skipIfNoMatch = skipJSON
			parserLog.Printf("Extracted on.skip-if-no-match from import: %s", item.fullPath)
		}
	}

	// Extract on.github-token (first-wins).
	if acc.activationGitHubToken == "" {
		if tokenJSON, tokenErr := extractOnSectionAnyFieldFromMap(fm, "github-token"); tokenErr == nil && tokenJSON != "" && tokenJSON != "null" {
			var token string
			if jsonErr := json.Unmarshal([]byte(tokenJSON), &token); jsonErr == nil && token != "" {
				acc.activationGitHubToken = token
				parserLog.Printf("Extracted on.github-token from import: %s", item.fullPath)
			}
		}
	}

	// Extract on.github-app (first-wins).
	if acc.activationGitHubApp == "" {
		if appJSON, appErr := extractOnSectionAnyFieldFromMap(fm, "github-app"); appErr == nil {
			if validated := validateGitHubAppJSON(appJSON); validated != "" {
				acc.activationGitHubApp = validated
				parserLog.Printf("Extracted on.github-app from import: %s", item.fullPath)
			}
		}
	}

	// Extract top-level github-app (first-wins).
	if acc.topLevelGitHubApp == "" {
		if appJSON, appErr := extractFieldJSONFromMap(fm, "github-app", ""); appErr == nil {
			if validated := validateGitHubAppJSON(appJSON); validated != "" {
				acc.topLevelGitHubApp = validated
				parserLog.Printf("Extracted top-level github-app from import: %s", item.fullPath)
			}
		}
	}

	// Extract checkout (append in order; main workflow's checkouts take precedence).
	// The checkout field may be a single object or an array of objects; store the raw JSON
	// for later parsing by the compiler.
	if checkoutJSON, checkoutErr := extractFieldJSONFromMap(fm, "checkout", ""); checkoutErr == nil && checkoutJSON != "" && checkoutJSON != "null" && checkoutJSON != "false" {
		acc.checkouts = append(acc.checkouts, checkoutJSON)
		parserLog.Printf("Extracted checkout from import: %s", item.fullPath)
	}
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
	// Extract labels (merge into set to avoid duplicates).
	if labelsContent, err := extractFieldJSONFromMap(fm, "labels", "[]"); err == nil && labelsContent != "" && labelsContent != "[]" {
		var importedLabels []string
		if jsonErr := json.Unmarshal([]byte(labelsContent), &importedLabels); jsonErr == nil {
			for _, label := range importedLabels {
				if !acc.labelsSet[label] {
					acc.labelsSet[label] = true
					acc.labels = append(acc.labels, label)
				}
			}
		}
	}

	// Extract cache (append to list of caches).
	if cacheContent, err := extractFieldJSONFromMap(fm, "cache", "{}"); err == nil && cacheContent != "" && cacheContent != "{}" {
		acc.caches = append(acc.caches, cacheContent)
	}

	// Extract features (parse as map structure).
	if featuresContent, err := extractFieldJSONFromMap(fm, "features", "{}"); err == nil && featuresContent != "" && featuresContent != "{}" {
		var featuresMap map[string]any
		if jsonErr := json.Unmarshal([]byte(featuresContent), &featuresMap); jsonErr == nil {
			acc.features = append(acc.features, featuresMap)
			parserLog.Printf("Extracted features from import: %d entries", len(featuresMap))
		}
	}

	// Extract model aliases (parse as map[string][]string structure).
	if modelsContent, err := extractFieldJSONFromMap(fm, "models", "{}"); err == nil && modelsContent != "" && modelsContent != "{}" {
		var rawModels map[string]any
		if jsonErr := json.Unmarshal([]byte(modelsContent), &rawModels); jsonErr == nil {
			modelsMap := make(map[string][]string, len(rawModels))
			for k, v := range rawModels {
				if patterns, ok := v.([]any); ok {
					strs := make([]string, 0, len(patterns))
					for _, p := range patterns {
						if s, ok := p.(string); ok {
							strs = append(strs, s)
						}
					}
					modelsMap[k] = strs
				}
			}
			if len(modelsMap) > 0 {
				acc.models = append(acc.models, modelsMap)
				parserLog.Printf("Extracted model aliases from import: %d entries", len(modelsMap))
			}
		}
	}

	// Extract run-install-scripts flag.
	// If global run-install-scripts: true is set OR if runtimes.node.run-install-scripts: true
	// is set, propagate to the accumulator (OR semantics: any import enabling it enables it overall).
	if !acc.runInstallScripts {
		if rsAny, hasRS := fm["run-install-scripts"]; hasRS {
			if rsBool, ok := rsAny.(bool); ok && rsBool {
				acc.runInstallScripts = true
				parserLog.Printf("Extracted run-install-scripts: true from import: %s", fullPath)
			}
		}
		// Also check runtimes.node.run-install-scripts
		if runtimesAny, hasRuntimes := fm["runtimes"]; hasRuntimes {
			if runtimesMap, ok := runtimesAny.(map[string]any); ok {
				if nodeAny, hasNode := runtimesMap["node"]; hasNode {
					if nodeMap, ok := nodeAny.(map[string]any); ok {
						if rsAny, hasRS := nodeMap["run-install-scripts"]; hasRS {
							if rsBool, ok := rsAny.(bool); ok && rsBool {
								acc.runInstallScripts = true
								parserLog.Printf("Extracted runtimes.node.run-install-scripts: true from import: %s", fullPath)
							}
						}
					}
				}
			}
		}
	}

	// Extract observability. All imports' OTLP endpoints are collected so that each import
	// can contribute endpoints for fan-out observability. Deduplication and merging into a
	// single array happens in toImportsResult.
	if obsContent, obsErr := extractFieldJSONFromMap(fm, "observability", "{}"); obsErr == nil && obsContent != "" && obsContent != "{}" {
		acc.observabilityConfigs = append(acc.observabilityConfigs, obsContent)
		parserLog.Printf("Extracted observability from import: %s", fullPath)
	}
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
		MergedMaxRuns:                 acc.mergedMaxRuns,
		MergedMaxEffectiveTokens:      acc.mergedMaxEffectiveTokens,
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
	}

	if len(allEndpoints) == 0 && len(mergedAttrs) == 0 {
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
	merged := map[string]any{"otlp": otlpMap}
	b, err := json.Marshal(merged)
	if err != nil {
		parserLog.Printf("Failed to marshal %d merged OTLP endpoints: %v", len(allEndpoints), err)
		return ""
	}
	return string(b)
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

	resolve := func(path string) (string, bool) {
		top, sub, hasDot := strings.Cut(path, ".")
		var value any
		var ok bool
		if !hasDot {
			value, ok = inputs[top]
		} else {
			// one-level deep: "obj.sub"
			topVal, topOK := inputs[top]
			if !topOK {
				return "", false
			}
			if obj, isMap := topVal.(map[string]any); isMap {
				value, ok = obj[sub]
			}
		}
		if !ok {
			return "", false
		}
		// Serialize the value: arrays and maps as JSON (valid YAML inline syntax),
		// scalars with fmt.Sprintf.
		// goccy/go-yaml may produce typed slices (e.g. []string) instead of []any,
		// so a reflection fallback handles those cases.
		switch v := value.(type) {
		case []any:
			if b, err := json.Marshal(v); err == nil {
				return string(b), true
			}
		case map[string]any:
			if b, err := json.Marshal(v); err == nil {
				return string(b), true
			}
		case nil:
			// Null import input — skip substitution to avoid panicking on reflect.ValueOf(nil).
			return "", false
		default:
			rv := reflect.ValueOf(v)
			switch rv.Kind() {
			case reflect.Slice:
				normalized := make([]any, rv.Len())
				for i := range rv.Len() {
					normalized[i] = rv.Index(i).Interface()
				}
				if b, err := json.Marshal(normalized); err == nil {
					return string(b), true
				}
			case reflect.Map:
				keys := make([]string, 0, rv.Len())
				for _, key := range rv.MapKeys() {
					keys = append(keys, key.String())
				}
				sort.Strings(keys)
				normalized := make(map[string]any, rv.Len())
				for _, k := range keys {
					normalized[k] = rv.MapIndex(reflect.ValueOf(k)).Interface()
				}
				if b, err := json.Marshal(normalized); err == nil {
					return string(b), true
				}
			}
		}
		return fmt.Sprintf("%v", value), true
	}

	replaceFunc := func(regex *regexp.Regexp) func(string) string {
		return func(match string) string {
			m := regex.FindStringSubmatch(match)
			if len(m) < 2 {
				return match
			}
			if strVal, found := resolve(m[1]); found {
				return strVal
			}
			return match
		}
	}

	result := legacyInputsExprRegex.ReplaceAllStringFunc(content, replaceFunc(legacyInputsExprRegex))
	result = importInputsExprRegex.ReplaceAllStringFunc(result, replaceFunc(importInputsExprRegex))
	return result
}
