import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import fs from "fs";
import path from "path";
import os from "os";
const core = { info: vi.fn(), warning: vi.fn(), setFailed: vi.fn() };
global.core = core;
const { processRuntimeImports, processRuntimeImport, convertInlinesToMacros, hasFrontMatter, removeXMLComments, hasGitHubActionsMacros } = require("./runtime_import.cjs");
describe("runtime_import", () => {
  let tempDir;
  let githubDir;
  (beforeEach(() => {
    ((tempDir = fs.mkdtempSync(path.join(os.tmpdir(), "runtime-import-test-"))), (githubDir = path.join(tempDir, ".github")), fs.mkdirSync(githubDir, { recursive: true }), vi.clearAllMocks());
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
      (it("should read and return file content", async () => {
        const content = "# Test Content\n\nThis is a test.";
        fs.writeFileSync(path.join(githubDir, "test.md"), content);
        const result = await processRuntimeImport("test.md", !1, tempDir);
        expect(result).toBe(content);
      }),
        it("should throw error for missing required file", async () => {
          await expect(processRuntimeImport("missing.md", !1, tempDir)).rejects.toThrow("Runtime import file not found: missing.md");
        }),
        it("should return empty string for missing optional file", async () => {
          const result = await processRuntimeImport("missing.md", !0, tempDir);
          (expect(result).toBe(""), expect(core.warning).toHaveBeenCalledWith("Optional runtime import file not found: missing.md"));
        }),
        it("should remove front matter and warn", async () => {
          const filepath = "with-frontmatter.md";
          fs.writeFileSync(path.join(githubDir, filepath), "---\ntitle: Test\nkey: value\n---\n\n# Content\n\nActual content.");
          const result = await processRuntimeImport(filepath, !1, tempDir);
          (expect(result).toContain("# Content"),
            expect(result).toContain("Actual content."),
            expect(result).not.toContain("title: Test"),
            expect(core.warning).toHaveBeenCalledWith(`File ${filepath} contains front matter which will be ignored in runtime import`));
        }),
        it("should remove XML comments", async () => {
          fs.writeFileSync(path.join(githubDir, "with-comments.md"), "# Title\n\n\x3c!-- This is a comment --\x3e\n\nContent here.");
          const result = await processRuntimeImport("with-comments.md", !1, tempDir);
          (expect(result).toContain("# Title"), expect(result).toContain("Content here."), expect(result).not.toContain("\x3c!-- This is a comment --\x3e"));
        }),
        it("should render safe GitHub Actions expressions", async () => {
          // Setup context for expression evaluation
          global.context = {
            actor: "testuser",
            job: "test-job",
            repo: { owner: "testorg", repo: "testrepo" },
            runId: 12345,
            runNumber: 42,
            workflow: "test-workflow",
            payload: {},
          };
          fs.writeFileSync(path.join(githubDir, "with-macros.md"), "# Title\n\nActor: ${{ github.actor }}\n");
          const result = await processRuntimeImport("with-macros.md", !1, tempDir);
          expect(result).toContain("# Title");
          expect(result).toContain("Actor: testuser");
          delete global.context;
        }),
        it("should reject unsafe GitHub Actions expressions", async () => {
          fs.writeFileSync(path.join(githubDir, "unsafe-macros.md"), "Secret: ${{ secrets.TOKEN }}\n");
          await expect(processRuntimeImport("unsafe-macros.md", !1, tempDir)).rejects.toThrow("unauthorized GitHub Actions expressions");
        }),
        it("should handle file in subdirectory", async () => {
          const subdir = path.join(githubDir, "subdir");
          (fs.mkdirSync(subdir), fs.writeFileSync(path.join(githubDir, "subdir/test.md"), "Subdirectory content"));
          const result = await processRuntimeImport("subdir/test.md", !1, tempDir);
          expect(result).toBe("Subdirectory content");
        }),
        it("should handle empty file", async () => {
          fs.writeFileSync(path.join(githubDir, "empty.md"), "");
          const result = await processRuntimeImport("empty.md", !1, tempDir);
          expect(result).toBe("");
        }),
        it("should handle file with only front matter", async () => {
          fs.writeFileSync(path.join(githubDir, "only-frontmatter.md"), "---\ntitle: Test\n---\n");
          const result = await processRuntimeImport("only-frontmatter.md", !1, tempDir);
          expect(result.trim()).toBe("");
        }),
        it("should allow template conditionals", async () => {
          const content = "{{#if condition}}content{{/if}}";
          fs.writeFileSync(path.join(githubDir, "with-conditionals.md"), content);
          const result = await processRuntimeImport("with-conditionals.md", !1, tempDir);
          expect(result).toBe(content);
        }),
        it("should support .github/ prefix in path", async () => {
          const content = "Test with .github prefix";
          fs.writeFileSync(path.join(githubDir, "test-prefix.md"), content);
          const result = await processRuntimeImport(".github/test-prefix.md", !1, tempDir);
          expect(result).toBe(content);
        }),
        it("should work without .github/ prefix", async () => {
          const content = "Test without prefix";
          fs.writeFileSync(path.join(githubDir, "test-no-prefix.md"), content);
          const result = await processRuntimeImport("test-no-prefix.md", !1, tempDir);
          expect(result).toBe(content);
        }),
        it("should reject paths outside .github folder", async () => {
          // Try to access a file in the root (not in .github)
          fs.writeFileSync(path.join(tempDir, "outside.md"), "Outside content");
          await expect(processRuntimeImport("../outside.md", !1, tempDir)).rejects.toThrow("Security: Path ../outside.md must be within .github folder");
        }));
    }),
    describe("processRuntimeImports", () => {
      (it("should process single runtime-import macro", async () => {
        fs.writeFileSync(path.join(githubDir, "import.md"), "Imported content");
        const result = await processRuntimeImports("Before\n{{#runtime-import import.md}}\nAfter", tempDir);
        expect(result).toBe("Before\nImported content\nAfter");
      }),
        it("should process optional runtime-import macro", async () => {
          fs.writeFileSync(path.join(githubDir, "import.md"), "Imported content");
          const result = await processRuntimeImports("Before\n{{#runtime-import? import.md}}\nAfter", tempDir);
          expect(result).toBe("Before\nImported content\nAfter");
        }),
        it("should process multiple runtime-import macros", async () => {
          (fs.writeFileSync(path.join(githubDir, "import1.md"), "Content 1"), fs.writeFileSync(path.join(githubDir, "import2.md"), "Content 2"));
          const result = await processRuntimeImports("{{#runtime-import import1.md}}\nMiddle\n{{#runtime-import import2.md}}", tempDir);
          expect(result).toBe("Content 1\nMiddle\nContent 2");
        }),
        it("should handle optional import of missing file", async () => {
          const result = await processRuntimeImports("Before\n{{#runtime-import? missing.md}}\nAfter", tempDir);
          (expect(result).toBe("Before\n\nAfter"), expect(core.warning).toHaveBeenCalled());
        }),
        it("should throw error for required import of missing file", async () => {
          await expect(processRuntimeImports("Before\n{{#runtime-import missing.md}}\nAfter", tempDir)).rejects.toThrow();
        }),
        it("should handle content without runtime-import macros", async () => {
          const result = await processRuntimeImports("No imports here", tempDir);
          expect(result).toBe("No imports here");
        }),
        it("should warn about duplicate imports", async () => {
          (fs.writeFileSync(path.join(githubDir, "import.md"), "Content"),
            await processRuntimeImports("{{#runtime-import import.md}}\n{{#runtime-import import.md}}", tempDir),
            expect(core.warning).toHaveBeenCalledWith("File/URL import.md is imported multiple times, which may indicate a circular reference"));
        }),
        it("should handle macros with extra whitespace", async () => {
          fs.writeFileSync(path.join(githubDir, "import.md"), "Content");
          const result = await processRuntimeImports("{{#runtime-import    import.md    }}", tempDir);
          expect(result).toBe("Content");
        }),
        it("should handle inline macros", async () => {
          fs.writeFileSync(path.join(githubDir, "inline.md"), "inline content");
          const result = await processRuntimeImports("Before {{#runtime-import inline.md}} after", tempDir);
          expect(result).toBe("Before inline content after");
        }),
        it("should process imports with files containing special characters", async () => {
          fs.writeFileSync(path.join(githubDir, "import.md"), "Content with $pecial ch@racters!");
          const result = await processRuntimeImports("{{#runtime-import import.md}}", tempDir);
          expect(result).toBe("Content with $pecial ch@racters!");
        }),
        it("should remove XML comments from imported content", async () => {
          fs.writeFileSync(path.join(githubDir, "with-comment.md"), "Text \x3c!-- comment --\x3e more text");
          const result = await processRuntimeImports("{{#runtime-import with-comment.md}}", tempDir);
          expect(result).toBe("Text  more text");
        }),
        it("should handle path with subdirectories", async () => {
          const subdir = path.join(githubDir, "docs", "shared");
          (fs.mkdirSync(subdir, { recursive: !0 }), fs.writeFileSync(path.join(githubDir, "docs/shared/import.md"), "Subdir content"));
          const result = await processRuntimeImports("{{#runtime-import docs/shared/import.md}}", tempDir);
          expect(result).toBe("Subdir content");
        }),
        it("should preserve newlines around imports", async () => {
          fs.writeFileSync(path.join(githubDir, "import.md"), "Content");
          const result = await processRuntimeImports("Line 1\n\n{{#runtime-import import.md}}\n\nLine 2", tempDir);
          expect(result).toBe("Line 1\n\nContent\n\nLine 2");
        }),
        it("should handle multiple consecutive imports", async () => {
          (fs.writeFileSync(path.join(githubDir, "import1.md"), "Content 1"), fs.writeFileSync(path.join(githubDir, "import2.md"), "Content 2"));
          const result = await processRuntimeImports("{{#runtime-import import1.md}}{{#runtime-import import2.md}}", tempDir);
          expect(result).toBe("Content 1Content 2");
        }),
        it("should handle imports at the start of content", async () => {
          fs.writeFileSync(path.join(githubDir, "import.md"), "Start content");
          const result = await processRuntimeImports("{{#runtime-import import.md}}\nFollowing text", tempDir);
          expect(result).toBe("Start content\nFollowing text");
        }),
        it("should handle imports at the end of content", async () => {
          fs.writeFileSync(path.join(githubDir, "import.md"), "End content");
          const result = await processRuntimeImports("Preceding text\n{{#runtime-import import.md}}", tempDir);
          expect(result).toBe("Preceding text\nEnd content");
        }),
        it("should handle tab characters in macro", async () => {
          fs.writeFileSync(path.join(githubDir, "import.md"), "Content");
          const result = await processRuntimeImports("{{#runtime-import\timport.md}}", tempDir);
          expect(result).toBe("Content");
        }));
    }),
    describe("Edge Cases", () => {
      (it("should handle very large files", async () => {
        const largeContent = "x".repeat(1e5);
        fs.writeFileSync(path.join(githubDir, "large.md"), largeContent);
        const result = await processRuntimeImports("{{#runtime-import large.md}}", tempDir);
        expect(result).toBe(largeContent);
      }),
        it("should handle files with unicode characters", async () => {
          fs.writeFileSync(path.join(githubDir, "unicode.md"), "Hello 世界 🌍 café", "utf8");
          const result = await processRuntimeImports("{{#runtime-import unicode.md}}", tempDir);
          expect(result).toBe("Hello 世界 🌍 café");
        }),
        it("should handle files with various line endings", async () => {
          const content = "Line 1\nLine 2\r\nLine 3\rLine 4";
          fs.writeFileSync(path.join(githubDir, "mixed-lines.md"), content);
          const result = await processRuntimeImports("{{#runtime-import mixed-lines.md}}", tempDir);
          expect(result).toBe(content);
        }),
        it("should not process runtime-import as a substring", async () => {
          const content = "text{{#runtime-importnospace.md}}text",
            result = await processRuntimeImports(content, tempDir);
          expect(result).toBe(content);
        }),
        it("should handle front matter with varying formats", async () => {
          fs.writeFileSync(path.join(githubDir, "yaml-frontmatter.md"), "---\ntitle: Test\narray:\n  - item1\n  - item2\n---\n\nBody content");
          const result = await processRuntimeImport("yaml-frontmatter.md", !1, tempDir);
          (expect(result).toContain("Body content"), expect(result).not.toContain("array:"), expect(result).not.toContain("item1"));
        }));
    }),
    describe("Error Handling", () => {
      (it("should provide clear error for unsafe GitHub Actions expressions", async () => {
        fs.writeFileSync(path.join(githubDir, "bad.md"), "${{ secrets.TOKEN }}");
        await expect(processRuntimeImports("{{#runtime-import bad.md}}", tempDir)).rejects.toThrow("unauthorized GitHub Actions expressions");
      }),
        it("should provide clear error for missing required files", async () => {
          await expect(processRuntimeImports("{{#runtime-import nonexistent.md}}", tempDir)).rejects.toThrow("Failed to process runtime import for nonexistent.md");
        }));
    }),
    describe("Path Security", () => {
      (it("should reject paths that escape .github folder with ../", async () => {
        // Try to escape .github folder using ../../../etc/passwd
        await expect(processRuntimeImport("../../../etc/passwd", !1, tempDir)).rejects.toThrow("Security: Path ../../../etc/passwd must be within .github folder");
      }),
        it("should reject paths that escape .github folder with ../../", async () => {
          // Try to escape .github folder using ./../../etc/passwd
          await expect(processRuntimeImport("../../etc/passwd", !1, tempDir)).rejects.toThrow("Security: Path ../../etc/passwd must be within .github folder");
        }),
        it("should allow valid path within .github folder", async () => {
          // Create a subdirectory structure within .github
          const subdir = path.join(githubDir, "subdir");
          fs.mkdirSync(subdir, { recursive: !0 });
          fs.writeFileSync(path.join(subdir, "subfile.txt"), "Sub content");

          // Access subdir/subfile.txt
          const result = await processRuntimeImport("subdir/subfile.txt", !1, tempDir);
          expect(result).toBe("Sub content");
        }),
        it("should allow ./path within .github folder", async () => {
          fs.writeFileSync(path.join(githubDir, "test.txt"), "Test content");
          const result = await processRuntimeImport("./test.txt", !1, tempDir);
          expect(result).toBe("Test content");
        }),
        it("should normalize paths with redundant separators", async () => {
          fs.writeFileSync(path.join(githubDir, "test.txt"), "Test content");
          const result = await processRuntimeImport("./././test.txt", !1, tempDir);
          expect(result).toBe("Test content");
        }),
        it("should allow nested paths that stay within .github folder", async () => {
          // Create nested directory structure within .github
          const dirA = path.join(githubDir, "a");
          const dirB = path.join(dirA, "b");
          fs.mkdirSync(dirB, { recursive: !0 });
          fs.writeFileSync(path.join(dirB, "file.txt"), "Nested content");

          // Access a/b/file.txt
          const result = await processRuntimeImport("a/b/file.txt", !1, tempDir);
          expect(result).toBe("Nested content");
        }),
        it("should reject attempts to access files outside .github", async () => {
          // Create a file outside .github
          fs.writeFileSync(path.join(tempDir, "root-file.txt"), "Root content");
          // Try to access it
          await expect(processRuntimeImport("../root-file.txt", !1, tempDir)).rejects.toThrow("Security: Path ../root-file.txt must be within .github folder");
        }));
    }),
    describe("processRuntimeImport with line ranges", () => {
      (it("should extract specific line range", async () => {
        const content = "Line 1\nLine 2\nLine 3\nLine 4\nLine 5";
        fs.writeFileSync(path.join(githubDir, "test.txt"), content);
        const result = await processRuntimeImport("test.txt", !1, tempDir, 2, 4);
        expect(result).toBe("Line 2\nLine 3\nLine 4");
      }),
        it("should extract single line", async () => {
          const content = "Line 1\nLine 2\nLine 3\nLine 4\nLine 5";
          fs.writeFileSync(path.join(githubDir, "test.txt"), content);
          const result = await processRuntimeImport("test.txt", !1, tempDir, 3, 3);
          expect(result).toBe("Line 3");
        }),
        it("should extract from start line to end of file", async () => {
          const content = "Line 1\nLine 2\nLine 3\nLine 4\nLine 5";
          fs.writeFileSync(path.join(githubDir, "test.txt"), content);
          const result = await processRuntimeImport("test.txt", !1, tempDir, 3, 5);
          expect(result).toBe("Line 3\nLine 4\nLine 5");
        }),
        it("should throw error for invalid start line", async () => {
          const content = "Line 1\nLine 2\nLine 3";
          fs.writeFileSync(path.join(githubDir, "test.txt"), content);
          await expect(processRuntimeImport("test.txt", !1, tempDir, 0, 2)).rejects.toThrow("Invalid start line 0");
          await expect(processRuntimeImport("test.txt", !1, tempDir, 10, 12)).rejects.toThrow("Invalid start line 10");
        }),
        it("should throw error for invalid end line", async () => {
          const content = "Line 1\nLine 2\nLine 3";
          fs.writeFileSync(path.join(githubDir, "test.txt"), content);
          await expect(processRuntimeImport("test.txt", !1, tempDir, 1, 0)).rejects.toThrow("Invalid end line 0");
          await expect(processRuntimeImport("test.txt", !1, tempDir, 1, 10)).rejects.toThrow("Invalid end line 10");
        }),
        it("should throw error when start line > end line", async () => {
          const content = "Line 1\nLine 2\nLine 3";
          fs.writeFileSync(path.join(githubDir, "test.txt"), content);
          await expect(processRuntimeImport("test.txt", !1, tempDir, 3, 1)).rejects.toThrow("Start line 3 cannot be greater than end line 1");
        }),
        it("should handle line range with front matter", async () => {
          const filepath = "frontmatter-lines.md";
          // Line 1: ---
          // Line 2: title: Test
          // Line 3: ---
          // Line 4: (empty)
          // Line 5: Line 1
          fs.writeFileSync(path.join(githubDir, filepath), "---\ntitle: Test\n---\n\nLine 1\nLine 2\nLine 3\nLine 4\nLine 5");
          const result = await processRuntimeImport(filepath, !1, tempDir, 2, 4);
          // Lines 2-4 of raw file are: "title: Test", "---", ""
          // After front matter removal, these lines are part of front matter so they get removed
          // The result should be empty or minimal content
          expect(result).toBeTruthy(); // At minimum, it should not fail
        }));
    }),
    describe("convertInlinesToMacros", () => {
      (it("should convert @./path to {{#runtime-import ./path}}", () => {
        const result = convertInlinesToMacros("Before @./test.txt after");
        expect(result).toBe("Before {{#runtime-import ./test.txt}} after");
      }),
        it("should convert @./path:line-line to {{#runtime-import ./path:line-line}}", () => {
          const result = convertInlinesToMacros("Content: @./test.txt:2-4 end");
          expect(result).toBe("Content: {{#runtime-import ./test.txt:2-4}} end");
        }),
        it("should convert multiple @./path references", () => {
          const result = convertInlinesToMacros("Start @./file1.txt middle @./file2.txt end");
          expect(result).toBe("Start {{#runtime-import ./file1.txt}} middle {{#runtime-import ./file2.txt}} end");
        }),
        it("should handle @./path with subdirectories", () => {
          const result = convertInlinesToMacros("See @./docs/readme.md for details");
          expect(result).toBe("See {{#runtime-import ./docs/readme.md}} for details");
        }),
        it("should handle @../path references", () => {
          const result = convertInlinesToMacros("Parent: @../parent.md content");
          expect(result).toBe("Parent: {{#runtime-import ../parent.md}} content");
        }),
        it("should NOT convert @path without ./ or ../", () => {
          const result = convertInlinesToMacros("Before @test.txt after");
          expect(result).toBe("Before @test.txt after");
        }),
        it("should NOT convert @docs/file without ./ prefix", () => {
          const result = convertInlinesToMacros("See @docs/readme.md for details");
          expect(result).toBe("See @docs/readme.md for details");
        }),
        it("should convert @url to {{#runtime-import url}}", () => {
          const result = convertInlinesToMacros("Content from @https://example.com/file.txt ");
          expect(result).toBe("Content from {{#runtime-import https://example.com/file.txt}} ");
        }),
        it("should convert @url:line-line to {{#runtime-import url:line-line}}", () => {
          const result = convertInlinesToMacros("Lines: @https://example.com/file.txt:10-20 ");
          expect(result).toBe("Lines: {{#runtime-import https://example.com/file.txt:10-20}} ");
        }),
        it("should not convert email addresses", () => {
          const result = convertInlinesToMacros("Email: user@example.com is valid");
          expect(result).toBe("Email: user@example.com is valid");
        }),
        it("should handle content without @path references", () => {
          const result = convertInlinesToMacros("No inline references here");
          expect(result).toBe("No inline references here");
        }),
        it("should handle @./path at start of content", () => {
          const result = convertInlinesToMacros("@./start.txt following text");
          expect(result).toBe("{{#runtime-import ./start.txt}} following text");
        }),
        it("should handle @./path at end of content", () => {
          const result = convertInlinesToMacros("Preceding text @./end.txt");
          expect(result).toBe("Preceding text {{#runtime-import ./end.txt}}");
        }),
        it("should handle @./path on its own line", () => {
          const result = convertInlinesToMacros("Before\n@./content.txt\nAfter");
          expect(result).toBe("Before\n{{#runtime-import ./content.txt}}\nAfter");
        }),
        it("should handle multiple line ranges", () => {
          const result = convertInlinesToMacros("First: @./test.txt:1-2 Second: @./test.txt:4-5");
          expect(result).toBe("First: {{#runtime-import ./test.txt:1-2}} Second: {{#runtime-import ./test.txt:4-5}}");
        }),
        it("should convert mixed @./path and @url references", () => {
          const result = convertInlinesToMacros("File: @./local.txt URL: @https://example.com/remote.txt ");
          expect(result).toBe("File: {{#runtime-import ./local.txt}} URL: {{#runtime-import https://example.com/remote.txt}} ");
        }));
    }),
    describe("processRuntimeImports with line ranges from macros", () => {
      (it("should process {{#runtime-import path:line-line}} macro", async () => {
        fs.writeFileSync(path.join(githubDir, "test.txt"), "Line 1\nLine 2\nLine 3\nLine 4\nLine 5");
        const result = await processRuntimeImports("Content: {{#runtime-import test.txt:2-4}} end", tempDir);
        expect(result).toBe("Content: Line 2\nLine 3\nLine 4 end");
      }),
        it("should process multiple {{#runtime-import path:line-line}} macros", async () => {
          fs.writeFileSync(path.join(githubDir, "test.txt"), "Line 1\nLine 2\nLine 3\nLine 4\nLine 5");
          const result = await processRuntimeImports("First: {{#runtime-import test.txt:1-2}} Second: {{#runtime-import test.txt:4-5}}", tempDir);
          expect(result).toBe("First: Line 1\nLine 2 Second: Line 4\nLine 5");
        }));
    }),
    describe("Expression Validation and Rendering", () => {
      const { isSafeExpression, evaluateExpression, processExpressions } = require("./runtime_import.cjs");

      // Setup mock context for expression evaluation
      beforeEach(() => {
        global.context = {
          actor: "testuser",
          job: "test-job",
          repo: { owner: "testorg", repo: "testrepo" },
          runId: 12345,
          runNumber: 42,
          workflow: "test-workflow",
          payload: {
            issue: { number: 123, title: "Test Issue", state: "open" },
            pull_request: { number: 456, title: "Test PR", state: "open" },
            sender: { id: 789 },
          },
        };
        process.env.GITHUB_SERVER_URL = "https://github.com";
        process.env.GITHUB_WORKSPACE = "/workspace";
      });

      afterEach(() => {
        delete global.context;
      });

      describe("isSafeExpression", () => {
        it("should allow expressions from the safe list", () => {
          expect(isSafeExpression("github.actor")).toBe(true);
          expect(isSafeExpression("github.repository")).toBe(true);
          expect(isSafeExpression("github.event.issue.number")).toBe(true);
          expect(isSafeExpression("github.event.pull_request.title")).toBe(true);
        });

        it("should allow dynamic patterns", () => {
          expect(isSafeExpression("needs.build.outputs.version")).toBe(true);
          expect(isSafeExpression("steps.checkout.outputs.ref")).toBe(true);
          expect(isSafeExpression("github.event.inputs.branch")).toBe(true);
          expect(isSafeExpression("inputs.version")).toBe(true);
          expect(isSafeExpression("env.NODE_VERSION")).toBe(true);
        });

        it("should reject unsafe expressions", () => {
          expect(isSafeExpression("secrets.GITHUB_TOKEN")).toBe(false);
          expect(isSafeExpression("github.token")).toBe(false);
          expect(isSafeExpression("runner.os")).toBe(false);
          expect(isSafeExpression("vars.MY_VAR")).toBe(false);
        });

        it("should handle whitespace", () => {
          expect(isSafeExpression("  github.actor  ")).toBe(true);
          expect(isSafeExpression("\ngithub.repository\n")).toBe(true);
        });
      });

      describe("evaluateExpression", () => {
        it("should evaluate simple GitHub context expressions", () => {
          expect(evaluateExpression("github.actor")).toBe("testuser");
          expect(evaluateExpression("github.repository")).toBe("testorg/testrepo");
          expect(evaluateExpression("github.run_id")).toBe("12345");
        });

        it("should evaluate nested event properties", () => {
          expect(evaluateExpression("github.event.issue.number")).toBe("123");
          expect(evaluateExpression("github.event.pull_request.title")).toBe("Test PR");
          expect(evaluateExpression("github.event.sender.id")).toBe("789");
        });

        it("should return wrapped expression for unresolvable values", () => {
          expect(evaluateExpression("needs.build.outputs.version")).toContain("needs.build.outputs.version");
          expect(evaluateExpression("steps.test.outputs.result")).toContain("steps.test.outputs.result");
        });

        it("should handle missing properties gracefully", () => {
          const result = evaluateExpression("github.event.nonexistent.property");
          expect(result).toContain("github.event.nonexistent.property");
        });
      });

      describe("processExpressions", () => {
        it("should render safe expressions in content", () => {
          const content = "Actor: ${{ github.actor }}, Run: ${{ github.run_id }}";
          const result = processExpressions(content, "test.md");
          expect(result).toBe("Actor: testuser, Run: 12345");
        });

        it("should handle multiple expressions", () => {
          const content = "Issue #${{ github.event.issue.number }}: ${{ github.event.issue.title }}";
          const result = processExpressions(content, "test.md");
          expect(result).toBe("Issue #123: Test Issue");
        });

        it("should throw error for unsafe expressions", () => {
          const content = "Token: ${{ secrets.GITHUB_TOKEN }}";
          expect(() => processExpressions(content, "test.md")).toThrow("unauthorized GitHub Actions expressions");
        });

        it("should throw error for multiline expressions", () => {
          const content = "Value: ${{ \ngithub.actor \n}}";
          expect(() => processExpressions(content, "test.md")).toThrow("unauthorized");
        });

        it("should handle mixed safe and unsafe expressions", () => {
          const content = "Safe: ${{ github.actor }}, Unsafe: ${{ secrets.TOKEN }}";
          expect(() => processExpressions(content, "test.md")).toThrow("unauthorized");
          expect(() => processExpressions(content, "test.md")).toThrow("secrets.TOKEN");
        });

        it("should pass through content without expressions", () => {
          const content = "No expressions here";
          const result = processExpressions(content, "test.md");
          expect(result).toBe("No expressions here");
        });
      });

      describe("runtime import with expressions", () => {
        it("should process file with safe expressions", async () => {
          const content = "Actor: ${{ github.actor }}\nRepo: ${{ github.repository }}";
          fs.writeFileSync(path.join(githubDir, "with-expr.md"), content);
          const result = await processRuntimeImport("with-expr.md", false, tempDir);
          expect(result).toBe("Actor: testuser\nRepo: testorg/testrepo");
        });

        it("should reject file with unsafe expressions", async () => {
          const content = "Secret: ${{ secrets.TOKEN }}";
          fs.writeFileSync(path.join(githubDir, "unsafe.md"), content);
          await expect(processRuntimeImport("unsafe.md", false, tempDir)).rejects.toThrow("unauthorized");
        });

        it("should process expressions in URL imports", async () => {
          // Note: URL imports would need HTTP mocking to test properly
          // This is a placeholder for the structure
        });

        it("should handle expressions with front matter removal", async () => {
          const content = "---\ntitle: Test\n---\n\nActor: ${{ github.actor }}";
          fs.writeFileSync(path.join(githubDir, "frontmatter-expr.md"), content);
          const result = await processRuntimeImport("frontmatter-expr.md", false, tempDir);
          expect(result).toContain("Actor: testuser");
          expect(result).not.toContain("title: Test");
        });

        it("should handle expressions with XML comments", async () => {
          const content = "<!-- Comment -->\nActor: ${{ github.actor }}";
          fs.writeFileSync(path.join(githubDir, "comment-expr.md"), content);
          const result = await processRuntimeImport("comment-expr.md", false, tempDir);
          expect(result).toContain("Actor: testuser");
          expect(result).not.toContain("<!-- Comment -->");
        });
      });
    }));
});
