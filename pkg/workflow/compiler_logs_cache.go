package workflow

import "strings"

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

func sharedLogsCacheRestoreSteps(data *WorkflowData) []GitHubActionStep {
	if !usesSharedLogsCache(data) {
		return nil
	}

	return []GitHubActionStep{
		{
			"      - name: Restore shared agentic logs cache",
			"        id: restore-agentic-logs-cache",
			"        continue-on-error: true",
			"        uses: " + getCachedActionPin("actions/cache/restore", data),
			"        with:",
			"          key: agentic-logs-${{ github.run_id }}",
			"          restore-keys: |",
			"            agentic-logs-",
			"          path: " + sharedLogsCachePath,
		},
	}
}

func generateSharedLogsCacheRestoreSteps(yaml *strings.Builder, data *WorkflowData) {
	for _, step := range sharedLogsCacheRestoreSteps(data) {
		for _, line := range step {
			yaml.WriteString(line)
			yaml.WriteByte('\n')
		}
	}
}
