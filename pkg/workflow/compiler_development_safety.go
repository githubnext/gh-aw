package workflow

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/github/gh-aw/pkg/stringutil"
)

// Authored host code cannot be made read-only by staging safe outputs.
func (c *Compiler) validateDryRunEffects(data *WorkflowData) error {
	if !c.dryRun {
		return nil
	}
	var unsupported []string
	for _, section := range []struct {
		name, content string
	}{
		{"steps", data.CustomSteps},
		{"pre-steps", data.PreSteps},
		{"pre-agent-steps", data.PreAgentSteps},
		{"post-steps", data.PostSteps},
	} {
		if strings.TrimSpace(section.content) != "" {
			unsupported = append(unsupported, section.name)
		}
	}
	if len(data.OnSteps) > 0 {
		unsupported = append(unsupported, "on.steps")
	}
	if len(data.ResolvedMCPServers) > 0 || (data.ParsedTools != nil && len(data.ParsedTools.Custom) > 0) {
		unsupported = append(unsupported, "custom MCP servers")
	}
	if data.MCPScripts != nil {
		unsupported = append(unsupported, "mcp-scripts")
	}
	if data.SafeOutputs != nil && len(data.SafeOutputs.Scripts) > 0 {
		unsupported = append(unsupported, "safe-outputs.scripts")
	}
	if data.Graders != nil {
		for _, name := range slices.Sorted(maps.Keys(data.Graders.Graders)) {
			grader := data.Graders.Graders[name]
			if grader != nil && (grader.Enabled == nil || *grader.Enabled) && grader.Run != "" {
				unsupported = append(unsupported, "graders."+name+".run")
			}
		}
	}
	permissions := NewPermissionsParser(data.Permissions).ToPermissions()
	if level, _ := permissions.Get(PermissionIdToken); level == PermissionWrite {
		unsupported = append(unsupported, "permissions.id-token")
	}
	if data.OnPermissions != nil {
		if level, _ := data.OnPermissions.Get(PermissionIdToken); level == PermissionWrite {
			unsupported = append(unsupported, "on.permissions.id-token")
		}
	}
	unsupported = append(unsupported, dryRunBuiltinJobEffects(data.Jobs)...)
	if len(unsupported) > 0 {
		return fmt.Errorf("--dry-run cannot safely execute %s: remove these declarations and their imports from the diagnostic workflow; staging does not isolate host scripts, external MCP effects, or cloud credentials", strings.Join(unsupported, ", "))
	}
	return nil
}

func dryRunBuiltinJobEffects(jobs map[string]any) []string {
	var unsupported []string
	for _, name := range slices.Sorted(maps.Keys(jobs)) {
		if !isBuiltinJobName(normalizeBuiltinJobAlias(name)) {
			continue
		}
		config, ok := jobs[name].(map[string]any)
		if !ok {
			continue // Job validation reports invalid configurations.
		}
		for _, field := range []string{"setup-steps", "pre-steps", "steps", "permissions", "uses", "with", "secrets"} {
			if value, exists := config[field]; exists && value != nil {
				unsupported = append(unsupported, "jobs."+name+"."+field)
			}
		}
	}
	return unsupported
}

func (c *Compiler) disableDryRunCustomJobs(data *WorkflowData) {
	if !c.dryRun {
		return
	}
	for name := range data.Jobs {
		if !isBuiltinJobName(normalizeBuiltinJobAlias(name)) {
			c.disableDryRunCustomJob(name)
		}
	}
	if data.SafeOutputs != nil {
		for name := range data.SafeOutputs.Jobs {
			c.disableDryRunCustomJob(stringutil.NormalizeSafeOutputIdentifier(name))
		}
	}
}

func (c *Compiler) disableDryRunCustomJob(name string) {
	job, exists := c.jobManager.GetJob(name)
	if !exists {
		return
	}
	makeDryRunJobInert(job)
	compilerDevelopmentLog.Printf("Dry-run mutation: disable custom job %q", name)
}

func makeDryRunJobInert(job *Job) {
	// Preserve dependency/output names without retaining executable code or credentials.
	outputs := make(map[string]string, len(job.Outputs))
	for output := range job.Outputs {
		outputs[output] = "${{ '' }}"
	}
	*job = Job{
		Name: job.Name, DisplayName: job.DisplayName, Needs: job.Needs, Outputs: outputs,
		If: "${{ !always() }}", RunsOn: "runs-on: ubuntu-latest",
		Permissions: "permissions: {}",
		Steps:       []string{"      - name: Custom job disabled in dry-run\n        run: echo 'Custom job disabled in dry-run'\n"},
	}
}
