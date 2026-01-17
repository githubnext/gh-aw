// @ts-check
/// <reference types="@actions/github-script" />

const { getErrorMessage } = require("./error_helpers.cjs");
const { sanitizeContent } = require("./sanitize_content.cjs");
const { renderTemplate } = require("./messages_core.cjs");
const { ensureParentIssue, linkSubIssue, findExistingIssue, addIssueComment, createIssue } = require("./issue_helpers.cjs");
const fs = require("fs");

/**
 * Handle missing secret configuration by creating or updating an issue
 * This script is called when secret validation fails in the agent job
 *
 * The script checks for /tmp/gh-aw/missing_secret_info.json file which contains:
 * {
 *   "missing_secrets": ["SECRET_NAME1", "SECRET_NAME2"],
 *   "engine_name": "Claude Code",
 *   "docs_url": "https://docs.anthropic.com"
 * }
 *
 * Expected environment variables:
 * - GH_AW_WORKFLOW_NAME: Name of the workflow
 * - GH_AW_WORKFLOW_SOURCE: Source path of the workflow
 * - GH_AW_WORKFLOW_SOURCE_URL: URL to the workflow source
 * - GH_AW_RUN_URL: URL to the workflow run
 */
async function main() {
  try {
    // Check if missing secret info file exists
    const missingSecretInfoPath = "/tmp/gh-aw/missing_secret_info.json";

    if (!fs.existsSync(missingSecretInfoPath)) {
      core.info("No missing secret info file found, skipping issue creation");
      return;
    }

    // Read missing secret info
    let missingSecretInfo;
    try {
      const fileContent = fs.readFileSync(missingSecretInfoPath, "utf8");
      missingSecretInfo = JSON.parse(fileContent);
      core.info(`Missing secret info loaded: ${JSON.stringify(missingSecretInfo)}`);
    } catch (error) {
      core.warning(`Failed to read or parse missing secret info: ${getErrorMessage(error)}`);
      return;
    }

    // Validate required fields in the JSON
    if (!missingSecretInfo.missing_secrets || !Array.isArray(missingSecretInfo.missing_secrets) || missingSecretInfo.missing_secrets.length === 0) {
      core.warning("Invalid missing_secrets field in missing_secret_info.json");
      return;
    }

    // Get workflow context from environment
    const workflowName = process.env.GH_AW_WORKFLOW_NAME || "unknown";
    const workflowSource = process.env.GH_AW_WORKFLOW_SOURCE || "";
    const workflowSourceURL = process.env.GH_AW_WORKFLOW_SOURCE_URL || "";
    const runUrl = process.env.GH_AW_RUN_URL || "";
    const engineName = missingSecretInfo.engine_name || "unknown";
    const missingSecrets = missingSecretInfo.missing_secrets;
    const docsUrl = missingSecretInfo.docs_url || "https://docs.github.com";

    core.info(`Workflow name: ${workflowName}`);
    core.info(`Engine name: ${engineName}`);
    core.info(`Missing secrets: ${missingSecrets.join(", ")}`);

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
    const issueTitle = `[agentics] ${sanitizedWorkflowName} missing secret`;

    core.info(`Checking for existing issue with title: "${issueTitle}"`);

    // Search for existing issue using helper
    const existingIssue = await findExistingIssue(issueTitle, "agentic-workflows");

    try {
      if (existingIssue) {
        // Issue exists, add a comment
        core.info(`Found existing issue #${existingIssue.number}: ${existingIssue.html_url}`);

        // Read comment template
        const commentTemplatePath = "/opt/gh-aw/prompts/missing_secret_comment.md";
        const commentTemplate = fs.readFileSync(commentTemplatePath, "utf8");

        // Extract run ID from URL (e.g., https://github.com/owner/repo/actions/runs/123 -> "123")
        let runId = "";
        const runIdMatch = runUrl.match(/\/actions\/runs\/(\d+)/);
        if (runIdMatch) {
          runId = runIdMatch[1];
        }

        // Build missing secrets list
        const missingSecretsListLines = [];
        missingSecrets.forEach(secretName => {
          missingSecretsListLines.push(`- \`${secretName}\``);
        });
        const missingSecretsList = missingSecretsListLines.join("\n");

        // Create template context
        const templateContext = {
          run_url: runUrl,
          run_id: runId,
          workflow_name: workflowName,
          workflow_source_url: workflowSourceURL || "#",
          engine_name: engineName,
          docs_url: docsUrl,
          missing_secrets_list: missingSecretsList,
          repo_owner: owner,
          repo_name: repo,
        };

        // Render the comment template
        const commentBody = renderTemplate(commentTemplate, templateContext);

        // Sanitize and add comment
        const fullCommentBody = sanitizeContent(commentBody, { maxLength: 65000 });
        await addIssueComment(existingIssue.number, fullCommentBody);
      } else {
        // No existing issue, create a new one
        core.info("No existing issue found, creating a new one");

        // Read issue template
        const issueTemplatePath = "/opt/gh-aw/prompts/missing_secret_issue.md";
        const issueTemplate = fs.readFileSync(issueTemplatePath, "utf8");

        // Build missing secrets list
        const missingSecretsListLines = [];
        missingSecrets.forEach(secretName => {
          missingSecretsListLines.push(`#### \`${secretName}\``);
          missingSecretsListLines.push(``);
          missingSecretsListLines.push(`**Status:** ❌ Not configured`);
          missingSecretsListLines.push(`**Documentation:** [${docsUrl}](${docsUrl})`);
          missingSecretsListLines.push(``);
        });
        const missingSecretsList = missingSecretsListLines.join("\n");

        // Create template context
        const templateContext = {
          workflow_name: sanitizedWorkflowName,
          run_url: runUrl,
          workflow_source_url: workflowSourceURL || "#",
          workflow_source: workflowSource,
          engine_name: engineName,
          docs_url: docsUrl,
          missing_secrets_list: missingSecretsList,
          repo_owner: owner,
          repo_name: repo,
        };

        // Render the issue template
        const issueBodyContent = renderTemplate(issueTemplate, templateContext);

        // Add expiration marker (7 days from now)
        const expirationDate = new Date();
        expirationDate.setDate(expirationDate.getDate() + 7);
        const issueBody = `${issueBodyContent}\n\n<!-- gh-aw-expires: ${expirationDate.toISOString()} -->`;

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
      core.warning(`Failed to create or update missing secret issue: ${getErrorMessage(error)}`);
      // Don't fail the workflow if we can't create the issue
    }
  } catch (error) {
    core.warning(`Error in handle_missing_secret: ${getErrorMessage(error)}`);
    // Don't fail the workflow
  }
}

module.exports = { main };
