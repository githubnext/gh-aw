import { readFileSync } from "node:fs";
import { resolve } from "node:path";
import { describe, expect, it } from "vitest";

describe("ESLint rule documentation", () => {
  it("documents every exported rule exactly once", () => {
    const readme = readFileSync(resolve(__dirname, "../README.md"), "utf8");
    const index = readFileSync(resolve(__dirname, "index.ts"), "utf8");
    const documentedRuleNames = [...readme.matchAll(/^### `([^`]+)`$/gm)].map(match => match[1]);
    const rulesBlock = index.match(/^\s+rules: \{\n([\s\S]*?)^\s+\},$/m)?.[1] ?? "";
    const exportedRuleNames = [...rulesBlock.matchAll(/^\s+"([^"]+)":/gm)].map(match => match[1]);

    expect(rulesBlock).not.toBe("");
    expect(documentedRuleNames).toHaveLength(exportedRuleNames.length);
    expect([...documentedRuleNames].sort()).toEqual([...exportedRuleNames].sort());
  });
});
