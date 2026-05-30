// @ts-check
/// <reference types="@actions/github-script" />

const { spawnSync } = require("child_process");
const { ERR_SYSTEM } = require("./error_codes.cjs");
const { getErrorMessage } = require("./error_helpers.cjs");

/**
 * Build GIT_CONFIG_* environment variables that inject an Authorization header
 * for git network operations (fetch, push, clone) without writing credentials
 * to .git/config on disk.
 *
 * Use this whenever .git/config credentials may have been cleaned (e.g. after
 * clean_git_credentials.sh runs in the agent job) to ensure git can still
 * authenticate against the GitHub server.
 *
 * SECURITY: Credentials are passed via GIT_CONFIG_* environment variables and
 * never written to .git/config, so they are not visible to file-monitoring
 * attacks and are not inherited by sub-processes that don't receive the env.
 *
 * @param {string} [token] - GitHub token to use. Falls back to GITHUB_TOKEN env var.
 * @returns {Object} Environment variables to spread into child_process/exec options.
 *   Returns an empty object when no token is available.
 */
function getGitAuthEnv(token) {
  const authToken = token || process.env.GITHUB_TOKEN;
  if (!authToken) {
    core.debug("getGitAuthEnv: no token available, git network operations may fail if credentials were cleaned");
    return {};
  }
  const serverUrl = (process.env.GITHUB_SERVER_URL || "https://github.com").replace(/\/$/, "");
  const tokenBase64 = Buffer.from(`x-access-token:${authToken}`).toString("base64");
  return {
    GIT_CONFIG_COUNT: "1",
    GIT_CONFIG_KEY_0: `http.${serverUrl}/.extraheader`,
    GIT_CONFIG_VALUE_0: `Authorization: basic ${tokenBase64}`,
  };
}

/**
 * Safely execute git command using spawnSync with args array to prevent shell injection
 * @param {string[]} args - Git command arguments
 * @param {Object} options - Spawn options; set suppressLogs: true to avoid core.error annotations for expected failures
 * @returns {string} Command output
 * @throws {Error} If command fails
 */
function execGitSync(args, options = {}) {
  // Extract suppressLogs before spreading into spawnSync options.
  // suppressLogs is a custom control flag (not a valid spawnSync option) that
  // routes failure details to core.debug instead of core.error, preventing
  // spurious GitHub Actions error annotations for expected failures (e.g., when
  // a branch does not yet exist).
  const { suppressLogs = false, ...spawnOptions } = options;

  // Log the git command being executed for debugging (but redact credentials)
  const gitCommand = `git ${args
    .map(arg => {
      // Redact credentials in URLs
      if (typeof arg === "string" && arg.includes("://") && arg.includes("@")) {
        return arg.replace(/(https?:\/\/)[^@]+@/, "$1***@");
      }
      return arg;
    })
    .join(" ")}`;

  core.debug(`Executing git command: ${gitCommand}`);

  const result = spawnSync("git", args, {
    encoding: "utf8",
    maxBuffer: 100 * 1024 * 1024, // 100 MB — prevents ENOBUFS on large diffs (e.g. git format-patch)
    ...spawnOptions,
  });

  if (result.error) {
    // Detect ENOBUFS (buffer overflow) and provide a more actionable message
    /** @type {NodeJS.ErrnoException} */
    const spawnError = result.error;
    if (spawnError.code === "ENOBUFS") {
      /** @type {NodeJS.ErrnoException} */
      const bufferError = new Error(`${ERR_SYSTEM}: Git command output exceeded buffer limit (ENOBUFS). The output from '${args[0]}' is too large for the configured maxBuffer. Consider reducing the diff size or increasing maxBuffer.`);
      bufferError.code = "ENOBUFS";
      core.error(`Git command buffer overflow: ${gitCommand}`);
      throw bufferError;
    }
    // Spawn-level errors (e.g. ENOENT, EACCES) are always unexpected — log
    // via core.error regardless of suppressLogs.
    core.error(`Git command failed with error: ${result.error.message}`);
    throw result.error;
  }

  if (result.status !== 0) {
    const errorMsg = `${ERR_SYSTEM}: ${result.stderr || `Git command failed with status ${result.status}`}`;
    if (suppressLogs) {
      core.debug(`Git command failed (expected): ${gitCommand}`);
      core.debug(`Exit status: ${result.status}`);
      if (result.stderr) {
        core.debug(`Stderr: ${result.stderr}`);
      }
    } else {
      core.error(`Git command failed: ${gitCommand}`);
      core.error(`Exit status: ${result.status}`);
      if (result.stderr) {
        core.error(`Stderr: ${result.stderr}`);
      }
    }
    throw new Error(errorMsg);
  }

  if (result.stdout) {
    core.debug(`Git command output: ${result.stdout.substring(0, 200)}${result.stdout.length > 200 ? "..." : ""}`);
  } else {
    core.debug("Git command completed successfully with no output");
  }

  return result.stdout;
}

