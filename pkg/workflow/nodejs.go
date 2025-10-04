package workflow

import "strings"

// addNodeJsSetupIfNeeded adds Node.js setup step if it's not already present in custom steps
// and if the engine requires it (npm-based engines like claude, codex, copilot)
func addNodeJsSetupIfNeeded(yaml *strings.Builder, data *WorkflowData) {
	// Check if Node.js is already set up in custom steps
	nodeJsAlreadySetup := false
	if data.CustomSteps != "" {
		if strings.Contains(data.CustomSteps, "actions/setup-node") || strings.Contains(data.CustomSteps, "Setup Node.js") {
			nodeJsAlreadySetup = true
		}
	}

	// If Node.js is not already set up and the engine is npm-based (claude, codex, copilot), add it
	if !nodeJsAlreadySetup && (data.AI == "claude" || data.AI == "codex" || data.AI == "copilot" || data.AI == "") {
		yaml.WriteString("      - name: Setup Node.js\n")
		yaml.WriteString("        uses: actions/setup-node@v4\n")
		yaml.WriteString("        with:\n")
		yaml.WriteString("          node-version: '24'\n")
	}
}
