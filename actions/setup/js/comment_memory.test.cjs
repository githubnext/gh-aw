// @ts-check
import { describe, it, expect } from "vitest";

describe("comment_memory", () => {
  it("sanitizes valid memory IDs", async () => {
    const module = await import("./comment_memory.cjs");
    expect(module.sanitizeMemoryID("Session_1")).toBe("session_1");
    expect(module.sanitizeMemoryID("notes-2026")).toBe("notes-2026");
  });

  it("rejects invalid memory IDs", async () => {
    const module = await import("./comment_memory.cjs");
    expect(module.sanitizeMemoryID("bad id")).toBeNull();
    expect(module.sanitizeMemoryID("../oops")).toBeNull();
  });

  it("builds managed comment body with xml memory marker", async () => {
    const module = await import("./comment_memory.cjs");
    const body = module.buildManagedMemoryBody("Hello world", "default", false, "https://example.com/run/1", "Workflow", "", "", undefined);
    expect(body).toContain('<comment-memory id="default">');
    expect(body).toContain("Hello world");
    expect(body).toContain("</comment-memory>");
  });
});
