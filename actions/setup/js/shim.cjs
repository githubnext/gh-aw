// @ts-check

/**
 * shim.cjs
 *
 * Provides minimal `global.core` and `global.context` shims so that modules
 * written for the GitHub Actions `github-script` context (which rely on the
 * built-in `core` and `context` globals) work correctly when executed as plain
 * Node.js processes, such as inside the safe-outputs and safe-inputs MCP servers.
 *
 * When `global.core` / `global.context` is already set (i.e. running inside
 * `github-script`) the respective block is a no-op.
 */

// @ts-expect-error - global.core is not declared in TypeScript but is provided by github-script
if (!global.core) {
  // @ts-expect-error - Assigning to global properties that are declared as const
  global.core = {
    debug: /** @param {string} message */ message => console.debug(`[debug] ${message}`),
    info: /** @param {string} message */ message => console.info(`[info] ${message}`),
    notice: /** @param {string} message */ message => console.info(`[notice] ${message}`),
    warning: /** @param {string} message */ message => console.warn(`[warning] ${message}`),
    error: /** @param {string} message */ message => console.error(`[error] ${message}`),
    setFailed: /** @param {string} message */ message => {
      console.error(`[error] ${message}`);
      if (typeof process !== "undefined") {
        if (process.exitCode === null || process.exitCode === undefined || process.exitCode === 0) {
          process.exitCode = 1;
        }
      }
    },
    setOutput: /** @param {string} name @param {unknown} value */ (name, value) => {
      console.info(`[output] ${name}=${value}`);
    },
  };
}

// @ts-expect-error - global.context is not declared in TypeScript but is provided by github-script
if (!global.context) {
  // Build a context object from GitHub Actions environment variables,
  // mirroring the shape of @actions/github's Context class.
  /** @type {Record<string, unknown>} */
  let payload = {};
  const eventPath = process.env.GITHUB_EVENT_PATH;
  if (eventPath) {
    try {
      const fs = require("fs");
      payload = JSON.parse(fs.readFileSync(eventPath, "utf8"));
    } catch {
      // Ignore errors reading the event payload – it may not be present when
      // the MCP server is started outside of a GitHub Actions runner.
    }
  }

  const repository = process.env.GITHUB_REPOSITORY || "";
  const slashIdx = repository.indexOf("/");
  // When GITHUB_REPOSITORY is absent or lacks a '/' separator, both fields
  // fall back to empty strings so callers can detect the missing value.
  const owner = slashIdx >= 0 ? repository.slice(0, slashIdx) : "";
  const repo = slashIdx >= 0 ? repository.slice(slashIdx + 1) : "";

  // @ts-expect-error - Assigning to global properties that are declared as const
  global.context = {
    eventName: process.env.GITHUB_EVENT_NAME || "",
    sha: process.env.GITHUB_SHA || "",
    ref: process.env.GITHUB_REF || "",
    workflow: process.env.GITHUB_WORKFLOW || "",
    action: process.env.GITHUB_ACTION || "",
    actor: process.env.GITHUB_ACTOR || "",
    job: process.env.GITHUB_JOB || "",
    runNumber: parseInt(process.env.GITHUB_RUN_NUMBER || "0", 10),
    runId: parseInt(process.env.GITHUB_RUN_ID || "0", 10),
    apiUrl: process.env.GITHUB_API_URL || "https://api.github.com",
    serverUrl: process.env.GITHUB_SERVER_URL || "https://github.com",
    graphqlUrl: process.env.GITHUB_GRAPHQL_URL || "https://api.github.com/graphql",
    payload,
    repo: { owner, repo },
  };
}
