package cli

import (
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

var assignToAgentCodemodLog = logger.New("cli:codemod_assign_to_agent")

// getAssignToAgentDefaultAgentCodemod creates a codemod for migrating the deprecated 'default-agent' key
// to the canonical 'name' key inside safe-outputs.assign-to-agent
func getAssignToAgentDefaultAgentCodemod() Codemod {
	return Codemod{
		ID:           "assign-to-agent-default-agent-to-name",
		Name:         "Migrate assign-to-agent default-agent to name",
		Description:  "Renames the deprecated 'default-agent' field to 'name' inside 'safe-outputs.assign-to-agent'",
		IntroducedIn: "0.12.0",
		Apply: func(content string, frontmatter map[string]any) (string, bool, error) {
			// Check if safe-outputs.assign-to-agent.default-agent exists
			safeOutputsValue, hasSafeOutputs := frontmatter["safe-outputs"]
			if !hasSafeOutputs {
				return content, false, nil
			}

			safeOutputsMap, ok := safeOutputsValue.(map[string]any)
			if !ok {
				return content, false, nil
			}

			assignToAgentValue, hasAssignToAgent := safeOutputsMap["assign-to-agent"]
			if !hasAssignToAgent {
				return content, false, nil
			}

			assignToAgentMap, ok := assignToAgentValue.(map[string]any)
			if !ok {
				return content, false, nil
			}

			// Check if deprecated 'default-agent' key exists
			_, hasDefaultAgent := assignToAgentMap["default-agent"]
			if !hasDefaultAgent {
				return content, false, nil
			}

			// Don't migrate if 'name' already exists to avoid overwriting it
			_, hasName := assignToAgentMap["name"]
			if hasName {
				assignToAgentCodemodLog.Print("Skipping migration: 'name' already exists in assign-to-agent config")
				return content, false, nil
			}

			// Parse frontmatter to get raw lines
			frontmatterLines, markdown, err := parseFrontmatterLines(content)
			if err != nil {
				return content, false, err
			}

			var modified bool
			var inSafeOutputsBlock bool
			var safeOutputsIndent string
			var inAssignToAgentBlock bool
			var assignToAgentIndent string

			result := make([]string, len(frontmatterLines))

			for i, line := range frontmatterLines {
				trimmedLine := strings.TrimSpace(line)

				// Track if we're in the safe-outputs block
				if strings.HasPrefix(trimmedLine, "safe-outputs:") {
					inSafeOutputsBlock = true
					safeOutputsIndent = getIndentation(line)
					result[i] = line
					continue
				}

				// Check if we've left the safe-outputs block
				if inSafeOutputsBlock && len(trimmedLine) > 0 && !strings.HasPrefix(trimmedLine, "#") {
					if hasExitedBlock(line, safeOutputsIndent) {
						inSafeOutputsBlock = false
						inAssignToAgentBlock = false
					}
				}

				// Track if we're in the assign-to-agent block within safe-outputs
				if inSafeOutputsBlock && strings.HasPrefix(trimmedLine, "assign-to-agent:") {
					inAssignToAgentBlock = true
					assignToAgentIndent = getIndentation(line)
					result[i] = line
					continue
				}

				// Check if we've left the assign-to-agent block
				if inAssignToAgentBlock && len(trimmedLine) > 0 && !strings.HasPrefix(trimmedLine, "#") {
					if hasExitedBlock(line, assignToAgentIndent) {
						inAssignToAgentBlock = false
					}
				}

				// Replace default-agent with name if in assign-to-agent block
				if inAssignToAgentBlock && strings.HasPrefix(trimmedLine, "default-agent:") {
					replacedLine, didReplace := findAndReplaceInLine(line, "default-agent", "name")
					if didReplace {
						result[i] = replacedLine
						modified = true
						assignToAgentCodemodLog.Printf("Replaced safe-outputs.assign-to-agent.default-agent with safe-outputs.assign-to-agent.name on line %d", i+1)
					} else {
						result[i] = line
					}
				} else {
					result[i] = line
				}
			}

			if !modified {
				return content, false, nil
			}

			newContent := reconstructContent(result, markdown)
			assignToAgentCodemodLog.Print("Applied assign-to-agent default-agent to name migration")
			return newContent, true, nil
		},
	}
}
