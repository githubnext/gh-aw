// @ts-check
"use strict";

const fs = require("node:fs");
const path = require("node:path");
const nativeAssert = require("node:assert/strict");

const DIAGNOSTICS_FILE = "smoke-evidence.json";
/** @type {Record<string, string[]>} */
const STEPS = {
  routing: ["Assert model-routing evidence", "Assert model-routing and sub-agent evidence"],
  subagents: ["Assert sub-agent model and wire API evidence"],
  goose: ["Assert Goose smoke evidence"],
  session: ["Verify agent session artifact"],
};
/** @type {Record<string, string>} */
const FILES = {
  session: "usage/aw_session.jsonl",
  proxy: "sandbox/firewall/logs/api-proxy-logs/token-usage.jsonl",
  proxyAudit: "sandbox/firewall/audit/api-proxy-logs/token-usage.jsonl",
  proxyLegacy: "sandbox/firewall-audit-logs/api-proxy-logs/token-usage.jsonl",
  reflect: "sandbox/firewall/awf-reflect.json",
  reflectAgent: "agent/awf-reflect.json",
  gooseResult: "agent/smoke-test-goose-<run-id>.json",
  gooseText: "agent/smoke-test-goose-<run-id>.txt",
  webReceipt: "runner.temp/smoke-goose/web-fetch.json",
  binary: "workspace/gh-aw",
  stdio: "agent-stdio.log",
  nativeSession: "agent-session.jsonl",
  copilotEvents: "sandbox/agent/logs/copilot-session-state/**/events.jsonl",
  tempStdio: "runner.temp/gh-aw/agent-stdio.log",
  tempCopilotEvents: "runner.temp/gh-aw/sandbox/agent/logs/copilot-session-state/**/events.jsonl",
};
const CHECKS = [
  "Smoke evidence schema or value mismatch",
  "Smoke evidence file is missing or unreadable",
  "Smoke evidence contains invalid JSON",
  "Smoke evidence must be a regular file",
  "Smoke evidence exceeds 16 KiB",
  "Goose execution failed",
  "Goose result fields are invalid",
  "Pull request evidence fields are invalid",
  "Pull request number is invalid",
  "Web fetch receipt is invalid",
  "Goose file-write receipt is invalid",
  "Goose cache-write usage differs from native usage",
  "Goose cache accounting is missing",
  ...["bash", "build", "fileWrite", "runtime", "webFetch"].map(key => `Goose smoke check ${key} did not pass`),
  "Exactly two merged PRs are required",
  "PRs must be distinct",
  "Pull request must be merged",
  "Pull request title does not match GitHub",
  "gh-aw build output is missing",
  "Pinned CLI verification is missing",
  "A Goose MCP extension failed to connect",
  "Goose did not finish a structured run",
  "No successful native GitHub MCP round-trip",
  "Goose session contains non-canonical events",
  "Goose session initialization is missing",
  "Goose assistant text is missing",
  "Goose tool events are missing",
  "Goose tool starts are duplicated",
  "Goose tool completion has no matching start",
  "Goose session result is not complete",
  ...["input_tokens", "output_tokens", "total_tokens", "cache_read_input_tokens"].map(key => `Goose session ${key} differs from native usage`),
  "Goose execution result is missing",
  "Control variant must not invoke sub-agents",
  "Copilot SDK execution failed",
  "Missing api-proxy token usage evidence",
  ...["haiku-whoami", "mini-whoami", "nano-whoami"].flatMap(name => [
    `${name} must actually run exactly once`,
    `${name} is missing an invocation ID`,
    `${name} did not complete`,
    `${name} failed`,
    `${name} has no recorded model`,
    `${name} changed or fell back from ${name === "haiku-whoami" ? "claude-haiku-4.5" : name === "mini-whoami" ? "gpt-5-mini" : "gpt-5-nano"}`,
    `${name} has no successful ${name === "haiku-whoami" ? "claude-haiku-4.5 request on /chat/completions" : name === "mini-whoami" ? "gpt-5-mini request on /responses" : "gpt-5-nano request on /responses"}`,
  ]),
  "No non-empty Copilot session events.jsonl found in the agent logs",
  "No non-empty agent-stdio.log found in the agent logs",
];
/** @type {Record<string, string>} */
const ROUTING_CHECKS = {
  C1: "Declared sub-agent models must be distinct and disjoint from allowed models",
  E1: "Agent execution did not succeed",
  E2: "Unified session or API proxy evidence is missing or invalid",
  R1: "Runner routing selection and an allowed firewall-selected model are required",
  R2: "Exactly one routing classification request is required",
  R3: "A successful main-agent request on the selected model and supported endpoint is required",
  R4: "Main-agent requests must exist and use only allowed models",
  M1: "Main-agent requests must exist and use the expected endpoint",
  S1: "Each declared sub-agent must start exactly once",
  S2: "Each declared sub-agent must complete without failure",
  S3: "Each declared sub-agent must use its declared model and endpoint successfully",
  S4: "Sub-agent events must carry their declared names",
};
const SUBAGENTS = ["haiku-whoami", "mini-whoami", "nano-whoami"];
const MISSING_ROUTING_FILES = ["Unified session usage/aw_session.jsonl is missing", "API proxy token-usage.jsonl is missing"];
const ALLOWED_CHECKS = new Set([
  ...CHECKS,
  ...Object.values(ROUTING_CHECKS),
  ...MISSING_ROUTING_FILES,
  ...SUBAGENTS.flatMap(name => ["S1", "S2", "S3", "S4"].map(id => `${name}: ${ROUTING_CHECKS[id]}`)),
  ...Object.values(FILES).flatMap(file => [`${CHECKS[1]}: ${file}`, `${CHECKS[2]}: ${file}`]),
]);

