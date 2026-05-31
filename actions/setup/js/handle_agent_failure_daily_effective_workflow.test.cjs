import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

let buildDailyEffectiveWorkflowExceededContext;

describe("handle_agent_failure daily workflow ET context", () => {
  beforeEach(async () => {
    vi.resetModules();
    const mod = await import("./handle_agent_failure.cjs");
    const exports = mod.default || mod;
    buildDailyEffectiveWorkflowExceededContext = exports.buildDailyEffectiveWorkflowExceededContext;
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("renders the daily workflow ET guardrail context when exceeded", () => {
    const rendered = buildDailyEffectiveWorkflowExceededContext(true, "2500", "2000", "https://github.com/octo/repo/issues/1");
    expect(rendered).toContain("Daily Workflow ET Guardrail Exceeded");
    expect(rendered).toContain("2500");
    expect(rendered).toContain("2000");
    expect(rendered).toContain("https://github.com/octo/repo/issues/1");
  });

  it("returns empty string when the guardrail did not trigger", () => {
    expect(buildDailyEffectiveWorkflowExceededContext(false, "2500", "2000", "")).toBe("");
  });
});

