import { describe, expect, it } from "vitest";
import { createRequire } from "module";

const require = createRequire(import.meta.url);
const { extractPatchBaseCommit, normalizeCommitSHA } = require("./commit_sha_helpers.cjs");

describe("normalizeCommitSHA", () => {
  it("accepts valid commit SHAs and trims whitespace", () => {
    expect(normalizeCommitSHA("  deadbeef  ")).toBe("deadbeef");
    expect(normalizeCommitSHA("A1B2C3D4")).toBe("A1B2C3D4");
    expect(normalizeCommitSHA("a".repeat(40))).toBe("a".repeat(40));
  });

  describe("extractPatchBaseCommit", () => {
    it("extracts a validated base commit from patch metadata", () => {
      expect(extractPatchBaseCommit("From abc123 Mon Sep 17 00:00:00 2001\nX-GH-AW-Base-Commit: deadbeef\nFrom: Test\n")).toBe("deadbeef");
    });

    it("ignores missing or malformed patch metadata", () => {
      expect(extractPatchBaseCommit("From abc123 Mon Sep 17 00:00:00 2001\nFrom: Test\n")).toBe("");
      expect(extractPatchBaseCommit("X-GH-AW-Base-Commit: main\n")).toBe("");
    });

    it("ignores metadata-like lines outside the patch header block", () => {
      expect(extractPatchBaseCommit("From abc123 Mon Sep 17 00:00:00 2001\nFrom: Test\n\nX-GH-AW-Base-Commit: deadbeef\n")).toBe("");
    });
  });

  it("rejects invalid commit references", () => {
    expect(normalizeCommitSHA("main")).toBe("");
    expect(normalizeCommitSHA("--upload-pack=/bin/echo")).toBe("");
    expect(normalizeCommitSHA("deadbee f")).toBe("");
    expect(normalizeCommitSHA("")).toBe("");
  });
});
