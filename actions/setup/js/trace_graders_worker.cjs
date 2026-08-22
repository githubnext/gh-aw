// @ts-check

const vm = require("vm");

/**
 * @param {any} obj
 * @returns {any}
 */
function deepClone(obj) {
  if (obj === null || obj === undefined) return obj;
  return JSON.parse(JSON.stringify(obj));
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

function makeSandboxMath() {
  const math = {};
  Object.defineProperties(math, Object.getOwnPropertyDescriptors(Math));
  Object.defineProperty(math, "random", {
    value: undefined,
    writable: false,
    enumerable: true,
    configurable: false,
  });
  return Object.freeze(math);
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
    const trace = deepFreeze(deepClone(payload.trace || {}));
    const config = deepFreeze(deepClone(payload.config || {}));
    const run = deepFreeze({ graderCount: 0 });
    const workflow = deepFreeze({});
    const helpers = deepFreeze({
      clamp: (v, lo, hi) => Math.max(lo, Math.min(hi, v)),
      ratio: (num, den) => (den === 0 ? 0 : num / den),
      sum: arr => arr.reduce((a, b) => a + b, 0),
    });

    const script = String(payload.script || "");
    const wrappedScript = `
      (() => {
        const __grader = (trace, run, workflow, config, helpers) => {
          "use strict";
          ${script}
        };
        return __grader(trace, run, workflow, config, helpers);
      })()
    `;

    const sandbox = {
      trace,
      run,
      workflow,
      config,
      helpers,
      Math: makeSandboxMath(),
      JSON: Object.freeze({ parse: JSON.parse, stringify: JSON.stringify }),
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
    const value = vm.runInContext(wrappedScript, context, {
      timeout: timeoutMs,
      filename: `grader:${String(payload.id || "unknown")}`,
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

main();
