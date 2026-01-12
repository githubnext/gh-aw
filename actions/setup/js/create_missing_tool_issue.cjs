// @ts-check
/// <reference types="@actions/github-script" />

const { getErrorMessage } = require("./error_helpers.cjs");

/**
 * @typedef {import('./types/handler-factory').HandlerFactoryFunction} HandlerFactoryFunction
 */

/** @type {string} Safe output type handled by this module */
const HANDLER_TYPE = "create_missing_tool_issue";

/**
 * Main handler factory for create_missing_tool_issue
 * Returns a message handler function that processes individual create_missing_tool_issue messages
 * @type {HandlerFactoryFunction}
 */
async function main(config = {}) {
  // Extract configuration
  const titlePrefix = config.title_prefix || "[missing tool]";
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
   * Create or update an issue for missing tools
   * @param {string} workflowName - Name of the workflow
   * @param {string} workflowSource - Source path of the workflow
   * @param {string} workflowSourceURL - URL to the workflow source
   * @param {string} runUrl - URL to the workflow run
   * @param {Array<Object>} missingTools - Array of missing tool objects
   * @returns {Promise<Object>} Result with success/error status
   */
  async function createOrUpdateIssue(workflowName, workflowSource, workflowSourceURL, runUrl, missingTools) {
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
        const commentLines = [`## Missing Tools Reported (${new Date().toISOString()})`, ``, `The following tools were reported as missing during [workflow run](${runUrl}):`, ``];

        missingTools.forEach((tool, index) => {
          commentLines.push(`### ${index + 1}. \`${tool.tool}\``);
          commentLines.push(`**Reason:** ${tool.reason}`);
          if (tool.alternatives) {
            commentLines.push(`**Alternatives:** ${tool.alternatives}`);
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

        // Build issue body with agentic task description
        const bodyLines = [`## Problem`, ``, `The workflow **${workflowName}** reported missing tools during execution. These tools are needed for the agent to complete its tasks effectively.`, ``, `### Missing Tools`, ``];

        missingTools.forEach((tool, index) => {
          bodyLines.push(`#### ${index + 1}. \`${tool.tool}\``);
          bodyLines.push(`**Reason:** ${tool.reason}`);
          if (tool.alternatives) {
            bodyLines.push(`**Alternatives:** ${tool.alternatives}`);
          }
          bodyLines.push(`**Reported at:** ${tool.timestamp}`);
          bodyLines.push(``);
        });

        bodyLines.push(`## Action Required`);
        bodyLines.push(``);
        bodyLines.push(`Please investigate why these tools are missing and either:`);
        bodyLines.push(`1. Add the missing tools to the agent's configuration`);
        bodyLines.push(`2. Update the workflow to use available alternatives`);
        bodyLines.push(`3. Document why these tools are intentionally unavailable`);
        bodyLines.push(``);
        bodyLines.push(`## Agent Instructions`);
        bodyLines.push(``);
        bodyLines.push(`**Agent:** \`agentic-workflows\``);
        bodyLines.push(`**Task:** Debug and resolve missing tool issue`);
        bodyLines.push(``);
        bodyLines.push(`**Steps:**`);
        bodyLines.push(``);
        bodyLines.push(`1. Invoke agent: \`/agent agentic-workflows\``);
        bodyLines.push(`2. Command: "Debug this missing tool issue"`);
        bodyLines.push(`3. Analyze which tools are missing and why they're needed`);
        bodyLines.push(`4. Determine the appropriate solution:`);
        bodyLines.push(`   - Add missing tools to the workflow configuration`);
        bodyLines.push(`   - Update the workflow to use available alternatives`);
        bodyLines.push(`   - Install required MCP servers or dependencies`);
        bodyLines.push(`   - Document why certain tools are intentionally unavailable`);
        bodyLines.push(`5. Implement the fix and validate tools are now accessible`);
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
   * Message handler function that processes a single create_missing_tool_issue message
   * @param {Object} message - The create_missing_tool_issue message to process
   * @returns {Promise<Object>} Result with success/error status and issue details
   */
  return async function handleCreateMissingToolIssue(message) {
    // Check if we've hit the max limit
    if (processedCount >= maxCount) {
      core.warning(`Skipping create_missing_tool_issue: max count of ${maxCount} reached`);
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

    if (!message.missing_tools || !Array.isArray(message.missing_tools) || message.missing_tools.length === 0) {
      core.warning(`Missing or empty missing_tools array`);
      return {
        success: false,
        error: "Missing or empty missing_tools array",
      };
    }

    // Extract fields from message
    const workflowName = message.workflow_name;
    const workflowSource = message.workflow_source || "";
    const workflowSourceURL = message.workflow_source_url || "";
    const runUrl = message.run_url || "";
    const missingTools = message.missing_tools;

    // Create or update the issue
    const result = await createOrUpdateIssue(workflowName, workflowSource, workflowSourceURL, runUrl, missingTools);

    if (result.success) {
      processedIssues.push(result);
    }

    return result;
  };
}

module.exports = { main };
