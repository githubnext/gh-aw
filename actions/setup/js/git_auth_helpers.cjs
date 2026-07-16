// @ts-check
/// <reference types="@actions/github-script" />
// This module relies on the `exec` and `core` globals injected by github-script at runtime.
// All callers must ensure these globals are set before invoking any helper.

const { getErrorMessage } = require("./error_helpers.cjs");

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
  const tokenBase64 = Buffer.from(`x-access-token:${token.trim()}`).toString("base64");
  if (cwd) {
    await exec.exec("git", ["config", "--replace-all", `http.${normalizedUrl}/.extraheader`, `Authorization: basic ${tokenBase64}`], { cwd });
  } else {
    await exec.exec("git", ["config", "--replace-all", `http.${normalizedUrl}/.extraheader`, `Authorization: basic ${tokenBase64}`]);
  }
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
  if (!previousValues || previousValues.length === 0) {
    core.info(`git_auth_helpers: no previous extraheader values — unsetting ${key}`);
    try {
      if (cwd) {
        await exec.exec("git", ["config", "--unset-all", key], { cwd });
      } else {
        await exec.exec("git", ["config", "--unset-all", key]);
      }
    } catch {
      // Nothing to restore/unset.
    }
    return;
  }

  core.info(`git_auth_helpers: restoring ${previousValues.length} previous extraheader value(s) for ${key}`);
  // --replace-all removes any existing values for the key (including the CI-token
  // entry) and writes previousValues[0]. Subsequent --add calls stack the remaining
  // previous values onto the same key without removing any already written.
  // If any --add call fails after --replace-all has already run, git config is left
  // in a partially-restored state. In that case we log a warning and attempt a
  // best-effort cleanup by unsetting the key entirely, then re-throw so the caller
  // is aware that restoration failed.
  try {
    if (cwd) {
      await exec.exec("git", ["config", "--replace-all", key, previousValues[0]], { cwd });
      for (const value of previousValues.slice(1)) {
        await exec.exec("git", ["config", "--add", key, value], { cwd });
      }
    } else {
      await exec.exec("git", ["config", "--replace-all", key, previousValues[0]]);
      for (const value of previousValues.slice(1)) {
        await exec.exec("git", ["config", "--add", key, value]);
      }
    }
  } catch (err) {
    core.warning(`git_auth_helpers: partial extraheader restore for ${key} — attempting cleanup: ${getErrorMessage(err)}`);
    try {
      if (cwd) {
        await exec.exec("git", ["config", "--unset-all", key], { cwd });
      } else {
        await exec.exec("git", ["config", "--unset-all", key]);
      }
    } catch {
      // Best-effort only; ignore secondary failure.
    }
    throw err;
  }
  core.info(`git_auth_helpers: extraheader restored`);
}

module.exports = {
  checkoutHasPersistedExtraheader,
  overridePersistedExtraheader,
  restorePersistedExtraheader,
};
