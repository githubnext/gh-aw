// @ts-check
/// <reference types="@actions/github-script" />

/**
 * Checkout PR branch when PR context is available
 *
 * This script handles checkout for different GitHub event types:
 *
 * 1. pull_request: Runs in merge commit context (PR head + base merged)
 *    - Can use direct git commands since we're already in PR context
 *    - Branch exists in current checkout
 *
 * 2. pull_request_target: Runs in BASE repository context (not PR head)
 *    - CRITICAL: For fork PRs, the head branch doesn't exist in base repo
 *    - Must use `gh pr checkout` to fetch from the fork
 *    - Has write permissions (be cautious with untrusted code)
 *
 * 3. Other PR events (issue_comment, pull_request_review, etc.):
 *    - Also run in base repository context
 *    - Must use `gh pr checkout` to get PR branch
 *
 * NOTE: This handler operates within the PR context from the workflow event
 * and does not support cross-repository operations or target-repo parameters.
 * No allowlist validation (checkAllowedRepo/validateTargetRepo) is needed as
 * it only works with the PR from the triggering event.
 */

const { getErrorMessage } = require("./error_helpers.cjs");
const { renderTemplate } = require("./messages_core.cjs");
const { detectForkPR } = require("./pr_helpers.cjs");
const fs = require("fs");
const { ERR_API } = require("./error_codes.cjs");

/**
 * Log detailed PR context information for debugging
 */
function logPRContext(eventName, pullRequest) {
  core.startGroup("📋 PR Context Details");

  core.info(`Event type: ${eventName}`);
  core.info(`PR number: ${pullRequest.number}`);
  core.info(`PR state: ${pullRequest.state || "unknown"}`);

  // Log head information
  if (pullRequest.head) {
    core.info(`Head ref: ${pullRequest.head.ref || "unknown"}`);
    core.info(`Head SHA: ${pullRequest.head.sha || "unknown"}`);

    if (pullRequest.head.repo) {
      core.info(`Head repo: ${pullRequest.head.repo.full_name || "unknown"}`);
      core.info(`Head repo owner: ${pullRequest.head.repo.owner?.login || "unknown"}`);
    } else {
      core.warning("⚠️ Head repo information not available (repo may be deleted)");
    }
  }

  // Log base information
  if (pullRequest.base) {
    core.info(`Base ref: ${pullRequest.base.ref || "unknown"}`);
    core.info(`Base SHA: ${pullRequest.base.sha || "unknown"}`);

    if (pullRequest.base.repo) {
      core.info(`Base repo: ${pullRequest.base.repo.full_name || "unknown"}`);
      core.info(`Base repo owner: ${pullRequest.base.repo.owner?.login || "unknown"}`);
    }
  }

  // Determine if this is a fork PR using the helper function
  const { isFork, reason: forkReason } = detectForkPR(pullRequest);
  core.info(`Is fork PR: ${isFork} (${forkReason})`);

  // Log current repository context
  core.info(`Current repository: ${context.repo.owner}/${context.repo.repo}`);
  core.info(`GitHub SHA: ${context.sha}`);

  core.endGroup();

  return { isFork };
}

/**
 * Fetch full PR commit count from the GitHub API.
 * Used when the commit count is not available in the webhook payload.
 */
async function fetchPRCommitCount(prNumber) {
  const { data } = await github.rest.pulls.get({
    owner: context.repo.owner,
    repo: context.repo.repo,
    pull_number: prNumber,
  });
  return { commitCount: data.commits, headRef: data.head.ref };
}

/**
 * Log the checkout strategy being used
 */
function logCheckoutStrategy(eventName, strategy, reason) {
  core.startGroup("🔄 Checkout Strategy");
  core.info(`Event type: ${eventName}`);
  core.info(`Strategy: ${strategy}`);
  core.info(`Reason: ${reason}`);
  core.endGroup();
}

