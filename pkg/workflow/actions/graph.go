package actions

import (
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"
)

var errInvalidStep = errors.New("workflow step cannot be nil or an unsupported type")

type producer struct {
	index   int
	outputs map[OutputName]bool
}

type resolver struct {
	jobs  map[JobID]*Job
	steps map[JobID]map[StepID]producer
}

// Resolve returns a copy with output-derived needs and job-output bindings.
// It validates explicit edges too, and leaves the input unchanged on failure.
func Resolve(input *Workflow) (*Workflow, error) {
	result, graph, err := indexWorkflow(input)
	if err != nil {
		return nil, err
	}
	for _, job := range result.Jobs {
		resolve := func(ref OutputRef, position int) (string, error) {
			return graph.resolveRef(job, ref, position)
		}
		if err := walkJob(job, resolve); err != nil {
			return nil, fmt.Errorf("workflow job %q: %w", job.ID, err)
		}
	}
	for _, job := range result.Jobs {
		slices.Sort(job.Needs)
		job.Needs = slices.Compact(job.Needs)
		for _, dep := range job.Needs {
			if _, exists := graph.jobs[dep]; !exists {
				return nil, fmt.Errorf("job '%s' depends on non-existent job '%s'", job.ID, dep)
			}
		}
	}
	if err := detectCycles(result.Jobs, graph.jobs); err != nil {
		return nil, err
	}
	return result, nil
}

func indexWorkflow(input *Workflow) (*Workflow, *resolver, error) {
	if input == nil {
		return nil, nil, errors.New("workflow cannot be nil")
	}
	if err := walkWorkflowEnv(input.Env); err != nil {
		return nil, nil, err
	}
	result := *input
	result.Jobs = make([]*Job, 0, len(input.Jobs))
	jobs := make(map[JobID]*Job, len(input.Jobs))
	steps := make(map[JobID]map[StepID]producer, len(input.Jobs))
	for _, original := range input.Jobs {
		if original == nil {
			return nil, nil, errors.New("workflow job cannot be nil")
		}
		if err := validateIdentifier("job id", string(original.ID)); err != nil {
			return nil, nil, err
		}
		if _, exists := jobs[original.ID]; exists {
			return nil, nil, fmt.Errorf("duplicate workflow job id %q", original.ID)
		}
		job := *original
		if err := validateCallOutputs(&job); err != nil {
			return nil, nil, err
		}
		job.Needs = slices.Clone(original.Needs)
		job.Outputs = maps.Clone(original.Outputs)
		if job.Outputs == nil {
			job.Outputs = make(map[OutputName]Expression)
		}
		jobs[job.ID] = &job
		result.Jobs = append(result.Jobs, &job)
		index, err := indexSteps(&job)
		if err != nil {
			return nil, nil, err
		}
		steps[job.ID] = index
	}
	return &result, &resolver{jobs: jobs, steps: steps}, nil
}

func (graph *resolver) resolveRef(job *Job, ref OutputRef, position int) (string, error) {
	target, exists := graph.jobs[ref.Job]
	if !exists {
		return "", fmt.Errorf("job %q references unknown producer job %q", job.ID, ref.Job)
	}
	if err := validateIdentifier("output name", string(ref.Name)); err != nil {
		return "", err
	}
	if ref.Step == "" {
		if ref.Job == job.ID {
			return "", fmt.Errorf("job %q cannot consume its own job output %q; reference the producing step instead", job.ID, ref.Name)
		}
		_, declared := target.Outputs[ref.Name]
		if !declared && !slices.ContainsFunc(target.CallOutputs, func(output OutputDecl) bool { return output.Name == ref.Name }) {
			return "", fmt.Errorf("job %q references unknown output %q on job %q", job.ID, ref.Name, ref.Job)
		}
		job.Needs = append(job.Needs, ref.Job)
		return jobOutput(ref.Job, ref.Name), nil
	}
	producer, exists := graph.steps[ref.Job][ref.Step]
	if !exists || !producer.outputs[ref.Name] {
		return "", fmt.Errorf("job %q references undeclared step output %q on %s/%s", job.ID, ref.Name, ref.Job, ref.Step)
	}
	if ref.Job == job.ID {
		if position <= producer.index {
			return "", fmt.Errorf("job %q consumes output of step %q before it runs", job.ID, ref.Step)
		}
		return stepOutput(ref.Step, ref.Name), nil
	}
	name, err := exposeOutput(target, ref)
	if err != nil {
		return "", err
	}
	job.Needs = append(job.Needs, ref.Job)
	return jobOutput(ref.Job, name), nil
}

