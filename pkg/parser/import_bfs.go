// Package parser provides functions for parsing and processing workflow markdown files.
// import_bfs.go implements the BFS traversal core for processing workflow imports.
// It orchestrates queue seeding, the BFS loop, queue item dispatch, and result assembly
// using the importAccumulator to collect results across all imported files.
package parser

import (
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"path"
	"strings"

	"github.com/goccy/go-yaml"
)

// processImportsFromFrontmatterWithManifestAndSource is the internal implementation that includes source tracking.
func processImportsFromFrontmatterWithManifestAndSource(frontmatter map[string]any, baseDir string, cache *ImportCache, workflowFilePath string, yamlContent string) (*ImportsResult, error) {
	// Check if imports field exists
	importsField, exists := frontmatter["imports"]
	if !exists {
		return &ImportsResult{}, nil
	}

	log.Print("Processing imports from frontmatter with recursive BFS")

	// Parse imports field - can be array of strings or objects with path and inputs
	var importSpecs []ImportSpec
	switch v := importsField.(type) {
	case []any:
		for _, item := range v {
			switch importItem := item.(type) {
			case string:
				// Simple string import
				importSpecs = append(importSpecs, ImportSpec{Path: importItem})
			case map[string]any:
				// Object import with path and optional inputs
				pathValue, hasPath := importItem["path"]
				if !hasPath {
					return nil, errors.New("import object must have a 'path' field")
				}
				pathStr, ok := pathValue.(string)
				if !ok {
					return nil, errors.New("import 'path' must be a string")
				}
				var inputs map[string]any
				if inputsValue, hasInputs := importItem["inputs"]; hasInputs {
					if inputsMap, ok := inputsValue.(map[string]any); ok {
						inputs = inputsMap
					} else {
						return nil, errors.New("import 'inputs' must be an object")
					}
				}
				importSpecs = append(importSpecs, ImportSpec{Path: pathStr, Inputs: inputs})
			default:
				return nil, errors.New("import item must be a string or an object with 'path' field")
			}
		}
	case []string:
		for _, s := range v {
			importSpecs = append(importSpecs, ImportSpec{Path: s})
		}
	default:
		return nil, errors.New("imports field must be an array of strings or objects")
	}

	if len(importSpecs) == 0 {
		return &ImportsResult{}, nil
	}

	log.Printf("Found %d direct imports to process", len(importSpecs))

	// Initialize BFS queue and visited set for cycle detection
	var queue []importQueueItem
	visited := make(map[string]bool)
	processedOrder := []string{} // Track processing order for manifest

	// Initialize result accumulator
	acc := newImportAccumulator()

	// Seed the queue with initial imports
	for _, importSpec := range importSpecs {
		importPath := importSpec.Path

		// Check if this is a repository-only import (owner/repo@ref without file path)
		if isRepositoryImport(importPath) {
			log.Printf("Detected repository import: %s", importPath)
			acc.repositoryImports = append(acc.repositoryImports, importPath)
			// Repository imports don't need further processing - they're handled at runtime
			continue
		}

		// Handle section references (file.md#Section)
		var filePath, sectionName string
		if strings.Contains(importPath, "#") {
			parts := strings.SplitN(importPath, "#", 2)
			filePath = parts[0]
			sectionName = parts[1]
		} else {
			filePath = importPath
		}

		// Resolve import path (supports workflowspec format)
		fullPath, err := ResolveIncludePath(filePath, baseDir, cache)
		if err != nil {
			// If we have source information, create a structured import error
			if workflowFilePath != "" && yamlContent != "" {
				line, column := findImportItemLocation(yamlContent, importPath)
				importErr := &ImportError{
					ImportPath: importPath,
					FilePath:   workflowFilePath,
					Line:       line,
					Column:     column,
					Cause:      err,
				}
				return nil, FormatImportError(importErr, yamlContent)
			}
			// Fallback to generic error if no source information
			return nil, fmt.Errorf("failed to resolve import '%s': %w", filePath, err)
		}

		// Validate that .lock.yml files are not imported
		if strings.HasSuffix(strings.ToLower(fullPath), ".lock.yml") {
			if workflowFilePath != "" && yamlContent != "" {
				line, column := findImportItemLocation(yamlContent, importPath)
				importErr := &ImportError{
					ImportPath: importPath,
					FilePath:   workflowFilePath,
					Line:       line,
					Column:     column,
					Cause:      errors.New("cannot import .lock.yml files. Lock files are compiled outputs from gh-aw. Import the source .md file instead"),
				}
				return nil, FormatImportError(importErr, yamlContent)
			}
			return nil, fmt.Errorf("cannot import .lock.yml files: '%s'. Lock files are compiled outputs from gh-aw. Import the source .md file instead", importPath)
		}

		// Track remote origin for workflowspec imports so nested relative imports
		// can be resolved against the same remote repository
		var origin *remoteImportOrigin
		if isWorkflowSpec(filePath) {
			origin = parseRemoteOrigin(filePath)
			if origin != nil {
				importLog.Printf("Tracking remote origin for workflowspec: %s/%s@%s", origin.Owner, origin.Repo, origin.Ref)
			}
		}

		// Check for duplicates before adding to queue
		if !visited[fullPath] {
			visited[fullPath] = true
			queue = append(queue, importQueueItem{
				importPath:   importPath,
				fullPath:     fullPath,
				sectionName:  sectionName,
				baseDir:      baseDir,
				inputs:       importSpec.Inputs,
				remoteOrigin: origin,
			})
			log.Printf("Queued import: %s (resolved to %s)", importPath, fullPath)
		} else {
			log.Printf("Skipping duplicate import: %s (already visited)", importPath)
		}
	}

	// BFS traversal: process queue until empty
	for len(queue) > 0 {
		// Dequeue first item (FIFO for BFS)
		item := queue[0]
		queue = queue[1:]

		log.Printf("Processing import from queue: %s", item.fullPath)

		// Merge inputs from this import into the aggregated inputs map
		maps.Copy(acc.importInputs, item.inputs)

		// Add to processing order
		processedOrder = append(processedOrder, item.importPath)

		// Check if this is a custom agent file (any markdown file under .github/agents)
		isAgentFile := strings.Contains(item.fullPath, "/.github/agents/") && strings.HasSuffix(strings.ToLower(item.fullPath), ".md")
		if isAgentFile {
			if acc.agentFile != "" {
				// Multiple agent files found - error
				log.Printf("Multiple agent files found: %s and %s", acc.agentFile, item.importPath)
				return nil, fmt.Errorf("multiple agent files found in imports: '%s' and '%s'. Only one agent file is allowed per workflow", acc.agentFile, item.importPath)
			}
			// Extract relative path from repository root (from .github/ onwards)
			// This ensures the path works at runtime with $GITHUB_WORKSPACE
			var importRelPath string
			if idx := strings.Index(item.fullPath, "/.github/"); idx >= 0 {
				acc.agentFile = item.fullPath[idx+1:] // +1 to skip the leading slash
				importRelPath = acc.agentFile
			} else {
				acc.agentFile = item.fullPath
				importRelPath = item.fullPath
			}
			log.Printf("Found agent file: %s (resolved to: %s)", item.fullPath, acc.agentFile)

			// Store the original import specification for remote agents
			// This allows runtime detection and .github folder merging
			acc.agentImportSpec = item.importPath
			log.Printf("Agent import specification: %s", acc.agentImportSpec)

			// Track import path for runtime-import macro generation (only if no inputs)
			// Imports with inputs must be inlined for compile-time substitution
			if len(item.inputs) == 0 {
				// No inputs - use runtime-import macro
				acc.importPaths = append(acc.importPaths, importRelPath)
				log.Printf("Added agent import path for runtime-import: %s", importRelPath)
			} else {
				// Has inputs - must inline for compile-time substitution
				log.Printf("Agent file has inputs - will be inlined instead of runtime-imported")

				// For agent files, extract markdown content (only when inputs are present)
				markdownContent, err := processIncludedFileWithVisited(item.fullPath, item.sectionName, false, visited)
				if err != nil {
					return nil, fmt.Errorf("failed to process markdown from agent file '%s': %w", item.fullPath, err)
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

			// Agent files don't have nested imports, skip to next item
			continue
		}

		// Check if this is a YAML workflow file (not .lock.yml)
		if isYAMLWorkflowFile(item.fullPath) {
			log.Printf("Detected YAML workflow file: %s", item.fullPath)

			// Process YAML workflow import to extract jobs/steps and services
			// Special case: copilot-setup-steps.yml returns steps YAML instead of jobs JSON
			jobsOrStepsData, servicesJSON, err := processYAMLWorkflowImport(item.fullPath)
			if err != nil {
				return nil, fmt.Errorf("failed to process YAML workflow '%s': %w", item.importPath, err)
			}

			// Check if this is copilot-setup-steps.yml (returns steps YAML instead of jobs JSON)
			if isCopilotSetupStepsFile(item.fullPath) {
				// For copilot-setup-steps.yml, jobsOrStepsData contains steps in YAML format
				// Add to CopilotSetupSteps instead of MergedSteps (inserted at start of workflow)
				if jobsOrStepsData != "" {
					acc.copilotSetupStepsBuilder.WriteString(jobsOrStepsData + "\n")
					log.Printf("Added copilot-setup steps (will be inserted at start): %s", item.importPath)
				}
			} else {
				// For regular YAML workflows, jobsOrStepsData contains jobs in JSON format
				if jobsOrStepsData != "" && jobsOrStepsData != "{}" {
					acc.jobsBuilder.WriteString(jobsOrStepsData + "\n")
					log.Printf("Added jobs from YAML workflow: %s", item.importPath)
				}
			}

			// Append services to merged services (services from YAML are already in JSON format)
			// Need to convert to YAML format for consistency with other services
			if servicesJSON != "" && servicesJSON != "{}" {
				// Convert JSON services to YAML format
				var services map[string]any
				if err := json.Unmarshal([]byte(servicesJSON), &services); err == nil {
					servicesWrapper := map[string]any{"services": services}
					servicesYAML, err := yaml.Marshal(servicesWrapper)
					if err == nil {
						acc.servicesBuilder.WriteString(string(servicesYAML) + "\n")
						log.Printf("Added services from YAML workflow: %s", item.importPath)
					}
				}
			}

			// YAML workflows don't have nested imports or markdown content, skip to next item
			continue
		}

		// Read the imported file to extract nested imports
		content, err := readFileFunc(item.fullPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read imported file '%s': %w", item.fullPath, err)
		}

		// Extract frontmatter from imported file to discover nested imports
		result, err := ExtractFrontmatterFromContent(string(content))
		if err != nil {
			// If frontmatter extraction fails, continue with other processing
			log.Printf("Failed to extract frontmatter from %s: %v", item.fullPath, err)
		} else if result.Frontmatter != nil {
			// Check for nested imports field
			if nestedImportsField, hasImports := result.Frontmatter["imports"]; hasImports {
				var nestedImports []string
				switch v := nestedImportsField.(type) {
				case []any:
					for _, nestedItem := range v {
						if str, ok := nestedItem.(string); ok {
							nestedImports = append(nestedImports, str)
						}
					}
				case []string:
					nestedImports = v
				}

				// Add nested imports to queue (BFS: append to end)
				// For local imports: resolve relative to the workflows directory (baseDir)
				// For remote imports: resolve relative to .github/workflows/ in the remote repo
				for _, nestedImportPath := range nestedImports {
					// Handle section references
					var nestedFilePath, nestedSectionName string
					if strings.Contains(nestedImportPath, "#") {
						parts := strings.SplitN(nestedImportPath, "#", 2)
						nestedFilePath = parts[0]
						nestedSectionName = parts[1]
					} else {
						nestedFilePath = nestedImportPath
					}

					// Determine the resolution path and propagate remote origin context
					resolvedPath := nestedFilePath
					var nestedRemoteOrigin *remoteImportOrigin

					if item.remoteOrigin != nil && !isWorkflowSpec(nestedFilePath) {
						// Parent was fetched from a remote repo and nested path is relative.
						// Convert to a workflowspec that resolves against the parent workflowspec's
						// base directory (e.g., gh-agent-workflows for gh-agent-workflows/gh-aw-workflows/file.md).
						cleanPath := path.Clean(strings.TrimPrefix(nestedFilePath, "./"))

						// Reject paths that escape the base directory (e.g., ../../../etc/passwd)
						if cleanPath == ".." || strings.HasPrefix(cleanPath, "../") || path.IsAbs(cleanPath) {
							return nil, fmt.Errorf("nested import '%s' from remote file '%s' escapes base directory", nestedFilePath, item.importPath)
						}

						// Use the parent's BasePath if available, otherwise default to .github/workflows
						basePath := item.remoteOrigin.BasePath
						if basePath == "" {
							basePath = ".github/workflows"
						}
						// Clean the basePath to ensure it's normalized
						basePath = path.Clean(basePath)

						resolvedPath = fmt.Sprintf("%s/%s/%s/%s@%s",
							item.remoteOrigin.Owner, item.remoteOrigin.Repo, basePath, cleanPath, item.remoteOrigin.Ref)
						// Parse a new remoteOrigin from resolvedPath to get the correct BasePath
						// for THIS file's nested imports, not the parent's BasePath
						nestedRemoteOrigin = parseRemoteOrigin(resolvedPath)
						importLog.Printf("Resolving nested import as remote workflowspec: %s -> %s (basePath=%s)", nestedFilePath, resolvedPath, basePath)
					} else if isWorkflowSpec(nestedFilePath) {
						// Nested import is itself a workflowspec - parse its remote origin
						nestedRemoteOrigin = parseRemoteOrigin(nestedFilePath)
						if nestedRemoteOrigin != nil {
							importLog.Printf("Nested workflowspec import detected: %s (origin: %s/%s@%s)", nestedFilePath, nestedRemoteOrigin.Owner, nestedRemoteOrigin.Repo, nestedRemoteOrigin.Ref)
						}
					}

					nestedFullPath, err := ResolveIncludePath(resolvedPath, baseDir, cache)
					if err != nil {
						// If we have source information for the parent workflow, create a structured error
						if workflowFilePath != "" && yamlContent != "" {
							// For nested imports, we should report the error at the location where the parent import is defined
							// since the nested import file itself might not have source location
							line, column := findImportItemLocation(yamlContent, item.importPath)
							importErr := &ImportError{
								ImportPath: nestedImportPath,
								FilePath:   workflowFilePath,
								Line:       line,
								Column:     column,
								Cause:      err,
							}
							return nil, FormatImportError(importErr, yamlContent)
						}
						// Fallback to generic error
						return nil, fmt.Errorf("failed to resolve nested import '%s' from '%s': %w", nestedFilePath, item.fullPath, err)
					}

					// Check for cycles - skip if already visited
					if !visited[nestedFullPath] {
						visited[nestedFullPath] = true
						queue = append(queue, importQueueItem{
							importPath:   nestedImportPath,
							fullPath:     nestedFullPath,
							sectionName:  nestedSectionName,
							baseDir:      baseDir, // Use original baseDir, not nestedBaseDir
							remoteOrigin: nestedRemoteOrigin,
						})
						log.Printf("Discovered nested import: %s -> %s (queued)", item.fullPath, nestedFullPath)
					} else {
						log.Printf("Skipping already visited nested import: %s (cycle detected)", nestedFullPath)
					}
				}
			}
		}

		// Extract all frontmatter fields from the imported file
		if err := acc.extractAllImportFields(content, item, visited); err != nil {
			return nil, err
		}
	}

	log.Printf("Completed BFS traversal. Processed %d imports in total", len(processedOrder))

	// Sort imports in topological order (roots first, dependencies before dependents)
	// Returns an error if a circular import is detected
	topologicalOrder, err := topologicalSortImports(processedOrder, baseDir, cache, workflowFilePath)
	if err != nil {
		return nil, err
	}
	log.Printf("Sorted imports in topological order: %v", topologicalOrder)

	return acc.toImportsResult(topologicalOrder), nil
}
