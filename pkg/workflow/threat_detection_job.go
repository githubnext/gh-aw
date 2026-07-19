// Package workflow - top-level detection job assembler.
package workflow

import (
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/sliceutil"
)

// buildDetectionJob creates a separate detection job that runs after the agent job.
// The job downloads the agent artifact to access output files, then runs all threat detection
// steps. It outputs detection_success and detection_conclusion for downstream jobs.
// Returns nil if threat detection is not configured.
func (c *Compiler) buildDetectionJob(data *WorkflowData) (*Job, error) {
	threatLog.Print("Building separate detection job")
	if data.SafeOutputs == nil || data.SafeOutputs.ThreatDetection == nil {
		threatLog.Print("Threat detection not configured, skipping detection job")
		return nil, nil
	}

	// When the engine is explicitly disabled and there are no custom steps,
	// there is nothing to run in the detection job — skip it entirely.
	// The detection job would only create an empty detection.log and the parser
	// would correctly fail with "No THREAT_DETECTION_RESULT found".
	if !IsDetectionJobEnabled(data.SafeOutputs) {
		threatLog.Print("Threat detection engine disabled with no custom steps, skipping detection job")
		return nil, nil
	}

	var steps []string
	steps = append(steps, c.buildDetectionJobSetupSteps(data)...)
	steps = append(steps, c.buildDetectionJobArtifactSteps(data)...)
	detectionStepsContent := c.buildDetectionJobSteps(data)
	steps = append(steps, detectionStepsContent...)

	outputs := map[string]string{
		"detection_success":    "${{ steps.detection_conclusion.outputs.success }}",
		"detection_conclusion": "${{ steps.detection_conclusion.outputs.conclusion }}",
		"detection_reason":     "${{ steps.detection_conclusion.outputs.reason }}",
		"aic":                  "${{ steps.parse_detection_token_usage.outputs.aic }}",
	}

	needs := c.buildDetectionJobNeeds(data)
	runsOn := buildDetectionJobRunsOn(data)
	jobCondition := buildDetectionJobCondition(data)
	permissions := buildDetectionJobPermissions(data)
	environment := buildDetectionJobEnvironment(data)

	job := &Job{
		Name:        string(constants.DetectionJobName),
		Needs:       needs,
		If:          jobCondition,
		RunsOn:      c.indentYAMLLines(runsOn, "    "),
		Environment: c.indentYAMLLines(environment, "    "),
		Permissions: permissions,
		Steps:       steps,
		Outputs:     outputs,
	}

	threatLog.Printf("Built detection job with %d steps, depends on: %v", len(steps), needs)
	return job, nil
}

func (c *Compiler) buildDetectionJobSetupSteps(data *WorkflowData) []string {
	setupActionRef := c.resolveActionReference("./actions/setup", data)
	if setupActionRef == "" && !c.actionMode.IsScript() {
		return nil
	}
	var steps []string
	steps = append(steps, c.generateCheckoutActionsFolder(data)...)
	detectionTraceID := fmt.Sprintf("${{ needs.%s.outputs.setup-trace-id }}", constants.ActivationJobName)
	detectionParentSpanID := setupParentSpanNeedsExpr(constants.ActivationJobName)
	steps = append(steps, c.generateSetupStep(data, setupActionRef, SetupActionDestination, false, detectionTraceID, detectionParentSpanID)...)
	return steps
}

func (c *Compiler) buildDetectionJobArtifactSteps(data *WorkflowData) []string {
	var steps []string
	agentArtifactPrefix := artifactPrefixExprForAgentDownstreamJob(data)
	steps = append(steps, buildAgentOutputDownloadSteps(agentArtifactPrefix, c.getActionPin)...)
	steps = append(steps, buildExperimentArtifactDownloadSteps(data, c.getActionPin)...)
	steps = append(steps, c.buildWorkspaceCheckoutForDetectionStep(data)...)
	return steps
}

