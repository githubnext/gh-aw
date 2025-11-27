// @ts-check
/// <reference types="@actions/github-script" />

const { runSafeOutput } = require("./safe_output_runner.cjs");
const { generateFooter } = require("./generate_footer.cjs");
const { getTrackerID } = require("./get_tracker_id.cjs");
const { getRepositoryUrl } = require("./get_repository_url.cjs");

/**
 * Get issue details using REST API
 * @param {any} github - GitHub REST API instance
 * @param {string} owner - Repository owner
 * @param {string} repo - Repository name
 * @param {number} issueNumber - Issue number
 * @returns {Promise<{number: number, title: string, labels: Array<{name: string}>, html_url: string, state: string}>} Issue details
 */
async function getIssueDetails(github, owner, repo, issueNumber) {
  const { data: issue } = await github.rest.issues.get({
    owner,
    repo,
    issue_number: issueNumber,
  });

  if (!issue) {
    throw new Error(`Issue #${issueNumber} not found in ${owner}/${repo}`);
  }

  return issue;
}

/**
 * Add comment to a GitHub Issue using REST API
 * @param {any} github - GitHub REST API instance
 * @param {string} owner - Repository owner
 * @param {string} repo - Repository name
 * @param {number} issueNumber - Issue number
 * @param {string} message - Comment body
 * @returns {Promise<{id: number, html_url: string}>} Comment details
 */
async function addIssueComment(github, owner, repo, issueNumber, message) {
  const { data: comment } = await github.rest.issues.createComment({
    owner,
    repo,
    issue_number: issueNumber,
    body: message,
  });

  return comment;
}

/**
 * Close a GitHub Issue using REST API
 * @param {any} github - GitHub REST API instance
 * @param {string} owner - Repository owner
 * @param {string} repo - Repository name
 * @param {number} issueNumber - Issue number
 * @returns {Promise<{number: number, html_url: string}>} Issue details
 */
async function closeIssue(github, owner, repo, issueNumber) {
  const { data: issue } = await github.rest.issues.update({
    owner,
    repo,
    issue_number: issueNumber,
    state: "closed",
  });

  return issue;
}

/**
 * Render function for staged preview
 * @param {any} item - The close_issue item
 * @param {number} index - Index of the item
 * @returns {string} Markdown content for the preview
 */
function renderCloseIssuePreview(item, index) {
  // Get configuration from environment
  const requiredLabels = process.env.GH_AW_CLOSE_ISSUE_REQUIRED_LABELS
    ? process.env.GH_AW_CLOSE_ISSUE_REQUIRED_LABELS.split(",").map(l => l.trim())
    : [];
  const requiredTitlePrefix = process.env.GH_AW_CLOSE_ISSUE_REQUIRED_TITLE_PREFIX || "";

  let content = `### Issue ${index + 1}\n`;

  const issueNumber = item.issue_number;
  if (issueNumber) {
    const repoUrl = getRepositoryUrl();
    const issueUrl = `${repoUrl}/issues/${issueNumber}`;
    content += `**Target Issue:** [#${issueNumber}](${issueUrl})\n\n`;
  } else {
    content += `**Target:** Current issue\n\n`;
  }

  content += `**Comment:**\n${item.body || "No content provided"}\n\n`;

  if (requiredLabels.length > 0) {
    content += `**Required Labels:** ${requiredLabels.join(", ")}\n\n`;
  }
  if (requiredTitlePrefix) {
    content += `**Required Title Prefix:** ${requiredTitlePrefix}\n\n`;
  }

  return content;
}

/**
 * Process close_issue items
 * @param {any[]} closeIssueItems - The close_issue items to process
 */
