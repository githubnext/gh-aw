package cli

import (
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

var bashAnonymousCodemodLog = logger.New("cli:codemod_bash_anonymous")

// getBashAnonymousRemovalCodemod creates a codemod for removing anonymous bash tool syntax
func getBashAnonymousRemovalCodemod() Codemod {
	return Codemod{
		ID:           "bash-anonymous-removal",
		Name:         "Replace anonymous bash tool syntax with explicit true",
		Description:  "Replaces 'bash:' (anonymous/nil syntax) with 'bash: true' to make configuration explicit",
		IntroducedIn: "0.9.0",
		Apply: func(content string, frontmatter map[string]any) (string, bool, error) {
			// Check if tools.bash exists
			toolsValue, hasTools := frontmatter["tools"]
			if !hasTools {
				return content, false, nil
			}

			toolsMap, ok := toolsValue.(map[string]any)
			if !ok {
				return content, false, nil
			}

			// Check if bash field exists and is nil
			bashValue, hasBash := toolsMap["bash"]
			if !hasBash {
				return content, false, nil
			}

			// Only modify if bash is nil (anonymous syntax)
			if bashValue != nil {
				return content, false, nil
			}

			newContent, applied, err := applyFrontmatterLineTransform(content, replaceBashAnonymousWithTrue)
			if applied {
				bashAnonymousCodemodLog.Print("Applied bash anonymous removal, replaced with 'bash: true'")
			}
			return newContent, applied, err
		},
	}
}

// replaceBashAnonymousWithTrue replaces 'bash:' with 'bash: true' in the tools block
func replaceBashAnonymousWithTrue(lines []string) ([]string, bool) {
	var result []string
	var modified bool
	var inToolsBlock bool
	var toolsIndent string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Track if we're in the tools block
		if trimmed == "tools:" {
			inToolsBlock = true
			toolsIndent = getIndentation(line)
			result = append(result, line)
			continue
		}

		// Check if we've left the tools block
		if inToolsBlock && len(trimmed) > 0 && !strings.HasPrefix(trimmed, "#") {
			if hasExitedBlock(line, toolsIndent) {
				inToolsBlock = false
			}
		}

		// Replace bash: with bash: true if in tools block
		if inToolsBlock && (trimmed == "bash:" || strings.HasPrefix(trimmed, "bash: ")) {
			// Check if it's just 'bash:' with nothing after the colon
			if trimmed == "bash:" {
				indent := getIndentation(line)
				result = append(result, indent+"bash: true")
				modified = true
				bashAnonymousCodemodLog.Printf("Replaced 'bash:' with 'bash: true'")
				continue
			}
		}

		result = append(result, line)
	}

	return result, modified
}
