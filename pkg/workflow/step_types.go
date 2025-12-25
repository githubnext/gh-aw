package workflow

import (
	"fmt"

	"github.com/githubnext/gh-aw/pkg/logger"
	"github.com/goccy/go-yaml"
)

var stepTypesLog = logger.New("workflow:step_types")

// WorkflowStep represents a single step in a GitHub Actions workflow job
// This struct provides type safety and compile-time validation for step configurations
type WorkflowStep struct {
	Name             string            `yaml:"name,omitempty"`
	ID               string            `yaml:"id,omitempty"`
	If               string            `yaml:"if,omitempty"`
	Uses             string            `yaml:"uses,omitempty"`
	Run              string            `yaml:"run,omitempty"`
	WorkingDirectory string            `yaml:"working-directory,omitempty"`
	Shell            string            `yaml:"shell,omitempty"`
	With             map[string]any    `yaml:"with,omitempty"`
	Env              map[string]string `yaml:"env,omitempty"`
	ContinueOnError  any               `yaml:"continue-on-error,omitempty"` // Can be bool or string expression
	TimeoutMinutes   int               `yaml:"timeout-minutes,omitempty"`
}

// IsUsesStep returns true if this step uses an action (has a "uses" field)
func (s *WorkflowStep) IsUsesStep() bool {
	return s.Uses != ""
}

// IsRunStep returns true if this step runs a command (has a "run" field)
func (s *WorkflowStep) IsRunStep() bool {
	return s.Run != ""
}

// ToMap converts a WorkflowStep to a map[string]any for YAML generation
// This is used when generating the final workflow YAML output
func (s *WorkflowStep) ToMap() map[string]any {
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
	if s.Uses != "" {
		result["uses"] = s.Uses
	}
	if s.Run != "" {
		result["run"] = s.Run
	}
	if s.WorkingDirectory != "" {
		result["working-directory"] = s.WorkingDirectory
	}
	if s.Shell != "" {
		result["shell"] = s.Shell
	}
	if len(s.With) > 0 {
		result["with"] = s.With
	}
	if len(s.Env) > 0 {
		result["env"] = s.Env
	}
	if s.ContinueOnError != nil {
		result["continue-on-error"] = s.ContinueOnError
	}
	if s.TimeoutMinutes > 0 {
		result["timeout-minutes"] = s.TimeoutMinutes
	}

	return result
}

// MapToStep converts a map[string]any to a WorkflowStep
// This is the inverse of ToMap and is used when parsing step configurations
func MapToStep(stepMap map[string]any) (*WorkflowStep, error) {
	stepTypesLog.Printf("Converting map to workflow step: map_keys=%d", len(stepMap))
	if stepMap == nil {
		return nil, fmt.Errorf("step map is nil")
	}

	step := &WorkflowStep{}

	if name, ok := stepMap["name"].(string); ok {
		step.Name = name
	}
	if id, ok := stepMap["id"].(string); ok {
		step.ID = id
	}
	if ifCond, ok := stepMap["if"].(string); ok {
		step.If = ifCond
	}
	if uses, ok := stepMap["uses"].(string); ok {
		step.Uses = uses
	}
	if run, ok := stepMap["run"].(string); ok {
		step.Run = run
	}
	if workingDir, ok := stepMap["working-directory"].(string); ok {
		step.WorkingDirectory = workingDir
	}
	if shell, ok := stepMap["shell"].(string); ok {
		step.Shell = shell
	}
	if with, ok := stepMap["with"].(map[string]any); ok {
		step.With = with
	}
	if env, ok := stepMap["env"].(map[string]any); ok {
		// Convert map[string]any to map[string]string
		step.Env = make(map[string]string)
		for k, v := range env {
			if strVal, ok := v.(string); ok {
				step.Env[k] = strVal
			}
		}
	}
	if continueOnError, ok := stepMap["continue-on-error"]; ok {
		// Preserve the original type (bool or string)
		step.ContinueOnError = continueOnError
	}
	if timeoutMinutes, ok := stepMap["timeout-minutes"].(int); ok {
		step.TimeoutMinutes = timeoutMinutes
	}

	stepType := "unknown"
	if step.Uses != "" {
		stepType = "uses"
	} else if step.Run != "" {
		stepType = "run"
	}
	stepTypesLog.Printf("Successfully converted step: type=%s, name=%s", stepType, step.Name)
	return step, nil
}

