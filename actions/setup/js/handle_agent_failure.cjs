// @ts-check
/// <reference types="@actions/github-script" />

const { getErrorMessage } = require("./error_helpers.cjs");
const { sanitizeContent } = require("./sanitize_content.cjs");
const { getFooterAgentFailureIssueMessage, getFooterAgentFailureCommentMessage, generateXMLMarker } = require("./messages.cjs");
const { renderTemplate } = require("./messages_core.cjs");
const fs = require("fs");

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

        // Read comment template
        const commentTemplatePath = "/opt/gh-aw/prompts/agent_failure_comment.md";
        let commentTemplate;
        try {
          commentTemplate = fs.readFileSync(commentTemplatePath, "utf8");
        } catch (error) {
          // Fallback for tests or if template file is missing
          core.warning(`Could not read comment template from ${commentTemplatePath}, using fallback: ${getErrorMessage(error)}`);
          commentTemplate = `Agent job failed: {run_url}`;
        }

        // Create template context
        const templateContext = {
          run_url: runUrl,
          workflow_name: workflowName,
          workflow_source: workflowSource,
          workflow_source_url: workflowSourceURL,
        };

        // Render the comment template
        const commentBody = renderTemplate(commentTemplate, templateContext);

        // Generate footer for the comment using templated message
        const ctx = {
          workflowName,
          runUrl,
          workflowSource,
          workflowSourceUrl: workflowSourceURL,
        };
        const footer = getFooterAgentFailureCommentMessage(ctx);

        // Combine comment body with footer
        const fullCommentBody = sanitizeContent(commentBody + "\n\n" + footer, { maxLength: 65000 });

        await github.rest.issues.createComment({
          owner,
          repo,
          issue_number: existingIssue.number,
          body: fullCommentBody,
        });

        core.info(`✓ Added comment to existing issue #${existingIssue.number}`);
      } else {
        // No existing issue, create a new one
        core.info("No existing issue found, creating a new one");

        // Read issue template
        const issueTemplatePath = "/opt/gh-aw/prompts/agent_failure_issue.md";
        let issueTemplate;
        try {
          issueTemplate = fs.readFileSync(issueTemplatePath, "utf8");
        } catch (error) {
          // Fallback for tests or if template file is missing
          core.warning(`Could not read issue template from ${issueTemplatePath}, using fallback: ${getErrorMessage(error)}`);
          issueTemplate = `## Problem

The agentic workflow **{workflow_name}** has failed. This typically indicates a configuration or runtime error that requires user intervention.

## Failed Run

- **Workflow:** [{workflow_name}]({workflow_source_url})
- **Failed Run:** {run_url}
- **Source:** {workflow_source}

## How to investigate

Use the **debug-agentic-workflow** agent to investigate this failure.

In GitHub Copilot Chat, type \`/agent\` and select **debug-agentic-workflow**.

When prompted, provide the workflow run URL: {run_url}

The debug agent will help you:
- Analyze the failure logs
- Identify the root cause
- Suggest fixes for configuration or runtime errors`;
        }

        // Create template context with sanitized workflow name
        const templateContext = {
          workflow_name: sanitizedWorkflowName,
          run_url: runUrl,
          workflow_source: sanitizeContent(workflowSource, { maxLength: 500 }),
          workflow_source_url: workflowSourceURL || "#",
        };

        // Render the issue template
        const issueBodyContent = renderTemplate(issueTemplate, templateContext);

        // Generate footer for the issue using templated message
        const ctx = {
          workflowName,
          runUrl,
          workflowSource,
          workflowSourceUrl: workflowSourceURL,
        };
        const footer = getFooterAgentFailureIssueMessage(ctx);

        // Combine issue body with footer, expiration marker, and XML marker
        const bodyLines = [issueBodyContent, "", footer];

        // Add expiration marker (7 days from now)
        const expirationDate = new Date();
        expirationDate.setDate(expirationDate.getDate() + 7);
        bodyLines.push(``);
        bodyLines.push(`<!-- gh-aw-expires: ${expirationDate.toISOString()} -->`);

        // Add XML marker for traceability
        bodyLines.push(``);
        bodyLines.push(generateXMLMarker(workflowName, runUrl));

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
