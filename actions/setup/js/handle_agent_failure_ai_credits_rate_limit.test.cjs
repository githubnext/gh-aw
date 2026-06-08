import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import path from "path";
import { fileURLToPath } from "url";

let buildAICreditsRateLimitErrorContext;
const __dirname = path.dirname(fileURLToPath(import.meta.url));

describe("handle_agent_failure AI Credits rate-limit context", () => {
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

  it("shows actual usage, guardrail details, and collapsible optimization guidance", () => {
    const rendered = buildAICreditsRateLimitErrorContext(true, "17.329230000000003", "10.1", "https://github.com/octo/repo/actions/runs/123");

    expect(rendered).toContain("AI Credits Budget Exceeded");
    expect(rendered).toContain("| AI credits used | `17.3` |");
    expect(rendered).toContain("| Guardrail limit (`max-ai-credits`) | `10.1` |");
    expect(rendered).toContain("| Over the limit by | `7.23` |");
    expect(rendered).toContain("| Run | [View workflow run](https://github.com/octo/repo/actions/runs/123) |");
    expect(rendered).toContain("<details>");
    expect(rendered).toContain("<summary>Tips for reducing AI credit usage</summary>");
    expect(rendered).toContain("https://github.github.com/gh-aw/reference/cost-management/");
    expect(rendered).not.toContain("Consult the billing dashboards for accurate usage and charges.");
  });

  it("returns empty string when the AI Credits rate-limit did not trigger", () => {
    expect(buildAICreditsRateLimitErrorContext(false, "17.3", "10", "")).toBe("");
  });
});
