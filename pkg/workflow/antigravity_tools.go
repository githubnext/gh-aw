package workflow

// This file provides Antigravity engine tool configuration logic.
//
// computeAntigravityToolsCore is a thin wrapper around computeGoogleCLIToolsCore
// that preserves the existing function signature for tests and callers while
// delegating all logic to the shared implementation.
//
// The settings-step generator (generateSettingsStep) lives on googleCLIEngine and
// calls computeGoogleCLIToolsCore directly with the engine's configured logger.

import (
	"github.com/github/gh-aw/pkg/logger"
)

var antigravityToolsLog = logger.New("workflow:antigravity_tools")

// computeAntigravityToolsCore maps neutral tool names to Antigravity CLI built-in tool names
// for use in the tools.core allowlist in .antigravity/settings.json.
//
// Neutral tool → Antigravity CLI tool mapping:
//   - bash: [cmd, ...]     → run_shell_command(cmd), ... (one entry per command)
//   - bash: * or bash: nil → run_shell_command           (allow all shell commands)
//   - edit: {}             → replace, write_file          (file write tools)
//
// Read-only file system tools are always included as they are essential for
// agentic workflows: glob, grep_search, list_directory, read_file, read_many_files.
//
// See: https://antigravity.google/docs/cli-overview
func computeAntigravityToolsCore(tools map[string]any) []string {
	return computeGoogleCLIToolsCore(tools, antigravityToolsLog)
}
