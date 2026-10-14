import { describe, expect, it } from "vitest";
import { execFileSync } from "node:child_process";

describe("MCP rate-limit parser dependencies", () => {
  it("loads unified session normalization without initializing GitHub Actions globals", () => {
    const script = `
      const assert = require("node:assert/strict");
      delete global.core;
      delete global.context;
      require("./unified_session_payload.cjs");
      assert.equal(global.core, undefined);
      assert.equal(global.context, undefined);
      assert.equal(require.cache[require.resolve("./shim.cjs")], undefined);
    `;
    expect(() => execFileSync(process.execPath, ["-e", script], { cwd: new URL(".", import.meta.url), stdio: "pipe" })).not.toThrow();
  });
});
