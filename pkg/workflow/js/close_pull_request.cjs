// @ts-check
/// <reference types="@actions/github-script" />

const { loadAgentOutput } = require("./load_agent_output.cjs");
const { generateFooter } = require("./generate_footer.cjs");
const { getTrackerID } = require("./get_tracker_id.cjs");
const { getRepositoryUrl } = require("./get_repository_url.cjs");

/**
 * Get pull request details using REST API
 * @param {any} github - GitHub REST API instance
 * @param {string} owner - Repository owner
 * @param {string} repo - Repository name
 * @param {number} prNumber - Pull request number
 * @returns {Promise<{number: number, title: string, labels: Array<{name: string}>, html_url: string, state: string}>} Pull request details
 */
async function getPullRequestDetails(github, owner, repo, prNumber) {
  const { data: pr } = await github.rest.pulls.get({
    owner,
    repo,
    pull_number: prNumber,
  });

  if (!pr) {
    throw new Error(`Pull request #${prNumber} not found in ${owner}/${repo}`);
  }

  return pr;
}

/**
 * Add comment to a GitHub Pull Request using REST API
 * @param {any} github - GitHub REST API instance
 * @param {string} owner - Repository owner
 * @param {string} repo - Repository name
 * @param {number} prNumber - Pull request number
 * @param {string} message - Comment body
 * @returns {Promise<{id: number, html_url: string}>} Comment details
 */
async function addPullRequestComment(github, owner, repo, prNumber, message) {
  const { data: comment } = await github.rest.issues.createComment({
    owner,
    repo,
    issue_number: prNumber,
    body: message,
  });

  return comment;
}

/**
 * Close a GitHub Pull Request using REST API
 * @param {any} github - GitHub REST API instance
 * @param {string} owner - Repository owner
 * @param {string} repo - Repository name
 * @param {number} prNumber - Pull request number
 * @returns {Promise<{number: number, html_url: string}>} Pull request details
 */
async function closePullRequest(github, owner, repo, prNumber) {
  const { data: pr } = await github.rest.pulls.update({
    owner,
    repo,
    pull_number: prNumber,
    state: "closed",
  });

  return pr;
}

