// @ts-check
/// <reference types="@actions/github-script" />

/**
 * Unlock a GitHub issue
 * This script is used in the conclusion job to ensure the issue is unlocked
 * after agent workflow execution completes or fails
 */

const { getErrorMessage } = require("./error_helpers.cjs");

async function main() {
  // Log actor and event information for debugging
  core.info(`Unlock-issue debug: actor=${context.actor}, eventName=${context.eventName}`);

  // Get issue number from context
  const issueNumber = context.issue.number;

  if (!issueNumber) {
    core.setFailed("Issue number not found in context");
    return;
  }

  const owner = context.repo.owner;
  const repo = context.repo.repo;

  core.info(`Unlock-issue debug: owner=${owner}, repo=${repo}, issueNumber=${issueNumber}`);

  try {
    // Check if issue is locked
    core.info(`Checking if issue #${issueNumber} is locked`);
    const { data: issue } = await github.rest.issues.get({
      owner,
      repo,
      issue_number: issueNumber,
    });

    // Skip unlocking if this is a pull request (PRs cannot be unlocked via issues API)
    if (issue.pull_request) {
      core.info(`ℹ️ Issue #${issueNumber} is a pull request, skipping unlock operation`);
      return;
    }

    if (!issue.locked) {
      core.info(`ℹ️ Issue #${issueNumber} is not locked, skipping unlock operation`);
      return;
    }

    core.info(`Unlocking issue #${issueNumber} after agent workflow execution`);

    // Unlock the issue
    await github.rest.issues.unlock({
      owner,
      repo,
      issue_number: issueNumber,
    });

    core.info(`✅ Successfully unlocked issue #${issueNumber}`);
  } catch (error) {
    const errorMessage = getErrorMessage(error);
    core.error(`Failed to unlock issue: ${errorMessage}`);
    core.setFailed(`Failed to unlock issue #${issueNumber}: ${errorMessage}`);
  }
}

module.exports = { main };