async function processCloseIssueItems(closeIssueItems) {
  // Get configuration from environment
  const requiredLabels = process.env.GH_AW_CLOSE_ISSUE_REQUIRED_LABELS
    ? process.env.GH_AW_CLOSE_ISSUE_REQUIRED_LABELS.split(",").map(l => l.trim())
    : [];
  const requiredTitlePrefix = process.env.GH_AW_CLOSE_ISSUE_REQUIRED_TITLE_PREFIX || "";
  const target = process.env.GH_AW_CLOSE_ISSUE_TARGET || "triggering";

  core.info(`Configuration: requiredLabels=${requiredLabels.join(",")}, requiredTitlePrefix=${requiredTitlePrefix}, target=${target}`);

  // Check if we're in an issue context
  const isIssueContext = context.eventName === "issues" || context.eventName === "issue_comment";

  // Validate context based on target configuration
  if (target === "triggering" && !isIssueContext) {
    core.info('Target is "triggering" but not running in issue context, skipping issue close');
    return;
  }

  // Extract triggering context for footer generation
  const triggeringIssueNumber = context.payload?.issue?.number;

  const closedIssues = [];

  // Process each close-issue item
  for (let i = 0; i < closeIssueItems.length; i++) {
    const item = closeIssueItems[i];
    core.info(`Processing close-issue item ${i + 1}/${closeIssueItems.length}: bodyLength=${item.body.length}`);

    // Determine the issue number
    let issueNumber;

    if (target === "*") {
      // For target "*", we need an explicit number from the item
      const targetNumber = item.issue_number;
      if (targetNumber) {
        issueNumber = parseInt(targetNumber, 10);
        if (isNaN(issueNumber) || issueNumber <= 0) {
          core.info(`Invalid issue number specified: ${targetNumber}`);
          continue;
        }
      } else {
        core.info(`Target is "*" but no issue_number specified in close-issue item`);
        continue;
      }
    } else if (target && target !== "triggering") {
      // Explicit number specified in target configuration
      issueNumber = parseInt(target, 10);
      if (isNaN(issueNumber) || issueNumber <= 0) {
        core.info(`Invalid issue number in target configuration: ${target}`);
        continue;
      }
    } else {
      // Default behavior: use triggering issue
      if (isIssueContext) {
        issueNumber = context.payload.issue?.number;
        if (!issueNumber) {
          core.info("Issue context detected but no issue found in payload");
          continue;
        }
      } else {
        core.info("Not in issue context and no explicit target specified");
        continue;
      }
    }

    try {
      // Fetch issue details to check filters
      const issue = await getIssueDetails(github, context.repo.owner, context.repo.repo, issueNumber);

      // Apply label filter
      if (requiredLabels.length > 0) {
        const issueLabels = issue.labels.map(l => l.name);
        const hasRequiredLabel = requiredLabels.some(required => issueLabels.includes(required));
        if (!hasRequiredLabel) {
          core.info(`Issue #${issueNumber} does not have required labels: ${requiredLabels.join(", ")}`);
          continue;
        }
      }

      // Apply title prefix filter
      if (requiredTitlePrefix && !issue.title.startsWith(requiredTitlePrefix)) {
        core.info(`Issue #${issueNumber} does not have required title prefix: ${requiredTitlePrefix}`);
        continue;
      }

      // Check if already closed
      if (issue.state === "closed") {
        core.info(`Issue #${issueNumber} is already closed, skipping`);
        continue;
      }

      // Build comment body with optional tracker ID
      const trackerID = getTrackerID();
      const footer = generateFooter(trackerID, triggeringIssueNumber);
      const commentBody = item.body + footer;

      // Add comment before closing
      const comment = await addIssueComment(github, context.repo.owner, context.repo.repo, issueNumber, commentBody);
      core.info(`✓ Added comment to issue #${issueNumber}: ${comment.html_url}`);

      // Close the issue
      const closedIssue = await closeIssue(github, context.repo.owner, context.repo.repo, issueNumber);
      core.info(`✓ Closed issue #${issueNumber}: ${closedIssue.html_url}`);

      closedIssues.push({
        issue: closedIssue,
        comment,
      });

      // Set outputs for the last closed issue (for backward compatibility)
      if (i === closeIssueItems.length - 1) {
        core.setOutput("issue_number", closedIssue.number);
        core.setOutput("issue_url", closedIssue.html_url);
        core.setOutput("comment_url", comment.html_url);
      }
    } catch (error) {
      core.error(`✗ Failed to close issue #${issueNumber}: ${error instanceof Error ? error.message : String(error)}`);
      throw error;
    }
  }

  // Write summary for all closed issues
  if (closedIssues.length > 0) {
    let summaryContent = "\n\n## Closed Issues\n";
    for (const { issue, comment } of closedIssues) {
      summaryContent += `- Issue #${issue.number}: [${issue.title}](${issue.html_url}) ([comment](${comment.html_url}))\n`;
    }
    await core.summary.addRaw(summaryContent).write();
  }

  core.info(`Successfully closed ${closedIssues.length} issue(s)`);
  return closedIssues;
}

async function main() {
  await runSafeOutput({
    itemType: "close_issue",
    itemTypePlural: "close-issue",
    stagedTitle: "Close Issues",
    stagedDescription: "The following issues would be closed if staged mode was disabled:",
    renderStagedItem: renderCloseIssuePreview,
    processItems: processCloseIssueItems,
  });
}

await main();
