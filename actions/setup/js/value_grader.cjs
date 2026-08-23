// @ts-check

const cp = require("child_process");
const fs = require("fs");
const os = require("os");
const path = require("path");
const { getErrorMessage } = require("./error_helpers.cjs");

const VALUE_FUNCTION_TIMEOUT_MS = 120000;
const VALUE_FUNCTION_MAX_OUTPUT = 1024 * 1024;
const VALUE_EVENT_MAX_SIZE = 1024 * 1024;

/** @param {unknown} value @returns {value is Record<string, any>} */
function isRecord(value) {
  return value !== null && typeof value === "object" && !Array.isArray(value);
}

/** @param {string} value @param {string} label @returns {number} */
function parseTimestamp(value, label) {
  if (typeof value !== "string" || !/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d{3})?Z$/.test(value)) {
    throw new Error(`${label} must be a UTC ISO-8601 timestamp`);
  }
  const timestamp = Date.parse(value);
  if (!Number.isFinite(timestamp)) {
    throw new Error(`${label} must be a valid timestamp`);
  }
  return timestamp;
}

/** @param {NodeJS.ProcessEnv} env @param {{createdAt?: string}} [metadata] */
function buildRunSubject(env, metadata = {}) {
  const runId = String(env.GITHUB_RUN_ID || "");
  if (!/^\d+$/.test(runId) || runId === "0") {
    throw new Error("GITHUB_RUN_ID must identify the workflow run");
  }
  return {
    id: runId,
    attempt: Number(env.GITHUB_RUN_ATTEMPT) || 1,
    repository: String(env.GITHUB_REPOSITORY || ""),
    workflow: String(env.GITHUB_WORKFLOW || ""),
    ref: String(env.GITHUB_REF || ""),
    sha: String(env.GITHUB_SHA || ""),
    eventName: String(env.GITHUB_EVENT_NAME || ""),
    createdAt: metadata.createdAt || null,
  };
}

/** @param {NodeJS.ProcessEnv} env */
function readEventPayload(env) {
  const eventPath = env.GITHUB_EVENT_PATH;
  if (!eventPath) return null;
  try {
    const stat = fs.statSync(eventPath);
    if (!stat.isFile() || stat.size > VALUE_EVENT_MAX_SIZE) return null;
    const event = JSON.parse(fs.readFileSync(eventPath, "utf8"));
    return isRecord(event) ? event : null;
  } catch {
    return null;
  }
}

/** @param {NodeJS.ProcessEnv} env */
function safeFunctionEnv(env) {
  /** @type {NodeJS.ProcessEnv} */
  const result = {};
  for (const key of ["PATH", "HOME", "TMPDIR", "TEMP", "TMP", "SystemRoot", "ComSpec", "GH_TOKEN", "GH_HOST", "GITHUB_API_URL", "GITHUB_SERVER_URL"]) {
    if (env[key]) result[key] = env[key];
  }
  return result;
}

function parseBaselineDefinition(rawDefinition) {
  let definition;
  try {
    definition = JSON.parse(rawDefinition || "{}");
  } catch (err) {
    throw new Error(`value function returned an invalid definition: ${getErrorMessage(err)}`, { cause: err });
  }
  if (!isRecord(definition) || definition.schemaVersion !== 4 || definition.grader !== "value" || !isRecord(definition.baseline)) {
    throw new Error("value function definition must use schemaVersion 4 and grader 'value'");
  }
  if (definition.baseline.mode === "attainment-only") {
    if (definition.baseline.value !== null) throw new Error("attainment-only value functions must have a null baseline value");
    return null;
  }
  if (definition.baseline.mode !== "baseline-comparable") {
    throw new Error("value function baseline mode must be 'baseline-comparable' or 'attainment-only'");
  }
  const baselineValue = definition.baseline.value;
  if (typeof baselineValue !== "number" || !Number.isFinite(baselineValue) || baselineValue < 0 || baselineValue > 1) {
    throw new Error("baseline-comparable value functions require a baseline value in [0,1]");
  }
  return baselineValue;
}

/**
 * Execute and validate one trusted, frozen value function.
 * @param {string} functionContent
 * @param {{digest?: string, config?: object}} meta
 * @param {{evidenceAt?: string, env?: NodeJS.ProcessEnv, event?: object|null, case?: object|null, runMetadata?: {createdAt?: string}, bashPath?: string}} [options]
 */
