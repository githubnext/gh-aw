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
    for (const path of [
      "/tmp/gh-aw/aw_info.json",
      "/tmp/gh-aw/agent_output.json",
      "/tmp/gh-aw/otlp-export-errors.count",
      "/tmp/gh-aw/otlp-export-errors.jsonl",
      "/tmp/gh-aw/mcp-logs/gateway.jsonl",
      "/tmp/gh-aw/mcp-logs/rpc-messages.jsonl",
      "/tmp/gh-aw/sandbox/firewall/logs/squid-logs/access.log",
      "/tmp/gh-aw/sandbox/firewall/logs/access.log",
    ]) {
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
    fs.writeFileSync("/tmp/gh-aw/otlp-export-errors.count", "2\n");
    fs.writeFileSync(
      "/tmp/gh-aw/otlp-export-errors.jsonl",
      [JSON.stringify({ host: "collector-a.example.com:4318", status: 401, reason: "Unauthorized" }), JSON.stringify({ host: "collector-b.example.com:4318", reason: "upstream timeout" })].join("\n")
    );

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
    expect(summary).toContain("- **otlp export errors**: 2");
    expect(summary).toContain("- OTLP export failures detected; telemetry may not be visible in the backend.");
    expect(summary).toContain("- **otlp export failure details**:");
    expect(summary).toContain("  - collector-a.example.com:4318 status=401 reason=Unauthorized");
    expect(summary).toContain("  - collector-b.example.com:4318 reason=upstream timeout");
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
    expect(summary).toContain("- **otlp export errors**: 0");
    expect(summary).not.toContain("12345678901-1");
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
    expect(summary).not.toContain("12345678901-1");
  });

  it("always generates summary regardless of env vars", async () => {
    await module.main(mockCore);

    expect(mockCore.summary.addRaw).toHaveBeenCalledTimes(1);
    expect(mockCore.summary.write).toHaveBeenCalledTimes(1);
  });

  it("reports squid access.log present when firewall enabled and file exists at squid-logs path", async () => {
    fs.mkdirSync("/tmp/gh-aw/sandbox/firewall/logs/squid-logs", { recursive: true });
    fs.writeFileSync("/tmp/gh-aw/sandbox/firewall/logs/squid-logs/access.log", '1761332530.474 172.30.0.20:35288 api.github.com:443 140.82.112.22:443 1.1 CONNECT 200 TCP_TUNNEL:HIER_DIRECT api.github.com:443 "-"\n');
    fs.writeFileSync("/tmp/gh-aw/aw_info.json", JSON.stringify({ workflow_name: "firewall-workflow", firewall_enabled: true }));

    await module.main(mockCore);

    const summary = mockCore.summary.addRaw.mock.calls[0][0];
    expect(summary).toContain("- **squid access.log present**: true");
    expect(summary).not.toContain("egress traffic for this run cannot be audited");
  });

  it("reports squid access.log present when firewall enabled and file exists at legacy path", async () => {
    fs.mkdirSync("/tmp/gh-aw/sandbox/firewall/logs", { recursive: true });
    fs.writeFileSync("/tmp/gh-aw/sandbox/firewall/logs/access.log", '1761332530.474 172.30.0.20:35288 api.github.com:443 140.82.112.22:443 1.1 CONNECT 200 TCP_TUNNEL:HIER_DIRECT api.github.com:443 "-"\n');
    fs.writeFileSync("/tmp/gh-aw/aw_info.json", JSON.stringify({ workflow_name: "firewall-workflow", firewall_enabled: true }));

    await module.main(mockCore);

    const summary = mockCore.summary.addRaw.mock.calls[0][0];
    expect(summary).toContain("- **squid access.log present**: true");
    expect(summary).not.toContain("egress traffic for this run cannot be audited");
  });

  it("warns when firewall enabled but squid access.log is missing", async () => {
    fs.writeFileSync("/tmp/gh-aw/aw_info.json", JSON.stringify({ workflow_name: "firewall-workflow", firewall_enabled: true }));

    await module.main(mockCore);

    const summary = mockCore.summary.addRaw.mock.calls[0][0];
    expect(summary).toContain("- **squid access.log present**: false");
    expect(summary).toContain("Squid access.log not found; egress traffic for this run cannot be audited.");
  });

  it("omits squid access.log status when firewall is disabled", async () => {
    fs.writeFileSync("/tmp/gh-aw/aw_info.json", JSON.stringify({ workflow_name: "no-firewall-workflow", firewall_enabled: false }));

    await module.main(mockCore);

    const summary = mockCore.summary.addRaw.mock.calls[0][0];
    expect(summary).not.toContain("squid access.log");
    expect(summary).not.toContain("egress traffic for this run cannot be audited");
  });
});