func (c *Compiler) buildDetectionJobNeeds(data *WorkflowData) []string {
	needs := []string{string(constants.AgentJobName), string(constants.ActivationJobName)}
	var detectionSpecificEnv map[string]string
	if data.SafeOutputs != nil && data.SafeOutputs.ThreatDetection != nil && data.SafeOutputs.ThreatDetection.EngineConfig != nil {
		detectionSpecificEnv = data.SafeOutputs.ThreatDetection.EngineConfig.Env
	}
	effectiveDetectionEnv := mergeThreatDetectionEngineEnv(data, detectionSpecificEnv)
	if len(effectiveDetectionEnv) > 0 {
		var engineEnvBuilder strings.Builder
		for _, envValue := range effectiveDetectionEnv {
			engineEnvBuilder.WriteByte('\n')
			engineEnvBuilder.WriteString(envValue)
		}
		engineEnvContent := engineEnvBuilder.String()
		hasNeedsReference := strings.Contains(engineEnvContent, "needs.")
		if len(data.Jobs) > 0 {
			engineEnvJobs := c.getReferencedCustomJobs(engineEnvContent, data.Jobs)
			for _, jobName := range engineEnvJobs {
				if isBuiltinJobName(jobName) {
					continue
				}
				if !slices.Contains(needs, jobName) {
					needs = append(needs, jobName)
					threatLog.Printf("Added custom job '%s' to detection needs because it's referenced in engine.env", jobName)
				}
			}
		}
		if hasNeedsReference {
			for _, builtinJobName := range sliceutil.SortedKeys(constants.KnownBuiltInJobNames) {
				if slices.Contains(needs, builtinJobName) {
					continue
				}
				if strings.Contains(engineEnvContent, fmt.Sprintf("needs.%s.", builtinJobName)) {
					warningMsg := fmt.Sprintf(
						"engine.env references built-in job '%s' in a detection-job needs expression. "+
							"Built-in jobs are managed by the compiler and cannot be added as direct detection dependencies; "+
							"this expression will silently evaluate to an empty string at runtime.",
						builtinJobName,
					)
					fmt.Fprintln(os.Stderr, console.FormatWarningMessage(warningMsg))
					c.IncrementWarningCount()
				}
			}
		}
	}
	return needs
}

func buildDetectionJobRunsOn(data *WorkflowData) string {
	runsOn := "runs-on: ubuntu-latest"
	if data.SafeOutputs.ThreatDetection.RunsOn != "" {
		runsOn = normalizeRunsOnSnippet(data.SafeOutputs.ThreatDetection.RunsOn)
	}
	return runsOn
}

func buildDetectionJobCondition(data *WorkflowData) string {
	alwaysFunc := BuildFunctionCall("always")
	agentNotSkipped := BuildNotEquals(
		BuildPropertyAccess(fmt.Sprintf("needs.%s.result", constants.AgentJobName)),
		BuildStringLiteral("skipped"),
	)
	jobConditionNode := BuildAnd(alwaysFunc, agentNotSkipped)

	// When detection is expression-controlled, add the caller expression to the condition so
	// GitHub Actions skips the detection job at runtime when the expression evaluates to false.
	if data.SafeOutputs.ThreatDetection.EnabledExpr != nil {
		rawExpr := extractRawExpression(*data.SafeOutputs.ThreatDetection.EnabledExpr)
		jobConditionNode = BuildAnd(jobConditionNode, &ExpressionNode{Expression: rawExpr})
		threatLog.Printf("Detection job condition includes runtime expression: %s", rawExpr)
	}
	return RenderCondition(jobConditionNode)
}

func buildDetectionJobPermissions(data *WorkflowData) string {
	copilotRequestsEnabled := hasCopilotRequestsWritePermission(data)
	perms := NewPermissionsContentsRead()
	if copilotRequestsEnabled {
		perms.Set(PermissionCopilotRequests, PermissionWrite)
	}
	if data.EngineConfig != nil && data.EngineConfig.Auth != nil && data.EngineConfig.Auth.Type == "github-oidc" {
		perms.Set(PermissionIdToken, PermissionWrite)
	}
	if hasOTLPGitHubOIDCAuth(data.ParsedFrontmatter, data.RawFrontmatter) {
		perms.Set(PermissionIdToken, PermissionWrite)
	}
	return perms.RenderToYAML()
}

func buildDetectionJobEnvironment(data *WorkflowData) string {
	environment := data.Environment
	if data.SafeOutputs.ThreatDetection.Environment != "" {
		environment = "environment: " + data.SafeOutputs.ThreatDetection.Environment
	}
	return environment
}