/** @param {string} line @returns {string} */
function routingFailureDescription(line) {
  const id = line.split(" ")[1];
  if (line === "FAIL E2 evidence: missing usage/aw_session.jsonl") return MISSING_ROUTING_FILES[0];
  if (line.startsWith("FAIL E2 evidence: missing api-proxy token-usage.jsonl")) return MISSING_ROUTING_FILES[1];
  const name = SUBAGENTS.find(name => line.startsWith(`FAIL ${id} sub-agent ${name}:`));
  if (name && ["S1", "S2", "S3", "S4"].includes(id)) return `${name}: ${ROUTING_CHECKS[id]}`;
  return ROUTING_CHECKS[id] || CHECKS[0];
}

/**
 * Persist only catalogued descriptions, never exception text, actual/expected
 * assertion values, file contents, or agent-controlled metadata.
 * @param {{kind: string, failures: string[], files: string[], rootDir?: string}} options
 */
function recordSmokeFailure({ kind, failures, files, rootDir = "/tmp/gh-aw" }) {
  if (!Object.hasOwn(STEPS, kind)) throw new Error("Unknown smoke evidence kind");
  const record = { kind, failures: [...new Set(failures.map(value => (ALLOWED_CHECKS.has(value) ? value : CHECKS[0])))], files: [...new Set(files.filter(key => Object.hasOwn(FILES, key)))] };
  fs.mkdirSync(rootDir, { recursive: true });
  const destination = path.join(rootDir, DIAGNOSTICS_FILE);
  const temporary = fs.mkdtempSync(path.join(rootDir, ".smoke-evidence-"));
  try {
    const file = path.join(temporary, DIAGNOSTICS_FILE);
    fs.writeFileSync(file, JSON.stringify(record), { mode: 0o600 });
    fs.renameSync(file, destination);
  } finally {
    fs.rmdirSync(temporary);
  }
}

/**
 * @param {{kind: string, core: any, paths: Record<string, string>, rootDir?: string, fs?: any}} options
 * @param {(tools: {assert: any, fs: any, parse: (text: string) => any}) => Promise<void>} verify
 */
async function withSmokeEvidence(options, verify) {
  /** @type {Set<string>} */
  const inspected = new Set();
  let currentFile = "";
  let missing = CHECKS[0];
  const sourceFs = options.fs || fs;
  const trackedFs = { ...sourceFs };
  for (const method of ["readFileSync", "lstatSync", "existsSync"]) {
    trackedFs[method] = (file, ...args) => {
      const key = Object.keys(options.paths).find(key => options.paths[key] === file);
      if (key) {
        inspected.add(key);
        currentFile = FILES[key];
      }
      missing = key ? `${CHECKS[1]}: ${FILES[key]}` : CHECKS[1];
      const result = sourceFs[method](file, ...args);
      missing = CHECKS[0];
      return result;
    };
  }
  const assert = new Proxy(/** @type {any} */ nativeAssert, {
    get(target, method) {
      const original = target[method];
      if (typeof original !== "function") return original;
      return (...args) => {
        const message = args.at(-1);
        missing = typeof message === "string" && ALLOWED_CHECKS.has(message) ? message : CHECKS[0];
        const result = original(...args);
        missing = CHECKS[0];
        return result;
      };
    },
  });
  try {
    await verify({
      assert,
      fs: trackedFs,
      parse: text => {
        missing = currentFile ? `${CHECKS[2]}: ${currentFile}` : CHECKS[2];
        const value = JSON.parse(text);
        missing = CHECKS[0];
        return value;
      },
    });
  } catch {
    const message = `${missing}. Files inspected: ${[...inspected].map(key => FILES[key]).join(", ") || "none recorded"}`;
    options.core.error(message);
    try {
      recordSmokeFailure({ ...options, failures: [missing], files: [...inspected] });
    } catch {
      options.core.warning("Unable to persist smoke evidence diagnostics");
    }
    throw new Error(message);
  }
}

