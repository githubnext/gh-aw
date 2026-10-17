package pipeline

import "errors"

// Pipeline is an Actions workflow. Envelope carries native Actions top-level
// fields such as on, permissions, concurrency, defaults, and run-name.
// It cannot override name or jobs.
type Pipeline struct {
	Name     string
	Envelope map[string]any
	Jobs     []*Job
}

type Job struct {
	ID        JobID
	Name      string
	RunsOn    any
	Needs     []JobID
	Condition Expression
	Env       map[string]Expression
	Steps     []Step
	Outputs   map[OutputName]Expression

	// Fields carries native job properties not yet modeled by the IR. It
	// cannot override typed properties (even when they are unset).
	Fields map[string]any
}

type Step interface {
	pipelineStep()
}

type OutputDecl struct {
	Name OutputName
}

type StepOptions struct {
	ID              StepID
	Name            string
	Condition       Expression
	Env             map[string]Expression
	ContinueOnError *bool
	TimeoutMinutes  int
	Outputs         []OutputDecl
}

type RunStep struct {
	StepOptions
	Script           string
	Shell            string
	WorkingDirectory string
}

type ActionStep struct {
	StepOptions
	Uses string
	With map[string]Expression
}

// RawStep is one complete authored step mapping, without a list marker.
// The graph indexes its id, but cannot discover output declarations or typed
// references inside arbitrary scripts/expressions. Use typed steps for those.
type RawStep struct {
	Fields map[string]any
}

func (RunStep) pipelineStep()    {}
func (ActionStep) pipelineStep() {}
func (RawStep) pipelineStep()    {}

func options(step Step) (StepOptions, error) {
	switch value := step.(type) {
	case RunStep:
		return value.StepOptions, nil
	case ActionStep:
		return value.StepOptions, nil
	case RawStep:
		var result StepOptions
		if raw, exists := value.Fields["id"]; exists {
			id, ok := raw.(string)
			if !ok {
				return StepOptions{}, errors.New("raw pipeline step id must be a string")
			}
			if err := validateIdentifier("step id", id); err != nil {
				return StepOptions{}, err
			}
			result.ID = StepID(id)
		}
		return result, nil
	default:
		return StepOptions{}, errInvalidStep
	}
}
