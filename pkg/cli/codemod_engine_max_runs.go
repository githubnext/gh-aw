package cli

import "github.com/github/gh-aw/pkg/logger"

var engineMaxRunsCodemodLog = logger.New("cli:codemod_engine_max_runs")

// getEngineMaxRunsToTopLevelCodemod migrates deprecated engine.max-runs to
// top-level max-turns.
func getEngineMaxRunsToTopLevelCodemod() Codemod {
	return Codemod{
		ID:           "engine-max-runs-to-top-level",
		Name:         "Move engine.max-runs to top-level max-turns",
		Description:  "Moves deprecated 'engine.max-runs' to top-level 'max-turns' so AWF enforces invocation caps consistently across all engines.",
		IntroducedIn: "0.17.0",
		Apply: func(content string, frontmatter map[string]any) (string, bool, error) {
			return migrateEngineFieldToTopLevel(
				content,
				frontmatter,
				"max-runs",
				"max-turns",
				[]string{"max-runs", "max-turns"},
				engineMaxRunsCodemodLog,
				"Skipping engine.max-runs migration for inline-map engine syntax; migrate to top-level max-turns manually",
				"Removed deprecated engine.max-runs (top-level max-runs/max-turns already present)",
				"Migrated engine.max-runs to top-level max-turns",
			)
		},
	}
}
