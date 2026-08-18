package cli

import (
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

var cliProxyBashCodemodLog = logger.New("cli:codemod_cli_proxy_bash")

// getCLIProxyBashDisabledCodemod disables shell-backed CLI modes when shell execution is refused.
//
// cli-proxy mounts MCP servers as CLI executables that can only be invoked from a shell, so it is
// incompatible with 'tools.bash: false' (or an empty bash allowlist). GitHub gh-proxy mode is
// shell-backed for the same reason.
func getCLIProxyBashDisabledCodemod() Codemod {
	return Codemod{
		ID:           "cli-proxy-false-when-bash-disabled",
		Name:         "Disable shell-backed CLI modes when 'tools.bash' is disabled",
		Description:  "Adds an explicit 'cli-proxy: false' (or disables 'cli-proxy: true') and rewrites GitHub gh-proxy mode to local when bash is disabled, since CLI-mounted MCP servers and gh-proxy require shell access.",
		IntroducedIn: "1.0.0",
		Apply: func(content string, frontmatter map[string]any) (string, bool, error) {
			return applyCLIProxyBashDisabledCodemod(content, frontmatter, "")
		},
		ApplyWithContext: func(content string, frontmatter map[string]any, filePath string) (string, bool, error) {
			return applyCLIProxyBashDisabledCodemod(content, frontmatter, filePath)
		},
	}
}

func applyCLIProxyBashDisabledCodemod(content string, frontmatter map[string]any, filePath string) (string, bool, error) {
	effectiveTools, err := resolveEffectiveTools(content, frontmatter, filePath)
	if err != nil {
		cliProxyBashCodemodLog.Printf("Failed to resolve effective tools: %v", err)
		effectiveTools, _ = frontmatter["tools"].(map[string]any)
	}
	if !toolsRefuseBash(effectiveTools) {
		return content, false, nil
	}

	needsCLIProxyFalse := !frontmatterHasCLIProxyDisabled(frontmatter)
	_, needsGitHubLocal := githubCLIProxyMode(effectiveTools)
	if !needsCLIProxyFalse && !needsGitHubLocal {
		return content, false, nil
	}

	newContent, applied, err := applyFrontmatterLineTransform(content, func(lines []string) ([]string, bool) {
		return setShellBackedModesDisabledInTools(lines, needsCLIProxyFalse, needsGitHubLocal)
	})
	if applied {
		cliProxyBashCodemodLog.Print("Disabled shell-backed CLI modes because tools.bash is disabled")
	}
	return newContent, applied, err
}

func toolsRefuseBash(toolsMap map[string]any) bool {
	bashValue, hasBash := toolsMap["bash"]
	if !hasBash {
		return false
	}
	switch v := bashValue.(type) {
	case bool:
		return !v
	case []any:
		return len(v) == 0
	}
	return false
}

// frontmatterHasCLIProxyDisabled reports whether tools.cli-proxy is already explicitly false.
func frontmatterHasCLIProxyDisabled(frontmatter map[string]any) bool {
	toolsMap, ok := frontmatter["tools"].(map[string]any)
	if !ok {
		return false
	}
	enabled, isBool := toolsMap["cli-proxy"].(bool)
	return isBool && !enabled
}

// setShellBackedModesDisabledInTools rewrites the top-level tools block so shell-backed tool
// paths are disabled when bash is disabled.
func setShellBackedModesDisabledInTools(lines []string, setCLIProxyFalse, setGitHubLocal bool) ([]string, bool) {
	toolsLine := -1
	// Only a top-level tools block is rewritten, so the block indentation is always empty.
	const toolsIndent = ""
	for i, line := range lines {
		if isTopLevelBlockKey(line, "tools") {
			toolsLine = i
			break
		}
	}
	if toolsLine == -1 {
		if hasTopLevelKey(lines, "tools") {
			cliProxyBashCodemodLog.Print("Top-level tools key is not block syntax, skipping")
			return lines, false
		}
		result := append([]string{}, lines...)
		result = append(result, "tools:")
		if setCLIProxyFalse {
			result = append(result, "  cli-proxy: false")
		}
		if setGitHubLocal {
			result = append(result, "  github:", "    mode: local")
		}
		return result, true
	}

	fieldIndent := toolsIndent + "  "
	insertAt := toolsLine + 1
	foundFirstField := false
	cliProxyLine := -1
	githubLine := -1
	githubIndent := ""
	githubModeLine := -1
	githubInsertAt := -1

	for i := toolsLine + 1; i < len(lines); i++ {
		line := lines[i]
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if hasExitedBlock(line, toolsIndent) {
			break
		}
		lineIndent := getIndentation(line)
		if !foundFirstField {
			fieldIndent = lineIndent
			insertAt = i
			foundFirstField = true
		}
		// Only rewrite a direct child of the tools block.
		if strings.HasPrefix(trimmed, "cli-proxy:") && lineIndent == fieldIndent {
			cliProxyLine = i
		}
		if isBlockKey(line, "github") && lineIndent == fieldIndent {
			githubLine = i
			githubIndent = lineIndent
			githubInsertAt = i + 1
			for j := i + 1; j < len(lines); j++ {
				githubChildLine := lines[j]
				githubChildTrimmed := strings.TrimSpace(githubChildLine)
				if githubChildTrimmed == "" || strings.HasPrefix(githubChildTrimmed, "#") {
					continue
				}
				if hasExitedBlock(githubChildLine, githubIndent) {
					break
				}
				githubChildIndent := getIndentation(githubChildLine)
				if githubInsertAt == i+1 {
					githubInsertAt = j
				}
				if strings.HasPrefix(githubChildTrimmed, "mode:") && githubChildIndent == githubIndent+"  " {
					githubModeLine = j
					break
				}
			}
		}
	}

	result := append([]string{}, lines...)
	applied := false

	if setCLIProxyFalse {
		if cliProxyLine >= 0 {
			result[cliProxyLine] = fieldIndent + "cli-proxy: false"
		} else {
			result = insertLine(result, insertAt, fieldIndent+"cli-proxy: false")
			if githubLine >= insertAt {
				githubLine++
			}
			if githubModeLine >= insertAt {
				githubModeLine++
			}
			if githubInsertAt >= insertAt {
				githubInsertAt++
			}
		}
		applied = true
	}

	if setGitHubLocal {
		if githubModeLine >= 0 {
			result[githubModeLine] = githubIndent + "  mode: local"
			applied = true
		} else if githubLine >= 0 {
			if githubInsertAt < 0 {
				githubInsertAt = githubLine + 1
			}
			result = insertLine(result, githubInsertAt, githubIndent+"  mode: local")
			applied = true
		} else {
			insertGithubAt := insertAt
			if setCLIProxyFalse {
				insertGithubAt++
			}
			result = insertLine(result, insertGithubAt, fieldIndent+"github:")
			result = insertLine(result, insertGithubAt+1, fieldIndent+"  mode: local")
			applied = true
		}
	}

	return result, applied
}

func insertLine(lines []string, index int, line string) []string {
	result := make([]string, 0, len(lines)+1)
	result = append(result, lines[:index]...)
	result = append(result, line)
	result = append(result, lines[index:]...)
	return result
}

func isTopLevelBlockKey(line, key string) bool {
	return getIndentation(line) == "" && isBlockKey(line, key)
}

func hasTopLevelKey(lines []string, key string) bool {
	prefix := key + ":"
	for _, line := range lines {
		if getIndentation(line) != "" {
			continue
		}
		if strings.HasPrefix(strings.TrimSpace(line), prefix) {
			return true
		}
	}
	return false
}

func isBlockKey(line, key string) bool {
	trimmed := strings.TrimSpace(line)
	prefix := key + ":"
	if !strings.HasPrefix(trimmed, prefix) {
		return false
	}
	rest := strings.TrimSpace(strings.TrimPrefix(trimmed, prefix))
	return rest == "" || strings.HasPrefix(rest, "#")
}

func githubCLIProxyMode(tools map[string]any) (string, bool) {
	githubValue, hasGitHub := tools["github"]
	if !hasGitHub {
		return "", false
	}
	githubMap, ok := githubValue.(map[string]any)
	if !ok {
		return "", false
	}
	modeValue, hasMode := githubMap["mode"]
	if !hasMode {
		return "", false
	}
	mode, ok := modeValue.(string)
	if !ok {
		return "", false
	}
	normalized := strings.ToLower(strings.TrimSpace(mode))
	return mode, normalized == "gh-proxy" || normalized == "cli"
}
