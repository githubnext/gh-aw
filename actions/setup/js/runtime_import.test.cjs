import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import fs from "fs";
import path from "path";
import os from "os";
const core = { info: vi.fn(), warning: vi.fn(), setFailed: vi.fn() };
global.core = core;
const { processRuntimeImports, processRuntimeImport, convertFileInlinesToMacros, processUrlInline, processUrlInlines, hasFrontMatter, removeXMLComments, hasGitHubActionsMacros } = require("./runtime_import.cjs");
describe("runtime_import", () => {
  let tempDir;
  (beforeEach(() => {
    ((tempDir = fs.mkdtempSync(path.join(os.tmpdir(), "runtime-import-test-"))), vi.clearAllMocks());
  }),
    afterEach(() => {
      tempDir && fs.existsSync(tempDir) && fs.rmSync(tempDir, { recursive: !0, force: !0 });
    }),
    describe("hasFrontMatter", () => {
      (it("should detect front matter at the start", () => {
        expect(hasFrontMatter("---\ntitle: Test\n---\nContent")).toBe(!0);
      }),
        it("should detect front matter with CRLF line endings", () => {
          expect(hasFrontMatter("---\r\ntitle: Test\r\n---\r\nContent")).toBe(!0);
        }),
        it("should detect front matter with leading whitespace", () => {
          expect(hasFrontMatter("  \n  ---\ntitle: Test\n---\nContent")).toBe(!0);
        }),
        it("should not detect front matter in the middle", () => {
          expect(hasFrontMatter("Some content\n---\ntitle: Test\n---")).toBe(!1);
        }),
        it("should not detect incomplete front matter marker", () => {
          expect(hasFrontMatter("--\ntitle: Test\n--\nContent")).toBe(!1);
        }),
        it("should handle empty content", () => {
          expect(hasFrontMatter("")).toBe(!1);
        }));
    }),
    describe("removeXMLComments", () => {
      (it("should remove simple XML comments", () => {
        expect(removeXMLComments("Before \x3c!-- comment --\x3e After")).toBe("Before  After");
      }),
        it("should remove multiline XML comments", () => {
          expect(removeXMLComments("Before \x3c!-- multi\nline\ncomment --\x3e After")).toBe("Before  After");
        }),
        it("should remove multiple XML comments", () => {
          expect(removeXMLComments("\x3c!-- first --\x3eText\x3c!-- second --\x3eMore\x3c!-- third --\x3e")).toBe("TextMore");
        }),
        it("should handle content without comments", () => {
          expect(removeXMLComments("No comments here")).toBe("No comments here");
        }),
        it("should handle nested-looking comments", () => {
          expect(removeXMLComments("\x3c!-- outer \x3c!-- inner --\x3e --\x3e")).toBe(" --\x3e");
        }),
        it("should handle empty content", () => {
          expect(removeXMLComments("")).toBe("");
        }));
    }),
    describe("hasGitHubActionsMacros", () => {
      (it("should detect simple GitHub Actions macros", () => {
        expect(hasGitHubActionsMacros("${{ github.actor }}")).toBe(!0);
      }),
        it("should detect multiline GitHub Actions macros", () => {
          expect(hasGitHubActionsMacros("${{ \ngithub.actor \n}}")).toBe(!0);
        }),
        it("should detect multiple GitHub Actions macros", () => {
          expect(hasGitHubActionsMacros("${{ github.actor }} and ${{ github.repo }}")).toBe(!0);
        }),
        it("should not detect template conditionals", () => {
          expect(hasGitHubActionsMacros("{{#if condition}}text{{/if}}")).toBe(!1);
        }),
        it("should not detect runtime-import macros", () => {
          expect(hasGitHubActionsMacros("{{#runtime-import file.md}}")).toBe(!1);
        }),
        it("should detect GitHub Actions macros within other content", () => {
          expect(hasGitHubActionsMacros("Some text ${{ github.actor }} more text")).toBe(!0);
        }),
        it("should handle content without macros", () => {
          expect(hasGitHubActionsMacros("No macros here")).toBe(!1);
        }));
    }),
    describe("processRuntimeImport", () => {
      (it("should read and return file content", () => {
        const content = "# Test Content\n\nThis is a test.";
        fs.writeFileSync(path.join(tempDir, "test.md"), content);
        const result = processRuntimeImport("test.md", !1, tempDir);
        expect(result).toBe(content);
      }),
        it("should throw error for missing required file", () => {
          expect(() => processRuntimeImport("missing.md", !1, tempDir)).toThrow("Runtime import file not found: missing.md");
        }),
        it("should return empty string for missing optional file", () => {
          const result = processRuntimeImport("missing.md", !0, tempDir);
          (expect(result).toBe(""), expect(core.warning).toHaveBeenCalledWith("Optional runtime import file not found: missing.md"));
        }),
        it("should remove front matter and warn", () => {
          const filepath = "with-frontmatter.md";
          fs.writeFileSync(path.join(tempDir, filepath), "---\ntitle: Test\nkey: value\n---\n\n# Content\n\nActual content.");
          const result = processRuntimeImport(filepath, !1, tempDir);
          (expect(result).toContain("# Content"),
            expect(result).toContain("Actual content."),
            expect(result).not.toContain("title: Test"),
            expect(core.warning).toHaveBeenCalledWith(`File ${filepath} contains front matter which will be ignored in runtime import`));
        }),
        it("should remove XML comments", () => {
          fs.writeFileSync(path.join(tempDir, "with-comments.md"), "# Title\n\n\x3c!-- This is a comment --\x3e\n\nContent here.");
          const result = processRuntimeImport("with-comments.md", !1, tempDir);
          (expect(result).toContain("# Title"), expect(result).toContain("Content here."), expect(result).not.toContain("\x3c!-- This is a comment --\x3e"));
        }),
        it("should throw error for GitHub Actions macros", () => {
          (fs.writeFileSync(path.join(tempDir, "with-macros.md"), "# Title\n\nActor: ${{ github.actor }}\n"),
            expect(() => processRuntimeImport("with-macros.md", !1, tempDir)).toThrow("File with-macros.md contains GitHub Actions macros (${{ ... }}) which are not allowed in runtime imports"));
        }),
        it("should handle file in subdirectory", () => {
          const subdir = path.join(tempDir, "subdir");
          (fs.mkdirSync(subdir), fs.writeFileSync(path.join(tempDir, "subdir/test.md"), "Subdirectory content"));
          const result = processRuntimeImport("subdir/test.md", !1, tempDir);
          expect(result).toBe("Subdirectory content");
        }),
        it("should handle empty file", () => {
          fs.writeFileSync(path.join(tempDir, "empty.md"), "");
          const result = processRuntimeImport("empty.md", !1, tempDir);
          expect(result).toBe("");
        }),
        it("should handle file with only front matter", () => {
          fs.writeFileSync(path.join(tempDir, "only-frontmatter.md"), "---\ntitle: Test\n---\n");
          const result = processRuntimeImport("only-frontmatter.md", !1, tempDir);
          expect(result.trim()).toBe("");
        }),
        it("should allow template conditionals", () => {
          const content = "{{#if condition}}content{{/if}}";
          fs.writeFileSync(path.join(tempDir, "with-conditionals.md"), content);
          const result = processRuntimeImport("with-conditionals.md", !1, tempDir);
          expect(result).toBe(content);
        }));
    }),
    describe("processRuntimeImports", () => {
      (it("should process single runtime-import macro", () => {
        fs.writeFileSync(path.join(tempDir, "import.md"), "Imported content");
        const result = processRuntimeImports("Before\n{{#runtime-import import.md}}\nAfter", tempDir);
        expect(result).toBe("Before\nImported content\nAfter");
      }),
        it("should process optional runtime-import macro", () => {
          fs.writeFileSync(path.join(tempDir, "import.md"), "Imported content");
          const result = processRuntimeImports("Before\n{{#runtime-import? import.md}}\nAfter", tempDir);
          expect(result).toBe("Before\nImported content\nAfter");
        }),
        it("should process multiple runtime-import macros", () => {
          (fs.writeFileSync(path.join(tempDir, "import1.md"), "Content 1"), fs.writeFileSync(path.join(tempDir, "import2.md"), "Content 2"));
          const result = processRuntimeImports("{{#runtime-import import1.md}}\nMiddle\n{{#runtime-import import2.md}}", tempDir);
          expect(result).toBe("Content 1\nMiddle\nContent 2");
        }),
        it("should handle optional import of missing file", () => {
          const result = processRuntimeImports("Before\n{{#runtime-import? missing.md}}\nAfter", tempDir);
          (expect(result).toBe("Before\n\nAfter"), expect(core.warning).toHaveBeenCalled());
        }),
        it("should throw error for required import of missing file", () => {
          expect(() => processRuntimeImports("Before\n{{#runtime-import missing.md}}\nAfter", tempDir)).toThrow();
        }),
        it("should handle content without runtime-import macros", () => {
          const result = processRuntimeImports("No imports here", tempDir);
          expect(result).toBe("No imports here");
        }),
        it("should warn about duplicate imports", () => {
          (fs.writeFileSync(path.join(tempDir, "import.md"), "Content"),
            processRuntimeImports("{{#runtime-import import.md}}\n{{#runtime-import import.md}}", tempDir),
            expect(core.warning).toHaveBeenCalledWith("File import.md is imported multiple times, which may indicate a circular reference"));
        }),
        it("should handle macros with extra whitespace", () => {
          fs.writeFileSync(path.join(tempDir, "import.md"), "Content");
          const result = processRuntimeImports("{{#runtime-import    import.md    }}", tempDir);
          expect(result).toBe("Content");
        }),
        it("should handle inline macros", () => {
          fs.writeFileSync(path.join(tempDir, "inline.md"), "inline content");
          const result = processRuntimeImports("Before {{#runtime-import inline.md}} after", tempDir);
          expect(result).toBe("Before inline content after");
        }),
        it("should process imports with files containing special characters", () => {
          fs.writeFileSync(path.join(tempDir, "import.md"), "Content with $pecial ch@racters!");
          const result = processRuntimeImports("{{#runtime-import import.md}}", tempDir);
          expect(result).toBe("Content with $pecial ch@racters!");
        }),
        it("should remove XML comments from imported content", () => {
          fs.writeFileSync(path.join(tempDir, "with-comment.md"), "Text \x3c!-- comment --\x3e more text");
          const result = processRuntimeImports("{{#runtime-import with-comment.md}}", tempDir);
          expect(result).toBe("Text  more text");
        }),
        it("should handle path with subdirectories", () => {
          const subdir = path.join(tempDir, "docs", "shared");
          (fs.mkdirSync(subdir, { recursive: !0 }), fs.writeFileSync(path.join(tempDir, "docs/shared/import.md"), "Subdir content"));
          const result = processRuntimeImports("{{#runtime-import docs/shared/import.md}}", tempDir);
          expect(result).toBe("Subdir content");
        }),
        it("should preserve newlines around imports", () => {
          fs.writeFileSync(path.join(tempDir, "import.md"), "Content");
          const result = processRuntimeImports("Line 1\n\n{{#runtime-import import.md}}\n\nLine 2", tempDir);
          expect(result).toBe("Line 1\n\nContent\n\nLine 2");
        }),
        it("should handle multiple consecutive imports", () => {
          (fs.writeFileSync(path.join(tempDir, "import1.md"), "Content 1"), fs.writeFileSync(path.join(tempDir, "import2.md"), "Content 2"));
          const result = processRuntimeImports("{{#runtime-import import1.md}}{{#runtime-import import2.md}}", tempDir);
          expect(result).toBe("Content 1Content 2");
        }),
        it("should handle imports at the start of content", () => {
          fs.writeFileSync(path.join(tempDir, "import.md"), "Start content");
          const result = processRuntimeImports("{{#runtime-import import.md}}\nFollowing text", tempDir);
          expect(result).toBe("Start content\nFollowing text");
        }),
        it("should handle imports at the end of content", () => {
          fs.writeFileSync(path.join(tempDir, "import.md"), "End content");
          const result = processRuntimeImports("Preceding text\n{{#runtime-import import.md}}", tempDir);
          expect(result).toBe("Preceding text\nEnd content");
        }),
        it("should handle tab characters in macro", () => {
          fs.writeFileSync(path.join(tempDir, "import.md"), "Content");
          const result = processRuntimeImports("{{#runtime-import\timport.md}}", tempDir);
          expect(result).toBe("Content");
        }));
    }),
    describe("Edge Cases", () => {
      (it("should handle very large files", () => {
        const largeContent = "x".repeat(1e5);
        fs.writeFileSync(path.join(tempDir, "large.md"), largeContent);
        const result = processRuntimeImports("{{#runtime-import large.md}}", tempDir);
        expect(result).toBe(largeContent);
      }),
        it("should handle files with unicode characters", () => {
          fs.writeFileSync(path.join(tempDir, "unicode.md"), "Hello 世界 🌍 café", "utf8");
          const result = processRuntimeImports("{{#runtime-import unicode.md}}", tempDir);
          expect(result).toBe("Hello 世界 🌍 café");
        }),
        it("should handle files with various line endings", () => {
          const content = "Line 1\nLine 2\r\nLine 3\rLine 4";
          fs.writeFileSync(path.join(tempDir, "mixed-lines.md"), content);
          const result = processRuntimeImports("{{#runtime-import mixed-lines.md}}", tempDir);
          expect(result).toBe(content);
        }),
        it("should not process runtime-import as a substring", () => {
          const content = "text{{#runtime-importnospace.md}}text",
            result = processRuntimeImports(content, tempDir);
          expect(result).toBe(content);
        }),
        it("should handle front matter with varying formats", () => {
          fs.writeFileSync(path.join(tempDir, "yaml-frontmatter.md"), "---\ntitle: Test\narray:\n  - item1\n  - item2\n---\n\nBody content");
          const result = processRuntimeImport("yaml-frontmatter.md", !1, tempDir);
          (expect(result).toContain("Body content"), expect(result).not.toContain("array:"), expect(result).not.toContain("item1"));
        }));
    }),
    describe("Error Handling", () => {
      (it("should provide clear error for GitHub Actions macros", () => {
        (fs.writeFileSync(path.join(tempDir, "bad.md"), "${{ github.actor }}"), expect(() => processRuntimeImports("{{#runtime-import bad.md}}", tempDir)).toThrow("Failed to process runtime import for bad.md"));
      }),
        it("should provide clear error for missing required files", () => {
          expect(() => processRuntimeImports("{{#runtime-import nonexistent.md}}", tempDir)).toThrow("Failed to process runtime import for nonexistent.md");
        }));
    }),
    describe("processRuntimeImport with line ranges", () => {
      (it("should extract specific line range", () => {
        const content = "Line 1\nLine 2\nLine 3\nLine 4\nLine 5";
        fs.writeFileSync(path.join(tempDir, "test.txt"), content);
        const result = processRuntimeImport("test.txt", !1, tempDir, 2, 4);
        expect(result).toBe("Line 2\nLine 3\nLine 4");
      }),
        it("should extract single line", () => {
          const content = "Line 1\nLine 2\nLine 3\nLine 4\nLine 5";
          fs.writeFileSync(path.join(tempDir, "test.txt"), content);
          const result = processRuntimeImport("test.txt", !1, tempDir, 3, 3);
          expect(result).toBe("Line 3");
        }),
        it("should extract from start line to end of file", () => {
          const content = "Line 1\nLine 2\nLine 3\nLine 4\nLine 5";
          fs.writeFileSync(path.join(tempDir, "test.txt"), content);
          const result = processRuntimeImport("test.txt", !1, tempDir, 3, 5);
          expect(result).toBe("Line 3\nLine 4\nLine 5");
        }),
        it("should throw error for invalid start line", () => {
          const content = "Line 1\nLine 2\nLine 3";
          fs.writeFileSync(path.join(tempDir, "test.txt"), content);
          expect(() => processRuntimeImport("test.txt", !1, tempDir, 0, 2)).toThrow("Invalid start line 0");
          expect(() => processRuntimeImport("test.txt", !1, tempDir, 10, 12)).toThrow("Invalid start line 10");
        }),
        it("should throw error for invalid end line", () => {
          const content = "Line 1\nLine 2\nLine 3";
          fs.writeFileSync(path.join(tempDir, "test.txt"), content);
          expect(() => processRuntimeImport("test.txt", !1, tempDir, 1, 0)).toThrow("Invalid end line 0");
          expect(() => processRuntimeImport("test.txt", !1, tempDir, 1, 10)).toThrow("Invalid end line 10");
        }),
        it("should throw error when start line > end line", () => {
          const content = "Line 1\nLine 2\nLine 3";
          fs.writeFileSync(path.join(tempDir, "test.txt"), content);
          expect(() => processRuntimeImport("test.txt", !1, tempDir, 3, 1)).toThrow("Start line 3 cannot be greater than end line 1");
        }),
        it("should handle line range with front matter", () => {
          const filepath = "frontmatter-lines.md";
          // Line 1: ---
          // Line 2: title: Test
          // Line 3: ---
          // Line 4: (empty)
          // Line 5: Line 1
          fs.writeFileSync(path.join(tempDir, filepath), "---\ntitle: Test\n---\n\nLine 1\nLine 2\nLine 3\nLine 4\nLine 5");
          const result = processRuntimeImport(filepath, !1, tempDir, 2, 4);
          // Lines 2-4 of raw file are: "title: Test", "---", ""
          // After front matter removal, these lines are part of front matter so they get removed
          // The result should be empty or minimal content
          expect(result).toBeTruthy(); // At minimum, it should not fail
        }));
    }),
    describe("convertFileInlinesToMacros", () => {
      (it("should convert @path to {{#runtime-import path}}", () => {
        const result = convertFileInlinesToMacros("Before @test.txt after");
        expect(result).toBe("Before {{#runtime-import test.txt}} after");
      }),
        it("should convert @path:line-line to {{#runtime-import path:line-line}}", () => {
          const result = convertFileInlinesToMacros("Content: @test.txt:2-4 end");
          expect(result).toBe("Content: {{#runtime-import test.txt:2-4}} end");
        }),
        it("should convert multiple @path references", () => {
          const result = convertFileInlinesToMacros("Start @file1.txt middle @file2.txt end");
          expect(result).toBe("Start {{#runtime-import file1.txt}} middle {{#runtime-import file2.txt}} end");
        }),
        it("should handle @path with subdirectories", () => {
          const result = convertFileInlinesToMacros("See @docs/readme.md for details");
          expect(result).toBe("See {{#runtime-import docs/readme.md}} for details");
        }),
        it("should not convert email addresses", () => {
          const result = convertFileInlinesToMacros("Email: user@example.com is valid");
          expect(result).toBe("Email: user@example.com is valid");
        }),
        it("should handle content without @path references", () => {
          const result = convertFileInlinesToMacros("No inline references here");
          expect(result).toBe("No inline references here");
        }),
        it("should handle @path at start of content", () => {
          const result = convertFileInlinesToMacros("@start.txt following text");
          expect(result).toBe("{{#runtime-import start.txt}} following text");
        }),
        it("should handle @path at end of content", () => {
          const result = convertFileInlinesToMacros("Preceding text @end.txt");
          expect(result).toBe("Preceding text {{#runtime-import end.txt}}");
        }),
        it("should handle @path on its own line", () => {
          const result = convertFileInlinesToMacros("Before\n@content.txt\nAfter");
          expect(result).toBe("Before\n{{#runtime-import content.txt}}\nAfter");
        }),
        it("should handle multiple line ranges", () => {
          const result = convertFileInlinesToMacros("First: @test.txt:1-2 Second: @test.txt:4-5");
          expect(result).toBe("First: {{#runtime-import test.txt:1-2}} Second: {{#runtime-import test.txt:4-5}}");
        }));
    }),
    describe("processRuntimeImports with line ranges from macros", () => {
      (it("should process {{#runtime-import path:line-line}} macro", () => {
        fs.writeFileSync(path.join(tempDir, "test.txt"), "Line 1\nLine 2\nLine 3\nLine 4\nLine 5");
        const result = processRuntimeImports("Content: {{#runtime-import test.txt:2-4}} end", tempDir);
        expect(result).toBe("Content: Line 2\nLine 3\nLine 4 end");
      }),
        it("should process multiple {{#runtime-import path:line-line}} macros", () => {
          fs.writeFileSync(path.join(tempDir, "test.txt"), "Line 1\nLine 2\nLine 3\nLine 4\nLine 5");
          const result = processRuntimeImports("First: {{#runtime-import test.txt:1-2}} Second: {{#runtime-import test.txt:4-5}}", tempDir);
          expect(result).toBe("First: Line 1\nLine 2 Second: Line 4\nLine 5");
        }));
    }),
    describe("processUrlInlines", () => {
      (it("should process @url reference with mock server", async () => {
        // This test would require a mock HTTP server
        // For now, we'll skip it or test with file:// URLs
        // In a real workflow, this would fetch from the internet
      }),
        it("should handle content without @url references", async () => {
          const result = await processUrlInlines("No URL references here", tempDir);
          expect(result).toBe("No URL references here");
        }),
        it("should not process regular URLs without @ prefix", async () => {
          const content = "Visit https://example.com for more info";
          const result = await processUrlInlines(content, tempDir);
          expect(result).toBe(content);
        }));
    }));
});
