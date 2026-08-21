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
	"strings"

	"github.com/github/gh-aw/pkg/constants"
)

// importAccumulator centralizes the builder/slice/set variables used during
// BFS import traversal. It accumulates results from all imported files and provides
// a method to convert the accumulated state into the final ImportsResult.
type importAccumulator struct {
	toolsBuilder               strings.Builder
	mcpServersBuilder          strings.Builder
	markdownBuilder            strings.Builder // imports with substituted inputs or schema defaults (compile-time substitution)
	importPaths                []string        // Import paths for runtime-import macro generation
	promptImports              []PromptImportEntry
	stepsBuilder               strings.Builder
	copilotSetupStepsBuilder   strings.Builder // Steps from copilot-setup-steps.yml (inserted at start)
	preStepsBuilder            strings.Builder
	preAgentStepsBuilder       strings.Builder
	runtimesBuilder            strings.Builder
	servicesBuilder            strings.Builder
	networkBuilder             strings.Builder
	permissionsBuilder         strings.Builder
	secretMaskingBuilder       strings.Builder
	postStepsBuilder           strings.Builder
	jobsBuilder                strings.Builder   // Jobs from imported YAML workflows
	envBuilder                 strings.Builder   // env vars from imported workflows (JSON, one object per line)
	envSources                 map[string]string // env var name → source import path (for conflict detection and header listing)
	observabilityConfigs       []string          // observability config JSON blobs from all imports (merged into endpoint array)
	engines                    []string
	plugins                    []string
	safeOutputs                []string
	mcpScripts                 []string
	bots                       []string
	botsSet                    map[string]bool
	labels                     []string
	labelsSet                  map[string]bool
	skipRoles                  []string
	skipRolesSet               map[string]bool
	skipBots                   []string
	skipBotsSet                map[string]bool
	skipIfMatch                string
	skipIfNoMatch              string
	ambientFolders             []string
	ambientFoldersSet          map[string]bool
	sandboxAgentMounts         []string
	sandboxAgentMountsSet      map[string]bool
	sandboxAgentRuntimeInstall *bool // false if any import sets sandbox.agent.runtime-install: false
	caches                     []string
	features                   []map[string]any
	models                     []map[string][]string // model alias maps from each imported file (appended in import order)
	modelPolicies              []map[string][]string // model policy sets from each imported file (appended in import order)
	modelCosts                 []map[string]any      // model pricing overlays from each imported file (appended in import order)
	defaultAiCreditsPricing    map[string]any        // first models.default-ai-credits-pricing object found in imports (first-wins)
	runInstallScripts          bool                  // true if any imported workflow sets runtimes.node.run-install-scripts: true
	agentFile                  string
	agentImportSpec            string
	repositoryImports          []string
	importInputs               map[string]any
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
	// First top-level max-turns / max-runs / max-ai-credits /
	// max-daily-ai-credits found across imports (first-wins).
	// Values are stored as JSON-encoded raw values so numeric literals and strings
	// round-trip consistently through import processing.
	mergedMaxTurns           string
	mergedMaxToolDenials     string
	mergedMaxRuns            string
	mergedMaxTurnCacheMisses string
	mergedMaxAICredits       string
	mergedMaxDailyAICredits  string
	// Union of excluded-env lists from all imported files (deduplicated).
	excludedEnv    []string
	excludedEnvSet map[string]bool
	// Best-effort sub-agent frontmatter warnings collected during BFS traversal.
	warnings []string
}

const (
	modelPolicyAllowedKey = "allowed"
	modelPolicyBlockedKey = "blocked"
)

// newImportAccumulator creates and initializes a new importAccumulator.
// Maps (botsSet, etc.) are explicitly initialized to prevent nil map panics
// during deduplication. Slices are left as nil, which is valid for append operations.
func newImportAccumulator() *importAccumulator {
	return &importAccumulator{
		botsSet:               make(map[string]bool),
		labelsSet:             make(map[string]bool),
		skipRolesSet:          make(map[string]bool),
		skipBotsSet:           make(map[string]bool),
		ambientFoldersSet:     make(map[string]bool),
		importInputs:          make(map[string]any),
		envSources:            make(map[string]string),
		sandboxAgentMountsSet: make(map[string]bool),
		excludedEnvSet:        make(map[string]bool),
	}
}

