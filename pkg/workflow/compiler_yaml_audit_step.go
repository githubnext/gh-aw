package workflow

import (
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

var auditStepLog = logger.New("workflow:compiler_yaml_audit_step")

// generatePreAgentAuditStep emits a step that lists files in agent-related directories
// for all known agentic engines (Copilot, Claude, Codex, Gemini, Crush, OpenCode, Pi)
// under the workspace and the agent user's home folder.
// The listing is saved to /tmp/gh-aw/pre-agent-audit.txt (via the shell script) and set
// as a GITHUB_OUTPUT value so it is accessible in subsequent steps and included in the
// agent artifact.
//
// Common cache directories (node_modules, __pycache__, .cache, vendor, .npm, .yarn,
// .pnpm-store, site-packages, .bundle) are excluded to keep the listing concise.
//
// The step delegates all logic to audit_pre_agent_workspace.sh and runs with
// continue-on-error so a missing directory or permission error does not block agent
// execution.
func (c *Compiler) generatePreAgentAuditStep(yaml *strings.Builder) {
	auditStepLog.Print("Generating pre-agent workspace audit step")
	yaml.WriteString("      - name: Audit pre-agent workspace\n")
	yaml.WriteString("        id: pre_agent_audit\n")
	yaml.WriteString("        continue-on-error: true\n")
	yaml.WriteString("        run: bash \"${RUNNER_TEMP}/gh-aw/actions/audit_pre_agent_workspace.sh\"\n")
}