func indexSteps(job *Job) (map[StepID]producer, error) {
	index := make(map[StepID]producer)
	for i, step := range job.Steps {
		opts, err := options(step)
		if err != nil {
			return nil, err
		}
		if opts.ID == "" {
			if len(opts.Outputs) > 0 {
				return nil, fmt.Errorf("job %q declares outputs on an anonymous step", job.ID)
			}
			continue
		}
		if err := validateIdentifier("step id", string(opts.ID)); err != nil {
			return nil, err
		}
		if _, exists := index[opts.ID]; exists {
			return nil, fmt.Errorf("duplicate workflow step id %q in job %q", opts.ID, job.ID)
		}
		outputs := make(map[OutputName]bool)
		for _, output := range opts.Outputs {
			if err := validateIdentifier("output name", string(output.Name)); err != nil {
				return nil, err
			}
			if outputs[output.Name] {
				return nil, fmt.Errorf("duplicate output %q on step %s/%s", output.Name, job.ID, opts.ID)
			}
			outputs[output.Name] = true
		}
		index[opts.ID] = producer{index: i, outputs: outputs}
	}
	return index, nil
}

func exposeOutput(job *Job, ref OutputRef) (OutputName, error) {
	for _, name := range sortedOutputNames(job.Outputs) {
		if existing, ok := job.Outputs[name].(OutputRef); ok && existing == ref {
			return name, nil
		}
	}
	// Length-prefix the step id: unlike joining with "_", this is injective
	// even when both the step id and output name contain underscores.
	name := OutputName(fmt.Sprintf("aw_%d_%s_%s", len(ref.Step), ref.Step, ref.Name))
	if _, exists := job.Outputs[name]; exists {
		return "", fmt.Errorf("generated output binding %q conflicts with an explicit output on job %q", name, job.ID)
	}
	job.Outputs[name] = ref
	return name, nil
}

func walkJob(job *Job, resolve func(OutputRef, int) (string, error)) error {
	check := func(expr Expression, position int) error {
		ctx := expressionContext{resolve: func(ref OutputRef) (string, error) { return resolve(ref, position) }}
		_, err := ctx.render(expr)
		return err
	}
	for _, expr := range []Expression{job.If, job.ContinueOnError, job.TimeoutMinutesExpression} {
		if expr != nil {
			if err := check(expr, -1); err != nil {
				return err
			}
		}
	}
	if err := walkEnv(job.Env, -1, check); err != nil {
		return err
	}
	for _, values := range []map[string]Expression{job.With, job.Secrets} {
		for _, key := range sortedKeys(values) {
			if err := check(values[key], -1); err != nil {
				return err
			}
		}
	}
	for _, name := range sortedOutputNames(job.Outputs) {
		if err := validateIdentifier("output name", string(name)); err != nil {
			return err
		}
		if err := check(job.Outputs[name], len(job.Steps)); err != nil {
			return err
		}
	}
	for i, step := range job.Steps {
		if err := walkStep(step, i, check); err != nil {
			return err
		}
	}
	return nil
}

