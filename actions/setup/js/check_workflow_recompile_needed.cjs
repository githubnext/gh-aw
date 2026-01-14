// @ts-check
/// <reference types="@actions/github-script" />

const { getErrorMessage } = require("./error_helpers.cjs");
const { generateFooterWithMessages, getFooterWorkflowRecompileMessage, getFooterWorkflowRecompileCommentMessage, generateXMLMarker } = require("./messages_footer.cjs");
const fs = require("fs");

/**
 * Check if workflows need recompilation and create an issue if needed.
 * This script:
 * 1. Checks if there are out-of-sync workflow lock files
 * 2. Searches for existing open issues about recompiling workflows
 * 3. If workflows are out of sync and no issue exists, creates a new issue with agentic instructions
 *
 * @returns {Promise<void>}
 */
async function main() {
  const owner = context.repo.owner;
  const repo = context.repo.repo;

  core.info("Checking for out-of-sync workflow lock files");

  // Execute git diff to check for changes in lock files
  let diffOutput = "";
  let hasChanges = false;

  try {
    // Run git diff to check if there are any changes in lock files
    await exec.exec("git", ["diff", "--exit-code", ".github/workflows/*.lock.yml"], {
      ignoreReturnCode: true,
      listeners: {
        stdout: data => {
          diffOutput += data.toString();
        },
        stderr: data => {
          diffOutput += data.toString();
        },
      },
    });

    // If git diff exits with code 0, there are no changes
    // If it exits with code 1, there are changes
    // We need to check if there's actual diff output
    hasChanges = diffOutput.trim().length > 0;
  } catch (error) {
    core.error(`Failed to check for workflow changes: ${getErrorMessage(error)}`);
    throw error;
  }

  if (!hasChanges) {
    core.info("✓ All workflow lock files are up to date");
    return;
  }

  core.info("⚠ Detected out-of-sync workflow lock files");

  // Capture the actual diff for the issue body
  let detailedDiff = "";
  try {
    await exec.exec("git", ["diff", ".github/workflows/*.lock.yml"], {
      listeners: {
        stdout: data => {
          detailedDiff += data.toString();
        },
      },
    });
  } catch (error) {
    core.warning(`Could not capture detailed diff: ${getErrorMessage(error)}`);
  }

  // Search for existing open issue about workflow recompilation
  const issueTitle = "[agentics] agentic workflows out of sync";
  const searchQuery = `repo:${owner}/${repo} is:issue is:open in:title "${issueTitle}"`;

  core.info(`Searching for existing issue with title: "${issueTitle}"`);

  try {
    const searchResult = await github.rest.search.issuesAndPullRequests({
      q: searchQuery,
      per_page: 1,
    });

    if (searchResult.data.total_count > 0) {
      const existingIssue = searchResult.data.items[0];
      core.info(`Found existing issue #${existingIssue.number}: ${existingIssue.html_url}`);
      core.info("Skipping issue creation (avoiding duplicate)");

      // Add a comment to the existing issue with the new workflow run info
      const githubServer = process.env.GITHUB_SERVER_URL || "https://github.com";
      const runUrl = context.payload.repository ? `${context.payload.repository.html_url}/actions/runs/${context.runId}` : `${githubServer}/${owner}/${repo}/actions/runs/${context.runId}`;

      // Get workflow metadata for footer
      const workflowName = process.env.GH_AW_WORKFLOW_NAME || "Agentic Maintenance";
      const repository = `${owner}/${repo}`;

      // Create custom footer for workflow recompile comment
      const ctx = {
        workflowName,
        runUrl,
        repository,
      };

      const footer = getFooterWorkflowRecompileCommentMessage(ctx);
      const xmlMarker = generateXMLMarker(workflowName, runUrl);

      const commentBody = `Workflows are still out of sync.\n\n---\n${footer}\n\n${xmlMarker}`;

      await github.rest.issues.createComment({
        owner,
        repo,
        issue_number: existingIssue.number,
        body: commentBody,
      });

      core.info(`✓ Added comment to existing issue #${existingIssue.number}`);
      return;
    }
  } catch (error) {
    core.error(`Failed to search for existing issues: ${getErrorMessage(error)}`);
    throw error;
  }

  // No existing issue found, create a new one
  core.info("No existing issue found, creating a new issue with agentic instructions");

  const githubServer = process.env.GITHUB_SERVER_URL || "https://github.com";
  const runUrl = context.payload.repository ? `${context.payload.repository.html_url}/actions/runs/${context.runId}` : `${githubServer}/${owner}/${repo}/actions/runs/${context.runId}`;

  // Read the issue template from the prompts directory
  // Allow override via environment variable for testing
  const promptsDir = process.env.GH_AW_PROMPTS_DIR || "/opt/gh-aw/prompts";
  const templatePath = `${promptsDir}/workflow_recompile_issue.md`;
  let issueTemplate;
  try {
    issueTemplate = fs.readFileSync(templatePath, "utf8");
  } catch (error) {
    core.error(`Failed to read issue template from ${templatePath}: ${getErrorMessage(error)}`);
    throw error;
  }

  // Replace placeholders in the template
  const diffContent = detailedDiff.substring(0, 50000) + (detailedDiff.length > 50000 ? "\n\n... (diff truncated)" : "");
  const repository = `${owner}/${repo}`;

  let issueBody = issueTemplate.replace("{DIFF_CONTENT}", diffContent).replace("{REPOSITORY}", repository);

  // Get workflow metadata for footer
  const workflowName = process.env.GH_AW_WORKFLOW_NAME || "Agentic Maintenance";

  // Create custom footer for workflow recompile issues
  const ctx = {
    workflowName,
    runUrl,
    repository,
  };

  // Use custom footer template if configured, with XML marker for traceability
  const footer = getFooterWorkflowRecompileMessage(ctx);
  const xmlMarker = generateXMLMarker(workflowName, runUrl);
  issueBody += "\n\n---\n" + footer + "\n\n" + xmlMarker + "\n";

  try {
    const newIssue = await github.rest.issues.create({
      owner,
      repo,
      title: issueTitle,
      body: issueBody,
      labels: ["maintenance", "workflows"],
    });

    core.info(`✓ Created issue #${newIssue.data.number}: ${newIssue.data.html_url}`);

    // Write to job summary
    await core.summary.addHeading("Workflow Recompilation Needed", 2).addRaw(`Created issue [#${newIssue.data.number}](${newIssue.data.html_url}) to track workflow recompilation.`).write();
  } catch (error) {
    core.error(`Failed to create issue: ${getErrorMessage(error)}`);
    throw error;
  }
}

module.exports = { main };
