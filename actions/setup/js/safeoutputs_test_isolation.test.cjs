import { describe, expect, it } from "vitest";
import { spawnSync } from "node:child_process";
import fs from "node:fs";
import path from "node:path";

describe("test safeoutputs isolation", () => {
  it.each([false, true])("persists child diagnostics locally with the CLI override removed: %s", removeOverride => {
    const outputPath = path.join(path.dirname(process.env.GH_AW_SAFEOUTPUTS_CLI), `child-${removeOverride}.jsonl`);
    const env = { ...process.env, GH_AW_SAFE_OUTPUTS: outputPath };
    if (removeOverride) delete env.GH_AW_SAFEOUTPUTS_CLI;
    const result = spawnSync(
      process.execPath,
      [
        "-e",
        `const {emitInfrastructureIncomplete, emitMissingToolPermissionIssue} = require("./safeoutputs_cli.cjs");
emitInfrastructureIncomplete("child fixture failure");
emitMissingToolPermissionIssue({deniedCommands: ["test command"]});`,
      ],
      { env, encoding: "utf8", timeout: 10000 }
    );
    expect(result.status, result.stderr).toBe(0);
    expect(result.stderr).not.toContain("emission failed");
    const items = fs
      .readFileSync(outputPath, "utf8")
      .trim()
      .split("\n")
      .map(line => JSON.parse(line));
    expect(items).toEqual([
      { type: "report_incomplete", reason: "infrastructure_error", details: "child fixture failure" },
      {
        type: "report_missing_tool",
        tool: "tool/permission",
        reason: "missing tool/permission issue: numerous permission denied errors detected",
        alternatives: expect.stringContaining("Denied commands: test command"),
      },
    ]);
  });

  it("fails explicitly rather than discarding a diagnostic without a destination", () => {
    const result = spawnSync(process.env.GH_AW_SAFEOUTPUTS_CLI, ["report_incomplete", "--details", "fixture failure"], {
      env: { ...process.env, GH_AW_SAFE_OUTPUTS: "" },
      encoding: "utf8",
      timeout: 10000,
    });
    expect(result.status).not.toBe(0);
    expect(result.stderr).toContain("Test safeoutputs requires GH_AW_SAFE_OUTPUTS");
  });
});
