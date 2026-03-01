// Package parser provides functions for parsing and processing workflow markdown files.
// import_field_extractor.go implements field extraction from imported workflow files.
// It defines the importAccumulator struct that centralizes all result-building state
// and provides the extractAllImportFields method for processing a single imported file.
package parser

import (
	"encoding/json"
	"fmt"
	"strings"
)

// importAccumulator centralizes the builder/slice/set variables used during
// BFS import traversal. It accumulates results from all imported files and provides
// a method to convert the accumulated state into the final ImportsResult.
type importAccumulator struct {
	toolsBuilder             strings.Builder
	mcpServersBuilder        strings.Builder
	markdownBuilder          strings.Builder // Only used for imports WITH inputs (compile-time substitution)
	importPaths              []string        // Import paths for runtime-import macro generation
	stepsBuilder             strings.Builder
	copilotSetupStepsBuilder strings.Builder // Steps from copilot-setup-steps.yml (inserted at start)
	runtimesBuilder          strings.Builder
	servicesBuilder          strings.Builder
	networkBuilder           strings.Builder
	permissionsBuilder       strings.Builder
	secretMaskingBuilder     strings.Builder
	postStepsBuilder         strings.Builder
	jobsBuilder              strings.Builder // Jobs from imported YAML workflows
	engines                  []string
	safeOutputs              []string
	safeInputs               []string
	bots                     []string
	botsSet                  map[string]bool
	plugins                  []string
	pluginsSet               map[string]bool
	labels                   []string
	labelsSet                map[string]bool
	skipRoles                []string
	skipRolesSet             map[string]bool
	skipBots                 []string
	skipBotsSet              map[string]bool
	caches                   []string
	features                 []map[string]any
	agentFile                string
	agentImportSpec          string
	repositoryImports        []string
	importInputs             map[string]any
}

// newImportAccumulator creates and initializes a new importAccumulator.
// Maps (botsSet, pluginsSet, etc.) are explicitly initialized to prevent nil map panics
// during deduplication. Slices are left as nil, which is valid for append operations.
func newImportAccumulator() *importAccumulator {
	return &importAccumulator{
		botsSet:      make(map[string]bool),
		pluginsSet:   make(map[string]bool),
		labelsSet:    make(map[string]bool),
		skipRolesSet: make(map[string]bool),
		skipBotsSet:  make(map[string]bool),
		importInputs: make(map[string]any),
	}
}

