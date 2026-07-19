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

	state := renameGitHubReposToAllowedReposState{}

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Skip empty lines without resetting state
		if trimmed == "" {
			result = append(result, line)
			continue
		}

		// Exit blocks when indentation signals we've left them
		renameGitHubReposToAllowedReposExitBlocks(line, trimmed, &state)

		// Detect 'tools:' block
		if strings.HasPrefix(trimmed, "tools:") {
			state.inTools = true
			state.inToolsGithub = false
			state.toolsIndent = getIndentation(line)
			result = append(result, line)
			continue
		}

		// Detect 'github:' block inside 'tools:'
		if state.inTools && strings.HasPrefix(trimmed, "github:") {
			state.inToolsGithub = true
			state.toolsGithubIndent = getIndentation(line)
			result = append(result, line)
			continue
		}

		// Rename 'repos:' to 'allowed-repos:' when inside tools.github
		if newLine, replaced := renameGitHubReposToAllowedReposLine(line, trimmed, &state, i); replaced {
			result = append(result, newLine)
			modified = true
			continue
		}

		result = append(result, line)
	}

	return result, modified
}

type renameGitHubReposToAllowedReposState struct {
	inTools           bool
	inToolsGithub     bool
	toolsIndent       string
	toolsGithubIndent string
}

func renameGitHubReposToAllowedReposExitBlocks(line string, trimmed string, state *renameGitHubReposToAllowedReposState) {
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
}

func renameGitHubReposToAllowedReposLine(line string, trimmed string, state *renameGitHubReposToAllowedReposState, index int) (string, bool) {
	if !state.inToolsGithub || !strings.HasPrefix(trimmed, "repos:") {
		return line, false
	}
	lineIndent := getIndentation(line)
	if !isDescendant(lineIndent, state.toolsGithubIndent) {
		return line, false
	}
	newLine, replaced := findAndReplaceInLine(line, "repos", "allowed-repos")
	if replaced {
		githubReposCodemodLog.Printf("Renamed 'repos' to 'allowed-repos' on line %d", index+1)
	}
	return newLine, replaced
}
