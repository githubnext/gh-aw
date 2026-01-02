// @ts-check
/// <reference types="@actions/github-script" />

const { loadAgentOutput } = require("./load_agent_output.cjs");
const { generateFooter } = require("./generate_footer.cjs");
const { sanitizeContent } = require("./sanitize_content.cjs");

/**
 * Generate staged preview for mark-pull-request-as-ready-for-review items
 * @param {Array<any>} items - Items to preview
 */
async function generateStagedPreview(items) {
  const workflowName = process.env.GH_AW_WORKFLOW_NAME || "GitHub Agentic Workflow";
  const runUrl = `${context.serverUrl}/${context.repo.owner}/${context.repo.repo}/actions/runs/${context.runId}`;

  let summaryContent = "## 🎭 Preview: Mark Pull Request as Ready for Review\n\n";
  summaryContent += `**Staged Mode**: The following ${items.length} pull request(s) would be marked as ready for review:\n\n`;

  for (let i = 0; i < items.length; i++) {
    const item = items[i];
    const prNumber = item.pull_request_number || context.payload?.pull_request?.number;

    summaryContent += `### Pull Request #${prNumber || "current"}\n\n`;
    summaryContent += `**Action**: Mark as ready for review (set draft=false)\n\n`;
    summaryContent += `**Comment**:\n\n`;
    summaryContent += "```markdown\n";
    summaryContent += item.reason;
    summaryContent += "\n```\n\n";
    summaryContent += "---\n\n";
  }

  await core.summary.addRaw(summaryContent).write();
}

/**
 * Mark a pull request as ready for review
 * @param {any} github - GitHub REST API instance
 * @param {string} owner - Repository owner
 * @param {string} repo - Repository name
 * @param {number} prNumber - Pull request number
 * @param {string} reason - Comment explaining why PR is ready
 * @returns {Promise<{number: number, html_url: string, title: string}>} Pull request details
 */
async function markPullRequestAsReadyForReview(github, owner, repo, prNumber, reason) {
  // First, get the current PR to check if it's a draft
  const { data: currentPR } = await github.rest.pulls.get({
    owner,
    repo,
    pull_number: prNumber,
  });

  if (!currentPR) {
    throw new Error(`Pull request #${prNumber} not found in ${owner}/${repo}`);
  }

  // Check if it's already not a draft
  if (!currentPR.draft) {
    core.info(`Pull request #${prNumber} is already marked as ready for review (not a draft)`);
    return currentPR;
  }

  // Update the PR to mark as ready for review
  const { data: pr } = await github.rest.pulls.update({
    owner,
    repo,
    pull_number: prNumber,
    draft: false,
  });

  // Add comment with reason
  const workflowName = process.env.GH_AW_WORKFLOW_NAME || "GitHub Agentic Workflow";
  const runUrl = `${context.serverUrl}/${owner}/${repo}/actions/runs/${context.runId}`;
  const workflowSource = process.env.GH_AW_WORKFLOW_SOURCE || "";
  const workflowSourceURL = process.env.GH_AW_WORKFLOW_SOURCE_URL || "";

  // Extract triggering context for footer generation
  const triggeringIssueNumber = context.payload?.issue?.number && !context.payload?.issue?.pull_request ? context.payload.issue.number : undefined;
  const triggeringPRNumber = context.payload?.pull_request?.number || (context.payload?.issue?.pull_request ? context.payload.issue.number : undefined);
  const triggeringDiscussionNumber = context.payload?.discussion?.number;

  const sanitizedReason = sanitizeContent(reason);
  const footer = generateFooter(workflowName, runUrl, workflowSource, workflowSourceURL, triggeringIssueNumber, triggeringPRNumber, triggeringDiscussionNumber);
  const commentBody = `${sanitizedReason}\n\n${footer}`;

  await github.rest.issues.createComment({
    owner,
    repo,
    issue_number: prNumber,
    body: commentBody,
  });

  core.info(`✓ Marked PR #${prNumber} as ready for review and added comment`);

  return pr;
}

async function main() {
  // Check if we're in staged mode
  const isStaged = process.env.GH_AW_SAFE_OUTPUTS_STAGED === "true";

  const result = loadAgentOutput();
  if (!result.success) {
    return;
  }

  // Find all mark_pull_request_as_ready_for_review items
  const items = result.items.filter(item => item.type === "mark_pull_request_as_ready_for_review");

  if (items.length === 0) {
    core.info("No mark_pull_request_as_ready_for_review items found in agent output");
    return;
  }

  core.info(`Found ${items.length} mark_pull_request_as_ready_for_review item(s)`);

  // If in staged mode, show preview
  if (isStaged) {
    await generateStagedPreview(items);
    return;
  }

  // Process each item
  const isPRContext = context.eventName === "pull_request" || context.eventName === "pull_request_target";

  for (let i = 0; i < items.length; i++) {
    const item = items[i];
    core.info(`Processing item ${i + 1}/${items.length}`);

    // Determine PR number
    let prNumber = item.pull_request_number;
    if (!prNumber) {
      if (!isPRContext) {
        core.warning("No pull_request_number specified and not in PR context, skipping");
        continue;
      }
      prNumber = context.payload?.pull_request?.number;
    }

    if (!prNumber) {
      core.warning("Could not determine pull request number, skipping");
      continue;
    }

    // Convert to number if string
    if (typeof prNumber === "string") {
      prNumber = parseInt(prNumber, 10);
    }

    // Validate reason
    if (!item.reason || typeof item.reason !== "string" || item.reason.trim().length === 0) {
      core.warning(`Item ${i + 1} has empty or invalid reason, skipping`);
      continue;
    }

    try {
      await markPullRequestAsReadyForReview(github, context.repo.owner, context.repo.repo, prNumber, item.reason);
    } catch (error) {
      const errorMessage = error instanceof Error ? error.message : String(error);
      core.error(`Failed to mark PR #${prNumber} as ready for review: ${errorMessage}`);
      core.setFailed(`Failed to mark PR #${prNumber} as ready for review: ${errorMessage}`);
    }
  }
}

module.exports = { main };
