package cli

import "github.com/github/gh-aw/pkg/logger"

var sandboxAgentFalseRemovalCodemodLog = logger.New("cli:codemod_sandbox_agent_false_removal")

// getSandboxAgentFalseRemovalCodemod creates a codemod that removes the deprecated
// sandbox.agent: false key. Setting sandbox.agent to false was previously supported as
// an escape hatch for testing but is now rejected in strict mode because it disables
// important agent sandbox security protections. Remove the key to restore default
// (sandboxed) behavior, or set 'strict: false' to opt out of strict mode.
func getSandboxAgentFalseRemovalCodemod() Codemod {
	return Codemod{
		ID:           "sandbox-agent-false-removal",
		Name:         "Remove deprecated sandbox.agent: false field",
		Description:  "Removes 'sandbox.agent: false' which is no longer allowed in strict mode. The agent sandbox firewall is now always enabled by default. Remove this key to restore sandboxed behavior, or set 'strict: false' to opt out of strict mode.",
		IntroducedIn: "0.26.0",
		Apply: func(content string, frontmatter map[string]any) (string, bool, error) {
			if isFrontmatterStrictFalse(frontmatter) {
				return content, false, nil
			}
			if !isSandboxAgentFalse(frontmatter) {
				return content, false, nil
			}
			newContent, applied, err := applyFrontmatterLineTransform(content, func(lines []string) ([]string, bool) {
				return removeFieldFromBlock(lines, "agent", "sandbox")
			})
			if applied {
				sandboxAgentFalseRemovalCodemodLog.Print("Removed deprecated sandbox.agent: false")
			}
			return newContent, applied, err
		},
	}
}

// isSandboxAgentFalse returns true when frontmatter["sandbox"]["agent"] is boolean false.
func isSandboxAgentFalse(frontmatter map[string]any) bool {
	sandboxVal, ok := frontmatter["sandbox"]
	if !ok {
		return false
	}
	sandboxMap, ok := sandboxVal.(map[string]any)
	if !ok {
		return false
	}
	agentVal, ok := sandboxMap["agent"]
	if !ok {
		return false
	}
	agentBool, ok := agentVal.(bool)
	return ok && !agentBool
}
