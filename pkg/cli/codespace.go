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

To fix this, you need to configure repository permissions in your 
devcontainer.json file.

🔧 Quick Fix:
   Add the following to .devcontainer/devcontainer.json:

   {
     "customizations": {
       "codespaces": {
         "repositories": {
           "owner/repo": {
             "permissions": {
               "actions": "write"
             }
           }
         }
       }
     }
   }

   Then rebuild your codespace to apply the changes.

📚 Documentation:
   https://docs.github.com/en/codespaces/setting-up-your-project-for-codespaces/adding-a-dev-container-configuration

`
}
