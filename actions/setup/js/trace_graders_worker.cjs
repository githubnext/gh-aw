// @ts-check

const vm = require("vm");

/**
 * @param {any} value
 * @param {string} label
 * @returns {any}
 */
function tryStructuredCloneOrUndefined(value, label) {
  try {
    return structuredClone(value);
  } catch {
    process.stderr.write(`grader worker: failed to structuredClone ${label}; value will default to {}\n`);
    return undefined;
  }
}

/**
 * @param {any} obj
 * @returns {any}
 */
function deepFreeze(obj) {
  if (obj === null || typeof obj !== "object") return obj;
  Object.freeze(obj);
  for (const key of Object.getOwnPropertyNames(obj)) {
    const value = obj[key];
    if (value !== null && typeof value === "object" && !Object.isFrozen(value)) {
      deepFreeze(value);
    }
  }
  return obj;
}

function readStdin() {
  return new Promise(resolve => {
    let data = "";
    process.stdin.setEncoding("utf-8");
    process.stdin.on("data", chunk => {
      data += chunk;
    });
    process.stdin.on("end", () => resolve(data));
  });
}

async function main() {
  const raw = await readStdin();
  let payload;
  try {
    payload = JSON.parse(raw || "{}");
  } catch (err) {
    const message = err instanceof Error ? err.message : String(err);
    process.stderr.write(`invalid worker payload: ${message}\n`);
    process.exit(1);
  }

  try {
    const trace = deepFreeze(tryStructuredCloneOrUndefined(payload.trace, "trace") ?? {});
    const config = deepFreeze(tryStructuredCloneOrUndefined(payload.config, "config") ?? {});
    const run = deepFreeze({ graderCount: Number(payload.graderCount) || 0 });
    const workflow = deepFreeze({});
    const script = String(payload.script || "");

    const sandbox = {
      trace,
      run,
      workflow,
      config,
      Date: undefined,
      fetch: undefined,
      require: undefined,
      process: undefined,
      global: undefined,
      globalThis: undefined,
      Function: undefined,
      eval: undefined,
      undefined,
      NaN,
      Infinity,
    };
    const context = vm.createContext(sandbox, { codeGeneration: { strings: false, wasm: false } });
    const timeoutMs = Number(payload.timeoutMs) || 5000;

    const runtimeBindings = vm.runInContext(
      `
        "use strict";
        (() => {
          const m = {};
          const descriptors = Object.getOwnPropertyDescriptors(Math);
          for (const key of Object.keys(descriptors)) {
            Object.defineProperty(m, key, descriptors[key]);
          }
          Object.defineProperty(m, "random", {
            value: undefined,
            writable: false,
            enumerable: true,
            configurable: false
          });
          const safeMath = Object.freeze(m);
          const helpers = Object.freeze({
            clamp: (v, lo, hi) => safeMath.max(lo, safeMath.min(hi, v)),
            ratio: (num, den) => (den === 0 ? 0 : num / den),
            sum: arr => arr.reduce((a, b) => a + b, 0)
          });
          return { safeMath, helpers };
        })();
      `,
      context,
      { timeout: 1000, filename: "grader:bootstrap" }
    );
    Object.defineProperty(context, "helpers", {
      value: runtimeBindings.helpers,
      writable: false,
      enumerable: true,
      configurable: false,
    });
    Object.defineProperty(context, "__math", {
      value: runtimeBindings.safeMath,
      writable: false,
      enumerable: false,
      configurable: false,
    });

    const graderFn = vm.compileFunction(`"use strict";\n${script}`, ["trace", "run", "workflow", "config", "helpers", "Math"], {
      parsingContext: context,
      filename: `grader:${String(payload.id || "unknown")}`,
    });
    context.__grader = graderFn;
    const value = vm.runInContext("__grader(trace, run, workflow, config, helpers, __math)", context, {
      timeout: timeoutMs,
      filename: `grader:${String(payload.id || "unknown")}:invoke`,
    });
    if (typeof value === "number" && !Number.isFinite(value)) {
      throw new Error("custom grader returned non-finite numeric value");
    }

    process.stdout.write(JSON.stringify({ ok: true, value }));
  } catch (err) {
    const message = err instanceof Error ? err.message : String(err);
    process.stdout.write(JSON.stringify({ ok: false, error: message }));
  }
}

main().catch(err => {
  const message = err instanceof Error ? err.stack || err.message : String(err);
  process.stdout.write(JSON.stringify({ ok: false, error: message }), () => {
    process.exit(1);
  });
});
