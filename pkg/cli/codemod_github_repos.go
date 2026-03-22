package cli

import (
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

var githubReposCodemodLog = logger.New("cli:codemod_github_repos")

// getGitHubReposToAllowedReposCodemod creates a codemod that renames the deprecated
// 'repos:' field to 'allowed-repos:' within the tools.github configuration block.
func getGitHubReposToAllowedReposCodemod() Codemod {
	return Codemod{
		ID:           "github-repos-to-allowed-repos",
		Name:         "Rename 'tools.github.repos' to 'tools.github.allowed-repos'",
		Description:  "Renames the deprecated 'repos:' field to 'allowed-repos:' inside the tools.github configuration block.",
		IntroducedIn: "1.0.0",
		Apply: func(content string, frontmatter map[string]any) (string, bool, error) {
			if !hasDeprecatedGitHubReposField(frontmatter) {
				return content, false, nil
			}
			newContent, applied, err := applyFrontmatterLineTransform(content, renameGitHubReposToAllowedRepos)
			if applied {
				githubReposCodemodLog.Print("Renamed 'tools.github.repos' to 'tools.github.allowed-repos'")
			}
			return newContent, applied, err
		},
	}
}

// hasDeprecatedGitHubReposField returns true if tools.github has a deprecated 'repos' field
// and does not already have an 'allowed-repos' field.
func hasDeprecatedGitHubReposField(frontmatter map[string]any) bool {
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
	_, hasRepos := githubMap["repos"]
	_, hasAllowedRepos := githubMap["allowed-repos"] // only check existence, not value
	if hasRepos && !hasAllowedRepos {
		githubReposCodemodLog.Print("Deprecated 'repos' field found in tools.github")
	}
	return hasRepos && !hasAllowedRepos
}

// renameGitHubReposToAllowedRepos renames 'repos:' to 'allowed-repos:' within the
// tools.github configuration block.
func renameGitHubReposToAllowedRepos(lines []string) ([]string, bool) {
	var result []string
	modified := false

	var inTools, inToolsGithub bool
	var toolsIndent, toolsGithubIndent string

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Skip empty lines without resetting state
		if len(trimmed) == 0 {
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

		// Rename 'repos:' to 'allowed-repos:' when inside tools.github
		if inToolsGithub && strings.HasPrefix(trimmed, "repos:") {
			lineIndent := getIndentation(line)
			if isDescendant(lineIndent, toolsGithubIndent) {
				newLine, replaced := findAndReplaceInLine(line, "repos", "allowed-repos")
				if replaced {
					result = append(result, newLine)
					modified = true
					githubReposCodemodLog.Printf("Renamed 'repos' to 'allowed-repos' on line %d", i+1)
					continue
				}
			}
		}

		result = append(result, line)
	}

	return result, modified
}
