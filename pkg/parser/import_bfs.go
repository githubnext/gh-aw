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

	"github.com/github/gh-aw/pkg/constants"
	"github.com/goccy/go-yaml"
)

// bfsNestedImportEntry holds a nested import path and its 'with' inputs discovered
// during BFS traversal of an imported file's frontmatter.
type bfsNestedImportEntry struct {
	path   string
	inputs map[string]any
}

// processImportsFromFrontmatterWithManifestAndSource is the internal implementation that includes source tracking.
func processImportsFromFrontmatterWithManifestAndSource(frontmatter map[string]any, baseDir string, cache *ImportCache, workflowFilePath string, yamlContent string) (*ImportsResult, error) {
	importsField, exists := frontmatter["imports"]
	if !exists {
		return &ImportsResult{}, nil
	}
	parserLog.Print("Processing imports from frontmatter with recursive BFS")
	importSpecs, err := parseImportFieldToSpecs(importsField)
	if err != nil {
		return nil, err
	}
	if len(importSpecs) == 0 {
		return &ImportsResult{}, nil
	}
	parserLog.Printf("Found %d direct imports to process", len(importSpecs))
	acc := newImportAccumulator()
	queue, visited, visitedInputs, err := seedInitialBFSQueue(importSpecs, baseDir, cache, workflowFilePath, yamlContent, acc)
	if err != nil {
		return nil, err
	}
	processedOrder, err := runBFSTraversal(queue, visited, visitedInputs, acc, baseDir, cache, workflowFilePath, yamlContent)
	if err != nil {
		return nil, err
	}
	parserLog.Printf("Completed BFS traversal. Processed %d imports in total", len(processedOrder))
	topologicalOrder, err := topologicalSortImports(processedOrder, baseDir, cache, workflowFilePath)
	if err != nil {
		return nil, err
	}
	parserLog.Printf("Sorted imports in topological order: %v", topologicalOrder)
	return acc.toImportsResult(topologicalOrder), nil
}

// parseImportFieldToSpecs parses the imports frontmatter field into a slice of ImportSpec.
// Accepts array-of-strings, array-of-objects, or object-with-aw-subfield forms.
func parseImportFieldToSpecs(importsField any) ([]ImportSpec, error) {
	switch v := importsField.(type) {
	case []any:
		return parseImportSpecsFromArray(v)
	case []string:
		specs := make([]ImportSpec, len(v))
		for i, s := range v {
			specs[i] = ImportSpec{Path: s}
		}
		return specs, nil
	case map[string]any:
		awAny, hasAW := v["aw"]
		if !hasAW {
			return nil, nil
		}
		switch awVal := awAny.(type) {
		case []any:
			specs, err := parseImportSpecsFromArray(awVal)
			if err != nil {
				return nil, fmt.Errorf("imports.aw: %w", err)
			}
			return specs, nil
		case []string:
			specs := make([]ImportSpec, len(awVal))
			for i, s := range awVal {
				specs[i] = ImportSpec{Path: s}
			}
			return specs, nil
		default:
			return nil, errors.New("imports.aw must be an array of strings or objects")
		}
	default:
		return nil, errors.New("imports field must be an array or an object with an 'aw' subfield")
	}
}

// seedInitialBFSQueue seeds the BFS queue with the initial set of import specs.
func seedInitialBFSQueue(importSpecs []ImportSpec, baseDir string, cache *ImportCache, workflowFilePath, yamlContent string, acc *importAccumulator) ([]importQueueItem, map[string]bool, map[string]map[string]any, error) {
	queue := make([]importQueueItem, 0, len(importSpecs))
	visited := make(map[string]bool)
	visitedInputs := make(map[string]map[string]any)
	for _, spec := range importSpecs {
		if err := processInitialImportSpec(spec, baseDir, cache, workflowFilePath, yamlContent, acc, &queue, visited, visitedInputs); err != nil {
			return nil, nil, nil, err
		}
	}
	return queue, visited, visitedInputs, nil
}

