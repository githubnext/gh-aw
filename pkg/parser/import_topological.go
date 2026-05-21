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

// buildDependencyGraph reads each import file to extract its nested imports, returning
// the dependency map and the set of all import paths.
func buildDependencyGraph(imports []string, baseDir string, cache *ImportCache) (map[string][]string, map[string]bool) {
	allImportsSet := make(map[string]bool)
	for _, imp := range imports {
		allImportsSet[imp] = true
	}
	dependencies := make(map[string][]string)
	for _, importPath := range imports {
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
		content, err := readFileFunc(fullPath)
		if err != nil {
			importLog.Printf("Failed to read file %s during topological sort: %v", fullPath, err)
			dependencies[importPath] = []string{}
			continue
		}
		var result *FrontmatterResult
		if strings.HasPrefix(fullPath, BuiltinPathPrefix) {
			result, err = ExtractFrontmatterFromBuiltinFile(fullPath, content)
		} else {
			result, err = ExtractFrontmatterFromContent(string(content))
		}
		if err != nil {
			importLog.Printf("Failed to extract frontmatter from %s during topological sort: %v", fullPath, err)
			dependencies[importPath] = []string{}
			continue
		}
		nestedImports := extractImportPaths(result.Frontmatter)
		dependencies[importPath] = nestedImports
		importLog.Printf("Import %s has %d dependencies: %v", importPath, len(nestedImports), nestedImports)
	}
	return dependencies, allImportsSet
}

// computeInDegrees calculates the number of in-set dependencies for each import.
func computeInDegrees(imports []string, dependencies map[string][]string, allImportsSet map[string]bool) map[string]int {
	inDegree := make(map[string]int)
	for _, imp := range imports {
		inDegree[imp] = 0
	}
	sortedImports := make([]string, 0, len(dependencies))
	for imp := range dependencies {
		sortedImports = append(sortedImports, imp)
	}
	sort.Strings(sortedImports)
	for _, imp := range sortedImports {
		for _, dep := range dependencies[imp] {
			if allImportsSet[dep] {
				inDegree[imp]++
			}
		}
	}
	importLog.Printf("Calculated in-degrees: %v", inDegree)
	return inDegree
}

// runKahnAlgorithm executes Kahn's topological sort and returns the sorted list.
func runKahnAlgorithm(imports []string, inDegree map[string]int, dependencies map[string][]string, allImportsSet map[string]bool) []string {
	var queue []string
	for _, imp := range imports {
		if inDegree[imp] == 0 {
			queue = append(queue, imp)
			importLog.Printf("Root import (no dependencies): %s", imp)
		}
	}
	result := make([]string, 0, len(imports))
	for len(queue) > 0 {
		sort.Strings(queue)
		current := queue[0]
		queue = queue[1:]
		result = append(result, current)
		importLog.Printf("Processing import %s (in-degree was 0)", current)
		sortedImps := make([]string, 0, len(dependencies))
		for imp := range dependencies {
			sortedImps = append(sortedImps, imp)
		}
		sort.Strings(sortedImps)
		for _, imp := range sortedImps {
			for _, dep := range dependencies[imp] {
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
	return result
}

// detectAndReportCycle checks whether all imports were processed, returning an error if a cycle exists.
func detectAndReportCycle(imports []string, sorted []string, dependencies map[string][]string, workflowFile string) error {
	if len(sorted) >= len(imports) {
		return nil
	}
	importLog.Printf("Cycle detected: processed %d/%d imports", len(sorted), len(imports))
	cycleNodes := make(map[string]bool)
	for _, imp := range imports {
		if !slices.Contains(sorted, imp) {
			cycleNodes[imp] = true
		}
	}
	cyclePath := findCyclePath(cycleNodes, dependencies)
	if len(cyclePath) > 0 {
		return &ImportCycleError{Chain: cyclePath, WorkflowFile: workflowFile}
	}
	return errors.New("circular import detected but could not determine cycle path")
}

// topologicalSortImports sorts imports in topological order using Kahn's algorithm.
// Returns imports sorted such that roots (files with no imports) come first,
// and each import has all its dependencies listed before it.
// workflowFile is the path to the top-level workflow file, used for error context
// when a circular import is detected.
// Returns an error if a circular import is detected.
func topologicalSortImports(imports []string, baseDir string, cache *ImportCache, workflowFile string) ([]string, error) {
	importLog.Printf("Starting topological sort of %d imports", len(imports))
	dependencies, allImportsSet := buildDependencyGraph(imports, baseDir, cache)
	inDegree := computeInDegrees(imports, dependencies, allImportsSet)
	sorted := runKahnAlgorithm(imports, inDegree, dependencies, allImportsSet)
	if err := detectAndReportCycle(imports, sorted, dependencies, workflowFile); err != nil {
		return nil, err
	}
	return sorted, nil
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
				} else if usesValue, hasUses := importItem["uses"]; hasUses {
					if pathStr, ok := usesValue.(string); ok {
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
