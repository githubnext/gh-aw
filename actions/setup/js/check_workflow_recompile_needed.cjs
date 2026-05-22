// @ts-check
/// <reference types="@actions/github-script" />

const { getErrorMessage } = require("./error_helpers.cjs");
const { getFooterWorkflowRecompileMessage, getFooterWorkflowRecompileCommentMessage, generateXMLMarker, getDetectionCautionAlert } = require("./messages_footer.cjs");
const fs = require("fs");
const { getGitAuthEnv } = require("./git_helpers.cjs");
const { buildWorkflowRunUrl } = require("./workflow_metadata_helpers.cjs");

const RECOMPILE_ISSUE_TITLE = "[aw] agentic workflows out of sync";
const RECOMPILE_PR_TITLE = "[aw] recompile agentic workflows";
const RECOMPILE_PR_BRANCH = "aw/recompile-workflows";
const MAX_RECOMPILE_DIFF_LENGTH = 50000;

function shouldCreatePullRequest() {
  return process.env.GH_AW_WORKFLOW_RECOMPILE_CREATE_PULL_REQUEST === "true";
}

async function execWithOutput(command, args, options = {}) {
  let stdout = "";
  let stderr = "";
  const exitCode = await exec.exec(command, args, {
    ignoreReturnCode: options.ignoreReturnCode === true,
    listeners: {
      stdout: data => {
        stdout += data.toString();
      },
      stderr: data => {
        stderr += data.toString();
      },
    },
  });
  return { stdout, stderr, exitCode };
}

function getDefaultBranch() {
  return context.payload?.repository?.default_branch || "main";
}

function getRecompileToken() {
  return process.env.GH_AW_MAINTENANCE_GITHUB_TOKEN || "";
}

function logConfiguration(createPullRequest) {
  core.info(`Workflow recompile mode: ${createPullRequest ? "pull-request" : "issue"}`);
  core.info(`Configured maintenance token present: ${getRecompileToken() !== ""}`);
}

function requireRecompileToken() {
  const token = getRecompileToken();
  if (!token) {
    throw new Error("Missing configured maintenance GitHub token secret for maintenance compile pull request creation");
  }
  return token;
}

function buildRecompilePullRequestBody(diffContent, changedFiles, repository, runUrl) {
  const workflowName = process.env.GH_AW_WORKFLOW_NAME || "Agentic Maintenance";
  const footer = getFooterWorkflowRecompileMessage({ workflowName, runUrl, repository });
  const xmlMarker = generateXMLMarker(workflowName, runUrl);
  const detectionCaution = getDetectionCautionAlert(workflowName, runUrl);
  const cautionPrefix = detectionCaution ? `${detectionCaution}\n\n` : "";
  const fileList = changedFiles.map(file => `- \`${file}\``).join("\n");

  return `${cautionPrefix}## Workflow Recompilation

This automated maintenance run detected generated workflow changes and prepared this pull request to update the lock files.

## Changed Files

${fileList}

## Detected Changes

<details>
<summary>View diff</summary>

\`\`\`diff
${diffContent}
\`\`\`

</details>

---
${footer}

${xmlMarker}
`;
}

async function configureGitIdentity() {
  core.info("Configuring git identity for maintenance workflow commit");
  await exec.exec("git", ["config", "user.email", "github-actions[bot]@users.noreply.github.com"]);
  await exec.exec("git", ["config", "user.name", "github-actions[bot]"]);
}

async function stageAndCommitRecompileBranch() {
  core.info(`Preparing maintenance branch ${RECOMPILE_PR_BRANCH}`);
  await configureGitIdentity();
  await exec.exec("git", ["checkout", "-B", RECOMPILE_PR_BRANCH]);
  await exec.exec("git", ["add", ".github/workflows/*.lock.yml"]);

  const { stdout: stagedOutput } = await execWithOutput("git", ["diff", "--cached", "--name-only"], { ignoreReturnCode: true });
  const stagedFiles = stagedOutput
    .split("\n")
    .map(file => file.trim())
    .filter(Boolean);
  if (stagedFiles.length === 0) {
    throw new Error("No staged workflow lock file changes found for pull request creation");
  }

  core.info(`Staged ${stagedFiles.length} workflow lock file(s): ${stagedFiles.join(", ")}`);
  await exec.exec("git", ["commit", "-m", "chore: recompile agentic workflows"]);
  return stagedFiles;
}

async function pushRecompileBranch(owner, repo, branchName) {
  const token = requireRecompileToken();

  const githubServerUrl = process.env.GITHUB_SERVER_URL || "https://github.com";
  let githubHost = "github.com";
  try {
    githubHost = new URL(githubServerUrl).hostname || githubHost;
  } catch {
    githubHost = "github.com";
  }
  const remoteUrl = `https://${githubHost}/${owner}/${repo}.git`;
  core.info(`Pushing maintenance branch ${branchName} to ${githubHost}/${owner}/${repo}`);
  await exec.exec("git", ["push", remoteUrl, `HEAD:refs/heads/${branchName}`, "--force-with-lease"], {
    env: {
      ...process.env,
      ...getGitAuthEnv(token),
    },
  });
}

