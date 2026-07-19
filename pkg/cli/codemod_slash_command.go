package cli

import (
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

var slashCommandCodemodLog = logger.New("cli:codemod_slash_command")

// getCommandToSlashCommandCodemod creates a codemod for migrating on.command to on.slash_command
func getCommandToSlashCommandCodemod() Codemod {
	return Codemod{
		ID:           "command-to-slash-command-migration",
		Name:         "Migrate on.command to on.slash_command",
		Description:  "Replaces deprecated 'on.command' field with 'on.slash_command'",
		IntroducedIn: "0.2.0",
		Apply:        getCommandToSlashCommandCodemodApply,
	}
}

func getCommandToSlashCommandCodemodApply(content string, frontmatter map[string]any) (string, bool, error) {
	if !getCommandToSlashCommandCodemodNeedsMigration(frontmatter) {
		return content, false, nil
	}

	newContent, applied, err := applyFrontmatterLineTransform(content, getCommandToSlashCommandCodemodTransform)
	if applied {
		slashCommandCodemodLog.Print("Applied on.command to on.slash_command migration")
	}
	return newContent, applied, err
}

func getCommandToSlashCommandCodemodNeedsMigration(frontmatter map[string]any) bool {
	// Check if on.command exists
	onValue, hasOn := frontmatter["on"]
	if !hasOn {
		return false
	}

	onMap, ok := onValue.(map[string]any)
	if !ok {
		return false
	}

	// Check if command field exists in on
	_, hasCommand := onMap["command"]
	return hasCommand
}

func getCommandToSlashCommandCodemodTransform(lines []string) ([]string, bool) {
	var modified bool
	var inOnBlock bool
	var onIndent string
	result := make([]string, len(lines))
	for i, line := range lines {
		trimmedLine := strings.TrimSpace(line)
		inOnBlock, onIndent = getCommandToSlashCommandCodemodTrackOnBlock(line, trimmedLine, inOnBlock, onIndent)
		result[i], modified = getCommandToSlashCommandCodemodReplaceLine(line, trimmedLine, inOnBlock, i, modified)
	}
	return result, modified
}

func getCommandToSlashCommandCodemodTrackOnBlock(line, trimmedLine string, inOnBlock bool, onIndent string) (bool, string) {
	// Track if we're in the on block
	if strings.HasPrefix(trimmedLine, "on:") {
		return true, getIndentation(line)
	}

	// Check if we've left the on block
	if inOnBlock && trimmedLine != "" && !strings.HasPrefix(trimmedLine, "#") && hasExitedBlock(line, onIndent) {
		return false, onIndent
	}
	return inOnBlock, onIndent
}

func getCommandToSlashCommandCodemodReplaceLine(line, trimmedLine string, inOnBlock bool, lineIndex int, modified bool) (string, bool) {
	// Replace command with slash_command if in on block
	if !inOnBlock || !strings.HasPrefix(trimmedLine, "command:") {
		return line, modified
	}
	replacedLine, didReplace := findAndReplaceInLine(line, "command", "slash_command")
	if didReplace {
		slashCommandCodemodLog.Printf("Replaced on.command with on.slash_command on line %d", lineIndex+1)
		return replacedLine, true
	}
	return line, modified
}
