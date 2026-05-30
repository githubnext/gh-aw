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
		Apply: func(content string, frontmatter map[string]any) (string, bool, error) {
			// Check if on.command exists
			onValue, hasOn := frontmatter["on"]
			if !hasOn {
				return content, false, nil
			}

			onMap, ok := onValue.(map[string]any)
			if !ok {
				return content, false, nil
			}

			// Check if command field exists in on
			_, hasCommand := onMap["command"]
			if !hasCommand {
				return content, false, nil
			}

			newContent, applied, err := applyFrontmatterLineTransform(content, func(lines []string) ([]string, bool) {
				var modified bool
				var inOnBlock bool
				var onIndent string
				result := make([]string, len(lines))
				for i, line := range lines {
					trimmedLine := strings.TrimSpace(line)

					// Track if we're in the on block
					if strings.HasPrefix(trimmedLine, "on:") {
						inOnBlock = true
						onIndent = getIndentation(line)
						result[i] = line
						continue
					}

					// Check if we've left the on block
					if inOnBlock && len(trimmedLine) > 0 && !strings.HasPrefix(trimmedLine, "#") {
						if hasExitedBlock(line, onIndent) {
							inOnBlock = false
						}
					}

					// Replace command with slash_command if in on block
					if inOnBlock && strings.HasPrefix(trimmedLine, "command:") {
						replacedLine, didReplace := findAndReplaceInLine(line, "command", "slash_command")
						if didReplace {
							result[i] = replacedLine
							modified = true
							slashCommandCodemodLog.Printf("Replaced on.command with on.slash_command on line %d", i+1)
						} else {
							result[i] = line
						}
					} else {
						result[i] = line
					}
				}
				return result, modified
			})
			if applied {
				slashCommandCodemodLog.Print("Applied on.command to on.slash_command migration")
			}
			return newContent, applied, err
		},
	}
}
