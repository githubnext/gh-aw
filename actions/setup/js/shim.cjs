// @ts-check

/**
 * shim.cjs
 *
 * Provides minimal `global.core` and `global.context` shims so that modules
 * written for the GitHub Actions `github-script` context (which rely on the
 * built-in `core` and `context` globals) work correctly when executed as plain
 * Node.js processes, such as inside the safe-outputs and mcp-scripts MCP servers.
 *
 * When `global.core` / `global.context` is already set (i.e. running inside
 * `github-script`) the respective block is a no-op.
 */

const secrets = new Set();

const maskSecrets = value => {
  const text = String(value);
  const ranges = [];
  for (const secret of secrets) {
    for (let start = text.indexOf(secret); start !== -1; start = text.indexOf(secret, start + 1)) {
      ranges.push([start, start + secret.length]);
    }
  }

  if (ranges.length === 0) {
    return text;
  }

  ranges.sort((left, right) => left[0] - right[0]);
  let masked = "";
  let cursor = 0;
  let [rangeStart, rangeEnd] = ranges[0];
  for (const [start, end] of ranges.slice(1)) {
    if (start <= rangeEnd) {
      rangeEnd = Math.max(rangeEnd, end);
      continue;
    }
    masked += `${text.slice(cursor, rangeStart)}***`;
    cursor = rangeEnd;
    rangeStart = start;
    rangeEnd = end;
  }

  masked += `${text.slice(cursor, rangeStart)}***${text.slice(rangeEnd)}`;
  return masked;
};

const setSecret = /** @param {string} value */ value => {
  const secret = String(value);
  if (secret.length === 0) {
    return;
  }

  // Do not emit an add-mask workflow command here: that would write the secret to stderr.
  secrets.add(secret);
};

if (!global.core) {
  /**
   * Write shim log lines to stderr so MCP servers that speak JSON-RPC on stdout
   * never interleave protocol frames with diagnostic output.
   * @param {string} level
   * @param {string} message
   */
  const writeShimLog = (level, message) => {
    process.stderr.write(`[${level}] ${maskSecrets(message)}\n`);
  };

  global.core = {
    debug: /** @param {string} message */ message => writeShimLog("debug", message),
    info: /** @param {string} message */ message => writeShimLog("info", message),
    notice: /** @param {string} message */ message => writeShimLog("notice", message),
    warning: /** @param {string} message */ message => writeShimLog("warning", message),
    error: /** @param {string} message */ message => writeShimLog("error", message),
    setFailed: /** @param {string} message */ message => {
      writeShimLog("error", message);
      if (typeof process !== "undefined") {
        if (process.exitCode === null || process.exitCode === undefined || process.exitCode === 0) {
          process.exitCode = 1;
        }
      }
    },
    setOutput: /** @param {string} name @param {unknown} value */ (name, value) => {
      writeShimLog("output", `${name}=${value}`);
    },
    setSecret,
  };
} else if (typeof global.core.setSecret !== "function") {
  global.core.setSecret = setSecret;
}

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
