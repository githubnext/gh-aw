// @ts-check
/// <reference types="@actions/github-script" />

/**
 * Check workflow file timestamps to detect outdated lock files
 * This script uses GitHub API to compare the last commit time of the source .md file
 * with the compiled .lock.yml file and warns if recompilation is needed
 */

async function main() {
  const workflowFile = process.env.GH_AW_WORKFLOW_FILE;

  if (!workflowFile) {
    core.setFailed("Configuration error: GH_AW_WORKFLOW_FILE not available.");
    return;
  }

  // Extract repository information from context
  const owner = context.repo.owner;
  const repo = context.repo.repo;
  const ref = context.sha || context.ref;

  // Construct file paths
  const workflowBasename = workflowFile.replace(/\.lock\.yml$/, "");
  const workflowMdPath = `.github/workflows/${workflowBasename}.md`;
  const lockPath = `.github/workflows/${workflowFile}`;

  core.info(`Checking workflow timestamps using GitHub API:`);
  core.info(`  Repository: ${owner}/${repo}`);
  core.info(`  Source: ${workflowMdPath}`);
  core.info(`  Lock file: ${lockPath}`);
  core.info(`  Ref: ${ref}`);

  // Fetch file information from GitHub API
  let workflowData;
  let lockData;

  try {
    const workflowResponse = await github.rest.repos.getContent({
      owner,
      repo,
      path: workflowMdPath,
      ref,
    });
    workflowData = workflowResponse.data;
  } catch (error) {
    if (error.status === 404) {
      core.info(`Source file does not exist: ${workflowMdPath}`);
    } else {
      const errorMsg = error instanceof Error ? error.message : String(error);
      core.warning(`Failed to fetch source file: ${errorMsg}`);
    }
  }

  try {
    const lockResponse = await github.rest.repos.getContent({
      owner,
      repo,
      path: lockPath,
      ref,
    });
    lockData = lockResponse.data;
  } catch (error) {
    if (error.status === 404) {
      core.info(`Lock file does not exist: ${lockPath}`);
    } else {
      const errorMsg = error instanceof Error ? error.message : String(error);
      core.warning(`Failed to fetch lock file: ${errorMsg}`);
    }
  }

  if (!workflowData || !lockData) {
    core.info("Skipping timestamp check - one or both files not found");
    return;
  }

  // Get the last commit for each file to compare timestamps
  let workflowCommit;
  let lockCommit;

  try {
    const workflowCommitsResponse = await github.rest.repos.listCommits({
      owner,
      repo,
      path: workflowMdPath,
      sha: ref,
      per_page: 1,
    });
    if (workflowCommitsResponse.data && workflowCommitsResponse.data.length > 0) {
      workflowCommit = workflowCommitsResponse.data[0];
    }
  } catch (error) {
    const errorMsg = error instanceof Error ? error.message : String(error);
    core.warning(`Failed to fetch commits for source file: ${errorMsg}`);
  }

  try {
    const lockCommitsResponse = await github.rest.repos.listCommits({
      owner,
      repo,
      path: lockPath,
      sha: ref,
      per_page: 1,
    });
    if (lockCommitsResponse.data && lockCommitsResponse.data.length > 0) {
      lockCommit = lockCommitsResponse.data[0];
    }
  } catch (error) {
    const errorMsg = error instanceof Error ? error.message : String(error);
    core.warning(`Failed to fetch commits for lock file: ${errorMsg}`);
  }

  if (!workflowCommit || !lockCommit) {
    core.info("Skipping timestamp check - could not fetch commit information");
    return;
  }

  // Compare commit dates
  const workflowDate = new Date(workflowCommit.commit.committer.date);
  const lockDate = new Date(lockCommit.commit.committer.date);

  core.info(`  Source last modified: ${workflowDate.toISOString()}`);
  core.info(`  Lock file last modified: ${lockDate.toISOString()}`);

  // Check if workflow file commit is newer than lock file commit
  if (workflowDate > lockDate) {
    const warningMessage = `WARNING: Lock file '${lockPath}' is outdated! The workflow file '${workflowMdPath}' has been modified more recently. Run 'gh aw compile' to regenerate the lock file.`;

    core.error(warningMessage);

    // Format timestamps for display
    const workflowTimestamp = workflowDate.toISOString();
    const lockTimestamp = lockDate.toISOString();

    // Get git commit SHA if available
    const gitSha = context.sha;

    // Add summary to GitHub Step Summary
    let summary = core.summary
      .addRaw("### ⚠️ Workflow Lock File Warning\n\n")
      .addRaw("**WARNING**: Lock file is outdated and needs to be regenerated.\n\n")
      .addRaw("**Files:**\n")
      .addRaw(`- Source: \`${workflowMdPath}\` (last modified: ${workflowTimestamp})\n`)
      .addRaw(`- Lock: \`${lockPath}\` (last modified: ${lockTimestamp})\n\n`);

    if (gitSha) {
      summary = summary.addRaw(`**Git Commit:** \`${gitSha}\`\n\n`);
    }

    summary = summary.addRaw("**Action Required:** Run `gh aw compile` to regenerate the lock file.\n\n");

    await summary.write();
  } else {
    core.info("✅ Lock file is up to date");
  }
}

await main();
