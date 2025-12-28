// @ts-check
/// <reference types="@actions/github-script" />

/**
 * Get the base branch name from environment variable
 * @returns {string} The base branch name (defaults to "main")
 */
function getBaseBranch() {
  return process.env.GH_AW_BASE_BRANCH || "main";
}

module.exports = {
  getBaseBranch,
};
