// @ts-check
/// <reference types="@actions/github-script" />

const { runUpdateWorkflow, createRenderStagedItem, createGetSummaryLine } = require("./update_runner.cjs");

/**
 * Check if the current context is a valid issue context
 * @param {string} eventName - GitHub event name
 * @param {any} _payload - GitHub event payload (unused but kept for interface consistency)
 * @returns {boolean} Whether context is valid for issue updates
 */
function isIssueContext(eventName, _payload) {
  return eventName === "issues" || eventName === "issue_comment";
}

/**
 * Get issue number from the context payload
 * @param {any} payload - GitHub event payload
 * @returns {number|undefined} Issue number or undefined
 */
function getIssueNumber(payload) {
  return payload.issue?.number;
}

// Use shared helper for staged preview rendering
const renderStagedItem = createRenderStagedItem({
  entityName: "Issue",
  numberField: "issue_number",
  targetLabel: "Target Issue:",
  currentTargetText: "Current issue",
  includeOperation: false,
});

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

// Use shared helper for summary line generation
const getSummaryLine = createGetSummaryLine({
  entityPrefix: "Issue",
});

async function main() {
  return await runUpdateWorkflow({
    itemType: "update_issue",
    displayName: "issue",
    displayNamePlural: "issues",
    numberField: "issue_number",
    outputNumberKey: "issue_number",
    outputUrlKey: "issue_url",
    isValidContext: isIssueContext,
    getContextNumber: getIssueNumber,
    supportsStatus: true,
    supportsOperation: false,
    renderStagedItem,
    executeUpdate: executeIssueUpdate,
    getSummaryLine,
  });
}

await main();
