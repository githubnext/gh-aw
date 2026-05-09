package cli

import "github.com/github/gh-aw/pkg/logger"

var inlineAgentsCodemodLog = logger.New("cli:codemod_inline_agents")

// getInlineAgentsFeatureRemovalCodemod removes deprecated features.inline-agents.
func getInlineAgentsFeatureRemovalCodemod() Codemod {
	return newFieldRemovalCodemod(fieldRemovalCodemodConfig{
		ID:           "features-inline-agents-removal",
		Name:         "Remove deprecated features.inline-agents",
		Description:  "Removes deprecated features.inline-agents. Inline sub-agents are now enabled by default.",
		IntroducedIn: "1.0.0",
		ParentKey:    "features",
		FieldKey:     "inline-agents",
		LogMsg:       "Removed deprecated features.inline-agents",
		Log:          inlineAgentsCodemodLog,
	})
}
