package workflow

import (
	"fmt"
	"strings"
)

const sharedLogsCachePath = ".github/aw/logs"

func usesSharedLogsCache(data *WorkflowData) bool {
	if !strings.Contains(data.On, "schedule") {
		return false
	}
	content := data.CustomSteps + "\n" + data.MarkdownContent
	for _, command := range []string{"gh aw logs", "./gh-aw logs", "gh aw audit", "./gh-aw audit"} {
		if strings.Contains(content, command) {
			return true
		}
	}
	return false
}

func generateSharedLogsCacheRestoreSteps(yaml *strings.Builder, data *WorkflowData) {
	if !usesSharedLogsCache(data) {
		return
	}

	yaml.WriteString("      - name: Restore shared agentic logs cache\n")
	yaml.WriteString("        id: restore-agentic-logs-cache\n")
	yaml.WriteString("        continue-on-error: true\n")
	fmt.Fprintf(yaml, "        uses: %s\n", getCachedActionPin("actions/cache/restore", data))
	yaml.WriteString("        with:\n")
	yaml.WriteString("          key: agentic-logs-${{ github.run_id }}\n")
	yaml.WriteString("          restore-keys: |\n")
	yaml.WriteString("            agentic-logs-\n")
	fmt.Fprintf(yaml, "          path: %s\n", sharedLogsCachePath)
	yaml.WriteString("      - name: Enforce shared agentic logs cache TTL\n")
	yaml.WriteString("        shell: bash\n")
	yaml.WriteString("        run: |\n")
	fmt.Fprintf(yaml, "          marker=%s/.cache-refreshed-at\n", sharedLogsCachePath)
	yaml.WriteString("          if [ -f \"$marker\" ]; then\n")
	yaml.WriteString("            refreshed_at=$(cat \"$marker\")\n")
	yaml.WriteString("            if ! refreshed_epoch=$(date -d \"$refreshed_at\" +%s 2>/dev/null) \\\n")
	yaml.WriteString("              || [ \"$refreshed_epoch\" -lt \"$(date -u -d '2 days ago' +%s)\" ]; then\n")
	fmt.Fprintf(yaml, "              rm -rf %s\n", sharedLogsCachePath)
	yaml.WriteString("            fi\n")
	yaml.WriteString("          fi\n")
}
