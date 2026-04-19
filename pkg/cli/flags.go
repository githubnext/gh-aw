package cli

import "github.com/spf13/cobra"

const engineFlagHelpList = "copilot, claude, codex, gemini, crush"

func engineFlagUsage(prefix string) string {
	return prefix + " (" + engineFlagHelpList + ")"
}

// addEngineFlag adds the --engine/-e flag to a command.
// This flag allows overriding the AI engine type.
func addEngineFlag(cmd *cobra.Command) {
	cmd.Flags().StringP("engine", "e", "", engineFlagUsage("Override AI engine"))
}

// addEngineFilterFlag adds the --engine/-e flag to a command for filtering.
// This flag allows filtering results by AI engine type.
func addEngineFilterFlag(cmd *cobra.Command) {
	cmd.Flags().StringP("engine", "e", "", engineFlagUsage("Filter logs by AI engine"))
}

// addRepoFlag adds the --repo/-r flag to a command.
// This flag allows specifying a target repository.
func addRepoFlag(cmd *cobra.Command) {
	cmd.Flags().StringP("repo", "r", "", "Target repository ([HOST/]owner/repo format). Defaults to current repository")
}

// addOutputFlag adds the --output/-o flag to a command.
// This flag allows specifying an output directory for generated files.
func addOutputFlag(cmd *cobra.Command, defaultValue string) {
	cmd.Flags().StringP("output", "o", defaultValue, "Output directory for generated files")
}

// addJSONFlag adds the --json/-j flag to a command.
// This flag enables JSON output format.
func addJSONFlag(cmd *cobra.Command) {
	cmd.Flags().BoolP("json", "j", false, "Output results in JSON format")
}

// addMinGitHubAPILimitsFlag adds the --min-github-api-limits flag to a command.
// This flag guards execution by requiring at least N remaining GitHub API core requests.
func addMinGitHubAPILimitsFlag(cmd *cobra.Command) {
	cmd.Flags().Int("min-github-api-limits", 0, "Cancel execution when remaining GitHub API core requests are below this value (0 disables guard)")
}
