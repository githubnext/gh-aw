package workflow

import (
	"strings"

	"github.com/github/gh-aw/pkg/constants"
)

// generatePreAgentAuditStep emits a step that lists files in agent-related directories
// (skills, agents, copilot config) under the workspace and the agent user's home folder.
// The listing is saved to PreAgentAuditFilePath and set as a GITHUB_OUTPUT value so it
// is accessible in subsequent steps and included in the agent artifact.
//
// The step runs with continue-on-error so a missing directory or permission error does
// not block agent execution. Common cache directories (node_modules, __pycache__, .cache,
// vendor, .npm, .yarn, site-packages, .bundle) are excluded to keep the listing concise.
func (c *Compiler) generatePreAgentAuditStep(yaml *strings.Builder) {
	yaml.WriteString("      - name: Audit pre-agent workspace\n")
	yaml.WriteString("        id: pre-agent-audit\n")
	yaml.WriteString("        continue-on-error: true\n")
	yaml.WriteString("        run: |\n")
	yaml.WriteString("          AUDIT_FILE=\"" + constants.PreAgentAuditFilePath + "\"\n")
	yaml.WriteString("          mkdir -p \"$(dirname \"${AUDIT_FILE}\")\"\n")
	yaml.WriteString("          PRUNE_OPTS=(\n")
	yaml.WriteString("            -not -path \"*/node_modules/*\"\n")
	yaml.WriteString("            -not -path \"*/__pycache__/*\"\n")
	yaml.WriteString("            -not -path \"*/.cache/*\"\n")
	yaml.WriteString("            -not -path \"*/vendor/*\"\n")
	yaml.WriteString("            -not -path \"*/.npm/*\"\n")
	yaml.WriteString("            -not -path \"*/.yarn/*\"\n")
	yaml.WriteString("            -not -path \"*/.pnpm-store/*\"\n")
	yaml.WriteString("            -not -path \"*/site-packages/*\"\n")
	yaml.WriteString("            -not -path \"*/.bundle/*\"\n")
	yaml.WriteString("          )\n")
	yaml.WriteString("          {\n")
	yaml.WriteString("            echo \"=== Pre-agent workspace audit ===\"\n")
	yaml.WriteString("            echo \"--- Workspace agents: ${GITHUB_WORKSPACE}/.github/agents/ ---\"\n")
	yaml.WriteString("            find \"${GITHUB_WORKSPACE}/.github/agents\" \"${PRUNE_OPTS[@]}\" -print 2>/dev/null || echo \"(not found)\"\n")
	yaml.WriteString("            echo \"--- Workspace skills: ${GITHUB_WORKSPACE}/.github/skills/ ---\"\n")
	yaml.WriteString("            find \"${GITHUB_WORKSPACE}/.github/skills\" \"${PRUNE_OPTS[@]}\" -print 2>/dev/null || echo \"(not found)\"\n")
	yaml.WriteString("            echo \"--- Workspace copilot config: ${GITHUB_WORKSPACE}/.github/copilot/ ---\"\n")
	yaml.WriteString("            find \"${GITHUB_WORKSPACE}/.github/copilot\" \"${PRUNE_OPTS[@]}\" -print 2>/dev/null || echo \"(not found)\"\n")
	yaml.WriteString("            echo \"--- Agent user home agents: ${HOME}/.github/ ---\"\n")
	yaml.WriteString("            find \"${HOME}/.github\" \"${PRUNE_OPTS[@]}\" -print 2>/dev/null || echo \"(not found)\"\n")
	yaml.WriteString("            echo \"--- gh extensions: ${HOME}/.local/share/gh/extensions/ ---\"\n")
	yaml.WriteString("            find \"${HOME}/.local/share/gh/extensions\" \"${PRUNE_OPTS[@]}\" -print 2>/dev/null || echo \"(not found)\"\n")
	yaml.WriteString("            echo \"--- gh-aw temp directory: ${RUNNER_TEMP}/gh-aw/ ---\"\n")
	yaml.WriteString("            find \"${RUNNER_TEMP}/gh-aw\" \"${PRUNE_OPTS[@]}\" -print 2>/dev/null || echo \"(not found)\"\n")
	yaml.WriteString("          } > \"${AUDIT_FILE}\"\n")
	yaml.WriteString("          LINE_COUNT=\"$(wc -l < \"${AUDIT_FILE}\" | tr -d ' ')\"\n")
	yaml.WriteString("          echo \"pre-agent-audit-file=${AUDIT_FILE}\" >> \"${GITHUB_OUTPUT}\"\n")
	yaml.WriteString("          echo \"pre-agent-audit-line-count=${LINE_COUNT}\" >> \"${GITHUB_OUTPUT}\"\n")
	yaml.WriteString("          echo \"Pre-agent audit written to ${AUDIT_FILE} (${LINE_COUNT} lines)\"\n")
}