// extractAllImportFields extracts all frontmatter fields from a single imported file
// and accumulates the results. Handles tools, engines, mcp-servers, safe-outputs,
// mcp-scripts, steps, runtimes, services, network, permissions, secret-masking, bots,
// skip-roles, skip-bots, pre-steps, pre-agent-steps, post-steps, labels, cache, and features.
// The work is delegated to focused helper methods, each handling one logical phase.
func (acc *importAccumulator) extractAllImportFields(content []byte, item importQueueItem, visited map[string]struct{}) error {
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
func (acc *importAccumulator) prepareFrontmatter(content []byte, item importQueueItem, visited map[string]struct{}) (origFm, fm map[string]any, err error) {
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

func (acc *importAccumulator) extractToolsContent(rawContent string, item importQueueItem, visited map[string]struct{}, wasSubstituted bool) (string, error) {
	if wasSubstituted {
		toolsContent, err := extractToolsFromContent(rawContent)
		if err != nil {
			return "", fmt.Errorf("could not extract tools from import %q; ensure the imported content has a valid tools block: %w", item.fullPath, err)
		}
		return toolsContent, nil
	}
	toolsContent, err := processIncludedFileWithVisited(item.fullPath, item.sectionName, true, visited)
	if err != nil {
		return "", fmt.Errorf("could not process import %q for tools extraction; ensure the imported workflow path and section are valid: %w", item.fullPath, err)
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
		return fmt.Errorf("could not extract markdown from import %q; ensure the file contains valid markdown after frontmatter: %w", fullPath, err)
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

// toImportsResult converts the accumulated state to a final ImportsResult.
// topologicalOrder is the result from topologicalSortImports.
func (acc *importAccumulator) toImportsResult(topologicalOrder []string) *ImportsResult {
	parserLog.Printf("Building ImportsResult: importedFiles=%d, importPaths=%d, engines=%d, bots=%d, labels=%d",
		len(topologicalOrder), len(acc.importPaths), len(acc.engines), len(acc.bots), len(acc.labels))
	result := acc.buildImportsResult()
	result.ImportedFiles = topologicalOrder
	return result
}

// buildImportsResult constructs the ImportsResult from accumulated state, excluding
// ImportedFiles which is populated separately from the topological sort order.
func (acc *importAccumulator) buildImportsResult() *ImportsResult {
	return &ImportsResult{
		MergedTools:                      acc.toolsBuilder.String(),
		MergedMCPServers:                 acc.mcpServersBuilder.String(),
		MergedEngines:                    acc.engines,
		MergedPlugins:                    acc.plugins,
		MergedSafeOutputs:                acc.safeOutputs,
		MergedMCPScripts:                 acc.mcpScripts,
		MergedMarkdown:                   acc.markdownBuilder.String(),
		ImportPaths:                      acc.importPaths,
		PromptImports:                    acc.promptImports,
		MergedSteps:                      acc.stepsBuilder.String(),
		CopilotSetupSteps:                acc.copilotSetupStepsBuilder.String(),
		MergedPreSteps:                   acc.preStepsBuilder.String(),
		MergedPreAgentSteps:              acc.preAgentStepsBuilder.String(),
		MergedRuntimes:                   acc.runtimesBuilder.String(),
		MergedRunInstallScripts:          acc.runInstallScripts,
		MergedServices:                   acc.servicesBuilder.String(),
		MergedNetwork:                    acc.networkBuilder.String(),
		MergedSandboxAgentMounts:         acc.sandboxAgentMounts,
		MergedSandboxAgentRuntimeInstall: acc.sandboxAgentRuntimeInstall,
		MergedPermissions:                acc.permissionsBuilder.String(),
		MergedSecretMasking:              acc.secretMaskingBuilder.String(),
		MergedBots:                       acc.bots,
		MergedSkipRoles:                  acc.skipRoles,
		MergedSkipBots:                   acc.skipBots,
		MergedSkipIfMatch:                acc.skipIfMatch,
		MergedSkipIfNoMatch:              acc.skipIfNoMatch,
		MergedAmbientFolders:             acc.ambientFolders,
		MergedPostSteps:                  acc.postStepsBuilder.String(),
		MergedLabels:                     acc.labels,
		MergedCaches:                     acc.caches,
		MergedJobs:                       acc.jobsBuilder.String(),
		MergedEnv:                        acc.envBuilder.String(),
		MergedEnvSources:                 acc.envSources,
		MergedFeatures:                   acc.features,
		MergedModels:                     acc.models,
		MergedModelPolicies:              acc.modelPolicies,
		MergedModelCosts:                 acc.modelCosts,
		MergedDefaultAiCreditsPricing:    acc.defaultAiCreditsPricing,
		MergedObservability:              mergeObservabilityConfigs(acc.observabilityConfigs),
		AgentFile:                        acc.agentFile,
		AgentImportSpec:                  acc.agentImportSpec,
		RepositoryImports:                acc.repositoryImports,
		ImportInputs:                     acc.importInputs,
		MergedActivationGitHubToken:      acc.activationGitHubToken,
		MergedActivationGitHubApp:        acc.activationGitHubApp,
		MergedTopLevelGitHubApp:          acc.topLevelGitHubApp,
		MergedCheckout:                   strings.Join(acc.checkouts, "\n"),
		MergedEngineMCPToolTimeout:       acc.mergedEngineMCPToolTimeout,
		MergedEngineMCPSessionTimeout:    acc.mergedEngineMCPSessionTimeout,
		MergedEngineModel:                acc.mergedEngineModel,
		MergedMaxTurns:                   acc.mergedMaxTurns,
		MergedMaxToolDenials:             acc.mergedMaxToolDenials,
		MergedMaxRuns:                    acc.mergedMaxRuns,
		MergedMaxTurnCacheMisses:         acc.mergedMaxTurnCacheMisses,
		MergedMaxAICredits:               acc.mergedMaxAICredits,
		MergedMaxDailyAICredits:          acc.mergedMaxDailyAICredits,
		MergedExcludedEnv:                acc.excludedEnv,
		Warnings:                         acc.warnings,
	}
}

func computeImportRelPath(fullPath, importPath string) string {
	normalizedFullPath := filepath.ToSlash(fullPath)
	if idx := strings.LastIndex(normalizedFullPath, "/.github/"); idx >= 0 {
		return normalizedFullPath[idx+1:] // +1 to skip the leading slash
	}
	if strings.HasPrefix(normalizedFullPath, constants.GithubDir) {
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
