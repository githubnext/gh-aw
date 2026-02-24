// @ts-check
/// <reference types="@actions/github-script" />

const fs = require("fs");
const path = require("path");

const { getBaseBranch } = require("./get_base_branch.cjs");
const { getErrorMessage } = require("./error_helpers.cjs");
const { execGitSync } = require("./git_helpers.cjs");

/**
 * Sanitize a branch name for use as a patch filename
 * Replaces path separators and special characters with dashes
 * @param {string} branchName - The branch name to sanitize
 * @returns {string} The sanitized branch name safe for use in a filename
 */
function sanitizeBranchNameForPatch(branchName) {
  if (!branchName) return "unknown";
  return branchName
    .replace(/[/\\:*?"<>|]/g, "-")
    .replace(/-{2,}/g, "-")
    .replace(/^-|-$/g, "")
    .toLowerCase();
}

/**
 * Get the patch file path for a given branch name
 * @param {string} branchName - The branch name
 * @returns {string} The full patch file path
 */
function getPatchPath(branchName) {
  const sanitized = sanitizeBranchNameForPatch(branchName);
  return `/tmp/gh-aw/aw-${sanitized}.patch`;
}

/**
 * Generates a git patch file for the current changes
 * @param {string} branchName - The branch name to generate patch for
 * @param {Object} [options] - Optional parameters
 * @param {string} [options.mode="full"] - Patch generation mode:
 *   - "full": Include all commits since merge-base with default branch (for create_pull_request)
 *   - "incremental": Only include commits since origin/branchName (for push_to_pull_request_branch)
 *     In incremental mode, origin/branchName is fetched explicitly and merge-base fallback is disabled.
 * @returns {Object} Object with patch info or error
 */
function generateGitPatch(branchName, options = {}) {
  const mode = options.mode || "full";
  const patchPath = getPatchPath(branchName);
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
        execGitSync(["show-ref", "--verify", "--quiet", `refs/heads/${branchName}`], { cwd });

        // Determine base ref for patch generation
        let baseRef;

        if (mode === "incremental") {
          // INCREMENTAL MODE (for push_to_pull_request_branch):
          // Only include commits that are new since origin/branchName.
          // This prevents including commits that already exist on the PR branch.
          // We must explicitly fetch origin/branchName and fail if it doesn't exist.

          try {
            // Explicitly fetch origin/branchName to ensure we have the latest
            // Use "--" to prevent branch names starting with "-" from being interpreted as options
            execGitSync(["fetch", "origin", "--", `${branchName}:refs/remotes/origin/${branchName}`], { cwd });
            baseRef = `origin/${branchName}`;
          } catch (fetchError) {
            // In incremental mode, we MUST have origin/branchName - no fallback
            errorMessage = `Cannot generate incremental patch: failed to fetch origin/${branchName}. ` + `This typically happens when the remote branch doesn't exist yet or was force-pushed. ` + `Error: ${getErrorMessage(fetchError)}`;
            // Don't try other strategies in incremental mode
            return {
              success: false,
              error: errorMessage,
              patchPath: patchPath,
            };
          }
        } else {
          // FULL MODE (for create_pull_request):
          // Include all commits since merge-base with default branch.
          // This is appropriate for creating new PRs where we want all changes.

          try {
            // Check if origin/branchName exists
            execGitSync(["show-ref", "--verify", "--quiet", `refs/remotes/origin/${branchName}`], { cwd });
            baseRef = `origin/${branchName}`;
          } catch {
            // origin/branchName doesn't exist - use merge-base with default branch
            // First check if origin/<defaultBranch> already exists locally (e.g., from checkout with fetch-depth: 0)
            // This is important for cross-repo checkouts where persist-credentials: false prevents fetching
            let hasLocalDefaultBranch = false;
            try {
              execGitSync(["show-ref", "--verify", "--quiet", `refs/remotes/origin/${defaultBranch}`], { cwd });
              hasLocalDefaultBranch = true;
            } catch {
              // origin/<defaultBranch> doesn't exist locally, try to fetch it
              try {
                // Use "--" to prevent branch names starting with "-" from being interpreted as options
                execGitSync(["fetch", "origin", "--", defaultBranch], { cwd });
                hasLocalDefaultBranch = true;
              } catch {
                // Fetch failed (likely due to persist-credentials: false in cross-repo checkout)
                // We'll try other strategies below
              }
            }

            if (hasLocalDefaultBranch) {
              baseRef = execGitSync(["merge-base", "--", `origin/${defaultBranch}`, branchName], { cwd }).trim();
            } else {
              // No remote refs available - fall through to Strategy 2
              throw new Error("No remote refs available for merge-base calculation");
            }
          }
        }

        // Count commits to be included
        const commitCount = parseInt(execGitSync(["rev-list", "--count", `${baseRef}..${branchName}`], { cwd }).trim(), 10);

        if (commitCount > 0) {
          // Generate patch from the determined base to the branch
          const patchContent = execGitSync(["format-patch", `${baseRef}..${branchName}`, "--stdout"], { cwd });

          if (patchContent && patchContent.trim()) {
            fs.writeFileSync(patchPath, patchContent, "utf8");
            patchGenerated = true;
          }
        } else if (mode === "incremental") {
          // In incremental mode, zero commits means nothing new to push
          return {
            success: false,
            error: "No new commits to push - your changes may already be on the remote branch",
            patchPath: patchPath,
            patchSize: 0,
            patchLines: 0,
          };
        }
      } catch (branchError) {
        // Branch does not exist locally
        if (mode === "incremental") {
          return {
            success: false,
            error: `Branch ${branchName} does not exist locally. Cannot generate incremental patch.`,
            patchPath: patchPath,
          };
        }
      }
    }

    // Strategy 2: Check if commits were made to current HEAD since checkout
    if (!patchGenerated) {
      const currentHead = execGitSync(["rev-parse", "HEAD"], { cwd }).trim();

      if (!githubSha) {
        errorMessage = "GITHUB_SHA environment variable is not set";
      } else if (currentHead === githubSha) {
        // No commits have been made since checkout
      } else {
        // First verify GITHUB_SHA exists in this repo's git history
        // In cross-repo checkout scenarios, GITHUB_SHA is from the workflow repo,
        // not the target repo that's currently checked out
        let shaExistsInRepo = false;
        try {
          execGitSync(["cat-file", "-e", githubSha], { cwd });
          shaExistsInRepo = true;
        } catch {
          // GITHUB_SHA doesn't exist in this repo - likely a cross-repo checkout
          // This is expected when workflow repo != checked out repo
        }

        if (shaExistsInRepo) {
          // Check if GITHUB_SHA is an ancestor of current HEAD
          try {
            execGitSync(["merge-base", "--is-ancestor", githubSha, "HEAD"], { cwd });

            // Count commits between GITHUB_SHA and HEAD
            const commitCount = parseInt(execGitSync(["rev-list", "--count", `${githubSha}..HEAD`], { cwd }).trim(), 10);

            if (commitCount > 0) {
              // Generate patch from GITHUB_SHA to HEAD
              const patchContent = execGitSync(["format-patch", `${githubSha}..HEAD`, "--stdout"], { cwd });

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
    }

    // Strategy 3: Cross-repo fallback - find commits not reachable from any remote ref
    // This handles cases where:
    // - Cross-repo checkout with persist-credentials: false (can't fetch)
    // - GITHUB_SHA is from a different repo
    // - No origin/<defaultBranch> available locally
    if (!patchGenerated && branchName) {
      try {
        // Get all remote refs
        const remoteRefsOutput = execGitSync(["for-each-ref", "--format=%(refname)", "refs/remotes/"], { cwd }).trim();

        if (remoteRefsOutput) {
          // Build exclusion list from all remote refs
          const remoteRefs = remoteRefsOutput.split("\n").filter(r => r);

          if (remoteRefs.length > 0) {
            // Find commits on current branch not reachable from any remote ref
            // This gets commits the agent added that haven't been pushed anywhere
            const excludeArgs = remoteRefs.flatMap(ref => ["--not", ref]);
            const revListArgs = ["rev-list", "--count", branchName, ...excludeArgs];

            const commitCount = parseInt(execGitSync(revListArgs, { cwd }).trim(), 10);

            if (commitCount > 0) {
              // Get the merge-base with the first remote ref (typically origin/HEAD or origin/main)
              // to determine the starting point for the patch
              let baseCommit;
              for (const ref of remoteRefs) {
                try {
                  baseCommit = execGitSync(["merge-base", ref, branchName], { cwd }).trim();
                  if (baseCommit) break;
                } catch {
                  // Try next ref
                }
              }

              if (baseCommit) {
                const patchContent = execGitSync(["format-patch", `${baseCommit}..${branchName}`, "--stdout"], { cwd });

                if (patchContent && patchContent.trim()) {
                  fs.writeFileSync(patchPath, patchContent, "utf8");
                  patchGenerated = true;
                }
              }
            }
          }
        }
      } catch {
        // Strategy 3 failed - no remote refs available at all
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
  getPatchPath,
  sanitizeBranchNameForPatch,
};
