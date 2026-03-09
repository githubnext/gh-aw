package cli

import (
	"github.com/github/gh-aw/pkg/logger"
)

var mcpScriptsModeCodemodLog = logger.New("cli:codemod_mcp_scripts")

// getMCPScriptsModeCodemod creates a codemod for removing the deprecated mcp-scripts.mode field
func getMCPScriptsModeCodemod() Codemod {
	return Codemod{
		ID:           "mcp-scripts-mode-removal",
		Name:         "Remove deprecated mcp-scripts.mode field",
		Description:  "Removes the deprecated 'mcp-scripts.mode' field (HTTP is now the only supported mode)",
		IntroducedIn: "0.2.0",
		Apply: func(content string, frontmatter map[string]any) (string, bool, error) {
			// Check if mcp-scripts.mode exists
			mcpScriptsValue, hasMCPScripts := frontmatter["mcp-scripts"]
			if !hasMCPScripts {
				return content, false, nil
			}

			mcpScriptsMap, ok := mcpScriptsValue.(map[string]any)
			if !ok {
				return content, false, nil
			}

			// Check if mode field exists in mcp-scripts
			_, hasMode := mcpScriptsMap["mode"]
			if !hasMode {
				return content, false, nil
			}

			newContent, applied, err := applyFrontmatterLineTransform(content, func(lines []string) ([]string, bool) {
				return removeFieldFromBlock(lines, "mode", "mcp-scripts")
			})
			if applied {
				mcpScriptsModeCodemodLog.Print("Applied mcp-scripts.mode removal")
			}
			return newContent, applied, err
		},
	}
}
