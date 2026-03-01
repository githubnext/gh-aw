// @ts-check
/// <reference types="@actions/github-script" />

/**
 * Dynamic repository checkout utilities for multi-repo scenarios
 * Enables switching between different repositories during handler execution
 */

/**
 * Get the currently checked out repository slug from git remote
 * @returns {Promise<string|null>} The repo slug (owner/repo) or null if not determinable
 */
async function getCurrentCheckoutRepo() {
  try {
    const result = await exec.getExecOutput("git", ["config", "--get", "remote.origin.url"], { silent: true });
    const url = result.stdout.trim();

    // Extract repo slug from URL
    // Handle HTTPS and SSH formats
    let slug = null;

    // Remove .git suffix if present
    let cleanUrl = url;
    if (cleanUrl.endsWith(".git")) {
      cleanUrl = cleanUrl.slice(0, -4);
    }

    // HTTPS: https://github.com/owner/repo
    const httpsMatch = cleanUrl.match(/https?:\/\/[^/]+\/([^/]+\/[^/]+)$/);
    if (httpsMatch) {
      slug = httpsMatch[1].toLowerCase();
    }

    // SSH: git@github.com:owner/repo
    const sshMatch = cleanUrl.match(/git@[^:]+:([^/]+\/[^/]+)$/);
    if (sshMatch) {
      slug = sshMatch[1].toLowerCase();
    }

    return slug;
  } catch {
    return null;
  }
}

/**
 * Checkout a different repository for patch application
 * This is used when processing entries with a `repo` parameter that differs from the current checkout
 *
 * @param {string} repoSlug - Repository slug (owner/repo) to checkout
 * @param {string} token - GitHub token for authentication
 * @param {Object} options - Additional options
 * @param {string} [options.baseBranch] - Base branch to checkout (defaults to 'main')
 * @returns {Promise<Object>} Result with success status
 */
async function checkoutRepo(repoSlug, token, options = {}) {
  const baseBranch = options.baseBranch || "main";
  const parts = (repoSlug || "").trim().split("/");

  if (parts.length !== 2 || !parts[0] || !parts[1]) {
    return {
      success: false,
      error: `Invalid repository slug: ${repoSlug}. Expected format: owner/repo`,
    };
  }

  const [owner, repo] = parts;

  core.info(`Switching checkout to repository: ${repoSlug}`);

  try {
    // Get GitHub server URL (for GHES support)
    const serverUrl = process.env.GITHUB_SERVER_URL || "https://github.com";

    // Configure the new remote URL
    // Use token in URL for authentication since we're not using credential helper
    const remoteUrl = `${serverUrl}/${repoSlug}.git`;

    // Change remote origin to the new repo
    core.info(`Configuring remote origin to: ${repoSlug}`);
    await exec.exec("git", ["remote", "set-url", "origin", remoteUrl]);

    // Configure token for authentication
    // Use extraheader to pass token without embedding in URL (more secure)
    const tokenBase64 = Buffer.from(`x-access-token:${token}`).toString("base64");
    await exec.exec("git", ["config", `http.${serverUrl}/.extraheader`, `Authorization: basic ${tokenBase64}`]);

    // Fetch the new repo
    core.info(`Fetching repository: ${repoSlug}`);
    await exec.exec("git", ["fetch", "origin", "--prune"]);

    // Reset to the base branch of the new repo
    core.info(`Checking out base branch: ${baseBranch}`);
    try {
      await exec.exec("git", ["checkout", "-B", baseBranch, `origin/${baseBranch}`]);
    } catch {
      // Try 'master' if 'main' doesn't exist
      if (baseBranch === "main") {
        core.info("main branch not found, trying master");
        await exec.exec("git", ["checkout", "-B", "master", "origin/master"]);
      } else {
        throw new Error(`Base branch ${baseBranch} not found in ${repoSlug}`);
      }
    }

    // Clean up any local changes
    await exec.exec("git", ["clean", "-fd"]);
    await exec.exec("git", ["reset", "--hard", "HEAD"]);

    core.info(`Successfully switched to repository: ${repoSlug}`);

    return {
      success: true,
      repoSlug: repoSlug,
    };
  } catch (error) {
    const errorMsg = error instanceof Error ? error.message : String(error);
    core.error(`Failed to checkout repository ${repoSlug}: ${errorMsg}`);
    return {
      success: false,
      error: `Failed to checkout repository ${repoSlug}: ${errorMsg}`,
    };
  }
}

/**
 * Track and manage the currently active checkout
 * Returns a manager object that handles repo switching
 *
 * @param {string} token - GitHub token for authentication
 * @param {Object} options - Options
 * @param {string} [options.defaultBaseBranch] - Default base branch
 * @returns {Object} Checkout manager with switchTo method
 */
function createCheckoutManager(token, options = {}) {
  let currentRepo = null;
  const defaultBaseBranch = options.defaultBaseBranch || "main";

  return {
    /**
     * Get the currently checked out repo
     * @returns {Promise<string|null>}
     */
    async getCurrent() {
      if (!currentRepo) {
        currentRepo = await getCurrentCheckoutRepo();
      }
      return currentRepo;
    },

    /**
     * Switch to a different repository if needed
     * @param {string} targetRepo - Target repo slug
     * @param {Object} [opts] - Options
     * @param {string} [opts.baseBranch] - Base branch to use
     * @returns {Promise<Object>} Result with success status
     */
    async switchTo(targetRepo, opts = {}) {
      const baseBranch = opts.baseBranch || defaultBaseBranch;
      const targetLower = targetRepo.toLowerCase();

      // Get current checkout if not known
      if (!currentRepo) {
        currentRepo = await getCurrentCheckoutRepo();
      }

      // Check if we're already on the right repo
      if (currentRepo === targetLower) {
        core.info(`Already on repository: ${targetRepo}`);
        return { success: true, switched: false };
      }

      // Need to switch
      core.info(`Switching from ${currentRepo || "unknown"} to ${targetRepo}`);
      const result = await checkoutRepo(targetRepo, token, { baseBranch });

      if (result.success) {
        currentRepo = targetLower;
        return { success: true, switched: true };
      }

      return result;
    },
  };
}

module.exports = {
  getCurrentCheckoutRepo,
  checkoutRepo,
  createCheckoutManager,
};
