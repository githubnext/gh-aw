package cli

import (
	"github.com/github/gh-aw/pkg/logger"
	frontmattercodemod "github.com/github/gh-aw/pkg/workflow/codemod"
)

var botsCodemodLog = logger.New("cli:codemod_bots")

func getBotsToOnBotsCodemod() Codemod {
	mod := newMoveTopLevelKeyToOnBlockCodemod(moveToOnBlockConfig{
		ID:           "bots-to-on-bots",
		Name:         "Move bots to on.bots",
		Description:  "Moves the top-level 'bots' field to 'on.bots' as per the new frontmatter structure",
		IntroducedIn: "0.10.0",
		FieldKey:     "bots",
		Log:          botsCodemodLog,
	})
	baseApply := mod.Apply
	mod.Apply = func(content string, frontmatter map[string]any) (string, bool, error) {
		topBots, hasTopBots := frontmatter["bots"]
		onMap, hasOnMap := frontmatter["on"].(map[string]any)
		onBots, hasOnBots := onMap["bots"]
		if !hasTopBots || !hasOnMap || !hasOnBots {
			return baseApply(content, frontmatter)
		}
		if _, ok := mergeLegacyBots(onBots, topBots); !ok {
			return content, false, nil
		}
		return frontmattercodemod.Apply(content, func(f *frontmattercodemod.Frontmatter) error {
			return f.MergeSequence([]string{"bots"}, []string{"on", "bots"})
		})
	}
	return mod
}

func mergeLegacyBots(onBots, topBots any) ([]string, bool) {
	merged := make([]string, 0)
	seen := make(map[string]struct{})
	for _, value := range []any{onBots, topBots} {
		bots, ok := value.([]any)
		if !ok {
			return nil, false
		}
		for _, botValue := range bots {
			bot, ok := botValue.(string)
			if !ok {
				return nil, false
			}
			if _, exists := seen[bot]; !exists {
				seen[bot] = struct{}{}
				merged = append(merged, bot)
			}
		}
	}
	return merged, true
}
