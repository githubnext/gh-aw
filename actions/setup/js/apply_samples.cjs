#!/usr/bin/env node
// @ts-check
/// <reference types="@actions/github-script" />

// apply_samples.cjs
//
// Deterministic replay driver for `gh aw compile --use-samples`.
//
// Reads `GH_AW_SAMPLES` (a JSON array of `{tool, arguments, sidecars}`
// entries produced by the compiler), spawns the safe-outputs MCP server
// (`safe_outputs_mcp_server.cjs`) as a child process, sends one JSON-RPC
// `tools/call` per sample over stdio, and writes a synthetic `agent-stdio.log`
// so downstream log-parsing / failure-handling steps continue to work.
//
// For samples whose tool is `create_pull_request` or `push_to_pull_request_branch`
// and whose sidecars include `patch`, the driver pre-stages a branch and commits
// the patch into the workspace BEFORE invoking the MCP tool. This lets the
// real `create_pull_request` MCP handler (which derives a git diff against the
// base branch) produce a meaningful transport payload.
//
// Env contract:
//   GH_AW_SAMPLES        — JSON array of replay entries (required)
//   GH_AW_AGENT_STDIO_LOG     — path where the synthetic stdio log is written
//   GH_AW_SAFE_OUTPUTS_CONFIG_PATH — path to the MCP server's config.json
//   GH_AW_SAFE_OUTPUTS        — path to the MCP server's outputs.jsonl
//   GITHUB_WORKSPACE          — git working directory for pre-staging (optional;
//                               falls back to cwd)

require("./shim.cjs");

const { spawn } = require("child_process");
const fs = require("fs");
const path = require("path");
const os = require("os");
const { getErrorMessage } = require("./error_helpers.cjs");
const { ERR_VALIDATION, ERR_PARSE, ERR_SYSTEM, ERR_API, ERR_CONFIG } = require("./error_codes.cjs");

const DEFAULT_BASE_BRANCH = process.env.GH_AW_CUSTOM_BASE_BRANCH || process.env.GITHUB_BASE_REF || process.env.GITHUB_REF_NAME || "main";
const PATCH_SIDECAR_TOOLS = new Set(["create_pull_request", "push_to_pull_request_branch"]);

/**
 * @typedef {Object} SampleEntry
 * @property {string} tool
 * @property {Record<string, any>} arguments
 * @property {Record<string, any>} [sidecars]
 */

/**
 * Read and parse the GH_AW_SAMPLES env var. Returns an empty array (with a
 * warning) when unset or empty so the workflow can still complete cleanly.
 * @returns {SampleEntry[]}
 */
function loadSamples() {
  const raw = process.env.GH_AW_SAMPLES;
  if (!raw || !raw.trim()) {
    core.warning("apply_samples: GH_AW_SAMPLES is empty — no samples to replay.");
    return [];
  }
  let parsed;
  try {
    parsed = JSON.parse(raw);
  } catch (err) {
    throw new Error(`${ERR_PARSE}: apply_samples: failed to parse GH_AW_SAMPLES as JSON: ${getErrorMessage(err)}`);
  }
  // Tolerate a literal JSON `null` payload (older compiler emitted it for
  // workflows with --use-samples but no `samples:` entries). Treat as empty.
  if (parsed === null) {
    core.warning("apply_samples: GH_AW_SAMPLES is null — treating as no samples to replay.");
    return [];
  }
  if (!Array.isArray(parsed)) {
    throw new Error(`${ERR_VALIDATION}: apply_samples: GH_AW_SAMPLES must be a JSON array`);
  }
  for (const [i, entry] of parsed.entries()) {
    if (!entry || typeof entry !== "object" || typeof entry.tool !== "string") {
      throw new Error(`${ERR_VALIDATION}: apply_samples: entry ${i} is missing a string "tool" field`);
    }
    if (!entry.arguments || typeof entry.arguments !== "object") {
      throw new Error(`${ERR_VALIDATION}: apply_samples: entry ${i} (tool=${entry.tool}) is missing an "arguments" object`);
    }
  }
  return parsed;
}

/**
 * Run a git subcommand synchronously and return stdout. Throws on non-zero exit.
 * @param {string[]} args
 * @param {string} cwd
 * @returns {string}
 */
function runGit(args, cwd) {
  const { spawnSync } = require("child_process");
  const result = spawnSync("git", args, { cwd, encoding: "utf8" });
  if (result.status !== 0) {
    throw new Error(`${ERR_SYSTEM}: git ${args.join(" ")} failed (exit ${result.status}): ${result.stderr || result.stdout}`);
  }
  return result.stdout;
}

/**
 * Ensure git user.email / user.name are configured so commits succeed in CI.
 * @param {string} cwd
 */
