package workflow

import (
	"fmt"
	"os"

	"github.com/github/gh-aw/pkg/console"
)

func (c *Compiler) warnUnknownConfiguredModels(data *WorkflowData, markdownPath string) {
	if c.configuredModelValidator == nil {
		return
	}
	for _, warning := range c.configuredModelValidator(data) {
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(
			formatCompilerMessage(markdownPath, "warning", warning)))
		c.IncrementWarningCount()
	}
}
