// Package workflow - content_redaction job assembler.
package workflow

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
)

var contentRedactionJobLog = logger.New("workflow:content_redaction_job")

// buildContentRedactionJob creates a separate content_redaction job that runs after the
// detection job (when threat detection is enabled) or after the agent job (when only content
// redaction is enabled). The job downloads the agent output, loads the configured policies,
// runs a fresh-context AI agent to review and rewrite text-bearing items, and outputs
// redaction_success and redaction_conclusion for downstream jobs.
// Returns nil if content redaction is not configured.
func (c *Compiler) buildContentRedactionJob(data *WorkflowData, detectionEnabled bool) (*Job, error) {
	contentRedactionJobLog.Print("Building separate content_redaction job")

	if !IsContentRedactionEnabled(data.SafeOutputs) && !IsConditionalContentRedaction(data.SafeOutputs) {
		contentRedactionJobLog.Print("Content redaction not configured, skipping content_redaction job")
		return nil, nil
	}

	cr := data.SafeOutputs.ContentRedaction

	var steps []string

	// Add setup action steps (same as other downstream jobs).
	setupActionRef := c.resolveActionReference("./actions/setup", data)
	if setupActionRef != "" || c.actionMode.IsScript() {
		steps = append(steps, c.generateCheckoutActionsFolder(data)...)
		redactionTraceID := fmt.Sprintf("${{ needs.%s.outputs.setup-trace-id }}", constants.ActivationJobName)
		redactionParentSpanID := setupParentSpanNeedsExpr(constants.ActivationJobName)
		steps = append(steps, c.generateSetupStep(data, setupActionRef, SetupActionDestination, false, redactionTraceID, redactionParentSpanID)...)
	}

	// Download agent output artifact to access output files.
	agentArtifactPrefix := artifactPrefixExprForAgentDownstreamJob(data)
	steps = append(steps, buildAgentOutputDownloadSteps(agentArtifactPrefix, c.getActionPin)...)

	// Build policy-loading steps for each configured source.
	steps = append(steps, buildLoadRedactionPoliciesStep(cr)...)

	// Build the main redaction step (github-script step).
	steps = append(steps, c.buildContentRedactionEngineStep(cr)...)

	// Build the conclusion step that sets job outputs.
	steps = append(steps, buildContentRedactionConclusionStep(cr)...)

	// Build job condition.
	// The job should run after the agent completes (not skipped) and after detection (if enabled).
	alwaysFunc := BuildFunctionCall("always")
	agentNotSkipped := BuildNotEquals(
		BuildPropertyAccess(fmt.Sprintf("needs.%s.result", constants.AgentJobName)),
		BuildStringLiteral("skipped"),
	)
	jobConditionNode := BuildAnd(alwaysFunc, agentNotSkipped)

	// Add detection success condition when detection is enabled.
	if detectionEnabled {
		jobConditionNode = BuildAnd(jobConditionNode, buildDetectionPassedCondition())
	}

	// When content redaction is expression-controlled, add the runtime expression.
	if cr != nil && cr.IfExpr != nil {
		rawExpr := extractRawExpression(*cr.IfExpr)
		jobConditionNode = BuildAnd(jobConditionNode, &ExpressionNode{Expression: rawExpr})
		contentRedactionJobLog.Printf("Content redaction job condition includes runtime expression: %s", rawExpr)
	}

	jobCondition := RenderCondition(jobConditionNode)

	// Build dependencies.
	needs := []string{string(constants.AgentJobName), string(constants.ActivationJobName)}
	if detectionEnabled {
		needs = append(needs, string(constants.DetectionJobName))
		contentRedactionJobLog.Print("Added detection job dependency to content_redaction job")
	}

	// Determine runs-on.
	runsOn := "runs-on: ubuntu-latest"
	if cr != nil && cr.RunsOn != "" {
		runsOn = normalizeRunsOnSnippet(cr.RunsOn)
	}

	// Determine permissions.
	perms := NewPermissionsContentsRead()
	if hasCopilotRequestsWritePermission(data) {
		perms.Set(PermissionCopilotRequests, PermissionWrite)
	}
	permissions := perms.RenderToYAML()

	// Determine environment.
	environment := data.Environment

	// Build job outputs.
	outputs := map[string]string{
		"redaction_success":    "${{ steps.redaction_conclusion.outputs.success }}",
		"redaction_conclusion": "${{ steps.redaction_conclusion.outputs.conclusion }}",
	}

	// Determine continue-on-error setting.
	var continueOnError *bool
	if cr != nil && cr.ContinueOnError != nil {
		continueOnError = cr.ContinueOnError
		if *continueOnError {
			contentRedactionJobLog.Print("Content redaction job will continue on error (failures emit warnings)")
		}
	}

	job := &Job{
		Name:            string(constants.ContentRedactionJobName),
		Needs:           needs,
		If:              jobCondition,
		RunsOn:          c.indentYAMLLines(runsOn, "    "),
		Environment:     c.indentYAMLLines(environment, "    "),
		Permissions:     permissions,
		Steps:           steps,
		Outputs:         outputs,
		ContinueOnError: continueOnError,
	}

	contentRedactionJobLog.Printf("Built content_redaction job with %d steps, depends on: %v", len(steps), needs)
	return job, nil
}

