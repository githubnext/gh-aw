package workflow

import "github.com/github/gh-aw/pkg/sliceutil"

// copilotBotNames is the list of bot identifiers that the "copilot" alias expands to.
// It covers the known GitHub Copilot bot identities:
//   - "copilot-swe-agent" — the Copilot Coding Agent (actor: copilot-swe-agent[bot])
//   - "Copilot"           — the @Copilot interactive bot (actor: Copilot)
//   - "copilot"           — the base copilot form (actor: copilot[bot])
var copilotBotNames = []string{
	"copilot-swe-agent",
	"Copilot",
	"copilot",
}

// copilotAliases is the set of shorthand aliases that all expand to copilotBotNames.
//   - "copilot"               — the canonical shorthand alias
//   - "@app/copilot-swe-agent" — the GitHub App slug alias for the Copilot Coding Agent
var copilotAliases = map[string]bool{
	"copilot":               true,
	"@app/copilot-swe-agent": true,
}

// expandBotNames expands known shorthand aliases in a list of bot names to the
// full set of GitHub Copilot bot identifiers. Other entries are passed through
// unchanged. Duplicates are removed from the result.
//
// A nil or empty input slice is returned as-is. The nil/empty distinction is
// preserved so callers can distinguish "no bots configured" (nil) from "bots
// field present but empty" ([]string{}).
//
// The "copilot" and "@app/copilot-swe-agent" aliases cover:
//   - copilot-swe-agent / copilot-swe-agent[bot] — Copilot Coding Agent
//   - Copilot                                     — @Copilot interactive bot
//   - copilot / copilot[bot]                      — base copilot bot form
func expandBotNames(bots []string) []string {
	if len(bots) == 0 {
		return bots
	}
	needsExpansion := false
	for _, b := range bots {
		if copilotAliases[b] {
			needsExpansion = true
			break
		}
	}
	if !needsExpansion {
		return bots
	}
	// Pre-allocate with the worst-case capacity: every entry is a copilot
	// alias that expands to len(copilotBotNames) entries.
	expanded := make([]string, 0, len(bots)*len(copilotBotNames))
	for _, b := range bots {
		if copilotAliases[b] {
			expanded = append(expanded, copilotBotNames...)
		} else {
			expanded = append(expanded, b)
		}
	}
	return sliceutil.Deduplicate(expanded)
}
