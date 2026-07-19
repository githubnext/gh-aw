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

	state := renameAppToGitHubAppState{}

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Skip empty lines without resetting state
		if trimmed == "" {
			result = append(result, line)
			continue
		}

		// Exit blocks when indentation signals we've left them
		renameAppToGitHubAppExitBlocks(line, trimmed, &state)

		// Detect block entries at any indentation level
		if renameAppToGitHubAppEnterBlock(line, trimmed, &state) {
			result = append(result, line)
			continue
		}

		if newLine, replaced := renameAppToGitHubAppLine(line, trimmed, &state, i); replaced {
			result = append(result, newLine)
			modified = true
			continue
		}

		result = append(result, line)
	}

	return result, modified
}

type renameAppToGitHubAppState struct {
	inTools           bool
	inToolsGithub     bool
	inSafeOutputs     bool
	inCheckout        bool
	toolsIndent       string
	toolsGithubIndent string
	safeOutputsIndent string
	checkoutIndent    string
}

func renameAppToGitHubAppExitBlocks(line string, trimmed string, state *renameAppToGitHubAppState) {
	if strings.HasPrefix(trimmed, "#") {
		return
	}
	if state.inToolsGithub && hasExitedBlock(line, state.toolsGithubIndent) {
		state.inToolsGithub = false
	}
	if state.inTools && hasExitedBlock(line, state.toolsIndent) {
		state.inTools = false
		state.inToolsGithub = false
	}
	if state.inSafeOutputs && hasExitedBlock(line, state.safeOutputsIndent) {
		state.inSafeOutputs = false
	}
	if state.inCheckout && hasExitedBlock(line, state.checkoutIndent) {
		state.inCheckout = false
	}
}

func renameAppToGitHubAppEnterBlock(line string, trimmed string, state *renameAppToGitHubAppState) bool {
	if strings.HasPrefix(trimmed, "tools:") {
		state.inTools = true
		state.inToolsGithub = false
		state.toolsIndent = getIndentation(line)
		return true
	}
	if state.inTools && strings.HasPrefix(trimmed, "github:") {
		state.inToolsGithub = true
		state.toolsGithubIndent = getIndentation(line)
		return true
	}
	if strings.HasPrefix(trimmed, "safe-outputs:") {
		state.inSafeOutputs = true
		state.safeOutputsIndent = getIndentation(line)
		return true
	}
	if strings.HasPrefix(trimmed, "checkout:") {
		state.inCheckout = true
		state.checkoutIndent = getIndentation(line)
		return true
	}
	return false
}

func renameAppToGitHubAppLine(line string, trimmed string, state *renameAppToGitHubAppState, index int) (string, bool) {
	if !strings.HasPrefix(trimmed, "app:") {
		return line, false
	}
	if isTopLevelKey(line) {
		return renameAppToGitHubAppReplace(line, "top-level 'app' to 'github-app'", index)
	}
	if renameAppToGitHubAppShouldRenameNested(line, state) {
		return renameAppToGitHubAppReplace(line, "'app' to 'github-app'", index)
	}
	return line, false
}

func renameAppToGitHubAppShouldRenameNested(line string, state *renameAppToGitHubAppState) bool {
	lineIndent := getIndentation(line)
	if state.inToolsGithub && isDescendant(lineIndent, state.toolsGithubIndent) {
		return true
	}
	if state.inSafeOutputs && isDescendant(lineIndent, state.safeOutputsIndent) {
		return true
	}
	return state.inCheckout && isDescendant(lineIndent, state.checkoutIndent)
}

func renameAppToGitHubAppReplace(line string, label string, index int) (string, bool) {
	newLine, replaced := findAndReplaceInLine(line, "app", "github-app")
	if replaced {
		githubAppCodemodLog.Printf("Renamed %s on line %d", label, index+1)
	}
	return newLine, replaced
}
