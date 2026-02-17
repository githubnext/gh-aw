/**
 * compiler-worker.js -- Web Worker that loads gh-aw.wasm and exposes
 * the compileWorkflow function via postMessage.
 *
 * Message protocol (inbound):
 *   { type: 'compile', id: <number|string>, markdown: <string>, files?: <object> }
 *
 * Message protocol (outbound):
 *   { type: 'ready' }
 *   { type: 'result', id: <number|string>, yaml: <string>, warnings: <string[]>, error: null }
 *   { type: 'error',  id: <number|string>, error: <string> }
 *
 * This file is a classic script (not an ES module) because Web Workers
 * need importScripts() to load wasm_exec.js synchronously.
 */

/* global importScripts, Go, compileWorkflow, WebAssembly */

'use strict';

(function () {
  // 1. Load Go's wasm_exec.js (provides the global `Go` class)
  importScripts('./wasm_exec.js');

  var ready = false;

  /**
   * Initialize the Go WebAssembly runtime.
   */
  async function init() {
    try {
      var go = new Go();

      // Try streaming instantiation first; fall back to array buffer
      // for servers that don't serve .wasm with application/wasm MIME type.
      var result;
      try {
        result = await WebAssembly.instantiateStreaming(
          fetch('./gh-aw.wasm'),
          go.importObject,
        );
      } catch (streamErr) {
        var resp = await fetch('./gh-aw.wasm');
        var buf = await resp.arrayBuffer();
        result = await WebAssembly.instantiate(buf, go.importObject);
      }

      // Start the Go program. go.run() never resolves because main()
      // does `select{}`, so we intentionally do NOT await it.
      go.run(result.instance);

      // Poll until the Go code has registered compileWorkflow on globalThis.
      await waitForGlobal('compileWorkflow', 5000);

      ready = true;
      self.postMessage({ type: 'ready' });
    } catch (err) {
      self.postMessage({
        type: 'error',
        id: null,
        error: 'Worker initialization failed: ' + err.message,
      });
    }
  }

  /**
   * Poll for a global property to appear.
   */
  function waitForGlobal(name, timeoutMs) {
    return new Promise(function (resolve, reject) {
      var start = Date.now();
      (function check() {
        if (typeof self[name] !== 'undefined') {
          resolve();
        } else if (Date.now() - start > timeoutMs) {
          reject(new Error('Timed out waiting for globalThis.' + name));
        } else {
          setTimeout(check, 10);
        }
      })();
    });
  }

  /**
   * Handle incoming messages from the main thread.
   */
  self.onmessage = async function (event) {
    var msg = event.data;

    if (msg.type !== 'compile') {
      return;
    }

    var id = msg.id;

    if (!ready) {
      self.postMessage({
        type: 'error',
        id: id,
        error: 'Compiler is not ready yet.',
      });
      return;
    }

    if (typeof msg.markdown !== 'string') {
      self.postMessage({
        type: 'error',
        id: id,
        error: 'markdown must be a string.',
      });
      return;
    }

    try {
      // compileWorkflow returns a Promise (Go side).
      // Pass optional files object for import resolution.
      var files = msg.files || null;
      var result = await compileWorkflow(msg.markdown, files);

      // The Go function returns { yaml: string, warnings: Array, error: null|string }
      var warnings = [];
      if (result.warnings) {
        for (var i = 0; i < result.warnings.length; i++) {
          warnings.push(result.warnings[i]);
        }
      }

      if (result.error) {
        self.postMessage({
          type: 'error',
          id: id,
          error: String(result.error),
        });
      } else {
        self.postMessage({
          type: 'result',
          id: id,
          yaml: result.yaml || '',
          warnings: warnings,
          error: null,
        });
      }
    } catch (err) {
      self.postMessage({
        type: 'error',
        id: id,
        error: err.message || String(err),
      });
    }
  };

  // Start initialization immediately.
  init();
})();
