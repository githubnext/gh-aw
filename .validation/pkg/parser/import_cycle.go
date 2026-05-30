// Package parser provides functions for parsing and processing workflow markdown files.
// import_cycle.go implements cycle detection in the import dependency graph using
// depth-first search to find and report circular import chains.
package parser

import "sort"

// findCyclePath uses DFS to find a complete cycle path in the dependency graph.
// Returns a path showing the full chain including the back-edge (e.g., ["b.md", "c.md", "d.md", "b.md"]).
func findCyclePath(cycleNodes map[string]bool, dependencies map[string][]string) []string {
	importLog.Printf("Finding cycle path among %d cycle nodes", len(cycleNodes))

	// Pick any node in the cycle as a starting point (use sorted order for determinism)
	var startNode string
	sortedNodes := make([]string, 0, len(cycleNodes))
	for node := range cycleNodes {
		sortedNodes = append(sortedNodes, node)
	}
	sort.Strings(sortedNodes)
	if len(sortedNodes) > 0 {
		startNode = sortedNodes[0]
	} else {
		importLog.Print("No cycle nodes found, cannot determine cycle path")
		return nil
	}

	importLog.Printf("Starting DFS cycle detection from node: %s", startNode)

	// Use DFS to find a path from startNode back to itself
	visited := make(map[string]bool)
	path := []string{}
	if dfsForCycle(startNode, startNode, cycleNodes, dependencies, visited, &path, true) {
		importLog.Printf("Cycle path found: %v", path)
		return path
	}

	importLog.Print("DFS completed but no cycle path could be constructed")
	return nil
}

// dfsForCycle performs DFS to find a cycle path.
// isFirst tracks if this is the first call (starting point).
func dfsForCycle(current, target string, cycleNodes map[string]bool, dependencies map[string][]string, visited map[string]bool, path *[]string, isFirst bool) bool {
	// Add current node to path
	*path = append(*path, current)
	visited[current] = true

	// Get dependencies of current node, sorted for determinism
	deps := dependencies[current]
	sortedDeps := make([]string, 0, len(deps))
	for _, dep := range deps {
		// Only follow edges within the cycle subgraph
		if cycleNodes[dep] {
			sortedDeps = append(sortedDeps, dep)
		}
	}
	sort.Strings(sortedDeps)

	// Explore each dependency
	for _, dep := range sortedDeps {
		// Found the cycle - we've reached the target again
		if !isFirst && dep == target {
			importLog.Printf("Cycle back-edge found: %s -> %s", current, dep)
			*path = append(*path, dep) // Add the back-edge
			return true
		}

		// Continue DFS if not visited
		if !visited[dep] {
			if dfsForCycle(dep, target, cycleNodes, dependencies, visited, path, false) {
				return true
			}
		}
	}

	// Backtrack
	*path = (*path)[:len(*path)-1]
	return false
}
