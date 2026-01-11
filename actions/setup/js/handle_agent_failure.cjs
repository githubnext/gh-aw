// @ts-check
/// <reference types="@actions/github-script" />

const { getErrorMessage } = require("./error_helpers.cjs");
const { sanitizeContent } = require("./sanitize_content.cjs");
const { getFooterAgentFailureIssueMessage, getFooterAgentFailureCommentMessage, generateXMLMarker } = require("./messages.cjs");
const { renderTemplate } = require("./messages_core.cjs");
const fs = require("fs");

/**
 * Search for or create the parent issue for all agentic workflow failures
 * @returns {Promise<{number: number, node_id: string}>} Parent issue number and node ID
 */
async function ensureParentIssue() {
  const { owner, repo } = context.repo;
  const parentTitle = "[aw] Agentic Workflow Issues";
  const parentLabel = "agentic-workflows";

  core.info(`Searching for parent issue: "${parentTitle}"`);

  // Search for existing parent issue
  const searchQuery = `repo:${owner}/${repo} is:issue is:open label:${parentLabel} in:title "${parentTitle}"`;

  try {
    const searchResult = await github.rest.search.issuesAndPullRequests({
      q: searchQuery,
      per_page: 1,
    });

    if (searchResult.data.total_count > 0) {
      const existingIssue = searchResult.data.items[0];
      core.info(`Found existing parent issue #${existingIssue.number}: ${existingIssue.html_url}`);
      return {
        number: existingIssue.number,
        node_id: existingIssue.node_id,
      };
    }
  } catch (error) {
    core.warning(`Error searching for parent issue: ${getErrorMessage(error)}`);
  }

  // Create parent issue if it doesn't exist
  core.info("No parent issue found, creating one");

  const parentBody = `# Agentic Workflow Failures

This issue tracks all failures from agentic workflows in this repository. Each failed workflow run creates a sub-issue linked here for organization and easy filtering.

## Purpose

This parent issue helps you:
- View all workflow failures in one place by checking the sub-issues below
- Filter out failure issues from your main issue list using \`no:parent-issue\`
- Track the health of your agentic workflows over time

## Sub-Issues

All individual workflow failure issues are linked as sub-issues below. Click on any sub-issue to see details about a specific failure.

## Troubleshooting Failed Workflows

### Using debug-agentic-workflow Agent (Recommended)

The fastest way to investigate a failure is with the **debug-agentic-workflow** custom agent:

1. In GitHub Copilot Chat, type \`/agent\` and select **debug-agentic-workflow**
2. When prompted, provide the workflow run URL
3. The agent will help you analyze logs, identify root causes, and suggest fixes

### Using gh-aw CLI

You can also debug failures using the \`gh-aw\` CLI:

\`\`\`bash
# Download and analyze workflow logs
gh aw logs <workflow-run-url>

# Audit a specific workflow run
gh aw audit <run-id>
\`\`\`

### Manual Investigation

1. Click on a sub-issue to see the failed workflow details
2. Follow the workflow run link in the issue
3. Review the agent job logs for error messages
4. Check the workflow configuration in your repository

## Resources

- [GitHub Agentic Workflows Documentation](https://github.com/githubnext/gh-aw)
- [Troubleshooting Guide](https://github.com/githubnext/gh-aw/blob/main/docs/troubleshooting.md)

---

> This issue is automatically managed by GitHub Agentic Workflows. Do not close this issue manually.`;

  try {
    const newIssue = await github.rest.issues.create({
      owner,
      repo,
      title: parentTitle,
      body: parentBody,
      labels: [parentLabel],
    });

    core.info(`✓ Created parent issue #${newIssue.data.number}: ${newIssue.data.html_url}`);
    return {
      number: newIssue.data.number,
      node_id: newIssue.data.node_id,
    };
  } catch (error) {
    core.error(`Failed to create parent issue: ${getErrorMessage(error)}`);
    throw error;
  }
}

/**
 * Link an issue as a sub-issue to a parent issue
 * @param {string} parentNodeId - GraphQL node ID of the parent issue
 * @param {string} subIssueNodeId - GraphQL node ID of the sub-issue
 * @param {number} parentNumber - Parent issue number (for logging)
 * @param {number} subIssueNumber - Sub-issue number (for logging)
 */
async function linkSubIssue(parentNodeId, subIssueNodeId, parentNumber, subIssueNumber) {
  core.info(`Linking issue #${subIssueNumber} as sub-issue of #${parentNumber}`);

  try {
    // Use GraphQL to link the sub-issue
    await github.graphql(
      `mutation($parentId: ID!, $subIssueId: ID!) {
        addSubIssue(input: {issueId: $parentId, subIssueId: $subIssueId}) {
          issue {
            id
            number
          }
          subIssue {
            id
            number
          }
        }
      }`,
      {
        parentId: parentNodeId,
        subIssueId: subIssueNodeId,
      }
    );

    core.info(`✓ Successfully linked #${subIssueNumber} as sub-issue of #${parentNumber}`);
  } catch (error) {
    const errorMessage = getErrorMessage(error);
    if (errorMessage.includes("Field 'addSubIssue' doesn't exist") || errorMessage.includes("not yet available")) {
      core.warning(`Sub-issue API not available. Issue #${subIssueNumber} created but not linked to parent.`);
    } else {
      core.warning(`Failed to link sub-issue: ${errorMessage}`);
    }
  }
}

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

    // Ensure parent issue exists first
    let parentIssue;
    try {
      parentIssue = await ensureParentIssue();
    } catch (error) {
      core.warning(`Could not create parent issue, proceeding without parent: ${getErrorMessage(error)}`);
      // Continue without parent issue
    }

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
          commentTemplate = `Agent job [{run_id}]({run_url}) failed.`;
        }

        // Extract run ID from URL (e.g., https://github.com/owner/repo/actions/runs/123 -> "123")
        let runId = "";
        const runIdMatch = runUrl.match(/\/actions\/runs\/(\d+)/);
        if (runIdMatch) {
          runId = runIdMatch[1];
        }

        // Create template context
        const templateContext = {
          run_url: runUrl,
          run_id: runId,
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

        // Link as sub-issue to parent if parent issue was created
        if (parentIssue) {
          try {
            await linkSubIssue(parentIssue.node_id, newIssue.data.node_id, parentIssue.number, newIssue.data.number);
          } catch (error) {
            core.warning(`Could not link issue as sub-issue: ${getErrorMessage(error)}`);
            // Continue even if linking fails
          }
        }
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
