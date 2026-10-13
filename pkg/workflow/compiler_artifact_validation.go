package workflow

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/github/gh-aw/pkg/constants"
)

const artifactValidationInputRoot = "/tmp/gh-aw/artifact-validation/input"
const artifactValidationPreparedRoot = "/tmp/gh-aw/artifact-validation/prepared"
const artifactValidationApprovedRoot = "/tmp/gh-aw/artifact-validation/approved"
const artifactPublicationRoot = "/tmp/gh-aw/approved-artifacts"

func hasArtifactValidation(data *WorkflowData) bool {
	return data != nil && data.SafeOutputs != nil && data.SafeOutputs.ArtifactValidation != nil
}

func (c *Compiler) buildArtifactValidationJob(data *WorkflowData) (*Job, error) {
	if !hasArtifactValidation(data) {
		return nil, nil
	}
	if err := validateArtifactValidation(data); err != nil {
		return nil, err
	}
	preparationData := *data
	preparationData.RawFrontmatter = nil
	preparationData.ParsedFrontmatter = nil
	preparationData.OTLPEndpoint = ""
	preparationData.OTLPHeaders = ""
	preparationData.OTLPEndpoints = ""
	preparationData.OTLPUsesEnterpriseDefaults = false
	preparationData.Env = ""

	env := map[string]string{
		"GH_AW_ARTIFACT_STAGING_ROOT":  fmt.Sprintf("%q", artifactValidationInputRoot),
		"GH_AW_ARTIFACT_PREPARED_ROOT": fmt.Sprintf("%q", artifactValidationPreparedRoot),
		"GH_AW_ARTIFACT_APPROVED_ROOT": fmt.Sprintf("%q", artifactValidationApprovedRoot),
	}
	for key := range parseEnvYAMLSection(data.Env) {
		if _, reserved := env[key]; !reserved {
			env[key] = `""`
		}
	}
	steps := c.buildSafeOutputsSetupSteps(&preparationData)
	steps = append(steps, c.buildArtifactValidatorCheckout(data)...)
	prepare, err := c.buildArtifactValidationRuntimeStep(data, "prepare")
	if err != nil {
		return nil, err
	}
	steps = append(steps, prepare...)
	for index, raw := range data.SafeOutputs.ArtifactValidation.Steps {
		step, ok := raw.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("CTR-033: invalid artifact validator step %d", index)
		}
		typed, err := MapToStep(step)
		if err != nil {
			return nil, err
		}
		rendered, err := c.buildIsolatedArtifactValidatorStep(data, typed)
		if err != nil {
			return nil, err
		}
		steps = append(steps, rendered)
	}
	finalize, err := c.buildArtifactValidationRuntimeStep(data, "finalize")
	if err != nil {
		return nil, err
	}
	steps = append(steps, finalize...)
	steps = append(steps,
		"      - name: Upload approved repository artifacts\n",
		"        id: approved_artifacts\n",
		fmt.Sprintf("        uses: %s\n", c.getActionPin("actions/upload-artifact")),
		"        with:\n",
		fmt.Sprintf("          name: %sapproved-repository-artifacts\n", artifactPrefixExprForDownstreamJob(data)),
		fmt.Sprintf("          path: %s/\n", artifactValidationApprovedRoot),
		"          if-no-files-found: error\n",
		"          retention-days: 1\n",
	)
	conditionData := *data
	conditionSafeOutputs := *data.SafeOutputs
	conditionSafeOutputs.ArtifactValidation = nil
	conditionData.SafeOutputs = &conditionSafeOutputs
	condition := BuildAnd(
		buildSafeOutputsJobCondition(&conditionData, IsDetectionJobEnabled(data.SafeOutputs)),
		BuildEquals(BuildPropertyAccess("needs.agent.result"), BuildStringLiteral("success")),
	)
	needs := []string{string(constants.AgentJobName), string(constants.ActivationJobName)}
	if IsDetectionJobEnabled(data.SafeOutputs) {
		needs = append(needs, string(constants.DetectionJobName))
	}
	return &Job{
		Name:           artifactValidationJobName,
		RunsOn:         "runs-on: ubuntu-latest",
		If:             RenderCondition(condition),
		Permissions:    "permissions:\n      contents: read",
		TimeoutMinutes: 15,
		Env:            env,
		Steps:          steps,
		Needs:          needs,
		Outputs: map[string]string{
			"artifact_id":     "${{ steps.approved_artifacts.outputs.artifact-id }}",
			"approval_digest": "${{ steps.artifact_finalize.outputs.approval_digest }}",
		},
	}, nil
}

func (c *Compiler) buildArtifactValidatorCheckout(data *WorkflowData) []string {
	return []string{
		"      - name: Checkout trusted artifact validator\n",
		fmt.Sprintf("        uses: %s\n", c.getActionPin("actions/checkout")),
		"        with:\n",
		fmt.Sprintf("          ref: ${{ %s || github.sha }}\n", pullRequestBaseSHAExpression),
		"          persist-credentials: false\n",
		"          fetch-depth: 0\n",
	}
}

