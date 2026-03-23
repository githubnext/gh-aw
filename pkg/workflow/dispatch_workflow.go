package workflow

import (
	"strings"

	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/parser"
	"github.com/goccy/go-yaml"
)

var dispatchWorkflowLog = logger.New("workflow:dispatch_workflow")

// awContextInputDescription is the description for the aw_context workflow_dispatch input.
// It signals to users that this input is managed internally by the agentic workflow system.
const awContextInputDescription = "(Internal) JSON context injected by the calling agentic workflow. Not intended for direct user input."

// injectAwContextIntoOnYAML adds the aw_context input to the workflow_dispatch trigger
// in the given on-section YAML string, if workflow_dispatch is present.
// The aw_context input carries caller metadata (repo, run_id, workflow_id, etc.) and is
// marked as optional and internal. This operates on the final on-section YAML string,
// after all other trigger processing, to avoid interfering with label_command or command
// trigger merging.
func injectAwContextIntoOnYAML(onSection string) string {
	if !strings.Contains(onSection, "workflow_dispatch") {
		return onSection
	}

	// Parse the on section YAML
	var onData map[string]any
	if err := yaml.Unmarshal([]byte(onSection), &onData); err != nil {
		dispatchWorkflowLog.Printf("Warning: failed to parse on section for aw_context injection: %v", err)
		return onSection
	}

	onMap, ok := onData["on"].(map[string]any)
	if !ok {
		return onSection
	}

	wdVal, hasWD := onMap["workflow_dispatch"]
	if !hasWD {
		return onSection
	}

	// Ensure workflow_dispatch is a map (it may be nil for bare "workflow_dispatch:" triggers)
	var wdMap map[string]any
	if wdVal == nil {
		wdMap = make(map[string]any)
	} else if m, ok := wdVal.(map[string]any); ok {
		wdMap = m
	} else {
		return onSection
	}

	// Get or create the inputs map
	var inputsMap map[string]any
	if existingInputs, hasInputs := wdMap["inputs"]; hasInputs {
		if im, ok := existingInputs.(map[string]any); ok {
			inputsMap = im
		} else {
			inputsMap = make(map[string]any)
		}
	} else {
		inputsMap = make(map[string]any)
	}

	// Inject aw_context as an optional internal input (skip if already present)
	if _, alreadyExists := inputsMap["aw_context"]; alreadyExists {
		return onSection
	}
	inputsMap["aw_context"] = map[string]any{
		"description": awContextInputDescription,
		"required":    false,
		"default":     "",
		"type":        "string",
	}

	wdMap["inputs"] = inputsMap
	onMap["workflow_dispatch"] = wdMap

	// Re-serialize to YAML preserving alphabetical key order and applying post-processing
	// identical to extractTopLevelYAMLSection: QuoteCronExpressions + CleanYAMLNullValues.
	orderedOn := OrderMapFields(onMap, []string{})
	wrappedData := yaml.MapSlice{{Key: "on", Value: orderedOn}}
	newYAML, err := yaml.MarshalWithOptions(wrappedData, DefaultMarshalOptions...)
	if err != nil {
		dispatchWorkflowLog.Printf("Warning: failed to marshal on section after aw_context injection: %v", err)
		return onSection
	}

	result := strings.TrimSuffix(string(newYAML), "\n")
	result = parser.QuoteCronExpressions(result)
	result = CleanYAMLNullValues(result)
	dispatchWorkflowLog.Print("Injected aw_context input into on.workflow_dispatch.inputs")
	return result
}

// DispatchWorkflowConfig holds configuration for dispatching workflows from agent output
type DispatchWorkflowConfig struct {
	BaseSafeOutputConfig `yaml:",inline"`
	Workflows            []string          `yaml:"workflows,omitempty"`      // List of workflow names (without .md extension) to allow dispatching
	WorkflowFiles        map[string]string `yaml:"workflow_files,omitempty"` // Map of workflow name to file extension (.lock.yml or .yml) - populated at compile time
	TargetRepoSlug       string            `yaml:"target-repo,omitempty"`    // Target repository for cross-repo dispatch (owner/repo or GitHub Actions expression)
	TargetRef            string            `yaml:"target-ref,omitempty"`     // Target ref for cross-repo dispatch; overrides the caller's GITHUB_REF
}

// parseDispatchWorkflowConfig handles dispatch-workflow configuration
func (c *Compiler) parseDispatchWorkflowConfig(outputMap map[string]any) *DispatchWorkflowConfig {
	dispatchWorkflowLog.Print("Parsing dispatch-workflow configuration")
	if configData, exists := outputMap["dispatch-workflow"]; exists {
		dispatchWorkflowConfig := &DispatchWorkflowConfig{}

		// Check if it's a list of workflow names (array format)
		if workflowsArray, ok := configData.([]any); ok {
			dispatchWorkflowLog.Printf("Found dispatch-workflow as array with %d workflows", len(workflowsArray))
			for _, workflow := range workflowsArray {
				if workflowStr, ok := workflow.(string); ok {
					dispatchWorkflowConfig.Workflows = append(dispatchWorkflowConfig.Workflows, workflowStr)
				}
			}
			// Set default max to 1
			dispatchWorkflowConfig.Max = defaultIntStr(1)
			return dispatchWorkflowConfig
		}

		// Check if it's a map with configuration options
		if configMap, ok := configData.(map[string]any); ok {
			dispatchWorkflowLog.Print("Found dispatch-workflow config map")

			// Parse workflows list
			if workflows, exists := configMap["workflows"]; exists {
				if workflowsArray, ok := workflows.([]any); ok {
					for _, workflow := range workflowsArray {
						if workflowStr, ok := workflow.(string); ok {
							dispatchWorkflowConfig.Workflows = append(dispatchWorkflowConfig.Workflows, workflowStr)
						}
					}
				}
			}

			// Parse common base fields with default max of 1
			c.parseBaseSafeOutputConfig(configMap, &dispatchWorkflowConfig.BaseSafeOutputConfig, 1)

			// Parse target-ref (optional ref for cross-repo dispatch)
			if targetRef, ok := configMap["target-ref"].(string); ok && targetRef != "" {
				dispatchWorkflowConfig.TargetRef = targetRef
			}

			// Cap max at 50 (absolute maximum allowed) – only for literal integer values
			if maxVal := templatableIntValue(dispatchWorkflowConfig.Max); maxVal > 50 {
				dispatchWorkflowLog.Printf("Max value %d exceeds limit, capping at 50", maxVal)
				dispatchWorkflowConfig.Max = defaultIntStr(50)
			}

			dispatchWorkflowLog.Printf("Parsed dispatch-workflow config: max=%v, workflows=%v",
				dispatchWorkflowConfig.Max, dispatchWorkflowConfig.Workflows)
			return dispatchWorkflowConfig
		}
	}

	return nil
}
