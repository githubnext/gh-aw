package workflow

import (
	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/sliceutil"
)

// copilotBotSet is a fast-lookup set built from constants.CopilotBotNames.
// Any entry in this set triggers expansion to the full CopilotBotNames list.
var copilotBotSet = func() map[string]bool {
	set := make(map[string]bool, len(constants.CopilotBotNames))
	for _, name := range constants.CopilotBotNames {
		set[name] = true
	}
	return set
}()

// expandBotNames expands any entry found in constants.CopilotBotNames to the
// full set of Copilot identifiers. Other entries are passed through unchanged.
// Duplicates are removed from the result.
//
// A nil or empty input slice is returned as-is. The nil/empty distinction is
// preserved so callers can distinguish "no bots configured" (nil) from "bots
// field present but empty" ([]string{}).
//
// The recognized identifiers are defined in constants.CopilotBotNames.
func expandBotNames(bots []string) []string {
	if len(bots) == 0 {
		return bots
	}
	needsExpansion := false
	for _, b := range bots {
		if copilotBotSet[b] {
			needsExpansion = true
			break
		}
	}
	if !needsExpansion {
		return bots
	}
	// Pre-allocate with the worst-case capacity: every entry is a copilot
	// identifier that expands to len(constants.CopilotBotNames) entries.
	expanded := make([]string, 0, len(bots)*len(constants.CopilotBotNames))
	for _, b := range bots {
		if copilotBotSet[b] {
			expanded = append(expanded, constants.CopilotBotNames...)
		} else {
			expanded = append(expanded, b)
		}
	}
	return sliceutil.Deduplicate(expanded)
}
