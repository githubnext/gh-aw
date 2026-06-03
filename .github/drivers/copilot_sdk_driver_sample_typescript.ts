import { spawnSync } from "node:child_process";

const builtInDriver = `${process.env.RUNNER_TEMP || "/tmp"}/gh-aw/actions/copilot_sdk_driver.cjs`;
const result = spawnSync("node", [builtInDriver, ...process.argv.slice(2)], {
  env: process.env,
  stdio: "inherit",
});

process.exit(result.status ?? 1);
