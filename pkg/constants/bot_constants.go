package constants

// CopilotBotNames is the canonical list of GitHub bot login identifiers that
// represent the Copilot family. Any "copilot" shorthand alias in a workflow
// expands to these three identities:
//
//   - "copilot-swe-agent" — the Copilot Coding Agent (actor: copilot-swe-agent[bot])
//   - "Copilot"           — the @Copilot interactive bot (actor: Copilot)
//   - "copilot"           — the base copilot bot form (actor: copilot[bot])
var CopilotBotNames = []string{
	"copilot-swe-agent",
	"Copilot",
	"copilot",
}

// CopilotBotAliases is the set of shorthand strings that all expand to
// CopilotBotNames. Keeping aliases here ensures a single authoritative list
// that any package can import without depending on pkg/workflow.
//
//   - "copilot"                — the canonical shorthand alias
//   - "@app/copilot-swe-agent" — the GitHub App slug alias for the Copilot Coding Agent
var CopilotBotAliases = map[string]bool{
	"copilot":               true,
	"@app/copilot-swe-agent": true,
}
