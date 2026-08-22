// @ts-check
/// <reference types="@actions/github-script" />
// This module relies on the `exec` and `core` globals injected by github-script at runtime.
// All callers must ensure these globals are set before invoking any helper.

const { getErrorMessage } = require("./error_helpers.cjs");
const { buildGitAuthEnv } = require("./git_auth_env.cjs");
const { maskSecret } = require("./actions_secret_masking.cjs");
const fs = require("fs");
const os = require("os");
const path = require("path");

/**
 * Build git authentication environment variables and mask both credential
 * representations in the surrounding GitHub Actions step.
 *
 * @param {string} [token]
 * @returns {Object}
 */
function getGitAuthEnv(token) {
  const authToken = token || process.env.GITHUB_TOKEN;
  if (!authToken) {
    core.debug("getGitAuthEnv: no token available, git network operations may fail if credentials were cleaned");
    return {};
  }
  maskSecret(authToken);
  return buildGitAuthEnv(authToken, maskSecret);
}

/**
 * Normalize a server URL by stripping any trailing slash so the git config key
 * matches exactly what actions/checkout writes (e.g. `http.https://github.com/.extraheader`).
 *
 * @param {string} serverUrl
 * @returns {string}
 */
function normalizeServerUrl(serverUrl) {
  return serverUrl.replace(/\/+$/, "");
}

/**
 * Find config files referenced by `includeIf.gitdir*.path` directives in the local
 * (repository) git config. `actions/checkout@v7` persists credentials in a separate
 * file (e.g. `$RUNNER_TEMP/git-credentials-<uuid>.config`) and wires it in via a
 * repository-local `includeIf.gitdir:<path>.path` entry rather than writing the
 * extraheader value directly into `.git/config`. Because `git config --local
 * --unset-all` only touches values set directly in the local scope, it never
 * clears values that arrive through an include, so the included credential
 * remains effective even after the local/global scopes are cleared.
 *
 * As a safety measure, only files that resolve under a known-safe temp directory
 * (`RUNNER_TEMP`, or the OS tmp dir as a fallback) are returned; this avoids ever
 * touching arbitrary paths referenced by a config file we do not control.
 *
 * @param {string} [cwd] - Optional working directory (the checkout) to resolve local config from
 * @returns {Promise<string[]>}
 */
async function findIncludedExtraheaderConfigFiles(cwd) {
  const baseOpts = { silent: true, ignoreReturnCode: true, ...(cwd ? { cwd } : {}) };

  // Step 1: get the key names only (avoids splitting issues for paths with spaces in the key).
  const namesResult = await exec.getExecOutput("git", ["config", "--local", "--name-only", "--get-regexp", "^includeif\\.gitdir.*\\.path$"], baseOpts);
  if (namesResult.exitCode !== 0 || !namesResult.stdout.trim()) {
    return [];
  }

  const safeRoots = [process.env.RUNNER_TEMP, os.tmpdir()].filter(/** @param {string | undefined} p @returns {p is string} */ p => p !== undefined && p !== "").map(p => path.resolve(p));

  const files = [];
  for (const keyName of namesResult.stdout
    .split("\n")
    .map(l => l.trim())
    .filter(Boolean)) {
    // Step 2: for each key name, retrieve its value(s) with --get-all.
    const valResult = await exec.getExecOutput("git", ["config", "--local", "--get-all", keyName], baseOpts);
    if (valResult.exitCode !== 0 || !valResult.stdout.trim()) continue;
    for (const valLine of valResult.stdout
      .split("\n")
      .map(l => l.trim())
      .filter(Boolean)) {
      const resolved = path.resolve(cwd || process.cwd(), valLine);
      // Harden against symlink escapes: resolve the real path before checking containment.
      let real;
      try {
        real = fs.realpathSync(resolved);
      } catch {
        // File may not exist yet (checkout hasn't created it) — fall back to the lexical path.
        real = resolved;
      }
      if (safeRoots.some(root => real === root || real.startsWith(root + path.sep))) {
        files.push(resolved);
      } else {
        core.warning(`git_auth_helpers: skipping includeIf-referenced config file outside safe temp directories: ${resolved}`);
      }
    }
  }
  return files;
}

/**
 * Unset the given git config key from all writable scopes (global and local),
 * as well as from any `includeIf.gitdir`-referenced config files (e.g. the
 * per-checkout credentials file written by `actions/checkout@v7`).
 * Errors are silently ignored because the key may be absent in some scopes
 * or a scope may not be accessible (e.g. read-only system scope on CI runners).
 *
 * This is necessary because `actions/checkout` typically writes credentials to
 * the global scope, and a plain `--replace-all` without an explicit scope flag
 * only removes values from the local scope, leaving the global value in place
 * and causing duplicate Authorization header HTTP 400 errors.
 *
 * @param {string} key - Full git config key (e.g. "http.https://github.com/.extraheader")
 * @param {string} [cwd] - Optional working directory for local-scope operations
 * @returns {Promise<void>}
 */
