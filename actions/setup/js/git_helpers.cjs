// @ts-check
/// <reference types="@actions/github-script" />

const { spawnSync } = require("child_process");
const { ERR_SYSTEM } = require("./error_codes.cjs");

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
};
