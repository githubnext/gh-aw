import { describe, it, expect } from "vitest";

describe("markdown_code_region_balancer.cjs", () => {
  let balancer;

  beforeEach(async () => {
    balancer = await import("./markdown_code_region_balancer.cjs");
  });

  describe("balanceCodeRegions", () => {
    describe("basic functionality", () => {
      it("should handle empty string", () => {
        expect(balancer.balanceCodeRegions("")).toBe("");
      });

      it("should handle null input", () => {
        expect(balancer.balanceCodeRegions(null)).toBe("");
      });

      it("should handle undefined input", () => {
        expect(balancer.balanceCodeRegions(undefined)).toBe("");
      });

      it("should not modify markdown without code blocks", () => {
        const input = `# Title
This is a paragraph.
## Section
More content.`;
        expect(balancer.balanceCodeRegions(input)).toBe(input);
      });

      it("should not modify properly balanced code blocks", () => {
        const input = `# Title

\`\`\`javascript
function test() {
  return true;
}
\`\`\`

End`;
        expect(balancer.balanceCodeRegions(input)).toBe(input);
      });
    });

    describe("nested code regions with same indentation", () => {
      it("should escape nested backtick fence inside code block", () => {
        const input = `\`\`\`javascript
function test() {
\`\`\`
nested
\`\`\`
}
\`\`\``;
        const expected = `\`\`\`javascript
function test() {
\\\`\`\`
nested
\\\`\`\`
}
\`\`\``;
        expect(balancer.balanceCodeRegions(input)).toBe(expected);
      });

      it("should escape nested tilde fence inside code block", () => {
        const input = `~~~markdown
Example:
~~~
nested
~~~
End
~~~`;
        const expected = `~~~markdown
Example:
\\~~~
nested
\\~~~
End
~~~`;
        expect(balancer.balanceCodeRegions(input)).toBe(expected);
      });

      it("should handle multiple nested fences", () => {
        const input = `\`\`\`javascript
function test() {
\`\`\`
first nested
\`\`\`
\`\`\`
second nested
\`\`\`
}
\`\`\``;
        const expected = `\`\`\`javascript
function test() {
\\\`\`\`
first nested
\\\`\`\`
\\\`\`\`
second nested
\\\`\`\`
}
\`\`\``;
        expect(balancer.balanceCodeRegions(input)).toBe(expected);
      });
    });

    describe("fence character types", () => {
      it("should not allow backticks to close tilde fence", () => {
        const input = `~~~markdown
Content
\`\`\`
Should be escaped
~~~`;
        const expected = `~~~markdown
Content
\\\`\`\`
Should be escaped
~~~`;
        expect(balancer.balanceCodeRegions(input)).toBe(expected);
      });

      it("should not allow tildes to close backtick fence", () => {
        const input = `\`\`\`markdown
Content
~~~
Should be escaped
\`\`\``;
        const expected = `\`\`\`markdown
Content
\\~~~
Should be escaped
\`\`\``;
        expect(balancer.balanceCodeRegions(input)).toBe(expected);
      });

      it("should handle alternating fence types", () => {
        const input = `\`\`\`javascript
code
\`\`\`

~~~markdown
content
~~~`;
        expect(balancer.balanceCodeRegions(input)).toBe(input);
      });
    });

    describe("fence lengths", () => {
      it("should require closing fence to be at least as long as opening", () => {
        const input = `\`\`\`\`\`
content
\`\`\`
should be escaped
\`\`\`\`\``;
        const expected = `\`\`\`\`\`
content
\\\`\`\`
should be escaped
\`\`\`\`\``;
        expect(balancer.balanceCodeRegions(input)).toBe(expected);
      });

      it("should allow longer closing fence", () => {
        const input = `\`\`\`
content
\`\`\`\`\`
end`;
        // This is valid - closing fence can be longer
        expect(balancer.balanceCodeRegions(input)).toBe(input);
      });

      it("should handle various fence lengths", () => {
        const input = `\`\`\`
three
\`\`\`

\`\`\`\`
four
\`\`\`\`

\`\`\`\`\`\`\`
seven
\`\`\`\`\`\`\``;
        expect(balancer.balanceCodeRegions(input)).toBe(input);
      });

      it("should escape shorter fence inside longer fence block", () => {
        const input = `\`\`\`\`\`\`
content
\`\`\`
nested short fence
\`\`\`
\`\`\`\`\`\``;
        const expected = `\`\`\`\`\`\`
content
\\\`\`\`
nested short fence
\\\`\`\`
\`\`\`\`\`\``;
        expect(balancer.balanceCodeRegions(input)).toBe(expected);
      });
    });

    describe("indentation", () => {
      it("should preserve indentation in code blocks", () => {
        const input = `  \`\`\`javascript
  function test() {
    return true;
  }
  \`\`\``;
        expect(balancer.balanceCodeRegions(input)).toBe(input);
      });

      it("should handle nested fence with different indentation", () => {
        const input = `\`\`\`markdown
Example:
  \`\`\`
  nested
  \`\`\`
\`\`\``;
        const expected = `\`\`\`markdown
Example:
  \\\`\`\`
  nested
  \\\`\`\`
\`\`\``;
        expect(balancer.balanceCodeRegions(input)).toBe(expected);
      });

      it("should preserve indentation when escaping", () => {
        const input = `\`\`\`markdown
    \`\`\`
    indented nested
    \`\`\`
\`\`\``;
        const expected = `\`\`\`markdown
    \\\`\`\`
    indented nested
    \\\`\`\`
\`\`\``;
        expect(balancer.balanceCodeRegions(input)).toBe(expected);
      });
    });

    describe("language specifiers", () => {
      it("should handle opening fence with language specifier", () => {
        const input = `\`\`\`javascript
code
\`\`\``;
        expect(balancer.balanceCodeRegions(input)).toBe(input);
      });

      it("should handle multiple language specifiers", () => {
        const input = `\`\`\`javascript
js code
\`\`\`

\`\`\`python
py code
\`\`\`

\`\`\`typescript
ts code
\`\`\``;
        expect(balancer.balanceCodeRegions(input)).toBe(input);
      });

      it("should handle language specifier with additional info", () => {
        const input = `\`\`\`javascript {1,3-5}
code
\`\`\``;
        expect(balancer.balanceCodeRegions(input)).toBe(input);
      });
    });

    describe("unclosed code blocks", () => {
      it("should close unclosed backtick code block", () => {
        const input = `\`\`\`javascript
function test() {
  return true;
}`;
        const expected = `\`\`\`javascript
function test() {
  return true;
}
\`\`\``;
        expect(balancer.balanceCodeRegions(input)).toBe(expected);
      });

      it("should close unclosed tilde code block", () => {
        const input = `~~~markdown
Content here
No closing fence`;
        const expected = `~~~markdown
Content here
No closing fence
~~~`;
        expect(balancer.balanceCodeRegions(input)).toBe(expected);
      });

      it("should close with matching fence length", () => {
        const input = `\`\`\`\`\`
five backticks
content`;
        const expected = `\`\`\`\`\`
five backticks
content
\`\`\`\`\``;
        expect(balancer.balanceCodeRegions(input)).toBe(expected);
      });

      it("should preserve indentation in closing fence", () => {
        const input = `  \`\`\`javascript
  code`;
        const expected = `  \`\`\`javascript
  code
  \`\`\``;
        expect(balancer.balanceCodeRegions(input)).toBe(expected);
      });
    });

    describe("complex real-world scenarios", () => {
      it("should handle AI-generated code with nested markdown", () => {
        const input = `# Example

Here's how to use code blocks:

\`\`\`markdown
You can create code blocks like this:
\`\`\`javascript
function hello() {
  console.log("world");
}
\`\`\`
\`\`\`

Text after`;
        const expected = `# Example

Here's how to use code blocks:

\`\`\`markdown
You can create code blocks like this:
\\\`\`\`javascript
function hello() {
  console.log("world");
}
\\\`\`\`
\`\`\`

Text after`;
        expect(balancer.balanceCodeRegions(input)).toBe(expected);
      });

      it("should handle documentation with multiple code examples", () => {
        const input = `## Usage

\`\`\`bash
npm install
\`\`\`

\`\`\`javascript
const x = 1;
\`\`\`

\`\`\`python
print("hello")
\`\`\``;
        expect(balancer.balanceCodeRegions(input)).toBe(input);
      });

      it("should handle mixed fence types in document", () => {
        const input = `\`\`\`javascript
const x = 1;
\`\`\`

~~~bash
echo "test"
~~~

\`\`\`
generic code
\`\`\``;
        expect(balancer.balanceCodeRegions(input)).toBe(input);
      });

      it("should handle deeply nested example", () => {
        const input = `\`\`\`markdown
# Tutorial

\`\`\`javascript
code here
\`\`\`

More text
\`\`\``;
        const expected = `\`\`\`markdown
# Tutorial

\\\`\`\`javascript
code here
\\\`\`\`

More text
\`\`\``;
        expect(balancer.balanceCodeRegions(input)).toBe(expected);
      });
    });

    describe("edge cases", () => {
      it("should handle Windows line endings", () => {
        const input = "\`\`\`javascript\r\ncode\r\n\`\`\`";
        const expected = "\`\`\`javascript\ncode\n\`\`\`";
        expect(balancer.balanceCodeRegions(input)).toBe(expected);
      });

      it("should handle mixed line endings", () => {
        const input = "\`\`\`\r\ncode\n\`\`\`\r\n";
        const expected = "\`\`\`\ncode\n\`\`\`\n";
        expect(balancer.balanceCodeRegions(input)).toBe(expected);
      });

      it("should handle empty code blocks", () => {
        const input = `\`\`\`
\`\`\``;
        expect(balancer.balanceCodeRegions(input)).toBe(input);
      });

      it("should handle single line with fence", () => {
        const input = "\`\`\`javascript";
        const expected = "\`\`\`javascript\n\`\`\`";
        expect(balancer.balanceCodeRegions(input)).toBe(expected);
      });

      it("should handle consecutive code blocks without blank lines", () => {
        const input = `\`\`\`javascript
code1
\`\`\`
\`\`\`python
code2
\`\`\``;
        expect(balancer.balanceCodeRegions(input)).toBe(input);
      });

      it("should not affect inline code", () => {
        const input = "Use `console.log()` to print";
        expect(balancer.balanceCodeRegions(input)).toBe(input);
      });

      it("should not affect multiple inline code", () => {
        const input = "Use `const x = 1` and `const y = 2` in code";
        expect(balancer.balanceCodeRegions(input)).toBe(input);
      });

      it("should handle very long fence", () => {
        const input = `\`\`\`\`\`\`\`\`\`\`\`\`\`\`\`\`
content
\`\`\`\`\`\`\`\`\`\`\`\`\`\`\`\``;
        expect(balancer.balanceCodeRegions(input)).toBe(input);
      });
    });

    describe("trailing content after fence", () => {
      it("should handle trailing content after opening fence", () => {
        const input = `\`\`\`javascript some extra text
code
\`\`\``;
        expect(balancer.balanceCodeRegions(input)).toBe(input);
      });

      it("should handle trailing content after closing fence", () => {
        const input = `\`\`\`javascript
code
\`\`\` trailing text`;
        expect(balancer.balanceCodeRegions(input)).toBe(input);
      });
    });
  });

  describe("isBalanced", () => {
    it("should return true for empty string", () => {
      expect(balancer.isBalanced("")).toBe(true);
    });

    it("should return true for null", () => {
      expect(balancer.isBalanced(null)).toBe(true);
    });

    it("should return true for undefined", () => {
      expect(balancer.isBalanced(undefined)).toBe(true);
    });

    it("should return true for markdown without code blocks", () => {
      const input = "# Title\nContent";
      expect(balancer.isBalanced(input)).toBe(true);
    });

    it("should return true for balanced code blocks", () => {
      const input = `\`\`\`javascript
code
\`\`\``;
      expect(balancer.isBalanced(input)).toBe(true);
    });

    it("should return false for unclosed code block", () => {
      const input = `\`\`\`javascript
code`;
      expect(balancer.isBalanced(input)).toBe(false);
    });

    it("should return false for nested unmatched fence", () => {
      const input = `\`\`\`javascript
\`\`\`
nested
\`\`\``;
      expect(balancer.isBalanced(input)).toBe(false);
    });

    it("should return true for multiple balanced blocks", () => {
      const input = `\`\`\`javascript
code1
\`\`\`

\`\`\`python
code2
\`\`\``;
      expect(balancer.isBalanced(input)).toBe(true);
    });
  });

  describe("countCodeRegions", () => {
    it("should return zero counts for empty string", () => {
      expect(balancer.countCodeRegions("")).toEqual({
        total: 0,
        balanced: 0,
        unbalanced: 0,
      });
    });

    it("should return zero counts for null", () => {
      expect(balancer.countCodeRegions(null)).toEqual({
        total: 0,
        balanced: 0,
        unbalanced: 0,
      });
    });

    it("should count single balanced block", () => {
      const input = `\`\`\`javascript
code
\`\`\``;
      expect(balancer.countCodeRegions(input)).toEqual({
        total: 1,
        balanced: 1,
        unbalanced: 0,
      });
    });

    it("should count unclosed block as unbalanced", () => {
      const input = `\`\`\`javascript
code`;
      expect(balancer.countCodeRegions(input)).toEqual({
        total: 1,
        balanced: 0,
        unbalanced: 1,
      });
    });

    it("should count multiple blocks correctly", () => {
      const input = `\`\`\`javascript
code1
\`\`\`

\`\`\`python
code2
\`\`\``;
      expect(balancer.countCodeRegions(input)).toEqual({
        total: 2,
        balanced: 2,
        unbalanced: 0,
      });
    });

    it("should count nested unmatched fences", () => {
      const input = `\`\`\`javascript
\`\`\`
nested
\`\`\``;
      // First ``` opens block, second ``` closes it, third ``` opens new block (unclosed)
      expect(balancer.countCodeRegions(input)).toEqual({
        total: 2,
        balanced: 1,
        unbalanced: 1,
      });
    });

    it("should count mixed fence types", () => {
      const input = `\`\`\`javascript
code
\`\`\`

~~~markdown
content
~~~`;
      expect(balancer.countCodeRegions(input)).toEqual({
        total: 2,
        balanced: 2,
        unbalanced: 0,
      });
    });
  });

  describe("fuzz testing", () => {
    it("should handle random combinations of fences", () => {
      // Generate various random but structured inputs
      const testCases = [
        "```\n```\n```\n```",
        "~~~\n~~~\n~~~",
        "```js\n~~~\n```\n~~~",
        "````\n```\n````",
        "```\n````\n```",
        "  ```\n```\n  ```",
        "```\n  ```\n```",
        "```\n\n```\n\n```\n\n```",
      ];

      testCases.forEach((input) => {
        // Should not throw an error
        expect(() => balancer.balanceCodeRegions(input)).not.toThrow();
        // Result should be a string
        expect(typeof balancer.balanceCodeRegions(input)).toBe("string");
      });
    });

    it("should handle long documents with many code blocks", () => {
      let input = "# Document\n\n";
      for (let i = 0; i < 50; i++) {
        input += `\`\`\`javascript\ncode${i}\n\`\`\`\n\n`;
      }
      const result = balancer.balanceCodeRegions(input);
      expect(result).toContain("code0");
      expect(result).toContain("code49");
      expect(balancer.isBalanced(result)).toBe(true);
    });

    it("should handle deeply nested structures", () => {
      let input = "```markdown\n";
      for (let i = 0; i < 10; i++) {
        input += "```\nnested " + i + "\n```\n";
      }
      input += "```";

      // Should not throw and should produce some output
      expect(() => balancer.balanceCodeRegions(input)).not.toThrow();
      const result = balancer.balanceCodeRegions(input);
      expect(result.length).toBeGreaterThan(0);
    });

    it("should handle very long lines", () => {
      const longLine = "a".repeat(10000);
      const input = `\`\`\`\n${longLine}\n\`\`\``;
      const result = balancer.balanceCodeRegions(input);
      expect(result).toContain(longLine);
    });

    it("should handle special characters in code blocks", () => {
      const input = `\`\`\`
<>&"'\n\t\r
\`\`\``;
      const result = balancer.balanceCodeRegions(input);
      expect(result).toContain("<>&\"'");
    });

    it("should handle unicode characters", () => {
      const input = `\`\`\`javascript
const emoji = "🚀";
const chinese = "你好";
const arabic = "مرحبا";
\`\`\``;
      expect(balancer.balanceCodeRegions(input)).toBe(input);
    });

    it("should handle empty lines in various positions", () => {
      const input = `

\`\`\`


code


\`\`\`

`;
      expect(balancer.balanceCodeRegions(input)).toBe(input);
    });
  });
});