// Clone creates a deep copy of the WorkflowStep
func (s *WorkflowStep) Clone() *WorkflowStep {
	clone := &WorkflowStep{
		Name:             s.Name,
		ID:               s.ID,
		If:               s.If,
		Uses:             s.Uses,
		Run:              s.Run,
		WorkingDirectory: s.WorkingDirectory,
		Shell:            s.Shell,
		ContinueOnError:  s.ContinueOnError,
		TimeoutMinutes:   s.TimeoutMinutes,
	}

	if s.With != nil {
		clone.With = make(map[string]any, len(s.With))
		for k, v := range s.With {
			clone.With[k] = v
		}
	}

	if s.Env != nil {
		clone.Env = make(map[string]string, len(s.Env))
		for k, v := range s.Env {
			clone.Env[k] = v
		}
	}

	return clone
}

// ToYAML converts the WorkflowStep to YAML string
func (s *WorkflowStep) ToYAML() (string, error) {
	stepTypesLog.Printf("Converting step to YAML: name=%s", s.Name)
	stepMap := s.ToMap()
	yamlBytes, err := yaml.Marshal(stepMap)
	if err != nil {
		stepTypesLog.Printf("Failed to marshal step to YAML: %v", err)
		return "", fmt.Errorf("failed to marshal step to YAML: %w", err)
	}
	stepTypesLog.Printf("Successfully converted step to YAML: size=%d bytes", len(yamlBytes))
	return string(yamlBytes), nil
}

// WorkflowJob represents a GitHub Actions job configuration from frontmatter
// This is different from the internal Job type used by the compiler
type WorkflowJob struct {
	Name            string            `yaml:"name,omitempty"`
	RunsOn          any               `yaml:"runs-on,omitempty"`     // Can be string or array
	Needs           []string          `yaml:"needs,omitempty"`       // Job dependencies
	If              string            `yaml:"if,omitempty"`          // Conditional expression
	Steps           []WorkflowStep    `yaml:"steps,omitempty"`       // Job steps
	Permissions     map[string]string `yaml:"permissions,omitempty"` // Job-level permissions
	Environment     any               `yaml:"environment,omitempty"` // Can be string or map
	Concurrency     any               `yaml:"concurrency,omitempty"` // Can be string or map
	TimeoutMinutes  int               `yaml:"timeout-minutes,omitempty"`
	Container       any               `yaml:"container,omitempty"` // Can be string or map
	Services        map[string]any    `yaml:"services,omitempty"`  // Service containers
	Env             map[string]string `yaml:"env,omitempty"`       // Environment variables
	Outputs         map[string]string `yaml:"outputs,omitempty"`   // Job outputs
	Strategy        map[string]any    `yaml:"strategy,omitempty"`  // Matrix strategy
	ContinueOnError bool              `yaml:"continue-on-error,omitempty"`

	// Reusable workflow fields
	Uses    string         `yaml:"uses,omitempty"`    // Path to reusable workflow
	With    map[string]any `yaml:"with,omitempty"`    // Inputs for reusable workflow
	Secrets any            `yaml:"secrets,omitempty"` // Can be "inherit" or map[string]string
}

// ToMap converts a WorkflowJob to a map[string]any for YAML generation
func (j *WorkflowJob) ToMap() map[string]any {
	result := make(map[string]any)

	if j.Name != "" {
		result["name"] = j.Name
	}
	if j.RunsOn != nil {
		result["runs-on"] = j.RunsOn
	}
	if len(j.Needs) > 0 {
		result["needs"] = j.Needs
	}
	if j.If != "" {
		result["if"] = j.If
	}
	if len(j.Steps) > 0 {
		steps := make([]map[string]any, len(j.Steps))
		for i, step := range j.Steps {
			steps[i] = step.ToMap()
		}
		result["steps"] = steps
	}
	if len(j.Permissions) > 0 {
		result["permissions"] = j.Permissions
	}
	if j.Environment != nil {
		result["environment"] = j.Environment
	}
	if j.Concurrency != nil {
		result["concurrency"] = j.Concurrency
	}
	if j.TimeoutMinutes > 0 {
		result["timeout-minutes"] = j.TimeoutMinutes
	}
	if j.Container != nil {
		result["container"] = j.Container
	}
	if len(j.Services) > 0 {
		result["services"] = j.Services
	}
	if len(j.Env) > 0 {
		result["env"] = j.Env
	}
	if len(j.Outputs) > 0 {
		result["outputs"] = j.Outputs
	}
	if len(j.Strategy) > 0 {
		result["strategy"] = j.Strategy
	}
	if j.ContinueOnError {
		result["continue-on-error"] = j.ContinueOnError
	}
	if j.Uses != "" {
		result["uses"] = j.Uses
	}
	if len(j.With) > 0 {
		result["with"] = j.With
	}
	if j.Secrets != nil {
		result["secrets"] = j.Secrets
	}

	return result
}

