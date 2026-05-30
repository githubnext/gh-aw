import { describe, it, expect } from "vitest";
import { createRequire } from "module";

const require = createRequire(import.meta.url);
const { runSafeOutputsCLI, buildMissingToolAlternatives, emitMissingToolPermissionIssue, emitInfrastructureIncomplete } = require("./safeoutputs_cli.cjs");

describe("safeoutputs_cli.cjs", () => {
  describe("runSafeOutputsCLI", () => {
    it("throws when an argument key contains invalid characters", () => {
      process.env.GH_AW_SAFEOUTPUTS_CLI = "true"; // 'true' exits 0, accepts any args
      expect(() => runSafeOutputsCLI("missing_tool", { "bad key!": "value" })).toThrow("invalid safeoutputs argument key");
      delete process.env.GH_AW_SAFEOUTPUTS_CLI;
    });

    it("skips empty string values", () => {
      // 'true' succeeds without checking args, so empty-value keys must be skipped silently
      process.env.GH_AW_SAFEOUTPUTS_CLI = "true";
      expect(() => runSafeOutputsCLI("missing_tool", { tool: "", reason: "test" })).not.toThrow();
      delete process.env.GH_AW_SAFEOUTPUTS_CLI;
    });

    it("wraps CLI errors with tool name and key summary", () => {
      process.env.GH_AW_SAFEOUTPUTS_CLI = "false"; // 'false' exits 1
      expect(() => runSafeOutputsCLI("missing_tool", { tool: "tool/permission", reason: "test" })).toThrow("safeoutputs missing_tool(tool, reason) failed");
      delete process.env.GH_AW_SAFEOUTPUTS_CLI;
    });
  });

  describe("buildMissingToolAlternatives", () => {
    const BASE = "Verify token scopes, repository permissions, and MCP/tool access configuration.";

    it("returns base alternatives unchanged when denied command list is empty", () => {
      expect(buildMissingToolAlternatives(BASE, [])).toBe(BASE);
    });

    it("returns base alternatives unchanged when denied commands is null", () => {
      expect(buildMissingToolAlternatives(BASE, null)).toBe(BASE);
    });

    it("appends denied commands when list is non-empty", () => {
      const result = buildMissingToolAlternatives(BASE, ["go version"]);
      expect(result).toContain("Denied commands: go version");
    });

    it("caps result to 512 characters", () => {
      const base = "base";
      const deniedCommands = Array.from({ length: 30 }, (_, i) => `command-${i}-${"x".repeat(30)}`);
      const result = buildMissingToolAlternatives(base, deniedCommands);
      expect(result.length).toBeLessThanOrEqual(512);
    });

    it("includes overflow marker when commands are truncated", () => {
      const base = "base";
      const deniedCommands = Array.from({ length: 30 }, (_, i) => `command-${i}-${"x".repeat(30)}`);
      const result = buildMissingToolAlternatives(base, deniedCommands);
      expect(result).toContain("... and");
      expect(result).toContain("more");
    });

    it("truncates a base alternatives string that already exceeds 512 chars", () => {
      const longBase = "x".repeat(600);
      const result = buildMissingToolAlternatives(longBase, ["go version"]);
      expect(result.length).toBe(512);
    });
  });

  describe("emitMissingToolPermissionIssue", () => {
    it("invokes safeoutputs CLI with correct tool and reason", () => {
      const calls = [];
      const logs = [];
      emitMissingToolPermissionIssue({
        safeOutputsPath: "/tmp/safeoutputs.jsonl",
        deniedCommands: ["go version"],
        runSafeOutputsCLI: (toolName, args) => calls.push({ toolName, args }),
        logger: message => logs.push(message),
      });
      expect(calls).toHaveLength(1);
      expect(calls[0].toolName).toBe("missing_tool");
      expect(calls[0].args.tool).toBe("tool/permission");
      expect(calls[0].args.reason).toContain("missing tool/permission issue");
      expect(calls[0].args.alternatives).toContain("Denied commands: go version");
      expect(logs.some(m => m.includes("missing_tool emitted"))).toBe(true);
    });

    it("skips emission when safeOutputsPath is empty", () => {
      const calls = [];
      const logs = [];
      emitMissingToolPermissionIssue({
        safeOutputsPath: "",
        runSafeOutputsCLI: () => calls.push("call"),
        logger: message => logs.push(message),
      });
      expect(calls).toHaveLength(0);
      expect(logs.some(m => m.includes("skipped"))).toBe(true);
    });

    it("logs error when CLI invocation fails", () => {
      const logs = [];
      emitMissingToolPermissionIssue({
        safeOutputsPath: "/tmp/safeoutputs.jsonl",
        runSafeOutputsCLI: () => {
          throw new Error("EROFS: read-only file system");
        },
        logger: message => logs.push(message),
      });
      expect(logs.some(m => m.includes("missing_tool emission failed"))).toBe(true);
      expect(logs.some(m => m.includes("EROFS"))).toBe(true);
    });
  });

  describe("emitInfrastructureIncomplete", () => {
    it("invokes safeoutputs CLI with reason and details", () => {
      const calls = [];
      const logs = [];
      emitInfrastructureIncomplete("temporary outage", {
        safeOutputsPath: "/tmp/safeoutputs.jsonl",
        runSafeOutputsCLI: (toolName, args) => calls.push({ toolName, args }),
        logger: message => logs.push(message),
      });
      expect(calls).toEqual([
        {
          toolName: "report_incomplete",
          args: { reason: "infrastructure_error", details: "temporary outage" },
        },
      ]);
      expect(logs.some(m => m.includes("report_incomplete emitted"))).toBe(true);
    });

    it("skips emission when safeOutputsPath is empty", () => {
      const calls = [];
      const logs = [];
      emitInfrastructureIncomplete("temporary outage", {
        safeOutputsPath: "",
        runSafeOutputsCLI: () => calls.push("call"),
        logger: message => logs.push(message),
      });
      expect(calls).toHaveLength(0);
      expect(logs.some(m => m.includes("skipped"))).toBe(true);
    });

    it("logs error when CLI invocation fails", () => {
      const logs = [];
      emitInfrastructureIncomplete("temporary outage", {
        safeOutputsPath: "/tmp/safeoutputs.jsonl",
        runSafeOutputsCLI: () => {
          throw new Error("EROFS: read-only file system");
        },
        logger: message => logs.push(message),
      });
      expect(logs.some(m => m.includes("report_incomplete emission failed"))).toBe(true);
      expect(logs.some(m => m.includes("EROFS"))).toBe(true);
    });
  });
});
