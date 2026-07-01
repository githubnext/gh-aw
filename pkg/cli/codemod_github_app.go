package cli

import (
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

var githubAppCodemodLog = logger.New("cli:codemod_github_app")

// getGitHubAppCodemod creates a codemod for renaming 'app:' to 'github-app:' in workflow frontmatter.
// The deprecated 'app:' field can appear at the top level and under tools.github,
// safe-outputs, and checkout.
func getGitHubAppCodemod() Codemod {
	return Codemod{
		ID:           "app-to-github-app",
		Name:         "Rename 'app' to 'github-app'",
		Description:  "Renames the deprecated 'app:' field to 'github-app:' at the top level and in tools.github, safe-outputs, and checkout configurations.",
		IntroducedIn: "0.15.0",
		Apply: func(content string, frontmatter map[string]any) (string, bool, error) {
			if !hasDeprecatedAppField(frontmatter) {
				return content, false, nil
			}
			newContent, applied, err := applyFrontmatterLineTransform(content, renameAppToGitHubApp)
			if applied {
				githubAppCodemodLog.Print("Renamed 'app' to 'github-app'")
			}
			return newContent, applied, err
		},
	}
}

// hasDeprecatedAppField returns true if the deprecated 'app:' field is present at the
// top level or in one of the supported nested sections.
func hasDeprecatedAppField(frontmatter map[string]any) bool {
	// Check top-level app
	if _, hasApp := frontmatter["app"]; hasApp {
		githubAppCodemodLog.Print("Deprecated 'app' field found at top level")
		return true
	}

	// Check tools.github.app
	if toolsAny, hasTools := frontmatter["tools"]; hasTools {
		if toolsMap, ok := toolsAny.(map[string]any); ok {
			if githubAny, hasGitHub := toolsMap["github"]; hasGitHub {
				if githubMap, ok := githubAny.(map[string]any); ok {
					if _, hasApp := githubMap["app"]; hasApp {
						githubAppCodemodLog.Print("Deprecated 'app' field found in tools.github")
						return true
					}
				}
			}
		}
	}

	// Check safe-outputs.app
	if soAny, hasSO := frontmatter["safe-outputs"]; hasSO {
		if soMap, ok := soAny.(map[string]any); ok {
			if _, hasApp := soMap["app"]; hasApp {
				githubAppCodemodLog.Print("Deprecated 'app' field found in safe-outputs")
				return true
			}
		}
	}

	// Check checkout.app (single object or array of objects)
	if checkoutAny, hasCheckout := frontmatter["checkout"]; hasCheckout {
		if checkoutMap, ok := checkoutAny.(map[string]any); ok {
			if _, hasApp := checkoutMap["app"]; hasApp {
				githubAppCodemodLog.Print("Deprecated 'app' field found in checkout")
				return true
			}
		}
		if checkoutArr, ok := checkoutAny.([]any); ok {
			for _, item := range checkoutArr {
				if itemMap, ok := item.(map[string]any); ok {
					if _, hasApp := itemMap["app"]; hasApp {
						githubAppCodemodLog.Print("Deprecated 'app' field found in checkout array item")
						return true
					}
				}
			}
		}
	}

	return false
}

// renameAppToGitHubApp renames top-level 'app:' keys and nested 'app:' keys within
// tools.github, safe-outputs, and checkout blocks.
func renameAppToGitHubApp(lines []string) ([]string, bool) {
	var result []string
	modified := false

	// Block tracking
	var inTools, inToolsGithub, inSafeOutputs, inCheckout bool
	var toolsIndent, toolsGithubIndent, safeOutputsIndent, checkoutIndent string

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
			if inSafeOutputs && hasExitedBlock(line, safeOutputsIndent) {
				inSafeOutputs = false
			}
			if inCheckout && hasExitedBlock(line, checkoutIndent) {
				inCheckout = false
			}
		}

		// Detect block entries at any indentation level
		if strings.HasPrefix(trimmed, "tools:") {
			inTools = true
			inToolsGithub = false
			toolsIndent = getIndentation(line)
			result = append(result, line)
			continue
		}

		if inTools && strings.HasPrefix(trimmed, "github:") {
			inToolsGithub = true
			toolsGithubIndent = getIndentation(line)
			result = append(result, line)
			continue
		}

		if strings.HasPrefix(trimmed, "safe-outputs:") {
			inSafeOutputs = true
			safeOutputsIndent = getIndentation(line)
			result = append(result, line)
			continue
		}

		if strings.HasPrefix(trimmed, "checkout:") {
			inCheckout = true
			checkoutIndent = getIndentation(line)
			result = append(result, line)
			continue
		}

		// Rename a top-level 'app:' key.
		if strings.HasPrefix(trimmed, "app:") && isTopLevelKey(line) {
			newLine, replaced := findAndReplaceInLine(line, "app", "github-app")
			if replaced {
				result = append(result, newLine)
				modified = true
				githubAppCodemodLog.Printf("Renamed top-level 'app' to 'github-app' on line %d", i+1)
				continue
			}
		}

		// Rename nested 'app:' keys when inside a target block
		if strings.HasPrefix(trimmed, "app:") {
			lineIndent := getIndentation(line)
			shouldRename := false

			// Child of tools.github (inside github: block)
			if inToolsGithub && isDescendant(lineIndent, toolsGithubIndent) {
				shouldRename = true
			}

			// Child of safe-outputs
			if inSafeOutputs && isDescendant(lineIndent, safeOutputsIndent) {
				shouldRename = true
			}

			// Child of checkout (or a list item inside checkout)
			if inCheckout && isDescendant(lineIndent, checkoutIndent) {
				shouldRename = true
			}

			if shouldRename {
				newLine, replaced := findAndReplaceInLine(line, "app", "github-app")
				if replaced {
					result = append(result, newLine)
					modified = true
					githubAppCodemodLog.Printf("Renamed 'app' to 'github-app' on line %d", i+1)
					continue
				}
			}
		}

		result = append(result, line)
	}

	return result, modified
}
