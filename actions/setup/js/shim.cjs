// @ts-check

/**
 * shim.cjs
 *
 * Provides `github-script`-style globals so modules written for that runtime
 * still work when executed as plain Node.js processes, such as inside the
 * safe-outputs and mcp-scripts MCP servers.
 *
 * When a global is already set (i.e. running inside `github-script`) the
 * respective block is a no-op.
 */

const fs = require("fs");
const os = require("os");
const path = require("path");

/**
 * Write shim log lines to stderr so MCP servers that speak JSON-RPC on stdout
 * never interleave protocol frames with diagnostic output.
 * @param {string} level
 * @param {unknown} message
 */
const writeShimLog = (level, message) => {
  process.stderr.write(`[${level}] ${String(message)}\n`);
};

let shimCommandFilesDir = null;

function ensureShimCommandFilesDir() {
  if (shimCommandFilesDir) {
    return shimCommandFilesDir;
  }
  const baseDir = process.env.RUNNER_TEMP || os.tmpdir();
  shimCommandFilesDir = fs.mkdtempSync(path.join(baseDir, "gh-aw-shim-"));
  return shimCommandFilesDir;
}

/**
 * Ensure @actions/core writes file commands to temp files instead of stdout
 * when running outside GitHub Actions.
 * @param {string} envName
 * @param {string} fileName
 */
function ensureFileCommandPath(envName, fileName) {
  if (process.env[envName]) {
    return;
  }
  const filePath = path.join(ensureShimCommandFilesDir(), fileName);
  fs.closeSync(fs.openSync(filePath, "a"));
  process.env[envName] = filePath;
}

function ensureFileCommandPaths() {
  ensureFileCommandPath("GITHUB_ENV", "github-env.txt");
  ensureFileCommandPath("GITHUB_OUTPUT", "github-output.txt");
  ensureFileCommandPath("GITHUB_PATH", "github-path.txt");
  ensureFileCommandPath("GITHUB_STATE", "github-state.txt");
  ensureFileCommandPath("GITHUB_STEP_SUMMARY", "github-step-summary.md");
}

/**
 * @param {string} filePath
 * @param {string} content
 */
function appendFileSafely(filePath, content) {
  try {
    fs.appendFileSync(filePath, content);
  } catch (error) {
    throw new Error(`Failed to append to ${filePath}: ${String(error)}`, { cause: error });
  }
}

/**
 * @param {string} filePath
 * @param {string} content
 */
function writeFileSafely(filePath, content) {
  try {
    fs.writeFileSync(filePath, content);
  } catch (error) {
    throw new Error(`Failed to write ${filePath}: ${String(error)}`, { cause: error });
  }
}

/**
 * @param {string} name
 * @returns {{ number?: number }}
 */
function buildIssueContext() {
  const payloadNumber =
    (typeof global.context?.payload?.issue?.number === "number" && global.context.payload.issue.number) || (typeof global.context?.payload?.pull_request?.number === "number" && global.context.payload.pull_request.number);
  return payloadNumber ? { number: payloadNumber } : {};
}

/**
 * @param {string} envName
 * @param {string} name
 * @param {unknown} value
 */
function appendFileCommand(envName, name, value) {
  ensureFileCommandPath(envName, envName.toLowerCase() + ".txt");
  const filePath = process.env[envName];
  const serialized = String(value ?? "");
  const delimiter = `gh-aw-${Date.now()}-${Math.random().toString(36).slice(2)}`;
  appendFileSafely(filePath, `${name}<<${delimiter}\n${serialized}\n${delimiter}\n`);
}

function createSummaryShim() {
  let buffer = "";
  return {
    addRaw(text, addEol = false) {
      buffer += String(text);
      if (addEol) {
        buffer += "\n";
      }
      return this;
    },
    addHeading(text, level = 1) {
      buffer += `${"#".repeat(Math.max(1, level))} ${text}\n\n`;
      return this;
    },
    addDetails(summary, details) {
      buffer += `<details><summary>${summary}</summary>${details}</details>\n\n`;
      return this;
    },
    async write(options = {}) {
      ensureFileCommandPath("GITHUB_STEP_SUMMARY", "github-step-summary.md");
      const summaryPath = process.env.GITHUB_STEP_SUMMARY;
      if (options.overwrite) {
        writeFileSafely(summaryPath, buffer);
      } else {
        appendFileSafely(summaryPath, buffer);
      }
      buffer = "";
      return this;
    },
  };
}

/**
 * @template T
 * @param {() => Promise<T>} loadTarget
 * @returns {T}
 */
