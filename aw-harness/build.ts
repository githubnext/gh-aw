/**
 * esbuild bundle script for aw_harness.
 *
 * Compiles src/index.ts → dist/aw_harness.cjs and copies to actions/setup/js/.
 * Run with: node --experimental-strip-types build.ts
 */

import { build } from "esbuild";
import { copyFileSync, mkdirSync } from "node:fs";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const __dirname = dirname(fileURLToPath(import.meta.url));

const outfile = resolve(__dirname, "dist/aw_harness.cjs");
const dest = resolve(__dirname, "../actions/setup/js/aw_harness.cjs");

await build({
  entryPoints: [resolve(__dirname, "src/index.ts")],
  bundle: true,
  platform: "node",
  target: "node24",
  format: "cjs",
  outfile,
  // Bundle all dependencies into the single .cjs (no runtime npm install in container)
  external: [],
  // Keep readable in Actions logs; inline sourcemap for CI debugging
  minify: false,
  sourcemap: "inline",
});

mkdirSync(dirname(dest), { recursive: true });
copyFileSync(outfile, dest);

console.log(`✓ Bundled: ${outfile}`);
console.log(`✓ Copied:  ${dest}`);
