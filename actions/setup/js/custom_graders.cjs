// @ts-check

const fs = require("fs");
const path = require("path");
const { spawnSync } = require("child_process");
const { createHash } = require("crypto");
const { createSessionLookups, serializeLookupsForWorker } = require("./grader_session.cjs");
const { buildGraderOutput, normalizeResult, BUILTIN_GRADERS, GRADER_VERSION } = require("./trace_graders.cjs");

/**
 * Orchestration boundary: JavaScript uses restricted workers; trusted Bash uses
 * the separate-job compatibility runtime, never conclusion.
 * @param {import("./types/unified_session").UnifiedSession} session
 * @param {any[]} graders
 */
function runCustomGraders(session, graders) {
  const lookups = createSessionLookups(session);
  const worker = path.join(__dirname, "custom_grader_worker.cjs");
  const permission = process.allowedNodeEnvironmentFlags.has("--permission") ? "--permission" : "--experimental-permission";
  if (graders.some(grader => grader?.source === "inline" && grader.enabled !== false) && !process.allowedNodeEnvironmentFlags.has(permission)) {
    throw new Error("custom JavaScript graders require Node.js permission support");
  }
  const ids = new Set();
  for (const grader of graders) {
    if (!grader || typeof grader.id !== "string" || !/^[a-z][a-z0-9-]{0,63}$/.test(grader.id) || Object.hasOwn(BUILTIN_GRADERS, grader.id) || !validCustomSource(grader)) {
      throw new Error("custom graders require a distinct JavaScript ID or operational-value with frozen Bash source; built-in overrides are unsupported");
    }
    if (ids.has(grader.id)) throw new Error(`duplicate custom grader ${grader.id}`);
    ids.add(grader.id);
  }
  const results = graders
    .filter(grader => grader.enabled !== false)
    .map(grader => {
      if (grader.source === "operational-value") return runOperationalValueGrader(session, grader);
      const input = JSON.stringify({
        id: grader.id,
        script: grader.script,
        data: {
          session,
          config: grader.config ?? {},
          lookups: serializeLookupsForWorker(lookups),
        },
      });
      const child = spawnSync(process.execPath, [permission, "--allow-fs-read=" + worker, "--max-old-space-size=256", worker], {
        input,
        encoding: "utf8",
        env: {},
        timeout: 7000,
        maxBuffer: 1024 * 1024,
        killSignal: "SIGKILL",
      });
      let result;
      try {
        if (child.error) throw child.error;
        if (child.status !== 0) throw new Error(`custom grader process exited with ${child.signal ?? child.status}`);
        const response = JSON.parse(child.stdout);
        if (!response.ok) throw new Error(response.error);
        result = normalizeResult(grader.id, response.value, grader);
      } catch (error) {
        result = { ...normalizeResult(grader.id, null, grader), status: "error", error: error instanceof Error ? error.message : String(error) };
      }
      return {
        ...result,
        source: "inline",
        implementation: { id: "gh-aw/custom-graders", version: GRADER_VERSION, digest: createHash("sha256").update(grader.script).digest("hex") },
      };
    });
  const priorManifest = lookups.eventsOfType("grader.manifest").at(-1)?.data;
  const priorResults = lookups.eventsOfType("grader.result").at(-1)?.data.results;
  const builtinResults = Array.isArray(priorResults) ? priorResults.filter(result => result.source === "builtin" && Object.hasOwn(BUILTIN_GRADERS, result.id)) : [];
  return {
    manifest: {
      version: GRADER_VERSION,
      customGraders: graders.filter(grader => grader.enabled !== false).map(grader => grader.id),
      graders: [...(Array.isArray(priorManifest?.graders) ? priorManifest.graders.filter(grader => grader.source === "builtin") : []), ...graders.map(({ script: _script, shellScript: _shellScript, ...metadata }) => metadata)],
    },
    output: buildGraderOutput([...builtinResults, ...results], lookups),
  };
}