function createLazyObjectProxy(loadTarget) {
  const makeProxy = pathParts =>
    new Proxy(function proxyTarget() {}, {
      get(_target, property) {
        if (property === "then") {
          return undefined;
        }
        return makeProxy([...pathParts, property]);
      },
      apply(_target, _thisArg, args) {
        return (async () => {
          const root = await loadTarget();
          if (pathParts.length === 0) {
            if (typeof root !== "function") {
              return root;
            }
            return root(...args);
          }
          let parent = root;
          for (let i = 0; i < pathParts.length - 1; i += 1) {
            parent = parent[pathParts[i]];
          }
          const leaf = parent[pathParts[pathParts.length - 1]];
          if (typeof leaf !== "function") {
            return leaf;
          }
          return leaf.apply(parent, args);
        })();
      },
    });

  return /** @type {T} */ makeProxy([]);
}

if (!global.core) {
  global.core = {
    debug: /** @param {unknown} message */ message => writeShimLog("debug", message),
    info: /** @param {unknown} message */ message => writeShimLog("info", message),
    notice: /** @param {unknown} message */ message => writeShimLog("notice", message),
    warning: /** @param {unknown} message */ message => writeShimLog("warning", message),
    error: /** @param {unknown} message */ message => writeShimLog("error", message),
    startGroup: /** @param {string} name */ name => writeShimLog("group", `START ${name}`),
    endGroup: () => writeShimLog("group", "END"),
    group: async (name, fn) => {
      global.core.startGroup(name);
      try {
        return await fn();
      } finally {
        global.core.endGroup();
      }
    },
    setFailed: /** @param {unknown} message */ message => {
      writeShimLog("error", message);
      if (typeof process !== "undefined") {
        if (process.exitCode === null || process.exitCode === undefined || process.exitCode === 0) {
          process.exitCode = 1;
        }
      }
    },
    setOutput: /** @param {string} name @param {unknown} value */ (name, value) => appendFileCommand("GITHUB_OUTPUT", name, value),
    exportVariable: /** @param {string} name @param {unknown} value */ (name, value) => {
      process.env[name] = String(value ?? "");
      appendFileCommand("GITHUB_ENV", name, value);
    },
    addPath: /** @param {string} newPath */ newPath => {
      process.env.PATH = process.env.PATH ? `${newPath}${path.delimiter}${process.env.PATH}` : String(newPath);
      ensureFileCommandPath("GITHUB_PATH", "github-path.txt");
      appendFileSafely(process.env.GITHUB_PATH, `${newPath}\n`);
    },
    setSecret: () => {},
    summary: createSummaryShim(),
  };
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
    refName: process.env.GITHUB_REF_NAME || "",
    baseRef: process.env.GITHUB_BASE_REF || "",
    headRef: process.env.GITHUB_HEAD_REF || "",
    workflow: process.env.GITHUB_WORKFLOW || "",
    action: process.env.GITHUB_ACTION || "",
    actionPath: process.env.GITHUB_ACTION_PATH || "",
    actor: process.env.GITHUB_ACTOR || "",
    job: process.env.GITHUB_JOB || "",
    runNumber: parseInt(process.env.GITHUB_RUN_NUMBER || "0", 10),
    runId: parseInt(process.env.GITHUB_RUN_ID || "0", 10),
    apiUrl: process.env.GITHUB_API_URL || "https://api.github.com",
    serverUrl: process.env.GITHUB_SERVER_URL || "https://github.com",
    graphqlUrl: process.env.GITHUB_GRAPHQL_URL || "https://api.github.com/graphql",
    workspace: process.env.GITHUB_WORKSPACE || "",
    payload,
    repo: { owner, repo },
  };
  global.context.issue = buildIssueContext();
}

if (!global.getOctokit || !global.github || !global.octokit) {
  const getOctokitWithDefaults =
    global.getOctokit ||
    ((token, options = {}) =>
      createLazyObjectProxy(async () => {
        if (typeof token === "string" && token.startsWith("gho_")) {
          throw new Error("OAuth token (gho_...) detected. OAuth tokens are not suitable for automation: " + "replace the token with a fine-grained Personal Access Token.");
        }
        const githubModule = await import("@actions/github");
        return githubModule.getOctokit(token || "", {
          ...options,
          headers: {
            "X-GitHub-Api-Version": "2022-11-28",
            ...(options.headers || {}),
          },
        });
      }));

  global.getOctokit = getOctokitWithDefaults;

  if (!global.github) {
    const defaultToken = process.env.GITHUB_TOKEN || process.env.GH_TOKEN || "";
    global.github = getOctokitWithDefaults(defaultToken);
  }

  if (!global.octokit) {
    global.octokit = global.github;
  }
}

if (!global.exec) {
  global.exec = createLazyObjectProxy(() => import("@actions/exec"));
}

if (!global.glob) {
  global.glob = createLazyObjectProxy(() => import("@actions/glob"));
}

if (!global.io) {
  global.io = createLazyObjectProxy(() => import("@actions/io"));
}

if (!global.__original_require__) {
  global.__original_require__ = require;
}
