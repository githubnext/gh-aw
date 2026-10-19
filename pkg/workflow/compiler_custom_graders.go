package workflow

import (
	"errors"
	"fmt"
	"path"

	"github.com/github/gh-aw/pkg/constants"
)

// Custom code runs downstream of conclusion, without its write credentials or environment.
func (c *Compiler) buildCustomGradersJob(data *WorkflowData) (*Job, error) {
	if len(data.Graders.CustomGraderIDs()) == 0 {
		return nil, nil
	}
	if isWorkQueueParticipant(data) {
		return nil, errors.New("custom graders require a published conclusion session and are not supported for work-queue participant workflows")
	}
	env, err := graderEnvironmentForJob(data.Graders, true)
	if err != nil {
		return nil, fmt.Errorf("cannot serialize custom graders: %w", err)
	}
	setupRef := c.resolveActionReference("./actions/setup", data)
	steps := c.generateCheckoutActionsFolder(data)
	// Deliberately omit workflow authentication and OTLP configuration from setup.
	steps = append(steps, c.generateSetupStepForJob("graders", &WorkflowData{Name: data.Name}, setupRef, SetupActionDestination, false, "", "", "")...)
	prefix := artifactPrefixExprForDownstreamJob(data)
	steps = append(steps, buildArtifactDownloadSteps(ArtifactDownloadConfig{
		ArtifactName:    prefix + constants.UsageArtifactName.String(),
		DownloadPath:    path.Join(constants.TmpGhAwDir, constants.UsageArtifactName.String()),
		StepName:        "Download unified session",
		ContinueOnError: new(false),
	}, c.getActionPin)...)
	steps = append(steps,
		"      - name: Run isolated custom graders\n",
		fmt.Sprintf("        uses: %s\n", c.getActionPin("actions/github-script")),
		"        env:\n", env,
		"        with:\n",
		"          script: |\n",
		fmt.Sprintf("            const { setupGlobals } = require('%s/setup_globals.cjs');\n", SetupActionDestination),
		"            setupGlobals(core, github, context, exec, io, getOctokit);\n",
		fmt.Sprintf("            const { main } = require('%s/custom_graders.cjs');\n", SetupActionDestination),
		"            await main();\n",
		"      - name: Upload custom grader results\n",
		"        if: always()\n",
		fmt.Sprintf("        uses: %s\n", c.getActionPin("actions/upload-artifact")),
		"        with:\n",
		fmt.Sprintf("          name: %s%s\n", prefix, constants.GradersArtifactName),
		"          path: /tmp/gh-aw/custom-graders/*.json\n",
		"          if-no-files-found: error\n",
	)
	return &Job{
		Name:           string(constants.GradersJobName),
		RunsOn:         "runs-on: ubuntu-latest",
		Needs:          []string{string(constants.ActivationJobName), string(constants.ConclusionJobName)},
		If:             "always() && needs.conclusion.result == 'success'",
		Permissions:    NewPermissionsContentsRead().RenderToYAML(),
		TimeoutMinutes: 10,
		Steps:          steps,
	}, nil
}
