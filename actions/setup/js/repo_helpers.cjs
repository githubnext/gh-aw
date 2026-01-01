// @ts-check
/// <reference types="@actions/github-script" />

/**
 * Repository-related helper functions for safe-output scripts
 * Provides common repository parsing, validation, and resolution logic
 */

/**
 * Parse the allowed repos from config value (array or comma-separated string)
 * @param {string[]|string|undefined} allowedReposValue - Allowed repos from config (array or comma-separated string)
 * @returns {Set<string>} Set of allowed repository slugs
 */
function parseAllowedRepos(allowedReposValue) {
  const set = new Set();
  if (Array.isArray(allowedReposValue)) {
    allowedReposValue
      .map(repo => repo.trim())
      .filter(repo => repo)
      .forEach(repo => set.add(repo));
  } else if (typeof allowedReposValue === "string") {
    allowedReposValue
      .split(",")
      .map(repo => repo.trim())
      .filter(repo => repo)
      .forEach(repo => set.add(repo));
  }
  return set;
}

/**
 * Get the default target repository
 * @param {Object} [config] - Optional config object with target-repo field
 * @returns {string} Repository slug in "owner/repo" format
 */
function getDefaultTargetRepo(config) {
  // First check if there's a target-repo in config
  if (config && config["target-repo"]) {
    return config["target-repo"];
  }
  // Fall back to env var for backward compatibility
  const targetRepoSlug = process.env.GH_AW_TARGET_REPO_SLUG;
  if (targetRepoSlug) {
    return targetRepoSlug;
  }
  // Fall back to context repo
  return `${context.repo.owner}/${context.repo.repo}`;
}

/**
 * Validate that a repo is allowed for operations
 * @param {string} repo - Repository slug to validate
 * @param {string} defaultRepo - Default target repository
 * @param {Set<string>} allowedRepos - Set of explicitly allowed repos
 * @returns {{valid: boolean, error: string|null}}
 */
function validateRepo(repo, defaultRepo, allowedRepos) {
  // Default repo is always allowed
  if (repo === defaultRepo) {
    return { valid: true, error: null };
  }
  // Check if it's in the allowed repos list
  if (allowedRepos.has(repo)) {
    return { valid: true, error: null };
  }
  return {
    valid: false,
    error: `Repository '${repo}' is not in the allowed-repos list. Allowed: ${defaultRepo}${allowedRepos.size > 0 ? ", " + Array.from(allowedRepos).join(", ") : ""}`,
  };
}

/**
 * Parse owner and repo from a repository slug
 * @param {string} repoSlug - Repository slug in "owner/repo" format
 * @returns {{owner: string, repo: string}|null}
 */
function parseRepoSlug(repoSlug) {
  const parts = repoSlug.split("/");
  if (parts.length !== 2 || !parts[0] || !parts[1]) {
    return null;
  }
  return { owner: parts[0], repo: parts[1] };
}

module.exports = {
  parseAllowedRepos,
  getDefaultTargetRepo,
  validateRepo,
  parseRepoSlug,
};
