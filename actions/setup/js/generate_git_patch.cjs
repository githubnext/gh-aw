// @ts-check
/// <reference types="@actions/github-script" />

// SEC-005: This module generates git patches via git CLI commands and does not make
// GitHub API calls using a user-supplied target repository. The "target repo" references
// in documentation describe cross-repo checkout scenarios only; no validateTargetRepo
// allowlist check is required in this handler.

const fs = require("fs");
const path = require("path");

const { getErrorMessage } = require("./error_helpers.cjs");
const { execGitSync, getGitAuthEnv } = require("./git_helpers.cjs");
const { ERR_SYSTEM } = require("./error_codes.cjs");
const { sanitizeForFilename, sanitizeBranchNameForPatch, sanitizeRepoSlugForPatch, getPatchPath, getPatchPathForRepo, buildExcludePathspecs, computeIncrementalDiffSize } = require("./git_patch_utils.cjs");

// sanitizeForFilename is re-exported below for backward compatibility with
// existing callers that imported it from this module.
void sanitizeForFilename;

/**
 * Debug logging helper - logs to stderr when DEBUG env var matches
 * @param {string} message - Debug message to log
 */
function debugLog(message) {
  const debug = process.env.DEBUG || "";
  if (debug === "*" || debug.includes("generate_git_patch") || debug.includes("patch")) {
    console.error(`[generate_git_patch] ${message}`);
  }
}

/**
 * Generates a git patch file for the current changes
 * @param {string} branchName - The branch name to generate patch for
 * @param {string} baseBranch - The base branch to diff against (e.g., "main", "master")
 * @param {Object} [options] - Optional parameters
 * @param {string} [options.mode="full"] - Patch generation mode:
 *   - "full": Include all commits since merge-base with default branch (for create_pull_request)
 *   - "incremental": Only include commits since origin/branchName (for push_to_pull_request_branch)
 *     In incremental mode, origin/branchName is fetched explicitly and merge-base fallback is disabled.
 * @param {string} [options.cwd] - Working directory for git commands. Defaults to GITHUB_WORKSPACE or process.cwd().
 *   Use this for multi-repo scenarios where repos are checked out to subdirectories.
 * @param {string} [options.repoSlug] - Repository slug (owner/repo) to include in patch filename for disambiguation.
 *   Required for multi-repo scenarios to prevent patch file collisions.
 * @param {string} [options.token] - GitHub token for git authentication. Falls back to GITHUB_TOKEN env var.
 *   Use this for cross-repo scenarios where a custom PAT with access to the target repo is needed.
 * @param {string[]} [options.excludedFiles] - Glob patterns for files to exclude from the patch.
 *   Each pattern is passed to `git format-patch` as a `:(exclude)<pattern>` magic pathspec so
 *   matching files are never included in the generated patch.
 * @returns {Promise<Object>} Object with patch info or error
 */
