// @ts-check
/// <reference types="@actions/github-script" />

const { parseAllowedRepos, validateRepo } = require("./repo_helpers.cjs");

/**
 * Get the repository URL for different purposes
 * This helper handles trial mode where target repository URLs are different from execution context
 * @param {Object} [config] - Optional config object with target-repo field
 * @returns {string} Repository URL
 */
function getRepositoryUrl(config) {
  // For trial mode, use target repository for issue/PR URLs but execution context for action runs
  let targetRepoSlug;

  // First check if there's a target-repo in config
  if (config && config["target-repo"]) {
    targetRepoSlug = config["target-repo"];

    // Validate target repository against allowlist
    const allowedRepos = parseAllowedRepos(config["allowed-repos"] || config.allowed_repos);
    const defaultRepo = `${context.repo.owner}/${context.repo.repo}`;

    const repoValidation = validateRepo(targetRepoSlug, defaultRepo, allowedRepos);
    if (!repoValidation.valid) {
      throw new Error(`E004: ${repoValidation.error}`);
    }
  } else {
    // Fall back to env var for backward compatibility
    targetRepoSlug = process.env.GH_AW_TARGET_REPO_SLUG;

    if (targetRepoSlug) {
      // Validate env var target repository against allowlist
      const allowedReposEnv = process.env.GH_AW_TARGET_REPO_ALLOWED_REPOS?.trim();
      const allowedRepos = parseAllowedRepos(allowedReposEnv);
      const defaultRepo = `${context.repo.owner}/${context.repo.repo}`;

      const repoValidation = validateRepo(targetRepoSlug, defaultRepo, allowedRepos);
      if (!repoValidation.valid) {
        throw new Error(`E004: ${repoValidation.error}`);
      }
    }
  }

  if (targetRepoSlug) {
    // Use target repository for issue/PR URLs in trial mode
    const githubServer = process.env.GITHUB_SERVER_URL || "https://github.com";
    return `${githubServer}/${targetRepoSlug}`;
  } else if (context.payload.repository?.html_url) {
    // Use execution context repository (default behavior)
    return context.payload.repository.html_url;
  } else {
    // Final fallback for action runs when context repo is not available
    const githubServer = process.env.GITHUB_SERVER_URL || "https://github.com";
    return `${githubServer}/${context.repo.owner}/${context.repo.repo}`;
  }
}

module.exports = {
  getRepositoryUrl,
};
