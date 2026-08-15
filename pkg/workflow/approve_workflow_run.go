package workflow

import "github.com/github/gh-aw/pkg/logger"

var approveWorkflowRunLog = logger.New("workflow:approve_workflow_run")

// ApproveWorkflowRunConfig holds configuration for approving workflow runs from fork pull requests.
type ApproveWorkflowRunConfig struct {
	BaseSafeOutputConfig  `yaml:",inline"`
	Fork                  bool     `yaml:"fork,omitempty"`
	AllowedPullRequests   []string `yaml:"allowed-pull-requests,omitempty"`
	ProtectedFilesExclude []string `yaml:"-"`
}

// parseApproveWorkflowRunConfig handles approve-workflow-run configuration.
func (c *Compiler) parseApproveWorkflowRunConfig(outputMap map[string]any) *ApproveWorkflowRunConfig {
	configData, exists := outputMap["approve-workflow-run"]
	if !exists {
		return nil
	}

	approveWorkflowRunLog.Print("Parsing approve-workflow-run configuration")
	config := &ApproveWorkflowRunConfig{}
	config.Max = defaultIntStr(1)
	if configMap, ok := configData.(map[string]any); ok {
		c.parseBaseSafeOutputConfig(configMap, &config.BaseSafeOutputConfig, 1)
		if fork, ok := configMap["fork"].(bool); ok {
			config.Fork = fork
		}
		config.AllowedPullRequests = ParseStringArrayOrExprFromConfig(configMap, "allowed-pull-requests", approveWorkflowRunLog)
		config.ProtectedFilesExclude = preprocessProtectedFilesField(configMap, approveWorkflowRunLog)
	}
	return config
}
