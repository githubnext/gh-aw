package workflow

import (
	"fmt"
	"strings"

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
			return fmt.Errorf("invalid bash tool configuration: anonymous syntax 'bash:' is not supported. Use 'bash: true' (enable all commands), 'bash: false' (disable), or 'bash: [\"cmd1\", \"cmd2\"]' (specific commands). Run 'gh aw fix' to automatically migrate")
		}
	}

	return nil
}

// isGitToolAllowed checks if git commands are allowed in bash tool configuration
func isGitToolAllowed(tools *Tools) bool {
	if tools == nil {
		// No tools configured - defaults will be applied which include git for PR operations
		return true
	}

	if tools.Bash == nil {
		// No bash tool configured - defaults will be applied which include git for PR operations
		return true
	}

	// If AllowedCommands is nil or empty, check which case it is:
	// - nil AllowedCommands = bash: true (all commands allowed, including git)
	// - empty slice = bash: false (explicitly disabled)
	if tools.Bash.AllowedCommands == nil {
		// bash: true - all commands allowed
		return true
	}

	if len(tools.Bash.AllowedCommands) == 0 {
		// bash: false or bash: [] - explicitly disabled or no commands
		return false
	}

	// Check if git is in the allowed commands list
	for _, cmd := range tools.Bash.AllowedCommands {
		if cmd == "*" {
			// Wildcard allows all commands
			return true
		}
		if cmd == "git" {
			// Exact match for git command
			return true
		}
		// Check for git with wildcards: "git *", "git:*", "git checkout:*", etc.
		if strings.HasPrefix(cmd, "git ") || strings.HasPrefix(cmd, "git:") {
			return true
		}
	}

	return false
}

// Note: validateGitToolForSafeOutputs was removed because git commands are automatically
// injected by the compiler when safe-outputs needs them (see compiler_safe_outputs.go).
// The validation was misleading - it would fail even though the compiler would add the
// necessary git commands during compilation.
