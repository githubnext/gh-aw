#!/usr/bin/env node

const { runPythonScript } = require("./aw_yield_shared.cjs");

function runPrecompute({ workspace, workflows, out }) {
  if (!workspace || !workflows || !out) {
    throw new Error("workspace, workflows, and out are required");
  }
  runPythonScript({
    workspace,
    scriptName: "aw_yield_precompute.py",
    args: ["--workflows", workflows, "--out", out],
    out,
  });
}

module.exports = { runPrecompute };
