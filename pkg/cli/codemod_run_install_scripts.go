package cli

import (
	"fmt"
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

var runInstallScriptsCodemodLog = logger.New("cli:codemod_run_install_scripts")

// getRunInstallScriptsToRuntimesNodeCodemod creates a codemod that moves a top-level
// run-install-scripts field into runtimes.node.run-install-scripts.
func getRunInstallScriptsToRuntimesNodeCodemod() Codemod {
	return Codemod{
		ID:           "run-install-scripts-to-runtimes-node",
		Name:         "Move run-install-scripts under runtimes.node",
		Description:  "Moves the deprecated top-level 'run-install-scripts' field to 'runtimes.node.run-install-scripts', which is the only runtime that generates npm install commands.",
		IntroducedIn: "1.5.0",
		Apply: func(content string, frontmatter map[string]any) (string, bool, error) {
			// Check if top-level run-install-scripts exists
			risValue, hasTopLevel := frontmatter["run-install-scripts"]
			if !hasTopLevel {
				return content, false, nil
			}

			// Check if runtimes.node.run-install-scripts already exists (idempotent)
			alreadyNested := false
			if runtimesVal, ok := frontmatter["runtimes"]; ok {
				if runtimesMap, ok := runtimesVal.(map[string]any); ok {
					if nodeVal, ok := runtimesMap["node"]; ok {
						if nodeMap, ok := nodeVal.(map[string]any); ok {
							if _, ok := nodeMap["run-install-scripts"]; ok {
								alreadyNested = true
							}
						}
					}
				}
			}

			// Determine the string representation of the value
			risStr := "true"
			switch v := risValue.(type) {
			case bool:
				if !v {
					risStr = "false"
				}
			case string:
				if strings.EqualFold(v, "false") {
					risStr = "false"
				}
			}

			newContent, applied, err := applyFrontmatterLineTransform(content, func(lines []string) ([]string, bool) {
				return migrateRunInstallScriptsLines(lines, risStr, alreadyNested)
			})
			if applied {
				runInstallScriptsCodemodLog.Printf("Moved top-level 'run-install-scripts' to 'runtimes.node.run-install-scripts: %s'", risStr)
			}
			return newContent, applied, err
		},
	}
}

// migrateRunInstallScriptsLines transforms frontmatter lines by removing the top-level
// run-install-scripts key and injecting it under runtimes.node.
func migrateRunInstallScriptsLines(lines []string, risStr string, alreadyNested bool) ([]string, bool) {
	// Step 1: Find the top-level run-install-scripts line
	topLevelIdx := -1
	for i, line := range lines {
		if isTopLevelKey(line) && strings.HasPrefix(strings.TrimSpace(line), "run-install-scripts:") {
			topLevelIdx = i
			break
		}
	}
	if topLevelIdx == -1 {
		return lines, false
	}

	// Remove the top-level line
	result := make([]string, 0, len(lines))
	for i, line := range lines {
		if i != topLevelIdx {
			result = append(result, line)
		}
	}
	runInstallScriptsCodemodLog.Printf("Removed top-level 'run-install-scripts' on line %d", topLevelIdx+1)

	if alreadyNested {
		// runtimes.node.run-install-scripts already exists; just remove the top-level duplicate.
		runInstallScriptsCodemodLog.Print("'runtimes.node.run-install-scripts' already present – removed top-level duplicate only")
		return result, true
	}

	// Step 2: Detect the primary indentation unit used in the frontmatter
	indent := detectFrontmatterIndent(result)

	// Step 3: Find the runtimes: block (top-level key)
	runtimesIdx := -1
	for i, line := range result {
		if isTopLevelKey(line) && strings.HasPrefix(strings.TrimSpace(line), "runtimes:") {
			runtimesIdx = i
			break
		}
	}

	if runtimesIdx == -1 {
		// No runtimes block – append a new one at the end
		newLines := []string{
			"runtimes:",
			indent + "node:",
			fmt.Sprintf("%s%srun-install-scripts: %s", indent, indent, risStr),
		}
		runInstallScriptsCodemodLog.Print("No 'runtimes:' block found – appending new block")
		return append(result, newLines...), true
	}

	// Step 4: Find node: as a direct child of runtimes:
	runtimesIndent := getIndentation(result[runtimesIdx])
	nodeIdx := -1
	for i := runtimesIdx + 1; i < len(result); i++ {
		line := result[i]
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		lineIndent := getIndentation(line)
		if len(lineIndent) <= len(runtimesIndent) {
			// Exited the runtimes block without finding node:
			break
		}
		if strings.HasPrefix(trimmed, "node:") {
			nodeIdx = i
			runInstallScriptsCodemodLog.Printf("Found 'node:' sub-block at line %d", i+1)
			break
		}
	}

	if nodeIdx == -1 {
		// No node: sub-block – insert one right after runtimes:
		nodeIndent := runtimesIndent + indent
		nodeLines := []string{
			nodeIndent + "node:",
			fmt.Sprintf("%s%srun-install-scripts: %s", nodeIndent, indent, risStr),
		}
		newResult := make([]string, 0, len(result)+2)
		newResult = append(newResult, result[:runtimesIdx+1]...)
		newResult = append(newResult, nodeLines...)
		newResult = append(newResult, result[runtimesIdx+1:]...)
		runInstallScriptsCodemodLog.Print("No 'node:' sub-block found – inserting new one under 'runtimes:'")
		return newResult, true
	}

	// Step 5: node: exists – inject run-install-scripts right after node:
	nodeIndent := getIndentation(result[nodeIdx])
	fieldLine := fmt.Sprintf("%s%srun-install-scripts: %s", nodeIndent, indent, risStr)
	newResult := make([]string, 0, len(result)+1)
	newResult = append(newResult, result[:nodeIdx+1]...)
	newResult = append(newResult, fieldLine)
	newResult = append(newResult, result[nodeIdx+1:]...)
	runInstallScriptsCodemodLog.Printf("Injected 'run-install-scripts: %s' under existing 'node:' sub-block", risStr)
	return newResult, true
}

// detectFrontmatterIndent returns the indentation unit used in the YAML frontmatter
// by examining the first indented non-empty, non-comment line. Defaults to two spaces.
func detectFrontmatterIndent(lines []string) string {
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		ind := getIndentation(line)
		if len(ind) > 0 {
			return ind
		}
	}
	return "  "
}
