// @ts-check
/// <reference types="@actions/github-script" />

const fs = require("fs");
const { getErrorMessage } = require("./error_helpers.cjs");
const { ERR_API } = require("./error_codes.cjs");
const { sanitizeContent } = require("./sanitize_content.cjs");
const { generateFooterWithExpiration } = require("./ephemerals.cjs");
const { renderTemplateFromFile, getPromptPath } = require("./messages_core.cjs");
const { getEffectiveTokensSuffix } = require("./effective_tokens.cjs");

/**
 * Search for or create the parent issue for all agentic workflow detection runs.
 * @returns {Promise<{number: number, node_id: string}>} Parent issue number and node ID
 */
async function ensureDetectionRunsIssue() {
  const { owner, repo } = context.repo;
  const parentTitle = "[aw] Detection Runs";
  const parentLabel = "agentic-workflows";

  core.info(`Searching for detection runs issue: "${parentTitle}"`);

  // Search for existing detection runs issue
  const searchQuery = `repo:${owner}/${repo} is:issue is:open label:${parentLabel} in:title "${parentTitle}"`;

  try {
    const { data } = await github.rest.search.issuesAndPullRequests({
      q: searchQuery,
      per_page: 1,
    });

    if (data.total_count > 0) {
      const existingIssue = data.items[0];
      core.info(`Found existing detection runs issue #${existingIssue.number}: ${existingIssue.html_url}`);

      return {
        number: existingIssue.number,
        node_id: existingIssue.node_id,
      };
    }
  } catch (error) {
    throw new Error(`${ERR_API}: Failed to search for existing detection runs issue: ${getErrorMessage(error)}`);
  }

  // Create detection runs issue if it doesn't exist
  core.info(`No detection runs issue found, creating one`);

  // Load template from file
  const templatePath = getPromptPath("detection_runs_issue.md");
  const parentBodyContent = fs.readFileSync(templatePath, "utf8");

  const parentBody = generateFooterWithExpiration({
    footerText: parentBodyContent,
    expiresHours: 24 * 30, // 30 days
  });

  const { data: newIssue } = await github.rest.issues.create({
    owner,
    repo,
    title: parentTitle,
    body: parentBody,
    labels: [parentLabel],
  });

  core.info(`✓ Created detection runs issue #${newIssue.number}: ${newIssue.html_url}`);
  return {
    number: newIssue.number,
    node_id: newIssue.node_id,
  };
}

/**
 * Log detection problems (warnings and failures) as comments on a tracking issue.
 * Similar to the noop handler, this step posts a comment to the "[aw] Detection Runs"
 * issue when threat detection produces a non-success conclusion.
 */
async function main() {
  try {
    const detectionConclusion = process.env.GH_AW_DETECTION_CONCLUSION || "";
    const detectionReason = process.env.GH_AW_DETECTION_REASON || "";
    const workflowName = process.env.GH_AW_WORKFLOW_NAME || "unknown";
    const runUrl = process.env.GH_AW_RUN_URL || "";

    core.info(`Detection conclusion: ${detectionConclusion}`);
    core.info(`Detection reason: ${detectionReason}`);
    core.info(`Workflow name: ${workflowName}`);
    core.info(`Run URL: ${runUrl}`);

    // Only log when detection produced a warning or failure
    if (detectionConclusion !== "warning" && detectionConclusion !== "failure") {
      core.info(`Detection conclusion is "${detectionConclusion}", no issue comment needed`);
      return;
    }

    core.info(`Detection problem detected (${detectionConclusion}), posting to tracking issue`);

    const { owner, repo } = context.repo;

    // Ensure detection runs issue exists
    let detectionRunsIssue;
    try {
      detectionRunsIssue = await ensureDetectionRunsIssue();
    } catch (error) {
      core.warning(`Could not create detection runs issue: ${getErrorMessage(error)}`);
      // Don't fail the workflow if we can't create the issue
      return;
    }

    // Load and render comment template from file
    const commentTemplatePath = getPromptPath("detection_runs_comment.md");

    // Compute effective tokens suffix from environment variable (set by parse_token_usage.cjs / parse_mcp_gateway_log.cjs)
    const effectiveTokensSuffix = getEffectiveTokensSuffix();

    const commentBody = renderTemplateFromFile(commentTemplatePath, {
      workflow_name: workflowName,
      conclusion: detectionConclusion,
      reason: detectionReason || "unknown",
      run_url: runUrl,
      effective_tokens_suffix: effectiveTokensSuffix,
    });

    // Sanitize the full comment body
    const fullCommentBody = sanitizeContent(commentBody, { maxLength: 65000 });

    try {
      await github.rest.issues.createComment({
        owner,
        repo,
        issue_number: detectionRunsIssue.number,
        body: fullCommentBody,
      });

      core.info(`✓ Posted detection run comment to issue #${detectionRunsIssue.number}`);
    } catch (error) {
      core.warning(`Failed to post comment to detection runs issue: ${getErrorMessage(error)}`);
      // Don't fail the workflow
    }
  } catch (error) {
    core.warning(`Error in handle_detection_runs: ${getErrorMessage(error)}`);
    // Don't fail the workflow
  }
}

module.exports = { main, ensureDetectionRunsIssue };
