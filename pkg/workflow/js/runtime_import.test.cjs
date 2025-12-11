// Tests for runtime_import.cjs
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import fs from "fs";
import path from "path";
import os from "os";

// Mock the core module
const core = {
  info: vi.fn(),
  warning: vi.fn(),
  setFailed: vi.fn(),
};
global.core = core;

// Import the functions to test
const {
  processRuntimeImports,
  processRuntimeImport,
  hasFrontMatter,
  removeXMLComments,
  hasGitHubActionsMacros,
} = require("./runtime_import.cjs");

describe("runtime_import", () => {
  let tempDir;

  beforeEach(() => {
    // Create a temporary directory for test files
    tempDir = fs.mkdtempSync(path.join(os.tmpdir(), "runtime-import-test-"));
    vi.clearAllMocks();
  });

  afterEach(() => {
    // Clean up temporary directory
    if (tempDir && fs.existsSync(tempDir)) {
      fs.rmSync(tempDir, { recursive: true, force: true });
    }
  });

  describe("hasFrontMatter", () => {
    it("should detect front matter at the start", () => {
      const content = "---\ntitle: Test\n---\nContent";
      expect(hasFrontMatter(content)).toBe(true);
    });

    it("should detect front matter with CRLF line endings", () => {
      const content = "---\r\ntitle: Test\r\n---\r\nContent";
      expect(hasFrontMatter(content)).toBe(true);
    });

    it("should detect front matter with leading whitespace", () => {
      const content = "  \n  ---\ntitle: Test\n---\nContent";
      expect(hasFrontMatter(content)).toBe(true);
    });

    it("should not detect front matter in the middle", () => {
      const content = "Some content\n---\ntitle: Test\n---";
      expect(hasFrontMatter(content)).toBe(false);
    });

    it("should not detect incomplete front matter marker", () => {
      const content = "--\ntitle: Test\n--\nContent";
      expect(hasFrontMatter(content)).toBe(false);
    });

    it("should handle empty content", () => {
      expect(hasFrontMatter("")).toBe(false);
    });
  });

  describe("removeXMLComments", () => {
    it("should remove simple XML comments", () => {
      const content = "Before <!-- comment --> After";
      expect(removeXMLComments(content)).toBe("Before  After");
    });

    it("should remove multiline XML comments", () => {
      const content = "Before <!-- multi\nline\ncomment --> After";
      expect(removeXMLComments(content)).toBe("Before  After");
    });

    it("should remove multiple XML comments", () => {
      const content = "<!-- first -->Text<!-- second -->More<!-- third -->";
      expect(removeXMLComments(content)).toBe("TextMore");
    });

    it("should handle content without comments", () => {
      const content = "No comments here";
      expect(removeXMLComments(content)).toBe("No comments here");
    });

    it("should handle nested-looking comments", () => {
      const content = "<!-- outer <!-- inner --> -->";
      // This should remove up to the first closing -->
      expect(removeXMLComments(content)).toBe(" -->");
    });

    it("should handle empty content", () => {
      expect(removeXMLComments("")).toBe("");
    });
  });

  describe("hasGitHubActionsMacros", () => {
    it("should detect simple GitHub Actions macros", () => {
      expect(hasGitHubActionsMacros("${{ github.actor }}")).toBe(true);
    });

    it("should detect multiline GitHub Actions macros", () => {
      expect(hasGitHubActionsMacros("${{ \ngithub.actor \n}}")).toBe(true);
    });

    it("should detect multiple GitHub Actions macros", () => {
      expect(hasGitHubActionsMacros("${{ github.actor }} and ${{ github.repo }}")).toBe(true);
    });

    it("should not detect template conditionals", () => {
      expect(hasGitHubActionsMacros("{{#if condition}}text{{/if}}")).toBe(false);
    });

    it("should not detect runtime-import macros", () => {
      expect(hasGitHubActionsMacros("{{#runtime-import file.md}}")).toBe(false);
    });

    it("should detect GitHub Actions macros within other content", () => {
      expect(hasGitHubActionsMacros("Some text ${{ github.actor }} more text")).toBe(true);
    });

    it("should handle content without macros", () => {
      expect(hasGitHubActionsMacros("No macros here")).toBe(false);
    });
  });

  describe("processRuntimeImport", () => {
    it("should read and return file content", () => {
      const filepath = "test.md";
      const content = "# Test Content\n\nThis is a test.";
      fs.writeFileSync(path.join(tempDir, filepath), content);

      const result = processRuntimeImport(filepath, false, tempDir);
      expect(result).toBe(content);
    });

    it("should throw error for missing required file", () => {
      expect(() => processRuntimeImport("missing.md", false, tempDir)).toThrow("Runtime import file not found: missing.md");
    });

    it("should return empty string for missing optional file", () => {
      const result = processRuntimeImport("missing.md", true, tempDir);
      expect(result).toBe("");
      expect(core.warning).toHaveBeenCalledWith("Optional runtime import file not found: missing.md");
    });

    it("should remove front matter and warn", () => {
      const filepath = "with-frontmatter.md";
      const content = "---\ntitle: Test\nkey: value\n---\n\n# Content\n\nActual content.";
      fs.writeFileSync(path.join(tempDir, filepath), content);

      const result = processRuntimeImport(filepath, false, tempDir);
      expect(result).toContain("# Content");
      expect(result).toContain("Actual content.");
      expect(result).not.toContain("title: Test");
      expect(core.warning).toHaveBeenCalledWith(`File ${filepath} contains front matter which will be ignored in runtime import`);
    });

    it("should remove XML comments", () => {
      const filepath = "with-comments.md";
      const content = "# Title\n\n<!-- This is a comment -->\n\nContent here.";
      fs.writeFileSync(path.join(tempDir, filepath), content);

      const result = processRuntimeImport(filepath, false, tempDir);
      expect(result).toContain("# Title");
      expect(result).toContain("Content here.");
      expect(result).not.toContain("<!-- This is a comment -->");
    });

    it("should throw error for GitHub Actions macros", () => {
      const filepath = "with-macros.md";
      const content = "# Title\n\nActor: ${{ github.actor }}\n";
      fs.writeFileSync(path.join(tempDir, filepath), content);

      expect(() => processRuntimeImport(filepath, false, tempDir)).toThrow(
        `File ${filepath} contains GitHub Actions macros ($\{{ ... }}) which are not allowed in runtime imports`
      );
    });

    it("should handle file in subdirectory", () => {
      const subdir = path.join(tempDir, "subdir");
      fs.mkdirSync(subdir);
      const filepath = "subdir/test.md";
      const content = "Subdirectory content";
      fs.writeFileSync(path.join(tempDir, filepath), content);

      const result = processRuntimeImport(filepath, false, tempDir);
      expect(result).toBe(content);
    });

    it("should handle empty file", () => {
      const filepath = "empty.md";
      fs.writeFileSync(path.join(tempDir, filepath), "");

      const result = processRuntimeImport(filepath, false, tempDir);
      expect(result).toBe("");
    });

    it("should handle file with only front matter", () => {
      const filepath = "only-frontmatter.md";
      const content = "---\ntitle: Test\n---\n";
      fs.writeFileSync(path.join(tempDir, filepath), content);

      const result = processRuntimeImport(filepath, false, tempDir);
      expect(result.trim()).toBe("");
    });

    it("should allow template conditionals", () => {
      const filepath = "with-conditionals.md";
      const content = "{{#if condition}}content{{/if}}";
      fs.writeFileSync(path.join(tempDir, filepath), content);

      const result = processRuntimeImport(filepath, false, tempDir);
      expect(result).toBe(content);
    });
  });

  describe("processRuntimeImports", () => {
    it("should process single runtime-import macro", () => {
      const filepath = "import.md";
      const importContent = "Imported content";
      fs.writeFileSync(path.join(tempDir, filepath), importContent);

      const content = "Before\n{{#runtime-import import.md}}\nAfter";
      const result = processRuntimeImports(content, tempDir);
      expect(result).toBe(`Before\n${importContent}\nAfter`);
    });

    it("should process optional runtime-import macro", () => {
      const filepath = "import.md";
      const importContent = "Imported content";
      fs.writeFileSync(path.join(tempDir, filepath), importContent);

      const content = "Before\n{{#runtime-import? import.md}}\nAfter";
      const result = processRuntimeImports(content, tempDir);
      expect(result).toBe(`Before\n${importContent}\nAfter`);
    });

    it("should process multiple runtime-import macros", () => {
      const file1 = "import1.md";
      const file2 = "import2.md";
      fs.writeFileSync(path.join(tempDir, file1), "Content 1");
      fs.writeFileSync(path.join(tempDir, file2), "Content 2");

      const content = "{{#runtime-import import1.md}}\nMiddle\n{{#runtime-import import2.md}}";
      const result = processRuntimeImports(content, tempDir);
      expect(result).toBe("Content 1\nMiddle\nContent 2");
    });

    it("should handle optional import of missing file", () => {
      const content = "Before\n{{#runtime-import? missing.md}}\nAfter";
      const result = processRuntimeImports(content, tempDir);
      expect(result).toBe("Before\n\nAfter");
      expect(core.warning).toHaveBeenCalled();
    });

    it("should throw error for required import of missing file", () => {
      const content = "Before\n{{#runtime-import missing.md}}\nAfter";
      expect(() => processRuntimeImports(content, tempDir)).toThrow();
    });

    it("should handle content without runtime-import macros", () => {
      const content = "No imports here";
      const result = processRuntimeImports(content, tempDir);
      expect(result).toBe(content);
    });

    it("should warn about duplicate imports", () => {
      const filepath = "import.md";
      fs.writeFileSync(path.join(tempDir, filepath), "Content");

      const content = "{{#runtime-import import.md}}\n{{#runtime-import import.md}}";
      processRuntimeImports(content, tempDir);
      expect(core.warning).toHaveBeenCalledWith(`File ${filepath} is imported multiple times, which may indicate a circular reference`);
    });

    it("should handle macros with extra whitespace", () => {
      const filepath = "import.md";
      fs.writeFileSync(path.join(tempDir, filepath), "Content");

      const content = "{{#runtime-import    import.md    }}";
      const result = processRuntimeImports(content, tempDir);
      expect(result).toBe("Content");
    });

    it("should handle inline macros", () => {
      const filepath = "inline.md";
      fs.writeFileSync(path.join(tempDir, filepath), "inline content");

      const content = "Before {{#runtime-import inline.md}} after";
      const result = processRuntimeImports(content, tempDir);
      expect(result).toBe("Before inline content after");
    });

    it("should process imports with files containing special characters", () => {
      const filepath = "import.md";
      const importContent = "Content with $pecial ch@racters!";
      fs.writeFileSync(path.join(tempDir, filepath), importContent);

      const content = "{{#runtime-import import.md}}";
      const result = processRuntimeImports(content, tempDir);
      expect(result).toBe(importContent);
    });

    it("should remove XML comments from imported content", () => {
      const filepath = "with-comment.md";
      fs.writeFileSync(path.join(tempDir, filepath), "Text <!-- comment --> more text");

      const content = "{{#runtime-import with-comment.md}}";
      const result = processRuntimeImports(content, tempDir);
      expect(result).toBe("Text  more text");
    });

    it("should handle path with subdirectories", () => {
      const subdir = path.join(tempDir, "docs", "shared");
      fs.mkdirSync(subdir, { recursive: true });
      const filepath = "docs/shared/import.md";
      fs.writeFileSync(path.join(tempDir, filepath), "Subdir content");

      const content = "{{#runtime-import docs/shared/import.md}}";
      const result = processRuntimeImports(content, tempDir);
      expect(result).toBe("Subdir content");
    });

    it("should preserve newlines around imports", () => {
      const filepath = "import.md";
      fs.writeFileSync(path.join(tempDir, filepath), "Content");

      const content = "Line 1\n\n{{#runtime-import import.md}}\n\nLine 2";
      const result = processRuntimeImports(content, tempDir);
      expect(result).toBe("Line 1\n\nContent\n\nLine 2");
    });

    it("should handle multiple consecutive imports", () => {
      const file1 = "import1.md";
      const file2 = "import2.md";
      fs.writeFileSync(path.join(tempDir, file1), "Content 1");
      fs.writeFileSync(path.join(tempDir, file2), "Content 2");

      const content = "{{#runtime-import import1.md}}{{#runtime-import import2.md}}";
      const result = processRuntimeImports(content, tempDir);
      expect(result).toBe("Content 1Content 2");
    });

    it("should handle imports at the start of content", () => {
      const filepath = "import.md";
      fs.writeFileSync(path.join(tempDir, filepath), "Start content");

      const content = "{{#runtime-import import.md}}\nFollowing text";
      const result = processRuntimeImports(content, tempDir);
      expect(result).toBe("Start content\nFollowing text");
    });

    it("should handle imports at the end of content", () => {
      const filepath = "import.md";
      fs.writeFileSync(path.join(tempDir, filepath), "End content");

      const content = "Preceding text\n{{#runtime-import import.md}}";
      const result = processRuntimeImports(content, tempDir);
      expect(result).toBe("Preceding text\nEnd content");
    });

    it("should handle tab characters in macro", () => {
      const filepath = "import.md";
      fs.writeFileSync(path.join(tempDir, filepath), "Content");

      const content = "{{#runtime-import\timport.md}}";
      const result = processRuntimeImports(content, tempDir);
      expect(result).toBe("Content");
    });
  });

  describe("Edge Cases", () => {
    it("should handle very large files", () => {
      const filepath = "large.md";
      const largeContent = "x".repeat(100000);
      fs.writeFileSync(path.join(tempDir, filepath), largeContent);

      const content = "{{#runtime-import large.md}}";
      const result = processRuntimeImports(content, tempDir);
      expect(result).toBe(largeContent);
    });

    it("should handle files with unicode characters", () => {
      const filepath = "unicode.md";
      const unicodeContent = "Hello 世界 🌍 café";
      fs.writeFileSync(path.join(tempDir, filepath), unicodeContent, "utf8");

      const content = "{{#runtime-import unicode.md}}";
      const result = processRuntimeImports(content, tempDir);
      expect(result).toBe(unicodeContent);
    });

    it("should handle files with various line endings", () => {
      const filepath = "mixed-lines.md";
      const content = "Line 1\nLine 2\r\nLine 3\rLine 4";
      fs.writeFileSync(path.join(tempDir, filepath), content);

      const importContent = "{{#runtime-import mixed-lines.md}}";
      const result = processRuntimeImports(importContent, tempDir);
      expect(result).toBe(content);
    });

    it("should not process runtime-import as a substring", () => {
      const content = "text{{#runtime-importnospace.md}}text";
      const result = processRuntimeImports(content, tempDir);
      // Should not match because there's no space after runtime-import
      expect(result).toBe(content);
    });

    it("should handle front matter with varying formats", () => {
      const filepath = "yaml-frontmatter.md";
      const content = "---\ntitle: Test\narray:\n  - item1\n  - item2\n---\n\nBody content";
      fs.writeFileSync(path.join(tempDir, filepath), content);

      const result = processRuntimeImport(filepath, false, tempDir);
      expect(result).toContain("Body content");
      expect(result).not.toContain("array:");
      expect(result).not.toContain("item1");
    });
  });

  describe("Error Handling", () => {
    it("should provide clear error for GitHub Actions macros", () => {
      const filepath = "bad.md";
      fs.writeFileSync(path.join(tempDir, filepath), "${{ github.actor }}");

      const content = "{{#runtime-import bad.md}}";
      expect(() => processRuntimeImports(content, tempDir)).toThrow("Failed to process runtime import for bad.md");
    });

    it("should provide clear error for missing required files", () => {
      const content = "{{#runtime-import nonexistent.md}}";
      expect(() => processRuntimeImports(content, tempDir)).toThrow("Failed to process runtime import for nonexistent.md");
    });
  });
});
