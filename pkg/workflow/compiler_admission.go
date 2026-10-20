package workflow

import (
	"fmt"

	"github.com/github/gh-aw/pkg/workflow/compilerenv"
)

func (c *Compiler) addActivationAdmissionStep(ctx *activationJobBuildContext) {
	for _, name := range []string{"result", "max-ai-credits", "max-ai-credits-threat-detection", "max-ai-credits-evals"} {
		ctx.outputs["admission-"+name] = "${{ steps.admission.outputs." + name + " }}"
	}
	ctx.steps = append(ctx.steps,
		"      - name: Check workflow admission\n",
		"        id: admission\n",
		fmt.Sprintf("        if: ${{ vars.%s != '' }}\n", compilerenv.DefaultAdmissionURL),
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
}
