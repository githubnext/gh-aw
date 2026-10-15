// @ts-check
"use strict";

function expectedGitHubHost() {
  try {
    const serverUrl = process.env.GITHUB_SERVER_URL || "https://github.com";
    const server = new URL(serverUrl);
    const authority = serverUrl.match(/^https:\/\/([^/?#]*)/i)?.[1];
    if (
      server.protocol !== "https:" ||
      server.username ||
      server.password ||
      server.port ||
      server.pathname !== "/" ||
      server.search ||
      server.hash ||
      serverUrl.includes("?") ||
      serverUrl.includes("#") ||
      !authority ||
      authority.toLowerCase() !== server.hostname
    ) {
      throw new Error("invalid server URL");
    }
    return server.hostname;
  } catch (cause) {
    throw new Error("Work queue requires a valid GITHUB_SERVER_URL", { cause });
  }
}

function isTrustedGitHubApiUrl() {
  const apiUrl = process.env.GITHUB_API_URL || "https://api.github.com";
  const host = expectedGitHubHost();
  let api;
  try {
    api = new URL(apiUrl);
  } catch (cause) {
    throw new Error("Work queue requires a valid GITHUB_API_URL", { cause });
  }
  const authority = apiUrl.match(/^https:\/\/([^/?#]*)/i)?.[1];
  if (api.protocol !== "https:" || api.username || api.password || api.port || api.search || api.hash || apiUrl.includes("?") || apiUrl.includes("#") || !authority || authority.toLowerCase() !== api.hostname) {
    throw new Error("Work queue requires a valid GITHUB_API_URL");
  }
  return api.hostname === host || api.hostname === `api.${host}`;
}

module.exports = { expectedGitHubHost, isTrustedGitHubApiUrl };
