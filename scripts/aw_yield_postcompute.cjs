#!/usr/bin/env node

const { runPythonScript } = require("./aw_yield_shared.cjs");

function runPostcompute({ workspace, precompute, agentOutput, out }) {
  if (!workspace || !precompute || !agentOutput || !out) {
    throw new Error("workspace, precompute, agentOutput, and out are required");
  }
  runPythonScript({
    workspace,
    scriptName: "aw_yield_postcompute.py",
    args: ["--precompute", precompute, "--agent-output", agentOutput, "--out", out],
    out,
  });
}

module.exports = { runPostcompute };
