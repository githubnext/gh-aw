// @ts-check
/// <reference types="@actions/github-script" />

const { getErrorMessage } = require("./error_helpers.cjs");

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
  const issueTitle = "Workflows need recompilation";
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
      const commentBody = `Workflows are still out of sync as of ${new Date().toISOString()}.\n\nSee [workflow run](${runUrl}) for details.`;

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

  // Build the issue body with agentic instructions
  const issueBody = `## Problem

The workflow lock files (\`.lock.yml\`) are out of sync with their source markdown files (\`.md\`). This means the workflows that run in GitHub Actions are not using the latest configuration.

## What needs to be done

The workflows need to be recompiled to regenerate the lock files from the markdown sources.

## Instructions for GitHub Copilot

Please recompile all workflows by running the following command:

\`\`\`bash
make recompile
\`\`\`

This will:
1. Build the latest version of \`gh-aw\`
2. Compile all workflow markdown files to YAML lock files
3. Ensure all workflows are up to date

After recompiling, commit the changes with a message like:
\`\`\`
Recompile workflows to update lock files
\`\`\`

## Detected Changes

The following workflow lock files have changes:

<details>
<summary>View diff</summary>

\`\`\`diff
${detailedDiff.substring(0, 50000)}${detailedDiff.length > 50000 ? "\n\n... (diff truncated)" : ""}
\`\`\`

</details>

## References

- **Failed Check:** [Workflow Run](${runUrl})
- **Repository:** ${owner}/${repo}

---

> This issue was automatically created by the agentics maintenance workflow.
`;

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
