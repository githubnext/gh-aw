// @ts-check

/**
 * shim.cjs
 *
 * Provides a minimal `global.core` shim so that modules written for the
 * GitHub Actions `github-script` context (which rely on the built-in `core`
 * global) work correctly when executed as plain Node.js processes, such as
 * inside the safe-outputs and safe-inputs MCP servers.
 *
 * When `global.core` is already set (i.e. running inside `github-script`)
 * this module is a no-op.
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
