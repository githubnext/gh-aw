// @ts-check

import { describe, it, expect } from "vitest";
import { createRequire } from "module";

const req = createRequire(import.meta.url);
const { getPatchDiffSizeBytes, getStagedPatchDiffSizeBytes } = req("./repo_memory_patch_size.cjs");

describe("getPatchDiffSizeBytes", () => {
  it("returns 0 for an empty diff", () => {
    expect(getPatchDiffSizeBytes("")).toBe(0);
  });

  it("counts only addition bytes for a new file (first push)", () => {
    const diff = ["diff --git a/history.jsonl b/history.jsonl", "new file mode 100644", "index 0000000..abc1234", "--- /dev/null", "+++ b/history.jsonl", "@@ -0,0 +1,2 @@", '+{"event":"a"}', '+{"event":"b"}'].join("\n");
    const result = getPatchDiffSizeBytes(diff);
    // Both lines are additions with no deletions → net = additions
    const expected = Buffer.byteLength('+{"event":"a"}\n', "utf8") + Buffer.byteLength('+{"event":"b"}\n', "utf8");
    expect(result).toBe(expected);
  });

  it("returns a small net value when a file is appended (not rewritten)", () => {
    // Simulates appending one new line to an existing JSONL file
    const diff = ["diff --git a/history.jsonl b/history.jsonl", "index abc1234..def5678 100644", "--- a/history.jsonl", "+++ b/history.jsonl", "@@ -1,2 +1,3 @@", ' {"event":"a"}', ' {"event":"b"}', '+{"event":"c"}'].join("\n");
    const result = getPatchDiffSizeBytes(diff);
    // Only the added line contributes; no deletions
    const expected = Buffer.byteLength('+{"event":"c"}\n', "utf8");
    expect(result).toBe(expected);
  });

  it("returns near-zero for a complete rewrite with the same size (the key bug fix)", () => {
    // Simulates a JSON object completely regenerated with equivalent size.
    // Previously getAddedPatchSizeBytesFromDiff would return the full new-file
    // size ("entire source code size").  getPatchDiffSizeBytes returns the net
    // change (additions − deletions), which is ≈ 0 for same-size rewrites.
    const oldLine = '{"key":"old_value_aaaa"}';
    const newLine = '{"key":"new_value_bbbb"}';
    const diff = ["diff --git a/state.json b/state.json", "index abc1234..def5678 100644", "--- a/state.json", "+++ b/state.json", "@@ -1 +1 @@", `-${oldLine}`, `+${newLine}`].join("\n");
    const result = getPatchDiffSizeBytes(diff);
    const additionBytes = Buffer.byteLength(`+${newLine}\n`, "utf8");
    const deletionBytes = Buffer.byteLength(`-${oldLine}\n`, "utf8");
    // Both lines have the same length, so net ≈ 0
    expect(result).toBe(Math.max(0, additionBytes - deletionBytes));
    // Explicitly: this should NOT equal the full new-file size
    expect(result).toBeLessThan(additionBytes);
  });

  it("returns 0 when deletions exceed additions (content shrinks)", () => {
    const diff = [
      "diff --git a/state.json b/state.json",
      "index abc1234..def5678 100644",
      "--- a/state.json",
      "+++ b/state.json",
      "@@ -1,3 +1,1 @@",
      '-{"key":"value_one"}',
      '-{"key":"value_two"}',
      '-{"key":"value_three"}',
      '+{"key":"v"}',
    ].join("\n");
    const result = getPatchDiffSizeBytes(diff);
    // Deletions > additions → clamped to 0
    expect(result).toBe(0);
  });

  it("handles multiple files in one diff", () => {
    const diff = [
      "diff --git a/a.json b/a.json",
      "index 0000000..abc1234 100644",
      "--- /dev/null",
      "+++ b/a.json",
      "@@ -0,0 +1 @@",
      '+{"a":1}',
      "diff --git a/b.json b/b.json",
      "index 0000000..def5678 100644",
      "--- /dev/null",
      "+++ b/b.json",
      "@@ -0,0 +1 @@",
      '+{"b":2}',
    ].join("\n");
    const result = getPatchDiffSizeBytes(diff);
    const expected = Buffer.byteLength('+{"a":1}\n', "utf8") + Buffer.byteLength('+{"b":2}\n', "utf8");
    expect(result).toBe(expected);
  });

  it("does not count +++ file header lines (they appear before any @@ hunk)", () => {
    const diff = [
      "diff --git a/file.json b/file.json",
      "index 0000000..abc1234 100644",
      "--- /dev/null",
      "+++ b/file.json",
      // No @@ yet — the +++ line above must NOT be counted
      "@@ -0,0 +1 @@",
      '+{"x":1}',
    ].join("\n");
    const result = getPatchDiffSizeBytes(diff);
    const expected = Buffer.byteLength('+{"x":1}\n', "utf8");
    expect(result).toBe(expected);
  });
});

describe("getStagedPatchDiffSizeBytes", () => {
  it("calls git diff --cached and returns net patch diff size", () => {
    const diffOutput = ["diff --git a/data.json b/data.json", "index abc..def 100644", "--- a/data.json", "+++ b/data.json", "@@ -1 +1,2 @@", '-{"old":1}', '+{"new":1}', '+{"extra":2}'].join("\n");

    const execGitSyncFn = (/** @type {string[]} */ args, /** @type {any} */ _opts) => {
      if (args[0] === "diff" && args[1] === "--cached") return diffOutput;
      return "";
    };

    const result = getStagedPatchDiffSizeBytes({ execGitSyncFn, cwd: "/some/dir" });
    const additions = Buffer.byteLength('+{"new":1}\n', "utf8") + Buffer.byteLength('+{"extra":2}\n', "utf8");
    const deletions = Buffer.byteLength('-{"old":1}\n', "utf8");
    expect(result).toBe(Math.max(0, additions - deletions));
  });

  it("passes the cwd option to execGitSyncFn", () => {
    const calls = /** @type {Array<{args: string[], opts: any}>} */ [];
    const execGitSyncFn = (/** @type {string[]} */ args, /** @type {any} */ opts) => {
      calls.push({ args, opts });
      return "";
    };

    getStagedPatchDiffSizeBytes({ execGitSyncFn, cwd: "/memory/dir" });
    expect(calls).toHaveLength(1);
    expect(calls[0].args).toEqual(["diff", "--cached"]);
    expect(calls[0].opts.cwd).toBe("/memory/dir");
  });
});
