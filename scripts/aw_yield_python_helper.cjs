#!/usr/bin/env node

const fs = require("fs");
const path = require("path");
const { execFileSync } = require("child_process");

function resolvePythonCommand(scriptLabel) {
  for (const command of [process.env.AW_YIELD_PYTHON, "python3"]) {
    if (!command) {
      continue;
    }
    try {
      execFileSync(command, ["--version"], { stdio: "ignore" });
      return command;
    } catch {}
  }

  throw new Error(`Unable to locate a Python interpreter for ${scriptLabel}`);
}

function runPythonScript({ workspace, scriptPath, scriptArgs, out, requiredFields }) {
  for (const field of requiredFields) {
    if (!field.value) {
      throw new Error(`${requiredFields.map((entry) => entry.name).join(", ")} are required`);
    }
  }

  fs.mkdirSync(path.dirname(out), { recursive: true });
  execFileSync(resolvePythonCommand(path.basename(scriptPath)), [scriptPath, ...scriptArgs], {
    cwd: workspace,
    stdio: "inherit",
  });
}

module.exports = { runPythonScript };