async function main() {
  // Check if we're in staged mode
  const isStaged = process.env.GH_AW_SAFE_OUTPUTS_STAGED === "true";

  const result = loadAgentOutput();
  if (!result.success) {
    return;
  }

  // Find all close-pull-request items
  const closePRItems = result.items.filter(/** @param {any} item */ item => item.type === "close_pull_request");
  if (closePRItems.length === 0) {
    core.info("No close-pull-request items found in agent output");
    return;
  }

  core.info(`Found ${closePRItems.length} close-pull-request item(s)`);

  // Get configuration from environment
  const requiredLabels = process.env.GH_AW_CLOSE_PR_REQUIRED_LABELS
    ? process.env.GH_AW_CLOSE_PR_REQUIRED_LABELS.split(",").map(l => l.trim())
    : [];
  const requiredTitlePrefix = process.env.GH_AW_CLOSE_PR_REQUIRED_TITLE_PREFIX || "";
  const target = process.env.GH_AW_CLOSE_PR_TARGET || "triggering";

  core.info(`Configuration: requiredLabels=${requiredLabels.join(",")}, requiredTitlePrefix=${requiredTitlePrefix}, target=${target}`);

  // Check if we're in a pull request context
  const isPRContext = context.eventName === "pull_request" || context.eventName === "pull_request_review_comment";

  // If in staged mode, emit step summary instead of closing pull requests
  if (isStaged) {
    let summaryContent = "## 🎭 Staged Mode: Close Pull Requests Preview\n\n";
    summaryContent += "The following pull requests would be closed if staged mode was disabled:\n\n";

    for (let i = 0; i < closePRItems.length; i++) {
      const item = closePRItems[i];
      summaryContent += `### Pull Request ${i + 1}\n`;

      const prNumber = item.pull_request_number;
      if (prNumber) {
        const repoUrl = getRepositoryUrl();
        const prUrl = `${repoUrl}/pull/${prNumber}`;
        summaryContent += `**Target Pull Request:** [#${prNumber}](${prUrl})\n\n`;
      } else {
        summaryContent += `**Target:** Current pull request\n\n`;
      }

      summaryContent += `**Comment:**\n${item.body || "No content provided"}\n\n`;

      if (requiredLabels.length > 0) {
        summaryContent += `**Required Labels:** ${requiredLabels.join(", ")}\n\n`;
      }
      if (requiredTitlePrefix) {
        summaryContent += `**Required Title Prefix:** ${requiredTitlePrefix}\n\n`;
      }

      summaryContent += "---\n\n";
    }

    // Write to step summary
    await core.summary.addRaw(summaryContent).write();
    core.info("📝 Pull request close preview written to step summary");
    return;
  }

  // Validate context based on target configuration
  if (target === "triggering" && !isPRContext) {
    core.info('Target is "triggering" but not running in pull request context, skipping PR close');
    return;
  }

  // Extract triggering context for footer generation
  const triggeringPRNumber = context.payload?.pull_request?.number;

  const closedPRs = [];

  // Process each close-pull-request item
  for (let i = 0; i < closePRItems.length; i++) {
    const item = closePRItems[i];
    core.info(`Processing close-pull-request item ${i + 1}/${closePRItems.length}: bodyLength=${item.body.length}`);

    // Determine the pull request number
    let prNumber;

    if (target === "*") {
      // For target "*", we need an explicit number from the item
      const targetNumber = item.pull_request_number;
      if (targetNumber) {
        prNumber = parseInt(targetNumber, 10);
        if (isNaN(prNumber) || prNumber <= 0) {
          core.info(`Invalid pull request number specified: ${targetNumber}`);
          continue;
        }
      } else {
        core.info(`Target is "*" but no pull_request_number specified in close-pull-request item`);
        continue;
      }
    } else if (target && target !== "triggering") {
      // Explicit number specified in target configuration
      prNumber = parseInt(target, 10);
      if (isNaN(prNumber) || prNumber <= 0) {
        core.info(`Invalid pull request number in target configuration: ${target}`);
        continue;
      }
    } else {
      // Default behavior: use triggering pull request
      if (isPRContext) {
        prNumber = context.payload.pull_request?.number;
        if (!prNumber) {
          core.info("Pull request context detected but no pull request found in payload");
          continue;
        }
      } else {
        core.info("Not in pull request context and no explicit target specified");
        continue;
      }
    }

    try {
      // Fetch pull request details to check filters
      const pr = await getPullRequestDetails(github, context.repo.owner, context.repo.repo, prNumber);

      // Apply label filter
      if (requiredLabels.length > 0) {
        const prLabels = pr.labels.map(l => l.name);
        const hasRequiredLabel = requiredLabels.some(required => prLabels.includes(required));
        if (!hasRequiredLabel) {
          core.info(`Pull request #${prNumber} does not have required labels: ${requiredLabels.join(", ")}`);
          continue;
        }
      }

      // Apply title prefix filter
      if (requiredTitlePrefix && !pr.title.startsWith(requiredTitlePrefix)) {
        core.info(`Pull request #${prNumber} does not have required title prefix: ${requiredTitlePrefix}`);
        continue;
      }

      // Check if already closed
      if (pr.state === "closed") {
        core.info(`Pull request #${prNumber} is already closed, skipping`);
        continue;
      }

      // Build comment body with workflow info
      const workflowName = process.env.GH_AW_WORKFLOW_NAME || "Workflow";
      const workflowSource = process.env.GH_AW_WORKFLOW_SOURCE || "";
      const workflowSourceURL = process.env.GH_AW_WORKFLOW_SOURCE_URL || "";
      const runId = context.runId;
      const githubServer = process.env.GITHUB_SERVER_URL || "https://github.com";
      const runUrl = context.payload.repository
        ? `${context.payload.repository.html_url}/actions/runs/${runId}`
        : `${githubServer}/${context.repo.owner}/${context.repo.repo}/actions/runs/${runId}`;

      // Add tracker ID to body
      let commentBody = item.body.trim();
      commentBody += getTrackerID("markdown");

      // Add footer with AI attribution
      commentBody += generateFooter(workflowName, runUrl, workflowSource, workflowSourceURL, undefined, triggeringPRNumber, undefined);

      // Add comment before closing
      const comment = await addPullRequestComment(github, context.repo.owner, context.repo.repo, prNumber, commentBody);
      core.info(`✓ Added comment to pull request #${prNumber}: ${comment.html_url}`);

      // Close the pull request
      const closedPR = await closePullRequest(github, context.repo.owner, context.repo.repo, prNumber);
      core.info(`✓ Closed pull request #${prNumber}: ${closedPR.html_url}`);

      closedPRs.push({
        pr: closedPR,
        comment,
      });

      // Set outputs for the last closed pull request (for backward compatibility)
      if (i === closePRItems.length - 1) {
        core.setOutput("pull_request_number", closedPR.number);
        core.setOutput("pull_request_url", closedPR.html_url);
        core.setOutput("comment_url", comment.html_url);
      }
    } catch (error) {
      core.error(`✗ Failed to close pull request #${prNumber}: ${error instanceof Error ? error.message : String(error)}`);
      throw error;
    }
  }

  // Write summary for all closed pull requests
  if (closedPRs.length > 0) {
    let summaryContent = "\n\n## Closed Pull Requests\n";
    for (const { pr, comment } of closedPRs) {
      // Escape special markdown characters in title
      const escapedTitle = pr.title.replace(/[[\]()]/g, "\\$&");
      summaryContent += `- Pull Request #${pr.number}: [${escapedTitle}](${pr.html_url}) ([comment](${comment.html_url}))\n`;
    }
    await core.summary.addRaw(summaryContent).write();
  }

  core.info(`Successfully closed ${closedPRs.length} pull request(s)`);
  return closedPRs;
}

await main();
