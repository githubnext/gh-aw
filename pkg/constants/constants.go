package constants

// CLIExtensionPrefix is the prefix used in user-facing output to refer to the CLI extension
const CLIExtensionPrefix = "gh aw"

// AllowedExpressions contains the GitHub Actions expressions that can be used in workflow markdown content
var AllowedExpressions = []string{
	"github.workflow",
	"github.repository",
	"github.run_id",
	"github.event.issue.number",
	"needs.task.outputs.text",
}
