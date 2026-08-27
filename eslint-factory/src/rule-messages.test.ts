import { readFileSync, readdirSync } from "node:fs";
import { resolve } from "node:path";
import { describe, expect, it } from "vitest";

describe("ESLint rule messages", () => {
  it('does not claim unhandled throws "will crash the action"', () => {
    const rulesDirectory = resolve(__dirname, "rules");
    const ruleSources = readdirSync(rulesDirectory).filter(file => file.endsWith(".ts") && !file.endsWith(".test.ts"));

    for (const ruleSource of ruleSources) {
      const source = readFileSync(resolve(rulesDirectory, ruleSource), "utf8");
      expect(source).not.toContain("will crash the action");
    }
  });
});