/**
 * Check whether a commit range contains any merge commits.
 *
 * `git am` (the default patch transport) cannot apply merge commits — it only
 * handles linear patches produced by `git format-patch`. Callers can use this
 * helper to detect when a range requires the `bundle` transport instead, which
 * preserves merge commit topology by transferring git objects directly.
 *
 * Returns `false` (rather than throwing) when the underlying git command fails
 * — for example when one of the refs cannot be resolved. Callers should treat
 * "unknown" as "no merge commits detected" so that a detection failure never
 * blocks the normal patch path.
 *
 * @param {string} baseRef - The base ref (exclusive). Example: "origin/feature".
 * @param {string} headRef - The head ref (inclusive). Example: "feature".
 * @param {Object} [options]
 * @param {string} [options.cwd] - Working directory for the git command.
 * @returns {boolean} True if at least one merge commit exists in baseRef..headRef.
 */
function hasMergeCommitsInRange(baseRef, headRef, options = {}) {
  if (!baseRef || !headRef) return false;
  try {
    const out = execGitSync(["rev-list", "--merges", "--count", `${baseRef}..${headRef}`], {
      cwd: options.cwd,
      suppressLogs: true,
    });
    const count = parseInt(out.trim(), 10);
    return Number.isFinite(count) && count > 0;
  } catch {
    // Detection failure — treat as no merge commits to avoid blocking the
    // normal patch path. The caller's downstream patch generation will surface
    // any actionable error.
    return false;
  }
}

/**
 * Probe shallow-repository status before fetching a git bundle.
 *
 * Bundles generated from a commit range can declare prerequisite commits. A
 * depth-1 checkout may not contain those prerequisites, and `git fetch <bundle>`
 * can reject the bundle before the caller can update refs.
 *
 * IMPORTANT: Do not unshallow here. Full-history fetches are prohibitively
 * expensive for large monorepos. Callers recover from prerequisite failures by
 * fetching only the missing commit objects from origin and retrying.
 *
 * @param {{ getExecOutput: Function, exec: Function }} execApi - Exec API to run git commands.
 * @param {Object} [options] - Options passed through to exec calls.
 * @returns {Promise<void>}
 */
async function ensureFullHistoryForBundle(execApi, options = {}) {
  let stdout;
  try {
    ({ stdout } = await execApi.getExecOutput("git", ["rev-parse", "--is-shallow-repository"], options));
  } catch (error) {
    const message = error instanceof Error ? error.message : String(error);
    core.warning(`Could not determine shallow repository status; skipping full-history fetch probe: ${message}`);
    return;
  }
  if (stdout.trim() === "true") {
    core.info("Repository is shallow; skipping full-history fetch and relying on prerequisite recovery");
  }
}

/**
 * Return true when the local repository is shallow OR has sparse-checkout enabled.
 *
 * This is the gate for using `--filter=blob:none` on follow-up fetches (e.g. bundle
 * prerequisite recovery). In a full, non-sparse clone the repo already contains all
 * blobs for committed history; adding `--filter=blob:none` to a fetch would convert
 * it to a partial clone and cause subsequent operations to lazily re-fetch blobs.
 * In shallow or sparse checkouts we already accept partial object availability, so
 * filtering blobs is consistent and saves bandwidth.
 *
 * Both probes are best-effort — on any error we return `false` (do not filter),
 * which is the safe default that preserves the legacy unfiltered fetch behavior.
 *
 * @param {{ getExecOutput: Function }} execApi - Exec API to run git commands.
 * @param {Object} [options] - Options passed through to exec calls.
 * @returns {Promise<boolean>}
 */
async function isShallowOrSparseCheckout(execApi, options = {}) {
  const probeOptions = { ...options, ignoreReturnCode: true };
  try {
    const { stdout, exitCode } = await execApi.getExecOutput("git", ["rev-parse", "--is-shallow-repository"], probeOptions);
    if (exitCode === 0 && stdout.trim() === "true") {
      return true;
    }
  } catch {
    // Fall through to sparse check; if both probes fail, return false (no filter).
  }
  try {
    const { stdout, exitCode } = await execApi.getExecOutput("git", ["config", "--get", "core.sparseCheckout"], probeOptions);
    if (exitCode === 0 && stdout.trim().toLowerCase() === "true") {
      return true;
    }
  } catch {
    // Fall through.
  }
  return false;
}

