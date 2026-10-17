package workflow

import (
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/github/gh-aw/pkg/workflow/pipeline"
	"github.com/goccy/go-yaml"
)

// resolvePipeline bridges legacy jobs to the typed graph. Legacy YAML remains
// untouched; only typed contributions are lowered, using the existing renderer.
// Commit changes only after every job has resolved and lowered successfully.
func (jm *JobManager) resolvePipeline() error {
	ir := &pipeline.Pipeline{}
	for _, name := range jm.jobOrder {
		job, err := jobToPipeline(jm.jobs[name])
		if err != nil {
			return err
		}
		ir.Jobs = append(ir.Jobs, job)
	}
	resolved, err := pipeline.Resolve(ir)
	if err != nil {
		return err
	}
	lowered := make(map[string]*Job, len(resolved.Jobs))
	for _, irJob := range resolved.Jobs {
		name := string(irJob.ID)
		original := jm.jobs[name]
		job := *original
		job.Needs = slices.Clone(original.Needs)
		job.Outputs = maps.Clone(original.Outputs)
		for _, dep := range irJob.Needs {
			if !slices.Contains(job.Needs, string(dep)) {
				job.Needs = append(job.Needs, string(dep))
			}
		}
		for key, value := range irJob.Outputs {
			if _, legacy := original.Outputs[string(key)]; legacy {
				continue
			}
			rendered, err := renderPipelineScalar(value, irJob.ID, resolved)
			if err != nil {
				return err
			}
			if job.Outputs == nil {
				job.Outputs = make(map[string]string)
			}
			job.Outputs[string(key)] = rendered
		}
		if typed := original.Pipeline; typed != nil {
			if err := lowerPipelineContribution(&job, typed, irJob.ID, resolved); err != nil {
				return fmt.Errorf("lowering pipeline job %q: %w", name, err)
			}
			job.Pipeline = nil
		}
		if err := validateJobDefinition(&job); err != nil {
			return err
		}
		lowered[name] = &job
	}
	jm.jobs = lowered
	return nil
}

func jobToPipeline(original *Job) (*pipeline.Job, error) {
	name := original.Name
	job := &pipeline.Job{ID: pipeline.JobID(name), Outputs: make(map[pipeline.OutputName]pipeline.Expression)}
	for _, dep := range original.Needs {
		job.Needs = append(job.Needs, pipeline.JobID(dep))
	}
	for key, value := range original.Outputs {
		// Existing output bindings are opaque; their declarations still let
		// typed consumers validate and depend on them.
		job.Outputs[pipeline.OutputName(key)] = pipeline.Literal(value)
	}
	typed := original.Pipeline
	if typed == nil {
		return job, nil
	}
	if typed.ID != "" && typed.ID != job.ID {
		return nil, fmt.Errorf("typed pipeline job id %q does not match job %q", typed.ID, name)
	}
	if typed.Name != "" || typed.RunsOn != nil || len(typed.Fields) > 0 {
		return nil, fmt.Errorf("job %q must set name, runs-on, and native fields on the enclosing job, not its typed contribution", name)
	}
	job.Condition = typed.Condition
	job.Env = typed.Env
	job.Needs = append(job.Needs, typed.Needs...)
	for key, value := range typed.Outputs {
		if _, exists := job.Outputs[key]; exists {
			return nil, fmt.Errorf("job %q has both legacy and typed output %q", name, key)
		}
		job.Outputs[key] = value
	}
	if len(original.Steps) > 0 {
		var steps []map[string]any
		if err := yaml.Unmarshal([]byte(strings.Join(original.Steps, "")), &steps); err != nil {
			return nil, fmt.Errorf("indexing legacy steps in job %q: %w", name, err)
		}
		for _, fields := range steps {
			job.Steps = append(job.Steps, pipeline.RawStep{Fields: fields})
		}
	}
	job.Steps = append(job.Steps, typed.Steps...)
	return job, nil
}

func lowerPipelineContribution(job *Job, typed *pipeline.Job, id pipeline.JobID, resolved *pipeline.Pipeline) error {
	if job.Uses != "" && (len(typed.Steps) > 0 || len(typed.Env) > 0) {
		return errors.New("reusable workflow job cannot declare typed steps or env")
	}
	if typed.Condition != nil {
		if job.If != "" {
			return errors.New("cannot set both a legacy and typed job condition")
		}
		rendered, err := pipeline.RenderExpression(typed.Condition, id, resolved, false)
		if err != nil {
			return err
		}
		job.If = "${{ " + rendered + " }}"
	}
	job.Env = maps.Clone(job.Env)
	for key, value := range typed.Env {
		if _, exists := job.Env[key]; exists {
			return fmt.Errorf("cannot set both a legacy and typed environment value %q", key)
		}
		rendered, err := renderPipelineScalar(value, id, resolved)
		if err != nil {
			return err
		}
		if job.Env == nil {
			job.Env = make(map[string]string)
		}
		job.Env[key] = rendered
	}
	job.Steps = slices.Clone(job.Steps)
	for _, step := range typed.Steps {
		fields, err := pipeline.LowerStep(step, id, resolved)
		if err != nil {
			return err
		}
		rendered, err := ConvertStepToYAML(fields)
		if err != nil {
			return err
		}
		job.Steps = append(job.Steps, rendered)
	}
	return nil
}

func renderPipelineScalar(value pipeline.Expression, id pipeline.JobID, resolved *pipeline.Pipeline) (string, error) {
	rendered, err := pipeline.RenderExpression(value, id, resolved, true)
	if err != nil {
		return "", err
	}
	encoded, err := json.Marshal(rendered)
	if err != nil {
		return "", err
	}
	return string(encoded), nil
}
