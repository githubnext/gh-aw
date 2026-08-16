// @ts-check

"use strict";

const path = require("path");
const { spawnSync } = require("child_process");

describe("shell_harness.cjs", () => {
  const harnessPath = path.join(__dirname, "shell_harness.cjs");

  it("runs a shell command through the shared process runner", () => {
    const result = spawnSync(process.execPath, [harnessPath, "test", "printf hello"], { encoding: "utf8" });

    expect(result.status).toBe(0);
    expect(result.stdout).toBe("hello");
    expect(result.stderr).toContain("[test-harness] attempt 1: process closed");
  });

  it("reports invalid invocations", () => {
    const result = spawnSync(process.execPath, [harnessPath], { encoding: "utf8" });

    expect(result.status).toBe(2);
    expect(result.stderr).toContain("Usage:");
  });
});
