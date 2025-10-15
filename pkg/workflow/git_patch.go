package workflow

import "strings"

// generateGitPatchStep uploads git patches created by the safe outputs MCP server
// Note: Patch generation now happens in the MCP server when create-pull-request or
// push-to-pull-request-branch tools are called, so this step only uploads the patches.
func (c *Compiler) generateGitPatchStep(yaml *strings.Builder) {
	yaml.WriteString("      - name: Upload git patches\n")
	yaml.WriteString("        if: always()\n")
	yaml.WriteString("        uses: actions/upload-artifact@v4\n")
	yaml.WriteString("        with:\n")
	yaml.WriteString("          name: patches\n")
	yaml.WriteString("          path: /tmp/gh-aw/patches/\n")
	yaml.WriteString("          if-no-files-found: ignore\n")
}
