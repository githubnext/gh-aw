package cli

import (
	"github.com/github/gh-aw/pkg/logger"
)

var safeInputsToMCPScriptsCodemodLog = logger.New("cli:codemod_safe_inputs_to_mcp_scripts")

// getSafeInputsToMCPScriptsCodemod creates a codemod for renaming the safe-inputs key to mcp-scripts
func getSafeInputsToMCPScriptsCodemod() Codemod {
	return Codemod{
		ID:           "safe-inputs-to-mcp-scripts",
		Name:         "Rename safe-inputs to mcp-scripts",
		Description:  "Renames the top-level 'safe-inputs' key to 'mcp-scripts' in workflow frontmatter",
		IntroducedIn: "0.3.0",
		Apply: func(content string, frontmatter map[string]any) (string, bool, error) {
			// Check if safe-inputs key exists in frontmatter
			_, hasSafeInputs := frontmatter["safe-inputs"]
			if !hasSafeInputs {
				return content, false, nil
			}

			newContent, applied, err := applyFrontmatterLineTransform(content, func(lines []string) ([]string, bool) {
				return renameTopLevelKey(lines, "safe-inputs", "mcp-scripts")
			})
			if applied {
				safeInputsToMCPScriptsCodemodLog.Print("Applied safe-inputs to mcp-scripts rename")
			}
			return newContent, applied, err
		},
	}
}

// renameTopLevelKey renames a top-level YAML key from oldKey to newKey, preserving formatting.
func renameTopLevelKey(lines []string, oldKey, newKey string) ([]string, bool) {
	var result []string
	applied := false

	for _, line := range lines {
		newLine, changed := findAndReplaceInLine(line, oldKey, newKey)
		result = append(result, newLine)
		if changed {
			applied = true
		}
	}

	return result, applied
}
