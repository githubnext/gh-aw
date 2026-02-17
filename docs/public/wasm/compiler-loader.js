/**
 * compiler-loader.js -- ES module that spawns compiler-worker.js and
 * provides a clean async API for the Astro docs site.
 *
 * Usage:
 *   import { createWorkerCompiler } from '/wasm/compiler-loader.js';
 *
 *   const compiler = createWorkerCompiler();
 *   await compiler.ready;
 *   const { yaml, warnings, error } = await compiler.compile(markdownString);
 *   // With imports:
 *   const { yaml } = await compiler.compile(markdown, { 'shared/tools.md': '...' });
 *   compiler.terminate();
 */

/**
 * Create a worker-backed compiler instance.
 *
 * @param {Object} [options]
 * @param {string} [options.workerUrl] - URL to compiler-worker.js
 *        (default: resolves relative to this module)
 * @returns {{ compile: (markdown: string, files?: Record<string,string>) => Promise<{yaml: string, warnings: string[], error: string|null}>,
 *             ready: Promise<void>,
 *             terminate: () => void }}
 */
export function createWorkerCompiler(options = {}) {
  const moduleDir = new URL('.', import.meta.url).href;
  const workerUrl = options.workerUrl || new URL('compiler-worker.js', moduleDir).href;

  const worker = new Worker(workerUrl);

  // Monotonically increasing message ID for correlating requests/responses.
  let nextId = 1;

  // Map of pending compile requests: id -> { resolve, reject }
  const pending = new Map();

  // Ready promise -- resolves when the worker sends { type: 'ready' }.
  let readyResolve;
  let readyReject;
  const ready = new Promise((resolve, reject) => {
    readyResolve = resolve;
    readyReject = reject;
  });

  let isReady = false;
  let isTerminated = false;

  /**
   * Handle messages from the worker.
   */
  worker.onmessage = function (event) {
    const msg = event.data;

    switch (msg.type) {
      case 'ready':
        isReady = true;
        readyResolve();
        break;

      case 'result': {
        const entry = pending.get(msg.id);
        if (entry) {
          pending.delete(msg.id);
          entry.resolve({
            yaml: msg.yaml,
            warnings: msg.warnings || [],
            error: null,
          });
        }
        break;
      }

      case 'error': {
        // An error with id === null means init failure.
        if (msg.id === null || msg.id === undefined) {
          readyReject(new Error(msg.error));
          break;
        }

        const entry = pending.get(msg.id);
        if (entry) {
          pending.delete(msg.id);
          entry.resolve({
            yaml: '',
            warnings: [],
            error: msg.error,
          });
        }
        break;
      }
    }
  };

  /**
   * Handle worker-level errors (e.g. script load failure).
   */
  worker.onerror = function (event) {
    const errorMsg = event.message || 'Unknown worker error';

    if (!isReady) {
      readyReject(new Error(errorMsg));
    }

    // Reject all pending requests.
    for (const [id, entry] of pending) {
      entry.reject(new Error(errorMsg));
    }
    pending.clear();
  };

  /**
   * Compile a markdown workflow string to GitHub Actions YAML.
   *
   * @param {string} markdown
   * @param {Record<string, string>} [files] - Optional map of file paths to content
   *        for import resolution (e.g. {"shared/tools.md": "---\ntools:..."})
   * @returns {Promise<{yaml: string, warnings: string[], error: string|null}>}
   */
  function compile(markdown, files) {
    if (isTerminated) {
      return Promise.reject(new Error('Compiler worker has been terminated.'));
    }

    const id = nextId++;

    return new Promise((resolve, reject) => {
      pending.set(id, { resolve, reject });
      const msg = { type: 'compile', id, markdown };
      if (files && Object.keys(files).length > 0) {
        msg.files = files;
      }
      worker.postMessage(msg);
    });
  }

  /**
   * Terminate the worker. After this, compile() will reject.
   */
  function terminate() {
    if (isTerminated) return;
    isTerminated = true;
    worker.terminate();

    // Reject anything still pending.
    for (const [id, entry] of pending) {
      entry.reject(new Error('Compiler worker was terminated.'));
    }
    pending.clear();
  }

  return {
    compile,
    ready,
    terminate,
  };
}