// processInitialImportSpec resolves, validates, and enqueues a single top-level import spec.
func processInitialImportSpec(spec ImportSpec, baseDir string, cache *ImportCache, workflowFilePath, yamlContent string, acc *importAccumulator, queue *[]importQueueItem, visited map[string]bool, visitedInputs map[string]map[string]any) error {
	importPath := spec.Path
	if isRepositoryImport(importPath) {
		parserLog.Printf("Detected repository import: %s", importPath)
		acc.repositoryImports = append(acc.repositoryImports, importPath)
		return nil
	}
	var filePath, sectionName string
	if strings.Contains(importPath, "#") {
		parts := strings.SplitN(importPath, "#", 2)
		filePath, sectionName = parts[0], parts[1]
	} else {
		filePath = importPath
	}
	fullPath, err := ResolveIncludePath(filePath, baseDir, cache)
	if err != nil {
		if workflowFilePath != "" && yamlContent != "" {
			line, column := findImportItemLocation(yamlContent, importPath)
			importErr := &ImportError{ImportPath: importPath, FilePath: workflowFilePath, Line: line, Column: column, Cause: err}
			return FormatImportError(importErr, yamlContent)
		}
		return fmt.Errorf("failed to resolve import '%s': %w", filePath, err)
	}
	if strings.HasSuffix(strings.ToLower(fullPath), ".lock.yml") {
		cause := errors.New("cannot import .lock.yml files. Lock files are compiled outputs from gh-aw. Import the source .md file instead")
		if workflowFilePath != "" && yamlContent != "" {
			line, column := findImportItemLocation(yamlContent, importPath)
			importErr := &ImportError{ImportPath: importPath, FilePath: workflowFilePath, Line: line, Column: column, Cause: cause}
			return FormatImportError(importErr, yamlContent)
		}
		return fmt.Errorf("cannot import .lock.yml files: '%s'. Lock files are compiled outputs from gh-aw. Import the source .md file instead", importPath)
	}
	var origin *remoteImportOrigin
	if isWorkflowSpec(filePath) {
		origin = parseRemoteOrigin(filePath)
		if origin != nil {
			importLog.Printf("Tracking remote origin for workflowspec: %s/%s@%s", origin.Owner, origin.Repo, origin.Ref)
		}
	}
	if !visited[fullPath] {
		visited[fullPath] = true
		visitedInputs[fullPath] = spec.Inputs
		*queue = append(*queue, importQueueItem{importPath: importPath, fullPath: fullPath, sectionName: sectionName, baseDir: baseDir, inputs: spec.Inputs, remoteOrigin: origin})
		parserLog.Printf("Queued import: %s (resolved to %s)", importPath, fullPath)
	} else {
		if err := checkImportInputsConsistency(importPath, visitedInputs[fullPath], spec.Inputs); err != nil {
			return err
		}
		parserLog.Printf("Skipping duplicate import: %s (already visited)", importPath)
	}
	return nil
}

// runBFSTraversal processes the BFS queue until empty, returning the ordered list of processed imports.
func runBFSTraversal(queue []importQueueItem, visited map[string]bool, visitedInputs map[string]map[string]any, acc *importAccumulator, baseDir string, cache *ImportCache, workflowFilePath, yamlContent string) ([]string, error) {
	var processedOrder []string
	for len(queue) > 0 {
		item := queue[0]
		queue = queue[1:]
		parserLog.Printf("Processing import from queue: %s", item.fullPath)
		maps.Copy(acc.importInputs, item.inputs)
		processedOrder = append(processedOrder, item.importPath)
		var err error
		queue, err = processBFSQueueItem(item, queue, visited, visitedInputs, acc, baseDir, cache, workflowFilePath, yamlContent)
		if err != nil {
			return nil, err
		}
	}
	return processedOrder, nil
}

// processBFSQueueItem dispatches a single BFS queue item to the appropriate handler.
func processBFSQueueItem(item importQueueItem, queue []importQueueItem, visited map[string]bool, visitedInputs map[string]map[string]any, acc *importAccumulator, baseDir string, cache *ImportCache, workflowFilePath, yamlContent string) ([]importQueueItem, error) {
	fullPathSlash := filepath.ToSlash(item.fullPath)
	if strings.Contains(fullPathSlash, "/.github/agents/") && strings.HasSuffix(strings.ToLower(fullPathSlash), ".md") {
		if err := handleAgentFileBFSItem(item, acc, visited); err != nil {
			return nil, err
		}
		return queue, nil
	}
	if isYAMLWorkflowFile(item.fullPath) {
		if err := handleYAMLWorkflowBFSItem(item, acc); err != nil {
			return nil, err
		}
		return queue, nil
	}
	newItems, err := processMarkdownBFSItem(item, visited, visitedInputs, acc, baseDir, cache, workflowFilePath, yamlContent)
	if err != nil {
		return nil, err
	}
	return append(queue, newItems...), nil
}

