package workflow

import (
	"fmt"
	"strings"
)

// generateGitPatchStep generates a step that creates and uploads a git patch of changes
func (c *Compiler) generateGitPatchStep(yaml *strings.Builder) {
	yaml.WriteString("      - name: Generate git patch\n")
	yaml.WriteString("        if: always()\n")
	yaml.WriteString(fmt.Sprintf("        uses: %s\n", GetActionPin("actions/github-script")))
	yaml.WriteString("        env:\n")
	yaml.WriteString("          GH_AW_SAFE_OUTPUTS: ${{ env.GH_AW_SAFE_OUTPUTS }}\n")
	yaml.WriteString("          GH_AW_DEFAULT_BRANCH: ${{ github.event.repository.default_branch }}\n")
	yaml.WriteString("        with:\n")
	yaml.WriteString("          script: |\n")
	WriteJavaScriptToYAML(yaml, generateGitPatchScript)
	yaml.WriteString("      - name: Upload git patch\n")
	yaml.WriteString("        if: always()\n")
	yaml.WriteString(fmt.Sprintf("        uses: %s\n", GetActionPin("actions/upload-artifact")))
	yaml.WriteString("        with:\n")
	yaml.WriteString("          name: aw.patch\n")
	yaml.WriteString("          path: /tmp/gh-aw/aw.patch\n")
	yaml.WriteString("          if-no-files-found: ignore\n")
}
