// @ts-check
/// <reference types="@actions/github-script" />

const { getErrorMessage } = require("./error_helpers.cjs");

/**
 * @typedef {import('./types/handler-factory').HandlerFactoryFunction} HandlerFactoryFunction
 */

/** @type {string} Safe output type handled by this module */
const HANDLER_TYPE = "create_missing_data_issue";

/**
 * Main handler factory for create_missing_data_issue
 * Returns a message handler function that processes individual create_missing_data_issue messages
 * @type {HandlerFactoryFunction}
 */
async function main(config = {}) {
  // Extract configuration
  const titlePrefix = config.title_prefix || "[missing data]";
  const envLabels = config.labels ? (Array.isArray(config.labels) ? config.labels : config.labels.split(",")).map(label => String(label).trim()).filter(label => label) : [];
  const maxCount = config.max || 1; // Default to 1 to create only one issue per workflow run

  core.info(`Title prefix: ${titlePrefix}`);
  if (envLabels.length > 0) {
    core.info(`Default labels: ${envLabels.join(", ")}`);
  }
  core.info(`Max count: ${maxCount}`);

  // Track how many items we've processed for max limit
  let processedCount = 0;

  // Track created/updated issues
  const processedIssues = [];

  /**
   * Create or update an issue for missing data
   * @param {string} workflowName - Name of the workflow
   * @param {string} workflowSource - Source path of the workflow
   * @param {string} workflowSourceURL - URL to the workflow source
   * @param {string} runUrl - URL to the workflow run
   * @param {Array<Object>} missingDataItems - Array of missing data objects
   * @returns {Promise<Object>} Result with success/error status
   */
  async function createOrUpdateIssue(workflowName, workflowSource, workflowSourceURL, runUrl, missingDataItems) {
    const { owner, repo } = context.repo;

    // Create issue title
    const issueTitle = `${titlePrefix} ${workflowName}`;

    core.info(`Checking for existing issue with title: "${issueTitle}"`);

    // Search for existing open issue with this title
    const searchQuery = `repo:${owner}/${repo} is:issue is:open in:title "${issueTitle}"`;

    try {
      const searchResult = await github.rest.search.issuesAndPullRequests({
        q: searchQuery,
        per_page: 1,
      });

      if (searchResult.data.total_count > 0) {
        // Issue exists, add a comment
        const existingIssue = searchResult.data.items[0];
        core.info(`Found existing issue #${existingIssue.number}: ${existingIssue.html_url}`);

        // Build comment body
        const commentLines = [`## Missing Data Reported (${new Date().toISOString()})`, ``, `The following data was reported as missing during [workflow run](${runUrl}):`, ``];

        missingDataItems.forEach((item, index) => {
          commentLines.push(`### ${index + 1}. **${item.data_type}**`);
          commentLines.push(`**Reason:** ${item.reason}`);
          if (item.context) {
            commentLines.push(`**Context:** ${item.context}`);
          }
          if (item.alternatives) {
            commentLines.push(`**Alternatives:** ${item.alternatives}`);
          }
          commentLines.push(``);
        });

        commentLines.push(`---`);
        commentLines.push(`> Workflow: [${workflowName}](${workflowSourceURL})`);
        commentLines.push(`> Run: ${runUrl}`);

        const commentBody = commentLines.join("\n");

        await github.rest.issues.createComment({
          owner,
          repo,
          issue_number: existingIssue.number,
          body: commentBody,
        });

        core.info(`✓ Added comment to existing issue #${existingIssue.number}`);

        return {
          success: true,
          issue_number: existingIssue.number,
          issue_url: existingIssue.html_url,
          action: "updated",
        };
      } else {
        // No existing issue, create a new one
        core.info("No existing issue found, creating a new one");

        // Build issue body with detailed context that rewards truthfulness
        const bodyLines = [
          `## Problem`,
          ``,
          `The workflow **${workflowName}** reported missing data during execution. The AI agent requires this data to complete its tasks effectively and has been **truthful** in acknowledging the data gaps rather than inventing information.`,
          ``,
          `> **Note:** This report demonstrates responsible AI behavior. The agent correctly identified missing information instead of hallucinating or making assumptions that could lead to incorrect results.`,
          ``,
          `### Missing Data`,
          ``,
        ];

        missingDataItems.forEach((item, index) => {
          bodyLines.push(`#### ${index + 1}. **${item.data_type}**`);
          bodyLines.push(`**Reason:** ${item.reason}`);
          if (item.context) {
            bodyLines.push(`**Context:** ${item.context}`);
          }
          if (item.alternatives) {
            bodyLines.push(`**Alternatives:** ${item.alternatives}`);
          }
          bodyLines.push(`**Reported at:** ${item.timestamp}`);
          bodyLines.push(``);
        });

        bodyLines.push(`## Why This Matters`);
        bodyLines.push(``);
        bodyLines.push(`AI agents work best when they can be honest about their limitations. By reporting missing data instead of:`);
        bodyLines.push(`- Inventing information that might be incorrect`);
        bodyLines.push(`- Making unsupported assumptions`);
        bodyLines.push(`- Proceeding with incomplete context`);
        bodyLines.push(``);
        bodyLines.push(`...the agent ensures higher quality, more reliable outputs. This truthfulness should be encouraged and rewarded.`);
        bodyLines.push(``);
        bodyLines.push(`## Action Required`);
        bodyLines.push(``);
        bodyLines.push(`To help the agent succeed, please:`);
        bodyLines.push(`1. **Provide the missing data** by updating relevant documentation, configuration, or context`);
        bodyLines.push(`2. **Make data accessible** by adjusting permissions, exposing APIs, or creating data sources`);
        bodyLines.push(`3. **Clarify requirements** if the agent's understanding is incomplete`);
        bodyLines.push(`4. **Consider alternatives** if the requested data cannot be provided`);
        bodyLines.push(``);
        bodyLines.push(`## Agent Instructions`);
        bodyLines.push(``);
        bodyLines.push(`**Agent:** \`agentic-workflows\``);
        bodyLines.push(`**Task:** Debug and resolve missing data issue`);
        bodyLines.push(``);
        bodyLines.push(`**Steps:**`);
        bodyLines.push(``);
        bodyLines.push(`1. Invoke agent: \`/agent agentic-workflows\``);
        bodyLines.push(`2. Command: "Debug this missing data issue"`);
        bodyLines.push(`3. Analyze what data the agent needs and why`);
        bodyLines.push(`4. Determine the appropriate solution:`);
        bodyLines.push(`   - Add data sources to the workflow configuration`);
        bodyLines.push(`   - Adjust permissions to make data accessible`);
        bodyLines.push(`   - Create APIs or endpoints for data access`);
        bodyLines.push(`   - Clarify requirements if the agent's understanding is incomplete`);
        bodyLines.push(`5. Implement the fix and validate data is now accessible`);
        bodyLines.push(``);
        bodyLines.push(`## References`);
        bodyLines.push(``);
        bodyLines.push(`- **Workflow:** [${workflowName}](${workflowSourceURL})`);
        bodyLines.push(`- **Failed Run:** ${runUrl}`);
        bodyLines.push(`- **Source:** ${workflowSource}`);

        // Add expiration marker (1 week from now)
        const expirationDate = new Date();
        expirationDate.setDate(expirationDate.getDate() + 7);
        bodyLines.push(``);
        bodyLines.push(`<!-- gh-aw-expires: ${expirationDate.toISOString()} -->`);

        const issueBody = bodyLines.join("\n");

        const newIssue = await github.rest.issues.create({
          owner,
          repo,
          title: issueTitle,
          body: issueBody,
          labels: envLabels,
        });

        core.info(`✓ Created new issue #${newIssue.data.number}: ${newIssue.data.html_url}`);

        return {
          success: true,
          issue_number: newIssue.data.number,
          issue_url: newIssue.data.html_url,
          action: "created",
        };
      }
    } catch (error) {
      core.warning(`Failed to create or update issue: ${getErrorMessage(error)}`);
      return {
        success: false,
        error: getErrorMessage(error),
      };
    }
  }

  /**
   * Message handler function that processes a single create_missing_data_issue message
   * @param {Object} message - The create_missing_data_issue message to process
   * @returns {Promise<Object>} Result with success/error status and issue details
   */
  return async function handleCreateMissingDataIssue(message) {
    // Check if we've hit the max limit
    if (processedCount >= maxCount) {
      core.warning(`Skipping create_missing_data_issue: max count of ${maxCount} reached`);
      return {
        success: false,
        error: `Max count of ${maxCount} reached`,
      };
    }

    processedCount++;

    // Validate required fields
    if (!message.workflow_name) {
      core.warning(`Missing required field: workflow_name`);
      return {
        success: false,
        error: "Missing required field: workflow_name",
      };
    }

    if (!message.missing_data || !Array.isArray(message.missing_data) || message.missing_data.length === 0) {
      core.warning(`Missing or empty missing_data array`);
      return {
        success: false,
        error: "Missing or empty missing_data array",
      };
    }

    // Extract fields from message
    const workflowName = message.workflow_name;
    const workflowSource = message.workflow_source || "";
    const workflowSourceURL = message.workflow_source_url || "";
    const runUrl = message.run_url || "";
    const missingDataItems = message.missing_data;

    // Create or update the issue
    const result = await createOrUpdateIssue(workflowName, workflowSource, workflowSourceURL, runUrl, missingDataItems);

    if (result.success) {
      processedIssues.push(result);
    }

    return result;
  };
}

module.exports = { main };
