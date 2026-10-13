package workflow

import (
	"strings"

	"github.com/github/gh-aw/pkg/constants"
)

// scopeEnvironmentOTLPEnv keeps telemetry credentials out of unprotected jobs.
// WorkflowData retains the configuration for telemetry step generation.
func (c *Compiler) scopeEnvironmentOTLPEnv(data *WorkflowData) string {
	if data.Environment == "" && c.environmentOverride == "" {
		return data.Env
	}
	env := parseEnvYAMLSection(data.Env)
	if len(env) == 0 {
		return data.Env
	}
	telemetry := make(map[string]string)
	for key, value := range env {
		if strings.HasPrefix(key, "OTEL_") || strings.HasPrefix(key, "GH_AW_OTLP_") {
			telemetry[key] = quoteYAMLEnvValue(serializeEnvStringValue(value))
			delete(env, key)
		}
	}
	for _, name := range []constants.JobName{
		constants.AgentJobName, constants.DetectionJobName,
		constants.SafeOutputsJobName, constants.ConclusionJobName,
	} {
		job := c.jobManager.GetAllJobs()[string(name)]
		if job == nil || job.Environment == "" || job.Uses != "" {
			continue
		}
		if job.Env == nil {
			job.Env = make(map[string]string)
		}
		for key, value := range telemetry {
			if _, exists := job.Env[key]; !exists {
				job.Env[key] = value
			}
		}
	}
	if len(env) == 0 {
		return ""
	}
	return c.extractTopLevelYAMLSection(map[string]any{"env": env}, "env")
}