function executeValueFunction(functionContent, meta, options = {}) {
  const env = options.env || process.env;
  const evidenceAt = options.evidenceAt || new Date().toISOString();
  const evidenceAtMs = parseTimestamp(evidenceAt, "evidenceAt");
  const run = buildRunSubject(env, options.runMetadata);
  const request = {
    schemaVersion: 1,
    run,
    evidenceAt,
    case: options.case || null,
    event: options.event === undefined ? readEventPayload(env) : options.event,
    config: meta.config || {},
  };

  const tempDir = fs.mkdtempSync(path.join(os.tmpdir(), "gh-aw-value-grader-"));
  const functionPath = path.join(tempDir, "value.sh");
  const bashPath = options.bashPath || "/bin/bash";
  try {
    fs.writeFileSync(functionPath, functionContent, { encoding: "utf8", mode: 0o700 });
    const syntax = cp.spawnSync(bashPath, ["-n", functionPath], {
      encoding: "utf8",
      timeout: 5000,
      env: safeFunctionEnv(env),
    });
    if (syntax.error || syntax.status !== 0) {
      throw new Error(`value function has invalid Bash syntax: ${syntax.stderr?.trim() || getErrorMessage(syntax.error)}`);
    }

    const definitionExecution = cp.spawnSync(bashPath, [functionPath, "--definition"], {
      encoding: "utf8",
      timeout: 5000,
      maxBuffer: VALUE_FUNCTION_MAX_OUTPUT,
      env: safeFunctionEnv(env),
    });
    if (definitionExecution.error) throw definitionExecution.error;
    if (definitionExecution.status !== 0) {
      throw new Error(definitionExecution.stderr?.trim() || `value function --definition exited with status ${String(definitionExecution.status)}`);
    }
    const baselineValue = parseBaselineDefinition(definitionExecution.stdout);

    const execution = cp.spawnSync(bashPath, [functionPath, "--grade-run"], {
      input: JSON.stringify(request),
      encoding: "utf8",
      timeout: VALUE_FUNCTION_TIMEOUT_MS,
      maxBuffer: VALUE_FUNCTION_MAX_OUTPUT,
      env: safeFunctionEnv(env),
    });
    if (execution.error) throw execution.error;
    if (execution.status !== 0) {
      throw new Error(execution.stderr?.trim() || `value function exited with status ${String(execution.status)}`);
    }

    let output;
    try {
      output = JSON.parse(execution.stdout || "{}");
    } catch (err) {
      throw new Error(`value function returned invalid JSON: ${getErrorMessage(err)}`, { cause: err });
    }
    if (!isRecord(output)) throw new Error("value function output must be an object");
    if (output.value !== null && (typeof output.value !== "number" || !Number.isFinite(output.value) || output.value < 0 || output.value > 1)) {
      throw new Error("value function value must be null or a finite number in [0,1]");
    }
    if (!isRecord(output.case)) throw new Error("value function output.case must be an object");
    if (typeof output.opportunityKey !== "string" || output.opportunityKey.trim() === "") {
      throw new Error("value function opportunityKey must be a non-empty string");
    }
    const evidenceCutoffMs = parseTimestamp(output.evidenceCutoff, "evidenceCutoff");
    const maturesAtMs = parseTimestamp(output.maturesAt, "maturesAt");
    if (evidenceCutoffMs > evidenceAtMs) throw new Error("value function evidenceCutoff cannot follow evidenceAt");
    if (evidenceCutoffMs > maturesAtMs) throw new Error("value function evidenceCutoff cannot follow maturesAt");
    if (!Array.isArray(output.provenance) || (output.value !== null && output.provenance.length === 0)) {
      throw new Error("value function must return provenance for a numeric value");
    }
    for (const provenance of output.provenance) {
      if (!isRecord(provenance) || !["repository", "kind", "ref"].every(key => typeof provenance[key] === "string" && provenance[key].length > 0)) {
        throw new Error("value function provenance entries require repository, kind, and ref");
      }
    }
    return {
      value: output.value,
      ...(typeof output.message === "string" ? { message: output.message } : {}),
      ...(isRecord(output.diagnostics) ? { diagnostics: output.diagnostics } : {}),
      observation: {
        subject: {
          type: "workflow-run",
          runId: run.id,
          attempt: run.attempt,
          repository: run.repository,
          workflow: run.workflow,
          ref: run.ref,
          sha: run.sha,
          eventName: run.eventName,
          createdAt: run.createdAt,
        },
        opportunityKey: output.opportunityKey,
        evidenceAt,
        evidenceCutoff: output.evidenceCutoff,
        maturesAt: output.maturesAt,
        mature: evidenceAtMs >= maturesAtMs,
        case: output.case,
        provenance: output.provenance,
      },
      baselineValue,
      deltaFromBaseline: typeof output.value === "number" && baselineValue !== null ? output.value - baselineValue : null,
    };
  } finally {
    fs.rmSync(tempDir, { recursive: true, force: true });
  }
}

module.exports = {
  executeValueFunction,
  buildRunSubject,
  readEventPayload,
  parseTimestamp,
  parseBaselineDefinition,
  VALUE_FUNCTION_TIMEOUT_MS,
  VALUE_FUNCTION_MAX_OUTPUT,
  VALUE_EVENT_MAX_SIZE,
};