/**
 * The downloaded artifact is untrusted: render only fixed catalogue entries.
 * @param {string} [rootDir]
 * @returns {{kind: string, failures: string[], files: string[]} | null}
 */
function loadSmokeFailure(rootDir = "/tmp/gh-aw") {
  const file = path.join(rootDir, DIAGNOSTICS_FILE);
  let stat;
  try {
    stat = fs.lstatSync(file, { throwIfNoEntry: false });
  } catch {
    global.core?.warning("Unable to inspect smoke evidence diagnostics");
    return null;
  }
  if (!stat) return null;
  if (!stat.isFile() || stat.isSymbolicLink() || stat.size > 16384) {
    global.core?.warning("Invalid smoke evidence diagnostics file");
    return null;
  }
  let record;
  try {
    record = JSON.parse(fs.readFileSync(file, "utf8"));
  } catch {
    global.core?.warning("Smoke evidence diagnostics contain invalid JSON");
    return null;
  }
  if (
    !record ||
    typeof record.kind !== "string" ||
    !Object.hasOwn(STEPS, record.kind) ||
    !Array.isArray(record.failures) ||
    !record.failures.length ||
    !record.failures.every(value => typeof value === "string" && ALLOWED_CHECKS.has(value)) ||
    !Array.isArray(record.files) ||
    !record.files.every(key => typeof key === "string" && Object.hasOwn(FILES, key))
  ) {
    global.core?.warning("Invalid smoke evidence diagnostics schema");
    return null;
  }
  return { kind: record.kind, failures: [...new Set(record.failures)], files: [...new Set(record.files)] };
}

/** @param {string} failingStep @param {string} [rootDir] @returns {string} */
function readSmokeFailure(failingStep, rootDir = "/tmp/gh-aw") {
  const record = loadSmokeFailure(rootDir);
  if (!record || !STEPS[record.kind].includes(failingStep)) return "";
  return `**Missing or invalid smoke evidence:**\n${[...new Set(record.failures)].map(value => `- ${value}`).join("\n")}\n\n**Files inspected:** ${[...new Set(record.files)].map(key => `\`${FILES[key]}\``).join(", ") || "none recorded"}\n\n`;
}

/** @param {string} [rootDir] @param {string} [outputFile] */
function sanitizeSmokeFailure(rootDir = "/tmp/gh-aw", outputFile = process.env.GITHUB_OUTPUT) {
  const record = loadSmokeFailure(rootDir);
  if (record) {
    recordSmokeFailure({ ...record, rootDir });
    if (outputFile) fs.appendFileSync(outputFile, "ready=true\n");
  } else {
    const file = path.join(rootDir, DIAGNOSTICS_FILE);
    if (fs.lstatSync(file, { throwIfNoEntry: false })) {
      console.error("Discarding invalid smoke evidence diagnostics before artifact upload");
      const quarantine = fs.mkdtempSync(path.join(rootDir, ".invalid-smoke-evidence-"));
      fs.renameSync(file, path.join(quarantine, DIAGNOSTICS_FILE));
    }
  }
}
module.exports = { DIAGNOSTICS_FILE, FILES, ROUTING_CHECKS, routingFailureDescription, recordSmokeFailure, withSmokeEvidence, readSmokeFailure, sanitizeSmokeFailure };

if (require.main === module) {
  if (process.argv[2] === "--sanitize") {
    sanitizeSmokeFailure();
  } else {
    const copilot = process.argv[2] === "copilot-events";
    const failures = [copilot ? "No non-empty Copilot session events.jsonl found in the agent logs" : "No non-empty agent-stdio.log found in the agent logs"];
    const files = copilot ? ["copilotEvents", "tempCopilotEvents"] : ["stdio", "tempStdio"];
    recordSmokeFailure({ kind: "session", failures, files });
    console.error(`${failures[0]}. Files inspected: ${files.map(key => FILES[key]).join(", ")}`);
  }
}
