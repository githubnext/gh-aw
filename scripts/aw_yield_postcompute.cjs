#!/usr/bin/env node

const path = require("path");
const { runPythonScript } = require("./aw_yield_python_helper.cjs");

function runPostcompute({ workspace, precompute, agentOutput, out }) {
  runPythonScript({
    workspace,
    scriptPath: path.join(workspace, "scripts/aw_yield_postcompute.py"),
    scriptArgs: ["--precompute", precompute, "--agent-output", agentOutput, "--out", out],
    out,
    requiredFields: [
      { name: "workspace", value: workspace },
      { name: "precompute", value: precompute },
      { name: "agentOutput", value: agentOutput },
      { name: "out", value: out },
    ],
  });
}

module.exports = { runPostcompute };