// buildLoadRedactionPoliciesStep creates steps that load the policy text from all configured
// sources into a concatenated policy file at /tmp/gh-aw/content-redaction/policy.md.
// Sources are processed in order: URLs are fetched via curl, repo-relative paths are read
// from the workspace checkout, and inline strings are written directly.
func buildLoadRedactionPoliciesStep(cr *ContentRedactionConfig) []string {
	if cr == nil || len(cr.Policies) == 0 {
		return []string{
			"      - name: Load redaction policies (no-op)\n",
			"        id: load_policies\n",
			"        run: mkdir -p /tmp/gh-aw/content-redaction && echo '' > /tmp/gh-aw/content-redaction/policy.md\n",
		}
	}

	var sb strings.Builder
	sb.WriteString("      - name: Load redaction policies\n")
	sb.WriteString("        id: load_policies\n")
	sb.WriteString("        env:\n")
	sb.WriteString("          GH_TOKEN: ${{ github.token }}\n")
	sb.WriteString("        run: |\n")
	sb.WriteString("          set -euo pipefail\n")
	sb.WriteString("          mkdir -p /tmp/gh-aw/content-redaction\n")
	sb.WriteString("          POLICY_FILE=/tmp/gh-aw/content-redaction/policy.md\n")
	sb.WriteString("          : > \"$POLICY_FILE\"\n")

	for i, source := range cr.Policies {
		comment := source
		if len(comment) > 60 {
			comment = comment[:60] + "..."
		}
		fmt.Fprintf(&sb, "          # Policy source %d: %s\n", i+1, comment)
		if strings.HasPrefix(source, "http://") || strings.HasPrefix(source, "https://") {
			// URL: fetch at runtime via curl.
			fmt.Fprintf(&sb, "          if ! curl -fsSL %q >> \"$POLICY_FILE\"; then\n", source)
			fmt.Fprintf(&sb, "            echo '::error::Content redaction: failed to fetch policy from %s'\n", source)
			fmt.Fprintf(&sb, "            exit 1\n")
			fmt.Fprintf(&sb, "          fi\n")
		} else if strings.HasPrefix(source, "./") || strings.HasPrefix(source, ".github/") || strings.HasPrefix(source, "/") {
			// Repo-relative or absolute path.
			fmt.Fprintf(&sb, "          if [ ! -f %q ]; then\n", source)
			fmt.Fprintf(&sb, "            echo '::error::Content redaction: policy file not found: %s'\n", source)
			fmt.Fprintf(&sb, "            exit 1\n")
			fmt.Fprintf(&sb, "          fi\n")
			fmt.Fprintf(&sb, "          cat %q >> \"$POLICY_FILE\"\n", source)
		} else {
			// Inline text literal: write directly.
			escaped := strings.ReplaceAll(source, "'", "'\\''")
			fmt.Fprintf(&sb, "          printf '%%s\\n' '%s' >> \"$POLICY_FILE\"\n", escaped)
		}
		sb.WriteString("          echo '' >> \"$POLICY_FILE\"\n")
	}

	sb.WriteString("          # Verify policy file is not empty\n")
	sb.WriteString("          if [ ! -s \"$POLICY_FILE\" ]; then\n")
	sb.WriteString("            echo '::error::Content redaction: all policy sources failed or were empty'\n")
	sb.WriteString("            exit 1\n")
	sb.WriteString("          fi\n")
	sb.WriteString("          echo \"Policy loaded ($(wc -c < \"$POLICY_FILE\") bytes)\"\n")

	return []string{sb.String()}
}

