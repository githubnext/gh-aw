// @ts-check
/// <reference types="@actions/github-script" />

const fs = require("fs");
const { getErrorMessage } = require("./error_helpers.cjs");
const { ERR_API } = require("./error_codes.cjs");
const { sanitizeContent } = require("./sanitize_content.cjs");
const { generateFooterWithExpiration } = require("./ephemerals.cjs");
const { renderTemplateFromFile } = require("./messages_core.cjs");
const { loadAgentOutput } = require("./load_agent_output.cjs");
const { isStagedMode } = require("./safe_output_helpers.cjs");

/**
 * Search for or create the parent issue for all agentic workflow no-op runs
 * @returns {Promise<{number: number, node_id: string}>} Parent issue number and node ID
 */
async function ensureAgentRunsIssue() {
  const { owner, repo } = context.repo;
  const parentTitle = "[aw] No-Op Runs";
  const parentLabel = "agentic-workflows";

  core.info(`Searching for no-op runs issue: "${parentTitle}"`);

  // Search for existing no-op runs issue
  const searchQuery = `repo:${owner}/${repo} is:issue is:open label:${parentLabel} in:title "${parentTitle}"`;

  try {
    const { data } = await github.rest.search.issuesAndPullRequests({
      q: searchQuery,
      per_page: 1,
    });

    if (data.total_count > 0) {
      const existingIssue = data.items[0];
      core.info(`Found existing no-op runs issue #${existingIssue.number}: ${existingIssue.html_url}`);

      return {
        number: existingIssue.number,
        node_id: existingIssue.node_id,
      };
    }
  } catch (error) {
    throw new Error(`${ERR_API}: Failed to search for existing no-op runs issue: ${getErrorMessage(error)}`);
  }

  // Create no-op runs issue if it doesn't exist
  core.info(`No no-op runs issue found, creating one`);

  // Load template from file
  const templatePath = `${process.env.RUNNER_TEMP}/gh-aw/prompts/noop_runs_issue.md`;
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

  core.info(`✓ Created no-op runs issue #${newIssue.number}: ${newIssue.html_url}`);
  return {
    number: newIssue.number,
    node_id: newIssue.node_id,
  };
}

/**
 * Process no-op safe outputs and optionally post to the no-op runs issue.
 * This merged step replaces the separate "Process no-op messages" + "Handle No-Op Message"
 * steps, eliminating the cross-step output dependency on GH_AW_NOOP_MESSAGE.
 *
 * Behaviour:
 * 1. Load noop items directly from the agent output artifact.
 * 2. In staged mode: write a summary preview and exit without posting.
 * 3. Otherwise: write a summary, set the `noop_message` step output, then post to the
 *    "[aw] No-Op Runs" tracking issue when the agent produced only noop outputs.
 */
