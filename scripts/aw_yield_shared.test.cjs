#!/usr/bin/env node

const assert = require("assert");
const fs = require("fs");
const os = require("os");
const path = require("path");

const { resolvePythonCommand, runPythonScript } = require("./aw_yield_shared.cjs");

const workspace = fs.mkdtempSync(path.join(os.tmpdir(), "aw-yield-shared-"));
const scriptsDir = path.join(workspace, "scripts");
const out = path.join(workspace, "tmp", "nested", "out.json");

fs.mkdirSync(scriptsDir, { recursive: true });
fs.writeFileSync(
  path.join(scriptsDir, "emit_context.py"),
  `#!/usr/bin/env python3
import json
import os
import sys
from pathlib import Path

argv = sys.argv[1:]
out = Path(argv[argv.index("--out") + 1])
out.write_text(json.dumps({"cwd": os.getcwd(), "argv": argv}) + "\\n", encoding="utf-8")
`,
  "utf8"
);

runPythonScript({
  workspace,
  scriptName: "emit_context.py",
  args: ["--flag", "value", "--out", out],
  out,
});

const payload = JSON.parse(fs.readFileSync(out, "utf8"));
assert.equal(payload.cwd, workspace);
assert.deepEqual(payload.argv, ["--flag", "value", "--out", out]);
assert.throws(
  () => runPythonScript({ workspace, scriptName: "emit_context.py", out }),
  /workspace, scriptName, args, and out are required/
);

const originalPath = process.env.PATH;
const originalPython = process.env.AW_YIELD_PYTHON;

try {
  process.env.PATH = "";
  process.env.AW_YIELD_PYTHON = "definitely-not-a-python";
  assert.throws(() => resolvePythonCommand("emit_context.py"), /Unable to locate a Python interpreter/);
} finally {
  if (originalPath === undefined) {
    delete process.env.PATH;
  } else {
    process.env.PATH = originalPath;
  }
  if (originalPython === undefined) {
    delete process.env.AW_YIELD_PYTHON;
  } else {
    process.env.AW_YIELD_PYTHON = originalPython;
  }
}

console.log("✓ aw_yield_shared.cjs runs Python scripts with the workspace cwd and creates the output directory");
