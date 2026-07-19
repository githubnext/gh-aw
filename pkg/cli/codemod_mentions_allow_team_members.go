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

	state := renameMentionsAllowTeamMembersState{}

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		indent := getIndentation(line)

		renameMentionsAllowTeamMembersExitBlocks(line, trimmed, &state)

		if strings.HasPrefix(trimmed, "safe-outputs:") {
			state.inSafeOutputs = true
			state.safeOutputsIndent = indent
			state.safeOutputsChildIndent = ""
			renameMentionsAllowTeamMembersResetMentions(&state)
			result = append(result, line)
			continue
		}

		if newLine, changed, handled := renameMentionsAllowTeamMembersSafeOutputsLine(line, trimmed, indent, i, &state); handled {
			if changed {
				modified = true
			}
			result = append(result, newLine)
			continue
		}

		if state.inMentions && state.mentionsChildIndent == "" && isDescendant(indent, state.mentionsIndent) && trimmed != "" && !strings.HasPrefix(trimmed, "#") {
			state.mentionsChildIndent = indent
		}

		if newLine, replaced := renameMentionsAllowTeamMembersLine(line, trimmed, indent, i, &state); replaced {
			result = append(result, newLine)
			modified = true
			continue
		}

		result = append(result, line)
	}

	return result, modified
}

type renameMentionsAllowTeamMembersState struct {
	inSafeOutputs          bool
	safeOutputsIndent      string
	safeOutputsChildIndent string
	inMentions             bool
	mentionsIndent         string
	mentionsChildIndent    string
}

func renameMentionsAllowTeamMembersResetMentions(state *renameMentionsAllowTeamMembersState) {
	state.inMentions = false
	state.mentionsIndent = ""
	state.mentionsChildIndent = ""
}

func renameMentionsAllowTeamMembersExitBlocks(line string, trimmed string, state *renameMentionsAllowTeamMembersState) {
	if strings.HasPrefix(trimmed, "#") {
		return
	}
	if state.inSafeOutputs && hasExitedBlock(line, state.safeOutputsIndent) {
		state.inSafeOutputs = false
		state.safeOutputsChildIndent = ""
		renameMentionsAllowTeamMembersResetMentions(state)
	}
	if state.inMentions && hasExitedBlock(line, state.mentionsIndent) {
		renameMentionsAllowTeamMembersResetMentions(state)
	}
}

func renameMentionsAllowTeamMembersSafeOutputsLine(
	line string,
	trimmed string,
	indent string,
	index int,
	state *renameMentionsAllowTeamMembersState,
) (string, bool, bool) {
	if !state.inSafeOutputs || !isDescendant(indent, state.safeOutputsIndent) || strings.HasPrefix(trimmed, "#") {
		return line, false, false
	}
	if (state.safeOutputsChildIndent == "" || indent == state.safeOutputsChildIndent) && strings.HasPrefix(trimmed, "mentions:") {
		return renameMentionsAllowTeamMembersMentionsLine(line, trimmed, indent, index, state)
	}
	if strings.HasSuffix(trimmed, ":") && (state.safeOutputsChildIndent == "" || indent == state.safeOutputsChildIndent) {
		if state.safeOutputsChildIndent == "" {
			state.safeOutputsChildIndent = indent
		}
		renameMentionsAllowTeamMembersResetMentions(state)
		return line, false, true
	}
	return line, false, false
}

func renameMentionsAllowTeamMembersMentionsLine(
	line string,
	trimmed string,
	indent string,
	index int,
	state *renameMentionsAllowTeamMembersState,
) (string, bool, bool) {
	if state.safeOutputsChildIndent == "" {
		state.safeOutputsChildIndent = indent
	}
	if strings.HasSuffix(trimmed, ":") {
		state.inMentions = true
		state.mentionsIndent = indent
		state.mentionsChildIndent = ""
		return line, false, true
	}
	renameMentionsAllowTeamMembersResetMentions(state)
	if strings.Contains(trimmed, "allow-team-members:") {
		newLine := strings.Replace(line, "allow-team-members:", "allowed-collaborators:", 1)
		mentionsAllowTeamMembersCodemodLog.Printf("Renamed allow-team-members to allowed-collaborators in safe-outputs.mentions on line %d", index+1)
		return newLine, true, true
	}
	return line, false, true
}

func renameMentionsAllowTeamMembersLine(
	line string,
	trimmed string,
	indent string,
	index int,
	state *renameMentionsAllowTeamMembersState,
) (string, bool) {
	if !state.inMentions || indent != state.mentionsChildIndent || !strings.HasPrefix(trimmed, "allow-team-members:") {
		return line, false
	}
	newLine, replaced := findAndReplaceInLine(line, "allow-team-members", "allowed-collaborators")
	if replaced {
		mentionsAllowTeamMembersCodemodLog.Printf("Renamed allow-team-members to allowed-collaborators in safe-outputs.mentions on line %d", index+1)
	}
	return newLine, replaced
}