/**
 * Extract prerequisite commit SHAs from git bundle fetch error output.
 *
 * When `git fetch <bundle>` fails because the local repository is missing the
 * bundle's base commits, git prints:
 *   error: Repository lacks these prerequisite commits:
 *   error: <sha1>
 *   error: <sha2>
 *   ...
 *
 * This function parses the raw stderr/error text and returns the deduplicated
 * list of missing commit SHAs so callers can fetch them from origin and retry.
 *
 * NOTE: The @actions/exec `exec()` function throws with a generic
 * "The process '...' failed with exit code 1" message that does NOT include
 * stderr. Callers must use `getExecOutput()` with `ignoreReturnCode: true`
 * and pass the returned `stderr` field to this function.
 *
 * @param {string} message - Raw stderr text from the failed bundle fetch.
 * @returns {string[]} Deduplicated lowercase 40-character commit SHAs, or [] if none found.
 */
function extractBundlePrerequisiteCommits(message) {
  if (!message || !/lacks these prerequisite commits/i.test(message)) {
    return [];
  }
  return [...new Set((message.match(/\b[0-9a-f]{40}\b/gi) || []).map(sha => sha.toLowerCase()))];
}

/**
 * Rewrite the commit range `baseRef..HEAD` as a single regular commit carrying the same tree.
 *
 * Saves the current HEAD, soft-resets to `baseRef`, validates that at least one file is
 * staged, and recommits under `commitMessage`.  On any failure the original HEAD is restored
 * via `reset --hard` and the error is re-thrown so the caller can surface an actionable
 * message.
 *
 * @param {string} baseRef - The base ref to reset to (e.g. `"origin/main"` or a SHA).
 * @param {string} commitMessage - Commit message for the linearized commit.
 * @param {{ exec: Function, getExecOutput: Function }} execApi - Actions exec API (e.g. the `exec` global).
 * @param {Object} [opts]
 * @param {Object} [opts.gitOpts] - Extra options passed to every exec call (e.g. `{ cwd }`).
 *   When omitted, exec calls are made without additional options.
 * @param {string[]} [opts.commitFlags] - Extra flags prepended before `-m` in the `git commit`
 *   invocation (e.g. `["--allow-empty", "--no-verify"]`).
 * @returns {Promise<string>} The new HEAD SHA after the rewrite.
 * @throws {Error} If the soft reset, staged-changes validation, or recommit fails.
 */
async function linearizeRangeAsCommit(baseRef, commitMessage, execApi, opts = {}) {
  const { gitOpts, commitFlags = [] } = opts;
  // Spread gitOpts into exec calls only when it is explicitly provided — passing
  // `undefined` as a third argument changes the arity seen by mocks in tests.
  const execArgs = gitOpts !== undefined ? [gitOpts] : [];

  const { stdout: originalHeadOut } = await execApi.getExecOutput("git", ["rev-parse", "HEAD"], ...execArgs);
  const originalHead = originalHeadOut.trim();
  if (!originalHead) {
    throw new Error("Could not resolve current HEAD before linearizing range");
  }

  try {
    await execApi.exec("git", ["reset", "--soft", baseRef], ...execArgs);
    const { stdout: stagedFilesOut } = await execApi.getExecOutput("git", ["diff", "--cached", "--name-only"], ...execArgs);
    if (!stagedFilesOut.trim()) {
      throw new Error(`No staged changes found after soft reset to ${baseRef}. ` + `The commit range may contain only no-op or empty commits. ` + `Ensure your commits contain actual file changes before pushing.`);
    }
    await execApi.exec("git", ["commit", ...commitFlags, "-m", commitMessage], ...execArgs);
    const { stdout: newHeadOut } = await execApi.getExecOutput("git", ["rev-parse", "HEAD"], ...execArgs);
    return newHeadOut.trim();
  } catch (rewriteError) {
    try {
      await execApi.exec("git", ["reset", "--hard", originalHead], ...execArgs);
      core.warning(`linearizeRangeAsCommit: rewrite failed; restored original HEAD ${originalHead}`);
    } catch (restoreError) {
      core.warning(`linearizeRangeAsCommit: rollback also failed: ${getErrorMessage(restoreError)}`);
    }
    throw new Error(`Failed to linearize ${baseRef}..HEAD as a single commit: ${getErrorMessage(rewriteError)}`, { cause: rewriteError });
  }
}

module.exports = {
  execGitSync,
  ensureFullHistoryForBundle,
  extractBundlePrerequisiteCommits,
  getGitAuthEnv,
  hasMergeCommitsInRange,
  isShallowOrSparseCheckout,
  linearizeRangeAsCommit,
};
