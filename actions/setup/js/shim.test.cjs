import { describe, expect, it } from "vitest";
import { spawnSync } from "child_process";
import { join } from "path";

describe("core shim", () => {
  it("emits an escaped add-mask workflow command for derived secrets", () => {
    const shimPath = join(import.meta.dirname, "shim.cjs");
    const result = spawnSync(process.execPath, ["-e", `require(${JSON.stringify(shimPath)}); core.setSecret("derived%secret\\r\\nvalue");`], {
      encoding: "utf8",
    });

    expect(result.status).toBe(0);
    expect(result.stderr).toBe("::add-mask::derived%25secret%0D%0Avalue\n");
  });

  it("adds setSecret to an existing partial core object", () => {
    const shimPath = join(import.meta.dirname, "shim.cjs");
    const result = spawnSync(process.execPath, ["-e", `global.core = { info() {} }; require(${JSON.stringify(shimPath)}); core.setSecret("derived-value");`], {
      encoding: "utf8",
    });

    expect(result.status).toBe(0);
    expect(result.stderr).toBe("::add-mask::derived-value\n");
  });
});
