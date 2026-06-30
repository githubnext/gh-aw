package workflow

import (
	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
)

var compilerMainJobLog = logger.New("workflow:compiler_main_job")

func isBuiltinJobName(jobName string) bool {
	_, isBuiltIn := constants.KnownBuiltInJobNames[jobName]
	return isBuiltIn
}

// buildMainJob creates the main agent job that runs the AI agent with the configured engine and tools.
// This job depends on the activation job if it exists, and handles the main workflow logic.
func (c *Compiler) buildMainJob(data *WorkflowData, activationJobCreated bool) (*Job, error) {
	workflowLog.Printf("Building main job for workflow: %s", data.Name)

	steps, err := c.buildMainJobSteps(data)
	if err != nil {
		return nil, err
	}
	if c.actionMode.IsScript() {
		steps = append(steps, c.generateScriptModeCleanupStep())
	}

	jobCondition := c.resolveAgentJobCondition(data, activationJobCreated)
	depends := c.buildDirectDependencies(data, activationJobCreated)
	depends, engineEnvContent := c.augmentDependenciesFromContent(data, depends)
	c.warnBuiltinEngineEnvRefs(depends, engineEnvContent)

	outputs := c.buildAgentJobOutputs(data)
	env := c.buildAgentJobEnv(data)

	permissions, err := c.buildAgentJobPermissions(data)
	if err != nil {
		return nil, err
	}

	agentConcurrency := GenerateJobConcurrencyConfig(data)

	return &Job{
		Name:        string(constants.AgentJobName),
		If:          jobCondition,
		RunsOn:      c.indentYAMLLines(data.RunsOn, "    "),
		Environment: c.indentYAMLLines(data.Environment, "    "),
		Container:   c.indentYAMLLines(data.Container, "    "),
		Services:    c.indentYAMLLines(data.Services, "    "),
		Permissions: c.indentYAMLLines(permissions, "    "),
		Concurrency: c.indentYAMLLines(agentConcurrency, "    "),
		Env:         env,
		Steps:       steps,
		Needs:       depends,
		Outputs:     outputs,
	}, nil
}
