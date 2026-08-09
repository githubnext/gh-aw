import { describe, expect, it } from "vitest";

const { buildNoopConclusionSummary } = await import("./conclusion_summary.cjs");

describe("conclusion_summary", () => {
  it("renders regular no-op entries inside a linked conclusion summary", () => {
    const summary = buildNoopConclusionSummary(["No changes were needed.", "Repository is already current."], {
      runUrl: "https://github.com/owner/repo/actions/runs/123",
    });

    expect(summary).toContain("<summary>✅ Conclusion Summary (2 no-op messages)</summary>");
    expect(summary).toContain("**Target:** [Workflow run](https://github.com/owner/repo/actions/runs/123)");
    expect(summary).toContain("<summary>✅ No-Op - Success (Message 1)</summary>");
    expect(summary).toContain("<summary>✅ No-Op - Success (Message 2)</summary>");
    expect(summary).toContain("No changes were needed.");
  });

  it("renders staged no-op messages as a preview", () => {
    const summary = buildNoopConclusionSummary(["No action needed."], { staged: true });

    expect(summary).toContain("<summary>⚠️ Staged No-Op Preview (1 no-op message)</summary>");
    expect(summary).toContain("<summary>⚠️ No-Op - Preview (Message 1)</summary>");
  });

  it("does not link non-http run targets", () => {
    const summary = buildNoopConclusionSummary(["No action needed."], { runUrl: "javascript:alert(1)" });

    expect(summary).not.toContain("javascript:");
    expect(summary).not.toContain("**Target:**");
  });
});
