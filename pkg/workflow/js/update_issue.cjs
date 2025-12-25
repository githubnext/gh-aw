// @ts-check
/// <reference types="@actions/github-script" />

const { createUpdateHandler, getUpdateHandlerConfig } = require("./update_runner.cjs");
const { isIssueContext, getIssueNumber } = require("./update_context_helpers.cjs");

/**
 * Execute the issue update API call
 * @param {any} github - GitHub API client
 * @param {any} context - GitHub Actions context
 * @param {number} issueNumber - Issue number to update
 * @param {any} updateData - Data to update
 * @returns {Promise<any>} Updated issue
 */
async function executeIssueUpdate(github, context, issueNumber, updateData) {
  // Remove internal fields used for operation handling
  const { _operation, _rawBody, ...apiData } = updateData;

  const { data: issue } = await github.rest.issues.update({
    owner: context.repo.owner,
    repo: context.repo.repo,
    issue_number: issueNumber,
    ...apiData,
  });

  return issue;
}

// Create the handler using the factory with centralized config
const main = createUpdateHandler({
  ...getUpdateHandlerConfig("issue"),
  isValidContext: isIssueContext,
  getContextNumber: getIssueNumber,
  executeUpdate: executeIssueUpdate,
});

module.exports = { main };
