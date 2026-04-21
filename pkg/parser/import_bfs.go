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
	"path/filepath"
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

	// Parse imports field - can be array of strings or objects with path and inputs,
	// or an object with an 'aw' (agentic workflow paths) subfield.
	var importSpecs []ImportSpec
	switch v := importsField.(type) {
	case []any:
		specs, err := parseImportSpecsFromArray(v)
		if err != nil {
			return nil, err
		}
		importSpecs = specs
	case []string:
		for _, s := range v {
			importSpecs = append(importSpecs, ImportSpec{Path: s})
		}
	case map[string]any:
		// Object form: {aw: [...]}
		// Extract 'aw' subfield for agentic workflow imports.
		if awAny, hasAW := v["aw"]; hasAW {
			switch awVal := awAny.(type) {
			case []any:
				specs, err := parseImportSpecsFromArray(awVal)
				if err != nil {
					return nil, fmt.Errorf("imports.aw: %w", err)
				}
				importSpecs = specs
			case []string:
				for _, s := range awVal {
					importSpecs = append(importSpecs, ImportSpec{Path: s})
				}
			default:
				return nil, errors.New("imports.aw must be an array of strings or objects")
			}
		}
	default:
		return nil, errors.New("imports field must be an array or an object with an 'aw' subfield")
	}

	if len(importSpecs) == 0 {
		return &ImportsResult{}, nil
	}

	log.Printf("Found %d direct imports to process", len(importSpecs))

	// Initialize BFS queue and visited set for cycle detection
	var queue []importQueueItem
	visited := make(map[string]bool)
	// visitedInputs tracks the 'with' values for each visited path so that
	// conflicting re-imports (same file, different inputs) can be detected.
	visitedInputs := make(map[string]map[string]any)
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
			visitedInputs[fullPath] = importSpec.Inputs
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
			// Same file imported again - verify the 'with' values are identical
			if err := checkImportInputsConsistency(importPath, visitedInputs[fullPath], importSpec.Inputs); err != nil {
				return nil, err
			}
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

		// Extract frontmatter from the imported file's original content.
		// Use the process-level cache for builtin virtual files to avoid repeated YAML parsing.
		var result *FrontmatterResult
		if strings.HasPrefix(item.fullPath, BuiltinPathPrefix) {
			result, err = ExtractFrontmatterFromBuiltinFile(item.fullPath, content)
		} else {
			result, err = ExtractFrontmatterFromContent(string(content))
		}

		// Apply import-schema defaults before discovering nested imports, even when no
		// explicit 'with:' inputs were provided. This resolves ${{ github.aw.import-inputs.* }}
		// expressions that appear in the 'with' values of nested imports, enabling
		// multi-level workflow composition.
		// We reuse the already-parsed frontmatter to extract import-schema defaults,
		// avoiding a second YAML parse inside applyImportSchemaDefaults.
		if err == nil && result != nil {
			inputsWithDefaults := applyImportSchemaDefaultsFromFrontmatter(result.Frontmatter, item.inputs)
			if len(inputsWithDefaults) > 0 {
				origContent := string(content)
				substituted := substituteImportInputsInContent(origContent, inputsWithDefaults)
				// Only re-parse when substitution actually changed the content.
				// If no ${{ github.aw.import-inputs.* }} expressions were present,
				// the content is unchanged and a YAML reparse would be wasteful.
				if substituted != origContent {
					if reparse, rerr := ExtractFrontmatterFromContent(substituted); rerr == nil {
						result = reparse
					}
				}
			}
		}
		if err != nil {
			// If frontmatter extraction fails, continue with other processing
			log.Printf("Failed to extract frontmatter from %s: %v", item.fullPath, err)
		} else if result.Frontmatter != nil {
			// Check for nested imports field
			type nestedImportEntry struct {
				path   string
				inputs map[string]any
			}
			var nestedImports []nestedImportEntry
			if nestedImportsField, hasImports := result.Frontmatter["imports"]; hasImports {
				switch v := nestedImportsField.(type) {
				case []any:
					for _, nestedItem := range v {
						if str, ok := nestedItem.(string); ok {
							nestedImports = append(nestedImports, nestedImportEntry{path: str})
						} else if nestedMap, ok := nestedItem.(map[string]any); ok {
							// Handle uses/with or path/inputs syntax
							var nestedPath string
							if usesPath, ok := nestedMap["uses"].(string); ok {
								nestedPath = usesPath
							} else if pathVal, ok := nestedMap["path"].(string); ok {
								nestedPath = pathVal
							}
							if nestedPath != "" {
								var nestedInputs map[string]any
								if withVal, ok := nestedMap["with"].(map[string]any); ok {
									nestedInputs = withVal
								} else if inputsVal, ok := nestedMap["inputs"].(map[string]any); ok {
									nestedInputs = inputsVal
								}
								nestedImports = append(nestedImports, nestedImportEntry{path: nestedPath, inputs: nestedInputs})
							}
						}
					}
				case []string:
					for _, str := range v {
						nestedImports = append(nestedImports, nestedImportEntry{path: str})
					}
				}
			}

			// Add nested imports to queue (BFS: append to end)
			// For local imports: resolve relative to the workflows directory (baseDir)
			// For remote imports: resolve relative to .github/workflows/ in the remote repo
			for _, nestedEntry := range nestedImports {
				nestedImportPath := nestedEntry.path
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

				// Determine the base directory for resolving this nested import.
				// Paths that are explicitly local-to-parent are resolved relative to the
				// parent file's directory:
				//   - bare filenames with no directory separator ("serena.md")
				//   - explicit same-directory prefix ("./serena.md")
				// Paths with a multi-component directory prefix (e.g. "shared/foo.md") use
				// the original baseDir so that the absolute-from-workflows-root convention
				// is preserved.
				//
				// Note: workflow import paths always use forward slashes ("/") regardless of
				// OS because they originate from YAML frontmatter, not OS filesystem paths.
				isLocalRelative := !strings.Contains(resolvedPath, "/") || strings.HasPrefix(resolvedPath, "./")
				nestedBaseDir := baseDir
				if item.remoteOrigin == nil && !isWorkflowSpec(resolvedPath) && isLocalRelative {
					nestedBaseDir = filepath.Dir(item.fullPath)
				}

				nestedFullPath, err := ResolveIncludePath(resolvedPath, nestedBaseDir, cache)
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

				// Check for cycles/duplicates - skip if already visited
				if !visited[nestedFullPath] {
					visited[nestedFullPath] = true
					visitedInputs[nestedFullPath] = nestedEntry.inputs

					// Use a canonical importPath for the manifest and topological sort.
					// When the import was resolved from a non-standard base directory
					// (e.g. a sibling "./" or bare-filename import resolved relative to
					// the parent file's directory), the raw nestedImportPath ("./serena.md")
					// is ambiguous — it's not meaningful without knowing the parent's
					// directory.  Store a root-relative path instead so that the manifest
					// header and topological sort always reference unambiguous locations.
					canonicalImportPath := nestedImportPath
					if nestedRemoteOrigin == nil && nestedBaseDir != baseDir {
						if rel, err := filepath.Rel(baseDir, nestedFullPath); err == nil {
							canonicalImportPath = filepath.ToSlash(rel)
						}
					}

					queue = append(queue, importQueueItem{
						importPath:   canonicalImportPath,
						fullPath:     nestedFullPath,
						sectionName:  nestedSectionName,
						baseDir:      baseDir, // Use original baseDir, not nestedBaseDir
						inputs:       nestedEntry.inputs,
						remoteOrigin: nestedRemoteOrigin,
					})
					log.Printf("Discovered nested import: %s -> %s (queued)", item.fullPath, nestedFullPath)
				} else {
					// Same file re-imported from a different path - verify inputs match
					if err := checkImportInputsConsistency(nestedImportPath, visitedInputs[nestedFullPath], nestedEntry.inputs); err != nil {
						return nil, err
					}
					log.Printf("Skipping already visited nested import: %s (cycle detected)", nestedFullPath)
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

// parseImportSpecsFromArray parses an []any slice into a list of ImportSpec values.
// Each element must be a string (simple path) or a map with a required "path" or "uses"
// key and an optional "inputs" or "with" map. The "uses"/"with" form mirrors GitHub Actions
// reusable workflow syntax and is an alias for "path"/"inputs".
func parseImportSpecsFromArray(items []any) ([]ImportSpec, error) {
	var specs []ImportSpec
	for _, item := range items {
		switch importItem := item.(type) {
		case string:
			specs = append(specs, ImportSpec{Path: importItem})
		case map[string]any:
			// Accept "uses" as an alias for "path"
			pathValue, hasPath := importItem["path"]
			if !hasPath {
				pathValue, hasPath = importItem["uses"]
			}
			if !hasPath {
				return nil, errors.New("import object must have a 'path' or 'uses' field")
			}
			pathStr, ok := pathValue.(string)
			if !ok {
				return nil, errors.New("import 'path'/'uses' must be a string")
			}
			// Accept "with" as an alias for "inputs"
			var inputs map[string]any
			inputsValue, hasInputs := importItem["inputs"]
			if !hasInputs {
				inputsValue, hasInputs = importItem["with"]
			}
			if hasInputs {
				if inputsMap, ok := inputsValue.(map[string]any); ok {
					inputs = inputsMap
				} else {
					return nil, errors.New("import 'inputs'/'with' must be an object")
				}
			}
			specs = append(specs, ImportSpec{Path: pathStr, Inputs: inputs})
		default:
			return nil, errors.New("import item must be a string or an object with 'path'/'uses' field")
		}
	}
	return specs, nil
}

// checkImportInputsConsistency returns an error if a file that has already been imported
// is being imported again with different 'with' values. A workflow file can appear at most
// once in the import graph; when it appears multiple times the 'with' values must be identical.
func checkImportInputsConsistency(importPath string, existingInputs, newInputs map[string]any) error {
	if importInputsEqual(existingInputs, newInputs) {
		return nil
	}
	return fmt.Errorf(
		"import conflict: '%s' is imported more than once with different 'with' values.\n"+
			"An imported workflow can only be imported once per workflow.\n"+
			"  Previous 'with': %s\n"+
			"  New 'with':      %s",
		importPath,
		formatImportInputs(existingInputs),
		formatImportInputs(newInputs),
	)
}

// importInputsEqual reports whether two import input maps are deeply equal.
// Both nil and empty maps are considered equal (both represent "no inputs").
// Map key ordering does not affect the result.
func importInputsEqual(a, b map[string]any) bool {
	if len(a) == 0 && len(b) == 0 {
		return true
	}
	// encoding/json sorts map keys deterministically, making this a safe deep-equality check.
	aJSON, err := json.Marshal(a)
	if err != nil {
		return false
	}
	bJSON, err := json.Marshal(b)
	if err != nil {
		return false
	}
	return string(aJSON) == string(bJSON)
}

// formatImportInputs serializes an import input map to a compact JSON string for
// use in error messages. Returns "{}" if the map is nil or empty.
func formatImportInputs(inputs map[string]any) string {
	if len(inputs) == 0 {
		return "{}"
	}
	b, err := json.Marshal(inputs)
	if err != nil {
		return "<unserializable>"
	}
	return string(b)
}
