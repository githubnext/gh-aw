package cli

import (
	"fmt"

	"github.com/githubnext/gh-aw/pkg/console"
	"github.com/githubnext/gh-aw/pkg/constants"
	"github.com/spf13/cobra"
)

// versionInfo is set by the main package
var versionInfo = "dev"

// SetVersionInfo sets the version information
func SetVersionInfo(version string) {
	versionInfo = version
}

// NewVersionCommand creates the version command
func NewVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Show version information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(console.FormatInfoMessage(fmt.Sprintf("%s version %s", constants.CLIExtensionPrefix, versionInfo)))
			fmt.Println(console.FormatInfoMessage("GitHub Agentic Workflows CLI from GitHub Next"))
		},
	}
}
