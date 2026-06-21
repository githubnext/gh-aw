package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/parser"
	"github.com/github/gh-aw/pkg/workflow"
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
		usesClaude, err := workflowUsesClaudeEngine(workflowFile)
		if err != nil {
			return fmt.Errorf("failed to inspect workflow %s for engine configuration: %w", workflowFile, err)
		}
		if usesClaude {
			return fmt.Errorf("%s is not supported for Claude workflows - set ANTHROPIC_API_KEY instead", claudeCodeOAuthTokenEnvVar)
		}
	}
	return nil
}

func workflowUsesClaudeEngine(workflowFile string) (bool, error) {
	content, err := readWorkflowFileContent(workflowFile)
	if err != nil {
		return false, err
	}
	result, err := parser.ExtractFrontmatterFromContent(content)
	if err != nil {
		return false, err
	}

	compiler := &workflow.Compiler{}
	engineSetting, engineConfig := compiler.ExtractEngineConfig(result.Frontmatter)

	engine := string(constants.CopilotEngine)
	if engineConfig != nil && engineConfig.ID != "" {
		engine = engineConfig.ID
	} else if engineSetting != "" {
		engine = engineSetting
	}
	return strings.EqualFold(engine, string(constants.ClaudeEngine)), nil
}
