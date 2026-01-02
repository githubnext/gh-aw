// @ts-check
/// <reference types="@actions/github-script" />

const { getErrorMessage } = require("./error_helpers.cjs");

/**
 * Add Copilot as a reviewer to a pull request.
 *
 * This script is used to add the GitHub Copilot pull request reviewer bot
 * to a pull request. It uses the `github` object from actions/github-script
 * instead of the `gh api` CLI command.
 *
 * Environment variables:
 * - PR_NUMBER: The pull request number to add the reviewer to
 */

// GitHub Copilot reviewer bot username
const COPILOT_REVIEWER_BOT = "copilot-pull-request-reviewer[bot]";

async function main() {
  // Validate required environment variables
  const prNumberStr = process.env.PR_NUMBER?.trim();

  if (!prNumberStr) {
    core.setFailed("PR_NUMBER environment variable is required but not set");
    return;
  }

  const prNumber = parseInt(prNumberStr, 10);
  if (isNaN(prNumber) || prNumber <= 0) {
    core.setFailed(`Invalid PR_NUMBER: ${prNumberStr}. Must be a positive integer.`);
    return;
  }

  core.info(`Adding Copilot as reviewer to PR #${prNumber}`);

  try {
    const { owner, repo } = context.repo;
    await github.rest.pulls.requestReviewers({
      owner,
      repo,
      pull_number: prNumber,
      reviewers: [COPILOT_REVIEWER_BOT],
    });

    core.info(`Successfully added Copilot as reviewer to PR #${prNumber}`);

    await core.summary
      .addRaw(
        `## Copilot Reviewer Added

Successfully added Copilot as a reviewer to PR #${prNumber}.`
      )
      .write();
  } catch (error) {
    const errorMessage = getErrorMessage(error);
    core.error(`Failed to add Copilot as reviewer: ${errorMessage}`);
    core.setFailed(`Failed to add Copilot as reviewer to PR #${prNumber}: ${errorMessage}`);
  }
}

module.exports = { main };
