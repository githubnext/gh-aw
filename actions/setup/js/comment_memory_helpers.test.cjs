import { describe, it, expect, vi } from "vitest";
import { extractCommentMemoryEntries, isSafeMemoryId, stripCommentMemoryCodeFence } from "./comment_memory_helpers.cjs";

describe("comment_memory_helpers", () => {
  it("extracts managed memory entries", () => {
    const entries = extractCommentMemoryEntries('<gh-aw-comment-memory id="default">\n``````\nhello\n``````\n</gh-aw-comment-memory>');
    expect(entries).toEqual([{ memoryId: "default", content: "hello" }]);
  });

  it("supports legacy memory entries without code fence markers", () => {
    const entries = extractCommentMemoryEntries('<gh-aw-comment-memory id="default">\nhello\n</gh-aw-comment-memory>');
    expect(entries).toEqual([{ memoryId: "default", content: "hello" }]);
  });

  it("keeps fenced text unchanged when trailing content exists after closing fence", () => {
    const content = "``````\nhello\n``````\ntrailing";
    expect(stripCommentMemoryCodeFence(content)).toBe(content);
  });

  it("keeps fenced text unchanged when closing fence is missing", () => {
    const content = "``````\nhello";
    expect(stripCommentMemoryCodeFence(content)).toBe(content);
  });

  it("keeps malformed fenced text unchanged", () => {
    const content = "``````hello\n``````";
    expect(stripCommentMemoryCodeFence(content)).toBe(content);
  });

  it("strips valid fenced text with extra newlines before content", () => {
    const content = "``````\n\nhello\n``````";
    expect(stripCommentMemoryCodeFence(content)).toBe("hello");
  });

  it("strips valid fenced text when content contains six-backtick lines", () => {
    const content = "``````\nline 1\n``````\nline 2\n``````";
    expect(stripCommentMemoryCodeFence(content)).toBe("line 1\n``````\nline 2");
  });

  it("keeps fenced text unchanged when closing fence has no leading newline", () => {
    const content = "``````\nhello``````";
    expect(stripCommentMemoryCodeFence(content)).toBe(content);
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