func (c *Compiler) buildArtifactValidationRuntimeStep(data *WorkflowData, phase string) ([]string, error) {
	var steps []string
	steps = append(steps,
		fmt.Sprintf("      - name: %s repository artifact validation\n", phase),
		fmt.Sprintf("        id: artifact_%s\n", phase),
		fmt.Sprintf("        uses: %s\n", c.getActionPin("actions/github-script")),
		"        env:\n",
	)
	if err := c.addArtifactValidationPolicyEnvVar(&steps, data); err != nil {
		return nil, fmt.Errorf("CTR-033: artifact validation policy: %w", err)
	}
	if err := validateArtifactValidatorValue(strings.Join(steps, "")); err != nil {
		return nil, fmt.Errorf("CTR-034: artifact validation policy configuration: %w", err)
	}
	steps = append(steps,
		fmt.Sprintf("          GH_AW_ARTIFACT_STAGING_ROOT: %q\n", artifactValidationInputRoot),
		fmt.Sprintf("          GH_AW_ARTIFACT_PREPARED_ROOT: %q\n", artifactValidationPreparedRoot),
		fmt.Sprintf("          GH_AW_ARTIFACT_APPROVED_ROOT: %q\n", artifactValidationApprovedRoot),
		fmt.Sprintf("          GH_AW_ARTIFACT_POLICY_REF: ${{ %s || github.sha }}\n", pullRequestBaseSHAExpression),
	)
	if phase == "prepare" {
		steps = append(steps, fmt.Sprintf("          GH_AW_ARTIFACT_INPUT_NAME: %q\n", artifactPrefixExprForDownstreamJob(data)+constants.AgentArtifactName.String()))
	} else {
		steps = append(steps, "          GH_AW_ARTIFACT_PREPARE_DIGEST: ${{ steps.artifact_prepare.outputs.prepare_digest }}\n")
	}
	steps = append(steps,
		"        with:\n",
		"          script: |\n",
		fmt.Sprintf("            const { %sArtifactValidation } = require('${{ runner.temp }}/gh-aw/actions/artifact_validation.cjs');\n", phase),
		fmt.Sprintf("            await %sArtifactValidation({ core, github, context });\n", phase),
	)
	return steps, nil
}

func (c *Compiler) buildIsolatedArtifactValidatorStep(data *WorkflowData, validator *WorkflowStep) (string, error) {
	keys := make([]string, 0, len(validator.Env))
	for key := range validator.Env {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	configuration, err := json.Marshal(struct {
		Run     string   `json:"run"`
		EnvKeys []string `json:"envKeys"`
	}{Run: validator.Run, EnvKeys: keys})
	if err != nil {
		return "", fmt.Errorf("CTR-033: serialize artifact validator: %w", err)
	}
	image := resolveContainerImage(constants.DefaultGhAwNodeImage, data)
	if !strings.Contains(image, "@sha256:") {
		return "", fmt.Errorf("CTR-034: artifact validation requires a digest-pinned gh-aw Node image")
	}
	env := make(map[string]string, len(validator.Env)+2)
	for key, value := range validator.Env {
		env[key] = value
	}
	env["GH_AW_ARTIFACT_VALIDATOR_STEP"] = string(configuration)
	env["GH_AW_ARTIFACT_VALIDATOR_IMAGE"] = image
	step := &WorkflowStep{
		Name:           validator.Name,
		ID:             validator.ID,
		Uses:           c.getActionPin("actions/github-script"),
		Env:            env,
		TimeoutMinutes: validator.TimeoutMinutes,
		With: map[string]any{"script": "const { runArtifactValidator } = require('${{ runner.temp }}/gh-aw/actions/artifact_validator.cjs');\n" +
			"await runArtifactValidator({ core });"},
	}
	return ConvertStepToYAML(step.ToMap())
}

func (c *Compiler) buildApprovedArtifactDownloadSteps() []string {
	return []string{
		"      - name: Download trusted approved repository artifacts\n",
		fmt.Sprintf("        uses: %s\n", c.getActionPin("actions/download-artifact")),
		"        with:\n",
		"          artifact-ids: ${{ needs.artifact_validation.outputs.artifact_id }}\n",
		fmt.Sprintf("          path: %s\n", artifactPublicationRoot),
	}
}

func (c *Compiler) buildArtifactPublicationInitializationStep(data *WorkflowData) ([]string, error) {
	steps := []string{
		"      - name: Verify approved repository artifacts\n",
		"        id: setup-agent-output-env\n",
		fmt.Sprintf("        uses: %s\n", c.getActionPin("actions/github-script")),
		"        env:\n",
	}
	if err := c.addArtifactValidationPolicyEnvVar(&steps, data); err != nil {
		return nil, fmt.Errorf("CTR-033: artifact publication policy: %w", err)
	}
	steps = append(steps,
		"        with:\n",
		"          script: |\n",
		"            const { initializeArtifactPublication } = require('${{ runner.temp }}/gh-aw/actions/artifact_validation.cjs');\n",
		"            await initializeArtifactPublication({ core, github, context });\n",
	)
	return steps, nil
}

func artifactApprovalCondition() ConditionNode {
	return BuildAnd(
		BuildAnd(
			BuildEquals(BuildPropertyAccess("needs.artifact_validation.result"), BuildStringLiteral("success")),
			BuildNotEquals(BuildPropertyAccess("needs.artifact_validation.outputs.artifact_id"), BuildStringLiteral("")),
		),
		BuildNotEquals(BuildPropertyAccess("needs.artifact_validation.outputs.approval_digest"), BuildStringLiteral("")),
	)
}
