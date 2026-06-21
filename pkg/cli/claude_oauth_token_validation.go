package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/github/gh-aw/pkg/constants"
)

const claudeCodeOAuthTokenEnvVar = "CLAUDE_CODE_OAUTH_TOKEN"

func validateUnsupportedClaudeOAuthTokenForEngine(engine string) error {
	if strings.TrimSpace(os.Getenv(claudeCodeOAuthTokenEnvVar)) == "" {
		return nil
	}
	if strings.EqualFold(strings.TrimSpace(engine), string(constants.ClaudeEngine)) {
		return fmt.Errorf("%s is not supported for Claude workflows - set ANTHROPIC_API_KEY instead", claudeCodeOAuthTokenEnvVar)
	}
	return nil
}

func validateUnsupportedClaudeOAuthTokenForWorkflowFiles(workflowFiles []string, engineOverride string) error {
	if err := validateUnsupportedClaudeOAuthTokenForEngine(engineOverride); err != nil {
		return err
	}
	if strings.TrimSpace(os.Getenv(claudeCodeOAuthTokenEnvVar)) == "" {
		return nil
	}
	for _, workflowFile := range workflowFiles {
		engine, _, _ := extractEngineConfigFromFile(workflowFile)
		if strings.EqualFold(engine, string(constants.ClaudeEngine)) {
			return fmt.Errorf("%s is not supported for Claude workflows - set ANTHROPIC_API_KEY instead", claudeCodeOAuthTokenEnvVar)
		}
	}
	return nil
}
