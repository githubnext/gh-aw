package workflow

import (
	"fmt"
	"strings"

	"github.com/goccy/go-yaml"
)

// Step represents a GitHub Actions workflow step
// It supports both the run and uses formats, along with all standard step properties
type Step struct {
	// Step identification
	ID   string `yaml:"id,omitempty"`
	Name string `yaml:"name,omitempty"`

	// Step execution - one of these should be set
	Run  string         `yaml:"run,omitempty"`
	Uses string         `yaml:"uses,omitempty"`
	With map[string]any `yaml:"with,omitempty"`

	// Step control
	If       string `yaml:"if,omitempty"`
	Continue string `yaml:"continue-on-error,omitempty"`

	// Environment and working directory
	Env              map[string]string `yaml:"env,omitempty"`
	WorkingDirectory string            `yaml:"working-directory,omitempty"`
	Shell            string            `yaml:"shell,omitempty"`

	// Timeout
	TimeoutMinutes int `yaml:"timeout-minutes,omitempty"`
}

// WorkflowSteps represents the different placement positions for custom steps in a workflow
// Steps can be placed in multiple positions relative to the agent execution:
//   - Pre: Before checkout and runtime setup
//   - PreAgent: After setup but before agent execution (where legacy "steps" field goes)
//   - PostAgent: Immediately after agent execution (where legacy "post-steps" field goes)
//   - Post: After all other steps are complete
type WorkflowSteps struct {
	Pre       []Step `yaml:"pre,omitempty"`
	PreAgent  []Step `yaml:"pre-agent,omitempty"`
	PostAgent []Step `yaml:"post-agent,omitempty"`
	Post      []Step `yaml:"post,omitempty"`
}

// ParseStepsFromFrontmatter parses the steps configuration from frontmatter
// It supports both legacy array format and new object format with named positions
func ParseStepsFromFrontmatter(data any) (*WorkflowSteps, error) {
	if data == nil {
		return nil, nil
	}

	steps := &WorkflowSteps{}

	// Check if it's an array (legacy format or simple pre-agent steps)
	if arrayData, ok := data.([]any); ok {
		// This is the legacy array format - these steps go in PreAgent position
		parsedSteps, err := parseStepArray(arrayData)
		if err != nil {
			return nil, err
		}
		steps.PreAgent = parsedSteps
		return steps, nil
	}

	// Check if it's an object with named positions
	if objData, ok := data.(map[string]any); ok {
		// Parse each named position
		if preData, ok := objData["pre"]; ok {
			if preArray, ok := preData.([]any); ok {
				parsedSteps, err := parseStepArray(preArray)
				if err != nil {
					return nil, fmt.Errorf("error parsing pre steps: %w", err)
				}
				steps.Pre = parsedSteps
			}
		}

		if preAgentData, ok := objData["pre-agent"]; ok {
			if preAgentArray, ok := preAgentData.([]any); ok {
				parsedSteps, err := parseStepArray(preAgentArray)
				if err != nil {
					return nil, fmt.Errorf("error parsing pre-agent steps: %w", err)
				}
				steps.PreAgent = parsedSteps
			}
		}

		if postAgentData, ok := objData["post-agent"]; ok {
			if postAgentArray, ok := postAgentData.([]any); ok {
				parsedSteps, err := parseStepArray(postAgentArray)
				if err != nil {
					return nil, fmt.Errorf("error parsing post-agent steps: %w", err)
				}
				steps.PostAgent = parsedSteps
			}
		}

		if postData, ok := objData["post"]; ok {
			if postArray, ok := postData.([]any); ok {
				parsedSteps, err := parseStepArray(postArray)
				if err != nil {
					return nil, fmt.Errorf("error parsing post steps: %w", err)
				}
				steps.Post = parsedSteps
			}
		}

		return steps, nil
	}

	return nil, fmt.Errorf("steps must be either an array or an object with named positions")
}

// parseStepArray parses an array of step definitions into Step structs
func parseStepArray(data []any) ([]Step, error) {
	steps := make([]Step, 0, len(data))

	for i, item := range data {
		// Convert the item to YAML and back to properly parse it into Step struct
		yamlBytes, err := yaml.Marshal(item)
		if err != nil {
			return nil, fmt.Errorf("error marshaling step %d: %w", i, err)
		}

		var step Step
		if err := yaml.Unmarshal(yamlBytes, &step); err != nil {
			return nil, fmt.Errorf("error parsing step %d: %w", i, err)
		}

		steps = append(steps, step)
	}

	return steps, nil
}

// MergeSteps merges imported steps with main workflow steps
// Imported steps are prepended to the corresponding position
func MergeSteps(main, imported *WorkflowSteps) *WorkflowSteps {
	if main == nil && imported == nil {
		return nil
	}
	if main == nil {
		return imported
	}
	if imported == nil {
		return main
	}

	merged := &WorkflowSteps{
		Pre:       append(imported.Pre, main.Pre...),
		PreAgent:  append(imported.PreAgent, main.PreAgent...),
		PostAgent: append(imported.PostAgent, main.PostAgent...),
		Post:      append(imported.Post, main.Post...),
	}

	return merged
}

// renderStepsAtPosition renders steps at a specific position with proper indentation
func renderStepsAtPosition(yaml *strings.Builder, steps []Step) {
	if len(steps) == 0 {
		return
	}

	for _, step := range steps {
		// Marshal step to YAML
		stepYAML, err := marshalStep(step)
		if err != nil {
			continue // Skip steps that fail to marshal
		}

		// Split into lines and add proper indentation
		lines := strings.Split(stepYAML, "\n")
		for i, line := range lines {
			if strings.TrimSpace(line) == "" {
				continue // Skip empty lines
			}
			if i == 0 {
				// First line gets the list marker: "      - name: ..."
				yaml.WriteString("      - " + line + "\n")
			} else {
				// Subsequent lines get extra indentation: "        run: ..."
				yaml.WriteString("        " + line + "\n")
			}
		}
	}
}

// marshalStep marshals a step to YAML string
func marshalStep(step Step) (string, error) {
	yamlBytes, err := yaml.Marshal(step)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(yamlBytes)), nil
}

// IsEmpty returns true if there are no steps in any position

func (ws *WorkflowSteps) IsEmpty() bool {
	return ws == nil || (len(ws.Pre) == 0 && len(ws.PreAgent) == 0 && len(ws.PostAgent) == 0 && len(ws.Post) == 0)
}

// ToYAML converts steps to YAML string for a specific position
func stepsToYAML(steps []Step) (string, error) {
	if len(steps) == 0 {
		return "", nil
	}

	yamlBytes, err := yaml.Marshal(steps)
	if err != nil {
		return "", err
	}

	return string(yamlBytes), nil
}
