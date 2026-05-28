#!/usr/bin/env node

const fs = require("fs");
const path = require("path");
const { execFileSync } = require("child_process");

function resolvePythonCommand(scriptName) {
  for (const command of [process.env.AW_YIELD_PYTHON, "python3"]) {
    if (!command) {
      continue;
    }
    try {
      execFileSync(command, ["--version"], { stdio: "ignore" });
      return command;
    } catch {}
  }
  throw new Error(`Unable to locate a Python interpreter for ${scriptName}`);
}

function runPythonScript({ workspace, scriptName, args, out }) {
  fs.mkdirSync(path.dirname(out), { recursive: true });
  execFileSync(resolvePythonCommand(scriptName), [path.join(workspace, "scripts", scriptName), ...args], {
    cwd: workspace,
    stdio: "inherit",
  });
}

module.exports = { resolvePythonCommand, runPythonScript };