async function unsetExtraheaderAllScopes(key, cwd) {
  // Clear both the global scope (typically written by actions/checkout) and the
  // local scope (written by this helper). The system scope is intentionally
  // skipped because it is usually read-only in CI environments.
  const scopes = ["--global", "--local"];
  for (const scope of scopes) {
    const opts = { silent: true, ignoreReturnCode: true, ...(cwd ? { cwd } : {}) };
    const result = await exec.getExecOutput("git", ["config", scope, "--unset-all", key], opts);
    // Exit code 5 means the key was absent in this scope — expected and safe to ignore.
    // Any other non-zero exit code indicates a real failure (e.g., permission denied,
    // write lock). Throw so the caller knows the credential may still be effective.
    if (result.exitCode !== 0 && result.exitCode !== 5) {
      throw new Error(`git config ${scope} --unset-all ${key} failed (exit ${result.exitCode}): ${result.stderr.trim()}`);
    }
  }

  // Also clear the key from any includeIf-referenced config files (e.g. checkout v7's
  // per-checkout credentials file). These are not touched by --local/--global unsets
  // because git only resolves includes for reads, not for scope-targeted writes.
  let includedFiles;
  try {
    includedFiles = await findIncludedExtraheaderConfigFiles(cwd);
  } catch (err) {
    core.warning(`git_auth_helpers: could not enumerate includeIf-referenced config files: ${getErrorMessage(err)}`);
    includedFiles = [];
  }
  for (const file of includedFiles) {
    const result = await exec.getExecOutput("git", ["config", "--file", file, "--unset-all", key], { silent: true, ignoreReturnCode: true });
    if (result.exitCode !== 0 && result.exitCode !== 5) {
      throw new Error(`git config --file ${file} --unset-all ${key} failed (exit ${result.exitCode}): ${result.stderr.trim()}`);
    }
  }
}

/**
 * Get all configured values for http.<serverUrl>/.extraheader.
 * Throws if `exec.getExecOutput` itself throws (e.g. git not available).
 * Returns an empty array when the key is absent (exit code ≠ 0).
 *
 * @param {string} serverUrl
 * @param {string} [cwd] - Optional working directory for the git config command
 * @returns {Promise<string[]>}
 */
async function getExtraheaderValues(serverUrl, cwd) {
  const normalizedUrl = normalizeServerUrl(serverUrl);
  const execOptions = cwd ? { silent: true, ignoreReturnCode: true, cwd } : { silent: true, ignoreReturnCode: true };
  const result = await exec.getExecOutput("git", ["config", "--get-all", `http.${normalizedUrl}/.extraheader`], execOptions);
  if (result.exitCode !== 0 || !result.stdout.trim()) {
    return [];
  }
  return result.stdout
    .split("\n")
    .map(line => line.trim())
    .filter(Boolean);
}

/**
 * Determine whether checkout persisted an extraheader credential.
 * Returns false on any read error (safe default: assume no persisted credential).
 *
 * @param {string} serverUrl
 * @returns {Promise<boolean>}
 */
async function checkoutHasPersistedExtraheader(serverUrl) {
  try {
    const values = await getExtraheaderValues(serverUrl);
    return values.length > 0;
  } catch {
    return false;
  }
}

/**
 * Run `git config` with `silent: true` (to prevent credential-bearing command
 * lines from reaching stdout / uploaded safe-output artifacts) while still
 * capturing stderr so that real git diagnostics surface on failure.
 *
 * @param {string[]} gitArgs - Arguments passed to `git` (e.g. `["config", "--local", ...]`)
 * @param {string} [cwd] - Optional working directory
 * @returns {Promise<void>}
 */
async function gitExecSilent(gitArgs, cwd) {
  let stderrBuf = "";
  const listeners = {
    stderr: (/** @type {Buffer} */ data) => {
      stderrBuf += data.toString();
    },
  };
  try {
    await exec.exec("git", gitArgs, { silent: true, listeners, ...(cwd ? { cwd } : {}) });
  } catch (err) {
    throw new Error(`git-config-credential failed: ${stderrBuf.trim() || getErrorMessage(err)}`, { cause: err });
  }
}

/**
 * Replace any existing extraheader values with a single token-based Authorization
 * header and return the previous values for restoration.
 *
 * @param {string} serverUrl
 * @param {string} token
 * @param {string} [cwd] - Optional working directory for the git config command
 * @returns {Promise<string[]>}
 */
