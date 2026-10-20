package workflow

import (
	"errors"
	"fmt"
	"strings"

	"github.com/github/gh-aw/pkg/workflow/compilerenv"
)

const activationAdmissionJobName = "activation_admission"

func (c *Compiler) buildActivationAdmissionJob(ctx *activationJobBuildContext) (*Job, error) {
	if _, exists := ctx.data.Jobs[activationAdmissionJobName]; exists {
		return nil, fmt.Errorf("jobs.%s is compiler-owned", activationAdmissionJobName)
	}
	setupActionRef := c.resolveActionReference("./actions/setup", ctx.data)
	if setupActionRef == "" {
		return nil, errors.New("setup action reference is required for workflow admission")
	}
	setupData := &WorkflowData{Name: ctx.data.Name, WorkflowID: ctx.data.WorkflowID}
	steps := c.generateCheckoutActionsFolder(ctx.data)
	steps = append(steps, c.generateSetupStepForJob(activationAdmissionJobName, setupData, setupActionRef, SetupActionDestination, false, "", "", "")...)
	steps = append(steps,
		"      - name: Check workflow admission\n",
		"        id: admission\n",
		fmt.Sprintf("        uses: %s\n", getCachedActionPin("actions/github-script", ctx.data)),
		"        env:\n",
		fmt.Sprintf("          %s: ${{ vars.%s }}\n", compilerenv.DefaultAdmissionURL, compilerenv.DefaultAdmissionURL),
		fmt.Sprintf("          %s: ${{ vars.%s }}\n", compilerenv.DefaultAdmissionAudience, compilerenv.DefaultAdmissionAudience),
		fmt.Sprintf("          %s: ${{ vars.%s }}\n", compilerenv.DefaultAdmissionOnOutage, compilerenv.DefaultAdmissionOnOutage),
		fmt.Sprintf("          GH_AW_ADMISSION_WORKFLOW_HASH: %q\n", ctx.data.FrontmatterHash+":"+ctx.data.BodyHash),
		"          GH_AW_INFO_WORKFLOW_ID: ${{ job.workflow_ref }}\n",
		"          GITHUB_ACTOR: ${{ github.actor }}\n",
		"          GITHUB_TRIGGERING_ACTOR: ${{ github.triggering_actor }}\n",
		"        with:\n",
		"          script: |\n",
		"            await require(require('node:path').join(process.env.RUNNER_TEMP, 'gh-aw', 'actions', 'activation_admission.cjs')).main(core, github, context, exec, io, getOctokit);\n",
	)

	permissions := NewPermissionsContentsRead()
	permissions.Set(PermissionIdToken, PermissionWrite)
	needs := append([]string(nil), ctx.activationNeeds...)
	admissionCondition := c.combineJobIfConditions(
		ctx.activationCondition,
		fmt.Sprintf("${{ vars.%s != '' }}", compilerenv.DefaultAdmissionURL),
	)
	return &Job{
		Name:        activationAdmissionJobName,
		If:          admissionCondition,
		RunsOn:      c.formatFrameworkJobRunsOn(ctx.data),
		Permissions: permissions.RenderToYAML(),
		Steps:       steps,
		Needs:       needs,
		Outputs: map[string]string{
			"result":                          "${{ steps.admission.outputs.result }}",
			"max-ai-credits":                  "${{ steps.admission.outputs.max-ai-credits }}",
			"max-ai-credits-threat-detection": "${{ steps.admission.outputs.max-ai-credits-threat-detection }}",
			"max-ai-credits-evals":            "${{ steps.admission.outputs.max-ai-credits-evals }}",
		},
	}, nil
}

func addActivationAdmissionOutputs(ctx *activationJobBuildContext) {
	for _, name := range []string{"result", "max-ai-credits", "max-ai-credits-threat-detection", "max-ai-credits-evals"} {
		ctx.outputs["admission-"+name] = "${{ needs." + activationAdmissionJobName + ".outputs." + name + " }}"
	}
}

func addActivationAdmissionDependency(ctx *activationJobBuildContext) {
	previousNeeds := append([]string(nil), ctx.activationNeeds...)
	ctx.activationNeeds = append(ctx.activationNeeds, activationAdmissionJobName)

	conditions := []string{
		"always()",
		"(needs." + activationAdmissionJobName + ".result == 'success' || needs." + activationAdmissionJobName + ".result == 'skipped')",
	}
	for _, need := range previousNeeds {
		conditions = append(conditions, "needs."+need+".result == 'success'")
	}
	if ctx.activationCondition != "" {
		conditions = append(conditions, "("+stripExpressionWrapper(ctx.activationCondition)+")")
	}
	ctx.activationCondition = RenderCondition(&ExpressionNode{Expression: strings.Join(conditions, " && ")})
}
