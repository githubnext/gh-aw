// @ts-check
/// <reference types="@actions/github-script" />

const { execSync } = require("child_process");

/**
 * Get the current git branch name
 * @returns {string} The current branch name
 */
function getCurrentBranch() {
  // Priority 1: Try git command first to get the actual checked-out branch
  // This is more reliable than environment variables which may not reflect
  // branch changes made during the workflow execution
  const cwd = process.env.GITHUB_WORKSPACE || process.cwd();
  try {
    const branch = execSync("git rev-parse --abbrev-ref HEAD", {
      encoding: "utf8",
      cwd: cwd,
    }).trim();
    return branch;
  } catch (error) {
    // Ignore error and try fallback
  }

  // Priority 2: Fallback to GitHub Actions environment variables
  // GITHUB_HEAD_REF is set for pull_request events and contains the source branch name
  // GITHUB_REF_NAME is set for all events and contains the branch/tag name
  const ghHeadRef = process.env.GITHUB_HEAD_REF;
  const ghRefName = process.env.GITHUB_REF_NAME;

  if (ghHeadRef) {
    return ghHeadRef;
  }

  if (ghRefName) {
    return ghRefName;
  }

  throw new Error("Failed to determine current branch: git command failed and no GitHub environment variables available");
}

module.exports = {
  getCurrentBranch,
};
