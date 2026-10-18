package actions

import (
	"errors"
	"fmt"
)

func validateJob(job *Job) error {
	if job.TimeoutMinutes < 0 {
		return errors.New("job timeout-minutes cannot be negative")
	}
	if (job.TimeoutMinutes != 0 || job.TimeoutMinutesSet) && job.TimeoutMinutesExpression != nil {
		return errors.New("job cannot combine a literal and expression timeout-minutes")
	}
	if job.SecretsInherit && len(job.Secrets) > 0 {
		return errors.New("reusable workflow job cannot combine inherited and explicit secrets")
	}
	if job.Uses == "" {
		if len(job.With) > 0 || len(job.Secrets) > 0 || job.SecretsInherit || len(job.CallOutputs) > 0 {
			return errors.New("with, secrets, and call outputs require a reusable workflow uses reference")
		}
		if job.RunsOn == nil {
			return errors.New("runner job requires runs-on")
		}
		if len(job.Steps) == 0 {
			return errors.New("runner job requires steps")
		}
		return nil
	}
	if len(job.Steps) > 0 || job.RunsOn != nil || len(job.Env) > 0 || len(job.Outputs) > 0 ||
		job.Defaults != nil || job.Environment != nil || job.Container != nil || job.Services != nil ||
		job.TimeoutMinutes != 0 || job.TimeoutMinutesSet || job.TimeoutMinutesExpression != nil || job.ContinueOnError != nil {
		return errors.New("reusable workflow job cannot declare runner properties such as steps, runs-on, env, or outputs")
	}
	return nil
}

// ValidateJob validates the runner or reusable-workflow shape of a job.
func ValidateJob(job *Job) error {
	if job == nil {
		return errors.New("workflow job cannot be nil")
	}
	return validateJob(job)
}

func lowerJobExpressions(fields map[string]any, job *Job, resolved *Workflow) error {
	if job.TimeoutMinutesExpression != nil {
		value, err := RenderExpression(job.TimeoutMinutesExpression, job.ID, resolved, true)
		if err != nil {
			return err
		}
		fields["timeout-minutes"] = value
	}
	if job.ContinueOnError != nil {
		value, err := RenderExpression(job.ContinueOnError, job.ID, resolved, true)
		if err != nil {
			return err
		}
		fields["continue-on-error"] = value
	}
	for key, values := range map[string]map[string]Expression{"with": job.With, "secrets": job.Secrets} {
		if len(values) == 0 {
			continue
		}
		lower := lowerValues
		if key == "with" {
			lower = lowerInputs
		}
		value, err := lower(values, job.ID, resolved)
		if err != nil {
			return err
		}
		fields[key] = value
	}
	if job.SecretsInherit {
		fields["secrets"] = "inherit"
	}
	return nil
}

func lowerInputs(values map[string]Expression, consumer JobID, resolved *Workflow) (map[string]any, error) {
	result, err := lowerValues(values, consumer, resolved)
	if err != nil {
		return nil, err
	}
	for key, value := range values {
		switch literal := value.(type) {
		case Boolean:
			result[key] = bool(literal)
		case Number:
			result[key] = float64(literal)
		}
	}
	return result, nil
}

func validateRawStep(fields map[string]any) error {
	if len(fields) == 0 {
		return errors.New("raw Actions step cannot be empty")
	}
	for _, key := range sortedKeys(fields) {
		switch key {
		case "id", "name", "if", "env", "continue-on-error", "timeout-minutes",
			"run", "shell", "working-directory", "uses", "with":
		default:
			return fmt.Errorf("unsupported GitHub Actions step property %q", key)
		}
	}
	run, hasRun := fields["run"]
	uses, hasUses := fields["uses"]
	if hasRun == hasUses {
		return errors.New("step requires exactly one of run or uses")
	}
	value := uses
	if hasRun {
		value = run
		if _, exists := fields["with"]; exists {
			return errors.New("run step cannot declare with")
		}
	} else {
		for _, key := range []string{"shell", "working-directory"} {
			if _, exists := fields[key]; exists {
				return fmt.Errorf("uses step cannot declare %s", key)
			}
		}
	}
	if text, ok := value.(string); !ok || text == "" {
		return errors.New("step run or uses must be a nonempty string")
	}
	return nil
}