function ensureGitIdentity(cwd) {
  try {
    runGit(["config", "user.email"], cwd);
  } catch {
    runGit(["config", "user.email", "gh-aw-samples@github.com"], cwd);
  }
  try {
    runGit(["config", "user.name"], cwd);
  } catch {
    runGit(["config", "user.name", "gh-aw samples"], cwd);
  }
}

/**
 * Pre-stage a branch + patch for samples whose tool reads the workspace diff.
 * Mutates `entry.arguments.branch` to the actual checked-out branch.
 * @param {SampleEntry} entry
 * @param {number} index
 * @param {string} workspace
 */
function preStagePatch(entry, index, workspace) {
  const patch = entry.sidecars && entry.sidecars.patch;
  if (typeof patch !== "string" || !patch.trim()) {
    return;
  }
  const branch = typeof entry.arguments.branch === "string" && entry.arguments.branch.trim() ? entry.arguments.branch.trim() : `gh-aw-sample-${index + 1}`;
  entry.arguments.branch = branch;

  ensureGitIdentity(workspace);

  // Start from the base branch so the diff is meaningful. Tolerate the case
  // where the base ref doesn't exist locally — fall back to HEAD.
  try {
    runGit(["checkout", DEFAULT_BASE_BRANCH], workspace);
  } catch (err) {
    core.warning(`apply_samples: could not check out base branch ${DEFAULT_BASE_BRANCH}: ${getErrorMessage(err)}; staying on current HEAD`);
  }

  // Create the branch (or check it out if it already exists from a previous sample).
  try {
    runGit(["checkout", "-b", branch], workspace);
  } catch {
    runGit(["checkout", branch], workspace);
  }

  // Write patch to a temp file and apply it.
  const tmpPatch = path.join(os.tmpdir(), `gh-aw-sample-${index + 1}.patch`);
  fs.writeFileSync(tmpPatch, patch.endsWith("\n") ? patch : patch + "\n");
  try {
    runGit(["apply", "--whitespace=nowarn", tmpPatch], workspace);
  } catch (err) {
    // Fall back to --3way for patches that don't apply cleanly on top of an
    // empty working tree (uncommon but possible for synthetic samples).
    runGit(["apply", "--3way", "--whitespace=nowarn", tmpPatch], workspace);
  }

  runGit(["add", "-A"], workspace);
  runGit(["commit", "-m", `gh-aw sample ${index + 1}: ${entry.tool}`, "--allow-empty"], workspace);
}

/**
 * Send a single JSON-RPC request to the MCP server child process and resolve
 * with the parsed JSON response (or reject on timeout).
 * @param {import("child_process").ChildProcess} child
 * @param {NodeJS.WritableStream} stdin
 * @param {object} request
 * @param {AsyncIterableIterator<string>} responseIterator
 * @returns {Promise<any>}
 */
async function sendJsonRpc(child, stdin, request, responseIterator) {
  stdin.write(JSON.stringify(request) + "\n");
  while (true) {
    const { value, done } = await responseIterator.next();
    if (done) {
      throw new Error(`${ERR_API}: apply_samples: MCP server closed stdout before responding to request id=${request.id}`);
    }
    const line = typeof value === "string" ? value.trim() : "";
    if (!line) {
      continue;
    }
    if (!line.startsWith("{")) {
      core.debug(`apply_samples: ignoring non-JSON stdout line: ${line}`);
      continue;
    }
    try {
      return JSON.parse(line);
    } catch (err) {
      throw new Error(`${ERR_PARSE}: apply_samples: failed to parse MCP JSON-RPC response for request id=${request.id}: ${getErrorMessage(err)} (line: ${line})`);
    }
  }
}

/**
 * Turn the MCP server's stdout into an async iterator of line strings.
 * @param {NodeJS.ReadableStream} stdout
 */
async function* lineIterator(stdout) {
  let buffer = "";
  for await (const chunk of stdout) {
    buffer += chunk.toString();
    let newlineIdx;
    while ((newlineIdx = buffer.indexOf("\n")) !== -1) {
      const line = buffer.slice(0, newlineIdx).trim();
      buffer = buffer.slice(newlineIdx + 1);
      if (line) {
        yield line;
      }
    }
  }
  if (buffer.trim()) {
    yield buffer.trim();
  }
}

/**
 * Locate the safe_outputs_mcp_server.cjs script. The setup action copies it
 * into ${RUNNER_TEMP}/gh-aw/actions/ alongside this driver; fall back to
 * resolving via __dirname for local-execution / tests.
 * @returns {string}
 */
