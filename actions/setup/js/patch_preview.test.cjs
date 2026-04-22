// @ts-check
import { describe, it, expect } from "vitest";
import { createRequire } from "module";

const require = createRequire(import.meta.url);
const { generatePatchPreview } = require("./patch_preview.cjs");

describe("patch_preview", () => {
  it("returns empty string for empty patch content", () => {
    expect(generatePatchPreview("")).toBe("");
  });

  it("formats non-truncated preview", () => {
    const patch = ["diff --git a/a.txt b/a.txt", "--- a/a.txt", "+++ b/a.txt"].join("\n");
    const preview = generatePatchPreview(patch);

    expect(preview).toContain("<summary>Show patch (3 lines)</summary>");
    expect(preview).toContain("```diff");
    expect(preview).not.toContain("... (truncated)");
  });

  it("truncates by line count", () => {
    const patch = Array.from({ length: 600 }, () => "x").join("\n");
    const preview = generatePatchPreview(patch);

    expect(preview).toContain("<summary>Show patch preview (500 of 600 lines)</summary>");
    expect(preview).toContain("... (truncated)");
  });

  it("truncates by character count", () => {
    const patch = "x".repeat(2500);
    const preview = generatePatchPreview(patch);

    expect(preview).toContain("<summary>Show patch preview (1 of 1 lines)</summary>");
    expect(preview).toContain("... (truncated)");
  });

  it("reports shown line count accurately when truncated by characters", () => {
    const patch = Array.from({ length: 800 }, (_, i) => `line-${i}`).join("\n");
    const preview = generatePatchPreview(patch);

    expect(preview).toMatch(/<summary>Show patch preview \(\d+ of 800 lines\)<\/summary>/);
    expect(preview).not.toContain("<summary>Show patch preview (800 of 800 lines)</summary>");
  });
});
