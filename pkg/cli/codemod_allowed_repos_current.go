package cli

import (
	"fmt"
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

var allowedReposCurrentCodemodLog = logger.New("cli:codemod_allowed_repos_current")

// getAllowedReposCurrentToGitHubRepositoryCodemod creates a codemod that migrates the
// unsupported legacy 'current' alias for tools.github.allowed-repos to the equivalent
// '${{ github.repository }}' expression, which is the only accepted way to scope
// access to the current repository.
func getAllowedReposCurrentToGitHubRepositoryCodemod() Codemod {
	return Codemod{
		ID:           "allowed-repos-current-to-github-repository",
		Name:         "Migrate 'tools.github.allowed-repos: current' to '${{ github.repository }}'",
		Description:  "Rewrites the legacy 'current' alias for tools.github.allowed-repos to the accepted '${{ github.repository }}' expression.",
		IntroducedIn: "1.0.0",
		Apply: func(content string, frontmatter map[string]any) (string, bool, error) {
			if !hasAllowedReposCurrentValue(frontmatter) {
				return content, false, nil
			}
			newContent, applied, err := applyFrontmatterLineTransform(content, rewriteAllowedReposCurrentValue)
			if applied {
				allowedReposCurrentCodemodLog.Print("Migrated 'tools.github.allowed-repos: current' to '${{ github.repository }}'")
			}
			return newContent, applied, err
		},
	}
}

// hasAllowedReposCurrentValue returns true if tools.github.allowed-repos is set to the
// legacy string value 'current'.
func hasAllowedReposCurrentValue(frontmatter map[string]any) bool {
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
	allowedReposAny, hasAllowedRepos := githubMap["allowed-repos"]
	if !hasAllowedRepos {
		return false
	}
	allowedReposStr, ok := allowedReposAny.(string)
	if !ok {
		return false
	}
	return allowedReposStr == "current"
}

// rewriteAllowedReposCurrentValue rewrites the value of 'allowed-repos: current' (or any
// quoted variant) to '${{ github.repository }}' within the tools.github configuration block.
func rewriteAllowedReposCurrentValue(lines []string) ([]string, bool) {
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

		// Rewrite the value of 'allowed-repos: current' when inside tools.github
		if inToolsGithub && strings.HasPrefix(trimmed, "allowed-repos:") {
			lineIndent := getIndentation(line)
			if isDescendant(lineIndent, toolsGithubIndent) {
				if newLine, replaced := replaceAllowedReposCurrentLineValue(line); replaced {
					result = append(result, newLine)
					modified = true
					allowedReposCurrentCodemodLog.Printf("Migrated 'allowed-repos: current' on line %d", i+1)
					continue
				}
			}
		}

		result = append(result, line)
	}

	return result, modified
}

// replaceAllowedReposCurrentLineValue replaces the value of an 'allowed-repos:' line with
// '${{ github.repository }}' if the current value (unquoted, single-quoted, or double-quoted)
// is 'current'. Preserves indentation and trailing comments.
func replaceAllowedReposCurrentLineValue(line string) (string, bool) {
	parts := strings.SplitN(line, ":", 2)
	if len(parts) < 2 {
		return line, false
	}

	leadingSpace := getIndentation(line)
	valuePart := strings.TrimSpace(parts[1])

	// Split off any trailing comment
	value := valuePart
	comment := ""
	if idx := strings.Index(valuePart, "#"); idx >= 0 {
		value = strings.TrimSpace(valuePart[:idx])
		comment = " " + valuePart[idx:]
	}

	unquoted := strings.Trim(value, `"'`)
	if unquoted != "current" {
		return line, false
	}

	allowedReposCurrentCodemodLog.Print("Replacing 'current' value with '${{ github.repository }}'")
	return fmt.Sprintf(`%sallowed-repos: "${{ github.repository }}"%s`, leadingSpace, comment), true
}