async function main() {
  try {
    // --- Load and filter noop items from agent output ---
    const result = loadAgentOutput();
    if (!result.success) {
      core.info("Could not load agent output, skipping");
      return;
    }

    const maxCount = parseInt(process.env.GH_AW_NOOP_MAX || "0", 10);
    const allNoopItems = (result.items || []).filter(/** @param {any} item */ item => item.type === "noop");
    const noopItems = maxCount > 0 ? allNoopItems.slice(0, maxCount) : allNoopItems;

    if (noopItems.length === 0) {
      core.info("No noop items found in agent output");
      return;
    }

    core.info(`Found ${noopItems.length} noop item(s)`);
    const noopMessage = noopItems[0].message;

    // --- Staged mode: preview only, do not post ---
    if (isStagedMode()) {
      let summaryContent = "## 🎭 Staged Mode: No-Op Messages Preview\n\n";
      summaryContent += "The following messages would be logged if staged mode was disabled:\n\n";
      for (let i = 0; i < noopItems.length; i++) {
        const item = noopItems[i];
        summaryContent += `### Message ${i + 1}\n`;
        summaryContent += `${item.message}\n\n`;
        summaryContent += "---\n\n";
      }
      await core.summary.addRaw(summaryContent).write();
      core.info("📝 No-op message preview written to step summary");
      return;
    }

    // --- Write step summary ---
    let summaryContent = "\n\n## No-Op Messages\n\n";
    summaryContent += "The following messages were logged for transparency:\n\n";
    for (let i = 0; i < noopItems.length; i++) {
      const item = noopItems[i];
      core.info(`No-op message ${i + 1}: ${item.message}`);
      summaryContent += `- ${item.message}\n`;
    }
    await core.summary.addRaw(summaryContent).write();

    // Export for downstream steps/jobs
    core.setOutput("noop_message", noopMessage);
    core.info(`Successfully processed ${noopItems.length} noop message(s)`);

    // --- Post to no-op runs issue ---
    const workflowName = process.env.GH_AW_WORKFLOW_NAME || "unknown";
    const runUrl = process.env.GH_AW_RUN_URL || "";
    const agentConclusion = process.env.GH_AW_AGENT_CONCLUSION || "";
    const reportAsIssue = process.env.GH_AW_NOOP_REPORT_AS_ISSUE !== "false"; // Default to true

    core.info(`Workflow name: ${workflowName}`);
    core.info(`Run URL: ${runUrl}`);
    core.info(`Agent conclusion: ${agentConclusion}`);
    core.info(`Report as issue: ${reportAsIssue}`);

    if (!reportAsIssue) {
      core.info("report-as-issue is disabled (set to false), skipping no-op message posting to issue");
      return;
    }

    // Only post to "agent runs" issue if:
    // 1. The agent succeeded (agentConclusion === "success"), OR
    // 2. The agent failed but produced only noop outputs, which indicates a transient AI model
    //    error after the meaningful work (noop) was already captured. Skipped/cancelled runs
    //    and other non-success/non-failure conclusions are always skipped.
    if (agentConclusion !== "success" && agentConclusion !== "failure") {
      core.info(`Agent did not succeed (conclusion: ${agentConclusion}), skipping no-op message posting`);
      return;
    }

    // Skip posting when there are non-noop outputs (agent did real work)
    const nonNoopItems = result.items.filter(/** @param {any} item */ ({ type }) => type !== "noop");
    if (nonNoopItems.length > 0) {
      core.info(`Found ${nonNoopItems.length} non-noop output(s), skipping no-op message posting`);
      return;
    }

    if (agentConclusion === "failure") {
      core.info("Agent failed but produced only noop outputs (transient AI model error after noop was captured) - posting noop message");
    } else {
      core.info("Agent succeeded with only noop outputs - posting to no-op runs issue");
    }

    const { owner, repo } = context.repo;

    // Ensure no-op runs issue exists
    let noopRunsIssue;
    try {
      noopRunsIssue = await ensureAgentRunsIssue();
    } catch (error) {
      core.warning(`Could not create no-op runs issue: ${getErrorMessage(error)}`);
      // Don't fail the workflow if we can't create the issue
      return;
    }

    // Load and render comment template from file
    const commentTemplatePath = `${process.env.RUNNER_TEMP}/gh-aw/prompts/noop_comment.md`;
    const commentBody = renderTemplateFromFile(commentTemplatePath, {
      workflow_name: workflowName,
      message: noopMessage,
      run_url: runUrl,
    });

    // Sanitize the full comment body
    const fullCommentBody = sanitizeContent(commentBody, { maxLength: 65000 });

    try {
      await github.rest.issues.createComment({
        owner,
        repo,
        issue_number: noopRunsIssue.number,
        body: fullCommentBody,
      });

      core.info(`✓ Posted no-op message to no-op runs issue #${noopRunsIssue.number}`);
    } catch (error) {
      core.warning(`Failed to post comment to no-op runs issue: ${getErrorMessage(error)}`);
      // Don't fail the workflow
    }
  } catch (error) {
    core.warning(`Error in handle_noop_message: ${getErrorMessage(error)}`);
    // Don't fail the workflow
  }
}

module.exports = { main, ensureAgentRunsIssue };
