// @ts-check
/// <reference types="@actions/github-script" />

const { getErrorMessage } = require("./error_helpers.cjs");
const { sanitizeContent } = require("./sanitize_content.cjs");
const { getFooterAgentFailureIssueMessage, getFooterAgentFailureCommentMessage, generateXMLMarker } = require("./messages.cjs");
const { renderTemplate } = require("./messages_core.cjs");
const { getCurrentBranch } = require("./get_current_branch.cjs");
const { ensureParentIssue, linkSubIssue, findExistingIssue, addIssueComment, createIssue } = require("./issue_helpers.cjs");
const fs = require("fs");

/**
 * Attempt to find a pull request for the current branch
 * @returns {Promise<{number: number, html_url: string} | null>} PR info or null if not found
 */
async function findPullRequestForCurrentBranch() {
  try {
    const { owner, repo } = context.repo;
    const currentBranch = getCurrentBranch();

    core.info(`Searching for pull request from branch: ${currentBranch}`);

    // Search for open PRs with the current branch as head
    const searchQuery = `repo:${owner}/${repo} is:pr is:open head:${currentBranch}`;

    const searchResult = await github.rest.search.issuesAndPullRequests({
      q: searchQuery,
      per_page: 1,
    });

    if (searchResult.data.total_count > 0) {
      const pr = searchResult.data.items[0];
      core.info(`Found pull request #${pr.number}: ${pr.html_url}`);
      return {
        number: pr.number,
        html_url: pr.html_url,
      };
    }

    core.info(`No pull request found for branch: ${currentBranch}`);
    return null;
  } catch (error) {
    core.warning(`Failed to find pull request for current branch: ${getErrorMessage(error)}`);
    return null;
  }
}

// Note: ensureParentIssue and linkSubIssue functions are now imported from issue_helpers.cjs

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

    // Try to find a pull request for the current branch
    const pullRequest = await findPullRequestForCurrentBranch();

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
    const issueTitle = `[agentics] ${sanitizedWorkflowName} failed`;

    core.info(`Checking for existing issue with title: "${issueTitle}"`);

    // Search for existing issue using helper
    const existingIssue = await findExistingIssue(issueTitle, "agentic-workflows");

    try {
      if (existingIssue) {
        // Issue exists, add a comment
        core.info(`Found existing issue #${existingIssue.number}: ${existingIssue.html_url}`);

        // Read comment template
        const commentTemplatePath = "/opt/gh-aw/prompts/agent_failure_comment.md";
        const commentTemplate = fs.readFileSync(commentTemplatePath, "utf8");

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

        await addIssueComment(existingIssue.number, fullCommentBody);
      } else {
        // No existing issue, create a new one
        core.info("No existing issue found, creating a new one");

        // Read issue template
        const issueTemplatePath = "/opt/gh-aw/prompts/agent_failure_issue.md";
        const issueTemplate = fs.readFileSync(issueTemplatePath, "utf8");

        // Get current branch information
        const currentBranch = getCurrentBranch();

        // Create template context with sanitized workflow name
        const templateContext = {
          workflow_name: sanitizedWorkflowName,
          run_url: runUrl,
          workflow_source_url: workflowSourceURL || "#",
          branch: currentBranch,
          pull_request_info: pullRequest ? `  \n**Pull Request:** [#${pullRequest.number}](${pullRequest.html_url})` : "",
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

        const newIssue = await createIssue(issueTitle, issueBody, ["agentic-workflows"]);

        // Link as sub-issue to parent if parent issue was created
        if (parentIssue) {
          try {
            await linkSubIssue(parentIssue.node_id, newIssue.node_id, parentIssue.number, newIssue.number);
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
