import { describe, expect, it } from "vitest";
import { spawnSync } from "child_process";
import { join } from "path";

describe("core shim", () => {
  it("does not emit derived secrets when registering them", () => {
    const shimPath = join(import.meta.dirname, "shim.cjs");
    const result = spawnSync(process.execPath, ["-e", `require(${JSON.stringify(shimPath)}); core.setSecret("derived%secret\\r\\nvalue");`], {
      encoding: "utf8",
    });

    expect(result.status).toBe(0);
    expect(result.stderr).toBe("");
  });

  it("masks registered secrets in every future shim output", () => {
    const shimPath = join(import.meta.dirname, "shim.cjs");
    const script = `
      require(${JSON.stringify(shimPath)});
      core.setSecret("derived.secret");
      core.debug("debug derived.secret derived.secret");
      core.info("info derived.secret");
      core.notice("notice derived.secret");
      core.warning("warning derived.secret");
      core.error("error derived.secret");
      core.setOutput("derived.secret-name", "derived.secret-value");
      core.setFailed("failed derived.secret");
    `;
    const result = spawnSync(process.execPath, ["-e", script], {
      encoding: "utf8",
    });

    expect(result.status).toBe(1);
    expect(result.stderr).toBe(["[debug] debug *** ***", "[info] info ***", "[notice] notice ***", "[warning] warning ***", "[error] error ***", "[output] ***-name=***-value", "[error] failed ***", ""].join("\n"));
  });

  it("masks overlapping secrets without revealing suffixes", () => {
    const shimPath = join(import.meta.dirname, "shim.cjs");
    const script = `
      require(${JSON.stringify(shimPath)});
      core.setSecret("token");
      core.setSecret("token-with-suffix");
      core.info("token-with-suffix token");
    `;
    const result = spawnSync(process.execPath, ["-e", script], {
      encoding: "utf8",
    });

    expect(result.status).toBe(0);
    expect(result.stderr).toBe("[info] *** ***\n");
  });

  it("masks self-overlapping secrets without revealing suffixes", () => {
    const shimPath = join(import.meta.dirname, "shim.cjs");
    const script = `
      require(${JSON.stringify(shimPath)});
      core.setSecret("aaa");
      core.info("aaaa");
    `;
    const result = spawnSync(process.execPath, ["-e", script], {
      encoding: "utf8",
    });

    expect(result.status).toBe(0);
    expect(result.stderr).toBe("[info] ***\n");
  });

  it("ignores empty secrets", () => {
    const shimPath = join(import.meta.dirname, "shim.cjs");
    const result = spawnSync(process.execPath, ["-e", `require(${JSON.stringify(shimPath)}); core.setSecret(""); core.info("visible");`], {
      encoding: "utf8",
    });

    expect(result.status).toBe(0);
    expect(result.stderr).toBe("[info] visible\n");
  });

  it("adds setSecret to an existing partial core object", () => {
    const shimPath = join(import.meta.dirname, "shim.cjs");
    const result = spawnSync(process.execPath, ["-e", `global.core = { info() {} }; require(${JSON.stringify(shimPath)}); core.setSecret("derived-value");`], {
      encoding: "utf8",
    });

    expect(result.status).toBe(0);
    expect(result.stderr).toBe("");
  });
});
