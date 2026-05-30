package cli

import "github.com/github/gh-aw/pkg/logger"

var grepToolCodemodLog = logger.New("cli:codemod_grep_tool")

// getGrepToolRemovalCodemod creates a codemod for removing the deprecated tools.grep field
func getGrepToolRemovalCodemod() Codemod {
	return newFieldRemovalCodemod(fieldRemovalCodemodConfig{
		ID:           "grep-tool-removal",
		Name:         "Remove deprecated tools.grep field",
		Description:  "Removes 'tools.grep' field as grep is now always enabled as part of default bash tools",
		IntroducedIn: "0.7.0",
		ParentKey:    "tools",
		FieldKey:     "grep",
		LogMsg:       "Applied grep tool removal",
		Log:          grepToolCodemodLog,
	})
}