func walkStep(step Step, position int, check func(Expression, int) error) error {
	opts, err := options(step)
	if err != nil {
		return err
	}
	for _, expr := range []Expression{opts.If, opts.ContinueOnError, opts.TimeoutMinutesExpression} {
		if expr != nil {
			if err := check(expr, position); err != nil {
				return err
			}
		}
	}
	if err := walkEnv(opts.Env, position, check); err != nil {
		return err
	}
	if action, ok := step.(ActionStep); ok {
		for _, key := range sortedKeys(action.With) {
			if err := check(action.With[key], position); err != nil {
				return err
			}
		}
	}
	return nil
}

func walkEnv(env map[string]Expression, position int, check func(Expression, int) error) error {
	for _, key := range sortedKeys(env) {
		if err := validateIdentifier("environment variable", key); err != nil {
			return err
		}
		if err := check(env[key], position); err != nil {
			return err
		}
	}
	return nil
}

func walkWorkflowEnv(env map[string]Expression) error {
	ctx := expressionContext{resolve: func(OutputRef) (string, error) {
		return "", errors.New("workflow env cannot reference job or step outputs; use job env instead")
	}}
	return walkEnv(env, -1, func(expr Expression, _ int) error {
		_, err := ctx.render(expr)
		return err
	})
}

func validateCallOutputs(job *Job) error {
	if len(job.CallOutputs) > 0 && job.Uses == "" {
		return errors.New("call outputs require a reusable workflow uses reference")
	}
	seen := make(map[OutputName]bool)
	for _, output := range job.CallOutputs {
		if err := validateIdentifier("output name", string(output.Name)); err != nil {
			return err
		}
		if seen[output.Name] {
			return fmt.Errorf("duplicate reusable workflow output %q on job %q", output.Name, job.ID)
		}
		seen[output.Name] = true
	}
	return nil
}

func detectCycles(ordered []*Job, jobs map[JobID]*Job) error {
	state := make(map[JobID]int)
	var visit func(JobID) error
	visit = func(id JobID) error {
		if state[id] == 1 {
			return fmt.Errorf("cycle detected in job dependencies through job '%s'", id)
		}
		if state[id] == 2 {
			return nil
		}
		state[id] = 1
		for _, dep := range jobs[id].Needs {
			if err := visit(dep); err != nil {
				return err
			}
		}
		state[id] = 2
		return nil
	}
	for _, job := range ordered {
		if err := visit(job.ID); err != nil {
			return err
		}
	}
	return nil
}

func stepOutput(step StepID, name OutputName) string {
	return fmt.Sprintf("steps.%s.outputs.%s", step, name)
}

func jobOutput(job JobID, name OutputName) string {
	return fmt.Sprintf("needs.%s.outputs.%s", job, name)
}

func sortedKeys[V any](values map[string]V) []string {
	return slices.Sorted(maps.Keys(values))
}

func sortedOutputNames(values map[OutputName]Expression) []OutputName {
	return slices.Sorted(maps.Keys(values))
}

// RenderExpression lowers an expression in a resolved workflow. Call Resolve
// first so cross-job bindings and dependency validation have been completed.
func RenderExpression(expr Expression, consumer JobID, resolved *Workflow, wrap bool) (string, error) {
	if resolved == nil {
		return "", errors.New("rendering workflow expressions requires a resolved workflow")
	}
	ctx := expressionContext{resolve: func(ref OutputRef) (string, error) {
		if ref.Step != "" && ref.Job == consumer {
			return stepOutput(ref.Step, ref.Name), nil
		}
		if ref.Step == "" {
			return jobOutput(ref.Job, ref.Name), nil
		}
		for _, job := range resolved.Jobs {
			if job.ID == ref.Job {
				for _, name := range sortedOutputNames(job.Outputs) {
					if existing, ok := job.Outputs[name].(OutputRef); ok && existing == ref {
						return jobOutput(ref.Job, name), nil
					}
				}
			}
		}
		return "", fmt.Errorf("unresolved workflow output reference %s/%s/%s", ref.Job, ref.Step, ref.Name)
	}}
	if wrap {
		return ctx.value(expr)
	}
	rendered, err := ctx.render(expr)
	return strings.TrimSpace(rendered), err
}
