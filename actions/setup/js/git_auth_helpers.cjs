// @ts-check
/// <reference types="@actions/github-script" />
// This module relies on the `exec` and `core` globals injected by github-script at runtime.
// All callers must ensure these globals are set before invoking any helper.

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
 *
 * @param {string} serverUrl
 * @returns {Promise<string[]>}
 */
async function getExtraheaderValues(serverUrl) {
  const normalizedUrl = normalizeServerUrl(serverUrl);
  try {
    const result = await exec.getExecOutput("git", ["config", "--get-all", `http.${normalizedUrl}/.extraheader`], {
      silent: true,
      ignoreReturnCode: true,
    });
    if (result.exitCode !== 0 || !result.stdout.trim()) {
      return [];
    }
    return result.stdout
      .split("\n")
      .map(line => line.trim())
      .filter(Boolean);
  } catch {
    return [];
  }
}

/**
 * Determine whether checkout persisted an extraheader credential.
 *
 * @param {string} serverUrl
 * @returns {Promise<boolean>}
 */
async function checkoutHasPersistedExtraheader(serverUrl) {
  const values = await getExtraheaderValues(serverUrl);
  return values.length > 0;
}

/**
 * Replace any existing extraheader values with a single token-based Authorization
 * header and return the previous values for restoration.
 *
 * @param {string} serverUrl
 * @param {string} token
 * @returns {Promise<string[]>}
 */
async function overridePersistedExtraheader(serverUrl, token) {
  const normalizedUrl = normalizeServerUrl(serverUrl);
  let previousValues;
  try {
    previousValues = await getExtraheaderValues(normalizedUrl);
  } catch (err) {
    core.warning(`git_auth_helpers: could not read existing extraheader — previous values will not be restored: ${err.message}`);
    previousValues = [];
  }
  const tokenBase64 = Buffer.from(`x-access-token:${token.trim()}`).toString("base64");
  await exec.exec("git", ["config", "--replace-all", `http.${normalizedUrl}/.extraheader`, `Authorization: basic ${tokenBase64}`], { silent: true });
  return previousValues;
}

/**
 * Restore a previously saved list of extraheader values.
 *
 * @param {string} serverUrl
 * @param {string[]} previousValues
 * @returns {Promise<void>}
 */
async function restorePersistedExtraheader(serverUrl, previousValues) {
  const key = `http.${normalizeServerUrl(serverUrl)}/.extraheader`;
  if (!previousValues || previousValues.length === 0) {
    try {
      await exec.exec("git", ["config", "--unset-all", key]);
    } catch {
      // Nothing to restore/unset.
    }
    return;
  }

  // --replace-all removes any existing values for the key (including the CI-token
  // entry) and writes previousValues[0]. Subsequent --add calls stack the remaining
  // previous values onto the same key without removing any already written.
  await exec.exec("git", ["config", "--replace-all", key, previousValues[0]]);
  for (const value of previousValues.slice(1)) {
    await exec.exec("git", ["config", "--add", key, value]);
  }
}

module.exports = {
  checkoutHasPersistedExtraheader,
  overridePersistedExtraheader,
  restorePersistedExtraheader,
};