// extractAllImportFields extracts all frontmatter fields from a single imported file
// and accumulates the results. Handles tools, engines, mcp-servers, safe-outputs,
// safe-inputs, steps, runtimes, services, network, permissions, secret-masking, bots,
// skip-roles, skip-bots, plugins, post-steps, labels, cache, and features.
func (acc *importAccumulator) extractAllImportFields(content []byte, item importQueueItem, visited map[string]bool) error {
	// Extract tools from imported file
	toolsContent, err := processIncludedFileWithVisited(item.fullPath, item.sectionName, true, visited)
	if err != nil {
		return fmt.Errorf("failed to process imported file '%s': %w", item.fullPath, err)
	}
	acc.toolsBuilder.WriteString(toolsContent + "\n")

	// Track import path for runtime-import macro generation (only if no inputs).
	// Imports with inputs must be inlined for compile-time substitution.
	// Extract relative path from repository root (from .github/ onwards).
	var importRelPath string
	if idx := strings.Index(item.fullPath, "/.github/"); idx >= 0 {
		importRelPath = item.fullPath[idx+1:] // +1 to skip the leading slash
	} else {
		// For files not under .github/, use the original import path
		importRelPath = item.importPath
	}

	if len(item.inputs) == 0 {
		// No inputs - use runtime-import macro
		acc.importPaths = append(acc.importPaths, importRelPath)
		log.Printf("Added import path for runtime-import: %s", importRelPath)
	} else {
		// Has inputs - must inline for compile-time substitution
		log.Printf("Import %s has inputs - will be inlined for compile-time substitution", importRelPath)
		markdownContent, err := processIncludedFileWithVisited(item.fullPath, item.sectionName, false, visited)
		if err != nil {
			return fmt.Errorf("failed to process markdown from imported file '%s': %w", item.fullPath, err)
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

	// Extract engines from imported file
	engineContent, err := extractFrontmatterField(string(content), "engine", "")
	if err == nil && engineContent != "" {
		acc.engines = append(acc.engines, engineContent)
	}

	// Extract mcp-servers from imported file
	mcpServersContent, err := extractFrontmatterField(string(content), "mcp-servers", "{}")
	if err == nil && mcpServersContent != "" && mcpServersContent != "{}" {
		acc.mcpServersBuilder.WriteString(mcpServersContent + "\n")
	}

	// Extract safe-outputs from imported file
	safeOutputsContent, err := extractFrontmatterField(string(content), "safe-outputs", "{}")
	if err == nil && safeOutputsContent != "" && safeOutputsContent != "{}" {
		acc.safeOutputs = append(acc.safeOutputs, safeOutputsContent)
	}

	// Extract safe-inputs from imported file
	safeInputsContent, err := extractFrontmatterField(string(content), "safe-inputs", "{}")
	if err == nil && safeInputsContent != "" && safeInputsContent != "{}" {
		acc.safeInputs = append(acc.safeInputs, safeInputsContent)
	}

	// Extract steps from imported file
	stepsContent, err := extractStepsFromContent(string(content))
	if err == nil && stepsContent != "" {
		acc.stepsBuilder.WriteString(stepsContent + "\n")
	}

	// Extract runtimes from imported file
	runtimesContent, err := extractFrontmatterField(string(content), "runtimes", "{}")
	if err == nil && runtimesContent != "" && runtimesContent != "{}" {
		acc.runtimesBuilder.WriteString(runtimesContent + "\n")
	}

	// Extract services from imported file
	servicesContent, err := extractServicesFromContent(string(content))
	if err == nil && servicesContent != "" {
		acc.servicesBuilder.WriteString(servicesContent + "\n")
	}

	// Extract network from imported file
	networkContent, err := extractFrontmatterField(string(content), "network", "{}")
	if err == nil && networkContent != "" && networkContent != "{}" {
		acc.networkBuilder.WriteString(networkContent + "\n")
	}

	// Extract permissions from imported file
	permissionsContent, err := ExtractPermissionsFromContent(string(content))
	if err == nil && permissionsContent != "" && permissionsContent != "{}" {
		acc.permissionsBuilder.WriteString(permissionsContent + "\n")
	}

	// Extract secret-masking from imported file
	secretMaskingContent, err := extractFrontmatterField(string(content), "secret-masking", "{}")
	if err == nil && secretMaskingContent != "" && secretMaskingContent != "{}" {
		acc.secretMaskingBuilder.WriteString(secretMaskingContent + "\n")
	}

	// Extract and merge bots from imported file (merge into set to avoid duplicates)
	botsContent, err := extractFrontmatterField(string(content), "bots", "[]")
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
	skipRolesContent, err := extractOnSectionField(string(content), "skip-roles")
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
	skipBotsContent, err := extractOnSectionField(string(content), "skip-bots")
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

	// Extract and merge plugins from imported file (merge into set to avoid duplicates).
	// Handles both simple string format and object format with MCP configs.
	pluginsContent, err := extractFrontmatterField(string(content), "plugins", "[]")
	if err == nil && pluginsContent != "" && pluginsContent != "[]" {
		var pluginsRaw []any
		if jsonErr := json.Unmarshal([]byte(pluginsContent), &pluginsRaw); jsonErr == nil {
			for _, plugin := range pluginsRaw {
				// Handle string format: "org/repo"
				if pluginStr, ok := plugin.(string); ok {
					if !acc.pluginsSet[pluginStr] {
						acc.pluginsSet[pluginStr] = true
						acc.plugins = append(acc.plugins, pluginStr)
					}
				} else if pluginObj, ok := plugin.(map[string]any); ok {
					// Handle object format: { "id": "org/repo", "mcp": {...} }
					if idVal, hasID := pluginObj["id"]; hasID {
						if pluginID, ok := idVal.(string); ok && !acc.pluginsSet[pluginID] {
							acc.pluginsSet[pluginID] = true
							acc.plugins = append(acc.plugins, pluginID)
							// Note: MCP configs from imports are currently not merged
							// They would need to be handled at a higher level in compiler_orchestrator_tools.go
						}
					}
				}
			}
		}
	}

	// Extract post-steps from imported file (append in order)
	postStepsContent, err := extractPostStepsFromContent(string(content))
	if err == nil && postStepsContent != "" {
		acc.postStepsBuilder.WriteString(postStepsContent + "\n")
	}

	// Extract labels from imported file (merge into set to avoid duplicates)
	labelsContent, err := extractFrontmatterField(string(content), "labels", "[]")
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
	cacheContent, err := extractFrontmatterField(string(content), "cache", "{}")
	if err == nil && cacheContent != "" && cacheContent != "{}" {
		acc.caches = append(acc.caches, cacheContent)
	}

	// Extract features from imported file (parse as map structure)
	featuresContent, err := extractFrontmatterField(string(content), "features", "{}")
	if err == nil && featuresContent != "" && featuresContent != "{}" {
		var featuresMap map[string]any
		if jsonErr := json.Unmarshal([]byte(featuresContent), &featuresMap); jsonErr == nil {
			acc.features = append(acc.features, featuresMap)
			log.Printf("Extracted features from import: %d entries", len(featuresMap))
		}
	}

	return nil
}

// toImportsResult converts the accumulated state to a final ImportsResult.
// topologicalOrder is the result from topologicalSortImports.
func (acc *importAccumulator) toImportsResult(topologicalOrder []string) *ImportsResult {
	return &ImportsResult{
		MergedTools:         acc.toolsBuilder.String(),
		MergedMCPServers:    acc.mcpServersBuilder.String(),
		MergedEngines:       acc.engines,
		MergedSafeOutputs:   acc.safeOutputs,
		MergedSafeInputs:    acc.safeInputs,
		MergedMarkdown:      acc.markdownBuilder.String(),
		ImportPaths:         acc.importPaths,
		MergedSteps:         acc.stepsBuilder.String(),
		CopilotSetupSteps:   acc.copilotSetupStepsBuilder.String(),
		MergedRuntimes:      acc.runtimesBuilder.String(),
		MergedServices:      acc.servicesBuilder.String(),
		MergedNetwork:       acc.networkBuilder.String(),
		MergedPermissions:   acc.permissionsBuilder.String(),
		MergedSecretMasking: acc.secretMaskingBuilder.String(),
		MergedBots:          acc.bots,
		MergedPlugins:       acc.plugins,
		MergedSkipRoles:     acc.skipRoles,
		MergedSkipBots:      acc.skipBots,
		MergedPostSteps:     acc.postStepsBuilder.String(),
		MergedLabels:        acc.labels,
		MergedCaches:        acc.caches,
		MergedJobs:          acc.jobsBuilder.String(),
		MergedFeatures:      acc.features,
		ImportedFiles:       topologicalOrder,
		AgentFile:           acc.agentFile,
		AgentImportSpec:     acc.agentImportSpec,
		RepositoryImports:   acc.repositoryImports,
		ImportInputs:        acc.importInputs,
	}
}
