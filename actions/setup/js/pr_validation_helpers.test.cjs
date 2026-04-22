// @ts-check
import { describe, it, expect } from "vitest";
import { createRequire } from "module";

const require = createRequire(import.meta.url);
const { parseAllowedBaseBranches, isBaseBranchAllowed } = require("./pr_validation_helpers.cjs");

describe("pr_validation_helpers", () => {
  describe("parseAllowedBaseBranches", () => {
    it("parses array values", () => {
      const parsed = parseAllowedBaseBranches([" main ", "release/*", ""]);
      expect([...parsed]).toEqual(["main", "release/*"]);
    });

    it("parses comma-separated values", () => {
      const parsed = parseAllowedBaseBranches("main, release/*, ");
      expect([...parsed]).toEqual(["main", "release/*"]);
    });
  });

  describe("isBaseBranchAllowed", () => {
    it("allows exact branch matches", () => {
      expect(isBaseBranchAllowed("main", new Set(["main"]))).toBe(true);
    });

    it("allows wildcard branch matches", () => {
      expect(isBaseBranchAllowed("release/2026.04", new Set(["release/*"]))).toBe(true);
    });

    it("allows all branches for star pattern", () => {
      expect(isBaseBranchAllowed("feature/x", new Set(["*"]))).toBe(true);
    });

    it("rejects non-matching branches", () => {
      expect(isBaseBranchAllowed("feature/x", new Set(["main", "release/*"]))).toBe(false);
    });
  });
});
