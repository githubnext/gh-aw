// @ts-check
/// <reference types="@actions/github-script" />

/**
 * Check workflow file timestamps to detect outdated lock files
 * This script uses the GitHub REST API to compare the commit timestamps
 * of the source .md file with the compiled .lock.yml file
 */

const path = require("path");

async function main() {
  const workflow = process.env.GITHUB_WORKFLOW;

  if (!workflow) {
    core.setFailed("Configuration error: GITHUB_WORKFLOW not available.");
    return;
  }

  // Construct file paths (relative to repository root)
  const workflowBasename = path.basename(workflow, ".lock.yml");
  const workflowPath = `.github/workflows/${workflowBasename}.md`;
  const lockPath = `.github/workflows/${workflow}`;

  core.info(`Checking workflow timestamps using GitHub API:`);
  core.info(`  Source: ${workflowPath}`);
  core.info(`  Lock file: ${lockPath}`);

  const owner = context.repo.owner;
  const repo = context.repo.repo;
  const ref = context.ref || context.sha;

  core.info(`  Repository: ${owner}/${repo}`);
  core.info(`  Ref: ${ref}`);

  // Get the latest commit for the workflow source file
  let workflowCommitDate = null;
  try {
    const workflowCommits = await github.rest.repos.listCommits({
      owner: owner,
      repo: repo,
      path: workflowPath,
      per_page: 1,
    });

    if (workflowCommits.data.length > 0) {
      const latestCommit = workflowCommits.data[0];
      workflowCommitDate = new Date(latestCommit.commit.committer.date);
      core.info(`  Source last commit: ${workflowCommitDate.toISOString()} (${latestCommit.sha.substring(0, 7)})`);
    } else {
      core.info(`  Source file not found in repository: ${workflowPath}`);
    }
  } catch (error) {
    core.info(`  Could not fetch commits for source file: ${error instanceof Error ? error.message : String(error)}`);
  }

  // Get the latest commit for the lock file
  let lockCommitDate = null;
  try {
    const lockCommits = await github.rest.repos.listCommits({
      owner: owner,
      repo: repo,
      path: lockPath,
      per_page: 1,
    });

    if (lockCommits.data.length > 0) {
      const latestCommit = lockCommits.data[0];
      lockCommitDate = new Date(latestCommit.commit.committer.date);
      core.info(`  Lock file last commit: ${lockCommitDate.toISOString()} (${latestCommit.sha.substring(0, 7)})`);
    } else {
      core.info(`  Lock file not found in repository: ${lockPath}`);
    }
  } catch (error) {
    core.info(`  Could not fetch commits for lock file: ${error instanceof Error ? error.message : String(error)}`);
  }

  // Skip check if either file is not found
  if (!workflowCommitDate || !lockCommitDate) {
    core.info("Skipping timestamp check - one or both files not found in repository");
    return;
  }

  // Check if workflow file commit is newer than lock file commit
  if (workflowCommitDate > lockCommitDate) {
    const warningMessage = `🔴🔴🔴 WARNING: Lock file '${lockPath}' is outdated! The workflow file '${workflowPath}' has been modified more recently. Run 'gh aw compile' to regenerate the lock file.`;

    core.error(warningMessage);

    // Add summary to GitHub Step Summary
    await core.summary
      .addRaw("## ⚠️ Workflow Lock File Warning\n\n")
      .addRaw(`🔴🔴🔴 **WARNING**: Lock file \`${lockPath}\` is outdated!\n\n`)
      .addRaw(`The workflow file \`${workflowPath}\` has been modified more recently.\n\n`)
      .addRaw("Run `gh aw compile` to regenerate the lock file.\n\n")
      .write();
  } else {
    core.info("✅ Lock file is up to date");
  }
}

main().catch(error => {
  core.setFailed(error instanceof Error ? error.message : String(error));
});
