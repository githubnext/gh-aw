// Package parser provides functions for parsing and processing workflow markdown files.
// import_topological.go implements topological ordering of imports using Kahn's algorithm,
// ensuring dependencies are processed before the files that depend on them.
package parser

import (
	"errors"
	"slices"
	"sort"
	"strings"
)

// topologicalSortImports sorts imports in topological order using Kahn's algorithm.
// Returns imports sorted such that roots (files with no imports) come first,
// and each import has all its dependencies listed before it.
// workflowFile is the path to the top-level workflow file, used for error context
// when a circular import is detected.
// Returns an error if a circular import is detected.
func topologicalSortImports(imports []string, baseDir string, cache *ImportCache, workflowFile string) ([]string, error) {
	importLog.Printf("Starting topological sort of %d imports", len(imports))

	// Build dependency graph: map each import to its list of nested imports
	dependencies := make(map[string][]string)
	allImportsSet := make(map[string]bool)

	// Track all imports (including the ones we're sorting)
	for _, imp := range imports {
		allImportsSet[imp] = true
	}

	// Extract dependencies for each import by reading and parsing each file
	for _, importPath := range imports {
		// Resolve the import path to get the full path
		var filePath string
		if strings.Contains(importPath, "#") {
			parts := strings.SplitN(importPath, "#", 2)
			filePath = parts[0]
		} else {
			filePath = importPath
		}

		fullPath, err := ResolveIncludePath(filePath, baseDir, cache)
		if err != nil {
			importLog.Printf("Failed to resolve import path %s during topological sort: %v", importPath, err)
			dependencies[importPath] = []string{}
			continue
		}

		// Read and parse the file to extract its imports
		content, err := readFileFunc(fullPath)
		if err != nil {
			importLog.Printf("Failed to read file %s during topological sort: %v", fullPath, err)
			dependencies[importPath] = []string{}
			continue
		}

		result, err := ExtractFrontmatterFromContent(string(content))
		if err != nil {
			importLog.Printf("Failed to extract frontmatter from %s during topological sort: %v", fullPath, err)
			dependencies[importPath] = []string{}
			continue
		}

		// Extract nested imports
		nestedImports := extractImportPaths(result.Frontmatter)
		dependencies[importPath] = nestedImports
		importLog.Printf("Import %s has %d dependencies: %v", importPath, len(nestedImports), nestedImports)
	}

	// Kahn's algorithm: Calculate in-degrees (number of dependencies for each import)
	inDegree := make(map[string]int)
	for _, imp := range imports {
		inDegree[imp] = 0
	}

	// Count dependencies: how many imports does each file depend on (within our import set)
	// Iterate over imports in sorted order for stable results
	sortedImportsForDegree := make([]string, 0, len(dependencies))
	for imp := range dependencies {
		sortedImportsForDegree = append(sortedImportsForDegree, imp)
	}
	sort.Strings(sortedImportsForDegree)

	for _, imp := range sortedImportsForDegree {
		deps := dependencies[imp]
		for _, dep := range deps {
			// Only count dependencies that are in our import set
			if allImportsSet[dep] {
				inDegree[imp]++
			}
		}
	}

	importLog.Printf("Calculated in-degrees: %v", inDegree)

	// Start with imports that have no dependencies (in-degree = 0) - these are the roots
	var queue []string
	for _, imp := range imports {
		if inDegree[imp] == 0 {
			queue = append(queue, imp)
			importLog.Printf("Root import (no dependencies): %s", imp)
		}
	}

	// Process imports in topological order
	result := make([]string, 0, len(imports))
	for len(queue) > 0 {
		// Sort queue for deterministic output when multiple imports have same in-degree
		sort.Strings(queue)

		// Take the first import from queue
		current := queue[0]
		queue = queue[1:]
		result = append(result, current)

		importLog.Printf("Processing import %s (in-degree was 0)", current)

		// For each import that depends on the current import, reduce its in-degree
		// Iterate over dependencies in sorted order for stable results
		sortedImports := make([]string, 0, len(dependencies))
		for imp := range dependencies {
			sortedImports = append(sortedImports, imp)
		}
		sort.Strings(sortedImports)

		for _, imp := range sortedImports {
			deps := dependencies[imp]
			for _, dep := range deps {
				if dep == current && allImportsSet[imp] {
					inDegree[imp]--
					importLog.Printf("Reduced in-degree of %s to %d (resolved dependency on %s)", imp, inDegree[imp], current)
					if inDegree[imp] == 0 {
						queue = append(queue, imp)
						importLog.Printf("Added %s to queue (in-degree reached 0)", imp)
					}
				}
			}
		}
	}

	importLog.Printf("Topological sort complete: %v", result)

	// If we didn't process all imports, there's a cycle
	if len(result) < len(imports) {
		importLog.Printf("Cycle detected: processed %d/%d imports", len(result), len(imports))

		// Find which imports are part of the cycle (those not in result)
		cycleNodes := make(map[string]bool)
		for _, imp := range imports {
			found := slices.Contains(result, imp)
			if !found {
				cycleNodes[imp] = true
			}
		}

		// Use DFS to find a cycle path in the subgraph of cycle nodes
		cyclePath := findCyclePath(cycleNodes, dependencies)
		if len(cyclePath) > 0 {
			return nil, &ImportCycleError{
				Chain:        cyclePath,
				WorkflowFile: workflowFile,
			}
		}

		// Fallback error if we couldn't construct the path (shouldn't happen)
		return nil, errors.New("circular import detected but could not determine cycle path")
	}

	return result, nil
}

// extractImportPaths extracts just the import paths from frontmatter.
func extractImportPaths(frontmatter map[string]any) []string {
	var imports []string

	if frontmatter == nil {
		return imports
	}

	importsField, exists := frontmatter["imports"]
	if !exists {
		return imports
	}

	// Parse imports field - can be array of strings or objects with path
	switch v := importsField.(type) {
	case []any:
		for _, item := range v {
			switch importItem := item.(type) {
			case string:
				imports = append(imports, importItem)
			case map[string]any:
				if pathValue, hasPath := importItem["path"]; hasPath {
					if pathStr, ok := pathValue.(string); ok {
						imports = append(imports, pathStr)
					}
				}
			}
		}
	case []string:
		imports = v
	}

	return imports
}
