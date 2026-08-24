package workflow

import (
	"testing"

	"github.com/github/gh-aw/pkg/constants"
)

func TestCTR026_PositiveIntLiteralAccepted(t *testing.T) {
	for _, jobName := range []string{string(constants.AgentJobName), string(constants.DetectionJobName), "custom"} {
		for _, timeout := range []any{15, int64(15), uint64(15), float64(15)} {
			job := &Job{TimeoutMinutesExpression: "${{ inputs.timeout }}"}
			if err := extractCustomJobTimeoutMinutes(job, jobName, map[string]any{"timeout-minutes": timeout}); err != nil {
				t.Fatalf("%s with %T(%v): %v", jobName, timeout, timeout, err)
			}
			if job.TimeoutMinutes != 15 || job.TimeoutMinutesExpression != "" {
				t.Fatalf("%s with %T(%v): got timeout=%d expression=%q", jobName, timeout, timeout, job.TimeoutMinutes, job.TimeoutMinutesExpression)
			}
		}
	}
}

func TestCTR026_NonPositiveRejected(t *testing.T) {
	for _, jobName := range ctr026JobNames() {
		for _, timeout := range []any{0, -1, int64(0), int64(-1), uint64(0), float64(0), float64(-1)} {
			ctr026RequireTimeoutError(t, jobName, timeout)
		}
	}
}

func TestCTR026_GeneratedJobExpressionRejected(t *testing.T) {
	for _, jobName := range []string{string(constants.AgentJobName), string(constants.DetectionJobName)} {
		for _, timeout := range []any{"not-an-expression", "${{ inputs.timeout }}"} {
			ctr026RequireTimeoutError(t, jobName, timeout)
		}
	}
}

func TestCTR026_NonGeneratedExpressionAccepted(t *testing.T) {
	const expression = "${{ inputs.timeout }}"
	job := &Job{TimeoutMinutes: 15}
	if err := extractCustomJobTimeoutMinutes(job, "custom", map[string]any{"timeout-minutes": expression}); err != nil {
		t.Fatal(err)
	}
	if job.TimeoutMinutes != 0 || job.TimeoutMinutesExpression != expression {
		t.Fatalf("got timeout=%d expression=%q", job.TimeoutMinutes, job.TimeoutMinutesExpression)
	}
}

func TestCTR026_EmptyStringRejected(t *testing.T) {
	for _, jobName := range ctr026JobNames() {
		for _, timeout := range []any{"", " \t\n"} {
			ctr026RequireTimeoutError(t, jobName, timeout)
		}
	}
}

func TestCTR026_NonIntegralFloatRejected(t *testing.T) {
	for _, jobName := range ctr026JobNames() {
		ctr026RequireTimeoutError(t, jobName, float64(15.5))
	}
}

func TestCTR026_OutOfRangeFloatRejected(t *testing.T) {
	for _, jobName := range ctr026JobNames() {
		ctr026RequireTimeoutError(t, jobName, float64(1<<53)+2)
	}
}

func TestCTR026_UnsupportedTypeRejected(t *testing.T) {
	for _, jobName := range ctr026JobNames() {
		for _, timeout := range []any{true, []any{15}, map[string]any{"timeout": 15}, nil} {
			ctr026RequireTimeoutError(t, jobName, timeout)
		}
	}
}

func TestCTR026_AbsentFieldIsNoOp(t *testing.T) {
	job := &Job{TimeoutMinutes: 15, TimeoutMinutesExpression: "${{ inputs.timeout }}"}
	if err := extractCustomJobTimeoutMinutes(job, string(constants.AgentJobName), map[string]any{}); err != nil {
		t.Fatal(err)
	}
	if job.TimeoutMinutes != 15 || job.TimeoutMinutesExpression != "${{ inputs.timeout }}" {
		t.Fatalf("got timeout=%d expression=%q", job.TimeoutMinutes, job.TimeoutMinutesExpression)
	}
}

func ctr026JobNames() []string {
	return []string{string(constants.AgentJobName), string(constants.DetectionJobName), "custom"}
}

func ctr026RequireTimeoutError(t *testing.T, jobName string, timeout any) {
	t.Helper()
	if err := extractCustomJobTimeoutMinutes(&Job{}, jobName, map[string]any{"timeout-minutes": timeout}); err == nil {
		t.Fatalf("%s with %T(%v): expected error", jobName, timeout, timeout)
	}
}
