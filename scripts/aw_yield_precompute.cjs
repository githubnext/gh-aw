#!/usr/bin/env node

const fs = require("fs");
const path = require("path");
const { execFileSync } = require("child_process");

function resolvePythonCommand() {
  for (const command of [process.env.AW_YIELD_PYTHON, "python3"]) {
    if (!command) {
      continue;
    }
    try {
      execFileSync(command, ["--version"], { stdio: "ignore" });
      return command;
    } catch {}
  }
  throw new Error("Unable to locate a Python interpreter for aw_yield_precompute.py");
}

function runPrecompute({ workspace, workflows, out }) {
  if (!workspace || !workflows || !out) {
    throw new Error("workspace, workflows, and out are required");
  }
  fs.mkdirSync(path.dirname(out), { recursive: true });
  execFileSync(
    resolvePythonCommand(),
    [path.join(workspace, "scripts/aw_yield_precompute.py"), "--workflows", workflows, "--out", out],
    {
      cwd: workspace,
      stdio: "inherit",
    }
  );
}

module.exports = { runPrecompute };
