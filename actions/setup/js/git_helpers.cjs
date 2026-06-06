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
 * Safely execute git command using spawnSync with args array to prevent shell injection.
 *
 * Hardened against indefinite hangs: always runs git with non-interactive
 * credential settings (GIT_TERMINAL_PROMPT=0, GCM_INTERACTIVE=Never,
 * GIT_ASKPASS=/bin/echo) and a default 60s timeout (override via options.timeout).
 *
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

  // Hard guards against indefinite hangs:
  //  - GIT_TERMINAL_PROMPT=0 / GCM_INTERACTIVE=Never / GIT_ASKPASS make any
  //    credential rejection fail fast instead of opening an interactive prompt.
  //  - timeout (default 60s) ensures a stuck network/TLS handshake cannot
  //    wedge the calling event loop. Callers can override via options.timeout.
  const callerEnv = spawnOptions.env || process.env;
  const safeEnv = {
    ...callerEnv,
    GIT_TERMINAL_PROMPT: "0",
    GCM_INTERACTIVE: "Never",
    GIT_ASKPASS: "/bin/echo",
  };
  const defaultTimeoutMs = 60_000;

  const result = spawnSync("git", args, {
    encoding: "utf8",
    maxBuffer: 100 * 1024 * 1024, // 100 MB — prevents ENOBUFS on large diffs (e.g. git format-patch)
    timeout: defaultTimeoutMs,
    killSignal: "SIGKILL",
    ...spawnOptions,
    env: safeEnv,
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
    if (spawnError.code === "ETIMEDOUT") {
      /** @type {NodeJS.ErrnoException} */
      const timeoutError = new Error(`${ERR_SYSTEM}: Git command timed out after ${spawnOptions.timeout || defaultTimeoutMs}ms: ${gitCommand}`);
      timeoutError.code = "ETIMEDOUT";
      core.error(`Git command timed out: ${gitCommand}`);
      throw timeoutError;
    }
    // Spawn-level errors (e.g. ENOENT, EACCES) are always unexpected — log
    // via core.error regardless of suppressLogs.
    core.error(`Git command failed with error: ${result.error.message}`);
    throw result.error;
  }

  // spawnSync sets signal when the process was killed (including by the timeout).
  if (result.signal === "SIGKILL" || result.signal === "SIGTERM") {
    /** @type {NodeJS.ErrnoException} */
    const timeoutError = new Error(`${ERR_SYSTEM}: Git command killed (${result.signal}), likely due to timeout (${spawnOptions.timeout || defaultTimeoutMs}ms): ${gitCommand}`);
    timeoutError.code = "ETIMEDOUT";
    core.error(`Git command killed by signal ${result.signal}: ${gitCommand}`);
    throw timeoutError;
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
 * Ensure refs/remotes/origin/<branch> is available locally, attempting a
 * single fetch when it is not. Returns whether the ref now exists and
 * whether a fetch was required.
 *
 * Safe to call from the credential-less safe-outputs MCP server: execGitSync
 * runs git with GIT_TERMINAL_PROMPT=0 / GIT_ASKPASS=/bin/echo and a 60s
 * timeout, so the fetch attempt either succeeds (public repos, or when a
 * token was provided) or fails fast (private repos without credentials).
 * Callers MUST treat exists=false as a recoverable negative result rather
 * than an error condition.
 *
 * @param {string} branch - Branch name (without origin/ prefix)
 * @param {Object} options
 * @param {string} options.cwd - Working directory for git commands
 * @param {string} [options.token] - Optional auth token used for fetch
 * @param {boolean} [options.suppressLogs=false] - Whether to suppress execGitSync error logs
 * @returns {{ exists: boolean, fetched: boolean, fetchError?: Error }}
 *   fetchError is populated only when exists=false after a failed fetch attempt.
 */
function ensureOriginRemoteTrackingRef(branch, options) {
  const ref = `refs/remotes/origin/${branch}`;
  try {
    execGitSync(["show-ref", "--verify", "--quiet", ref], {
      cwd: options.cwd,
      suppressLogs: options.suppressLogs || false,
    });
    return { exists: true, fetched: false };
  } catch {
    try {
      const fetchEnv = { ...process.env, ...getGitAuthEnv(options.token) };
      execGitSync(["fetch", "origin", "--", `${branch}:refs/remotes/origin/${branch}`], {
        cwd: options.cwd,
        env: fetchEnv,
        suppressLogs: options.suppressLogs || false,
      });
      return { exists: true, fetched: true };
    } catch (fetchError) {
      return { exists: false, fetched: false, fetchError: /** @type {Error} */ fetchError };
    }
  }
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
 * Deepen sequence (per call to `git fetch --deepen=N`). Each value adds N
 * commits to the existing shallow history. Total reachable depth after the
 * final step is the sum of these values (~7850 commits).
 */
const BUNDLE_DEEPEN_STEPS = [50, 100, 200, 500, 1000, 2000, 4000];

/**
 * Extract prerequisite commit SHAs declared in a git bundle file.
 *
 * Runs `git bundle verify <file>` (with `ignoreReturnCode`) and parses the
 * "The bundle requires this ref:" section as well as the
 * "Repository lacks these prerequisite commits:" error block. Both formats
 * list the prerequisite commit SHAs.
 *
 * @param {{ getExecOutput: Function }} execApi
 * @param {string} bundleFilePath
 * @param {Object} [options]
 * @returns {Promise<string[]>} Deduplicated lowercase 40-char SHAs, or [] on failure.
 */
async function getBundlePrerequisites(execApi, bundleFilePath, options = {}) {
  try {
    const { stdout, stderr } = await execApi.getExecOutput("git", ["bundle", "verify", bundleFilePath], { ...options, ignoreReturnCode: true, silent: true });
    const combined = `${stdout || ""}\n${stderr || ""}`;
    const prereqs = new Set();
    const lines = combined.split(/\r?\n/);
    let inRequires = false;
    for (const line of lines) {
      if (/the bundle (requires|records) (this|these)/i.test(line)) {
        inRequires = true;
        continue;
      }
      if (/the bundle contains/i.test(line)) {
        inRequires = false;
        continue;
      }
      if (inRequires) {
        const match = line.match(/\b([0-9a-f]{40})\b/i);
        if (match) {
          prereqs.add(match[1].toLowerCase());
          continue;
        }
        if (line.trim() === "") {
          inRequires = false;
        }
      }
    }
    // Also pick up "Repository lacks these prerequisite commits:" block.
    for (const sha of extractBundlePrerequisiteCommits(combined)) {
      prereqs.add(sha);
    }
    return [...prereqs];
  } catch (error) {
    core.debug(`getBundlePrerequisites failed: ${getErrorMessage(error)}`);
    return [];
  }
}

/**
 * Check which of the given SHAs are NOT yet ancestors of `targetRef`.
 *
 * @param {{ getExecOutput: Function }} execApi
 * @param {string[]} shas
 * @param {string} targetRef
 * @param {Object} [options]
 * @returns {Promise<string[]>} SHAs still missing (not ancestors / not present).
 */
async function findMissingAncestors(execApi, shas, targetRef, options = {}) {
  const missing = [];
  for (const sha of shas) {
    const { exitCode } = await execApi.getExecOutput("git", ["merge-base", "--is-ancestor", sha, targetRef], { ...options, ignoreReturnCode: true, silent: true });
    if (exitCode !== 0) {
      missing.push(sha);
    }
  }
  return missing;
}

/**
 * Probe shallow-repository status before fetching a git bundle, and deepen
 * the local clone as needed so the bundle's prerequisite commits become
 * reachable from `origin/<baseRef>`.
 *
 * Bundles generated from a commit range can declare prerequisite commits. A
 * shallow checkout (e.g. `fetch-depth: 20`) may not contain those prerequisites,
 * and `git fetch <bundle>` will reject the bundle before the caller can update
 * refs. On a high-churn monorepo, `git fetch --unshallow` is catastrophic — it
 * downloads the entire history. Instead we iterate `git fetch origin <baseRef>
 * --deepen=<N>` with progressively larger N until every declared prerequisite
 * satisfies `git merge-base --is-ancestor <prereq> origin/<baseRef>`.
 *
 * When `deepenOptions.baseRef` or `deepenOptions.bundleFilePath` is missing
 * (legacy callers), the function falls back to the previous behavior of a
 * single `git fetch --unshallow origin`.
 *
 * @param {{ getExecOutput: Function, exec: Function }} execApi - Exec API to run git commands.
 * @param {Object} [options] - Options passed through to exec calls.
 * @param {Object} [deepenOptions]
 * @param {string} [deepenOptions.baseRef] - Remote branch name to deepen (no `origin/` prefix).
 * @param {string} [deepenOptions.bundleFilePath] - Path to the bundle file whose prerequisites must become reachable.
 * @returns {Promise<void>}
 */
async function ensureFullHistoryForBundle(execApi, options = {}, deepenOptions = {}) {
  let stdout;
  try {
    ({ stdout } = await execApi.getExecOutput("git", ["rev-parse", "--is-shallow-repository"], options));
  } catch (error) {
    const message = error instanceof Error ? error.message : String(error);
    core.warning(`Could not determine shallow repository status; skipping full-history fetch probe: ${message}`);
    return;
  }
  if (stdout.trim() !== "true") {
    return;
  }

  const { baseRef, bundleFilePath } = deepenOptions || {};

  // Legacy path: no base ref / bundle info known — fall back to a single
  // unshallow. Callers in monorepos should always supply baseRef + bundleFilePath
  // to get incremental deepening instead.
  if (!baseRef || !bundleFilePath) {
    core.info("Repository is shallow; fetching full history before bundle processing (no baseRef/bundle info; using --unshallow)");
    await execApi.exec("git", ["fetch", "--unshallow", "origin"], options);
    return;
  }

  const prereqs = await getBundlePrerequisites(execApi, bundleFilePath, options);
  if (prereqs.length === 0) {
    core.info("Bundle declares no prerequisites; no deepen required");
    return;
  }

  const targetRef = `origin/${baseRef}`;
  const alreadyMissing = await findMissingAncestors(execApi, prereqs, targetRef, options);
  if (alreadyMissing.length === 0) {
    core.info(`Bundle prerequisites already reachable from ${targetRef}; no deepen required`);
    return;
  }

  core.info(`Repository is shallow; iteratively deepening ${targetRef} to satisfy ${alreadyMissing.length} bundle prerequisite commit(s)`);
  let missing = alreadyMissing;
  for (const depth of BUNDLE_DEEPEN_STEPS) {
    core.info(`Fetching origin ${baseRef} with --deepen=${depth} (${missing.length} prerequisite(s) still missing)`);
    try {
      await execApi.exec("git", ["fetch", `--deepen=${depth}`, "origin", baseRef], options);
    } catch (fetchError) {
      core.warning(`git fetch --deepen=${depth} origin ${baseRef} failed: ${getErrorMessage(fetchError)}; aborting iterative deepen`);
      break;
    }
    missing = await findMissingAncestors(execApi, prereqs, targetRef, options);
    if (missing.length === 0) {
      core.info(`Bundle prerequisites reachable after --deepen=${depth}`);
      return;
    }
  }

  core.warning(`Bundle prerequisites still not reachable after iterative deepen (${missing.length} remaining); attempting --unshallow as a last resort`);
  try {
    await execApi.exec("git", ["fetch", "--unshallow", "origin", baseRef], options);
  } catch (unshallowError) {
    core.warning(`Fallback --unshallow fetch failed: ${getErrorMessage(unshallowError)}; bundle apply may still fail`);
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
  ensureOriginRemoteTrackingRef,
  extractBundlePrerequisiteCommits,
  getGitAuthEnv,
  hasMergeCommitsInRange,
  isShallowOrSparseCheckout,
  linearizeRangeAsCommit,
};
