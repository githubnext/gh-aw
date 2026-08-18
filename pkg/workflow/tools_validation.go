package workflow

import (
	"errors"

	"github.com/github/gh-aw/pkg/logger"
)

var toolsValidationLog = logger.New("workflow:tools_validation")

// validateBashToolConfig validates that bash tool configuration is explicit (not nil/anonymous)
func validateBashToolConfig(tools *Tools, workflowName string) error {
	if tools == nil {
		return nil
	}

	// Check if bash is present in the raw map but Bash field is nil
	// This indicates the anonymous syntax (bash:) was used
	if rawMap := tools.ToMap(); rawMap != nil {
		if _, hasBash := rawMap["bash"]; hasBash && tools.Bash == nil {
			toolsValidationLog.Printf("Invalid bash tool configuration in workflow: %s", workflowName)
			return errors.New("invalid bash tool configuration: anonymous syntax 'bash:' is not supported. Use 'bash: true' (enable all commands), 'bash: false' (disable), or 'bash: [\"cmd1\", \"cmd2\"]' (specific commands). Run 'gh aw fix' to automatically migrate")
		}
	}

	return nil
}

// validateCLIProxyBashCompatibility validates that tools.cli-proxy is not enabled when shell
// execution is fully refused (tools.bash: false or tools.bash: []).
//
// cli-proxy mounts MCP servers as CLI executables on PATH and instructs the agent to invoke
// them from a shell. Without bash the agent cannot call any of them, so the generated prompt
// would point at an unusable tool path.
//
// tools is the merged tools map before default-tool resolution, so an explicit "bash: false"
// is still visible.
func validateCLIProxyBashCompatibility(tools map[string]any, workflowName string) error {
	if !isBashExplicitlyRefused(tools) {
		return nil
	}
	cliProxy, hasCLIProxy := tools["cli-proxy"]
	if enabled, ok := cliProxy.(bool); hasCLIProxy && ok && enabled {
		toolsValidationLog.Printf("cli-proxy enabled with bash disabled in workflow: %s", workflowName)
		return NewValidationError(
			"tools.cli-proxy",
			"true",
			"'tools.cli-proxy: true' requires shell access, but 'tools.bash' is disabled: CLI-mounted MCP servers can only be invoked from a shell",
			"Set 'tools.cli-proxy: false' (MCP servers stay reachable as MCP tools), or enable bash:\n\ntools:\n  bash: [\"cat\", \"ls\", \"grep\"]\n  cli-proxy: true\n\nRun 'gh aw fix' to apply this change automatically.",
		)
	}
	return nil
}

// isBashExplicitlyRefused reports whether the given tools map (before default-tool resolution)
// explicitly refuses shell execution, i.e. bash: false or bash: [] (empty allowlist).
func isBashExplicitlyRefused(tools map[string]any) bool {
	bashVal, hasBash := tools["bash"]
	if !hasBash {
		return false
	}
	switch v := bashVal.(type) {
	case bool:
		return !v
	case []any:
		return len(v) == 0
	}
	return false
}
