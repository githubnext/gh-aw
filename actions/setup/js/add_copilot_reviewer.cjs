// @ts-check
/// <reference types="@actions/github-script" />

const { getErrorMessage } = require("./error_helpers.cjs");
const { ERR_CONFIG, ERR_NOT_FOUND, ERR_VALIDATION } = require("./error_codes.cjs");
const { COPILOT_REVIEWER_BOT } = require("./constants.cjs");

/**
 * Add Copilot as a reviewer to a pull request.
 *
 * Runs in github-script context. Requires `PR_NUMBER` environment variable.
 */
async function main() {
  const prNumberStr = process.env.PR_NUMBER?.trim();

  if (!prNumberStr) {
    core.setFailed(`${ERR_CONFIG}: PR_NUMBER environment variable is required but not set`);
    return;
  }

  const prNumber = parseInt(prNumberStr, 10);
  if (isNaN(prNumber) || prNumber <= 0) {
    core.setFailed(`${ERR_VALIDATION}: Invalid PR_NUMBER: ${prNumberStr}. Must be a positive integer.`);
    return;
  }

  core.info(`Adding Copilot as reviewer to PR #${prNumber}`);

  const { owner, repo } = context.repo;
  try {
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
    core.setFailed(`${ERR_NOT_FOUND}: Failed to add Copilot as reviewer to PR #${prNumber}: ${errorMessage}`);
  }
}

module.exports = { main };
