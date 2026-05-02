import { describe, it, expect } from "vitest";
import { assemblePrompt } from "../src/context.js";
import type { ImportEntry } from "../src/types.js";

describe("assemblePrompt", () => {
  it("returns prompt body unchanged when there are no imports", () => {
    const result = assemblePrompt([], "Do the task.");
    expect(result).toBe("Do the task.");
  });

  it("prepends a single import with delimiters", () => {
    const imports: ImportEntry[] = [{ path: "skills/foo/SKILL.md", content: "# Foo skill\nContent." }];
    const result = assemblePrompt(imports, "Do the task.");

    expect(result).toContain("<!-- import: skills/foo/SKILL.md -->");
    expect(result).toContain("# Foo skill");
    expect(result).toContain("<!-- /import: skills/foo/SKILL.md -->");
    expect(result).toContain("Do the task.");
    // Imports come before the prompt body
    expect(result.indexOf("<!-- import:")).toBeLessThan(result.indexOf("Do the task."));
  });

  it("prepends multiple imports in declaration order", () => {
    const imports: ImportEntry[] = [
      { path: "skills/a/SKILL.md", content: "Skill A" },
      { path: "skills/b/SKILL.md", content: "Skill B" },
    ];
    const result = assemblePrompt(imports, "Do the task.");

    const posA = result.indexOf("Skill A");
    const posB = result.indexOf("Skill B");
    const posPrompt = result.indexOf("Do the task.");

    expect(posA).toBeGreaterThanOrEqual(0);
    expect(posB).toBeGreaterThanOrEqual(0);
    expect(posA).toBeLessThan(posB);
    expect(posB).toBeLessThan(posPrompt);
  });

  it("trims trailing whitespace from import content", () => {
    const imports: ImportEntry[] = [{ path: "foo.md", content: "Content   \n\n" }];
    const result = assemblePrompt(imports, "Prompt");
    // trimEnd should remove trailing whitespace/newlines from content
    expect(result).not.toMatch(/Content   \n\n\n<!-- \/import/);
  });

  it("handles import with empty content", () => {
    const imports: ImportEntry[] = [{ path: "empty.md", content: "" }];
    const result = assemblePrompt(imports, "Prompt");
    expect(result).toContain("<!-- import: empty.md -->");
    expect(result).toContain("Prompt");
  });

  it("handles empty prompt body", () => {
    const imports: ImportEntry[] = [{ path: "foo.md", content: "Content" }];
    const result = assemblePrompt(imports, "");
    expect(result).toContain("Content");
    // Should include the empty prompt (might be just the imports + separator)
    expect(typeof result).toBe("string");
  });
});
