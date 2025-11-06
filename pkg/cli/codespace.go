package cli

import (
	"os"
	"strings"

	"github.com/githubnext/gh-aw/pkg/console"
)

// isRunningInCodespace checks if the current process is running in a GitHub Codespace
// by checking for the CODESPACES environment variable
func isRunningInCodespace() bool {
	// GitHub Codespaces sets CODESPACES=true environment variable
	return strings.ToLower(os.Getenv("CODESPACES")) == "true"
}

// is403PermissionError checks if an error message contains indicators of a 403 permission error
func is403PermissionError(errorMsg string) bool {
	errorLower := strings.ToLower(errorMsg)
	// Check for common 403 error patterns
	return strings.Contains(errorLower, "403") ||
		strings.Contains(errorLower, "forbidden") ||
		(strings.Contains(errorLower, "permission") && strings.Contains(errorLower, "denied"))
}

// getCodespacePermissionErrorMessage returns a helpful error message for codespace users
// experiencing 403 permission errors when running workflows
func getCodespacePermissionErrorMessage() string {
	var msg strings.Builder

	msg.WriteString("\n")
	msg.WriteString(console.FormatErrorMessage("GitHub Codespace Permission Error"))
	msg.WriteString("\n\n")

	msg.WriteString("The default GitHub token in Codespaces does not have 'actions:write'\n")
	msg.WriteString("permission, which is required to trigger GitHub Actions workflows.\n\n")

	msg.WriteString("To fix this, you need to configure repository permissions in your\n")
	msg.WriteString("devcontainer.json file.\n\n")

	msg.WriteString(console.FormatInfoMessage("Quick Fix:"))
	msg.WriteString("\n")
	msg.WriteString("   Add the following to .devcontainer/devcontainer.json:\n\n")

	msg.WriteString("   {\n")
	msg.WriteString("     \"customizations\": {\n")
	msg.WriteString("       \"codespaces\": {\n")
	msg.WriteString("         \"repositories\": {\n")
	msg.WriteString("           \"owner/repo\": {\n")
	msg.WriteString("             \"permissions\": {\n")
	msg.WriteString("               \"actions\": \"write\"\n")
	msg.WriteString("             }\n")
	msg.WriteString("           }\n")
	msg.WriteString("         }\n")
	msg.WriteString("       }\n")
	msg.WriteString("     }\n")
	msg.WriteString("   }\n\n")

	msg.WriteString("   Then rebuild your codespace to apply the changes.\n\n")

	msg.WriteString(console.FormatInfoMessage("Documentation:"))
	msg.WriteString("\n")
	msg.WriteString("   https://docs.github.com/en/codespaces/setting-up-your-project-for-codespaces/adding-a-dev-container-configuration\n\n")

	return msg.String()
}
