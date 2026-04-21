import { describe, it, expect, vi } from "vitest";
import { extractCommentMemoryEntries, isSafeMemoryId } from "./comment_memory_helpers.cjs";

describe("comment_memory_helpers", () => {
  it("extracts managed memory entries", () => {
    const entries = extractCommentMemoryEntries('<gh-aw-comment-memory id="default">\nhello\n</gh-aw-comment-memory>');
    expect(entries).toEqual([{ memoryId: "default", content: "hello" }]);
  });

  it("rejects unsafe memory IDs", () => {
    const warning = vi.fn();
    const entries = extractCommentMemoryEntries('<gh-aw-comment-memory id="../bad">\nhello\n</gh-aw-comment-memory>', warning);
    expect(entries).toEqual([]);
    expect(warning).toHaveBeenCalled();
    expect(isSafeMemoryId("../bad")).toBe(false);
  });

  it("allows memory IDs up to 128 characters", () => {
    const maxLengthId = "a".repeat(128);
    const tooLongId = "b".repeat(129);
    expect(isSafeMemoryId(maxLengthId)).toBe(true);
    expect(isSafeMemoryId(tooLongId)).toBe(false);
  });
});