async function overridePersistedExtraheader(serverUrl, token, cwd) {
  const normalizedUrl = normalizeServerUrl(serverUrl);
  let previousValues;
  try {
    previousValues = await getExtraheaderValues(serverUrl, cwd);
    core.info(`git_auth_helpers: read ${previousValues.length} existing extraheader value(s) for ${normalizedUrl}`);
  } catch (err) {
    core.warning(`git_auth_helpers: could not read existing extraheader values — restoration will proceed with empty defaults: ${getErrorMessage(err)}`);
    previousValues = [];
  }
  core.info(`git_auth_helpers: overriding http.${normalizedUrl}/.extraheader with CI trigger token`);
  maskSecret(token);
  const tokenBase64 = Buffer.from(`x-access-token:${token.trim()}`).toString("base64");
  maskSecret(tokenBase64);
  const authHeader = `Authorization: basic ${tokenBase64}`;

  // Clear from ALL writable scopes before writing our token to prevent duplicate
  // Authorization headers. actions/checkout writes to the global scope; without
  // this step a plain --replace-all only writes to the local scope, leaving the
  // global value in place and causing duplicate-header HTTP 400 errors.
  await unsetExtraheaderAllScopes(`http.${normalizedUrl}/.extraheader`, cwd);

  await gitExecSilent(["config", "--local", "--replace-all", `http.${normalizedUrl}/.extraheader`, authHeader], cwd);
  core.info(`git_auth_helpers: extraheader override applied`);
  return previousValues;
}

/**
 * Restore a previously saved list of extraheader values.
 *
 * @param {string} serverUrl
 * @param {string[]} previousValues
 * @param {string} [cwd] - Optional working directory for the git config command
 * @returns {Promise<void>}
 */
async function restorePersistedExtraheader(serverUrl, previousValues, cwd) {
  const key = `http.${normalizeServerUrl(serverUrl)}/.extraheader`;

  // Clear from ALL writable scopes first. This removes the override written by
  // overridePersistedExtraheader and prevents values from accumulating across
  // multiple override/restore cycles (which happens when global scope values
  // are not cleared and get re-read by getExtraheaderValues on each attempt).
  await unsetExtraheaderAllScopes(key, cwd);

  if (!previousValues || previousValues.length === 0) {
    core.info(`git_auth_helpers: no previous extraheader values — ${key} cleared`);
    return;
  }

  core.info(`git_auth_helpers: restoring ${previousValues.length} previous extraheader value(s) for ${key}`);
  // Restore to the local scope. Even when the original values were written to
  // the global scope by actions/checkout, restoring to local is functionally
  // equivalent: with the global scope cleared the restored local value is the
  // only source, so no duplicate headers can arise on subsequent retries.
  // If any call fails after --replace-all has already run, git config is left
  // in a partially-restored state. In that case we log a warning and attempt a
  // best-effort cleanup, then re-throw so the caller is aware that restoration
  // failed.
  try {
    await gitExecSilent(["config", "--local", "--replace-all", key, previousValues[0]], cwd);
    for (const value of previousValues.slice(1)) {
      await gitExecSilent(["config", "--local", "--add", key, value], cwd);
    }
  } catch (err) {
    core.warning(`git_auth_helpers: partial extraheader restore for ${key} — attempting cleanup: ${getErrorMessage(err)}`);
    try {
      await unsetExtraheaderAllScopes(key, cwd);
    } catch {
      // Best-effort only; ignore secondary failure.
    }
    throw err;
  }
  core.info(`git_auth_helpers: extraheader restored`);
}

/**
 * Temporarily override the persisted GitHub extraheader for remote git operations.
 *
 * Saves the current extraheader value(s), replaces them with the fork token for the
 * duration of the callback, then restores the original value(s). This ensures that
 * only one Authorization source is active at a time, preventing duplicate-header HTTP 400s
 * when a fork token and the checkout-persisted upstream token both apply to the same host.
 *
 * @template T
 * @param {string} token
 * @param {() => Promise<T>} callback
 * @param {string} [cwd] - Optional working directory; scopes the git config override to the correct checkout
 * @returns {Promise<T>}
 */
async function withGitHubHostToken(token, callback, cwd) {
  if (!token) {
    return callback();
  }
  const githubServerUrl = (process.env.GITHUB_SERVER_URL || "https://github.com").replace(/\/+$/, "");
  let previousExtraheaders = [];
  let overrideApplied = false;
  try {
    previousExtraheaders = await overridePersistedExtraheader(githubServerUrl, token, cwd);
    overrideApplied = true;
    return await callback();
  } finally {
    if (overrideApplied) {
      await restorePersistedExtraheader(githubServerUrl, previousExtraheaders, cwd);
    }
  }
}

module.exports = {
  checkoutHasPersistedExtraheader,
  findIncludedExtraheaderConfigFiles,
  getGitAuthEnv,
  gitExecSilent,
  overridePersistedExtraheader,
  restorePersistedExtraheader,
  unsetExtraheaderAllScopes,
  withGitHubHostToken,
};