// buildContentRedactionEngineStep creates the main redaction step that executes
// the content_redaction.cjs script with AWF. It reads the agent output items,
// applies the loaded policy, and writes the redacted output back for safe_outputs to consume.
func (c *Compiler) buildContentRedactionEngineStep(cr *ContentRedactionConfig) []string {
	// Determine the model to use (with fallback to a cost-effective default).
	model := "gpt-4o-mini"
	if cr != nil && cr.Model != "" {
		model = cr.Model
	}

	onFailure := ContentRedactionOnFailureBlock
	if cr != nil && cr.OnFailure == ContentRedactionOnFailureWarn {
		onFailure = ContentRedactionOnFailureWarn
	}

	// Build the scope list as a JSON string literal for env var.
	scopeJSONString := "[]"
	if cr != nil && len(cr.Scope) > 0 {
		quoted := make([]string, len(cr.Scope))
		for i, s := range cr.Scope {
			quoted[i] = fmt.Sprintf("\"%s\"", s)
		}
		scopeJSONString = "[" + strings.Join(quoted, ", ") + "]"
	}

	var steps []string
	steps = append(steps, "      - name: Run content redaction\n")
	steps = append(steps, "        id: run_redaction\n")
	steps = append(steps, "        if: always() && steps.load_policies.outcome == 'success'\n")
	steps = append(steps, "        env:\n")
	steps = append(steps, "          GH_AW_AGENT_OUTPUT: ${{ steps.setup-agent-output-env.outputs.GH_AW_AGENT_OUTPUT }}\n")
	steps = append(steps, "          POLICY_FILE: /tmp/gh-aw/content-redaction/policy.md\n")
	steps = append(steps, "          REDACTION_MODEL: "+model+"\n")
	steps = append(steps, "          ON_FAILURE: "+onFailure+"\n")
	steps = append(steps, "          SCOPE: '"+scopeJSONString+"'\n")
	steps = append(steps, "        run: |\n")
	steps = append(steps, "          set -e\n")
	steps = append(steps, "          # Resolve node executable\n")
	steps = append(steps, "          GH_AW_NODE_EXEC=\"${GH_AW_NODE_BIN:-}\"\n")
	steps = append(steps, "          if [ -z \"$GH_AW_NODE_EXEC\" ] || [ ! -x \"$GH_AW_NODE_EXEC\" ]; then\n")
	steps = append(steps, "            GH_AW_NODE_EXEC=\"$(command -v node 2>/dev/null || true)\"\n")
	steps = append(steps, "          fi\n")
	steps = append(steps, "          if [ -z \"$GH_AW_NODE_EXEC\" ]; then\n")
	steps = append(steps, "            echo \"::error::node runtime missing on this runner — check runtimes.node in workflow YAML\"\n")
	steps = append(steps, "            exit 127\n")
	steps = append(steps, "          fi\n")
	steps = append(steps, "          # Configure AWF for content redaction\n")
	steps = append(steps, "          AWF_CONFIG_FILE=\"${RUNNER_TEMP}/gh-aw/content-redaction-awf-config.json\"\n")
	steps = append(steps, c.buildContentRedactionAWFConfigSetup())
	steps = append(steps, "          # Execute content redaction script via AWF\n")
	steps = append(steps, "          awf --config \"${AWF_CONFIG_FILE}\" \\\n")
	steps = append(steps, "            --container-workdir \"${GITHUB_WORKSPACE}\" \\\n")
	steps = append(steps, "            --mount \"${RUNNER_TEMP}/gh-aw:${RUNNER_TEMP}/gh-aw:rw\" \\\n")
	steps = append(steps, "            --mount \"/tmp/gh-aw:/tmp/gh-aw:rw\" \\\n")
	steps = append(steps, "            --env-all \\\n")
	steps = append(steps, "            --log-level info \\\n")
	steps = append(steps, "            --skip-pull \\\n")
	steps = append(steps, "            -- bash -c \"\\\"\\$GH_AW_NODE_EXEC\\\" \\\"\\${RUNNER_TEMP}/gh-aw/actions/content_redaction.cjs\\\"\"\n")

	return steps
}

