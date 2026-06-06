import { describe, it, expect } from "vitest";
import { createRequire } from "module";

const require = createRequire(import.meta.url);
const { countPermissionDeniedIssues, hasNumerousPermissionDeniedIssues, extractDeniedCommands, buildMissingToolPermissionIssuePayload } = require("./permission_denied_helpers.cjs");

describe("permission_denied_helpers.cjs", () => {
  it("counts repeated permission-denied signals", () => {
    const output = "permission denied\nEACCES: permission denied\nEPERM operation not permitted\npermissions denied";
    expect(countPermissionDeniedIssues(output)).toBe(5);
  });

  it("detects numerous permission-denied issues at threshold", () => {
    const output = "permission denied\npermission denied\npermission denied";
    expect(hasNumerousPermissionDeniedIssues(output)).toBe(true);
  });

  it("extracts denied commands from pipe-marked output", () => {
    const output = ["  \u2502 go version 2>&1", "  Permission denied", "  \u2502 ls /usr/local/go", "  Permission denied"].join("\n");
    expect(extractDeniedCommands(output)).toEqual(["go version 2>&1", "ls /usr/local/go"]);
  });

  it("extracts denied commands from Copilot SDK driver tool denial logs", () => {
    const output = [
      "[copilot-sdk-driver] [sdk-driver] permission denied by workflow tool permissions: shell(agenticworkflows --help 2>&1 | head -60)",
      "[copilot-sdk-driver] [sdk-driver] tool denial 1/5: permission denied: shell(agenticworkflows --help 2>&1 | head -60)",
      "[copilot-sdk-driver] [sdk-driver] permission denied by workflow tool permissions: read",
      "[copilot-sdk-driver] [sdk-driver] tool denial 3/5: permission denied: read",
    ].join("\n");
    expect(extractDeniedCommands(output)).toEqual(["shell(agenticworkflows --help 2>&1 | head -60)", "read"]);
  });

  it("builds missing_tool payload with default denied commands", () => {
    const payload = JSON.parse(buildMissingToolPermissionIssuePayload());
    expect(payload).toEqual({
      type: "missing_tool",
      tool: "tool/permission",
      reason: "missing tool/permission issue: numerous permission denied errors detected",
      alternatives: "Verify token scopes, repository permissions, and MCP/tool access configuration.",
      denied_commands: [],
    });
  });
});
