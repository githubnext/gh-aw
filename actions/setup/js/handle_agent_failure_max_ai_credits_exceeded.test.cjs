// @ts-check
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import path from "path";
import { fileURLToPath } from "url";

let buildAICreditsRateLimitErrorContext;
const __dirname = path.dirname(fileURLToPath(import.meta.url));

describe("handle_agent_failure Max AI Credits exceeded context", () => {
  beforeEach(async () => {
    vi.resetModules();
    process.env.GH_AW_PROMPTS_DIR = path.join(__dirname, "../md");
    const mod = await import("./handle_agent_failure.cjs");
    const exports = mod.default || mod;
    buildAICreditsRateLimitErrorContext = exports.buildAICreditsRateLimitErrorContext;
  });

  afterEach(() => {
    vi.restoreAllMocks();
    delete process.env.GH_AW_PROMPTS_DIR;
  });

  it("shows budget exhaustion message with usage, limit, and overage details", () => {
    const rendered = buildAICreditsRateLimitErrorContext(true, "105000", "100000", "https://github.com/octo/repo/actions/runs/456");

    expect(rendered).toContain("AI Credits Budget Exceeded");
    expect(rendered).toContain("hit the configured `max-ai-credits` guardrail");
    expect(rendered).toContain("| AI credits used |");
    expect(rendered).toContain("| Guardrail limit (`max-ai-credits`) |");
    expect(rendered).toContain("| Over the limit by |");
    expect(rendered).toContain("| Run | [View workflow run](https://github.com/octo/repo/actions/runs/456) |");
    expect(rendered).toContain("<details>");
    expect(rendered).toContain("<summary>Tips for reducing AI credit usage</summary>");
    expect(rendered).toContain("https://github.github.com/gh-aw/reference/cost-management/");
  });

  it("shows message without metrics rows when no credit data is available", () => {
    const rendered = buildAICreditsRateLimitErrorContext(true, "", "", "");

    expect(rendered).toContain("AI Credits Budget Exceeded");
    expect(rendered).not.toContain("| AI credits used |");
    expect(rendered).not.toContain("| Guardrail limit");
    expect(rendered).not.toContain("| Run |");
  });

  it("does not show overage row when usage does not exceed limit", () => {
    const rendered = buildAICreditsRateLimitErrorContext(true, "50000", "100000", "");

    expect(rendered).toContain("AI Credits Budget Exceeded");
    expect(rendered).toContain("| AI credits used |");
    expect(rendered).toContain("| Guardrail limit (`max-ai-credits`) |");
    expect(rendered).not.toContain("| Over the limit by |");
  });

  it("returns empty string when max_ai_credits_exceeded is false", () => {
    expect(buildAICreditsRateLimitErrorContext(false, "105000", "100000", "https://github.com/octo/repo/actions/runs/456")).toBe("");
  });
});
