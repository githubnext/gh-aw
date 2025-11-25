// Tests for render_template.cjs
import { describe, it, expect, vi } from "vitest";
import fs from "fs";
import path from "path";
import { fileURLToPath } from "url";

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

// Mock the core module
const core = {
  info: vi.fn(),
  warning: vi.fn(),
  setFailed: vi.fn(),
  summary: {
    addHeading: vi.fn().mockReturnThis(),
    addRaw: vi.fn().mockReturnThis(),
    write: vi.fn(),
  },
};
global.core = core;

// Import the template renderer functions
const renderTemplateScript = fs.readFileSync(path.join(__dirname, "render_template.cjs"), "utf8");

// Extract the functions from the script
const isTruthyMatch = renderTemplateScript.match(/function isTruthy\(expr\)\s*{[\s\S]*?return[\s\S]*?;[\s\S]*?}/);
const renderMarkdownTemplateMatch = renderTemplateScript.match(
  /function renderMarkdownTemplate\(markdown\)\s*{[\s\S]*?return result;[\s\S]*?}/
);

if (!isTruthyMatch || !renderMarkdownTemplateMatch) {
  throw new Error("Could not extract functions from render_template.cjs");
}

// eslint-disable-next-line no-eval
const isTruthy = eval(`(${isTruthyMatch[0]})`);
// eslint-disable-next-line no-eval
const renderMarkdownTemplate = eval(`(${renderMarkdownTemplateMatch[0]})`);

describe("isTruthy", () => {
  it("should return false for empty string", () => {
    expect(isTruthy("")).toBe(false);
  });

  it('should return false for "false"', () => {
    expect(isTruthy("false")).toBe(false);
    expect(isTruthy("FALSE")).toBe(false);
    expect(isTruthy("False")).toBe(false);
  });

  it('should return false for "0"', () => {
    expect(isTruthy("0")).toBe(false);
  });

  it('should return false for "null"', () => {
    expect(isTruthy("null")).toBe(false);
    expect(isTruthy("NULL")).toBe(false);
  });

  it('should return false for "undefined"', () => {
    expect(isTruthy("undefined")).toBe(false);
    expect(isTruthy("UNDEFINED")).toBe(false);
  });

  it('should return true for "true"', () => {
    expect(isTruthy("true")).toBe(true);
    expect(isTruthy("TRUE")).toBe(true);
  });

  it("should return true for any non-falsy string", () => {
    expect(isTruthy("yes")).toBe(true);
    expect(isTruthy("1")).toBe(true);
    expect(isTruthy("hello")).toBe(true);
  });

  it("should trim whitespace", () => {
    expect(isTruthy("  false  ")).toBe(false);
    expect(isTruthy("  true  ")).toBe(true);
  });
});

describe("renderMarkdownTemplate", () => {
  it("should keep content in truthy blocks", () => {
    const input = "{{#if true}}\nHello\n{{/if}}";
    const output = renderMarkdownTemplate(input);
    expect(output).toBe("Hello\n");
  });

  it("should remove content in falsy blocks", () => {
    const input = "{{#if false}}\nHello\n{{/if}}";
    const output = renderMarkdownTemplate(input);
    expect(output).toBe("");
  });

  it("should process multiple blocks", () => {
    const input = "{{#if true}}\nKeep this\n{{/if}}\n{{#if false}}\nRemove this\n{{/if}}";
    const output = renderMarkdownTemplate(input);
    expect(output).toBe("Keep this\n");
  });

  it("should handle nested content", () => {
    const input = `# Title

{{#if true}}
## Section 1
This should be kept.
{{/if}}

{{#if false}}
## Section 2
This should be removed.
{{/if}}

## Section 3
This is always visible.`;

    // With empty line cleanup, we expect at most 2 consecutive newlines
    const expected = `# Title

## Section 1
This should be kept.

## Section 3
This is always visible.`;

    const output = renderMarkdownTemplate(input);
    expect(output).toBe(expected);
  });

  it("should leave content without conditionals unchanged", () => {
    const input = "# Normal Markdown\n\nNo conditionals here.";
    const output = renderMarkdownTemplate(input);
    expect(output).toBe(input);
  });

  it("should handle conditionals with various expressions", () => {
    const input1 = "{{#if 1}}\nKeep\n{{/if}}";
    expect(renderMarkdownTemplate(input1)).toBe("Keep\n");

    const input2 = "{{#if 0}}\nRemove\n{{/if}}";
    expect(renderMarkdownTemplate(input2)).toBe("");

    const input3 = "{{#if null}}\nRemove\n{{/if}}";
    expect(renderMarkdownTemplate(input3)).toBe("");

    const input4 = "{{#if undefined}}\nRemove\n{{/if}}";
    expect(renderMarkdownTemplate(input4)).toBe("");
  });

  it("should preserve markdown formatting inside blocks", () => {
    const input = `{{#if true}}
## Header
- List item 1
- List item 2

\`\`\`javascript
const x = 1;
\`\`\`
{{/if}}`;

    const expected = `## Header
- List item 1
- List item 2

\`\`\`javascript
const x = 1;
\`\`\`
`;

    const output = renderMarkdownTemplate(input);
    expect(output).toBe(expected);
  });

  it("should handle whitespace in conditionals", () => {
    const input1 = "{{#if   true  }}\nKeep\n{{/if}}";
    expect(renderMarkdownTemplate(input1)).toBe("Keep\n");

    const input2 = "{{#if\ttrue\t}}\nKeep\n{{/if}}";
    expect(renderMarkdownTemplate(input2)).toBe("Keep\n");
  });

  it("should clean up multiple consecutive empty lines", () => {
    // When a false block is removed, it should not leave more than 2 consecutive newlines
    const input = `# Title

{{#if false}}
## Hidden Section
This should be removed.
{{/if}}

## Visible Section
This is always visible.`;

    const expected = `# Title

## Visible Section
This is always visible.`;

    const output = renderMarkdownTemplate(input);
    expect(output).toBe(expected);
  });

  it("should collapse multiple false blocks without excessive empty lines", () => {
    const input = `Start

{{#if false}}
Block 1
{{/if}}

{{#if false}}
Block 2
{{/if}}

{{#if false}}
Block 3
{{/if}}

End`;

    const output = renderMarkdownTemplate(input);
    // Should not have more than 2 consecutive newlines anywhere
    expect(output).not.toMatch(/\n{3,}/);
    expect(output).toContain("Start");
    expect(output).toContain("End");
  });
});