/** @param {any} grader */
function validCustomSource(grader) {
  if (grader.source === "operational-value") {
    return (
      grader.id === "operational-value" &&
      typeof grader.run === "string" &&
      grader.run.endsWith(".sh") &&
      typeof grader.shellScript === "string" &&
      hasBashShebang(grader.shellScript) &&
      Buffer.byteLength(grader.shellScript) <= 64 * 1024 &&
      !Object.hasOwn(grader, "script")
    );
  }
  return (
    grader.id !== "operational-value" &&
    grader.source === "inline" &&
    typeof grader.script === "string" &&
    !!grader.script.trim() &&
    Buffer.byteLength(grader.script) <= 256 * 1024 &&
    !Object.hasOwn(grader, "run") &&
    !Object.hasOwn(grader, "shellScript")
  );
}

/** @param {string} source */
function hasBashShebang(source) {
  const [line] = source.split("\n", 1);
  if (!line.startsWith("#!")) return false;
  const fields = line.slice(2).trim().split(/\s+/);
  if (path.posix.isAbsolute(fields[0]) && path.posix.basename(fields[0]) === "bash") return true;
  if (fields[0] !== "/usr/bin/env" && fields[0] !== "/bin/env") return false;
  return (fields.length === 2 && fields[1] === "bash") || (fields.length >= 3 && fields[1] === "-S" && fields[2] === "bash");
}

/** @param {import("./types/unified_session").UnifiedSession} session @param {any} grader */
function runOperationalValueGrader(session, grader) {
  let result;
  try {
    const { executeOperationalValueEvaluator } = require("./operational_value_grader.cjs");
    const metrics = executeOperationalValueEvaluator(grader.shellScript, session, grader.config ?? {});
    result = {
      ...normalizeResult(
        grader.id,
        {
          value: metrics[0].value,
          diagnostics: Object.fromEntries(metrics.slice(1).map(metric => [metric.id, metric.value])),
        },
        grader
      ),
      metrics,
    };
  } catch (error) {
    result = { ...normalizeResult(grader.id, null, grader), status: "error", error: error instanceof Error ? error.message : String(error) };
  }
  return {
    ...result,
    source: "operational-value",
    implementation: { id: "gh-aw/operational-value", version: GRADER_VERSION, digest: createHash("sha256").update(grader.shellScript).digest("hex") },
  };
}

/** @param {{sessionPath?: string, outputDirectory?: string}} [options] */
async function main(options = {}) {
  const { iterateJsonlEvents } = require("./jsonl_helpers.cjs");
  const { readGraderManifestEnvironment } = require("./grader_artifacts.cjs");
  const session = [];
  const content = fs.readFileSync(options.sessionPath ?? "/tmp/gh-aw/usage/aw_session.jsonl", "utf8");
  for (const entry of iterateJsonlEvents(content)) {
    if (entry.type === "json") session.push(entry.value);
    else if (entry.reason !== "blank") throw new Error("invalid unified session for custom graders");
  }
  const { manifest, output } = runCustomGraders(session, readGraderManifestEnvironment("GH_AW_CUSTOM_GRADER_MANIFEST").graders);
  const directory = options.outputDirectory ?? "/tmp/gh-aw/custom-graders";
  fs.mkdirSync(directory, { recursive: true });
  fs.writeFileSync(path.join(directory, "grader_manifest.json"), JSON.stringify(manifest, null, 2), { mode: 0o600 });
  fs.writeFileSync(path.join(directory, "grader_results.json"), JSON.stringify(output, null, 2), { mode: 0o600 });
  for (const result of output.results) {
    if (result.source !== "builtin" && result.error) global.core.warning(`Custom grader ${result.id}: ${result.error}`);
  }
  if (output.results.some(result => result.source !== "builtin" && result.status === "error")) global.core.setFailed("One or more custom graders failed");
}

module.exports = { runCustomGraders, main };
