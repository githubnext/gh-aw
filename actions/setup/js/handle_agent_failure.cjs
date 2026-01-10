// @ts-check
/// <reference types="@actions/github-script" />

const { getErrorMessage } = require("./error_helpers.cjs");
const { sanitizeContent } = require("./sanitize_content.cjs");
const { generateFooter } = require("./generate_footer.cjs");

/**
 * Handle agent job failure by creating or updating a failure tracking issue
 * This script is called from the conclusion job when the agent job has failed
 */
async function main() {
  try {
    // Get workflow context
    const workflowName = process.env.GH_AW_WORKFLOW_NAME || "unknown";
    const agentConclusion = process.env.GH_AW_AGENT_CONCLUSION || "";
    const runUrl = process.env.GH_AW_RUN_URL || "";
    const workflowSource = process.env.GH_AW_WORKFLOW_SOURCE || "";
    const workflowSourceURL = process.env.GH_AW_WORKFLOW_SOURCE_URL || "";

    core.info(`Agent conclusion: ${agentConclusion}`);
    core.info(`Workflow name: ${workflowName}`);

    // Only proceed if the agent job actually failed
    if (agentConclusion !== "failure") {
      core.info(`Agent job did not fail (conclusion: ${agentConclusion}), skipping failure handling`);
      return;
    }

    const { owner, repo } = context.repo;

    // Sanitize workflow name for title
    const sanitizedWorkflowName = sanitizeContent(workflowName, { maxLength: 100 });
    const issueTitle = `[aw] ${sanitizedWorkflowName} failed`;

    core.info(`Checking for existing issue with title: "${issueTitle}"`);

    // Search for existing open issue with this title and label
    const searchQuery = `repo:${owner}/${repo} is:issue is:open label:agentic-workflows in:title "${issueTitle}"`;

    try {
      const searchResult = await github.rest.search.issuesAndPullRequests({
        q: searchQuery,
        per_page: 1,
      });

      if (searchResult.data.total_count > 0) {
        // Issue exists, add a comment
        const existingIssue = searchResult.data.items[0];
        core.info(`Found existing issue #${existingIssue.number}: ${existingIssue.html_url}`);

        // Generate AI header for the comment
        const footer = generateFooter(
          workflowName,
          runUrl,
          workflowSource,
          workflowSourceURL,
          undefined, // no triggering issue
          undefined, // no triggering PR
          undefined // no triggering discussion
        );

        // Build sanitized comment body
        const timestamp = new Date().toISOString();
        const commentLines = [
          `## Agent Job Failed (${timestamp})`,
          ``,
          `The agent job failed during [workflow run](${runUrl}).`,
          ``,
          `### How to investigate`,
          ``,
          `Use the **debug-agentic-workflow** agent to investigate this failure:`,
          ``,
          `\`\`\`bash`,
          `gh copilot --agent debug-agentic-workflow`,
          `\`\`\``,
          ``,
          `Or in GitHub Copilot Chat, type \`/agent\` and select **debug-agentic-workflow**.`,
          ``,
          `Provide the workflow run URL: ${runUrl}`,
          footer,
        ];

        const commentBody = sanitizeContent(commentLines.join("\n"), { maxLength: 65000 });

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

        // Generate AI header for the issue
        const footer = generateFooter(
          workflowName,
          runUrl,
          workflowSource,
          workflowSourceURL,
          undefined, // no triggering issue
          undefined, // no triggering PR
          undefined // no triggering discussion
        );

        // Build issue body
        const bodyLines = [
          `## Problem`,
          ``,
          `The agentic workflow **${sanitizedWorkflowName}** has failed. This typically indicates a configuration or runtime error that requires user intervention.`,
          ``,
          `### Failed Run`,
          ``,
          `- **Workflow:** [${sanitizedWorkflowName}](${workflowSourceURL})`,
          `- **Failed Run:** ${runUrl}`,
          `- **Source:** ${sanitizeContent(workflowSource, { maxLength: 500 })}`,
          ``,
          `## How to investigate`,
          ``,
          `Use the **debug-agentic-workflow** agent to investigate this failure:`,
          ``,
          `\`\`\`bash`,
          `gh copilot --agent debug-agentic-workflow`,
          `\`\`\``,
          ``,
          `Or in GitHub Copilot Chat, type \`/agent\` and select **debug-agentic-workflow**.`,
          ``,
          `When prompted, provide the workflow run URL: ${runUrl}`,
          ``,
          `The debug agent will help you:`,
          `- Analyze the failure logs`,
          `- Identify the root cause`,
          `- Suggest fixes for configuration or runtime errors`,
          ``,
          `## Common Causes`,
          ``,
          `- Missing or misconfigured tools`,
          `- Invalid workflow configuration`,
          `- Network or connectivity issues`,
          `- Permission problems`,
          `- Resource constraints`,
        ];

        // Add footer (sanitize it separately)
        const sanitizedFooter = sanitizeContent(footer, { maxLength: 5000 });
        bodyLines.push(sanitizedFooter);

        // Add expiration marker (7 days from now) - after sanitization to preserve it
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
          labels: ["agentic-workflows"],
        });

        core.info(`✓ Created new issue #${newIssue.data.number}: ${newIssue.data.html_url}`);
      }
    } catch (error) {
      core.warning(`Failed to create or update failure tracking issue: ${getErrorMessage(error)}`);
      // Don't fail the workflow if we can't create the issue
    }
  } catch (error) {
    core.warning(`Error in handle_agent_failure: ${getErrorMessage(error)}`);
    // Don't fail the workflow
  }
}

module.exports = { main };
