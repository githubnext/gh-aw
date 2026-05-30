import { describe, expect, it } from "vitest";
import { createRequire } from "module";

const require = createRequire(import.meta.url);
const { normalizeCommitSHA } = require("./commit_sha_helpers.cjs");

describe("normalizeCommitSHA", () => {
  it("accepts valid commit SHAs and trims whitespace", () => {
    expect(normalizeCommitSHA("  deadbeef  ")).toBe("deadbeef");
    expect(normalizeCommitSHA("A1B2C3D4")).toBe("A1B2C3D4");
    expect(normalizeCommitSHA("a".repeat(40))).toBe("a".repeat(40));
  });

  it("rejects invalid commit references", () => {
    expect(normalizeCommitSHA("main")).toBe("");
    expect(normalizeCommitSHA("--upload-pack=/bin/echo")).toBe("");
    expect(normalizeCommitSHA("deadbee f")).toBe("");
    expect(normalizeCommitSHA("")).toBe("");
  });
});
