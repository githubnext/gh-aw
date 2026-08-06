import { describe, expect, it } from "vitest";
import { spawnSync } from "child_process";
import { join } from "path";

describe("core shim", () => {
  it("rejects setSecret outside the github-script runtime", () => {
    const shimPath = join(import.meta.dirname, "shim.cjs");
    const result = spawnSync(process.execPath, ["-e", `require(${JSON.stringify(shimPath)}); core.setSecret("derived-value");`], {
      encoding: "utf8",
    });

    expect(result.status).not.toBe(0);
    expect(result.stderr).toContain("core.setSecret is unavailable outside the github-script runtime");
  });

  it("adds a throwing setSecret to an existing partial core object", () => {
    const shimPath = join(import.meta.dirname, "shim.cjs");
    const result = spawnSync(process.execPath, ["-e", `global.core = { info() {} }; require(${JSON.stringify(shimPath)}); core.setSecret("derived-value");`], {
      encoding: "utf8",
    });

    expect(result.status).not.toBe(0);
    expect(result.stderr).toContain("core.setSecret is unavailable outside the github-script runtime");
  });
});