// buildContentRedactionAWFConfigSetup generates the shell script to create AWF config JSON
// for content redaction. This config enables minimal network access needed for AI model API calls.
func (c *Compiler) buildContentRedactionAWFConfigSetup() string {
	// Build minimal AWF config for content redaction:
	// - Enable API proxy for model access
	// - Allow minimal required domains
	// - Use standard firewall version
	config := map[string]interface{}{
		"apiProxy": map[string]interface{}{
			"enabled": true,
		},
	}

	// Marshal as compact JSON (single line) to avoid YAML parsing issues
	configJSON, err := json.Marshal(config)
	if err != nil {
		contentRedactionJobLog.Printf("Warning: failed to marshal AWF config for content redaction: %v", err)
		// Fallback to minimal valid config
		configJSON = []byte(`{"apiProxy":{"enabled":true}}`)
	}

	// Escape the JSON for shell (preserve $ for env vars)
	escaped := shellEscapeArg(string(configJSON))
	return fmt.Sprintf("          printf '%%s\\n' %s > \"${AWF_CONFIG_FILE}\"\n", escaped)
}

// buildContentRedactionConclusionStep creates the step that sets the job conclusion outputs.
func buildContentRedactionConclusionStep(cr *ContentRedactionConfig) []string {
	onFailure := ContentRedactionOnFailureBlock
	if cr != nil && cr.OnFailure == ContentRedactionOnFailureWarn {
		onFailure = ContentRedactionOnFailureWarn
	}

	return []string{
		"      - name: Set redaction conclusion\n",
		"        id: redaction_conclusion\n",
		"        if: always()\n",
		"        env:\n",
		"          HAS_BLOCKED: ${{ steps.run_redaction.outputs.has_blocked_items }}\n",
		"          SKIPPED: ${{ steps.run_redaction.outputs.skipped }}\n",
		"        run: |\n",
		"          if [[ \"$SKIPPED\" == \"true\" ]]; then\n",
		"            echo \"success=true\" >> \"$GITHUB_OUTPUT\"\n",
		"            echo \"conclusion=skipped\" >> \"$GITHUB_OUTPUT\"\n",
		fmt.Sprintf("          elif [[ \"$HAS_BLOCKED\" == \"true\" && \"%s\" == \"%s\" ]]; then\n", onFailure, ContentRedactionOnFailureBlock),
		"            echo \"success=false\" >> \"$GITHUB_OUTPUT\"\n",
		"            echo \"conclusion=blocked\" >> \"$GITHUB_OUTPUT\"\n",
		"            echo \"::error::Content redaction blocked items that could not be made policy-compliant\"\n",
		"            exit 1\n",
		"          else\n",
		"            echo \"success=true\" >> \"$GITHUB_OUTPUT\"\n",
		"            echo \"conclusion=passed\" >> \"$GITHUB_OUTPUT\"\n",
		"          fi\n",
	}
}
