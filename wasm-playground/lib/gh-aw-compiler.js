/**
 * gh-aw-compiler.js — JavaScript glue library for gh-aw.wasm
 *
 * Loads the Go WebAssembly module and exposes a clean async API
 * for compiling agentic workflow markdown to GitHub Actions YAML.
 *
 * Usage:
 *   import { createCompiler } from './gh-aw-compiler.js';
 *   const compiler = await createCompiler();
 *   const { yaml, warnings, error } = await compiler.compile(markdown);
 */

// Resolve asset paths relative to this module's location
const MODULE_DIR = new URL('.', import.meta.url).href;
const WASM_EXEC_URL = new URL('../../wasm_exec.js', MODULE_DIR).href;
const WASM_URL = new URL('../../gh-aw.wasm', MODULE_DIR).href;

/**
 * Compiler states
 */
export const CompilerState = Object.freeze({
  LOADING: 'loading',
  READY: 'ready',
  ERROR: 'error',
});

/**
 * Create and initialize a gh-aw compiler instance.
 *
 * @param {Object} [options]
 * @param {string} [options.wasmExecUrl] - URL to wasm_exec.js (default: auto-resolved from module location)
 * @param {string} [options.wasmUrl]     - URL to gh-aw.wasm (default: auto-resolved from module location)
 * @returns {Promise<Compiler>} A ready-to-use compiler instance
 */
export async function createCompiler(options = {}) {
  const compiler = new Compiler(options);
  await compiler.init();
  return compiler;
}

/**
 * Compiler wraps the gh-aw WebAssembly module and provides
 * a clean async API for workflow compilation.
 */
export class Compiler {
  #state = CompilerState.LOADING;
  #error = null;
  #initPromise = null;
  #wasmExecUrl;
  #wasmUrl;

  /**
   * @param {Object} [options]
   * @param {string} [options.wasmExecUrl] - URL to wasm_exec.js
   * @param {string} [options.wasmUrl]     - URL to gh-aw.wasm
   */
  constructor(options = {}) {
    this.#wasmExecUrl = options.wasmExecUrl || WASM_EXEC_URL;
    this.#wasmUrl = options.wasmUrl || WASM_URL;
  }

  /** Current compiler state: 'loading' | 'ready' | 'error' */
  get state() {
    return this.#state;
  }

  /** Error that occurred during initialization, or null */
  get error() {
    return this.#error;
  }

  /**
   * Initialize the compiler by loading wasm_exec.js and the wasm module.
   * Safe to call multiple times — subsequent calls return the same promise.
   *
   * @returns {Promise<void>}
   */
  async init() {
    if (this.#initPromise) return this.#initPromise;
    this.#initPromise = this.#doInit();
    return this.#initPromise;
  }

  async #doInit() {
    try {
      this.#state = CompilerState.LOADING;

      // 1. Load wasm_exec.js if the Go class isn't already available
      if (typeof globalThis.Go === 'undefined') {
        await this.#loadScript(this.#wasmExecUrl);
      }

      if (typeof globalThis.Go === 'undefined') {
        throw new Error('Failed to load wasm_exec.js: globalThis.Go is not defined');
      }

      // 2. Instantiate the Go runtime and wasm module
      const go = new globalThis.Go();

      let result;
      if (typeof WebAssembly.instantiateStreaming === 'function') {
        try {
          result = await WebAssembly.instantiateStreaming(
            fetch(this.#wasmUrl),
            go.importObject,
          );
        } catch (streamErr) {
          // Fallback: instantiateStreaming may fail if the server does not
          // serve .wasm with the correct application/wasm MIME type.
          // Re-fetch because the previous response body may have been consumed.
          const fallbackResp = await fetch(this.#wasmUrl);
          const buf = await fallbackResp.arrayBuffer();
          result = await WebAssembly.instantiate(buf, go.importObject);
        }
      } else {
        // Environment does not support streaming compilation at all.
        const resp = await fetch(this.#wasmUrl);
        const buf = await resp.arrayBuffer();
        result = await WebAssembly.instantiate(buf, go.importObject);
      }

      // 3. Start the Go program (registers compileWorkflow globally).
      //    go.run() never resolves because main() does `select{}`,
      //    so we don't await it — we just kick it off.
      go.run(result.instance);

      // 4. Wait briefly for the global function to be registered
      await this.#waitForGlobal('compileWorkflow', 2000);

      this.#state = CompilerState.READY;
    } catch (err) {
      this.#state = CompilerState.ERROR;
      this.#error = err;
      throw err;
    }
  }

  /**
   * Compile a markdown workflow to GitHub Actions YAML.
   *
   * @param {string} markdown - The workflow markdown source
   * @returns {Promise<{yaml: string, warnings: string[], error: null}>}
   * @throws {Error} If the compiler is not ready or compilation fails
   */
  async compile(markdown) {
    if (this.#state !== CompilerState.READY) {
      throw new Error(`Compiler is not ready (state: ${this.#state})`);
    }

    if (typeof markdown !== 'string') {
      throw new TypeError('markdown must be a string');
    }

    return globalThis.compileWorkflow(markdown);
  }

  /**
   * Load a script by injecting a <script> tag (browser) or using importScripts (worker).
   */
  async #loadScript(url) {
    // Web Worker context
    if (typeof importScripts === 'function') {
      importScripts(url);
      return;
    }

    // Browser context
    if (typeof document !== 'undefined') {
      return new Promise((resolve, reject) => {
        const script = document.createElement('script');
        script.src = url;
        script.onload = resolve;
        script.onerror = () => reject(new Error(`Failed to load script: ${url}`));
        document.head.appendChild(script);
      });
    }

    throw new Error('Unsupported environment: neither document nor importScripts available');
  }

  /**
   * Poll for a global property to become defined.
   */
  async #waitForGlobal(name, timeoutMs) {
    const start = Date.now();
    while (typeof globalThis[name] === 'undefined') {
      if (Date.now() - start > timeoutMs) {
        throw new Error(`Timed out waiting for globalThis.${name} to be defined`);
      }
      await new Promise((r) => setTimeout(r, 10));
    }
  }
}
