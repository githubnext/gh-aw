package cli

import (
	"github.com/github/gh-aw/pkg/logger"
)

var mcpScriptsModeCodemodLog = logger.New("cli:codemod_mcp_scripts")

// getMCPScriptsModeCodemod creates a codemod for removing the deprecated mcp-scripts.mode field
func getMCPScriptsModeCodemod() Codemod {
	return newFieldRemovalCodemod(fieldRemovalCodemodConfig{
		ID:           "mcp-scripts-mode-removal",
		Name:         "Remove deprecated mcp-scripts.mode field",
		Description:  "Removes the deprecated 'mcp-scripts.mode' field (HTTP is now the only supported mode)",
		IntroducedIn: "0.2.0",
		ParentKey:    "mcp-scripts",
		FieldKey:     "mode",
		LogMsg:       "Applied mcp-scripts.mode removal",
		Log:          mcpScriptsModeCodemodLog,
	})
}
