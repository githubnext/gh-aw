import { Linter } from "eslint";
import { describe, expect, it } from "vitest";

const cjsConfig = {
  languageOptions: {
    ecmaVersion: "latest" as const,
    sourceType: "commonjs" as const,
  },
};

describe("CommonJS syntax validation", () => {
  it("rejects await inside non-async functions", () => {
    const linter = new Linter();
    const messages = linter.verify(`function writeStepSummaryWithTokenUsage(coreObj) { await coreObj.summary.write(); }`, cjsConfig, "parse_mcp_gateway_log.cjs");

    expect(messages).toContainEqual(
      expect.objectContaining({
        fatal: true,
        ruleId: null,
        severity: 2,
      })
    );
  });

  it("allows await inside async functions", () => {
    const linter = new Linter();
    const messages = linter.verify(`async function writeStepSummaryWithTokenUsage(coreObj) { await coreObj.summary.write(); }`, cjsConfig, "parse_mcp_gateway_log.cjs");

    expect(messages).toEqual([]);
  });
});
