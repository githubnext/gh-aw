package cli

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// checkSecretExists checks if a secret exists in the repository using GitHub CLI
func checkSecretExists(secretName string) (bool, error) {
	// Use gh CLI to list repository secrets
	cmd := exec.Command("gh", "secret", "list", "--json", "name")
	output, err := cmd.Output()
	if err != nil {
		// Check if it's a 403 error by examining the error
		if exitError, ok := err.(*exec.ExitError); ok {
			if strings.Contains(string(exitError.Stderr), "403") {
				return false, fmt.Errorf("403 access denied")
			}
		}
		return false, fmt.Errorf("failed to list secrets: %w", err)
	}

	// Parse the JSON output
	var secrets []struct {
		Name string `json:"name"`
	}

	if err := json.Unmarshal(output, &secrets); err != nil {
		return false, fmt.Errorf("failed to parse secrets list: %w", err)
	}

	// Check if our secret exists
	for _, secret := range secrets {
		if secret.Name == secretName {
			return true, nil
		}
	}

	return false, nil
}
