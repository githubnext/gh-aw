import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import path from "path";
import { fileURLToPath } from "url";

let buildDailyEffectiveWorkflowExceededContext;
const __dirname = path.dirname(fileURLToPath(import.meta.url));

describe("handle_agent_failure daily workflow ET context", () => {
  beforeEach(async () => {
    vi.resetModules();
    process.env.GH_AW_PROMPTS_DIR = path.join(__dirname, "../md");
    const mod = await import("./handle_agent_failure.cjs");
    const exports = mod.default || mod;
    buildDailyEffectiveWorkflowExceededContext = exports.buildDailyEffectiveWorkflowExceededContext;
  });

  afterEach(() => {
    vi.restoreAllMocks();
    delete process.env.GH_AW_PROMPTS_DIR;
  });

  it("renders the daily workflow ET guardrail context when exceeded", () => {
    const rendered = buildDailyEffectiveWorkflowExceededContext(true, "2500", "2000");
    expect(rendered).toContain("Daily Workflow AIC Guardrail Exceeded");
    expect(rendered).toContain("2500");
    expect(rendered).toContain("2000");
    expect(rendered).not.toContain("Activation Issue:");
    // Progressive disclosure sections
    expect(rendered).toContain("How to raise the daily limit");
    expect(rendered).toContain("max-daily-ai-credits");
    expect(rendered).toContain("What is the daily AI Credits guardrail");
    expect(rendered).toContain("How to disable this guardrail");
  });

  it("returns empty string when the guardrail did not trigger", () => {
    expect(buildDailyEffectiveWorkflowExceededContext(false, "2500", "2000", "")).toBe("");
  });
});
