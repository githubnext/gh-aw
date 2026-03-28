// @ts-check
/// <reference types="@actions/github-script" />

/**
 * GitHub API helper functions
 * Provides common GitHub API operations with consistent error handling
 */

const { getErrorMessage } = require("./error_helpers.cjs");

/**
 * @typedef {Object} GraphQLErrorHints
 * @property {string} [insufficientScopesHint] - Message shown when INSUFFICIENT_SCOPES error type is present
 * @property {string} [notFoundHint] - Message shown when NOT_FOUND error type is present
 * @property {function(string): boolean} [notFoundPredicate] - Additional condition for showing the NOT_FOUND hint (receives the error message string)
 */

/**
 * Log detailed GraphQL error information for diagnosing API failures.
 * Surfaces the errors array, type codes, paths, HTTP status, and optional domain-specific hints.
 *
 * @param {Error & { errors?: Array<{ type?: string, message: string, path?: unknown, locations?: unknown }>, request?: unknown, data?: unknown, status?: number }} error - GraphQL error
 * @param {string} operation - Human-readable description of the failing operation
 * @param {GraphQLErrorHints} [hints] - Optional domain-specific hint messages
 */
function logGraphQLError(error, operation, hints = {}) {
  core.info(`GraphQL error during: ${operation}`);
  core.info(`Message: ${getErrorMessage(error)}`);

  const errorList = Array.isArray(error.errors) ? error.errors : [];
  const hasInsufficientScopes = errorList.some(e => e?.type === "INSUFFICIENT_SCOPES");
  const hasNotFound = errorList.some(e => e?.type === "NOT_FOUND");

  if (hasInsufficientScopes && hints.insufficientScopesHint) {
    core.info(hints.insufficientScopesHint);
  }

  if (hasNotFound && hints.notFoundHint) {
    const predicatePasses = !hints.notFoundPredicate || hints.notFoundPredicate(getErrorMessage(error));
    if (predicatePasses) {
      core.info(hints.notFoundHint);
    }
  }

  if (error.errors) {
    core.info(`Errors array (${error.errors.length} error(s)):`);
    error.errors.forEach((err, idx) => {
      core.info(`  [${idx + 1}] ${err.message}`);
      if (err.type) core.info(`      Type: ${err.type}`);
      if (err.path) core.info(`      Path: ${JSON.stringify(err.path)}`);
      if (err.locations) core.info(`      Locations: ${JSON.stringify(err.locations)}`);
    });
  }

  if (error.status) core.info(`HTTP status: ${error.status}`);
  if (error.request) core.info(`Request: ${JSON.stringify(error.request, null, 2)}`);
  if (error.data) core.info(`Response data: ${JSON.stringify(error.data, null, 2)}`);
}

/**
 * Get file content from GitHub repository using the API
 * @param {Object} github - GitHub API client (@actions/github)
 * @param {string} owner - Repository owner
 * @param {string} repo - Repository name
 * @param {string} path - File path within the repository
 * @param {string} ref - Git reference (branch, tag, or commit SHA)
 * @returns {Promise<string|null>} File content as string, or null if not found/error
 */
async function getFileContent(github, owner, repo, path, ref) {
  try {
    const response = await github.rest.repos.getContent({
      owner,
      repo,
      path,
      ref,
    });

    // Handle case where response is an array (directory listing)
    if (Array.isArray(response.data)) {
      core.info(`Path ${path} is a directory, not a file`);
      return null;
    }

    // Check if this is a file (not a symlink or submodule)
    if (response.data.type !== "file") {
      core.info(`Path ${path} is not a file (type: ${response.data.type})`);
      return null;
    }

    // Decode base64 content
    if (response.data.encoding === "base64" && response.data.content) {
      return Buffer.from(response.data.content, "base64").toString("utf8");
    }

    return response.data.content || null;
  } catch (error) {
    const errorMessage = getErrorMessage(error);
    core.info(`Could not fetch content for ${path}: ${errorMessage}`);
    return null;
  }
}

module.exports = {
  getFileContent,
  logGraphQLError,
};
