package pipeline

import (
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/goccy/go-yaml"
)

// Lower resolves the graph and creates a deterministic YAML value tree.
func Lower(input *Pipeline) (yaml.MapSlice, error) {
	resolved, err := Resolve(input)
	if err != nil {
		return nil, err
	}
	root := yaml.MapSlice{{Key: "name", Value: resolved.Name}}
	for _, key := range sortedKeys(resolved.Envelope) {
		if key == "name" || key == "jobs" {
			return nil, fmt.Errorf("pipeline envelope cannot override typed field %q", key)
		}
		root = append(root, yaml.MapItem{Key: key, Value: resolved.Envelope[key]})
	}
	jobs := slices.Clone(resolved.Jobs)
	slices.SortFunc(jobs, func(a, b *Job) int {
		if a.ID < b.ID {
			return -1
		}
		if a.ID > b.ID {
			return 1
		}
		return 0
	})
	jobMap := make(yaml.MapSlice, 0, len(jobs))
	for _, job := range jobs {
		fields, err := lowerJob(job, resolved)
		if err != nil {
			return nil, fmt.Errorf("lowering pipeline job %q: %w", job.ID, err)
		}
		jobMap = append(jobMap, yaml.MapItem{Key: string(job.ID), Value: fields})
	}
	return append(root, yaml.MapItem{Key: "jobs", Value: jobMap}), nil
}

func lowerJob(job *Job, resolved *Pipeline) (map[string]any, error) {
	fields, err := nativeJobFields(job)
	if err != nil {
		return nil, err
	}
	if job.Name != "" {
		fields["name"] = job.Name
	}
	if job.RunsOn != nil {
		fields["runs-on"] = job.RunsOn
	}
	if len(job.Needs) > 0 {
		needs := make([]string, 0, len(job.Needs))
		for _, id := range job.Needs {
			needs = append(needs, string(id))
		}
		fields["needs"] = needs
	}
	if job.Condition != nil {
		value, err := RenderExpression(job.Condition, job.ID, resolved, true)
		if err != nil {
			return nil, err
		}
		fields["if"] = value
	}
	if len(job.Env) > 0 {
		env, err := lowerValues(job.Env, job.ID, resolved)
		if err != nil {
			return nil, err
		}
		fields["env"] = env
	}
	if len(job.Outputs) > 0 {
		outputs := make(map[string]any, len(job.Outputs))
		for _, name := range sortedOutputNames(job.Outputs) {
			value, err := RenderExpression(job.Outputs[name], job.ID, resolved, true)
			if err != nil {
				return nil, err
			}
			outputs[string(name)] = value
		}
		fields["outputs"] = outputs
	}
	if len(job.Steps) > 0 {
		steps := make([]map[string]any, 0, len(job.Steps))
		for _, step := range job.Steps {
			value, err := LowerStep(step, job.ID, resolved)
			if err != nil {
				return nil, err
			}
			steps = append(steps, value)
		}
		fields["steps"] = steps
	}
	return fields, nil
}

func nativeJobFields(job *Job) (map[string]any, error) {
	fields := maps.Clone(job.Fields)
	if fields == nil {
		fields = make(map[string]any)
	}
	for _, key := range []string{"name", "runs-on", "needs", "if", "env", "steps", "outputs"} {
		if _, exists := fields[key]; exists {
			return nil, fmt.Errorf("native job fields cannot override typed field %q", key)
		}
	}
	if uses, exists := fields["uses"]; exists {
		if ref, ok := uses.(string); !ok || ref == "" {
			return nil, errors.New("reusable workflow uses must be a nonempty string")
		}
		if len(job.Steps) > 0 || job.RunsOn != nil || len(job.Env) > 0 {
			return nil, errors.New("reusable workflow job cannot declare steps, runs-on, or env")
		}
	}
	return fields, nil
}

// LowerStep is also used by the legacy compiler bridge, retaining its existing
// field ordering, action-pin comments, and runner-specific rendering.
func LowerStep(step Step, consumer JobID, resolved *Pipeline) (map[string]any, error) {
	opts, err := options(step)
	if err != nil {
		return nil, err
	}
	if opts.TimeoutMinutes < 0 {
		return nil, errors.New("step timeout-minutes cannot be negative")
	}
	fields := make(map[string]any)
	switch value := step.(type) {
	case RunStep:
		if value.Script == "" {
			return nil, errors.New("run step requires a nonempty script")
		}
		if strings.Contains(value.Script, "${{") {
			return nil, errors.New("typed run steps must pass Actions expressions through env instead of interpolating them into scripts")
		}
		fields["run"] = value.Script
		if value.Shell != "" {
			fields["shell"] = value.Shell
		}
		if value.WorkingDirectory != "" {
			fields["working-directory"] = value.WorkingDirectory
		}
	case ActionStep:
		if value.Uses == "" {
			return nil, errors.New("action step requires a nonempty uses reference")
		}
		fields["uses"] = value.Uses
		if len(value.With) > 0 {
			with, err := lowerValues(value.With, consumer, resolved)
			if err != nil {
				return nil, err
			}
			fields["with"] = with
		}
	case RawStep:
		if len(value.Fields) == 0 {
			return nil, errors.New("raw pipeline step cannot be empty")
		}
		return maps.Clone(value.Fields), nil
	}
	if err := lowerStepOptions(fields, opts, consumer, resolved); err != nil {
		return nil, err
	}
	return fields, nil
}

func lowerStepOptions(fields map[string]any, opts StepOptions, consumer JobID, resolved *Pipeline) error {
	if opts.ID != "" {
		fields["id"] = string(opts.ID)
	}
	if opts.Name != "" {
		fields["name"] = opts.Name
	}
	if opts.Condition != nil {
		value, err := RenderExpression(opts.Condition, consumer, resolved, true)
		if err != nil {
			return err
		}
		fields["if"] = value
	}
	if len(opts.Env) > 0 {
		env, err := lowerValues(opts.Env, consumer, resolved)
		if err != nil {
			return err
		}
		fields["env"] = env
	}
	if opts.ContinueOnError != nil {
		fields["continue-on-error"] = *opts.ContinueOnError
	}
	if opts.TimeoutMinutes > 0 {
		fields["timeout-minutes"] = opts.TimeoutMinutes
	}
	return nil
}

func lowerValues(values map[string]Expression, consumer JobID, resolved *Pipeline) (map[string]any, error) {
	result := make(map[string]any, len(values))
	for _, key := range sortedKeys(values) {
		value, err := RenderExpression(values[key], consumer, resolved, true)
		if err != nil {
			return nil, fmt.Errorf("pipeline value %q: %w", key, err)
		}
		result[key] = value
	}
	return result, nil
}

func Emit(input *Pipeline) (string, error) {
	value, err := Lower(input)
	if err != nil {
		return "", err
	}
	content, err := yaml.Marshal(value)
	if err != nil {
		return "", fmt.Errorf("serializing pipeline YAML: %w", err)
	}
	return string(content), nil
}
