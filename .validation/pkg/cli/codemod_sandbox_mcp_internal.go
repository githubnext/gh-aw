package cli

import "github.com/github/gh-aw/pkg/logger"

var sandboxMCPInternalCodemodLog = logger.New("cli:codemod_sandbox_mcp_internal")

// getSandboxMCPContainerRemovalCodemod creates a codemod that removes the deprecated
// sandbox.mcp.container field. The MCP gateway container is now managed internally by
// gh-aw based on the declared MCP toolsets.
func getSandboxMCPContainerRemovalCodemod() Codemod {
	return Codemod{
		ID:           "sandbox-mcp-container-removal",
		Name:         "Remove deprecated sandbox.mcp.container field",
		Description:  "Removes 'sandbox.mcp.container' as the MCP gateway container is now managed internally. Remove this key or set 'strict: false' to disable strict mode.",
		IntroducedIn: "0.26.0",
		Apply: func(content string, frontmatter map[string]any) (string, bool, error) {
			if isFrontmatterStrictFalse(frontmatter) {
				return content, false, nil
			}
			if !hasSandboxMCPField(frontmatter, "container") {
				return content, false, nil
			}
			newContent, applied, err := applyFrontmatterLineTransform(content, func(lines []string) ([]string, bool) {
				result, modified := removeFieldFromBlock(lines, "container", "mcp")
				if modified {
					// If mcp: became empty and was removed, also clean up sandbox: if it
					// is now empty to avoid leaving a dangling "sandbox: null" key.
					result = removeParentBlockIfTrulyEmpty(result, "sandbox")
				}
				return result, modified
			})
			if applied {
				sandboxMCPInternalCodemodLog.Print("Removed deprecated sandbox.mcp.container")
			}
			return newContent, applied, err
		},
	}
}

// getSandboxMCPVersionRemovalCodemod creates a codemod that removes the deprecated
// sandbox.mcp.version field. The MCP gateway version is now managed internally by
// gh-aw based on the declared MCP toolsets.
func getSandboxMCPVersionRemovalCodemod() Codemod {
	return Codemod{
		ID:           "sandbox-mcp-version-removal",
		Name:         "Remove deprecated sandbox.mcp.version field",
		Description:  "Removes 'sandbox.mcp.version' as the MCP gateway version is now managed internally. Remove this key or set 'strict: false' to disable strict mode.",
		IntroducedIn: "0.26.0",
		Apply: func(content string, frontmatter map[string]any) (string, bool, error) {
			if isFrontmatterStrictFalse(frontmatter) {
				return content, false, nil
			}
			if !hasSandboxMCPField(frontmatter, "version") {
				return content, false, nil
			}
			newContent, applied, err := applyFrontmatterLineTransform(content, func(lines []string) ([]string, bool) {
				result, modified := removeFieldFromBlock(lines, "version", "mcp")
				if modified {
					// If mcp: became empty and was removed, also clean up sandbox: if it
					// is now empty to avoid leaving a dangling "sandbox: null" key.
					result = removeParentBlockIfTrulyEmpty(result, "sandbox")
				}
				return result, modified
			})
			if applied {
				sandboxMCPInternalCodemodLog.Print("Removed deprecated sandbox.mcp.version")
			}
			return newContent, applied, err
		},
	}
}

// hasSandboxMCPField checks whether frontmatter["sandbox"]["mcp"][fieldName] exists.
func hasSandboxMCPField(frontmatter map[string]any, fieldName string) bool {
	sandboxVal, ok := frontmatter["sandbox"]
	if !ok {
		return false
	}
	sandboxMap, ok := sandboxVal.(map[string]any)
	if !ok {
		return false
	}
	mcpVal, ok := sandboxMap["mcp"]
	if !ok {
		return false
	}
	mcpMap, ok := mcpVal.(map[string]any)
	if !ok {
		return false
	}
	_, hasField := mcpMap[fieldName]
	return hasField
}
