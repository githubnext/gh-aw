package workflow

import (
	"github.com/github/gh-aw/pkg/logger"
)

var labelCommandParserLog = logger.New("workflow:label_command_parser")

// expandLabelCommandShorthand takes a label name and returns a map that represents
// the expanded label_command + workflow_dispatch configuration.
// This is the intermediate form stored in the frontmatter "on" map before
// parseOnSection processes it into WorkflowData.LabelCommand.
func expandLabelCommandShorthand(labelName string) map[string]any {
	labelCommandParserLog.Printf("Expanding label-command shorthand for label: %s", labelName)
	return map[string]any{
		"label_command":     labelName,
		"workflow_dispatch": nil,
	}
}
