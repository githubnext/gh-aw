// @ts-check
/// <reference types="@actions/github-script" />

const { getErrorMessage } = require("./error_helpers.cjs");

/**
 * Create or update an issue for a missing tool
 * @param {any} github - GitHub API client
 * @param {any} context - GitHub Actions context
 * @param {string} workflowName - Name of the workflow
 * @param {string} workflowSource - Source path of the workflow
 * @param {string} workflowSourceURL - URL to the workflow source
 * @param {string} runUrl - URL to the workflow run
 * @param {string} titlePrefix - Prefix for the issue title
 * @param {string[]} labels - Labels to add to the issue
 * @param {any[]} missingTools - Array of missing tool objects
 */
async function createOrUpdateIssue(github, context, workflowName, workflowSource, workflowSourceURL, runUrl, titlePrefix, labels, missingTools) {
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
      bodyLines.push(`## Debugging`);
      bodyLines.push(``);
      bodyLines.push(`To debug this issue, use the **debug-agentic-workflow** agent by running:`);
      bodyLines.push(`\`\`\``);
      bodyLines.push(`gh copilot --agent debug-agentic-workflow`);
      bodyLines.push(`\`\`\``);
      bodyLines.push(``);
      bodyLines.push(`Or in GitHub Copilot Chat, type \`/agent\` and select **debug-agentic-workflow**.`);
      bodyLines.push(``);
      bodyLines.push(`## References`);
      bodyLines.push(``);
      bodyLines.push(`- **Workflow:** [${workflowName}](${workflowSourceURL})`);
      bodyLines.push(`- **Failed Run:** ${runUrl}`);
      bodyLines.push(`- **Source:** ${workflowSource}`);

      const issueBody = bodyLines.join("\n");

      const newIssue = await github.rest.issues.create({
        owner,
        repo,
        title: issueTitle,
        body: issueBody,
        labels: labels,
      });

      core.info(`✓ Created new issue #${newIssue.data.number}: ${newIssue.data.html_url}`);
    }
  } catch (error) {
    core.warning(`Failed to create or update issue: ${getErrorMessage(error)}`);
    core.warning("Continuing with workflow execution...");
  }
}

