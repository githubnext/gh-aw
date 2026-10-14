// @ts-check
"use strict";

function expectedGitHubHost() {
  try {
    return new URL(process.env.GITHUB_SERVER_URL || "https://github.com").hostname;
  } catch (cause) {
    throw new Error("Work queue requires a valid GITHUB_SERVER_URL", { cause });
  }
}

module.exports = { expectedGitHubHost };
