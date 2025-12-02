// @ts-check
/// <reference types="@actions/github-script" />

/**
 * Helper to get an Octokit instance with a custom token
 * Uses @actions/github's getOctokit function via __original_require__
 * to properly instantiate a new Octokit client with different authentication.
 *
 * See: https://github.com/actions/github-script/blob/main/src/async-function.ts
 */

/**
 * Factory function for getOctokit - can be overridden for testing
 * @type {Function|null}
 */
let getOctokitFactory = null;

/**
 * Set the getOctokit factory function for testing purposes
 * @param {Function} factory - Factory function that returns an Octokit instance
 */
function setGetOctokitFactory(factory) {
  getOctokitFactory = factory;
}

/**
 * Get an Octokit client with a custom token
 * @param {string} token - GitHub token for authentication
 * @returns {Object} Octokit instance with graphql method
 */
function getOctokitClient(token) {
  if (getOctokitFactory) {
    return getOctokitFactory(token);
  }

  // Use __original_require__ to import @actions/github
  // This is necessary because github-script wraps require
  // eslint-disable-next-line no-undef
  const { getOctokit } = __original_require__("@actions/github");
  return getOctokit(token);
}

module.exports = {
  getOctokitClient,
  setGetOctokitFactory,
};