async function main() {
  const eventName = context.eventName;
  // For pull_request events, the PR context is in context.payload.pull_request.
  // For issue_comment events on PRs, context.payload.pull_request is not set;
  // instead context.payload.issue.pull_request indicates the issue is a PR.
  let pullRequest = context.payload.pull_request;

  // Handle issue_comment (and similar) events triggered on a PR
  if (!pullRequest && context.payload.issue?.pull_request) {
    pullRequest = {
      number: context.payload.issue.number,
      state: context.payload.issue.state || "open",
    };
    core.info(`Detected ${eventName} event on PR #${pullRequest.number}, will use gh pr checkout`);
  }

  if (!pullRequest) {
    core.info("No pull request context available, skipping checkout");
    core.setOutput("checkout_pr_success", "true");
    return;
  }

  core.info(`Event: ${eventName}`);
  core.info(`Pull Request #${pullRequest.number}`);

  // Check if PR is closed
  const isClosed = pullRequest.state === "closed";
  if (isClosed) {
    core.info("⚠️ Pull request is closed");
  }

  try {
    // Log detailed context for debugging
    const { isFork } = logPRContext(eventName, pullRequest);

    if (eventName === "pull_request") {
      // For pull_request events, we run in the merge commit context
      // The PR branch is already available, so we can use direct git commands
      const branchName = pullRequest.head.ref;
      // commits is in the payload for pull_request events; +1 to include the merge base
      const commitCount = pullRequest.commits || 1;
      const fetchDepth = commitCount + 1;

      logCheckoutStrategy(eventName, "git fetch + checkout", "pull_request event runs in merge commit context with PR branch available");

      core.info(`Fetching branch: ${branchName} from origin (depth: ${fetchDepth} for ${commitCount} PR commit(s))`);
      await exec.exec("git", ["fetch", "origin", branchName, `--depth=${fetchDepth}`]);

      core.info(`Checking out branch: ${branchName}`);
      await exec.exec("git", ["checkout", branchName]);

      core.info(`✅ Successfully checked out branch: ${branchName}`);
    } else {
      // For pull_request_target and other PR events, we run in base repository context
      // IMPORTANT: For fork PRs, the head branch doesn't exist in the base repo
      // We must use `gh pr checkout` which handles fetching from forks
      const prNumber = pullRequest.number;

      const strategyReason = eventName === "pull_request_target" ? "pull_request_target runs in base repo context; for fork PRs, head branch doesn't exist in origin" : `${eventName} event runs in base repo context; must fetch PR branch`;

      logCheckoutStrategy(eventName, "gh pr checkout", strategyReason);

      if (isFork) {
        core.warning("⚠️ Fork PR detected - gh pr checkout will fetch from fork repository");
      }

      core.info(`Checking out PR #${prNumber} using gh CLI`);
      await exec.exec("gh", ["pr", "checkout", prNumber.toString()]);

      // Log the resulting branch after checkout
      let currentBranch = "";
      await exec.exec("git", ["branch", "--show-current"], {
        listeners: {
          stdout: data => {
            currentBranch += data.toString();
          },
        },
      });
      currentBranch = currentBranch.trim();

      // Deepen history to cover all PR commits (gh pr checkout fetches --depth=1 by default)
      try {
        const { commitCount, headRef } = await fetchPRCommitCount(prNumber);
        const fetchDepth = commitCount + 1; // +1 to include the merge base
        const branchRef = headRef || currentBranch;
        core.info(`Deepening history for PR #${prNumber}: fetching ${fetchDepth} commits (${commitCount} PR commit(s) + merge base)`);
        if (branchRef) {
          await exec.exec("git", ["fetch", "origin", branchRef, `--depth=${fetchDepth}`]);
        } else {
          await exec.exec("git", ["fetch", `--depth=${fetchDepth}`]);
        }
      } catch (depthError) {
        core.warning(`Could not deepen PR history: ${getErrorMessage(depthError)}`);
      }

      core.info(`✅ Successfully checked out PR #${prNumber}`);
      core.info(`Current branch: ${currentBranch || "detached HEAD"}`);
    }

    // Set output to indicate successful checkout
    core.setOutput("checkout_pr_success", "true");
  } catch (error) {
    const errorMsg = getErrorMessage(error);

    // Check if PR is closed - if so, treat checkout failure as a warning
    if (isClosed) {
      core.startGroup("⚠️ Closed PR Checkout Warning");
      core.warning(`Event type: ${eventName}`);
      core.warning(`PR number: ${pullRequest.number}`);
      core.warning(`PR state: closed`);
      core.warning(`Checkout failed (expected for closed PR): ${errorMsg}`);

      if (pullRequest.head?.ref) {
        core.warning(`Branch likely deleted: ${pullRequest.head.ref}`);
      }

      core.warning("This is expected behavior when a PR is closed - the branch may have been deleted.");
      core.endGroup();

      // Set output to indicate successful handling of closed PR
      core.setOutput("checkout_pr_success", "true");

      // Add a brief summary noting this is expected
      const warningMessage = `## ⚠️ Closed Pull Request

Pull request #${pullRequest.number} is closed. The checkout failed because the branch has likely been deleted, which is expected behavior.

**This is not an error** - workflows targeting closed PRs will continue normally.`;

      await core.summary.addRaw(warningMessage).write();

      // Do NOT call setFailed - this should not fail the step
      return;
    }

    // For open PRs, treat checkout failure as an error
    // Log detailed error context
    core.startGroup("❌ Checkout Error Details");
    core.error(`Event type: ${eventName}`);
    core.error(`PR number: ${pullRequest.number}`);
    core.error(`Error message: ${errorMsg}`);

    if (pullRequest.head?.ref) {
      core.error(`Attempted to check out: ${pullRequest.head.ref}`);
    }

    // Log current git state for debugging
    try {
      core.info("Current git status:");
      await exec.exec("git", ["status"]);

      core.info("Available remotes:");
      await exec.exec("git", ["remote", "-v"]);

      core.info("Current branch:");
      await exec.exec("git", ["branch", "--show-current"]);
    } catch (gitError) {
      core.warning(`Could not retrieve git state: ${getErrorMessage(gitError)}`);
    }

    core.endGroup();

    // Set output to indicate checkout failure
    core.setOutput("checkout_pr_success", "false");

    // Load and render step summary template
    const templatePath = "/opt/gh-aw/prompts/pr_checkout_failure.md";
    const template = fs.readFileSync(templatePath, "utf8");
    const summaryContent = renderTemplate(template, {
      error_message: errorMsg,
    });

    await core.summary.addRaw(summaryContent).write();
    core.setFailed(`${ERR_API}: Failed to checkout PR branch: ${errorMsg}`);
  }
}

module.exports = { main };
