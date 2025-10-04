package workflow

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/goccy/go-yaml"
)

// Step represents a GitHub Actions workflow step with YAML serialization support
type Step struct {
	Name string `yaml:"name,omitempty"`
	ID   string `yaml:"id,omitempty"`
	If   string `yaml:"if,omitempty"`
	Run  string `yaml:"run,omitempty"`
	Uses string `yaml:"uses,omitempty"`
	Env  map[string]string `yaml:"env,omitempty"`
	With map[string]any `yaml:"with,omitempty"`
	// Additional fields can be stored in the Extra map
	Extra map[string]any `yaml:",inline"`
}

// NewStep creates a new Step with the given name
func NewStep(name string) *Step {
	return &Step{
		Name: name,
	}
}

// NewStepWithRun creates a new Step with name and run command
func NewStepWithRun(name, run string) *Step {
	return &Step{
		Name: name,
		Run:  run,
	}
}

// NewStepWithUses creates a new Step with name and uses action
func NewStepWithUses(name, uses string) *Step {
	return &Step{
		Name: name,
		Uses: uses,
	}
}

// NewGitHubScriptStep creates a new Step that uses actions/github-script@v8
// with the provided JavaScript code and environment variables
func NewGitHubScriptStep(name string, script string, env map[string]string) *Step {
	step := &Step{
		Name: name,
		Uses: "actions/github-script@v8",
		Env:  env,
		With: map[string]any{
			"script": script,
		},
	}
	return step
}

// SetID sets the step ID
func (s *Step) SetID(id string) *Step {
	s.ID = id
	return s
}

// SetIf sets the step conditional expression
func (s *Step) SetIf(ifExpr string) *Step {
	s.If = ifExpr
	return s
}

// AddEnv adds an environment variable to the step
func (s *Step) AddEnv(key, value string) *Step {
	if s.Env == nil {
		s.Env = make(map[string]string)
	}
	s.Env[key] = value
	return s
}

// AddEnvMap adds multiple environment variables from a map
func (s *Step) AddEnvMap(env map[string]string) *Step {
	if s.Env == nil {
		s.Env = make(map[string]string)
	}
	for k, v := range env {
		s.Env[k] = v
	}
	return s
}

// AddWith adds a with parameter to the step
func (s *Step) AddWith(key string, value any) *Step {
	if s.With == nil {
		s.With = make(map[string]any)
	}
	s.With[key] = value
	return s
}

// SetGitHubToken adds the github-token with parameter
func (s *Step) SetGitHubToken(token string) *Step {
	if s.With == nil {
		s.With = make(map[string]any)
	}
	s.With["github-token"] = token
	return s
}

// ToMap converts the Step to a map[string]any for serialization
func (s *Step) ToMap() map[string]any {
	result := make(map[string]any)
	
	if s.Name != "" {
		result["name"] = s.Name
	}
	if s.ID != "" {
		result["id"] = s.ID
	}
	if s.If != "" {
		result["if"] = s.If
	}
	if s.Run != "" {
		result["run"] = s.Run
	}
	if s.Uses != "" {
		result["uses"] = s.Uses
	}
	if len(s.Env) > 0 {
		result["env"] = s.Env
	}
	if len(s.With) > 0 {
		result["with"] = s.With
	}
	
	// Add any extra fields
	for k, v := range s.Extra {
		result[k] = v
	}
	
	return result
}

// ToYAML converts the Step to YAML string with proper indentation for GitHub Actions (6-space indent)
func (s *Step) ToYAML() (string, error) {
	return ConvertStepToYAML(s.ToMap())
}

// WriteStepsToYAML writes one or more steps to a writer with proper indentation
func WriteStepsToYAML(w io.Writer, steps ...*Step) error {
	for _, step := range steps {
		yaml, err := step.ToYAML()
		if err != nil {
			return fmt.Errorf("failed to convert step to YAML: %w", err)
		}
		if _, err := w.Write([]byte(yaml)); err != nil {
			return fmt.Errorf("failed to write step YAML: %w", err)
		}
	}
	return nil
}

// WriteStepsToString writes one or more steps to a string with proper indentation
func WriteStepsToString(steps ...*Step) (string, error) {
	var builder strings.Builder
	if err := WriteStepsToYAML(&builder, steps...); err != nil {
		return "", err
	}
	return builder.String(), nil
}

// StepsToYAMLLines converts steps to GitHubActionStep format ([]string lines)
// This is useful for compatibility with existing code
func StepsToYAMLLines(steps ...*Step) ([]GitHubActionStep, error) {
	var result []GitHubActionStep
	
	for _, step := range steps {
		yaml, err := step.ToYAML()
		if err != nil {
			return nil, fmt.Errorf("failed to convert step to YAML: %w", err)
		}
		
		// Split YAML into lines for GitHubActionStep format
		lines := strings.Split(strings.TrimRight(yaml, "\n"), "\n")
		
		// Remove empty lines at the end
		for len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "" {
			lines = lines[:len(lines)-1]
		}
		
		result = append(result, GitHubActionStep(lines))
	}
	
	return result, nil
}