async function findExistingRecompilePullRequest(owner, repo) {
  core.info(`Searching for an existing maintenance PR from branch ${owner}:${RECOMPILE_PR_BRANCH}`);
  const result = await github.rest.pulls.list({
    owner,
    repo,
    state: "open",
    head: `${owner}:${RECOMPILE_PR_BRANCH}`,
    per_page: 1,
  });
  return result.data[0] || null;
}

async function handlePullRequest(owner, repo, detailedDiff) {
  const repository = `${owner}/${repo}`;
  const runUrl = buildWorkflowRunUrl(context, context.repo);
  core.info(`Preparing maintenance PR for ${repository}`);
  const changedFiles = await stageAndCommitRecompileBranch();
  await pushRecompileBranch(owner, repo, RECOMPILE_PR_BRANCH);

  const existingPR = await findExistingRecompilePullRequest(owner, repo);
  if (existingPR) {
    core.info(`Found existing pull request #${existingPR.number}: ${existingPR.html_url}`);
    core.info("Updated existing pull request branch (avoiding duplicate)");
    await core.summary.addHeading("Workflow Recompilation Needed", 2).addRaw(`Updated existing pull request [#${existingPR.number}](${existingPR.html_url}) with the latest compiled workflow changes.`).write();
    return;
  }

  const diffContent = detailedDiff.substring(0, MAX_RECOMPILE_DIFF_LENGTH) + (detailedDiff.length > MAX_RECOMPILE_DIFF_LENGTH ? "\n\n... (diff truncated)" : "");
  core.info(`Creating maintenance pull request against ${getDefaultBranch()} with ${changedFiles.length} changed file(s)`);
  const pullRequest = await github.rest.pulls.create({
    owner,
    repo,
    title: RECOMPILE_PR_TITLE,
    head: RECOMPILE_PR_BRANCH,
    base: getDefaultBranch(),
    body: buildRecompilePullRequestBody(diffContent, changedFiles, repository, runUrl),
  });

  core.info(`✓ Created pull request #${pullRequest.data.number}: ${pullRequest.data.html_url}`);
  await core.summary.addHeading("Workflow Recompilation Needed", 2).addRaw(`Created pull request [#${pullRequest.data.number}](${pullRequest.data.html_url}) to update compiled workflow lock files.`).write();
}

/**
 * Check if workflows need recompilation and create an issue or pull request if needed.
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
  const createPullRequest = shouldCreatePullRequest();

  core.info("Checking for out-of-sync workflow lock files");
  logConfiguration(createPullRequest);

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
  core.info(`Workflow diff size from detection step: ${diffOutput.length} byte(s)`);

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
  core.info(`Detailed workflow diff captured: ${detailedDiff.length} byte(s)`);

  if (createPullRequest) {
    requireRecompileToken();
    await handlePullRequest(owner, repo, detailedDiff);
    return;
  }

  // Search for existing open issue about workflow recompilation
  const searchQuery = `repo:${owner}/${repo} is:issue is:open in:title "${RECOMPILE_ISSUE_TITLE}"`;

  core.info(`Searching for existing issue with title: "${RECOMPILE_ISSUE_TITLE}"`);

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
      const runUrl = buildWorkflowRunUrl(context, context.repo);

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

      // Inject CAUTION at top of body if threat detection warning was raised
      const detectionCaution = getDetectionCautionAlert(workflowName, runUrl);
      const cautionPrefix = detectionCaution ? detectionCaution + "\n\n" : "";

      // Sanitize the message text but not the footer/marker which are system-generated
      const commentBody = `${cautionPrefix}Workflows are still out of sync.\n\n---\n${footer}\n\n${xmlMarker}`;

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

  const runUrl = buildWorkflowRunUrl(context, context.repo);

  // Read the issue template from the prompts directory
  // Allow override via environment variable for testing
  const promptsDir = process.env.GH_AW_PROMPTS_DIR || `${process.env.RUNNER_TEMP}/gh-aw/prompts`;
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

  // Inject CAUTION at top of body if threat detection warning was raised
  const detectionCaution = getDetectionCautionAlert(workflowName, runUrl);
  if (detectionCaution) {
    issueBody = detectionCaution + "\n\n" + issueBody;
  }

  // Note: issueBody is built from a template render, no user content to sanitize
  issueBody += "\n\n---\n" + footer + "\n\n" + xmlMarker + "\n";

  try {
    const newIssue = await github.rest.issues.create({
      owner,
      repo,
      title: RECOMPILE_ISSUE_TITLE,
      body: issueBody,
      labels: ["agentic-workflows", "maintenance"],
    });

    core.info(`✓ Created issue #${newIssue.data.number}: ${newIssue.data.html_url}`);

    // Write to job summary
    await core.summary.addHeading("Workflow Recompilation Needed", 2).addRaw(`Created issue [#${newIssue.data.number}](${newIssue.data.html_url}) to track workflow recompilation.`).write();
  } catch (error) {
    core.error(`Failed to create issue: ${getErrorMessage(error)}`);
    throw error;
  }
}

module.exports = { main, buildRecompilePullRequestBody, shouldCreatePullRequest };