async function generateGitPatch(branchName, baseBranch, options = {}) {
  const mode = options.mode || "full";
  // Support custom cwd for multi-repo scenarios
  const cwd = options.cwd || process.env.GITHUB_WORKSPACE || process.cwd();
  // Include repo slug in patch path for multi-repo disambiguation

  // Build :(exclude) pathspec arguments from the excludedFiles option.
  // These are appended after "--" so git treats them as pathspecs, not revisions.
  // Using git's native pathspec magic keeps the exclusions out of the patch entirely
  // without any post-processing of the generated patch file.
  const excludeArgsArr = buildExcludePathspecs(options.excludedFiles);

  /**
   * Returns the arguments to append to a format-patch call when excludedFiles is set.
   * @returns {string[]}
   */
  function excludeArgs() {
    return excludeArgsArr;
  }
  const patchPath = options.repoSlug ? getPatchPathForRepo(branchName, options.repoSlug) : getPatchPath(branchName);

  // Validate baseBranch early to avoid confusing git errors (e.g., origin/undefined)
  if (typeof baseBranch !== "string" || baseBranch.trim() === "") {
    const errorMessage = "baseBranch is required and must be a non-empty string (received: " + String(baseBranch) + ")";
    debugLog(`Invalid baseBranch: ${errorMessage}`);
    return {
      patchPath,
      patchGenerated: false,
      errorMessage,
    };
  }

  const defaultBranch = baseBranch;
  const githubSha = process.env.GITHUB_SHA;

  debugLog(`Starting patch generation: mode=${mode}, branch=${branchName}, defaultBranch=${defaultBranch}`);
  debugLog(`Environment: cwd=${cwd}, GITHUB_SHA=${githubSha || "(not set)"}`);

  // Ensure /tmp/gh-aw directory exists
  const patchDir = path.dirname(patchPath);
  if (!fs.existsSync(patchDir)) {
    fs.mkdirSync(patchDir, { recursive: true });
  }

  let patchGenerated = false;
  let errorMessage = null;
  // Track the resolved base commit SHA so consumers (e.g. create_pull_request fallback)
  // can use it directly. The From <sha> header in format-patch output contains the
  // *new* commit SHA which won't exist in the target checkout.
  let baseCommitSha = null;

  try {
    // Strategy 1: If we have a branch name, check if that branch exists and get its diff
    if (branchName) {
      debugLog(`Strategy 1: Checking if branch '${branchName}' exists locally`);
      // Check if the branch exists locally
      try {
        execGitSync(["show-ref", "--verify", "--quiet", `refs/heads/${branchName}`], { cwd });
        debugLog(`Strategy 1: Branch '${branchName}' exists locally`);

        // Determine base ref for patch generation
        let baseRef;

        if (mode === "incremental") {
          // INCREMENTAL MODE (for push_to_pull_request_branch):
          // Only include commits that are new since origin/branchName.
          // This prevents including commits that already exist on the PR branch.
          // Prefer a fresh fetch of origin/branchName; fall back to the existing
          // remote tracking ref (set up by the initial shallow checkout) when the
          // fetch fails (e.g. due to shallow clone limitations or missing credentials).

          debugLog(`Strategy 1 (incremental): Fetching origin/${branchName}`);
          // Configure git authentication via GIT_CONFIG_* environment variables.
          // This ensures the fetch works when .git/config credentials are unavailable
          // (e.g. after clean_git_credentials.sh) and on GitHub Enterprise Server (GHES).
          // Use options.token when provided (cross-repo PAT), falling back to GITHUB_TOKEN.
          // SECURITY: The auth header is passed via env vars so it is never written to
          // .git/config on disk, preventing file-monitoring attacks.
          const fetchEnv = { ...process.env, ...getGitAuthEnv(options.token) };

          try {
            // Explicitly fetch origin/branchName to ensure we have the latest
            // Use "--" to prevent branch names starting with "-" from being interpreted as options
            execGitSync(["fetch", "origin", "--", `${branchName}:refs/remotes/origin/${branchName}`], { cwd, env: fetchEnv });
            baseRef = `origin/${branchName}`;
            debugLog(`Strategy 1 (incremental): Successfully fetched, baseRef=${baseRef}`);
          } catch (fetchError) {
            // Fetch failed. Check if origin/branchName already exists from the initial shallow checkout.
            // This handles cases where git fetch fails due to shallow clone limitations or when
            // GITHUB_TOKEN is unavailable in the MCP server process (e.g. after clean_git_credentials.sh).
            // Using the existing remote tracking ref as a fallback is safe: it represents the state
            // of the branch at checkout time, so the incremental patch will include all commits
            // made by the agent since then.
            debugLog(`Strategy 1 (incremental): Fetch failed - ${getErrorMessage(fetchError)}, checking for existing remote tracking ref`);
            try {
              execGitSync(["show-ref", "--verify", "--quiet", `refs/remotes/origin/${branchName}`], { cwd });
              // Remote tracking ref exists from initial shallow checkout — use it as base
              baseRef = `origin/${branchName}`;
              debugLog(`Strategy 1 (incremental): Using existing remote tracking ref as fallback, baseRef=${baseRef}`);
            } catch (refCheckError) {
              // No remote tracking ref at all — cannot safely generate an incremental patch.
              // Report both errors: the original fetch failure and the missing ref.
              debugLog(`Strategy 1 (incremental): No existing remote tracking ref found (${getErrorMessage(refCheckError)}), failing`);
              errorMessage = `Cannot generate incremental patch: failed to fetch origin/${branchName} and no existing remote tracking ref found. This typically happens when the remote branch doesn't exist yet or was force-pushed. Fetch error: ${getErrorMessage(fetchError)}`;
              return {
                success: false,
                error: errorMessage,
                patchPath: patchPath,
              };
            }
          }
        } else {
          // FULL MODE (for create_pull_request):
          // Include all commits since merge-base with default branch.
          // This is appropriate for creating new PRs where we want all changes.

          debugLog(`Strategy 1 (full): Checking if origin/${branchName} exists`);
          try {
            // Check if origin/branchName exists
            execGitSync(["show-ref", "--verify", "--quiet", `refs/remotes/origin/${branchName}`], { cwd });
            baseRef = `origin/${branchName}`;
            debugLog(`Strategy 1 (full): Using existing origin/${branchName} as baseRef`);
          } catch {
            // origin/branchName doesn't exist - use merge-base with default branch
            debugLog(`Strategy 1 (full): origin/${branchName} not found, trying merge-base with ${defaultBranch}`);
            // First check if origin/<defaultBranch> already exists locally (e.g., from checkout with fetch-depth: 0)
            // This is important for cross-repo checkouts where persist-credentials: false prevents fetching
            let hasLocalDefaultBranch = false;
            try {
              execGitSync(["show-ref", "--verify", "--quiet", `refs/remotes/origin/${defaultBranch}`], { cwd });
              hasLocalDefaultBranch = true;
              debugLog(`Strategy 1 (full): origin/${defaultBranch} exists locally`);
            } catch {
              // origin/<defaultBranch> doesn't exist locally, try to fetch it
              debugLog(`Strategy 1 (full): origin/${defaultBranch} not found locally, attempting fetch`);
              try {
                // Configure git authentication via GIT_CONFIG_* environment variables.
                // This ensures the fetch works when .git/config credentials are unavailable
                // (e.g. after clean_git_credentials.sh) and on GitHub Enterprise Server (GHES).
                // Use options.token when provided (cross-repo PAT), falling back to GITHUB_TOKEN.
                // SECURITY: The auth header is passed via env vars so it is never written to
                // .git/config on disk, preventing file-monitoring attacks.
                const fullFetchEnv = { ...process.env, ...getGitAuthEnv(options.token) };
                // Use "--" to prevent branch names starting with "-" from being interpreted as options
                execGitSync(["fetch", "origin", "--", defaultBranch], { cwd, env: fullFetchEnv });
                hasLocalDefaultBranch = true;
                debugLog(`Strategy 1 (full): Successfully fetched origin/${defaultBranch}`);
              } catch (fetchErr) {
                // Fetch failed (likely due to persist-credentials: false in cross-repo checkout)
                // We'll try other strategies below
                debugLog(`Strategy 1 (full): Fetch failed - ${getErrorMessage(fetchErr)} (will try other strategies)`);
              }
            }

            if (hasLocalDefaultBranch) {
              baseRef = execGitSync(["merge-base", "--", `origin/${defaultBranch}`, branchName], { cwd }).trim();
              debugLog(`Strategy 1 (full): Computed merge-base: ${baseRef}`);
            } else {
              // No remote refs available - fall through to Strategy 2
              debugLog(`Strategy 1 (full): No remote refs available, falling through to Strategy 2`);
              throw new Error(`${ERR_SYSTEM}: No remote refs available for merge-base calculation`);
            }
          }
        }

        // Resolve baseRef to a SHA so we can record it for consumers
        baseCommitSha = execGitSync(["rev-parse", baseRef], { cwd }).trim();
        debugLog(`Strategy 1: Resolved baseRef ${baseRef} to SHA ${baseCommitSha}`);

        // Count commits to be included
        const commitCount = parseInt(execGitSync(["rev-list", "--count", `${baseRef}..${branchName}`], { cwd }).trim(), 10);
        debugLog(`Strategy 1: Found ${commitCount} commits between ${baseRef} and ${branchName}`);

        if (commitCount > 0) {
          // Generate patch from the determined base to the branch
          const patchContent = execGitSync(["format-patch", `${baseRef}..${branchName}`, "--stdout", ...excludeArgs()], { cwd });

          if (patchContent && patchContent.trim()) {
            fs.writeFileSync(patchPath, patchContent, "utf8");
            patchGenerated = true;
            debugLog(`Strategy 1: SUCCESS - Generated patch with ${patchContent.split("\n").length} lines`);
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

        // In incremental mode, the patch must be measured relative to the existing
        // PR branch head (origin/<branch>), never relative to the default branch.
        // If Strategy 1 did not produce a patch (e.g. format-patch yielded empty
        // output for an unusual commit shape — excluded-files filtering away every
        // change, or binary-only commits with unusual encoding), do NOT fall
        // through to Strategy 2 or Strategy 3 — those use GITHUB_SHA..HEAD or
        // merge-base with a remote ref and would produce a checkout-base diff
        // (which can be many MB on a long-running branch). Returning an explicit
        // error preserves the "incremental" contract that the patch reflects only
        // the new commits.
        if (!patchGenerated && mode === "incremental") {
          debugLog(`Strategy 1 (incremental): format-patch produced no output for ${baseRef}..${branchName} despite ${commitCount} incremental commit(s), refusing to fall through to checkout-base strategies`);
          return {
            success: false,
            error: `Cannot generate incremental patch: git format-patch produced no output for ${baseRef}..${branchName} despite ${commitCount} incremental commit(s).`,
            patchPath: patchPath,
          };
        }
      } catch (branchError) {
        // Branch does not exist locally
        debugLog(`Strategy 1: Branch '${branchName}' does not exist locally - ${getErrorMessage(branchError)}`);
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
      debugLog(`Strategy 2: Checking commits since GITHUB_SHA`);
      const currentHead = execGitSync(["rev-parse", "HEAD"], { cwd }).trim();
      debugLog(`Strategy 2: currentHead=${currentHead}, GITHUB_SHA=${githubSha || "(not set)"}`);

      if (!githubSha) {
        debugLog(`Strategy 2: GITHUB_SHA not set, cannot use this strategy`);
        errorMessage = "GITHUB_SHA environment variable is not set";
      } else if (currentHead === githubSha) {
        // No commits have been made since checkout
        debugLog(`Strategy 2: HEAD equals GITHUB_SHA - no new commits`);
      } else {
        // First verify GITHUB_SHA exists in this repo's git history
        // In cross-repo checkout scenarios, GITHUB_SHA is from the workflow repo,
        // not the checked-out repository
        let shaExistsInRepo = false;
        try {
          execGitSync(["cat-file", "-e", githubSha], { cwd });
          shaExistsInRepo = true;
          debugLog(`Strategy 2: GITHUB_SHA exists in this repo`);
        } catch {
          // GITHUB_SHA doesn't exist in this repo - likely a cross-repo checkout
          // This is expected when workflow repo != checked out repo
          debugLog(`Strategy 2: GITHUB_SHA not found in repo (cross-repo checkout?)`);
        }

        if (shaExistsInRepo) {
          // Check if GITHUB_SHA is an ancestor of current HEAD
          try {
            execGitSync(["merge-base", "--is-ancestor", githubSha, "HEAD"], { cwd });
            debugLog(`Strategy 2: GITHUB_SHA is an ancestor of HEAD`);

            // Record GITHUB_SHA as the base commit
            baseCommitSha = githubSha;

            // Count commits between GITHUB_SHA and HEAD
            const commitCount = parseInt(execGitSync(["rev-list", "--count", `${githubSha}..HEAD`], { cwd }).trim(), 10);
            debugLog(`Strategy 2: Found ${commitCount} commits between GITHUB_SHA and HEAD`);

            if (commitCount > 0) {
              // Generate patch from GITHUB_SHA to HEAD
              const patchContent = execGitSync(["format-patch", `${githubSha}..HEAD`, "--stdout", ...excludeArgs()], { cwd });

              if (patchContent && patchContent.trim()) {
                fs.writeFileSync(patchPath, patchContent, "utf8");
                patchGenerated = true;
                debugLog(`Strategy 2: SUCCESS - Generated patch with ${patchContent.split("\n").length} lines`);
              }
            }
          } catch (ancestorErr) {
            // GITHUB_SHA is not an ancestor of HEAD - repository state has diverged
            debugLog(`Strategy 2: GITHUB_SHA is not an ancestor of HEAD - ${getErrorMessage(ancestorErr)}`);
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
      debugLog(`Strategy 3: Cross-repo fallback - finding commits not reachable from remote refs`);
      try {
        // Get all remote refs
        const remoteRefsOutput = execGitSync(["for-each-ref", "--format=%(refname)", "refs/remotes/"], { cwd }).trim();

        if (remoteRefsOutput) {
          // Build exclusion list from all remote refs
          const remoteRefs = remoteRefsOutput.split("\n").filter(r => r);
          debugLog(`Strategy 3: Found ${remoteRefs.length} remote refs: ${remoteRefs.slice(0, 5).join(", ")}${remoteRefs.length > 5 ? "..." : ""}`);

          if (remoteRefs.length > 0) {
            // Find commits on current branch not reachable from any remote ref
            // This gets commits the agent added that haven't been pushed anywhere
            const remoteExcludeArgs = remoteRefs.flatMap(ref => ["--not", ref]);
            const revListArgs = ["rev-list", "--count", branchName, ...remoteExcludeArgs];

            const commitCount = parseInt(execGitSync(revListArgs, { cwd }).trim(), 10);
            debugLog(`Strategy 3: Found ${commitCount} commits not reachable from any remote ref`);

            if (commitCount > 0) {
              // Get the merge-base with the first remote ref (typically origin/HEAD or origin/main)
              // to determine the starting point for the patch
              let baseCommit;
              for (const ref of remoteRefs) {
                try {
                  baseCommit = execGitSync(["merge-base", ref, branchName], { cwd }).trim();
                  if (baseCommit) {
                    debugLog(`Strategy 3: Found merge-base ${baseCommit} with ref ${ref}`);
                    break;
                  }
                } catch {
                  // Try next ref
                }
              }

              if (baseCommit) {
                baseCommitSha = baseCommit;
                const patchContent = execGitSync(["format-patch", `${baseCommit}..${branchName}`, "--stdout", ...excludeArgs()], { cwd });

                if (patchContent && patchContent.trim()) {
                  fs.writeFileSync(patchPath, patchContent, "utf8");
                  patchGenerated = true;
                  debugLog(`Strategy 3: SUCCESS - Generated patch with ${patchContent.split("\n").length} lines`);
                }
              } else {
                debugLog(`Strategy 3: Could not find merge-base with any remote ref`);
              }
            }
          }
        } else {
          debugLog(`Strategy 3: No remote refs found`);
        }
      } catch (strategy3Err) {
        // Strategy 3 failed - no remote refs available at all
        debugLog(`Strategy 3: Failed - ${getErrorMessage(strategy3Err)}`);
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
      debugLog(`Final: Patch file exists but is empty`);
      return {
        success: false,
        error: "No changes to commit - patch is empty",
        patchPath: patchPath,
        patchSize: 0,
        patchLines: 0,
      };
    }

    // In incremental mode, also compute the net diff size between baseRef and the
    // branch tip. The format-patch file size (patchSize) is the sum of every
    // commit's individual diff plus per-commit metadata headers, which can be
    // significantly larger than the actual net change. Consumers (e.g.
    // push_to_pull_request_branch) should validate `max_patch_size` against the
    // incremental net diff so the limit reflects how much the branch will
    // actually change, not the cumulative size of the commit history.
    //
    // The measurement itself (stream to temp file via `git diff --output`, stat,
    // cleanup) is extracted into git_patch_utils.computeIncrementalDiffSize so
    // it is O(1) memory and independently unit-testable against a real repo.
    let diffSize = null;
    if (mode === "incremental" && baseCommitSha && branchName) {
      diffSize = computeIncrementalDiffSize({
        baseRef: baseCommitSha,
        headRef: branchName,
        cwd,
        tmpPath: `${patchPath}.diff.tmp`,
        excludedFiles: options.excludedFiles,
      });
      debugLog(`Final: diffSize=${diffSize ?? "(n/a)"} bytes (baseRef=${baseCommitSha}..${branchName})`);
    }

    debugLog(`Final: SUCCESS - patchSize=${patchSize} bytes, patchLines=${patchLines}, diffSize=${diffSize ?? "(n/a)"} bytes, baseCommit=${baseCommitSha || "(unknown)"}`);
    return {
      success: true,
      patchPath: patchPath,
      patchSize: patchSize,
      patchLines: patchLines,
      diffSize: diffSize,
      baseCommit: baseCommitSha,
    };
  }

  // No patch generated
  debugLog(`Final: FAILED - ${errorMessage || "No changes to commit - no commits found"}`);
  return {
    success: false,
    error: errorMessage || "No changes to commit - no commits found",
    patchPath: patchPath,
  };
}

module.exports = {
  generateGitPatch,
  getPatchPath,
  getPatchPathForRepo,
  sanitizeBranchNameForPatch,
  sanitizeRepoSlugForPatch,
};
