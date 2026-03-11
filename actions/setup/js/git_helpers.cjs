// @ts-check
/// <reference types="@actions/github-script" />

const { spawnSync } = require("child_process");
const { ERR_SYSTEM } = require("./error_codes.cjs");

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
    ...spawnOptions,
  });

  if (result.error) {
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

module.exports = {
  execGitSync,
  getGitAuthEnv,
};
