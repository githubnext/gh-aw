import { describe, it, expect, vi } from "vitest";
import fs from "fs";
import path from "path";
import { fileURLToPath } from "url";
const __filename = fileURLToPath(import.meta.url),
  __dirname = path.dirname(__filename),
  core = { info: vi.fn(), warning: vi.fn(), setFailed: vi.fn(), summary: { addHeading: vi.fn().mockReturnThis(), addRaw: vi.fn().mockReturnThis(), write: vi.fn() } };
global.core = core;
const renderTemplateScript = fs.readFileSync(path.join(__dirname, "render_template.cjs"), "utf8"),
  isTruthyMatch = renderTemplateScript.match(/function isTruthy\(expr\)\s*{[\s\S]*?return[\s\S]*?;[\s\S]*?}/),
  renderMarkdownTemplateMatch = renderTemplateScript.match(/function renderMarkdownTemplate\(markdown\)\s*{[\s\S]*?return result;[\s\S]*?}/);
if (!isTruthyMatch || !renderMarkdownTemplateMatch) throw new Error("Could not extract functions from render_template.cjs");
const isTruthy = eval(`(${isTruthyMatch[0]})`),
  renderMarkdownTemplate = eval(`(${renderMarkdownTemplateMatch[0]})`);
(describe("isTruthy", () => {
  (it("should return false for empty string", () => {
    expect(isTruthy("")).toBe(!1);
  }),
    it('should return false for "false"', () => {
      (expect(isTruthy("false")).toBe(!1), expect(isTruthy("FALSE")).toBe(!1), expect(isTruthy("False")).toBe(!1));
    }),
    it('should return false for "0"', () => {
      expect(isTruthy("0")).toBe(!1);
    }),
    it('should return false for "null"', () => {
      (expect(isTruthy("null")).toBe(!1), expect(isTruthy("NULL")).toBe(!1));
    }),
    it('should return false for "undefined"', () => {
      (expect(isTruthy("undefined")).toBe(!1), expect(isTruthy("UNDEFINED")).toBe(!1));
    }),
    it('should return true for "true"', () => {
      (expect(isTruthy("true")).toBe(!0), expect(isTruthy("TRUE")).toBe(!0));
    }),
    it("should return true for any non-falsy string", () => {
      (expect(isTruthy("yes")).toBe(!0), expect(isTruthy("1")).toBe(!0), expect(isTruthy("hello")).toBe(!0));
    }),
    it("should trim whitespace", () => {
      (expect(isTruthy("  false  ")).toBe(!1), expect(isTruthy("  true  ")).toBe(!0));
    }));
}),
  describe("renderMarkdownTemplate", () => {
    (it("should keep content in truthy blocks", () => {
      const output = renderMarkdownTemplate("{{#if true}}\nHello\n{{/if}}");
      expect(output).toBe("Hello\n");
    }),
      it("should remove content in falsy blocks", () => {
        const output = renderMarkdownTemplate("{{#if false}}\nHello\n{{/if}}");
        expect(output).toBe("");
      }),
      it("should process multiple blocks", () => {
        const output = renderMarkdownTemplate("{{#if true}}\nKeep this\n{{/if}}\n{{#if false}}\nRemove this\n{{/if}}");
        expect(output).toBe("Keep this\n");
      }),
      it("should handle nested content", () => {
        const output = renderMarkdownTemplate("# Title\n\n{{#if true}}\n## Section 1\nThis should be kept.\n{{/if}}\n\n{{#if false}}\n## Section 2\nThis should be removed.\n{{/if}}\n\n## Section 3\nThis is always visible.");
        expect(output).toBe("# Title\n\n## Section 1\nThis should be kept.\n\n## Section 3\nThis is always visible.");
      }),
      it("should leave content without conditionals unchanged", () => {
        const input = "# Normal Markdown\n\nNo conditionals here.",
          output = renderMarkdownTemplate(input);
        expect(output).toBe(input);
      }),
      it("should handle conditionals with various expressions", () => {
        (expect(renderMarkdownTemplate("{{#if 1}}\nKeep\n{{/if}}")).toBe("Keep\n"),
          expect(renderMarkdownTemplate("{{#if 0}}\nRemove\n{{/if}}")).toBe(""),
          expect(renderMarkdownTemplate("{{#if null}}\nRemove\n{{/if}}")).toBe(""),
          expect(renderMarkdownTemplate("{{#if undefined}}\nRemove\n{{/if}}")).toBe(""));
      }),
      it("should preserve markdown formatting inside blocks", () => {
        const output = renderMarkdownTemplate("{{#if true}}\n## Header\n- List item 1\n- List item 2\n\n```javascript\nconst x = 1;\n```\n{{/if}}");
        expect(output).toBe("## Header\n- List item 1\n- List item 2\n\n```javascript\nconst x = 1;\n```\n");
      }),
      it("should handle whitespace in conditionals", () => {
        (expect(renderMarkdownTemplate("{{#if   true  }}\nKeep\n{{/if}}")).toBe("Keep\n"), expect(renderMarkdownTemplate("{{#if\ttrue\t}}\nKeep\n{{/if}}")).toBe("Keep\n"));
      }),
      it("should clean up multiple consecutive empty lines", () => {
        const output = renderMarkdownTemplate("# Title\n\n{{#if false}}\n## Hidden Section\nThis should be removed.\n{{/if}}\n\n## Visible Section\nThis is always visible.");
        expect(output).toBe("# Title\n\n## Visible Section\nThis is always visible.");
      }),
      it("should collapse multiple false blocks without excessive empty lines", () => {
        const output = renderMarkdownTemplate("Start\n\n{{#if false}}\nBlock 1\n{{/if}}\n\n{{#if false}}\nBlock 2\n{{/if}}\n\n{{#if false}}\nBlock 3\n{{/if}}\n\nEnd");
        (expect(output).not.toMatch(/\n{3,}/), expect(output).toContain("Start"), expect(output).toContain("End"));
      }),
      it("should preserve leading spaces with truthy block", () => {
        const output = renderMarkdownTemplate("  {{#if true}}\n  Content with leading spaces\n  {{/if}}");
        expect(output).toBe("  Content with leading spaces\n");
      }),
      it("should remove leading spaces when block is falsy", () => {
        const output = renderMarkdownTemplate("  {{#if false}}\n  Content that should be removed\n  {{/if}}");
        expect(output).toBe("");
      }),
      it("should handle mixed indentation levels", () => {
        const output = renderMarkdownTemplate("{{#if true}}\nNo indent\n{{/if}}\n  {{#if true}}\n  Two space indent\n  {{/if}}\n    {{#if true}}\n    Four space indent\n    {{/if}}");
        expect(output).toBe("No indent\n  Two space indent\n    Four space indent\n");
      }),
      it("should preserve indentation in content when using leading spaces", () => {
        const output = renderMarkdownTemplate("# Header\n\n  {{#if true}}\n  ## Indented subsection\n  This content has two leading spaces\n  {{/if}}\n\nNormal content");
        expect(output).toBe("# Header\n\n  ## Indented subsection\n  This content has two leading spaces\n\nNormal content");
      }),
      it("should handle tabs as leading characters", () => {
        const output = renderMarkdownTemplate("\t{{#if true}}\n\tContent with tab\n\t{{/if}}");
        expect(output).toBe("\tContent with tab\n");
      }),
      it("should handle realistic linter-formatted markdown", () => {
        const inputWithValue = "# Analysis\n\n  {{#if github.event.issue.number}}\n  ## Issue Analysis\n  \n  Analyzing issue #123\n  \n  - Check description\n  - Review labels\n  {{/if}}\n\nContinue with other tasks".replace(
            "github.event.issue.number",
            "123"
          ),
          output = renderMarkdownTemplate(inputWithValue);
        expect(output).toBe("# Analysis\n\n  ## Issue Analysis\n  \n  Analyzing issue #123\n  \n  - Check description\n  - Review labels\n\nContinue with other tasks");
      }),
      it("should preserve closing tag indentation", () => {
        const output = renderMarkdownTemplate("  {{#if true}}\n  Content\n  {{/if}}\nNext line");
        expect(output).toBe("  Content\nNext line");
      }));
  }));