async function main() {
  const fs = require("fs");

  // Get environment variables
  const agentOutputFile = process.env.GH_AW_AGENT_OUTPUT || "";
  const maxReports = process.env.GH_AW_MISSING_TOOL_MAX ? parseInt(process.env.GH_AW_MISSING_TOOL_MAX) : null;
  const createIssue = process.env.GH_AW_MISSING_TOOL_CREATE_ISSUE === "true";
  const titlePrefix = process.env.GH_AW_MISSING_TOOL_TITLE_PREFIX || "[missing tool]";
  const labelsJSON = process.env.GH_AW_MISSING_TOOL_LABELS || "[]";
  const workflowName = process.env.GH_AW_WORKFLOW_NAME || "Workflow";
  const workflowSource = process.env.GH_AW_WORKFLOW_SOURCE || "";
  const workflowSourceURL = process.env.GH_AW_WORKFLOW_SOURCE_URL || "";

  // Parse labels
  let labels = [];
  try {
    labels = JSON.parse(labelsJSON);
  } catch (error) {
    core.warning(`Failed to parse labels JSON: ${getErrorMessage(error)}`);
  }

  core.info("Processing missing-tool reports...");
  if (maxReports) {
    core.info(`Maximum reports allowed: ${maxReports}`);
  }
  if (createIssue) {
    core.info(`Issue creation enabled with title prefix: "${titlePrefix}"`);
    if (labels.length > 0) {
      core.info(`Issue labels: ${labels.join(", ")}`);
    }
  }

  /** @type {any[]} */
  const missingTools = [];

  // Return early if no agent output
  if (!agentOutputFile.trim()) {
    core.info("No agent output to process");
    core.setOutput("tools_reported", JSON.stringify(missingTools));
    core.setOutput("total_count", missingTools.length.toString());
    return;
  }

  // Read agent output from file
  let agentOutput;
  try {
    agentOutput = fs.readFileSync(agentOutputFile, "utf8");
  } catch (error) {
    core.info(`Agent output file not found or unreadable: ${getErrorMessage(error)}`);
    core.setOutput("tools_reported", JSON.stringify(missingTools));
    core.setOutput("total_count", missingTools.length.toString());
    return;
  }

  if (agentOutput.trim() === "") {
    core.info("No agent output to process");
    core.setOutput("tools_reported", JSON.stringify(missingTools));
    core.setOutput("total_count", missingTools.length.toString());
    return;
  }

  core.info(`Agent output length: ${agentOutput.length}`);

  // Parse the validated output JSON
  let validatedOutput;
  try {
    validatedOutput = JSON.parse(agentOutput);
  } catch (error) {
    core.setFailed(`Error parsing agent output JSON: ${getErrorMessage(error)}`);
    return;
  }

  if (!validatedOutput.items || !Array.isArray(validatedOutput.items)) {
    core.info("No valid items found in agent output");
    core.setOutput("tools_reported", JSON.stringify(missingTools));
    core.setOutput("total_count", missingTools.length.toString());
    return;
  }

  core.info(`Parsed agent output with ${validatedOutput.items.length} entries`);

  // Process all parsed entries
  for (const entry of validatedOutput.items) {
    if (entry.type === "missing_tool") {
      // Validate required fields
      if (!entry.tool) {
        core.warning(`missing-tool entry missing 'tool' field: ${JSON.stringify(entry)}`);
        continue;
      }
      if (!entry.reason) {
        core.warning(`missing-tool entry missing 'reason' field: ${JSON.stringify(entry)}`);
        continue;
      }

      const missingTool = {
        tool: entry.tool,
        reason: entry.reason,
        alternatives: entry.alternatives || null,
        timestamp: new Date().toISOString(),
      };

      missingTools.push(missingTool);
      core.info(`Recorded missing tool: ${missingTool.tool}`);

      // Check max limit
      if (maxReports && missingTools.length >= maxReports) {
        core.info(`Reached maximum number of missing tool reports (${maxReports})`);
        break;
      }
    }
  }

  core.info(`Total missing tools reported: ${missingTools.length}`);

  // Output results
  core.setOutput("tools_reported", JSON.stringify(missingTools));
  core.setOutput("total_count", missingTools.length.toString());

  // Log details for debugging and create step summary
  if (missingTools.length > 0) {
    core.info("Missing tools summary:");

    // Create structured summary for GitHub Actions step summary
    core.summary.addHeading("Missing Tools Report", 3).addRaw(`Found **${missingTools.length}** missing tool${missingTools.length > 1 ? "s" : ""} in this workflow execution.\n\n`);

    missingTools.forEach((tool, index) => {
      core.info(`${index + 1}. Tool: ${tool.tool}`);
      core.info(`   Reason: ${tool.reason}`);
      if (tool.alternatives) {
        core.info(`   Alternatives: ${tool.alternatives}`);
      }
      core.info(`   Reported at: ${tool.timestamp}`);
      core.info("");

      // Add to summary with structured formatting
      core.summary.addRaw(`#### ${index + 1}. \`${tool.tool}\`\n\n`).addRaw(`**Reason:** ${tool.reason}\n\n`);

      if (tool.alternatives) {
        core.summary.addRaw(`**Alternatives:** ${tool.alternatives}\n\n`);
      }

      core.summary.addRaw(`**Reported at:** ${tool.timestamp}\n\n---\n\n`);
    });

    core.summary.write();

    // Create or update issue if configured
    if (createIssue) {
      const runId = context.runId;
      const githubServer = process.env.GITHUB_SERVER_URL || "https://github.com";
      const runUrl = context.payload.repository ? `${context.payload.repository.html_url}/actions/runs/${runId}` : `${githubServer}/${context.repo.owner}/${context.repo.repo}/actions/runs/${runId}`;

      await createOrUpdateIssue(github, context, workflowName, workflowSource, workflowSourceURL, runUrl, titlePrefix, labels, missingTools);
    }
  } else {
    core.info("No missing tools reported in this workflow execution.");
    core.summary.addHeading("Missing Tools Report", 3).addRaw("✅ No missing tools reported in this workflow execution.").write();
  }
}

module.exports = { main };
