package cli

import (
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

var botsCodemodLog = logger.New("cli:codemod_bots")

// getBotsToOnBotsCodemod creates a codemod for moving top-level 'bots' to 'on.bots'
func getBotsToOnBotsCodemod() Codemod {
	return newMoveTopLevelKeyToOnBlockCodemod(moveToOnBlockConfig{
		ID:           "bots-to-on-bots",
		Name:         "Move bots to on.bots",
		Description:  "Moves the top-level 'bots' field to 'on.bots' as per the new frontmatter structure",
		IntroducedIn: "0.10.0",
		FieldKey:     "bots",
		IsInlineSingle: func(v string) bool {
			return strings.HasPrefix(v, "[")
		},
		Log: botsCodemodLog,
	})
}
