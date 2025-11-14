package cli

import "fmt"

// ValidateEngine validates the engine flag value
func ValidateEngine(engine string) error {
	if engine != "" && engine != "claude" && engine != "codex" && engine != "copilot" {
		return fmt.Errorf("invalid engine value '%s'. Must be 'claude', 'codex', or 'copilot'", engine)
	}
	return nil
}
