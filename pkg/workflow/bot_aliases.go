package workflow

import (
	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/sliceutil"
)

// expandBotNames expands known shorthand aliases in a list of bot names to the
// full set of GitHub Copilot bot identifiers. Other entries are passed through
// unchanged. Duplicates are removed from the result.
//
// A nil or empty input slice is returned as-is. The nil/empty distinction is
// preserved so callers can distinguish "no bots configured" (nil) from "bots
// field present but empty" ([]string{}).
//
// The recognized aliases and expanded bot names are defined in
// constants.CopilotBotAliases and constants.CopilotBotNames respectively.
func expandBotNames(bots []string) []string {
	if len(bots) == 0 {
		return bots
	}
	needsExpansion := false
	for _, b := range bots {
		if constants.CopilotBotAliases[b] {
			needsExpansion = true
			break
		}
	}
	if !needsExpansion {
		return bots
	}
	// Pre-allocate with the worst-case capacity: every entry is a copilot
	// alias that expands to len(constants.CopilotBotNames) entries.
	expanded := make([]string, 0, len(bots)*len(constants.CopilotBotNames))
	for _, b := range bots {
		if constants.CopilotBotAliases[b] {
			expanded = append(expanded, constants.CopilotBotNames...)
		} else {
			expanded = append(expanded, b)
		}
	}
	return sliceutil.Deduplicate(expanded)
}
