// Additional comprehensive tests for renderMarkdownTemplate function
import { describe, it, expect } from "vitest";
import fs from "fs";
import path from "path";
import { fileURLToPath } from "url";

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

// Import isTruthy from its own module
const { isTruthy } = require("./is_truthy.cjs");

// Import the interpolation script and extract renderMarkdownTemplate
const interpolatePromptScript = fs.readFileSync(path.join(__dirname, "interpolate_prompt.cjs"), "utf8");

const renderMarkdownTemplateMatch = interpolatePromptScript.match(
  /function renderMarkdownTemplate\(markdown\)\s*{[\s\S]*?return result;[\s\S]*?}/
);

if (!renderMarkdownTemplateMatch) {
  throw new Error("Could not extract renderMarkdownTemplate function from interpolate_prompt.cjs");
}

// eslint-disable-next-line no-eval
const renderMarkdownTemplate = eval(`(${renderMarkdownTemplateMatch[0]})`);

describe("renderMarkdownTemplate - Additional Edge Cases", () => {
  describe("inline conditionals (tags not on their own lines)", () => {
    it("should handle inline conditional at start of line", () => {
      const input = "{{#if true}}Keep{{/if}}";
      const output = renderMarkdownTemplate(input);
      expect(output).toBe("Keep");
    });

    it("should handle inline conditional in middle of line", () => {
      const input = "Before {{#if true}}Middle{{/if}} After";
      const output = renderMarkdownTemplate(input);
      expect(output).toBe("Before Middle After");
    });

    it("should handle inline conditional with false condition", () => {
      const input = "Before {{#if false}}Remove{{/if}} After";
      const output = renderMarkdownTemplate(input);
      expect(output).toBe("Before  After");
    });

    it("should handle multiple inline conditionals on same line", () => {
      const input = "{{#if true}}A{{/if}} {{#if false}}B{{/if}} {{#if true}}C{{/if}}";
      const output = renderMarkdownTemplate(input);
      expect(output).toBe("A  C");
    });
  });

  describe("block conditionals (tags on their own lines)", () => {
    it("should handle block conditional with tags on own lines", () => {
      const input = `{{#if true}}
Content
{{/if}}`;
      const output = renderMarkdownTemplate(input);
      expect(output).toBe("Content\n");
    });

    it("should remove block conditional with false condition", () => {
      const input = `{{#if false}}
Content
{{/if}}`;
      const output = renderMarkdownTemplate(input);
      expect(output).toBe("");
    });

    it("should handle block conditional with indentation", () => {
      const input = `  {{#if true}}
  Content
  {{/if}}`;
      const output = renderMarkdownTemplate(input);
      expect(output).toBe("  Content\n");
    });

    it("should handle block conditional with tabs", () => {
      const input = `\t{{#if true}}
\tContent
\t{{/if}}`;
      const output = renderMarkdownTemplate(input);
      expect(output).toBe("\tContent\n");
    });
  });

  describe("whitespace handling", () => {
    it("should handle spaces before opening tag", () => {
      const input = "   {{#if true}}Content{{/if}}";
      const output = renderMarkdownTemplate(input);
      expect(output).toBe("   Content");
    });

    it("should handle spaces after closing tag", () => {
      const input = "{{#if true}}Content{{/if}}   ";
      const output = renderMarkdownTemplate(input);
      expect(output).toBe("Content   ");
    });

    it("should handle trailing spaces on tag lines", () => {
      const input = `{{#if true}}   
Content
{{/if}}   `;
      const output = renderMarkdownTemplate(input);
      // The first pass should match this and preserve content
      expect(output.trim()).toBe("Content");
    });

    it("should handle no newline after opening tag (inline)", () => {
      const input = "{{#if true}}Content on same line{{/if}}";
      const output = renderMarkdownTemplate(input);
      expect(output).toBe("Content on same line");
    });

    it("should handle no newline before closing tag (inline)", () => {
      const input = `{{#if true}}
Content{{/if}}`;
      const output = renderMarkdownTemplate(input);
      // The first-pass regex matches this because opening tag has newline after it
      // It preserves the body without the leading newline (no leadNL before the tag)
      expect(output).toBe("Content");
    });
  });

  describe("nested and complex structures", () => {
    it("should handle conditional with markdown formatting", () => {
      const input = `{{#if true}}
## Header

**Bold** and *italic* text

- List item 1
- List item 2
{{/if}}`;
      const expected = `## Header

**Bold** and *italic* text

- List item 1
- List item 2
`;
      const output = renderMarkdownTemplate(input);
      expect(output).toBe(expected);
    });

    it("should handle conditional with code blocks", () => {
      const input = `{{#if true}}
\`\`\`javascript
const x = 1;
\`\`\`
{{/if}}`;
      const expected = `\`\`\`javascript
const x = 1;
\`\`\`
`;
      const output = renderMarkdownTemplate(input);
      expect(output).toBe(expected);
    });

    it("should NOT handle nested conditionals (not supported)", () => {
      // This test documents current behavior - nested conditionals are not supported
      const input = `{{#if true}}
Outer
{{#if true}}
Inner
{{/if}}
{{/if}}`;
      // The regex will match the first {{#if}} with the first {{/if}}, leaving the rest
      const output = renderMarkdownTemplate(input);
      // Exact behavior depends on regex matching, but it won't handle nesting correctly
      expect(output).toContain("Outer");
    });
  });

  describe("edge cases with empty lines", () => {
    it("should clean up multiple consecutive blank lines", () => {
      const input = `Start


{{#if false}}
Content
{{/if}}


End`;
      const output = renderMarkdownTemplate(input);
      // Should not have more than 2 consecutive newlines
      expect(output).not.toMatch(/\n{3,}/);
      expect(output).toContain("Start");
      expect(output).toContain("End");
    });

    it("should preserve single blank line", () => {
      const input = `Line 1

Line 2`;
      const output = renderMarkdownTemplate(input);
      expect(output).toBe(input);
    });

    it("should clean up triple newlines in input", () => {
      const input = `Line 1


Line 2`;
      const output = renderMarkdownTemplate(input);
      // The cleanup phase converts 3+ newlines to 2 newlines (one blank line)
      // This is actually desirable behavior - triple newlines become double
      expect(output).toBe(`Line 1

Line 2`);
    });

    it("should collapse triple blank line to double", () => {
      const input = `Line 1



Line 2`;
      const expected = `Line 1

Line 2`;
      const output = renderMarkdownTemplate(input);
      expect(output).toBe(expected);
    });
  });

  describe("mixed scenarios", () => {
    it("should handle mix of inline and block conditionals", () => {
      const input = `Start {{#if true}}inline{{/if}} text
{{#if true}}
Block content
{{/if}}
End`;
      const expected = `Start inline text
Block content
End`;
      const output = renderMarkdownTemplate(input);
      expect(output).toBe(expected);
    });

    it("should handle mix of true and false conditionals", () => {
      const input = `{{#if true}}
Keep this
{{/if}}
{{#if false}}
Remove this
{{/if}}
{{#if true}}
Keep this too
{{/if}}`;
      const output = renderMarkdownTemplate(input);
      expect(output).toContain("Keep this");
      expect(output).toContain("Keep this too");
      expect(output).not.toContain("Remove this");
    });

    it("should handle complex real-world example", () => {
      const input = `# Workflow Prompt

Some intro text

{{#if github.event.issue.number}}
## Issue Information

Issue #${"{github.event.issue.number}"}
Title: ${"{github.event.issue.title}"}
{{/if}}

{{#if github.event.pull_request.number}}
## Pull Request Information

PR #${"{github.event.pull_request.number}"}
{{/if}}

## Instructions

Always visible instructions here.`;

      // With both conditions true
      const result = renderMarkdownTemplate(input.replace(/github\.event\.\w+\.\w+/g, "true"));
      expect(result).toContain("## Issue Information");
      expect(result).toContain("## Pull Request Information");
      expect(result).toContain("## Instructions");
    });
  });

  describe("boundary conditions", () => {
    it("should handle empty string", () => {
      const output = renderMarkdownTemplate("");
      expect(output).toBe("");
    });

    it("should handle string with only whitespace", () => {
      const input = "   \n\n   ";
      const output = renderMarkdownTemplate(input);
      expect(output).toBe(input);
    });

    it("should handle unclosed conditional (malformed)", () => {
      // This documents behavior with malformed input
      const input = "{{#if true}} Content";
      const output = renderMarkdownTemplate(input);
      // Without closing tag, nothing should be replaced
      expect(output).toBe(input);
    });

    it("should handle closing tag without opening (malformed)", () => {
      const input = "Content {{/if}}";
      const output = renderMarkdownTemplate(input);
      // Without opening tag, nothing should be replaced
      expect(output).toBe(input);
    });

    it("should handle empty condition expression (not matched)", () => {
      const input = "{{#if }}Content{{/if}}";
      const output = renderMarkdownTemplate(input);
      // The regex requires at least one character in condition: [^}]+
      // So {{#if }} doesn't match and is left unchanged
      expect(output).toBe(input);
    });

    it("should handle condition with only whitespace", () => {
      const input = "{{#if   }}Content{{/if}}";
      const output = renderMarkdownTemplate(input);
      // Whitespace that trims to empty is falsy
      expect(output).toBe("");
    });
  });

  describe("special characters in content", () => {
    it("should handle content with curly braces", () => {
      const input = "{{#if true}}{} and {{}}{{/if}}";
      const output = renderMarkdownTemplate(input);
      expect(output).toBe("{} and {{}}");
    });

    it("should handle content with template-like strings", () => {
      const input = "{{#if true}}This looks like {{template}} but isn't{{/if}}";
      const output = renderMarkdownTemplate(input);
      expect(output).toBe("This looks like {{template}} but isn't");
    });

    it("should handle content with dollar signs", () => {
      const input = "{{#if true}}Price: $100{{/if}}";
      const output = renderMarkdownTemplate(input);
      expect(output).toBe("Price: $100");
    });

    it("should handle content with backslashes", () => {
      const input = "{{#if true}}Path: C:\\\\Users\\\\test{{/if}}";
      const output = renderMarkdownTemplate(input);
      expect(output).toBe("Path: C:\\\\Users\\\\test");
    });

    it("should handle content with newlines and special chars", () => {
      const input = `{{#if true}}
Line 1 with \$pecial ch@rs!
Line 2 with {{braces}}
{{/if}}`;
      const output = renderMarkdownTemplate(input);
      expect(output).toContain("Line 1 with $pecial ch@rs!");
      expect(output).toContain("Line 2 with {{braces}}");
    });
  });

  describe("performance and edge cases", () => {
    it("should handle large number of conditionals", () => {
      let input = "";
      for (let i = 0; i < 100; i++) {
        input += `{{#if true}}Block ${i}{{/if}}\n`;
      }
      const output = renderMarkdownTemplate(input);
      expect(output).toContain("Block 0");
      expect(output).toContain("Block 99");
    });

    it("should handle long content blocks", () => {
      const longContent = "x".repeat(10000);
      const input = `{{#if true}}\n${longContent}\n{{/if}}`;
      const output = renderMarkdownTemplate(input);
      expect(output).toContain(longContent);
    });
  });

  describe("leading newline preservation", () => {
    it("should preserve leading newline when condition is true", () => {
      const input = `Line before

{{#if true}}
Content
{{/if}}`;
      const output = renderMarkdownTemplate(input);
      expect(output).toBe(`Line before

Content
`);
    });

    it("should remove leading newline when condition is false", () => {
      const input = `Line before

{{#if false}}
Content
{{/if}}
Line after`;
      const output = renderMarkdownTemplate(input);
      // The leading newline before {{#if false}} should be preserved initially,
      // but after cleanup, excessive blank lines are reduced
      expect(output).toContain("Line before");
      expect(output).toContain("Line after");
      expect(output).not.toMatch(/\n{3,}/);
    });

    it("should handle no leading newline with true condition", () => {
      const input = `{{#if true}}
Content
{{/if}}`;
      const output = renderMarkdownTemplate(input);
      expect(output).toBe("Content\n");
    });

    it("should handle no leading newline with false condition", () => {
      const input = `{{#if false}}
Content
{{/if}}Line after`;
      const output = renderMarkdownTemplate(input);
      expect(output).toBe("Line after");
    });
  });
});
