package workflow

import (
	_ "embed"
	"fmt"
	"strings"
)

//go:embed sh/pr_context_prompt.md
var prContextPromptText string

//go:embed sh/print_prompt_summary.sh
var printPromptSummaryScript string

//go:embed sh/create_prompt_first.sh
var createPromptFirstScript string

//go:embed sh/generate_git_patch.sh
var generateGitPatchScript string

//go:embed sh/capture_agent_version.sh
var captureAgentVersionScript string

//go:embed sh/extract_squid_logs_setup.sh
var extractSquidLogsSetupScript string

//go:embed sh/extract_squid_log_per_tool.sh
var extractSquidLogPerToolScript string

//go:embed sh/create_cache_memory_dir.sh
var createCacheMemoryDirScript string

//go:embed sh/create_gh_aw_tmp_dir.sh
var createGhAwTmpDirScript string

//go:embed sh/xpia_prompt.md
var xpiaPromptText string

// WriteShellScriptToYAML writes a shell script with proper indentation to a strings.Builder
func WriteShellScriptToYAML(yaml *strings.Builder, script string, indent string) {
	scriptLines := strings.Split(script, "\n")
	for _, line := range scriptLines {
		// Skip empty lines at the beginning or end
		if strings.TrimSpace(line) != "" {
			fmt.Fprintf(yaml, "%s%s\n", indent, line)
		}
	}
}

// WritePromptTextToYAML writes prompt text to a YAML heredoc with proper indentation
func WritePromptTextToYAML(yaml *strings.Builder, text string, indent string) {
	yaml.WriteString(indent + "cat >> $GITHUB_AW_PROMPT << 'EOF'\n")
	textLines := strings.Split(text, "\n")
	for _, line := range textLines {
		fmt.Fprintf(yaml, "%s%s\n", indent, line)
	}
	yaml.WriteString(indent + "EOF\n")
}
