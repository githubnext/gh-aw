package cli

import (
	"slices"
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

var githubAppClientIDCodemodLog = logger.New("cli:codemod_github_app_client_id")

// getGitHubAppClientIDCodemod creates a codemod that migrates github-app.app-id to github-app.client-id.
func getGitHubAppClientIDCodemod() Codemod {
	return Codemod{
		ID:           "github-app-app-id-to-client-id",
		Name:         "Rename 'github-app.app-id' to 'github-app.client-id'",
		Description:  "Renames deprecated 'app-id:' to 'client-id:' inside github-app blocks.",
		IntroducedIn: "0.68.4",
		Apply: func(content string, frontmatter map[string]any) (string, bool, error) {
			if !hasDeprecatedGitHubAppIDField(frontmatter) {
				return content, false, nil
			}
			newContent, applied, err := applyFrontmatterLineTransform(content, renameGitHubAppIDToClientID)
			if applied {
				githubAppClientIDCodemodLog.Print("Renamed github-app app-id to client-id")
			}
			return newContent, applied, err
		},
	}
}

// hasDeprecatedGitHubAppIDField returns true when any github-app object contains app-id.
func hasDeprecatedGitHubAppIDField(frontmatter map[string]any) bool {
	return containsDeprecatedGitHubAppID(frontmatter)
}

func containsDeprecatedGitHubAppID(value any) bool {
	switch v := value.(type) {
	case map[string]any:
		for key, child := range v {
			if key == "github-app" {
				if appMap, ok := child.(map[string]any); ok {
					if _, hasAppID := appMap["app-id"]; hasAppID {
						return true
					}
				}
			}
			if containsDeprecatedGitHubAppID(child) {
				return true
			}
		}
	case []any:
		return slices.ContainsFunc(v, containsDeprecatedGitHubAppID)
	}
	return false
}

func renameGitHubAppIDToClientID(lines []string) ([]string, bool) {
	result := make([]string, 0, len(lines))
	modified := false

	inGitHubApp := false
	var githubAppIndent string

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		if len(trimmed) == 0 {
			result = append(result, line)
			continue
		}

		if !strings.HasPrefix(trimmed, "#") && inGitHubApp && hasExitedBlock(line, githubAppIndent) {
			inGitHubApp = false
		}

		if strings.HasPrefix(trimmed, "github-app:") {
			inGitHubApp = true
			githubAppIndent = getIndentation(line)
			result = append(result, line)
			continue
		}

		if inGitHubApp {
			lineIndent := getIndentation(line)
			if isDescendant(lineIndent, githubAppIndent) && strings.HasPrefix(trimmed, "app-id:") {
				newLine, replaced := findAndReplaceInLine(line, "app-id", "client-id")
				if replaced {
					result = append(result, newLine)
					modified = true
					githubAppClientIDCodemodLog.Printf("Renamed github-app app-id on line %d", i+1)
					continue
				}
			}
		}

		result = append(result, line)
	}

	return result, modified
}
