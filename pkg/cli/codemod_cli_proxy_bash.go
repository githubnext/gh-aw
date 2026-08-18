package cli

import (
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

var cliProxyBashCodemodLog = logger.New("cli:codemod_cli_proxy_bash")

// getCLIProxyBashDisabledCodemod sets tools.cli-proxy: false when shell execution is refused.
//
// cli-proxy mounts MCP servers as CLI executables that can only be invoked from a shell, so it is
// incompatible with 'tools.bash: false' (or an empty bash allowlist). The compiler requires the
// setting to be explicit so that the incompatibility is visible in the workflow source.
func getCLIProxyBashDisabledCodemod() Codemod {
	return Codemod{
		ID:           "cli-proxy-false-when-bash-disabled",
		Name:         "Set 'tools.cli-proxy: false' when 'tools.bash' is disabled",
		Description:  "Adds an explicit 'cli-proxy: false' (or disables 'cli-proxy: true') in the tools block when bash is disabled, since CLI-mounted MCP servers require shell access.",
		IntroducedIn: "1.0.0",
		Apply: func(content string, frontmatter map[string]any) (string, bool, error) {
			if !frontmatterRefusesBash(frontmatter) || frontmatterHasCLIProxyDisabled(frontmatter) {
				return content, false, nil
			}

			newContent, applied, err := applyFrontmatterLineTransform(content, setCLIProxyFalseInTools)
			if applied {
				cliProxyBashCodemodLog.Print("Set tools.cli-proxy: false because tools.bash is disabled")
			}
			return newContent, applied, err
		},
	}
}

// frontmatterRefusesBash reports whether tools.bash explicitly refuses shell execution,
// i.e. 'bash: false' or 'bash: []' (empty allowlist).
func frontmatterRefusesBash(frontmatter map[string]any) bool {
	toolsMap, ok := frontmatter["tools"].(map[string]any)
	if !ok {
		return false
	}
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

// setCLIProxyFalseInTools rewrites an existing 'cli-proxy:' entry in the tools block to false,
// or inserts 'cli-proxy: false' as the first entry of the tools block when absent.
func setCLIProxyFalseInTools(lines []string) ([]string, bool) {
	toolsLine := -1
	// Only a top-level tools block is rewritten, so the block indentation is always empty.
	const toolsIndent = ""
	for i, line := range lines {
		if strings.TrimSpace(line) == "tools:" && getIndentation(line) == toolsIndent {
			toolsLine = i
			break
		}
	}
	if toolsLine == -1 {
		cliProxyBashCodemodLog.Print("No top-level tools block found, skipping")
		return lines, false
	}

	fieldIndent := toolsIndent + "  "
	insertAt := toolsLine + 1
	foundFirstField := false

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
			result := make([]string, len(lines))
			copy(result, lines)
			result[i] = lineIndent + "cli-proxy: false"
			return result, true
		}
	}

	result := make([]string, 0, len(lines)+1)
	result = append(result, lines[:insertAt]...)
	result = append(result, fieldIndent+"cli-proxy: false")
	result = append(result, lines[insertAt:]...)
	return result, true
}
