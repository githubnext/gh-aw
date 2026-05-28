#!/usr/bin/env node

const assert = require("assert");
const fs = require("fs");
const os = require("os");
const path = require("path");

const { runPythonScript } = require("./aw_yield_shared.cjs");

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

console.log("✓ aw_yield_shared.cjs runs Python scripts with the workspace cwd and creates the output directory");
