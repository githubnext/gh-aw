// @ts-check
/// <reference types="@actions/github-script" />

/**
 * Lock a GitHub issue without providing a reason
 * This script is used in the activation job when lock-for-agent is enabled
 * to prevent concurrent modifications during agent workflow execution
 */

const { getErrorMessage } = require("./error_helpers.cjs");

async function main() {
  // Log actor and event information for debugging
  core.info(`Lock-issue debug: actor=${context.actor}, eventName=${context.eventName}`);

  // Get issue number from context
  const issueNumber = context.issue.number;

  if (!issueNumber) {
    core.setFailed("Issue number not found in context");
    return;
  }

  const owner = context.repo.owner;
  const repo = context.repo.repo;

  core.info(`Lock-issue debug: owner=${owner}, repo=${repo}, issueNumber=${issueNumber}`);

  try {
    // Check if issue is already locked
    core.info(`Checking if issue #${issueNumber} is already locked`);
    const { data: issue } = await github.rest.issues.get({
      owner,
      repo,
      issue_number: issueNumber,
    });

    // Skip locking if this is a pull request (PRs cannot be locked via issues API)
    if (issue.pull_request) {
      core.info(`ℹ️ Issue #${issueNumber} is a pull request, skipping lock operation`);
      core.setOutput("locked", "false");
      return;
    }

    if (issue.locked) {
      core.info(`ℹ️ Issue #${issueNumber} is already locked, skipping lock operation`);
      core.setOutput("locked", "false");
      return;
    }

    core.info(`Locking issue #${issueNumber} for agent workflow execution`);

    // Lock the issue without providing a lock_reason parameter
    await github.rest.issues.lock({
      owner,
      repo,
      issue_number: issueNumber,
    });

    core.info(`✅ Successfully locked issue #${issueNumber}`);
    // Set output to indicate the issue was locked and needs to be unlocked
    core.setOutput("locked", "true");
  } catch (error) {
    const errorMessage = getErrorMessage(error);
    core.error(`Failed to lock issue: ${errorMessage}`);
    core.setFailed(`Failed to lock issue #${issueNumber}: ${errorMessage}`);
    core.setOutput("locked", "false");
  }
}

module.exports = { main };