// handleAgentFileBFSItem processes a custom agent file (.github/agents/*.md) BFS queue item.
func handleAgentFileBFSItem(item importQueueItem, acc *importAccumulator, visited map[string]bool) error {
	if acc.agentFile != "" {
		parserLog.Printf("Multiple agent files found: %s and %s", acc.agentFile, item.importPath)
		return fmt.Errorf("multiple agent files found in imports: '%s' and '%s'. Only one agent file is allowed per workflow", acc.agentFile, item.importPath)
	}
	fullPathSlash := filepath.ToSlash(item.fullPath)
	var importRelPath string
	if idx := strings.Index(fullPathSlash, "/.github/"); idx >= 0 {
		acc.agentFile = fullPathSlash[idx+1:] // +1 to skip the leading slash
		importRelPath = acc.agentFile
	} else {
		acc.agentFile = fullPathSlash
		importRelPath = fullPathSlash
	}
	parserLog.Printf("Found agent file: %s (resolved to: %s)", item.fullPath, acc.agentFile)
	acc.agentImportSpec = item.importPath
	parserLog.Printf("Agent import specification: %s", acc.agentImportSpec)
	if len(item.inputs) == 0 {
		acc.importPaths = append(acc.importPaths, importRelPath)
		parserLog.Printf("Added agent import path for runtime-import: %s", importRelPath)
		return nil
	}
	parserLog.Printf("Agent file has inputs - will be inlined instead of runtime-imported")
	markdownContent, err := processIncludedFileWithVisited(item.fullPath, item.sectionName, false, visited)
	if err != nil {
		return fmt.Errorf("failed to process markdown from agent file '%s': %w", item.fullPath, err)
	}
	if markdownContent != "" {
		acc.markdownBuilder.WriteString(markdownContent)
		if !strings.HasSuffix(markdownContent, "\n\n") {
			if strings.HasSuffix(markdownContent, "\n") {
				acc.markdownBuilder.WriteString("\n")
			} else {
				acc.markdownBuilder.WriteString("\n\n")
			}
		}
	}
	return nil
}

// handleYAMLWorkflowBFSItem processes a YAML workflow file BFS queue item.
func handleYAMLWorkflowBFSItem(item importQueueItem, acc *importAccumulator) error {
	parserLog.Printf("Detected YAML workflow file: %s", item.fullPath)
	jobsOrStepsData, servicesJSON, err := processYAMLWorkflowImport(item.fullPath)
	if err != nil {
		return fmt.Errorf("failed to process YAML workflow '%s': %w", item.importPath, err)
	}
	if isCopilotSetupStepsFile(item.fullPath) {
		if jobsOrStepsData != "" {
			acc.copilotSetupStepsBuilder.WriteString(jobsOrStepsData + "\n")
			parserLog.Printf("Added copilot-setup steps (will be inserted at start): %s", item.importPath)
		}
	} else if jobsOrStepsData != "" && jobsOrStepsData != "{}" {
		acc.jobsBuilder.WriteString(jobsOrStepsData + "\n")
		parserLog.Printf("Added jobs from YAML workflow: %s", item.importPath)
	}
	if servicesJSON != "" && servicesJSON != "{}" {
		var services map[string]any
		if jsonErr := json.Unmarshal([]byte(servicesJSON), &services); jsonErr == nil {
			servicesWrapper := map[string]any{"services": services}
			if servicesYAML, marshalErr := yaml.Marshal(servicesWrapper); marshalErr == nil {
				acc.servicesBuilder.WriteString(string(servicesYAML) + "\n")
				parserLog.Printf("Added services from YAML workflow: %s", item.importPath)
			}
		}
	}
	return nil
}

