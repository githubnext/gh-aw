package workflow

import "fmt"

func (c *Compiler) buildWorkQueueTerminationStep(data *WorkflowData) []string {
	if !isWorkQueueWorker(data) && !workQueueRequiresAssignment(data) {
		return nil
	}
	steps := []string{
		"      - name: Terminate work queue worker\n",
		"        id: worker_termination\n",
		"        if: always()\n",
		fmt.Sprintf("        uses: %s\n", c.getActionPin("actions/github-script")),
	}
	steps = append(steps, workQueuePolicyEnvironment(data)...)
	var staged *TemplatableBool
	if data.SafeOutputs != nil {
		staged = data.SafeOutputs.Staged
	}
	if value := resolveSafeOutputsStagedValue(c.trialMode, staged); value != nil {
		steps = append(steps, fmt.Sprintf("          GH_AW_SAFE_OUTPUTS_STAGED: %q\n", *value))
	}
	return append(steps,
		"        with:\n",
		"          script: |\n",
		"            const { main } = require('${{ runner.temp }}/gh-aw/actions/reconcile_work_queue_run.cjs');\n",
		"            await main({ core, githubClient: github, context });\n",
	)
}
