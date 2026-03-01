package cli

import (
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

var serenaLocalModeCodemodLog = logger.New("cli:codemod_serena_local_mode")

// getSerenaLocalModeCodemod creates a codemod that replaces 'mode: local' with 'mode: docker'
// in tools.serena configurations. The 'local' mode executed serena via uvx directly from an
// unpinned git repository (supply chain risk) and has been removed. Docker is now the only
// supported mode.
func getSerenaLocalModeCodemod() Codemod {
	return Codemod{
		ID:           "serena-local-to-docker",
		Name:         "Migrate Serena 'mode: local' to 'mode: docker'",
		Description:  "Replaces 'mode: local' with 'mode: docker' in tools.serena configurations. The 'local' mode has been removed as it executed serena from an unpinned git repository (supply chain risk). Docker is now the only supported mode.",
		IntroducedIn: "0.17.0",
		Apply: func(content string, frontmatter map[string]any) (string, bool, error) {
			// Check if tools.serena exists and has mode: local
			toolsValue, hasTools := frontmatter["tools"]
			if !hasTools {
				return content, false, nil
			}

			toolsMap, ok := toolsValue.(map[string]any)
			if !ok {
				return content, false, nil
			}

			serenaValue, hasSerena := toolsMap["serena"]
			if !hasSerena {
				return content, false, nil
			}

			serenaMap, ok := serenaValue.(map[string]any)
			if !ok {
				return content, false, nil
			}

			modeValue, hasMode := serenaMap["mode"]
			if !hasMode {
				return content, false, nil
			}

			modeStr, ok := modeValue.(string)
			if !ok || modeStr != "local" {
				return content, false, nil
			}

			newContent, applied, err := applyFrontmatterLineTransform(content, replaceSerenaLocalModeWithDocker)
			if applied {
				serenaLocalModeCodemodLog.Print("Applied Serena local-to-docker migration")
			}
			return newContent, applied, err
		},
	}
}

// replaceSerenaLocalModeWithDocker replaces 'mode: local' with 'mode: docker' within the
// tools.serena block in frontmatter lines.
func replaceSerenaLocalModeWithDocker(lines []string) ([]string, bool) {
	var result []string
	var modified bool
	var inTools bool
	var toolsIndent string
	var inSerena bool
	var serenaIndent string

	for i, line := range lines {
		trimmedLine := strings.TrimSpace(line)

		// Track entering the tools block
		if strings.HasPrefix(trimmedLine, "tools:") && !inTools {
			inTools = true
			toolsIndent = getIndentation(line)
			result = append(result, line)
			continue
		}

		// Check if we've left the tools block
		if inTools && len(trimmedLine) > 0 && !strings.HasPrefix(trimmedLine, "#") {
			if hasExitedBlock(line, toolsIndent) {
				inTools = false
				inSerena = false
			}
		}

		// Track entering the serena sub-block inside tools
		if inTools && strings.HasPrefix(trimmedLine, "serena:") {
			inSerena = true
			serenaIndent = getIndentation(line)
			result = append(result, line)
			continue
		}

		// Check if we've left the serena block
		if inSerena && len(trimmedLine) > 0 && !strings.HasPrefix(trimmedLine, "#") {
			if hasExitedBlock(line, serenaIndent) {
				inSerena = false
			}
		}

		// Replace 'mode: local' with 'mode: docker' inside tools.serena
		if inSerena && strings.HasPrefix(trimmedLine, "mode:") {
			if strings.Contains(trimmedLine, "local") {
				newLine, replaced := findAndReplaceValueInLine(line, "mode", "local", "docker")
				if replaced {
					result = append(result, newLine)
					modified = true
					serenaLocalModeCodemodLog.Printf("Replaced 'mode: local' with 'mode: docker' on line %d", i+1)
					continue
				}
			}
		}

		result = append(result, line)
	}

	return result, modified
}

// findAndReplaceValueInLine replaces oldValue with newValue for a specific key in a YAML line,
// preserving indentation and inline comments.
func findAndReplaceValueInLine(line, key, oldValue, newValue string) (string, bool) {
	trimmedLine := strings.TrimSpace(line)
	if !strings.HasPrefix(trimmedLine, key+":") {
		return line, false
	}

	leadingSpace := getIndentation(line)
	_, afterColon, found := strings.Cut(line, ":")
	if !found {
		return line, false
	}

	// Split on the first '#' to separate value from inline comment
	commentIdx := strings.Index(afterColon, "#")
	var valueSection, commentSection string
	if commentIdx >= 0 {
		valueSection = afterColon[:commentIdx]
		commentSection = afterColon[commentIdx:]
	} else {
		valueSection = afterColon
		commentSection = ""
	}

	trimmedValue := strings.TrimSpace(valueSection)
	if trimmedValue != oldValue {
		return line, false
	}

	// Preserve the whitespace between the colon and the value
	spaceBeforeValue := valueSection[:strings.Index(valueSection, trimmedValue)]
	newLine := leadingSpace + key + ":" + spaceBeforeValue + newValue
	if commentSection != "" {
		newLine += " " + commentSection
	}
	return newLine, true
}
