package cli

import (
	"fmt"
	"strings"

	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/workflow"
)

var bashAllowlistUnsupportedEngineCodemodLog = logger.New("cli:codemod_bash_allowlist_unsupported_engine")

// getBashAllowlistUnsupportedEngineCodemod creates a codemod that emits a guided error when
// a workflow restricts bash commands (tools.bash with a specific command list, an empty list,
// or bash: false) while using an engine that cannot enforce the restriction (for example codex).
//
// The restriction is silently ignored at runtime by such engines, so the compiler rejects it in
// strict mode. It is not auto-corrected because both remediations change semantics: rewriting the
// allow-list to bash: ["*"] widens the effective (declared) permissions, and switching engines
// changes which agent runs the workflow. The user must choose.
func getBashAllowlistUnsupportedEngineCodemod() Codemod {
	return Codemod{
		ID:           "bash-allowlist-unsupported-engine-guided-error",
		Name:         "Detect bash allow-list on an engine that ignores it (manual fix required)",
		Description:  "Detects a restricted 'tools.bash' configuration combined with an engine that does not enforce bash command allow-listing (such as codex), and emits a guided error because the fix (widening to bash: [\"*\"] or switching engines) changes workflow semantics.",
		IntroducedIn: "0.78.0",
		Guided:       true,
		Apply: func(content string, frontmatter map[string]any) (string, bool, error) {
			tools, ok := frontmatter["tools"].(map[string]any)
			if !ok {
				return content, false, nil
			}
			if !workflow.HasBashExplicitRestriction(tools) {
				return content, false, nil
			}

			engineID := extractEngineIDFromFrontmatter(frontmatter)
			engine, err := workflow.GetGlobalEngineRegistry().GetEngine(engineID)
			if err != nil {
				bashAllowlistUnsupportedEngineCodemodLog.Printf("Unknown engine %q, skipping bash allow-list check", engineID)
				return content, false, nil
			}
			if engine.GetCapabilities().BashCommandAllowlist {
				return content, false, nil
			}

			bashAllowlistUnsupportedEngineCodemodLog.Printf("Engine %s ignores the restricted tools.bash configuration, emitting guided error", engineID)

			return content, false, fmt.Errorf(
				"engine '%s' does not support bash command allow-listing: %s is silently ignored at runtime for this engine. "+
					"Manual fix required: switch to an engine that enforces the allow-list (copilot, claude, or gemini), "+
					"or replace the configuration with 'bash: [\"*\"]' to make the unrestricted access explicit. "+
					"See: https://github.github.com/gh-aw/reference/tools/",
				engineID,
				describeBashRestriction(tools["bash"]),
			)
		},
	}
}

// describeBashRestriction renders a short human-readable description of the offending
// tools.bash configuration for use in the guided error message.
func describeBashRestriction(bashConfig any) string {
	switch value := bashConfig.(type) {
	case bool:
		return fmt.Sprintf("'bash: %t'", value)
	case []any:
		if len(value) == 0 {
			return "'bash: []'"
		}
		commands := make([]string, 0, len(value))
		for _, cmd := range value {
			commands = append(commands, fmt.Sprintf("%v", cmd))
		}
		return fmt.Sprintf("'bash: [%s]'", strings.Join(commands, ", "))
	}
	return "the tools.bash configuration"
}
