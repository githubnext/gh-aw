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
    fs.writeFileSync(
      filePath,
      [
        JSON.stringify({ model: "gpt-5.5", effective_tokens: 125 }),
        JSON.stringify({ model: "gpt-5.5", effective_tokens: 75 }),
      ].join("\n"),
      "utf8"
    );

    expect(exports.sumEffectiveTokensFromTokenUsageFile(filePath)).toBe(200);
  });
});
