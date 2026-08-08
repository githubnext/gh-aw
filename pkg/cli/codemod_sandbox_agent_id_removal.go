package cli

import "github.com/github/gh-aw/pkg/logger"

var sandboxAgentIDRemovalCodemodLog = logger.New("cli:codemod_sandbox_agent_id_removal")

// getSandboxAgentIDRemovalCodemod creates a codemod that removes the redundant
// sandbox.agent.id: awf field. Since awf is the only supported engine and the
// default when id is omitted, explicitly writing id: awf adds no information.
func getSandboxAgentIDRemovalCodemod() Codemod {
	return Codemod{
		ID:           "sandbox-agent-id-awf-removal",
		Name:         "Remove redundant sandbox.agent.id: awf field",
		Description:  "Removes 'id: awf' from sandbox.agent blocks. awf is the only supported engine and the default when id is omitted, so explicitly setting it is redundant.",
		IntroducedIn: "0.28.0",
		Apply: func(content string, frontmatter map[string]any) (string, bool, error) {
			if !isSandboxAgentIDAwf(frontmatter) {
				return content, false, nil
			}
			newContent, applied, err := applyFrontmatterLineTransform(content, func(lines []string) ([]string, bool) {
				return removeFieldFromBlock(lines, "id", "agent")
			})
			if applied {
				sandboxAgentIDRemovalCodemodLog.Print("Removed redundant sandbox.agent.id: awf")
			}
			return newContent, applied, err
		},
	}
}

// isSandboxAgentIDAwf returns true when frontmatter["sandbox"]["agent"]["id"] is "awf".
func isSandboxAgentIDAwf(frontmatter map[string]any) bool {
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
	agentMap, ok := agentVal.(map[string]any)
	if !ok {
		return false
	}
	idVal, ok := agentMap["id"]
	if !ok {
		return false
	}
	idStr, ok := idVal.(string)
	return ok && idStr == "awf"
}
