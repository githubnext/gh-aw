package workflow

import (
	"fmt"
	"regexp"

	"github.com/githubnext/gh-aw/pkg/logger"
)

var secretsValidationLog = logger.New("workflow:secrets_validation")

// secretsExpressionPattern matches GitHub Actions secrets expressions for jobs.secrets validation.
// Pattern matches: ${{ secrets.NAME }} or ${{ secrets.NAME1 || secrets.NAME2 }}
// This is the same pattern used in the github_token schema definition ($defs/github_token).
var secretsExpressionPattern = regexp.MustCompile(`^\$\{\{\s*secrets\.[A-Za-z_][A-Za-z0-9_]*(\s*\|\|\s*secrets\.[A-Za-z_][A-Za-z0-9_]*)*\s*\}\}$`)

// validateSecretsExpression validates that a value is a proper GitHub Actions secrets expression.
// Returns an error if the value is not in the format: ${{ secrets.NAME }} or ${{ secrets.NAME || secrets.NAME2 }}
func validateSecretsExpression(key, value string) error {
	if !secretsExpressionPattern.MatchString(value) {
		secretsValidationLog.Printf("Invalid secret expression detected")
		// Note: We intentionally do NOT include the key name in the error message to avoid
		// logging sensitive information (secret key names) that could expose details about
		// the organization's security infrastructure. The key name is available in the
		// calling context for debugging purposes if needed.
		return fmt.Errorf("invalid secrets expression: must be a GitHub Actions expression with secrets reference (e.g., '${{ secrets.MY_SECRET }}' or '${{ secrets.SECRET1 || secrets.SECRET2 }}')")
	}
	secretsValidationLog.Printf("Valid secret expression validated")
	return nil
}
