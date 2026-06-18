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
	allImportsSet := toImportSet(imports)
	dependencies := buildImportDependencies(imports, baseDir, cache)
	inDegree := calculateInDegree(imports, dependencies, allImportsSet)
	importLog.Printf("Calculated in-degrees: %v", inDegree)
	queue := collectRootImports(imports, inDegree)
	result := runKahnTopologicalSort(imports, dependencies, allImportsSet, inDegree, queue)

	importLog.Printf("Topological sort complete: %v", result)
	if len(result) < len(imports) {
		importLog.Printf("Cycle detected: processed %d/%d imports", len(result), len(imports))
		cycleNodes := findCycleNodes(imports, result)
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

func toImportSet(imports []string) map[string]struct {
} {
	allImportsSet := make(map[string]struct {
	}, len(imports))
	for _, imp := range imports {
		allImportsSet[imp] = struct {
		}{}
	}
	return allImportsSet
}

func buildImportDependencies(imports []string, baseDir string, cache *ImportCache) map[string][]string {
	dependencies := make(map[string][]string, len(imports))
	for _, importPath := range imports {
		nestedImports, err := resolveNestedImportPaths(importPath, baseDir, cache)
		if err != nil {
			importLog.Printf("Failed to resolve dependencies for %s during topological sort: %v", importPath, err)
			dependencies[importPath] = []string{}
			continue
		}
		dependencies[importPath] = nestedImports
		importLog.Printf("Import %s has %d dependencies: %v", importPath, len(nestedImports), nestedImports)
	}
	return dependencies
}

func resolveNestedImportPaths(importPath, baseDir string, cache *ImportCache) ([]string, error) {
	filePath := stripImportSection(importPath)
	fullPath, err := ResolveIncludePath(filePath, baseDir, cache)
	if err != nil {
		return nil, err
	}
	content, err := readFileFunc(fullPath)
	if err != nil {
		return nil, err
	}
	frontmatter, err := extractFrontmatterForTopologicalSort(fullPath, content)
	if err != nil {
		return nil, err
	}
	return extractImportPaths(frontmatter), nil
}

func stripImportSection(importPath string) string {
	if strings.Contains(importPath, "#") {
		parts := strings.SplitN(importPath, "#", 2)
		return parts[0]
	}
	return importPath
}

func extractFrontmatterForTopologicalSort(fullPath string, content []byte) (map[string]any, error) {
	var (
		result *FrontmatterResult
		err    error
	)
	if strings.HasPrefix(fullPath, BuiltinPathPrefix) {
		result, err = ExtractFrontmatterFromBuiltinFile(fullPath, content)
	} else {
		result, err = ExtractFrontmatterFromContent(string(content))
	}
	if err != nil {
		return nil, err
	}
	return result.Frontmatter, nil
}

func calculateInDegree(imports []string, dependencies map[string][]string, allImportsSet map[string]struct {
}) map[string]int {
	inDegree := make(map[string]int, len(imports))
	for _, imp := range imports {
		inDegree[imp] = 0
	}
	sortedImports := sortedDependencyKeys(dependencies)
	for _, imp := range sortedImports {
		for _, dep := range dependencies[imp] {
			if hasStringKey(allImportsSet, dep) {
				inDegree[imp]++
			}
		}
	}
	return inDegree
}

func sortedDependencyKeys(dependencies map[string][]string) []string {
	sortedImports := make([]string, 0, len(dependencies))
	for imp := range dependencies {
		sortedImports = append(sortedImports, imp)
	}
	sort.Strings(sortedImports)
	return sortedImports
}

func collectRootImports(imports []string, inDegree map[string]int) []string {
	var queue []string
	for _, imp := range imports {
		if inDegree[imp] == 0 {
			queue = append(queue, imp)
			importLog.Printf("Root import (no dependencies): %s", imp)
		}
	}
	return queue
}

func runKahnTopologicalSort(
	imports []string,
	dependencies map[string][]string,
	allImportsSet map[string]struct {
	}, inDegree map[string]int,
	queue []string,
) []string {
	result := make([]string, 0, len(imports))
	sortedImports := sortedDependencyKeys(dependencies)
	for len(queue) > 0 {
		sort.Strings(queue)
		current := queue[0]
		queue = queue[1:]
		result = append(result, current)
		importLog.Printf("Processing import %s (in-degree was 0)", current)
		queue = reduceDependentInDegrees(current, sortedImports, dependencies, allImportsSet, inDegree, queue)
	}
	return result
}

func reduceDependentInDegrees(
	current string,
	sortedImports []string,
	dependencies map[string][]string,
	allImportsSet map[string]struct {
	}, inDegree map[string]int,
	queue []string,
) []string {
	for _, imp := range sortedImports {
		for _, dep := range dependencies[imp] {
			if dep == current && hasStringKey(allImportsSet, imp) {
				inDegree[imp]--
				importLog.Printf("Reduced in-degree of %s to %d (resolved dependency on %s)", imp, inDegree[imp], current)
				if inDegree[imp] == 0 {
					queue = append(queue, imp)
					importLog.Printf("Added %s to queue (in-degree reached 0)", imp)
				}
			}
		}
	}
	return queue
}

func findCycleNodes(imports, result []string) map[string]struct {
} {
	cycleNodes := make(map[string]struct {
	})
	for _, imp := range imports {
		if !slices.Contains(result, imp) {
			cycleNodes[imp] = struct {
			}{}
		}
	}
	return cycleNodes
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
