package cli

import "github.com/github/gh-aw/pkg/logger"

var engineModelCodemodLog = logger.New("cli:codemod_engine_model")

// getEngineModelToTopLevelCodemod migrates deprecated engine.model to top-level model.
func getEngineModelToTopLevelCodemod() Codemod {
	return Codemod{
		ID:           "engine-model-to-top-level",
		Name:         "Move engine.model to top-level model",
		Description:  "Moves deprecated 'engine.model' to the top-level 'model' field. The top-level field is the canonical location for LLM model configuration and takes precedence over engine.model.",
		IntroducedIn: "0.78.0",
		Apply: func(content string, frontmatter map[string]any) (string, bool, error) {
			return migrateEngineFieldToTopLevel(
				content,
				frontmatter,
				migrateEngineFieldToTopLevelOptions{
					engineField:            "model",
					targetTopLevelField:    "model",
					preserveTopLevelFields: []string{"model"},
					log:                    engineModelCodemodLog,
					skipInlineMessage:      "Skipping engine.model migration for inline-map engine syntax; migrate to top-level model manually",
					removedMessage:         "Removed deprecated engine.model (top-level model already present)",
					migratedMessage:        "Migrated engine.model to top-level model",
				},
			)
		},
	}
}
