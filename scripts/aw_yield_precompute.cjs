#!/usr/bin/env node

const path = require("path");
const { runPythonScript } = require("./aw_yield_python_helper.cjs");

function runPrecompute({ workspace, workflows, out }) {
  runPythonScript({
    workspace,
    scriptPath: path.join(workspace, "scripts/aw_yield_precompute.py"),
    scriptArgs: ["--workflows", workflows, "--out", out],
    out,
    requiredFields: [
      { name: "workspace", value: workspace },
      { name: "workflows", value: workflows },
      { name: "out", value: out },
    ],
  });
}

module.exports = { runPrecompute };
