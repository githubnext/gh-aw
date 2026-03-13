package workflow

import (
	"github.com/github/gh-aw/pkg/logger"
)

var callWorkflowLog = logger.New("workflow:call_workflow")

// CallWorkflowConfig holds configuration for calling workflows via workflow_call chaining.
// Unlike dispatch-workflow (which uses the GitHub Actions API at runtime), call-workflow
// generates static conditional `uses:` jobs at compile time. The agent selects which
// worker to activate at runtime; the compiler validates and wires up all fan-out jobs.
type CallWorkflowConfig struct {
	BaseSafeOutputConfig `yaml:",inline"`
	Workflows            []string          `yaml:"workflows,omitempty"`      // List of workflow names (without .md extension) to allow calling
	WorkflowFiles        map[string]string `yaml:"workflow_files,omitempty"` // Map of workflow name to file path (relative, e.g. ./.github/workflows/x.lock.yml) - populated at compile time
}

// parseCallWorkflowConfig handles call-workflow configuration
func (c *Compiler) parseCallWorkflowConfig(outputMap map[string]any) *CallWorkflowConfig {
	callWorkflowLog.Print("Parsing call-workflow configuration")
	if configData, exists := outputMap["call-workflow"]; exists {
		callWorkflowConfig := &CallWorkflowConfig{}

		// Check if it's a list of workflow names (array format)
		if workflowsArray, ok := configData.([]any); ok {
			callWorkflowLog.Printf("Found call-workflow as array with %d workflows", len(workflowsArray))
			for _, workflow := range workflowsArray {
				if workflowStr, ok := workflow.(string); ok {
					callWorkflowConfig.Workflows = append(callWorkflowConfig.Workflows, workflowStr)
				}
			}
			// Set default max to 1
			callWorkflowConfig.Max = defaultIntStr(1)
			return callWorkflowConfig
		}

		// Check if it's a map with configuration options
		if configMap, ok := configData.(map[string]any); ok {
			callWorkflowLog.Print("Found call-workflow config map")

			// Parse workflows list
			if workflows, exists := configMap["workflows"]; exists {
				if workflowsArray, ok := workflows.([]any); ok {
					for _, workflow := range workflowsArray {
						if workflowStr, ok := workflow.(string); ok {
							callWorkflowConfig.Workflows = append(callWorkflowConfig.Workflows, workflowStr)
						}
					}
				}
			}

			// Parse common base fields with default max of 1
			c.parseBaseSafeOutputConfig(configMap, &callWorkflowConfig.BaseSafeOutputConfig, 1)

			// Cap max at 50 (absolute maximum allowed) – only for literal integer values
			if maxVal := templatableIntValue(callWorkflowConfig.Max); maxVal > 50 {
				callWorkflowLog.Printf("Max value %d exceeds limit, capping at 50", maxVal)
				callWorkflowConfig.Max = defaultIntStr(50)
			}

			callWorkflowLog.Printf("Parsed call-workflow config: max=%v, workflows=%v",
				callWorkflowConfig.Max, callWorkflowConfig.Workflows)
			return callWorkflowConfig
		}
	}

	return nil
}
