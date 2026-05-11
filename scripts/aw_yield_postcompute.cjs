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
  throw new Error("Unable to locate a Python interpreter for aw_yield_postcompute.py");
}

function runPostcompute({ workspace, precompute, agentOutput, out }) {
  if (!workspace || !precompute || !agentOutput || !out) {
    throw new Error("workspace, precompute, agentOutput, and out are required");
  }
  fs.mkdirSync(path.dirname(out), { recursive: true });
  execFileSync(
    resolvePythonCommand(),
    [
      path.join(workspace, "scripts/aw_yield_postcompute.py"),
      "--precompute",
      precompute,
      "--agent-output",
      agentOutput,
      "--out",
      out,
    ],
    {
      cwd: workspace,
      stdio: "inherit",
    }
  );
}

module.exports = { runPostcompute };
