// @ts-check
/// <reference types="@actions/github-script" />

const fs = require("fs");
const path = require("path");
const { execSync } = require("child_process");

const { getBaseBranch } = require("./get_base_branch.cjs");
const { getErrorMessage } = require("./error_helpers.cjs");

/**
 * Generates a git patch file for the current changes
 * @param {string} branchName - The branch name to generate patch for
 * @returns {Object} Object with patch info or error
 */
function generateGitPatch(branchName) {
  const patchPath = "/tmp/gh-aw/aw.patch";
  const cwd = process.env.GITHUB_WORKSPACE || process.cwd();
  const defaultBranch = process.env.DEFAULT_BRANCH || getBaseBranch();
  const githubSha = process.env.GITHUB_SHA;

  // Ensure /tmp/gh-aw directory exists
  const patchDir = path.dirname(patchPath);
  if (!fs.existsSync(patchDir)) {
    fs.mkdirSync(patchDir, { recursive: true });
  }

  let patchGenerated = false;
  let errorMessage = null;

  try {
    // Strategy 1: If we have a branch name, check if that branch exists and get its diff
    if (branchName) {
      // Check if the branch exists locally
      try {
        execSync(`git show-ref --verify --quiet refs/heads/${branchName}`, { cwd, encoding: "utf8" });

        // Determine base ref for patch generation
        let baseRef;
        try {
          // Check if origin/branchName exists
          execSync(`git show-ref --verify --quiet refs/remotes/origin/${branchName}`, { cwd, encoding: "utf8" });
          baseRef = `origin/${branchName}`;
        } catch {
          // Use merge-base with default branch
          execSync(`git fetch origin ${defaultBranch}`, { cwd, encoding: "utf8" });
          baseRef = execSync(`git merge-base origin/${defaultBranch} ${branchName}`, { cwd, encoding: "utf8" }).trim();
        }

        // Count commits to be included
        const commitCount = parseInt(execSync(`git rev-list --count ${baseRef}..${branchName}`, { cwd, encoding: "utf8" }).trim(), 10);

        if (commitCount > 0) {
          // Generate patch from the determined base to the branch
          const patchContent = execSync(`git format-patch ${baseRef}..${branchName} --stdout`, {
            cwd,
            encoding: "utf8",
          });

          if (patchContent && patchContent.trim()) {
            fs.writeFileSync(patchPath, patchContent, "utf8");
            patchGenerated = true;
          }
        }
      } catch (branchError) {
        // Branch does not exist locally
      }
    }

    // Strategy 2: Check if commits were made to current HEAD since checkout
    if (!patchGenerated) {
      const currentHead = execSync("git rev-parse HEAD", { cwd, encoding: "utf8" }).trim();

      if (!githubSha) {
        errorMessage = "GITHUB_SHA environment variable is not set";
      } else if (currentHead === githubSha) {
        // No commits have been made since checkout
      } else {
        // Check if GITHUB_SHA is an ancestor of current HEAD
        try {
          execSync(`git merge-base --is-ancestor ${githubSha} HEAD`, { cwd, encoding: "utf8" });

          // Count commits between GITHUB_SHA and HEAD
          const commitCount = parseInt(execSync(`git rev-list --count ${githubSha}..HEAD`, { cwd, encoding: "utf8" }).trim(), 10);

          if (commitCount > 0) {
            // Generate patch from GITHUB_SHA to HEAD
            const patchContent = execSync(`git format-patch ${githubSha}..HEAD --stdout`, {
              cwd,
              encoding: "utf8",
            });

            if (patchContent && patchContent.trim()) {
              fs.writeFileSync(patchPath, patchContent, "utf8");
              patchGenerated = true;
            }
          }
        } catch {
          // GITHUB_SHA is not an ancestor of HEAD - repository state has diverged
        }
      }
    }
  } catch (error) {
    errorMessage = `Failed to generate patch: ${getErrorMessage(error)}`;
  }

  // Check if patch was generated and has content
  if (patchGenerated && fs.existsSync(patchPath)) {
    const patchContent = fs.readFileSync(patchPath, "utf8");
    const patchSize = Buffer.byteLength(patchContent, "utf8");
    const patchLines = patchContent.split("\n").length;

    if (!patchContent.trim()) {
      // Empty patch
      return {
        success: false,
        error: "No changes to commit - patch is empty",
        patchPath: patchPath,
        patchSize: 0,
        patchLines: 0,
      };
    }

    return {
      success: true,
      patchPath: patchPath,
      patchSize: patchSize,
      patchLines: patchLines,
    };
  }

  // No patch generated
  return {
    success: false,
    error: errorMessage || "No changes to commit - no commits found",
    patchPath: patchPath,
  };
}

module.exports = {
  generateGitPatch,
};
