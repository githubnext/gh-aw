// @ts-check
/// <reference types="@actions/github-script" />

/**
 * Get all configured values for http.<serverUrl>/.extraheader.
 *
 * @param {string} serverUrl
 * @returns {Promise<string[]>}
 */
async function getExtraheaderValues(serverUrl) {
  try {
    const result = await exec.getExecOutput("git", ["config", "--get-all", `http.${serverUrl}/.extraheader`], {
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
  const previousValues = await getExtraheaderValues(serverUrl);
  const tokenBase64 = Buffer.from(`x-access-token:${token}`).toString("base64");
  await exec.exec("git", ["config", "--replace-all", `http.${serverUrl}/.extraheader`, `Authorization: basic ${tokenBase64}`]);
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
  const key = `http.${serverUrl}/.extraheader`;
  if (!previousValues || previousValues.length === 0) {
    try {
      await exec.exec("git", ["config", "--unset-all", key]);
    } catch {
      // Nothing to restore/unset.
    }
    return;
  }

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
