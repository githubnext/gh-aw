// @ts-check
/// <reference types="@actions/github-script" />

/**
 * @fileoverview Extra Empty Commit Helper
 *
 * Pushes an empty commit to a branch using a different token to trigger CI events.
 * This works around the GitHub Actions limitation where events created with
 * GITHUB_TOKEN do not trigger other workflow runs.
 *
 * The token comes from `github-token-for-extra-empty-commit` in safe-outputs config
 * and is passed in as the GH_AW_CI_TRIGGER_TOKEN environment variable.
 * By the time this script runs, GH_AW_CI_TRIGGER_TOKEN must contain an actual
 * GitHub authentication token (for example, a GitHub App token or a PAT).
 * Any selection or defaulting behavior (such as resolving `app`, `default`,
 * or a specific secret reference) is handled in the workflow compiler/config
 * layer before this script is invoked.
 */

/**
 * Push an empty commit to a branch using a dedicated token.
 * This commit is pushed with different authentication so that push/PR events
 * are triggered for CI checks to run.
 *
 * @param {Object} options - Options for the extra empty commit
 * @param {string} options.branchName - The branch to push the empty commit to
 * @param {string} options.repoOwner - Repository owner
 * @param {string} options.repoName - Repository name
 * @param {string} [options.commitMessage] - Custom commit message (default: "ci: trigger CI checks")
 * @param {number} [options.newCommitCount] - Number of new commits being pushed. Only pushes the
 *   empty commit when exactly 1 new commit was pushed, preventing accidental workflow-file
 *   modifications on multi-commit branches and reducing loop risk.
 * @returns {Promise<{success: boolean, skipped?: boolean, error?: string}>}
 */
async function pushExtraEmptyCommit({ branchName, repoOwner, repoName, commitMessage, newCommitCount }) {
  const token = process.env.GH_AW_CI_TRIGGER_TOKEN;

  if (!token || !token.trim()) {
    core.info("No extra empty commit token configured - skipping");
    return { success: true, skipped: true };
  }

  if (newCommitCount !== undefined && newCommitCount !== 1) {
    core.info(`Skipping extra empty commit: ${newCommitCount} new commit(s) pushed (only triggers for exactly 1 commit)`);
    return { success: true, skipped: true };
  }

  core.info("Extra empty commit token detected - pushing empty commit to trigger CI events");

  try {
    // Cycle prevention: count empty commits in the last 60 commits on this branch.
    // If 30 or more are empty, skip pushing to avoid infinite trigger loops.
    const MAX_EMPTY_COMMITS = 30;
    const COMMITS_TO_CHECK = 60;
    let emptyCommitCount = 0;

    try {
      let logOutput = "";
      // List last N commits: for each, output "COMMIT:<hash>" then changed file names.
      // Empty commits will have no files listed after the hash line.
      await exec.exec("git", ["log", `--max-count=${COMMITS_TO_CHECK}`, "--format=COMMIT:%H", "--name-only", "HEAD"], {
        listeners: {
          stdout: data => {
            logOutput += data.toString();
          },
        },
        silent: true,
      });
      // Split by COMMIT: markers; each chunk starts with the hash, followed by filenames
      const chunks = logOutput.split("COMMIT:").filter(c => c.trim());
      for (const chunk of chunks) {
        const lines = chunk.split("\n").filter(l => l.trim());
        // First line is the hash, remaining lines are changed files
        if (lines.length <= 1) {
          emptyCommitCount++;
        }
      }
    } catch {
      // If we can't check, default to allowing the push
      emptyCommitCount = 0;
    }

    if (emptyCommitCount >= MAX_EMPTY_COMMITS) {
      core.warning(`Cycle prevention: found ${emptyCommitCount} empty commits in the last ${COMMITS_TO_CHECK} commits on ${branchName}. ` + `Skipping extra empty commit to avoid potential infinite loop.`);
      return { success: true, skipped: true };
    }

    core.info(`Cycle check passed: ${emptyCommitCount} empty commit(s) in last ${COMMITS_TO_CHECK} (limit: ${MAX_EMPTY_COMMITS})`);

    // Configure git remote with the token for authentication
    const remoteUrl = `https://x-access-token:${token}@github.com/${repoOwner}/${repoName}.git`;

    // Add a temporary remote with the token
    try {
      await exec.exec("git", ["remote", "remove", "ci-trigger"]);
    } catch {
      // Remote doesn't exist yet, that's fine
    }
    await exec.exec("git", ["remote", "add", "ci-trigger", remoteUrl]);

    // Create and push an empty commit
    const message = commitMessage || "ci: trigger CI checks";
    await exec.exec("git", ["commit", "--allow-empty", "-m", message]);
    await exec.exec("git", ["push", "ci-trigger", branchName]);

    core.info(`Extra empty commit pushed to ${branchName} successfully`);

    // Clean up the temporary remote
    try {
      await exec.exec("git", ["remote", "remove", "ci-trigger"]);
    } catch {
      // Non-fatal cleanup error
    }

    return { success: true };
  } catch (error) {
    const errorMessage = error instanceof Error ? error.message : String(error);
    core.warning(`Failed to push extra empty commit: ${errorMessage}`);

    // Clean up the temporary remote on failure
    try {
      await exec.exec("git", ["remote", "remove", "ci-trigger"]);
    } catch {
      // Non-fatal cleanup error
    }

    // Extra empty commit failure is not fatal - the main push already succeeded
    return { success: false, error: errorMessage };
  }
}

module.exports = { pushExtraEmptyCommit };
