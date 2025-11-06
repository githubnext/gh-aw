package cli

import (
	"os"
	"strings"
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
	return `
╔══════════════════════════════════════════════════════════════════════════╗
║ GitHub Codespace Permission Error                                       ║
╚══════════════════════════════════════════════════════════════════════════╝

The default GitHub token in Codespaces does not have 'actions:write' 
permission, which is required to trigger GitHub Actions workflows.

To fix this, you need to update your devcontainer configuration to use
a Personal Access Token (PAT) with the necessary permissions.

📚 Documentation:
   https://docs.github.com/en/codespaces/managing-your-codespaces/managing-secrets-for-your-codespaces

🔧 Quick Fix:
   1. Create a Personal Access Token with 'actions:write' permission
   2. Add it as a Codespace secret named GITHUB_TOKEN
   3. Rebuild your codespace or restart the session

For more information about devcontainer configuration, see:
   https://docs.github.com/en/codespaces/setting-up-your-project-for-codespaces/adding-a-dev-container-configuration

`
}
