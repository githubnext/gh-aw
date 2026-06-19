// @ts-check
/// <reference types="@actions/github-script" />

const { validateTargetRepo, parseAllowedRepos, getDefaultTargetRepo } = require("./repo_helpers.cjs");
const { ERR_VALIDATION } = require("./error_codes.cjs");

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
 * Determine whether the current checkout already has a persisted
 * http.<serverUrl>/.extraheader credential in .git/config.
 *
 * When actions/checkout runs with persist-credentials: true (as the safe_outputs job
 * does), it writes an http.<serverUrl>/.extraheader entry that authenticates every
 * <serverUrl> URL. In that case a dynamic repo switch must NOT inject a second
 * extraheader, because git treats the key as multi-valued and would send two
 * Authorization headers, which GitHub rejects with "Duplicate header: 'Authorization'".
 *
 * @param {string} serverUrl - GitHub server URL (e.g. "https://github.com")
 * @returns {Promise<boolean>} true if a non-empty extraheader is already configured
 */
async function checkoutHasPersistedExtraheader(serverUrl) {
  try {
    const result = await exec.getExecOutput("git", ["config", "--get-all", `http.${serverUrl}/.extraheader`], {
      silent: true,
      ignoreReturnCode: true,
    });
    return result.exitCode === 0 && result.stdout.trim() !== "";
  } catch {
    return false;
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
 * @param {string[]|string} [options.allowedRepos] - Allowed repository patterns for allowlist validation
 * @returns {Promise<Object>} Result with success status
 */
async function checkoutRepo(repoSlug, token, options = {}) {
  const baseBranch = options.baseBranch || "main";
  const parts = (repoSlug || "").trim().split("/");

  if (parts.length !== 2 || !parts[0] || !parts[1]) {
    return {
      success: false,
      error: `${ERR_VALIDATION}: Invalid repository slug: ${repoSlug}. Expected format: owner/repo`,
    };
  }

  // Validate target repo against configured allowlist before any git operations
  const allowedRepos = parseAllowedRepos(options.allowedRepos);
  if (allowedRepos.size > 0) {
    const defaultRepo = getDefaultTargetRepo();
    const validation = validateTargetRepo(repoSlug, defaultRepo, allowedRepos);
    if (!validation.valid) {
      return { success: false, error: `${ERR_VALIDATION}: ${validation.error}` };
    }
  }

  const [owner, repo] = parts;

  core.info(`Switching checkout to repository: ${repoSlug}`);

  try {
    // Get GitHub server URL (for GHES support)
    const serverUrl = process.env.GITHUB_SERVER_URL || "https://github.com";

    // Configure the new remote URL (no embedded credentials). Authentication is
    // provided by the http.<serverUrl>/.extraheader credential that the safe_outputs
    // checkout persisted into .git/config (persist-credentials: true with the resolved
    // push token). That extraheader applies to every <serverUrl> URL, so it already
    // authenticates the repository we are switching to.
    const remoteUrl = `${serverUrl}/${repoSlug}.git`;

    // Change remote origin to the new repo. Replacing the URL also drops any
    // embedded-credential URL configure_git_credentials.sh may have set, leaving the
    // persisted extraheader as the single source of auth.
    core.info(`Configuring remote origin to: ${repoSlug}`);
    await exec.exec("git", ["remote", "set-url", "origin", remoteUrl]);

    // Trust the persisted credential rather than injecting a second one. git treats
    // http.<url>.extraheader as multi-valued, so adding our own header on top of the
    // checkout's persisted header would put two Authorization headers on the wire,
    // which GitHub rejects with "Duplicate header: 'Authorization'" (HTTP 400). Only
    // configure an extraheader when the checkout did not already persist one (e.g. a
    // caller that runs without persist-credentials).
    const hasPersistedAuth = await checkoutHasPersistedExtraheader(serverUrl);
    if (!hasPersistedAuth) {
      // Use extraheader to pass the token without embedding it in the URL (more secure).
      const tokenBase64 = Buffer.from(`x-access-token:${token}`).toString("base64");
      await exec.exec("git", ["config", `http.${serverUrl}/.extraheader`, `Authorization: basic ${tokenBase64}`]);
    } else {
      core.info("Reusing persisted git credential for authentication (skipping extraheader injection)");
    }

    // Fetch the new repo
    core.info(`Fetching repository: ${repoSlug}`);
    await exec.exec("git", ["fetch", "origin", "--prune"]);

    // Reset to the base branch of the new repo
    core.info(`Checking out base branch: ${baseBranch}`);
    try {
      await exec.exec("git", ["checkout", "-B", baseBranch, `origin/${baseBranch}`]);
    } catch {
      throw new Error(`Base branch ${baseBranch} not found in ${repoSlug}`);
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
