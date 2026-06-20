package cli

import (
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

var mentionsAllowTeamMembersCodemodLog = logger.New("cli:codemod_mentions_allow_team_members")

func getMentionsAllowTeamMembersCodemod() Codemod {
	return Codemod{
		ID:           "mentions-allow-team-members-to-allowed-collaborators",
		Name:         "Rename allow-team-members to allowed-collaborators in mentions",
		Description:  "Renames allow-team-members to allowed-collaborators in safe-outputs.mentions.",
		IntroducedIn: "1.0.0",
		Apply: func(content string, frontmatter map[string]any) (string, bool, error) {
			if !mentionsAllowTeamMembersNeedsMigration(frontmatter) {
				return content, false, nil
			}

			newContent, applied, err := applyFrontmatterLineTransform(content, func(lines []string) ([]string, bool) {
				return renameMentionsAllowTeamMembers(lines)
			})
			if applied {
				mentionsAllowTeamMembersCodemodLog.Print("Renamed allow-team-members to allowed-collaborators in safe-outputs.mentions")
			}
			return newContent, applied, err
		},
	}
}

func mentionsAllowTeamMembersNeedsMigration(frontmatter map[string]any) bool {
	safeOutputsAny, ok := frontmatter["safe-outputs"]
	if !ok {
		return false
	}
	safeOutputsMap, ok := safeOutputsAny.(map[string]any)
	if !ok {
		return false
	}
	mentionsAny, ok := safeOutputsMap["mentions"]
	if !ok {
		return false
	}
	mentionsMap, ok := mentionsAny.(map[string]any)
	if !ok {
		return false
	}

	_, hasOld := mentionsMap["allow-team-members"]
	_, hasNew := mentionsMap["allowed-collaborators"]
	return hasOld && !hasNew
}

func renameMentionsAllowTeamMembers(lines []string) ([]string, bool) {
	result := make([]string, 0, len(lines))
	modified := false

	inSafeOutputs := false
	safeOutputsIndent := ""
	safeOutputsChildIndent := ""
	inMentions := false
	mentionsIndent := ""
	mentionsChildIndent := ""

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		indent := getIndentation(line)

		if !strings.HasPrefix(trimmed, "#") {
			if inSafeOutputs && hasExitedBlock(line, safeOutputsIndent) {
				inSafeOutputs = false
				safeOutputsChildIndent = ""
				inMentions = false
				mentionsIndent = ""
				mentionsChildIndent = ""
			}
			if inMentions && hasExitedBlock(line, mentionsIndent) {
				inMentions = false
				mentionsIndent = ""
				mentionsChildIndent = ""
			}
		}

		if strings.HasPrefix(trimmed, "safe-outputs:") {
			inSafeOutputs = true
			safeOutputsIndent = indent
			safeOutputsChildIndent = ""
			inMentions = false
			mentionsIndent = ""
			mentionsChildIndent = ""
			result = append(result, line)
			continue
		}

		if inSafeOutputs && isDescendant(indent, safeOutputsIndent) && !strings.HasPrefix(trimmed, "#") {
			if (safeOutputsChildIndent == "" || indent == safeOutputsChildIndent) && strings.HasPrefix(trimmed, "mentions:") {
				if safeOutputsChildIndent == "" {
					safeOutputsChildIndent = indent
				}
				if strings.HasSuffix(trimmed, ":") {
					inMentions = true
					mentionsIndent = indent
					mentionsChildIndent = ""
				} else {
					inMentions = false
					mentionsIndent = ""
					mentionsChildIndent = ""
					if strings.Contains(trimmed, "allow-team-members:") {
						newLine := strings.Replace(line, "allow-team-members:", "allowed-collaborators:", 1)
						result = append(result, newLine)
						modified = true
						mentionsAllowTeamMembersCodemodLog.Printf("Renamed allow-team-members to allowed-collaborators in safe-outputs.mentions on line %d", i+1)
						continue
					}
				}
				result = append(result, line)
				continue
			}
			if strings.HasSuffix(trimmed, ":") && (safeOutputsChildIndent == "" || indent == safeOutputsChildIndent) {
				if safeOutputsChildIndent == "" {
					safeOutputsChildIndent = indent
				}
				inMentions = false
				mentionsIndent = ""
				mentionsChildIndent = ""
				result = append(result, line)
				continue
			}
		}

		if inMentions && mentionsChildIndent == "" && isDescendant(indent, mentionsIndent) && trimmed != "" && !strings.HasPrefix(trimmed, "#") {
			mentionsChildIndent = indent
		}

		if inMentions && indent == mentionsChildIndent && strings.HasPrefix(trimmed, "allow-team-members:") {
			newLine, replaced := findAndReplaceInLine(line, "allow-team-members", "allowed-collaborators")
			if replaced {
				result = append(result, newLine)
				modified = true
				mentionsAllowTeamMembersCodemodLog.Printf("Renamed allow-team-members to allowed-collaborators in safe-outputs.mentions on line %d", i+1)
				continue
			}
		}

		result = append(result, line)
	}

	return result, modified
}
