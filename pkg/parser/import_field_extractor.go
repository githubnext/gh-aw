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
	"regexp"
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
	runtimesBuilder          strings.Builder
	servicesBuilder          strings.Builder
	networkBuilder           strings.Builder
	permissionsBuilder       strings.Builder
	secretMaskingBuilder     strings.Builder
	postStepsBuilder         strings.Builder
	jobsBuilder              strings.Builder   // Jobs from imported YAML workflows
	envBuilder               strings.Builder   // env vars from imported workflows (JSON, one object per line)
	envSources               map[string]string // env var name → source import path (for conflict detection and header listing)
	observabilityBuilder     strings.Builder   // observability config (first-wins for OTLP endpoint)
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
	caches                   []string
	features                 []map[string]any
	runInstallScripts        bool // true if any imported workflow sets run-install-scripts: true (global or node-level)
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
// skip-roles, skip-bots, pre-steps, post-steps, labels, cache, and features.
func (acc *importAccumulator) extractAllImportFields(content []byte, item importQueueItem, visited map[string]bool) error {
	log.Printf("Extracting all import fields: path=%s, section=%s, inputs=%d, content_size=%d bytes", item.fullPath, item.sectionName, len(item.inputs), len(content))

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
	var origFm map[string]any
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

	// Extract tools from imported file.
	// When content was modified by substitution (either explicit inputs or schema defaults),
	// we use the already-substituted content (to pick up any ${{ github.aw.import-inputs.* }}
	// expressions in the tools/mcp-servers frontmatter) rather than re-reading the original file.
	var toolsContent string
	if wasSubstituted {
		var err error
		toolsContent, err = extractToolsFromContent(rawContent)
		if err != nil {
			return fmt.Errorf("failed to extract tools from '%s': %w", item.fullPath, err)
		}
	} else {
		var err error
		toolsContent, err = processIncludedFileWithVisited(item.fullPath, item.sectionName, true, visited)
		if err != nil {
			return fmt.Errorf("failed to process imported file '%s': %w", item.fullPath, err)
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
		log.Printf("Added import path for runtime-import: %s", importRelPath)
	} else if wasSubstituted {
		// Content was modified by substitution - inline for compile-time substitution.
		// Extract markdown from the already-substituted content so that import-inputs
		// expressions embedded in the markdown body are resolved here.
		log.Printf("Import %s has substituted inputs - will be inlined for compile-time substitution", importRelPath)
		markdownContent, err := ExtractMarkdownContent(rawContent)
		if err != nil {
			return fmt.Errorf("failed to extract markdown from imported file '%s': %w", item.fullPath, err)
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
	var fm map[string]any
	if wasSubstituted {
		if reparsed, rerr := ExtractFrontmatterFromContent(rawContent); rerr == nil {
			fm = reparsed.Frontmatter
		} else {
			fm = make(map[string]any)
		}
	} else {
		fm = origFm
	}

	// Validate 'with'/'inputs' values against the imported workflow's 'import-schema' (if present).
	// Always use the ORIGINAL (unsubstituted) frontmatter for schema lookup so the import-schema
	// declaration itself is not affected by expression substitution.
	if _, hasSchema := origFm["import-schema"]; hasSchema {
		if err := validateWithImportSchema(item.inputs, origFm, item.importPath); err != nil {
			return err
		}
	}

	// Extract engines from imported file
	engineContent, err := extractFieldJSONFromMap(fm, "engine", "")
	if err == nil && engineContent != "" {
		log.Printf("Found engine config in import: %s", item.fullPath)
		acc.engines = append(acc.engines, engineContent)
	}

	// Extract mcp-servers from imported file
	mcpServersContent, err := extractFieldJSONFromMap(fm, "mcp-servers", "{}")
	if err == nil && mcpServersContent != "" && mcpServersContent != "{}" {
		acc.mcpServersBuilder.WriteString(mcpServersContent + "\n")
	}

	// Extract safe-outputs from imported file
	safeOutputsContent, err := extractFieldJSONFromMap(fm, "safe-outputs", "{}")
	if err == nil && safeOutputsContent != "" && safeOutputsContent != "{}" {
		acc.safeOutputs = append(acc.safeOutputs, safeOutputsContent)
	}

	// Extract mcp-scripts from imported file
	mcpScriptsContent, err := extractFieldJSONFromMap(fm, "mcp-scripts", "{}")
	if err == nil && mcpScriptsContent != "" && mcpScriptsContent != "{}" {
		acc.mcpScripts = append(acc.mcpScripts, mcpScriptsContent)
	}

	// Extract steps from imported file
	stepsContent, err := extractYAMLFieldFromMap(fm, "steps")
	if err == nil && stepsContent != "" {
		acc.stepsBuilder.WriteString(stepsContent + "\n")
	}

	// Extract runtimes from imported file
	runtimesContent, err := extractFieldJSONFromMap(fm, "runtimes", "{}")
	if err == nil && runtimesContent != "" && runtimesContent != "{}" {
		acc.runtimesBuilder.WriteString(runtimesContent + "\n")
	}

	// Extract services from imported file
	servicesContent, err := extractYAMLFieldFromMap(fm, "services")
	if err == nil && servicesContent != "" {
		acc.servicesBuilder.WriteString(servicesContent + "\n")
	}

	// Extract network from imported file
	networkContent, err := extractFieldJSONFromMap(fm, "network", "{}")
	if err == nil && networkContent != "" && networkContent != "{}" {
		acc.networkBuilder.WriteString(networkContent + "\n")
	}

	// Extract permissions from imported file
	permissionsContent, err := extractFieldJSONFromMap(fm, "permissions", "{}")
	if err == nil && permissionsContent != "" && permissionsContent != "{}" {
		acc.permissionsBuilder.WriteString(permissionsContent + "\n")
	}

	// Extract secret-masking from imported file
	secretMaskingContent, err := extractFieldJSONFromMap(fm, "secret-masking", "{}")
	if err == nil && secretMaskingContent != "" && secretMaskingContent != "{}" {
		acc.secretMaskingBuilder.WriteString(secretMaskingContent + "\n")
	}

	// Extract and merge bots from imported file (merge into set to avoid duplicates)
	botsContent, err := extractFieldJSONFromMap(fm, "bots", "[]")
	if err == nil && botsContent != "" && botsContent != "[]" {
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

	// Extract and merge skip-roles from imported file (merge into set to avoid duplicates)
	skipRolesContent, err := extractOnSectionFieldFromMap(fm, "skip-roles")
	if err == nil && skipRolesContent != "" && skipRolesContent != "[]" {
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

	// Extract and merge skip-bots from imported file (merge into set to avoid duplicates)
	skipBotsContent, err := extractOnSectionFieldFromMap(fm, "skip-bots")
	if err == nil && skipBotsContent != "" && skipBotsContent != "[]" {
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

	// Extract on.github-token from imported file (first-wins: only set if not yet populated)
	if acc.activationGitHubToken == "" {
		if tokenJSON, tokenErr := extractOnSectionAnyFieldFromMap(fm, "github-token"); tokenErr == nil && tokenJSON != "" && tokenJSON != "null" {
			var token string
			if jsonErr := json.Unmarshal([]byte(tokenJSON), &token); jsonErr == nil && token != "" {
				acc.activationGitHubToken = token
				log.Printf("Extracted on.github-token from import: %s", item.fullPath)
			}
		}
	}

	// Extract on.github-app from imported file (first-wins: only set if not yet populated)
	if acc.activationGitHubApp == "" {
		if appJSON, appErr := extractOnSectionAnyFieldFromMap(fm, "github-app"); appErr == nil {
			if validated := validateGitHubAppJSON(appJSON); validated != "" {
				acc.activationGitHubApp = validated
				log.Printf("Extracted on.github-app from import: %s", item.fullPath)
			}
		}
	}

	// Extract top-level github-app from imported file (first-wins: only set if not yet populated)
	if acc.topLevelGitHubApp == "" {
		if appJSON, appErr := extractFieldJSONFromMap(fm, "github-app", ""); appErr == nil {
			if validated := validateGitHubAppJSON(appJSON); validated != "" {
				acc.topLevelGitHubApp = validated
				log.Printf("Extracted top-level github-app from import: %s", item.fullPath)
			}
		}
	}

	// Extract checkout from imported file (append in order; main workflow's checkouts take precedence).
	// The checkout field may be a single object or an array of objects; store the raw JSON for
	// later parsing by the compiler.
	if checkoutJSON, checkoutErr := extractFieldJSONFromMap(fm, "checkout", ""); checkoutErr == nil && checkoutJSON != "" && checkoutJSON != "null" && checkoutJSON != "false" {
		acc.checkouts = append(acc.checkouts, checkoutJSON)
		log.Printf("Extracted checkout from import: %s", item.fullPath)
	}

	// Extract pre-steps from imported file (prepend in order)
	preStepsContent, err := extractYAMLFieldFromMap(fm, "pre-steps")
	if err == nil && preStepsContent != "" {
		acc.preStepsBuilder.WriteString(preStepsContent + "\n")
	}

	// Extract post-steps from imported file (append in order)
	postStepsContent, err := extractYAMLFieldFromMap(fm, "post-steps")
	if err == nil && postStepsContent != "" {
		acc.postStepsBuilder.WriteString(postStepsContent + "\n")
	}

	// Extract jobs from imported file (append in order; merged into custom jobs map)
	jobsContent, err := extractFieldJSONFromMap(fm, "jobs", "{}")
	if err == nil && jobsContent != "" && jobsContent != "{}" {
		acc.jobsBuilder.WriteString(jobsContent + "\n")
	}

	// Extract env from imported file (append in order; main workflow env takes precedence).
	// Conflicts between two imports are disallowed — only the main workflow may override imported vars.
	envContent, err := extractFieldJSONFromMap(fm, "env", "{}")
	if err == nil && envContent != "" && envContent != "{}" {
		var envMap map[string]any
		if jsonErr := json.Unmarshal([]byte(envContent), &envMap); jsonErr == nil {
			for key := range envMap {
				if existingSource, exists := acc.envSources[key]; exists {
					return fmt.Errorf("env variable %q is defined in multiple imports: %q and %q; remove the duplicate definition from one of the imports, or move it to the main workflow to override imported values", key, existingSource, item.importPath)
				}
				acc.envSources[key] = item.importPath
			}
			acc.envBuilder.WriteString(envContent + "\n")
		}
	}

	// Extract labels from imported file (merge into set to avoid duplicates)
	labelsContent, err := extractFieldJSONFromMap(fm, "labels", "[]")
	if err == nil && labelsContent != "" && labelsContent != "[]" {
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

	// Extract cache from imported file (append to list of caches)
	cacheContent, err := extractFieldJSONFromMap(fm, "cache", "{}")
	if err == nil && cacheContent != "" && cacheContent != "{}" {
		acc.caches = append(acc.caches, cacheContent)
	}

	// Extract features from imported file (parse as map structure)
	featuresContent, err := extractFieldJSONFromMap(fm, "features", "{}")
	if err == nil && featuresContent != "" && featuresContent != "{}" {
		var featuresMap map[string]any
		if jsonErr := json.Unmarshal([]byte(featuresContent), &featuresMap); jsonErr == nil {
			acc.features = append(acc.features, featuresMap)
			log.Printf("Extracted features from import: %d entries", len(featuresMap))
		}
	}

	// Extract run-install-scripts flag from imported file.
	// If global run-install-scripts: true is set OR if runtimes.node.run-install-scripts: true is set,
	// propagate to the accumulator (OR semantics: any import enabling it enables it overall).
	if !acc.runInstallScripts {
		if rsAny, hasRS := fm["run-install-scripts"]; hasRS {
			if rsBool, ok := rsAny.(bool); ok && rsBool {
				acc.runInstallScripts = true
				log.Printf("Extracted run-install-scripts: true from import: %s", item.fullPath)
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
								log.Printf("Extracted runtimes.node.run-install-scripts: true from import: %s", item.fullPath)
							}
						}
					}
				}
			}
		}
	}

	// Extract observability from imported file (first-wins: only set if not yet populated).
	// This ensures an imported workflow's OTLP config is visible to injectOTLPConfig even
	// when the main workflow's frontmatter does not contain an observability section.
	if acc.observabilityBuilder.Len() == 0 {
		obsContent, obsErr := extractFieldJSONFromMap(fm, "observability", "{}")
		if obsErr == nil && obsContent != "" && obsContent != "{}" {
			acc.observabilityBuilder.WriteString(obsContent)
			log.Printf("Extracted observability from import: %s", item.fullPath)
		}
	}

	return nil
}

// toImportsResult converts the accumulated state to a final ImportsResult.
// topologicalOrder is the result from topologicalSortImports.
func (acc *importAccumulator) toImportsResult(topologicalOrder []string) *ImportsResult {
	log.Printf("Building ImportsResult: importedFiles=%d, importPaths=%d, engines=%d, bots=%d, labels=%d",
		len(topologicalOrder), len(acc.importPaths), len(acc.engines), len(acc.bots), len(acc.labels))
	return &ImportsResult{
		MergedTools:                 acc.toolsBuilder.String(),
		MergedMCPServers:            acc.mcpServersBuilder.String(),
		MergedEngines:               acc.engines,
		MergedSafeOutputs:           acc.safeOutputs,
		MergedMCPScripts:            acc.mcpScripts,
		MergedMarkdown:              acc.markdownBuilder.String(),
		ImportPaths:                 acc.importPaths,
		MergedSteps:                 acc.stepsBuilder.String(),
		CopilotSetupSteps:           acc.copilotSetupStepsBuilder.String(),
		MergedPreSteps:              acc.preStepsBuilder.String(),
		MergedRuntimes:              acc.runtimesBuilder.String(),
		MergedRunInstallScripts:     acc.runInstallScripts,
		MergedServices:              acc.servicesBuilder.String(),
		MergedNetwork:               acc.networkBuilder.String(),
		MergedPermissions:           acc.permissionsBuilder.String(),
		MergedSecretMasking:         acc.secretMaskingBuilder.String(),
		MergedBots:                  acc.bots,
		MergedSkipRoles:             acc.skipRoles,
		MergedSkipBots:              acc.skipBots,
		MergedPostSteps:             acc.postStepsBuilder.String(),
		MergedLabels:                acc.labels,
		MergedCaches:                acc.caches,
		MergedJobs:                  acc.jobsBuilder.String(),
		MergedEnv:                   acc.envBuilder.String(),
		MergedEnvSources:            acc.envSources,
		MergedFeatures:              acc.features,
		MergedObservability:         acc.observabilityBuilder.String(),
		ImportedFiles:               topologicalOrder,
		AgentFile:                   acc.agentFile,
		AgentImportSpec:             acc.agentImportSpec,
		RepositoryImports:           acc.repositoryImports,
		ImportInputs:                acc.importInputs,
		MergedActivationGitHubToken: acc.activationGitHubToken,
		MergedActivationGitHubApp:   acc.activationGitHubApp,
		MergedTopLevelGitHubApp:     acc.topLevelGitHubApp,
		MergedCheckout:              strings.Join(acc.checkouts, "\n"),
	}
}

// computeImportRelPath returns the repository-root-relative path for a workflow file,
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
		switch v := value.(type) {
		case []any:
			if b, err := json.Marshal(v); err == nil {
				return string(b), true
			}
		case map[string]any:
			if b, err := json.Marshal(v); err == nil {
				return string(b), true
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