function resolveMcpServerPath() {
  const candidates = [
    path.join(__dirname, "safe_outputs_mcp_server.cjs"),
    process.env.RUNNER_TEMP ? path.join(process.env.RUNNER_TEMP, "gh-aw", "actions", "safe_outputs_mcp_server.cjs") : null,
    process.env.RUNNER_TEMP ? path.join(process.env.RUNNER_TEMP, "gh-aw", "safeoutputs", "safe_outputs_mcp_server.cjs") : null,
  ].filter(/** @returns {p is string} */ p => typeof p === "string");
  for (const candidate of candidates) {
    if (fs.existsSync(candidate)) {
      return candidate;
    }
  }
  throw new Error(`${ERR_CONFIG}: apply_samples: could not locate safe_outputs_mcp_server.cjs. Looked in: ${candidates.join(", ")}`);
}

/**
 * Append a synthetic terminal_reason: completed marker to the engine stdio log
 * so downstream parsers / handle_agent_failure recognize the replay as a
 * successful agent run.
 * @param {string} logPath
 * @param {number} sampleCount
 */
function writeSyntheticStdioLog(logPath, sampleCount) {
  if (!logPath) return;
  try {
    fs.mkdirSync(path.dirname(logPath), { recursive: true });
  } catch {
    /* ignore */
  }
  const lines = [
    `gh-aw samples replay: ${sampleCount} MCP tools/call invocation(s) completed deterministically.`,
    JSON.stringify({
      type: "result",
      subtype: "success",
      terminal_reason: "completed",
      num_turns: sampleCount,
      driver: "apply_samples",
    }),
    "",
  ];
  fs.appendFileSync(logPath, lines.join("\n"));
}

async function main() {
  const samples = loadSamples();
  const workspace = process.env.GITHUB_WORKSPACE || process.cwd();
  const logPath = process.env.GH_AW_AGENT_STDIO_LOG || "";

  // Pre-stage branches/patches.
  samples.forEach((sample, i) => {
    if (PATCH_SIDECAR_TOOLS.has(sample.tool)) {
      preStagePatch(sample, i, workspace);
    }
  });

  if (samples.length === 0) {
    core.info("apply_samples: nothing to replay; exiting cleanly.");
    writeSyntheticStdioLog(logPath, 0);
    return;
  }

  const serverPath = resolveMcpServerPath();
  core.info(`apply_samples: spawning MCP server ${serverPath}`);
  const child = spawn(process.execPath, [serverPath], {
    stdio: ["pipe", "pipe", "inherit"],
    env: process.env,
  });

  const stdoutIter = lineIterator(child.stdout);
  let nextId = 1;
  const failures = [];

  try {
    // Initialize handshake.
    const initRsp = await sendJsonRpc(
      child,
      child.stdin,
      {
        jsonrpc: "2.0",
        id: nextId++,
        method: "initialize",
        params: {
          protocolVersion: "2025-06-18",
          capabilities: {},
          clientInfo: { name: "apply_samples", version: "1.0.0" },
        },
      },
      stdoutIter
    );
    if (initRsp.error) {
      throw new Error(`${ERR_API}: MCP initialize failed: ${JSON.stringify(initRsp.error)}`);
    }

    // Send one tools/call per sample.
    for (const [i, sample] of samples.entries()) {
      const callRsp = await sendJsonRpc(
        child,
        child.stdin,
        {
          jsonrpc: "2.0",
          id: nextId++,
          method: "tools/call",
          params: { name: sample.tool, arguments: sample.arguments },
        },
        stdoutIter
      );
      if (callRsp.error) {
        failures.push(`sample[${i}] (tool=${sample.tool}): ${JSON.stringify(callRsp.error)}`);
        continue;
      }
      const result = callRsp.result;
      if (result && result.isError) {
        const text = result.content && result.content[0] && result.content[0].text;
        failures.push(`sample[${i}] (tool=${sample.tool}): ${text || JSON.stringify(result)}`);
      } else {
        core.info(`apply_samples: sample[${i}] (tool=${sample.tool}) ok`);
      }
    }
  } finally {
    try {
      child.stdin.end();
    } catch {
      /* ignore */
    }
    // Give the server up to 2s to exit cleanly.
    await new Promise(resolve => {
      const timer = setTimeout(() => {
        try {
          child.kill("SIGTERM");
        } catch {
          /* ignore */
        }
        resolve(undefined);
      }, 2000);
      child.once("exit", () => {
        clearTimeout(timer);
        resolve(undefined);
      });
    });
  }

  writeSyntheticStdioLog(logPath, samples.length);

  if (failures.length > 0) {
    throw new Error(`${ERR_API}: apply_samples: ${failures.length} sample(s) failed:\n  - ${failures.join("\n  - ")}`);
  }
  core.info(`apply_samples: ${samples.length} sample(s) replayed successfully.`);
}

if (require.main === module) {
  main().catch(err => {
    core.setFailed(err && err.stack ? err.stack : String(err));
  });
}

module.exports = { main, loadSamples, preStagePatch, resolveMcpServerPath, sendJsonRpc };
