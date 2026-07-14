package cli

import (
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

var playwrightModeCodemodLog = logger.New("cli:codemod_playwright_mcp_mode_to_cli")

// getPlaywrightMCPModeToCLICodemod creates a codemod that ensures tools.playwright.mode is set to cli.
func getPlaywrightMCPModeToCLICodemod() Codemod {
	return Codemod{
		ID:           "playwright-mcp-mode-to-cli",
		Name:         "Migrate playwright MCP mode to CLI mode",
		Description:  "Ensures tools.playwright.mode is explicitly set to 'cli' so workflows avoid deprecated MCP auto-derivation.",
		IntroducedIn: "0.9.0",
		Apply: func(content string, frontmatter map[string]any) (string, bool, error) {
			toolsValue, hasTools := frontmatter["tools"]
			if !hasTools {
				return content, false, nil
			}

			toolsMap, ok := toolsValue.(map[string]any)
			if !ok {
				return content, false, nil
			}

			playwrightValue, hasPlaywright := toolsMap["playwright"]
			if !hasPlaywright {
				return content, false, nil
			}

			switch v := playwrightValue.(type) {
			case bool:
				if !v {
					return content, false, nil
				}
			case map[string]any:
				if mode, ok := v["mode"].(string); ok && strings.EqualFold(strings.TrimSpace(mode), "cli") {
					return content, false, nil
				}
			}

			newContent, applied, err := applyFrontmatterLineTransform(content, ensurePlaywrightModeCLI)
			if applied {
				playwrightModeCodemodLog.Print("Applied tools.playwright.mode: cli migration")
			}
			return newContent, applied, err
		},
	}
}

func ensurePlaywrightModeCLI(lines []string) ([]string, bool) {
	var result []string
	var modified bool
	var inTools bool
	var toolsIndent string
	var inPlaywright bool
	var playwrightIndent string
	var playwrightChildIndent string
	var modePresent bool

	appendMode := func() {
		indent := playwrightChildIndent
		if indent == "" {
			indent = playwrightIndent + "  "
		}
		result = append(result, indent+"mode: cli")
		modified = true
	}

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if inTools && inPlaywright && trimmed != "" && !strings.HasPrefix(trimmed, "#") && hasExitedBlock(line, playwrightIndent) {
			if !modePresent {
				appendMode()
			}
			inPlaywright = false
			playwrightChildIndent = ""
			modePresent = false
		}

		if inTools && trimmed != "" && !strings.HasPrefix(trimmed, "#") && hasExitedBlock(line, toolsIndent) {
			inTools = false
			inPlaywright = false
			playwrightChildIndent = ""
			modePresent = false
		}

		if strings.HasPrefix(trimmed, "tools:") {
			inTools = true
			toolsIndent = getIndentation(line)
			result = append(result, line)
			continue
		}

		if inTools && strings.HasPrefix(trimmed, "playwright:") {
			playwrightIndent = getIndentation(line)
			value := strings.TrimSpace(strings.TrimPrefix(trimmed, "playwright:"))
			if value == "" || strings.HasPrefix(value, "#") {
				inPlaywright = true
				playwrightChildIndent = ""
				modePresent = false
				result = append(result, line)
				continue
			}
			if strings.HasPrefix(value, "false") {
				result = append(result, line)
				continue
			}

			result = append(result, playwrightIndent+"playwright:")
			appendMode()
			continue
		}

		if inPlaywright && trimmed != "" && !strings.HasPrefix(trimmed, "#") {
			lineIndent := getIndentation(line)
			if playwrightChildIndent == "" && isDescendant(lineIndent, playwrightIndent) {
				playwrightChildIndent = lineIndent
			}
		}

		if inPlaywright && strings.HasPrefix(trimmed, "mode:") {
			lineIndent := getIndentation(line)
			if !isDescendant(lineIndent, playwrightIndent) {
				result = append(result, line)
				continue
			}
			if playwrightChildIndent == "" {
				playwrightChildIndent = lineIndent
			}

			modePresent = true
			if isModeValueCLI(trimmed) {
				result = append(result, line)
				continue
			}
			result = append(result, lineIndent+"mode: cli")
			modified = true
			continue
		}

		result = append(result, line)
	}

	if inTools && inPlaywright && !modePresent {
		appendMode()
	}

	return result, modified
}

func isModeValueCLI(modeLine string) bool {
	modeValue := strings.TrimSpace(strings.TrimPrefix(modeLine, "mode:"))
	if parts := strings.SplitN(modeValue, "#", 2); len(parts) > 1 {
		modeValue = strings.TrimSpace(parts[0])
	}
	return strings.EqualFold(modeValue, "cli")
}
