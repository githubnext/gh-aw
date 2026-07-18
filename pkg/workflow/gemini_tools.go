package workflow

// This file provides Gemini engine tool configuration logic.
//
// computeGeminiToolsCore is a thin wrapper around computeGoogleCLIToolsCore that
// preserves the existing function signature for tests and callers while delegating
// all logic to the shared implementation.
//
// The settings-step generator (generateSettingsStep) lives on googleCLIEngine and
// calls computeGoogleCLIToolsCore directly with the engine's configured logger.

import (
	"github.com/github/gh-aw/pkg/logger"
)

var geminiToolsLog = logger.New("workflow:gemini_tools")

// computeGeminiToolsCore maps neutral tool names to Gemini CLI built-in tool names
// for use in the tools.core allowlist in .gemini/settings.json.
//
// Neutral tool → Gemini CLI tool mapping:
//   - bash: [cmd, ...]     → run_shell_command(cmd), ... (one entry per command)
//   - bash: * or bash: nil → run_shell_command           (allow all shell commands)
//   - edit: {}             → replace, write_file          (file write tools)
//
// Read-only file system tools are always included as they are essential for
// agentic workflows: glob, grep_search, list_directory, read_file, read_many_files.
//
// See: https://github.com/google-gemini/gemini-cli/blob/main/docs/tools/file-system.md
// See: https://github.com/google-gemini/gemini-cli/blob/main/docs/tools/shell.md
func computeGeminiToolsCore(tools map[string]any) []string {
	return computeGoogleCLIToolsCore(tools, geminiToolsLog)
}
