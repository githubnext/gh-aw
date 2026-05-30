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
 *    - Uses refs/pull/N/head to fetch from origin (works for forks too)
 *    - Has write permissions (be cautious with untrusted code)
 *
 * 3. Other PR events (issue_comment, pull_request_review, etc.):
 *    - Also run in base repository context
 *    - Uses refs/pull/N/head to fetch PR branch
 *
 * NOTE: This handler operates within the PR context from the workflow event
 * and does not support cross-repository operations or target-repo parameters.
 * No allowlist validation (checkAllowedRepo/validateTargetRepo) is needed as
 * it only works with the PR from the triggering event.
 */

const { getErrorMessage } = require("./error_helpers.cjs");
const { renderTemplateFromFile, getPromptPath } = require("./messages_core.cjs");
const { detectForkPR } = require("./pr_helpers.cjs");
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

  // Determine if this is a fork PR using the helper function.
  // Only call detectForkPR when head/base data is present (pull_request and
  // pull_request_target payloads). For minimal PR objects (e.g. issue_comment)
  // fork status is unknown until we fetch full PR details from the API.
  let isFork = null;
  if (pullRequest.head?.repo && pullRequest.base?.repo) {
    const { isFork: detected, reason: forkReason } = detectForkPR(pullRequest);
    isFork = detected;
    core.info(`Is fork PR: ${isFork} (${forkReason})`);
  } else {
    core.info("Is fork PR: unknown (head/base repo details not available in event payload)");
  }

  // Log current repository context
  core.info(`Current repository: ${context.repo.owner}/${context.repo.repo}`);
  core.info(`GitHub SHA: ${context.sha}`);

  core.endGroup();

  return { isFork };
}

/**
 * Fetch PR details from the GitHub API.
 * Returns head ref and commit count needed for checkout.
 */
async function fetchPRDetails(prNumber) {
  const { data } = await github.rest.pulls.get({
    owner: context.repo.owner,
    repo: context.repo.repo,
    pull_number: prNumber,
  });
  return { commitCount: data.commits, headRef: data.head.ref, pullRequest: data };
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
    core.info(`Detected ${eventName} event on PR #${pullRequest.number}, will fetch PR ref`);
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

    if (eventName === "pull_request" && isFork === false) {
      // For non-fork pull_request events, we run in the merge commit context.
      // The PR branch is in the same repo as origin, so we can use direct git commands.
      // Fork PRs cannot use git fetch because their head branch only exists in the fork
      // (not in origin/base repo), so they must use gh pr checkout in the else branch below.
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
      // For pull_request_target, fork pull_request events, and other PR events,
      // we run in base repository context.
      // Use refs/pull/N/head which GitHub makes available for all PRs (including forks)
      // so we don't need `gh pr checkout` and avoid GH_HOST / DIFC proxy issues.
      const prNumber = pullRequest.number;

      // Get PR details from API to determine head ref name and commit count.
      // This also gives us the full PR object for accurate fork detection
      // when the event payload only had a minimal PR (e.g. issue_comment).
      const { commitCount, headRef, pullRequest: fullPR } = await fetchPRDetails(prNumber);

      // Re-evaluate fork status with full PR data when it was unknown
      const fullPRForkDetection = detectForkPR(fullPR);
      const actualIsFork = isFork ?? fullPRForkDetection.isFork;
      if (isFork === null) {
        core.info(`Is fork PR (from API): ${actualIsFork} (${fullPRForkDetection.reason})`);
      }

      const strategyReason =
        eventName === "pull_request_target"
          ? "pull_request_target runs in base repo context; fetching via refs/pull/N/head"
          : eventName === "pull_request" && actualIsFork
            ? "pull_request event from fork repository; fetching via refs/pull/N/head"
            : `${eventName} event runs in base repo context; fetching via refs/pull/N/head`;

      logCheckoutStrategy(eventName, "git fetch refs/pull + checkout", strategyReason);

      if (actualIsFork) {
        core.warning("⚠️ Fork PR detected - fetching via refs/pull/N/head from origin");
      }
      const fetchDepth = (commitCount || 1) + 1; // +1 to include the merge base

      core.info(`Fetching PR #${prNumber} head via refs/pull/${prNumber}/head (depth: ${fetchDepth} for ${commitCount} PR commit(s))`);
      await exec.exec("git", ["fetch", "origin", `+refs/pull/${prNumber}/head:refs/remotes/origin/pr-head`, `--depth=${fetchDepth}`]);

      const branchName = headRef || `pr-${prNumber}`;
      core.info(`Checking out branch: ${branchName}`);
      await exec.exec("git", ["checkout", "-B", branchName, "origin/pr-head"]);

      core.info(`✅ Successfully checked out PR #${prNumber}`);
      core.info(`Current branch: ${branchName}`);
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

    // Re-check current PR state via API to handle race conditions where
    // the PR was merged/closed after the webhook payload was captured but
    // before the agent job ran (e.g. PR merged within seconds of triggering).
    let isNowClosed = false;
    try {
      const { data: currentPR } = await github.rest.pulls.get({
        owner: context.repo.owner,
        repo: context.repo.repo,
        pull_number: pullRequest.number,
      });
      isNowClosed = currentPR.state === "closed";
      if (isNowClosed) {
        core.info(`ℹ️ PR #${pullRequest.number} is now closed (was '${pullRequest.state}' in webhook payload) — treating checkout failure as expected`);
      }
    } catch (apiError) {
      const apiErrorMsg = getErrorMessage(apiError);
      const statusCode = /** @type {any} */ apiError?.status;
      const statusSuffix = statusCode ? ` (HTTP ${statusCode})` : "";
      core.warning(`Could not fetch current PR state${statusSuffix}: ${apiErrorMsg}`);
    }

    if (isNowClosed) {
      core.startGroup("⚠️ Closed PR Checkout Warning");
      core.warning(`Event type: ${eventName}`);
      core.warning(`PR number: ${pullRequest.number}`);
      core.warning(`PR state: closed (merged after workflow was triggered)`);
      core.warning(`Checkout failed (expected for closed PR): ${errorMsg}`);

      if (pullRequest.head?.ref) {
        core.warning(`Branch likely deleted: ${pullRequest.head.ref}`);
      }

      core.warning("This is expected behavior when a PR is closed - the branch may have been deleted.");
      core.endGroup();

      // Set output to indicate successful handling of closed PR
      core.setOutput("checkout_pr_success", "true");

      const warningMessage = `## ⚠️ Closed Pull Request

Pull request #${pullRequest.number} was merged after this workflow was triggered. The checkout failed because the branch has been deleted, which is expected behavior.

**This is not an error** - workflows targeting closed PRs will continue normally.`;

      await core.summary.addRaw(warningMessage).write();
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
    const templatePath = getPromptPath("pr_checkout_failure.md");
    const summaryContent = renderTemplateFromFile(templatePath, {
      error_message: errorMsg,
    });

    await core.summary.addRaw(summaryContent).write();
    core.setFailed(`${ERR_API}: Failed to checkout PR branch: ${errorMsg}`);
  }
}

module.exports = { main };
