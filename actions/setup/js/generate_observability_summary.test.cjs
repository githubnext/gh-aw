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
        context: {
          episode_id: "episode-42",
          hop_id: "hop-2",
          parent_hop_id: "hop-1",
          origin_event: "workflow_run",
          root_repo: "owner/repo",
          root_workflow_id: "owner/repo/.github/workflows/root.yml@refs/heads/main",
          workflow_call_id: "12345678901-1",
          otel_trace_id: "a3f2c8d1e4b7091f6a5c2e3d8f401b72",
        },
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
    expect(summary).toContain("- **episode**: episode-42");
    expect(summary).toContain("- **hop**: hop-2");
    expect(summary).toContain("- **parent hop**: hop-1");
    expect(summary).toContain("- **origin event**: workflow_run");
    expect(summary).toContain("- **root repo**: owner/repo");
    expect(summary).toContain("- **root workflow**: owner/repo/.github/workflows/root.yml@refs/heads/main");
    expect(summary).not.toContain("12345678901-1");
    expect(summary).toContain("- **posture**: write-capable");
    expect(summary).toContain("- **created items**: 2");
    expect(summary).toContain("- **blocked requests**: 1");
    expect(summary).toContain("- **agent output errors**: 1");
    expect(summary).toContain("  - add_comment");
    expect(summary).toContain("  - create_issue");
    expect(mockCore.summary.write).toHaveBeenCalledTimes(1);
  });

  it("uses GITHUB_AW_OTEL_TRACE_ID env var when set (root-level workflow)", async () => {
    process.env.GITHUB_AW_OTEL_TRACE_ID = "deadbeef01234567deadbeef01234567";
    fs.writeFileSync(
      "/tmp/gh-aw/aw_info.json",
      JSON.stringify({
        workflow_name: "daily-workflow",
        engine_id: "copilot",
        staged: false,
        firewall_enabled: false,
        context: { workflow_call_id: "12345678901-1" },
      })
    );

    await module.main(mockCore);

    delete process.env.GITHUB_AW_OTEL_TRACE_ID;

    expect(mockCore.summary.addRaw).toHaveBeenCalledTimes(1);
    const summary = mockCore.summary.addRaw.mock.calls[0][0];
    expect(summary).toContain("- **trace id**: deadbeef01234567deadbeef01234567");
    expect(summary).not.toContain("- **trace id**: 12345678901-1");
    expect(summary).toContain("- **episode**: 12345678901-1");
    expect(summary).toContain("- **hop**: 12345678901-1");
  });

  it("does not show workflow_call_id as trace id when no OTLP trace ID is available", async () => {
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
    expect(summary).not.toContain("trace id");
    expect(summary).toContain("- **episode**: 12345678901-1");
    expect(summary).toContain("- **hop**: 12345678901-1");
  });

  it("falls back to legacy workflow_call_id for lineage when canonical fields are absent", async () => {
    fs.writeFileSync(
      "/tmp/gh-aw/aw_info.json",
      JSON.stringify({
        workflow_name: "triage-workflow",
        engine_id: "copilot",
        staged: false,
        firewall_enabled: false,
        context: { workflow_call_id: "legacy-hop-1", repo: "owner/repo", workflow_id: "owner/repo/.github/workflows/legacy.yml@refs/heads/main" },
      })
    );

    await module.main(mockCore);

    const summary = mockCore.summary.addRaw.mock.calls[0][0];
    expect(summary).toContain("- **episode**: legacy-hop-1");
    expect(summary).toContain("- **hop**: legacy-hop-1");
    expect(summary).toContain("- **root repo**: owner/repo");
    expect(summary).toContain("- **root workflow**: owner/repo/.github/workflows/legacy.yml@refs/heads/main");
  });

  it("always generates summary regardless of env vars", async () => {
    await module.main(mockCore);

    expect(mockCore.summary.addRaw).toHaveBeenCalledTimes(1);
    expect(mockCore.summary.write).toHaveBeenCalledTimes(1);
  });
});
