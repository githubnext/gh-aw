package cli

import (
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

var toolsetSingularCodemodLog = logger.New("cli:codemod_toolset_singular")

// getToolsetSingularToToolsetsCodemod creates a codemod that renames the mistyped
// singular 'toolset:' field to the correct plural 'toolsets:' field within the
// tools.github configuration block.
func getToolsetSingularToToolsetsCodemod() Codemod {
	return Codemod{
		ID:           "toolset-singular-to-toolsets",
		Name:         "Rename 'tools.github.toolset' to 'tools.github.toolsets'",
		Description:  "Renames the mistyped singular 'toolset:' field to the correct plural 'toolsets:' inside the tools.github configuration block.",
		IntroducedIn: "1.0.0",
		Apply: func(content string, frontmatter map[string]any) (string, bool, error) {
			if !hasSingularToolsetField(frontmatter) {
				return content, false, nil
			}
			newContent, applied, err := applyFrontmatterLineTransform(content, renameToolsetSingularToToolsets)
			if applied {
				toolsetSingularCodemodLog.Print("Renamed 'tools.github.toolset' to 'tools.github.toolsets'")
			}
			return newContent, applied, err
		},
	}
}

// hasSingularToolsetField returns true if tools.github has a mistyped singular
// 'toolset' field and does not already have a 'toolsets' field.
func hasSingularToolsetField(frontmatter map[string]any) bool {
	toolsAny, hasTools := frontmatter["tools"]
	if !hasTools {
		return false
	}
	toolsMap, ok := toolsAny.(map[string]any)
	if !ok {
		return false
	}
	githubAny, hasGitHub := toolsMap["github"]
	if !hasGitHub {
		return false
	}
	githubMap, ok := githubAny.(map[string]any)
	if !ok {
		return false
	}
	_, hasToolset := githubMap["toolset"]
	_, hasToolsets := githubMap["toolsets"] // only check existence, not value
	if hasToolset && !hasToolsets {
		toolsetSingularCodemodLog.Print("Mistyped singular 'toolset' field found in tools.github")
	}
	return hasToolset && !hasToolsets
}

// renameToolsetSingularToToolsets renames 'toolset:' to 'toolsets:' within the
// tools.github configuration block.
func renameToolsetSingularToToolsets(lines []string) ([]string, bool) {
	var result []string
	modified := false

	var inTools, inToolsGithub bool
	var toolsIndent, toolsGithubIndent string

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Skip empty lines without resetting state
		if trimmed == "" {
			result = append(result, line)
			continue
		}

		// Exit blocks when indentation signals we've left them
		if !strings.HasPrefix(trimmed, "#") {
			if inToolsGithub && hasExitedBlock(line, toolsGithubIndent) {
				inToolsGithub = false
			}
			if inTools && hasExitedBlock(line, toolsIndent) {
				inTools = false
				inToolsGithub = false
			}
		}

		// Detect 'tools:' block
		if strings.HasPrefix(trimmed, "tools:") {
			inTools = true
			inToolsGithub = false
			toolsIndent = getIndentation(line)
			result = append(result, line)
			continue
		}

		// Detect 'github:' block inside 'tools:'
		if inTools && strings.HasPrefix(trimmed, "github:") {
			inToolsGithub = true
			toolsGithubIndent = getIndentation(line)
			result = append(result, line)
			continue
		}

		// Rename 'toolset:' to 'toolsets:' when inside tools.github
		if inToolsGithub && strings.HasPrefix(trimmed, "toolset:") {
			lineIndent := getIndentation(line)
			if isDescendant(lineIndent, toolsGithubIndent) {
				newLine, replaced := findAndReplaceInLine(line, "toolset", "toolsets")
				if replaced {
					result = append(result, newLine)
					modified = true
					toolsetSingularCodemodLog.Printf("Renamed 'toolset' to 'toolsets' on line %d", i+1)
					continue
				}
			}
		}

		result = append(result, line)
	}

	return result, modified
}
