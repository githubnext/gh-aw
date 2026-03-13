// @ts-check
/// <reference types="@actions/github-script" />

/**
 * Builds the GitHub search query, optionally scoping it to the current repository.
 * @param {string} skipQuery - The base query string
 * @param {string|undefined} skipScope - The scope setting ('none' to disable repo scoping)
 * @returns {string} The final search query
 */
function buildSearchQuery(skipQuery, skipScope) {
  if (skipScope === "none") {
    core.info(`Using raw query (scope: none): ${skipQuery}`);
    return skipQuery;
  }
  const { owner, repo } = context.repo;
  const searchQuery = `${skipQuery} repo:${owner}/${repo}`;
  core.info(`Scoped query: ${searchQuery}`);
  return searchQuery;
}

module.exports = { buildSearchQuery };
