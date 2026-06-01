import fs from "fs";
import os from "os";
import path from "path";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

let exports;

describe("check_daily_effective_workflow_guardrail", () => {
  beforeEach(async () => {
    vi.resetModules();
    process.env.GITHUB_EVENT_NAME = "";
    process.env.GH_AW_WORKFLOW_DISPATCH_AW_CONTEXT = "";
    const mod = await import("./check_daily_effective_workflow_guardrail.cjs");
    exports = mod.default || mod;
  });

  afterEach(() => {
    delete process.env.GITHUB_EVENT_NAME;
    delete process.env.GH_AW_WORKFLOW_DISPATCH_AW_CONTEXT;
  });

  it("skips workflow_call, repository_dispatch, and workflow_dispatch with aw_context", () => {
    process.env.GITHUB_EVENT_NAME = "workflow_call";
    expect(exports.shouldSkipDailyEffectiveWorkflowGuardrail()).toBe(true);

    process.env.GITHUB_EVENT_NAME = "repository_dispatch";
    expect(exports.shouldSkipDailyEffectiveWorkflowGuardrail()).toBe(true);

    process.env.GITHUB_EVENT_NAME = "workflow_dispatch";
    process.env.GH_AW_WORKFLOW_DISPATCH_AW_CONTEXT = '{"item_number":123}';
    expect(exports.shouldSkipDailyEffectiveWorkflowGuardrail()).toBe(true);

    process.env.GH_AW_WORKFLOW_DISPATCH_AW_CONTEXT = "";
    expect(exports.shouldSkipDailyEffectiveWorkflowGuardrail()).toBe(false);
  });

  it("matches both firewall-audit-logs and unified agent artifacts", () => {
    expect(exports.matchesGuardrailArtifactName("firewall-audit-logs")).toBe(true);
    expect(exports.matchesGuardrailArtifactName("agent")).toBe(true);
    expect(exports.matchesGuardrailArtifactName("prefix-firewall-audit-logs")).toBe(true);
    expect(exports.matchesGuardrailArtifactName("prefix-agent")).toBe(true);
    expect(exports.matchesGuardrailArtifactName("activation")).toBe(false);
  });

  it("sums effective tokens from explicit token-usage entries", () => {
    const tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), "daily-guardrail-token-usage-"));
    const filePath = path.join(tmpDir, "token-usage.jsonl");
    fs.writeFileSync(filePath, [JSON.stringify({ model: "gpt-5.5", effective_tokens: 125 }), JSON.stringify({ model: "gpt-5.5", effective_tokens: 75 })].join("\n"), "utf8");

    expect(exports.sumEffectiveTokensFromTokenUsageFile(filePath)).toBe(200);
  });

  it("computes aggregate ET statistics for prior runs", () => {
    expect(exports.calculateDailyEffectiveWorkflowStats([{ effective_tokens: 100 }, { effective_tokens: 200 }, { effective_tokens: 300 }])).toEqual({
      count: 3,
      total: 600,
      average: 200,
      min: 100,
      max: 300,
      stddev: 100,
    });
  });

  it("caps inspection when GitHub API rate limit headroom is low", () => {
    expect(exports.computeMaxInspectableRuns(110)).toBe(0);
    expect(exports.computeMaxInspectableRuns(120)).toBeGreaterThan(0);
  });

  it("formats structured daily ET log messages", () => {
    const message = exports.formatDailyGuardrailLogMessage("Resolved current workflow ET guardrail context", {
      currentRunId: 123,
      workflowId: 456,
      currentEffectiveTokens: 789,
    });
    const prefix = "[daily-workflow-et] Resolved current workflow ET guardrail context: ";
    expect(message).toContain(prefix);
    expect(JSON.parse(message.slice(prefix.length))).toEqual({
      currentRunId: 123,
      workflowId: 456,
      currentEffectiveTokens: 789,
    });
    expect(exports.formatDailyGuardrailLogMessage("Completed ET inspection window")).toBe("[daily-workflow-et] Completed ET inspection window");
  });

  it("renders a daily ET details summary with stats and prior runs", () => {
    const markdown = exports.renderDailyEffectiveWorkflowSummary(
      "Nightly triage",
      "copilot-swe-agent[bot]",
      1_500_000,
      [
        {
          id: 11,
          html_url: "https://example.test/runs/11",
          created_at: "2026-05-31T10:00:00Z",
          conclusion: "success",
          effective_tokens: 1_200_000,
        },
        {
          id: 10,
          html_url: "https://example.test/runs/10",
          created_at: "2026-05-31T09:00:00Z",
          conclusion: "failure",
          effective_tokens: 300_000,
        },
      ],
      {
        remaining: 4321,
        limit: 5000,
        used: 679,
        reset: "2026-05-31T12:00:00.000Z",
      },
      {
        candidateRunsCount: 5,
        inspectedRunsCount: 2,
        truncatedByRateLimit: true,
      }
    );

    expect(markdown).toContain("| 24h total ET | 1.5M |");
    expect(markdown).toContain("| Threshold | 1.5M |");
    expect(markdown).toContain("| Avg ET / run | 750K |");
    expect(markdown).toContain("| Std dev ET | 636.4K |");
    expect(markdown).toContain("| [#11](https://example.test/runs/11) | 2026-05-31T10:00:00Z | success | 1.2M |");
    expect(markdown).toContain("Stopped early to preserve GitHub API rate limit headroom");
    expect(markdown).not.toContain("Guardrail issue:");
  });
});
