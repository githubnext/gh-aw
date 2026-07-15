// @ts-check
import fs from "fs";
import os from "os";
import path from "path";
import { afterEach, beforeEach, describe, expect, it } from "vitest";

let exports;

describe("daily_aic_workflow_helpers", () => {
  let tmpDir;

  beforeEach(async () => {
    tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), "daily-aic-helpers-test-"));
    const mod = await import("./daily_aic_workflow_helpers.cjs");
    exports = mod.default || mod;
  });

  afterEach(() => {
    fs.rmSync(tmpDir, { recursive: true, force: true });
  });

  describe("sumAICFromUsageJSONLFiles", () => {
    it("returns 0 for an empty array", () => {
      expect(exports.sumAICFromUsageJSONLFiles([])).toBe(0);
    });

    it("returns 0 for a non-array argument", () => {
      expect(exports.sumAICFromUsageJSONLFiles(null)).toBe(0);
      expect(exports.sumAICFromUsageJSONLFiles(undefined)).toBe(0);
    });

    it("returns 0 for a file path that does not exist", () => {
      expect(exports.sumAICFromUsageJSONLFiles([path.join(tmpDir, "nonexistent.jsonl")])).toBe(0);
    });

    it("returns 0 for an empty file", () => {
      const filePath = path.join(tmpDir, "empty.jsonl");
      fs.writeFileSync(filePath, "", "utf8");
      expect(exports.sumAICFromUsageJSONLFiles([filePath])).toBe(0);
    });

    it("sums explicit top-level aic fields", () => {
      const filePath = path.join(tmpDir, "usage.jsonl");
      fs.writeFileSync(filePath, [JSON.stringify({ aic: 1.5 }), JSON.stringify({ aic: 2.5 })].join("\n"), "utf8");
      expect(exports.sumAICFromUsageJSONLFiles([filePath])).toBe(4);
    });

    it("sums explicit aic fields nested under usage object", () => {
      const filePath = path.join(tmpDir, "usage.jsonl");
      fs.writeFileSync(filePath, [JSON.stringify({ usage: { aic: 1.5 } }), JSON.stringify({ usage: { aic: 0.5 } })].join("\n"), "utf8");
      expect(exports.sumAICFromUsageJSONLFiles([filePath])).toBe(2);
    });

    it("prefers usage.aic over top-level aic", () => {
      const filePath = path.join(tmpDir, "usage.jsonl");
      fs.writeFileSync(filePath, JSON.stringify({ aic: 9, usage: { aic: 0.25 } }), "utf8");
      expect(exports.sumAICFromUsageJSONLFiles([filePath])).toBe(0.25);
    });

    it("sums explicit ai_credits fields", () => {
      const filePath = path.join(tmpDir, "usage.jsonl");
      fs.writeFileSync(filePath, JSON.stringify({ ai_credits: 8, usage: { ai_credits: 0.1 } }), "utf8");
      expect(exports.sumAICFromUsageJSONLFiles([filePath])).toBe(0.1);
    });

    it("sums explicit aiCredits (camelCase) fields", () => {
      const filePath = path.join(tmpDir, "usage.jsonl");
      fs.writeFileSync(filePath, JSON.stringify({ aiCredits: 7, usage: { aiCredits: 0.15 } }), "utf8");
      expect(exports.sumAICFromUsageJSONLFiles([filePath])).toBe(0.15);
    });

    it("falls back to inference-cost computation from token fields when no explicit AIC", () => {
      const filePath = path.join(tmpDir, "usage.jsonl");
      // gpt-4o has a known cost; supply enough tokens to get a non-zero AIC
      fs.writeFileSync(filePath, JSON.stringify({ model: "gpt-4o", provider: "openai", input_tokens: 1000, output_tokens: 500 }), "utf8");
      const result = exports.sumAICFromUsageJSONLFiles([filePath]);
      expect(result).toBeGreaterThan(0);
      expect(Number.isFinite(result)).toBe(true);
    });

    it("ignores malformed (non-JSON) lines", () => {
      const filePath = path.join(tmpDir, "usage.jsonl");
      fs.writeFileSync(filePath, ["not-json", JSON.stringify({ aic: 3 }), "{bad}"].join("\n"), "utf8");
      expect(exports.sumAICFromUsageJSONLFiles([filePath])).toBe(3);
    });

    it("ignores blank lines and lines not starting with {", () => {
      const filePath = path.join(tmpDir, "usage.jsonl");
      fs.writeFileSync(filePath, [" ", "", JSON.stringify({ aic: 2 }), "[1,2,3]"].join("\n"), "utf8");
      expect(exports.sumAICFromUsageJSONLFiles([filePath])).toBe(2);
    });

    it("accumulates across multiple files", () => {
      const fileA = path.join(tmpDir, "a.jsonl");
      const fileB = path.join(tmpDir, "b.jsonl");
      fs.writeFileSync(fileA, JSON.stringify({ aic: 1 }), "utf8");
      fs.writeFileSync(fileB, JSON.stringify({ aic: 2 }), "utf8");
      expect(exports.sumAICFromUsageJSONLFiles([fileA, fileB])).toBe(3);
    });

    it("skips empty-string aic/aiCredits values (treats them as missing)", () => {
      const filePath = path.join(tmpDir, "usage.jsonl");
      fs.writeFileSync(filePath, [JSON.stringify({ aiCredits: 0.2, usage: { aiCredits: "" } }), JSON.stringify({ aic: 0.3, usage: { aic: "" } })].join("\n"), "utf8");
      // usage fields are empty strings → fall through to top-level values
      expect(exports.sumAICFromUsageJSONLFiles([filePath])).toBe(0.5);
    });
  });
});
