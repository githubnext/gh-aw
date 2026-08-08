import { build } from "esbuild";

await build({
  bundle: true,
  entryPoints: ["src/index.ts"],
  format: "cjs",
  outfile: "dist/aw_harness.cjs",
  platform: "node",
  target: "node24",
});
