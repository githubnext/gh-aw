// @ts-check
/// <reference types="@actions/github-script" />

const { runUpdateWorkflow } = require("./update_runner.cjs");

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

/**
 * Render a staged preview item for issue updates
 * @param {any} item - Update item
 * @param {number} index - Item index (0-based)
 * @returns {string} Markdown content for the preview
 */
function renderStagedItem(item, index) {
  let content = `### Issue Update ${index + 1}\n`;
  if (item.issue_number) {
    content += `**Target Issue:** #${item.issue_number}\n\n`;
  } else {
    content += `**Target:** Current issue\n\n`;
  }

  if (item.title !== undefined) {
    content += `**New Title:** ${item.title}\n\n`;
  }
  if (item.body !== undefined) {
    content += `**New Body:**\n${item.body}\n\n`;
  }
  if (item.status !== undefined) {
    content += `**New Status:** ${item.status}\n\n`;
  }
  return content;
}

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

/**
 * Generate summary line for an updated issue
 * @param {any} issue - Updated issue
 * @returns {string} Markdown summary line
 */
function getSummaryLine(issue) {
  return `- Issue #${issue.number}: [${issue.title}](${issue.html_url})\n`;
}

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