// processMarkdownBFSItem reads, parses, queues nested imports, and extracts fields from a markdown file.
func processMarkdownBFSItem(item importQueueItem, visited map[string]bool, visitedInputs map[string]map[string]any, acc *importAccumulator, baseDir string, cache *ImportCache, workflowFilePath, yamlContent string) ([]importQueueItem, error) {
	content, result, err := parseFrontmatterWithDefaultsForBFS(item)
	if err != nil {
		return nil, err
	}
	var newQueue []importQueueItem
	if result != nil && result.Frontmatter != nil {
		nestedImports := extractNestedImportEntries(result.Frontmatter)
		newQueue, err = queueNestedImports(nestedImports, item, visited, visitedInputs, baseDir, cache, workflowFilePath, yamlContent)
		if err != nil {
			return nil, err
		}
	}
	if err := acc.extractAllImportFields(content, item, visited); err != nil {
		return nil, err
	}
	return newQueue, nil
}

// parseFrontmatterWithDefaultsForBFS reads a file and parses its frontmatter, applying
// import-schema defaults to resolve ${{ github.aw.import-inputs.* }} expressions.
func parseFrontmatterWithDefaultsForBFS(item importQueueItem) ([]byte, *FrontmatterResult, error) {
	content, err := readFileFunc(item.fullPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read imported file '%s': %w", item.fullPath, err)
	}
	var result *FrontmatterResult
	if strings.HasPrefix(item.fullPath, BuiltinPathPrefix) {
		result, err = ExtractFrontmatterFromBuiltinFile(item.fullPath, content)
	} else {
		result, err = ExtractFrontmatterFromContent(string(content))
	}
	if err == nil && result != nil {
		inputsWithDefaults := applyImportSchemaDefaultsFromFrontmatter(result.Frontmatter, item.inputs)
		if len(inputsWithDefaults) > 0 {
			origContent := string(content)
			substituted := substituteImportInputsInContent(origContent, inputsWithDefaults)
			if substituted != origContent {
				if reparse, rerr := ExtractFrontmatterFromContent(substituted); rerr == nil {
					result = reparse
				}
			}
		}
	}
	if err != nil {
		parserLog.Printf("Failed to extract frontmatter from %s: %v", item.fullPath, err)
		return content, nil, nil
	}
	return content, result, nil
}

// extractNestedImportEntries extracts nested import entries from a frontmatter map.
func extractNestedImportEntries(frontmatter map[string]any) []bfsNestedImportEntry {
	nestedImportsField, hasImports := frontmatter["imports"]
	if !hasImports {
		return nil
	}
	var entries []bfsNestedImportEntry
	switch v := nestedImportsField.(type) {
	case []any:
		for _, nestedItem := range v {
			if str, ok := nestedItem.(string); ok {
				entries = append(entries, bfsNestedImportEntry{path: str})
			} else if nestedMap, ok := nestedItem.(map[string]any); ok {
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
					entries = append(entries, bfsNestedImportEntry{path: nestedPath, inputs: nestedInputs})
				}
			}
		}
	case []string:
		for _, str := range v {
			entries = append(entries, bfsNestedImportEntry{path: str})
		}
	}
	return entries
}

// queueNestedImports resolves and enqueues each nested import entry discovered in a file.
func queueNestedImports(entries []bfsNestedImportEntry, item importQueueItem, visited map[string]bool, visitedInputs map[string]map[string]any, baseDir string, cache *ImportCache, workflowFilePath, yamlContent string) ([]importQueueItem, error) {
	var newItems []importQueueItem
	for _, entry := range entries {
		newItem, err := resolveAndEnqueueNestedImport(entry, item, visited, visitedInputs, baseDir, cache, workflowFilePath, yamlContent)
		if err != nil {
			return nil, err
		}
		if newItem != nil {
			newItems = append(newItems, *newItem)
		}
	}
	return newItems, nil
}

// resolveNestedRemotePath resolves the path for a nested import when the parent was fetched
// from a remote repo, building the workflowspec form needed for further resolution.
func resolveNestedRemotePath(nestedFilePath string, item importQueueItem) (string, *remoteImportOrigin, error) {
	cleanPath := path.Clean(strings.TrimPrefix(nestedFilePath, "./"))
	if cleanPath == ".." || strings.HasPrefix(cleanPath, "../") || path.IsAbs(cleanPath) {
		return "", nil, fmt.Errorf("nested import '%s' from remote file '%s' escapes base directory", nestedFilePath, item.importPath)
	}
	basePath := item.remoteOrigin.BasePath
	if basePath == "" {
		basePath = constants.GetWorkflowDir()
	}
	basePath = path.Clean(basePath)
	resolvedPath := fmt.Sprintf("%s/%s/%s/%s@%s", item.remoteOrigin.Owner, item.remoteOrigin.Repo, basePath, cleanPath, item.remoteOrigin.Ref)
	nestedRemoteOrigin := parseRemoteOrigin(resolvedPath)
	importLog.Printf("Resolving nested import as remote workflowspec: %s -> %s (basePath=%s)", nestedFilePath, resolvedPath, basePath)
	return resolvedPath, nestedRemoteOrigin, nil
}

