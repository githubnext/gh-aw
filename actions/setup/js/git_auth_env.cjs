// @ts-check

/**
 * Build GIT_CONFIG_* environment variables that inject an Authorization header
 * for git network operations without writing credentials to disk.
 *
 * This helper intentionally does not call core.setSecret so it is safe to use
 * from MCP server processes.
 *
 * @param {string} authToken
 * @param {(value: string) => void} [setSecret]
 * @returns {Object}
 */
function buildGitAuthEnv(authToken, setSecret) {
  const serverUrl = (process.env.GITHUB_SERVER_URL || "https://github.com").replace(/\/$/, "");
  const tokenBase64 = Buffer.from(`x-access-token:${authToken}`).toString("base64");
  setSecret?.(tokenBase64);
  return {
    GIT_CONFIG_COUNT: "1",
    GIT_CONFIG_KEY_0: `http.${serverUrl}/.extraheader`,
    GIT_CONFIG_VALUE_0: `Authorization: basic ${tokenBase64}`,
  };
}

module.exports = { buildGitAuthEnv };