// MapToJob converts a map[string]any to a WorkflowJob
func MapToJob(jobMap map[string]any) (*WorkflowJob, error) {
	stepTypesLog.Printf("Converting map to workflow job: map_keys=%d", len(jobMap))
	if jobMap == nil {
		return nil, fmt.Errorf("job map is nil")
	}

	job := &WorkflowJob{}

	if name, ok := jobMap["name"].(string); ok {
		job.Name = name
	}
	if runsOn, ok := jobMap["runs-on"]; ok {
		job.RunsOn = runsOn
	}
	if needs, ok := jobMap["needs"]; ok {
		switch v := needs.(type) {
		case []any:
			for _, need := range v {
				if needStr, ok := need.(string); ok {
					job.Needs = append(job.Needs, needStr)
				}
			}
		case []string:
			job.Needs = v
		case string:
			job.Needs = []string{v}
		}
	}
	if ifCond, ok := jobMap["if"].(string); ok {
		job.If = ifCond
	}
	if steps, ok := jobMap["steps"].([]any); ok {
		for _, stepAny := range steps {
			if stepMap, ok := stepAny.(map[string]any); ok {
				step, err := MapToStep(stepMap)
				if err != nil {
					return nil, fmt.Errorf("failed to convert step: %w", err)
				}
				job.Steps = append(job.Steps, *step)
			}
		}
	}
	if permissions, ok := jobMap["permissions"].(map[string]any); ok {
		job.Permissions = make(map[string]string)
		for k, v := range permissions {
			if strVal, ok := v.(string); ok {
				job.Permissions[k] = strVal
			}
		}
	}
	if environment, ok := jobMap["environment"]; ok {
		job.Environment = environment
	}
	if concurrency, ok := jobMap["concurrency"]; ok {
		job.Concurrency = concurrency
	}
	if timeoutMinutes, ok := jobMap["timeout-minutes"].(int); ok {
		job.TimeoutMinutes = timeoutMinutes
	}
	if container, ok := jobMap["container"]; ok {
		job.Container = container
	}
	if services, ok := jobMap["services"].(map[string]any); ok {
		job.Services = services
	}
	if env, ok := jobMap["env"].(map[string]any); ok {
		job.Env = make(map[string]string)
		for k, v := range env {
			if strVal, ok := v.(string); ok {
				job.Env[k] = strVal
			}
		}
	}
	if outputs, ok := jobMap["outputs"].(map[string]any); ok {
		job.Outputs = make(map[string]string)
		for k, v := range outputs {
			if strVal, ok := v.(string); ok {
				job.Outputs[k] = strVal
			}
		}
	}
	if strategy, ok := jobMap["strategy"].(map[string]any); ok {
		job.Strategy = strategy
	}
	if continueOnError, ok := jobMap["continue-on-error"].(bool); ok {
		job.ContinueOnError = continueOnError
	}
	if uses, ok := jobMap["uses"].(string); ok {
		job.Uses = uses
	}
	if with, ok := jobMap["with"].(map[string]any); ok {
		job.With = with
	}
	if secrets, ok := jobMap["secrets"]; ok {
		job.Secrets = secrets
	}

	stepTypesLog.Printf("Successfully converted job: name=%s, steps=%d", job.Name, len(job.Steps))
	return job, nil
}

// StepsToAny converts []WorkflowStep to []any for compatibility with existing code
func StepsToAny(steps []WorkflowStep) []any {
	result := make([]any, len(steps))
	for i, step := range steps {
		result[i] = step.ToMap()
	}
	return result
}

// StepsFromAny converts []any to []WorkflowStep
func StepsFromAny(stepsAny []any) ([]WorkflowStep, error) {
	steps := make([]WorkflowStep, 0, len(stepsAny))
	for i, stepAny := range stepsAny {
		stepMap, ok := stepAny.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("step %d is not a map[string]any", i)
		}
		step, err := MapToStep(stepMap)
		if err != nil {
			return nil, fmt.Errorf("failed to convert step %d: %w", i, err)
		}
		steps = append(steps, *step)
	}
	return steps, nil
}
