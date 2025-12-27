---
"gh-aw": patch
---

Configure jsweep workflow to use Node.js v20 and compile JavaScript to CommonJS.

This change documents that `jsweep.md` pins `runtimes.node.version: "20"` and
updates `actions/setup/js/tsconfig.json` to emit CommonJS (`module: commonjs`) and
target ES2020 (`target: es2020`) for the JavaScript files in `actions/setup/js/`.