// resolveNestedImportPath determines the resolved path and remote origin for a nested import.
func resolveNestedImportPath(nestedFilePath string, item importQueueItem) (string, *remoteImportOrigin, error) {
	if item.remoteOrigin != nil && !isWorkflowSpec(nestedFilePath) {
		return resolveNestedRemotePath(nestedFilePath, item)
	}
	if isWorkflowSpec(nestedFilePath) {
		nestedRemoteOrigin := parseRemoteOrigin(nestedFilePath)
		if nestedRemoteOrigin != nil {
			importLog.Printf("Nested workflowspec import detected: %s (origin: %s/%s@%s)", nestedFilePath, nestedRemoteOrigin.Owner, nestedRemoteOrigin.Repo, nestedRemoteOrigin.Ref)
		}
		return nestedFilePath, nestedRemoteOrigin, nil
	}
	return nestedFilePath, nil, nil
}

// resolveAndEnqueueNestedImport resolves a nested import path and returns a queue item if not yet visited.
func resolveAndEnqueueNestedImport(entry bfsNestedImportEntry, item importQueueItem, visited map[string]bool, visitedInputs map[string]map[string]any, baseDir string, cache *ImportCache, workflowFilePath, yamlContent string) (*importQueueItem, error) {
	nestedImportPath := entry.path
	var nestedFilePath, nestedSectionName string
	if strings.Contains(nestedImportPath, "#") {
		parts := strings.SplitN(nestedImportPath, "#", 2)
		nestedFilePath, nestedSectionName = parts[0], parts[1]
	} else {
		nestedFilePath = nestedImportPath
	}
	resolvedPath, nestedRemoteOrigin, err := resolveNestedImportPath(nestedFilePath, item)
	if err != nil {
		return nil, err
	}
	isLocalRelative := !strings.Contains(resolvedPath, "/") || strings.HasPrefix(resolvedPath, "./")
	nestedBaseDir := baseDir
	if item.remoteOrigin == nil && !isWorkflowSpec(resolvedPath) && isLocalRelative {
		nestedBaseDir = filepath.Dir(item.fullPath)
	}
	nestedFullPath, err := ResolveIncludePath(resolvedPath, nestedBaseDir, cache)
	if err != nil {
		if workflowFilePath != "" && yamlContent != "" {
			line, column := findImportItemLocation(yamlContent, item.importPath)
			importErr := &ImportError{ImportPath: nestedImportPath, FilePath: workflowFilePath, Line: line, Column: column, Cause: err}
			return nil, FormatImportError(importErr, yamlContent)
		}
		return nil, fmt.Errorf("failed to resolve nested import '%s' from '%s': %w", nestedFilePath, item.fullPath, err)
	}
	if visited[nestedFullPath] {
		if err := checkImportInputsConsistency(nestedImportPath, visitedInputs[nestedFullPath], entry.inputs); err != nil {
			return nil, err
		}
		parserLog.Printf("Skipping already visited nested import: %s (cycle detected)", nestedFullPath)
		return nil, nil
	}
	visited[nestedFullPath] = true
	visitedInputs[nestedFullPath] = entry.inputs
	canonicalImportPath := nestedImportPath
	if nestedRemoteOrigin == nil && nestedBaseDir != baseDir {
		if rel, relErr := filepath.Rel(baseDir, nestedFullPath); relErr == nil {
			canonicalImportPath = filepath.ToSlash(rel)
		}
	}
	newItem := importQueueItem{importPath: canonicalImportPath, fullPath: nestedFullPath, sectionName: nestedSectionName, baseDir: baseDir, inputs: entry.inputs, remoteOrigin: nestedRemoteOrigin}
	parserLog.Printf("Discovered nested import: %s -> %s (queued)", item.fullPath, nestedFullPath)
	return &newItem, nil
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
