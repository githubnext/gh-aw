// @ts-check
/// <reference types="@actions/github-script" />

const { createUpdateHandler } = require("./update_runner.cjs");
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

// Create the handler using the factory
const main = createUpdateHandler({
  itemType: "update_issue",
  displayName: "issue",
  displayNamePlural: "issues",
  numberField: "issue_number",
  outputNumberKey: "issue_number",
  outputUrlKey: "issue_url",
  entityName: "Issue",
  entityPrefix: "Issue",
  targetLabel: "Target Issue:",
  currentTargetText: "Current issue",
  supportsStatus: true,
  supportsOperation: false,
  isValidContext: isIssueContext,
  getContextNumber: getIssueNumber,
  executeUpdate: executeIssueUpdate,
});

await main();
