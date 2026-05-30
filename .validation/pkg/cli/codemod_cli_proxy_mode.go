package cli

import (
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

var cliProxyModeCodemodLog = logger.New("cli:codemod_cli_proxy_mode")

// getCliProxyFeatureToGitHubModeCodemod migrates features.cli-proxy: true to tools.github.mode: gh-proxy.
func getCliProxyFeatureToGitHubModeCodemod() Codemod {
	return Codemod{
		ID:           "features-cli-proxy-to-tools-github-mode",
		Name:         "Migrate 'features.cli-proxy: true' to 'tools.github.mode: gh-proxy'",
		Description:  "Removes deprecated features.cli-proxy: true and sets tools.github.mode: gh-proxy (equivalent behavior).",
		IntroducedIn: "1.0.0",
		Apply: func(content string, frontmatter map[string]any) (string, bool, error) {
			if !hasLegacyCliProxyFeatureEnabled(frontmatter) {
				return content, false, nil
			}
			hasMode := hasToolsGitHubMode(frontmatter)

			newContent, applied, err := applyFrontmatterLineTransform(content, func(lines []string) ([]string, bool) {
				result, modified := removeFieldFromBlock(lines, "cli-proxy", "features")
				if !modified {
					return lines, false
				}
				if !hasMode {
					result = addGitHubModeGhProxyToTools(result)
				}
				return result, true
			})
			if applied {
				cliProxyModeCodemodLog.Print("Migrated features.cli-proxy: true to tools.github.mode: gh-proxy")
			}
			return newContent, applied, err
		},
	}
}

func hasLegacyCliProxyFeatureEnabled(frontmatter map[string]any) bool {
	featuresAny, ok := frontmatter["features"]
	if !ok {
		return false
	}
	featuresMap, ok := featuresAny.(map[string]any)
	if !ok {
		return false
	}
	value, has := featuresMap["cli-proxy"]
	if !has {
		return false
	}
	enabled, ok := value.(bool)
	return ok && enabled
}

func hasToolsGitHubMode(frontmatter map[string]any) bool {
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
	_, hasMode := githubMap["mode"]
	return hasMode
}

func addGitHubModeGhProxyToTools(lines []string) []string {
	toolsLine := -1
	toolsIndent := ""

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "tools:") {
			toolsLine = i
			toolsIndent = getIndentation(line)
			break
		}
	}

	if toolsLine == -1 {
		return append(lines, "tools:", "  github:", "    mode: gh-proxy")
	}

	githubLine := -1
	githubIndent := ""
	toolsEnd := len(lines)
	for i := toolsLine + 1; i < len(lines); i++ {
		trimmed := strings.TrimSpace(lines[i])
		if len(trimmed) > 0 && !strings.HasPrefix(trimmed, "#") && hasExitedBlock(lines[i], toolsIndent) {
			toolsEnd = i
			break
		}
		if strings.HasPrefix(trimmed, "github:") && strings.HasPrefix(getIndentation(lines[i]), toolsIndent+"  ") {
			githubLine = i
			githubIndent = getIndentation(lines[i])
			break
		}
	}

	if githubLine == -1 {
		result := make([]string, 0, len(lines)+2)
		result = append(result, lines[:toolsEnd]...)
		result = append(result, toolsIndent+"  github:")
		result = append(result, toolsIndent+"    mode: gh-proxy")
		result = append(result, lines[toolsEnd:]...)
		return result
	}

	if strings.TrimSpace(lines[githubLine]) != "github:" {
		cliProxyModeCodemodLog.Print("Skipping mode addition: github line has inline content")
		return lines
	}

	fieldIndent := githubIndent + "  "
	insertAt := githubLine + 1
	for i := githubLine + 1; i < len(lines); i++ {
		trimmed := strings.TrimSpace(lines[i])
		if len(trimmed) > 0 && !strings.HasPrefix(trimmed, "#") {
			if hasExitedBlock(lines[i], githubIndent) {
				insertAt = i
			} else {
				fieldIndent = getIndentation(lines[i])
				insertAt = i
			}
			break
		}
	}

	result := make([]string, 0, len(lines)+1)
	result = append(result, lines[:insertAt]...)
	result = append(result, fieldIndent+"mode: gh-proxy")
	result = append(result, lines[insertAt:]...)
	return result
}
