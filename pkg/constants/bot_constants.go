package constants

// CopilotBotNames is the canonical list of all GitHub identifiers associated
// with the Copilot family — both runtime bot logins and recognized input aliases.
// When any entry from this list appears in a workflow's bots field, it is
// expanded to the entire set:
//
//   - "copilot-swe-agent"    — Copilot Coding Agent runtime login (actor: copilot-swe-agent[bot])
//   - "Copilot"              — @Copilot interactive bot (actor: Copilot)
//   - "copilot"              — base copilot bot form + canonical shorthand alias (actor: copilot[bot])
//   - "@app/copilot-swe-agent" — GitHub App slug alias for the Copilot Coding Agent
var CopilotBotNames = []string{
	"copilot-swe-agent",
	"Copilot",
	"copilot",
	"@app/copilot-swe-agent",
}
