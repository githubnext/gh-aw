package cli

import (
	"fmt"
	"strings"
)

// convertToGitHubActionsEnv converts environment variables from shell syntax to GitHub Actions syntax
// Converts "${TOKEN_NAME}" to "${{ secrets.TOKEN_NAME }}"
// Leaves existing GitHub Actions syntax unchanged
func convertToGitHubActionsEnv(env interface{}) map[string]string {
	result := make(map[string]string)

	if envMap, ok := env.(map[string]interface{}); ok {
		for key, value := range envMap {
			if valueStr, ok := value.(string); ok {
				// Only convert shell syntax ${TOKEN_NAME}, leave GitHub Actions syntax unchanged
				if strings.HasPrefix(valueStr, "${") && strings.HasSuffix(valueStr, "}") && !strings.Contains(valueStr, "{{") {
					tokenName := valueStr[2 : len(valueStr)-1] // Remove ${ and }
					result[key] = fmt.Sprintf("${{ secrets.%s }}", tokenName)
				} else {
					// Keep as-is if not shell syntax or already GitHub Actions syntax
					result[key] = valueStr
				}
			}
		}
	}

	return result
}
