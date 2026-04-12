// @ts-check
/// <reference types="@actions/github-script" />

const { getErrorMessage } = require("./error_helpers.cjs");
const { generateFooterWithMessages, generateXMLMarker } = require("./messages_footer.cjs");
const { buildWorkflowRunUrl } = require("./workflow_metadata_helpers.cjs");

/**
 * Run full workflow validation using gh-aw compile --validate and all known
 * checkers (zizmor, actionlint, poutine). If any errors or warnings are found,
 * file a GitHub issue with the findings.
 *
 * Required environment variables:
 *   GH_AW_CMD_PREFIX - Command prefix: './gh-aw' (dev) or 'gh aw' (release)
 *
 * @returns {Promise<void>}
 */
async function main() {
  const owner = context.repo.owner;
  const repo = context.repo.repo;

  const cmdPrefixStr = process.env.GH_AW_CMD_PREFIX || "gh aw";
  const [bin, ...prefixArgs] = cmdPrefixStr.split(" ").filter(Boolean);

  core.info("Running full workflow validation with all known checkers");

  // Run: gh aw compile --validate --no-emit --zizmor --actionlint --poutine --verbose
  const validateArgs = [...prefixArgs, "compile", "--validate", "--no-emit", "--zizmor", "--actionlint", "--poutine", "--verbose"];
  const fullCmd = [bin, ...validateArgs].join(" ");
  core.info(`Running: ${fullCmd}`);

  let stdout = "";
  let stderr = "";
  let exitCode = 0;

  try {
    exitCode = await exec.exec(bin, validateArgs, {
      ignoreReturnCode: true,
      listeners: {
        stdout: data => {
          stdout += data.toString();
        },
        stderr: data => {
          stderr += data.toString();
        },
      },
    });
  } catch (error) {
    core.error(`Validation command failed: ${getErrorMessage(error)}`);
    throw error;
  }

  const combinedOutput = (stderr + "\n" + stdout).trim();

  // Check if there were any errors or warnings
  const hasErrors = exitCode !== 0;
  const hasWarnings = /\bwarn(ing)?\b/i.test(combinedOutput);

  if (!hasErrors && !hasWarnings) {
    core.info("✓ All workflow validations passed with no errors or warnings");
    return;
  }

  if (hasErrors) {
    core.warning(`Validation exited with code ${exitCode}`);
  }
  if (hasWarnings) {
    core.warning("Validation produced warnings");
  }

  // Search for existing open issue about workflow validation
  const issueTitle = "[aw] workflow validation findings";
  const searchQuery = `repo:${owner}/${repo} is:issue is:open in:title "${issueTitle}"`;

  core.info(`Searching for existing issue with title: "${issueTitle}"`);

  const runUrl = buildWorkflowRunUrl(context, context.repo);
  const workflowName = process.env.GH_AW_WORKFLOW_NAME || "Agentic Maintenance";
  const repository = `${owner}/${repo}`;

  try {
    const searchResult = await github.rest.search.issuesAndPullRequests({
      q: searchQuery,
      per_page: 1,
    });

    if (searchResult.data.total_count > 0) {
      const existingIssue = searchResult.data.items[0];
      core.info(`Found existing issue #${existingIssue.number}: ${existingIssue.html_url}`);

      const truncatedOutput = combinedOutput.substring(0, 50000) + (combinedOutput.length > 50000 ? "\n\n... (output truncated)" : "");

      const xmlMarker = generateXMLMarker(workflowName, runUrl);
      const commentBody = `Validation still has findings (exit code: ${exitCode}).

<details>
<summary>Validation output</summary>

\`\`\`
${truncatedOutput}
\`\`\`

</details>

[View workflow run](${runUrl})

---
${xmlMarker}`;

      await github.rest.issues.createComment({
        owner,
        repo,
        issue_number: existingIssue.number,
        body: commentBody,
      });

      core.info(`✓ Added comment to existing issue #${existingIssue.number}`);

      if (hasErrors) {
        core.setFailed(`Workflow validation failed. See issue #${existingIssue.number}`);
      }
      return;
    }
  } catch (error) {
    core.error(`Failed to search for existing issues: ${getErrorMessage(error)}`);
    throw error;
  }

  // No existing issue found, create a new one
  core.info("No existing issue found, creating a new issue with validation findings");

  const truncatedOutput = combinedOutput.substring(0, 50000) + (combinedOutput.length > 50000 ? "\n\n... (output truncated)" : "");

  const xmlMarker = generateXMLMarker(workflowName, runUrl);
  const issueBody = `## Problem

Workflow validation found errors or warnings that need to be addressed.

**Validation command:** \`${fullCmd}\`
**Exit code:** ${exitCode}

## Validation Output

<details>
<summary>Full output</summary>

\`\`\`
${truncatedOutput}
\`\`\`

</details>

## Instructions

Fix the reported issues and re-run validation:

\`\`\`bash
${cmdPrefixStr} compile --validate --no-emit --zizmor --actionlint --poutine --verbose
\`\`\`

Or use the validate shorthand:

\`\`\`bash
${cmdPrefixStr} validate
\`\`\`

## References

- **Repository:** ${repository}
- **Workflow run:** [View run](${runUrl})

---
${xmlMarker}
`;

  try {
    const newIssue = await github.rest.issues.create({
      owner,
      repo,
      title: issueTitle,
      body: issueBody,
      labels: ["agentic-workflows", "maintenance"],
    });

    core.info(`✓ Created issue #${newIssue.data.number}: ${newIssue.data.html_url}`);

    await core.summary.addHeading("Workflow Validation Findings", 2).addRaw(`Created issue [#${newIssue.data.number}](${newIssue.data.html_url}) to track validation findings.`).write();
  } catch (error) {
    core.error(`Failed to create issue: ${getErrorMessage(error)}`);
    throw error;
  }

  if (hasErrors) {
    core.setFailed("Workflow validation failed. See created issue for details.");
  }
}

module.exports = { main };
