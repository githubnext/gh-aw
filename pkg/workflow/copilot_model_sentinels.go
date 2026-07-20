package workflow

import (
	"strings"

	"github.com/github/gh-aw/pkg/constants"
)

func isCopilotAutomaticSelectionSentinel(model string) bool {
	normalizedModel := strings.ToLower(strings.TrimSpace(model))
	return normalizedModel == constants.CopilotAutoModelSentinel || normalizedModel == constants.CopilotNoModelSentinel
}
