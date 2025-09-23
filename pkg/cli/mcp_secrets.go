package cli

import (
	"fmt"
	"strings"

	"github.com/githubnext/gh-aw/pkg/console"
)

// checkAndSuggestSecrets checks if required secrets exist in the repository and suggests CLI commands to add them
func checkAndSuggestSecrets(toolConfig map[string]any, verbose bool) error {
	// Extract environment variables from the tool config
	var requiredSecrets []string

	// Check for environment variables in both new and old formats
	var envMap map[string]string
	
	// Check new format - direct env field
	if env, hasEnv := toolConfig["env"].(map[string]any); hasEnv {
		envMap = make(map[string]string)
		for k, v := range env {
			if str, ok := v.(string); ok {
				envMap[k] = str
			}
		}
	} else if mcpSection, ok := toolConfig["mcp"].(map[string]any); ok {
		// Fall back to old format - nested mcp.env field
		if env, hasEnv := mcpSection["env"].(map[string]string); hasEnv {
			envMap = env
		}
	}
	
	if envMap != nil {
		for _, value := range envMap {
			// Extract secret name from GitHub Actions syntax: ${{ secrets.SECRET_NAME }}
			if strings.HasPrefix(value, "${{ secrets.") && strings.HasSuffix(value, " }}") {
				secretName := value[12 : len(value)-3] // Remove "${{ secrets." and " }}"
				requiredSecrets = append(requiredSecrets, secretName)
			}
		}
	}

	if len(requiredSecrets) == 0 {
		return nil
	}

	if verbose {
		fmt.Println(console.FormatInfoMessage("Checking repository secrets..."))
	}

	// Check each secret using GitHub CLI
	var missingSecrets []string
	for _, secretName := range requiredSecrets {
		exists, err := checkSecretExists(secretName)
		if err != nil {
			// If we get a 403 error, ignore it as requested
			if strings.Contains(err.Error(), "403") {
				if verbose {
					fmt.Println(console.FormatWarningMessage("Repository secrets check skipped (insufficient permissions)"))
				}
				return nil
			}
			return err
		}

		if !exists {
			missingSecrets = append(missingSecrets, secretName)
		}
	}

	// Suggest CLI commands for missing secrets
	if len(missingSecrets) > 0 {
		fmt.Println(console.FormatWarningMessage("The following secrets are required but not found in the repository:"))
		for _, secretName := range missingSecrets {
			fmt.Println(console.FormatInfoMessage(fmt.Sprintf("To add %s secret:", secretName)))
			fmt.Println(console.FormatCommandMessage(fmt.Sprintf("gh secret set %s", secretName)))
		}
	} else if verbose {
		fmt.Println(console.FormatSuccessMessage("All required secrets are available in the repository"))
	}

	return nil
}
