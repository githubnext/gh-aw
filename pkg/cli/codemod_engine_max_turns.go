package cli

import "github.com/github/gh-aw/pkg/logger"

var engineMaxTurnsCodemodLog = logger.New("cli:codemod_engine_max_turns")

// getEngineMaxTurnsToTopLevelCodemod migrates deprecated engine.max-turns to
// top-level max-turns.
func getEngineMaxTurnsToTopLevelCodemod() Codemod {
	return Codemod{
		ID:           "engine-max-turns-to-top-level",
		Name:         "Move engine.max-turns to top-level max-turns",
		Description:  "Moves deprecated 'engine.max-turns' to top-level 'max-turns' so AWF enforces turn caps consistently across all agentic engines.",
		IntroducedIn: "0.68.4",
		Apply: func(content string, frontmatter map[string]any) (string, bool, error) {
			return migrateEngineFieldToTopLevel(
				content,
				frontmatter,
				migrateEngineFieldToTopLevelOptions{
					engineField:            "max-turns",
					targetTopLevelField:    "max-turns",
					preserveTopLevelFields: []string{"max-turns"},
					log:                    engineMaxTurnsCodemodLog,
					skipInlineMessage:      "Skipping engine.max-turns migration for inline-map engine syntax; migrate to top-level max-turns manually",
					removedMessage:         "Removed deprecated engine.max-turns (top-level max-turns already present)",
					migratedMessage:        "Migrated engine.max-turns to top-level max-turns",
				},
			)
		},
	}
}
