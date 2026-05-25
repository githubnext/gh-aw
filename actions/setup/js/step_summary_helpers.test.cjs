import { describe, expect, it } from "vitest";

const { stripLeadingEmoji, normalizeHeadingLevel, normalizeStepSummaryMarkdown, normalizeSummaryTableRows } = await import("./step_summary_helpers.cjs");

describe("step_summary_helpers", () => {
  it("removes leading emoji from headings", () => {
    expect(stripLeadingEmoji("🔐 Secret Validation Report")).toBe("Secret Validation Report");
  });

  it("normalizes heading levels to H2-H3", () => {
    expect(normalizeHeadingLevel(1)).toBe(2);
    expect(normalizeHeadingLevel(2)).toBe(2);
    expect(normalizeHeadingLevel(4)).toBe(3);
  });

  it("normalizes heading markdown and markdown tables", () => {
    const input = ["# ✅ Title", "", "| A |  B|", "|:----|----:|", "| 1|2 |"].join("\n");
    const output = normalizeStepSummaryMarkdown(input);

    expect(output).toContain("## Title");
    expect(output).toContain("| A | B |");
    expect(output).toContain("| --- | --- |");
    expect(output).toContain("| 1 | 2 |");
  });

  it("redacts known secret formats", () => {
    const output = normalizeStepSummaryMarkdown(`token=${"ghp_" + "a".repeat(36)}`);
    expect(output).toContain("***REDACTED***");
    expect(output).not.toContain("ghp_");
  });

  it("normalizes addTable rows", () => {
    const rows = normalizeSummaryTableRows([
      [{ data: "AKIA1234567890ABCDEF", header: true }, " value "],
      ["key", "ghp_" + "b".repeat(36)],
    ]);
    expect(rows[0][0].data).toBe("***REDACTED***");
    expect(rows[1][1]).toBe("***REDACTED***");
  });
});
