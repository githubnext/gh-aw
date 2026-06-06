// @ts-check
//
// apply_samples.test.cjs
//
// Smoke test for the deterministic samples replay driver. Spawns the
// driver as a subprocess (so it actually launches the real MCP server) and
// asserts that:
//   - the driver exits 0
//   - the MCP server appends the expected JSONL entry to GH_AW_SAFE_OUTPUTS
//   - the synthetic agent-stdio log includes a `terminal_reason: completed` marker
//
// Tests intentionally use the simplest safe-output tool (`create_issue`) so we
// do not need to set up a git working tree for patch sidecars.

import { describe, it, expect, beforeAll } from "vitest";
import { spawnSync } from "child_process";
import fs from "fs";
import path from "path";
import os from "os";
import { fileURLToPath } from "url";

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

const driverPath = path.join(__dirname, "apply_samples.cjs");

function makeTempDir(prefix) {
  return fs.mkdtempSync(path.join(os.tmpdir(), prefix));
}

describe.sequential("apply_samples.cjs", () => {
  let tempDir;
  let configPath;
  let outputsPath;
  let logPath;

  beforeAll(() => {
    tempDir = makeTempDir("gh-aw-apply-samples-");
    configPath = path.join(tempDir, "config.json");
    outputsPath = path.join(tempDir, "outputs.jsonl");
    logPath = path.join(tempDir, "agent-stdio.log");

    // Minimal safe-outputs config enabling only the `create_issue` tool. The
    // bootstrap loader keys off the snake-case keys present here.
    fs.writeFileSync(
      configPath,
      JSON.stringify({
        create_issue: { max: 1 },
      })
    );
  });

  it("replays a create_issue sample through the real MCP server and emits a completed marker", () => {
    const samples = [
      {
        tool: "create_issue",
        arguments: {
          title: "Deterministic sample issue",
          body: "This issue was emitted by the apply_samples driver during a unit test.",
        },
      },
    ];

    const result = spawnSync(process.execPath, [driverPath], {
      env: {
        ...process.env,
        GH_AW_SAMPLES: JSON.stringify(samples),
        GH_AW_SAFE_OUTPUTS_CONFIG_PATH: configPath,
        GH_AW_SAFE_OUTPUTS: outputsPath,
        GH_AW_AGENT_STDIO_LOG: logPath,
      },
      encoding: "utf8",
      timeout: 15000,
    });

    if (result.status !== 0) {
      // Surface stderr so failures are diagnosable in CI.
      throw new Error(`driver exited with status ${result.status}\nstderr:\n${result.stderr}\nstdout:\n${result.stdout}`);
    }

    expect(fs.existsSync(outputsPath)).toBe(true);
    const outputLines = fs
      .readFileSync(outputsPath, "utf8")
      .split("\n")
      .filter(line => line.trim().length > 0);
    expect(outputLines.length).toBeGreaterThanOrEqual(1);

    const firstEntry = JSON.parse(outputLines[0]);
    expect(firstEntry.type).toBe("create_issue");
    expect(firstEntry.title).toBe("Deterministic sample issue");

    expect(fs.existsSync(logPath)).toBe(true);
    const logText = fs.readFileSync(logPath, "utf8");
    expect(logText).toContain("terminal_reason");
    expect(logText).toContain("completed");
  });

  it("exits cleanly when GH_AW_SAMPLES is empty", () => {
    const result = spawnSync(process.execPath, [driverPath], {
      env: {
        ...process.env,
        GH_AW_SAMPLES: "[]",
        GH_AW_SAFE_OUTPUTS_CONFIG_PATH: configPath,
        GH_AW_SAFE_OUTPUTS: outputsPath,
        GH_AW_AGENT_STDIO_LOG: path.join(tempDir, "empty-log.log"),
      },
      encoding: "utf8",
      timeout: 10000,
    });

    expect(result.status).toBe(0);
    const logText = fs.readFileSync(path.join(tempDir, "empty-log.log"), "utf8");
    expect(logText).toContain("terminal_reason");
  });
});