// StepToYAMLLines converts a single step to GitHubActionStep format
func StepToYAMLLines(step *Step) (GitHubActionStep, error) {
	yaml, err := step.ToYAML()
	if err != nil {
		return nil, fmt.Errorf("failed to convert step to YAML: %w", err)
	}
	
	// Split YAML into lines for GitHubActionStep format
	lines := strings.Split(strings.TrimRight(yaml, "\n"), "\n")
	
	// Remove empty lines at the end
	for len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "" {
		lines = lines[:len(lines)-1]
	}
	
	return GitHubActionStep(lines), nil
}

// convertStepMapToYAML is a helper that converts a step map to YAML with proper field ordering
// This is used internally and by ConvertStepToYAML
func convertStepMapToYAML(stepMap map[string]any) (string, error) {
	// Define the priority field order: name, id, if, run, uses, env, with, ...
	priorityFields := []string{"name", "id", "if", "run", "uses", "env", "with"}

	// Create an ordered map using yaml.MapSlice to maintain field order
	var step yaml.MapSlice

	// First, add priority fields in the specified order
	for _, fieldName := range priorityFields {
		if value, exists := stepMap[fieldName]; exists {
			step = append(step, yaml.MapItem{Key: fieldName, Value: value})
		}
	}

	// Then add remaining fields in alphabetical order
	var remainingKeys []string
	for key := range stepMap {
		// Skip if it's already been added as a priority field
		isPriority := false
		for _, priorityField := range priorityFields {
			if key == priorityField {
				isPriority = true
				break
			}
		}
		if !isPriority {
			remainingKeys = append(remainingKeys, key)
		}
	}

	// Sort remaining keys alphabetically
	sort.Strings(remainingKeys)

	// Add remaining fields to the ordered map
	for _, key := range remainingKeys {
		step = append(step, yaml.MapItem{Key: key, Value: stepMap[key]})
	}

	// Serialize the step using YAML package with proper options for multiline strings
	yamlBytes, err := yaml.MarshalWithOptions([]yaml.MapSlice{step},
		yaml.Indent(2),                        // Use 2-space indentation
		yaml.UseLiteralStyleIfMultiline(true), // Use literal block scalars for multiline strings
	)
	if err != nil {
		return "", fmt.Errorf("failed to marshal step to YAML: %w", err)
	}

	// Convert to string and adjust base indentation to match GitHub Actions format
	yamlStr := string(yamlBytes)

	// Add 6 spaces to the beginning of each line to match GitHub Actions step indentation
	lines := strings.Split(strings.TrimSpace(yamlStr), "\n")
	var result strings.Builder

	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			result.WriteString("\n")
		} else {
			result.WriteString("      " + line + "\n")
		}
	}

	return result.String(), nil
}

// BuildGitHubScriptStepLines creates a GitHub Actions step that uses actions/github-script@v8
// and returns it as []string lines for compatibility with existing code.
// This helper simplifies the common pattern of creating github-script steps.
func BuildGitHubScriptStepLines(name, id string, script string, env map[string]string, withParams map[string]string) []string {
	var lines []string
	
	// Add step header
	lines = append(lines, fmt.Sprintf("      - name: %s\n", name))
	if id != "" {
		lines = append(lines, fmt.Sprintf("        id: %s\n", id))
	}
	lines = append(lines, "        uses: actions/github-script@v8\n")
	
	// Add environment variables if provided
	if len(env) > 0 {
		lines = append(lines, "        env:\n")
		
		// Sort environment keys for consistent output
		envKeys := make([]string, 0, len(env))
		for key := range env {
			envKeys = append(envKeys, key)
		}
		sort.Strings(envKeys)
		
		for _, key := range envKeys {
			value := env[key]
			// Properly quote values that need it
			lines = append(lines, fmt.Sprintf("          %s: %s\n", key, value))
		}
	}
	
	// Add with parameters
	lines = append(lines, "        with:\n")
	
	// Add github-token first if present
	if token, hasToken := withParams["github-token"]; hasToken {
		lines = append(lines, fmt.Sprintf("          github-token: %s\n", token))
	}
	
	// Add other with parameters (sorted, excluding github-token)
	if len(withParams) > 0 {
		withKeys := make([]string, 0, len(withParams))
		for key := range withParams {
			if key != "github-token" {
				withKeys = append(withKeys, key)
			}
		}
		sort.Strings(withKeys)
		
		for _, key := range withKeys {
			value := withParams[key]
			lines = append(lines, fmt.Sprintf("          %s: %s\n", key, value))
		}
	}
	
	// Add the script
	lines = append(lines, "          script: |\n")
	
	return lines
}

