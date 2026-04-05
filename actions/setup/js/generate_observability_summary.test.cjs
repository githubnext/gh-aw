import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import fs from "fs";

const mockCore = {
  info: vi.fn(),
  summary: {
    addRaw: vi.fn().mockReturnThis(),
    write: vi.fn().mockResolvedValue(),
  },
};

global.core = mockCore;

describe("generate_observability_summary.cjs", () => {
  let module;

  beforeEach(async () => {
    vi.clearAllMocks();
    fs.mkdirSync("/tmp/gh-aw/mcp-logs", { recursive: true });
    module = await import("./generate_observability_summary.cjs");
  });

  afterEach(() => {
    for (const path of ["/tmp/gh-aw/aw_info.json", "/tmp/gh-aw/agent_output.json", "/tmp/gh-aw/mcp-logs/gateway.jsonl", "/tmp/gh-aw/mcp-logs/rpc-messages.jsonl"]) {
      if (fs.existsSync(path)) {
        fs.unlinkSync(path);
      }
    }
  });

  it("builds summary from runtime observability files", async () => {
    fs.writeFileSync(
      "/tmp/gh-aw/aw_info.json",
      JSON.stringify({
        workflow_name: "triage-workflow",
        engine_id: "copilot",
        staged: false,
        firewall_enabled: true,
        context: { workflow_call_id: "12345678901-1", otel_trace_id: "a3f2c8d1e4b7091f6a5c2e3d8f401b72" },
      })
    );
    fs.writeFileSync(
      "/tmp/gh-aw/agent_output.json",
      JSON.stringify({
        items: [{ type: "create_issue" }, { type: "add_comment" }],
        errors: ["validation failed"],
      })
    );
    fs.writeFileSync("/tmp/gh-aw/mcp-logs/gateway.jsonl", [JSON.stringify({ type: "DIFC_FILTERED" }), JSON.stringify({ type: "REQUEST" })].join("\n"));

    await module.main(mockCore);

    expect(mockCore.summary.addRaw).toHaveBeenCalledTimes(1);
    const summary = mockCore.summary.addRaw.mock.calls[0][0];
    expect(summary).toContain("<summary>Observability</summary>");
    expect(summary).toContain("- **workflow**: triage-workflow");
    expect(summary).toContain("- **engine**: copilot");
    expect(summary).toContain("- **trace id**: a3f2c8d1e4b7091f6a5c2e3d8f401b72");
    expect(summary).not.toContain("12345678901-1");
    expect(summary).toContain("- **posture**: write-capable");
    expect(summary).toContain("- **created items**: 2");
    expect(summary).toContain("- **blocked requests**: 1");
    expect(summary).toContain("- **agent output errors**: 1");
    expect(summary).toContain("  - add_comment");
    expect(summary).toContain("  - create_issue");
    expect(mockCore.summary.write).toHaveBeenCalledTimes(1);
  });

  it("falls back to workflow_call_id when otel_trace_id is absent", async () => {
    fs.writeFileSync(
      "/tmp/gh-aw/aw_info.json",
      JSON.stringify({
        workflow_name: "triage-workflow",
        engine_id: "copilot",
        staged: false,
        firewall_enabled: false,
        context: { workflow_call_id: "12345678901-1" },
      })
    );

    await module.main(mockCore);

    expect(mockCore.summary.addRaw).toHaveBeenCalledTimes(1);
    const summary = mockCore.summary.addRaw.mock.calls[0][0];
    expect(summary).toContain("- **trace id**: 12345678901-1");
  });

  it("always generates summary regardless of env vars", async () => {
    await module.main(mockCore);

    expect(mockCore.summary.addRaw).toHaveBeenCalledTimes(1);
    expect(mockCore.summary.write).toHaveBeenCalledTimes(1);
  });
});
