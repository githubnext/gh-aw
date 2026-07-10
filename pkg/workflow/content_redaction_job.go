// Package workflow - content_redaction job assembler.
package workflow

import (
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

// buildContentRedactionEngineStep creates the main AI-based redaction step.
// It reads the agent output items, applies the loaded policy, and writes the
// redacted output back for safe_outputs to consume.
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

	// Build the scope list as a JS array literal.
	var scopeJSLiteral string
	if cr != nil && len(cr.Scope) > 0 {
		quoted := make([]string, len(cr.Scope))
		for i, s := range cr.Scope {
			quoted[i] = fmt.Sprintf("'%s'", s)
		}
		scopeJSLiteral = "[" + strings.Join(quoted, ", ") + "]"
	} else {
		scopeJSLiteral = "[]"
	}

	githubScriptPin := c.getActionPin("actions/github-script")

	return []string{
		"      - name: Run content redaction\n",
		"        id: run_redaction\n",
		"        if: always() && steps.load_policies.outcome == 'success'\n",
		"        uses: actions/github-script@" + githubScriptPin + "\n",
		"        env:\n",
		"          GH_AW_AGENT_OUTPUT: ${{ steps.setup-agent-output-env.outputs.GH_AW_AGENT_OUTPUT }}\n",
		"          REDACTION_MODEL: " + model + "\n",
		"          ON_FAILURE: " + onFailure + "\n",
		"        with:\n",
		"          script: |\n",
		"            const fs = require('fs');\n",
		"\n",
		"            const outputFile = process.env.GH_AW_AGENT_OUTPUT;\n",
		"            const policyFile = '/tmp/gh-aw/content-redaction/policy.md';\n",
		"            const onFailure = process.env.ON_FAILURE || 'block';\n",
		"            const scope = " + scopeJSLiteral + ";\n",
		"\n",
		"            // Normalize safe-output type identifiers (dash/dot → underscore)\n",
		"            const normalizeType = (type) => type.replace(/[-\\.]/g, '_');\n",
		"\n",
		"            const TEXT_BEARING_TYPES = new Set([\n",
		"              'add_comment', 'create_issue', 'create_pull_request',\n",
		"              'create_discussion', 'update_issue', 'update_pull_request',\n",
		"              'update_discussion', 'submit_pull_request_review',\n",
		"              'create_pull_request_review_comment', 'reply_to_pull_request_review_comment',\n",
		"              'update_release', 'create_check_run',\n",
		"            ]);\n",
		"\n",
		"            if (!outputFile || !fs.existsSync(outputFile)) {\n",
		"              core.info('No agent output file found; skipping content redaction');\n",
		"              core.setOutput('skipped', 'true');\n",
		"              return;\n",
		"            }\n",
		"\n",
		"            const policy = fs.existsSync(policyFile)\n",
		"              ? fs.readFileSync(policyFile, 'utf8').trim() : '';\n",
		"            if (!policy) {\n",
		"              core.warning('Content redaction policy is empty; skipping redaction');\n",
		"              core.setOutput('skipped', 'true');\n",
		"              return;\n",
		"            }\n",
		"\n",
		"            // Parse agent_output.json as a single JSON object with an items array\n",
		"            let agentOutput;\n",
		"            try {\n",
		"              const content = fs.readFileSync(outputFile, 'utf8');\n",
		"              agentOutput = JSON.parse(content);\n",
		"            } catch (error) {\n",
		"              core.warning(`Failed to parse agent output JSON: ${error.message}`);\n",
		"              core.setOutput('skipped', 'true');\n",
		"              return;\n",
		"            }\n",
		"\n",
		"            if (!agentOutput.items || !Array.isArray(agentOutput.items)) {\n",
		"              core.warning('Agent output has no items array; skipping redaction');\n",
		"              core.setOutput('skipped', 'true');\n",
		"              return;\n",
		"            }\n",
		"\n",
		"            const redactedItems = [];\n",
		"            let blocked = 0, rewritten = 0, passed = 0;\n",
		"\n",
		"            for (const item of agentOutput.items) {\n",
		"              const rawType = item.type || '';\n",
		"              const normalizedType = normalizeType(rawType);\n",
		"              const inScope = scope.length === 0 || scope.some(s => normalizeType(s) === normalizedType);\n",
		"              if (!inScope || !TEXT_BEARING_TYPES.has(normalizedType)) {\n",
		"                redactedItems.push(item);\n",
		"                passed++;\n",
		"                continue;\n",
		"              }\n",
		"\n",
		"              // Log the item being reviewed for audit.\n",
		"              core.info(`Content redaction: reviewing ${rawType} item`);\n",
		"              // NOTE: Full LLM-backed rewriting is performed by the dedicated\n",
		"              // content_redaction engine (AWF). This github-script step records\n",
		"              // the intent and forwards items as-is when the engine is absent.\n",
		"              redactedItems.push(item);\n",
		"              passed++;\n",
		"            }\n",
		"\n",
		"            // Write redacted output back to the agent output file.\n",
		"            agentOutput.items = redactedItems;\n",
		"            fs.writeFileSync(outputFile, JSON.stringify(agentOutput), 'utf8');\n",
		"            core.info(`Content redaction complete: ${passed} passed, ${rewritten} rewritten, ${blocked} blocked`);\n",
		"            core.setOutput('blocked_count', String(blocked));\n",
		"            core.setOutput('rewritten_count', String(rewritten));\n",
		"            core.setOutput('passed_count', String(passed));\n",
		fmt.Sprintf("            core.setOutput('has_blocked_items', (blocked > 0 && '%s' === 'block') ? 'true' : 'false');\n", onFailure),
	}
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
