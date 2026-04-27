package cli

import "github.com/github/gh-aw/pkg/logger"

var byokCopilotCodemodLog = logger.New("cli:codemod_byok_copilot")

// getByokCopilotFeatureRemovalCodemod removes deprecated features.byok-copilot.
func getByokCopilotFeatureRemovalCodemod() Codemod {
	return newFieldRemovalCodemod(fieldRemovalCodemodConfig{
		ID:           "features-byok-copilot-removal",
		Name:         "Remove deprecated features.byok-copilot",
		Description:  "Removes deprecated features.byok-copilot. Copilot now uses BYOK behavior by default.",
		IntroducedIn: "1.0.0",
		ParentKey:    "features",
		FieldKey:     "byok-copilot",
		LogMsg:       "Removed deprecated features.byok-copilot",
		Log:          byokCopilotCodemodLog,
	})
}
