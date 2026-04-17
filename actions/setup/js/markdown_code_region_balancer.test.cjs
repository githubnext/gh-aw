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
      it("should use greedy matching for bare fences without intermediate language openers", () => {
        // Without intermediate language-tagged openers, bare fences form sequential
        // code blocks (greedy matching per CommonMark), not nested content.
        const input = `\`\`\`javascript
function test() {
\`\`\`
nested
\`\`\`
}
\`\`\``;
        // Already balanced: two sequential code blocks
        expect(balancer.balanceCodeRegions(input)).toBe(input);
        expect(balancer.isBalanced(balancer.balanceCodeRegions(input))).toBe(true);
      });

      it("should use greedy matching for bare tilde fences without intermediate language openers", () => {
        // Without intermediate language-tagged openers, bare fences form sequential
        // code blocks (greedy matching per CommonMark), not nested content.
        const input = `~~~markdown
Example:
~~~
nested
~~~
End
~~~`;
        // Already balanced: two sequential code blocks
        expect(balancer.balanceCodeRegions(input)).toBe(input);
        expect(balancer.isBalanced(balancer.balanceCodeRegions(input))).toBe(true);
      });

      it("should use greedy matching for multiple bare fences without intermediate language openers", () => {
        // Multiple bare fences without intermediate language-tagged openers
        // are sequential code blocks, not nested content.
        // With 5 fences (odd), greedy matching pairs 0-1, 2-3, and fence 4 is unclosed.
        // The balancer adds a closing fence for the unclosed block.
        const input = `\`\`\`javascript
function test() {
\`\`\`
first nested
\`\`\`
second nested
\`\`\`
}
\`\`\``;
        const expected = `\`\`\`javascript
function test() {
\`\`\`
first nested
\`\`\`
second nested
\`\`\`
}
\`\`\`
\`\`\``;
        // Greedy matching: 2 paired blocks + 1 unclosed (auto-closed)
        const result = balancer.balanceCodeRegions(input);
        expect(result).toBe(expected);
        expect(balancer.isBalanced(result)).toBe(true);
      });

      it("should not make things worse when intermediate language-tagged openers exist", () => {
        // This pattern is inherently ambiguous under CommonMark greedy matching:
        // ```markdown pairs with the first bare ```, leaving the final ``` unclosed.
        // The balancer cannot resolve this, but must not make it worse.
        const input = `\`\`\`markdown
Here's an example:
\`\`\`python
print("hello")
\`\`\`
End
\`\`\``;
        const result = balancer.balanceCodeRegions(input);
        const inputCounts = balancer.countCodeRegions(input);
        const resultCounts = balancer.countCodeRegions(result);
        expect(resultCounts.unbalanced).toBeLessThanOrEqual(inputCounts.unbalanced);
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
\`\`\`
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
~~~
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
      it("should treat shorter bare fence inside longer fence block as content", () => {
        // A 5-backtick opener cannot be closed by a 3-backtick fence (CommonMark rule),
        // so the 3-backtick fences are content inside the block, not separate blocks.
        const input = `\`\`\`\`\`
content
\`\`\`
should be escaped
\`\`\`\`\``;
        expect(balancer.balanceCodeRegions(input)).toBe(input);
      });

      it("should allow longer closing fence", () => {
        const input = `\`\`\`
content
\`\`\`\`\`
end`;
        // This is valid - closing fence can be longer
        expect(balancer.balanceCodeRegions(input)).toBe(input);
      });

      it("should handle various fence lengths as separate blocks", () => {
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

      it("should treat shorter fences inside longer fence block as content", () => {
        // A 6-backtick opener cannot be closed by 3-backtick fences,
        // so they're content, not closers.
        const input = `\`\`\`\`\`\`
content
\`\`\`
nested short fence
\`\`\`
\`\`\`\`\`\``;
        expect(balancer.balanceCodeRegions(input)).toBe(input);
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
        // Indented fences inside a markdown block are treated as content (examples), not active fences
        // No escaping needed
        expect(balancer.balanceCodeRegions(input)).toBe(input);
      });

      it("should preserve indentation when escaping", () => {
        const input = `\`\`\`markdown
    \`\`\`
    indented nested
    \`\`\`
\`\`\``;
        // Indented fences inside a markdown block are treated as content (examples), not active fences
        // No escaping needed
        expect(balancer.balanceCodeRegions(input)).toBe(input);
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
        // The markdown and javascript fences are sequential blocks with greedy matching.
        // markdown opens, first bare ``` closes it, javascript opens, second bare ``` closes it.
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
        expect(balancer.balanceCodeRegions(input)).toBe(input);
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
        // Greedy matching: markdown opens, first bare ``` closes it,
        // javascript opens, second bare ``` closes it. Two sequential blocks.
        const input = `\`\`\`markdown
# Tutorial

\`\`\`javascript
code here
\`\`\`

More text
\`\`\``;
        expect(balancer.balanceCodeRegions(input)).toBe(input);
      });

      it("should not modify markdown block containing indented bare fences as examples (issue #11081)", () => {
        // This reproduces the issue from GitHub issue #11081
        // A markdown code block containing examples of code blocks with indentation
        const input = `**Add to AGENTS.md:**

\`\`\`markdown
## Safe Outputs Schema Synchronization

**CRITICAL: When modifying safe output templates or handlers:**

1. **Update all related files:**
   - Source: \`actions/setup/js/handle_*.cjs\`
   - Schema: \`pkg/workflow/js/safe_outputs_tools.json\`

2. **Schema sync checklist:**
   \`\`\`
   # After modifying any handle_*.cjs file:
   cd actions/setup/js
   npm test  # MUST pass
   \`\`\`

3. **Common pitfalls:**
   - ❌ Changing issue titles without updating schema
   
4. **Pattern to follow:**
   \`\`\`
   # Find all related definitions
   grep -r "your-new-text" actions/setup/js/
   \`\`\`
\`\`\`

## Historical Context`;
        // No changes expected - the indented bare ``` inside the markdown block are examples
        expect(balancer.balanceCodeRegions(input)).toBe(input);
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

      it("should close unmatched opening fence when shorter fence cannot close it", () => {
        // Regression test for GitHub Issue #11630
        // When a 4-backtick fence is opened but only a 3-backtick fence follows,
        // the 3-backtick fence should be treated as content inside the code block,
        // not as a separate unclosed fence.
        const input = `#### NPM Versions Available

\`\`\`\`
0.0.56
0.0.57
0.0.58
\`\`\``;
        const expected = `#### NPM Versions Available

\`\`\`\`
0.0.56
0.0.57
0.0.58
\`\`\`
\`\`\`\``;
        expect(balancer.balanceCodeRegions(input)).toBe(expected);
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
      const testCases = ["```\n```\n```\n```", "~~~\n~~~\n~~~", "```js\n~~~\n```\n~~~", "````\n```\n````", "```\n````\n```", "  ```\n```\n  ```", "```\n  ```\n```", "```\n\n```\n\n```\n\n```"];

      testCases.forEach(input => {
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

    it("should never create MORE unbalanced regions than input", () => {
      // Test quality degradation detection
      const testCases = [
        "```\ncode\n```", // Balanced - should not modify
        "```javascript\nunclosed", // Unclosed - should add closing
        "```\ncode1\n```\n```\ncode2\n```", // Multiple balanced - should not modify
        "```\nnested\n```\n```\n```", // Unbalanced sequence
        "```markdown\n```\nexample\n```\n```", // Nested example
        "```\nfirst\n```\nsecond\n```\nthird\n```", // Partially balanced
      ];

      testCases.forEach(input => {
        const originalCounts = balancer.countCodeRegions(input);
        const result = balancer.balanceCodeRegions(input);
        const resultCounts = balancer.countCodeRegions(result);

        // Key quality invariant: never create MORE unbalanced regions
        expect(resultCounts.unbalanced).toBeLessThanOrEqual(originalCounts.unbalanced);
      });
    });

    it("should preserve balanced markdown exactly (except line ending normalization)", () => {
      const balancedExamples = ["```javascript\nconst x = 1;\n```", "~~~markdown\ntext\n~~~", "```\ngeneric\n```\n\n```python\ncode\n```", "# Title\n\n```bash\necho test\n```\n\nMore text", "````\nfour backticks\n````"];

      balancedExamples.forEach(input => {
        const result = balancer.balanceCodeRegions(input);
        expect(result).toBe(input);
      });
    });

    it("should handle AI-generated common error patterns", () => {
      // Common error pattern: AI generates nested markdown examples without proper escaping
      const aiPattern1 = `How to use code blocks:

\`\`\`markdown
You can write code like this:
\`\`\`javascript
code here
\`\`\`
\`\`\``;

      const result1 = balancer.balanceCodeRegions(aiPattern1);
      const counts1 = balancer.countCodeRegions(result1);

      // Result should have fewer or equal unbalanced regions
      const originalCounts1 = balancer.countCodeRegions(aiPattern1);
      expect(counts1.unbalanced).toBeLessThanOrEqual(originalCounts1.unbalanced);

      // Common error pattern: Unclosed code block at end of content
      const aiPattern2 = `Here's some code:

\`\`\`javascript
function example() {
  console.log("test");
}`;

      const result2 = balancer.balanceCodeRegions(aiPattern2);
      expect(balancer.isBalanced(result2)).toBe(true);

      // Common error pattern: Mixed fence types causing confusion
      const aiPattern3 = `\`\`\`markdown
Example with tilde:
~~~
content
~~~
\`\`\``;

      const result3 = balancer.balanceCodeRegions(aiPattern3);
      const counts3 = balancer.countCodeRegions(result3);
      expect(counts3.unbalanced).toBe(0);
    });

    it("should handle pathological cases without hanging", () => {
      // Generate pathological input: alternating fences
      let pathological = "";
      for (let i = 0; i < 100; i++) {
        pathological += i % 2 === 0 ? "```\n" : "~~~\n";
      }

      // Should complete in reasonable time (not hang)
      const start = Date.now();
      const result = balancer.balanceCodeRegions(pathological);
      const elapsed = Date.now() - start;

      expect(elapsed).toBeLessThan(1000); // Should complete in less than 1 second
      expect(typeof result).toBe("string");
    });

    it("should not corrupt sequential code blocks with different languages", () => {
      // Regression test: two separate code blocks where the first has a language tag
      // and the second is bare. The balancer should NOT merge them into one block
      // by increasing fence lengths.
      const input = `## C++ Source

\`\`\`cpp
template<typename T, size_t N>
const T interpolateN(const T &value, const T(&y)[N])
{
    return y[0];
}
\`\`\`

Used for motor thrust curves.

## Verification Status

\`\`\`
LEAN_AVAILABLE=true
LAKE_BUILD=passed
\`\`\`

All 19 build jobs passed.`;

      const result = balancer.balanceCodeRegions(input);
      // The output should be identical to the input - two separate balanced blocks
      expect(result).toBe(input);
      expect(balancer.isBalanced(result)).toBe(true);
    });

    it("should not corrupt three sequential code blocks", () => {
      // Three separate code blocks, none nested
      const input = `\`\`\`python
print("hello")
\`\`\`

Some text.

\`\`\`javascript
console.log("world")
\`\`\`

More text.

\`\`\`
plain code
\`\`\``;

      const result = balancer.balanceCodeRegions(input);
      expect(result).toBe(input);
      expect(balancer.isBalanced(result)).toBe(true);
    });

    it("should not corrupt two code blocks where first has language and second is bare", () => {
      // This is the exact pattern that caused the bug: ```lang ... ``` followed by ``` ... ```
      const input = `\`\`\`cpp
code1
\`\`\`

\`\`\`
code2
\`\`\``;

      const result = balancer.balanceCodeRegions(input);
      expect(result).toBe(input);
      expect(balancer.isBalanced(result)).toBe(true);
    });

    it("should handle random fence variations", () => {
      // Generate random fence lengths and types
      const fenceChars = ["`", "~"];
      const fenceLengths = [3, 4, 5, 6, 10];

      for (let i = 0; i < 20; i++) {
        const char = fenceChars[i % fenceChars.length];
        const length = fenceLengths[i % fenceLengths.length];
        const fence = char.repeat(length);
        const input = `${fence}javascript\ncode${i}\n${fence}`;

        const result = balancer.balanceCodeRegions(input);
        expect(balancer.isBalanced(result)).toBe(true);
      }
    });

    describe("real-world regression tests", () => {
      it("should not corrupt PR body with cpp code block followed by bare verification block", () => {
        // Exact pattern from the bug report: a ```cpp block followed by a ``` block
        // The balancer was wrapping both in ````...```` and treating the middle
        // closing/opening fences as content.
        const input = `## C++ Source

\`\`\`cpp
// src/lib/mathlib/math/Functions.hpp (~line 180)
template<typename T, size_t N>
const T interpolateN(const T &value, const T(&y)[N])
{
    size_t index = constrain((int)(value * (N - 1)), 0, (int)(N - 2));
    return interpolate(value,
                       (T)index / (T)(N - 1),
                       (T)(index + 1) / (T)(N - 1),
                       y[index], y[index + 1]);
}
\`\`\`

Used for motor thrust curves (N=5 or N=9), RC stick sensitivity curves, and control surface deflection mappings.

## Verification Status

> Proofs verified: lake build passed with Lean 4.29.0. 0 sorry remain.

\`\`\`
LEAN_AVAILABLE=true
Lean (version 4.29.0, x86_64-unknown-linux-gnu)
LAKE_BUILD=passed
\`\`\`

All 19 build jobs passed (including all 17 prior Lean files).`;

        const result = balancer.balanceCodeRegions(input);
        expect(result).toBe(input);
        expect(balancer.isBalanced(result)).toBe(true);
      });

      it("should not corrupt markdown with multiple language-tagged blocks and one bare block", () => {
        // Pattern: several language-tagged blocks + one bare block at the end
        const input = `## Code

\`\`\`python
def hello():
    print("Hello")
\`\`\`

## Tests

\`\`\`python
def test_hello():
    assert True
\`\`\`

## Output

\`\`\`
Hello
\`\`\``;

        const result = balancer.balanceCodeRegions(input);
        expect(result).toBe(input);
        expect(balancer.isBalanced(result)).toBe(true);
      });

      it("should not corrupt a document with 4+ sequential code blocks", () => {
        // Common in documentation: many sequential blocks
        const input = `\`\`\`bash
npm install
\`\`\`

\`\`\`javascript
const x = 1;
\`\`\`

\`\`\`json
{"key": "value"}
\`\`\`

\`\`\`
plain text output
\`\`\`

\`\`\`yaml
key: value
\`\`\``;

        const result = balancer.balanceCodeRegions(input);
        expect(result).toBe(input);
        expect(balancer.isBalanced(result)).toBe(true);
      });

      it("should handle GitHub-style blockquote containing a code block", () => {
        // Code blocks inside blockquotes are common in GitHub comments
        const input = `Some text before.

> Here is a quote with code:
> \`\`\`
> echo "hello"
> \`\`\`

Text after.`;

        const result = balancer.balanceCodeRegions(input);
        // The > prefix means these fences have different indentation patterns
        // and won't match top-level fences
        expect(result).toBe(input);
      });

      it("should handle PR body with footer containing fenced install command", () => {
        // Common pattern in gh-aw: PR bodies end with a fenced install command
        const input = `## Summary

Some PR summary text.

\`\`\`python
code here
\`\`\`

> Generated by workflow.
>
> To install, run
> \`\`\`
> gh aw add owner/repo/workflow.md@sha
> \`\`\``;

        const result = balancer.balanceCodeRegions(input);
        expect(result).toBe(input);
        expect(balancer.isBalanced(result)).toBe(true);
      });
    });

    describe("ambiguous nesting with intermediate language openers", () => {
      it("should not make things worse for markdown block with language-tagged inner block", () => {
        // Inherently ambiguous: greedy matching pairs ```markdown with the first
        // bare ```, so ```python and the final ``` become separate (unbalanced) blocks.
        // The balancer cannot resolve this but must not degrade the output.
        const input = `\`\`\`markdown
Here's an example:
\`\`\`python
print("hello")
\`\`\`
End of example
\`\`\``;
        const result = balancer.balanceCodeRegions(input);
        const inputCounts = balancer.countCodeRegions(input);
        const resultCounts = balancer.countCodeRegions(result);
        expect(resultCounts.unbalanced).toBeLessThanOrEqual(inputCounts.unbalanced);
      });

      it("should not make things worse for markdown block with multiple inner blocks", () => {
        // Multiple language-tagged inner blocks inside a markdown fence.
        // Greedy matching pairs the outer opener with the first bare closer,
        // leaving subsequent blocks ambiguous.
        const input = `\`\`\`markdown
## Usage

\`\`\`javascript
const x = 1;
\`\`\`

\`\`\`python
y = 2
\`\`\`

End
\`\`\``;
        const result = balancer.balanceCodeRegions(input);
        const inputCounts = balancer.countCodeRegions(input);
        const resultCounts = balancer.countCodeRegions(result);
        expect(resultCounts.unbalanced).toBeLessThanOrEqual(inputCounts.unbalanced);
      });
    });

    describe("idempotency", () => {
      it("should be idempotent - running twice gives same result", () => {
        const inputs = [
          "```javascript\ncode\n```",
          "```\nblock1\n```\n\n```\nblock2\n```",
          "```cpp\ncode\n```\n\n```\noutput\n```",
          "```markdown\n```python\ncode\n```\nend\n```",
          "```javascript\nunclosed",
          "```\nfirst\n```\nsecond\n```\nthird\n```",
        ];

        for (const input of inputs) {
          const first = balancer.balanceCodeRegions(input);
          const second = balancer.balanceCodeRegions(first);
          expect(second).toBe(first);
        }
      });
    });

    describe("output correctness invariants", () => {
      it("should always produce balanced output or no worse than input", () => {
        const inputs = [
          "```\ncode\n```",
          "```js\ncode",
          "```\na\n```\nb\n```\nc\n```",
          "```\na\n```\n```\nb\n```\n```\nc\n```",
          "````\na\n```\nb\n```\nc\n````",
          "```markdown\n```js\ncode\n```\n```",
          "~~~\na\n~~~\n```\nb\n```",
          "```\n```\n```",
        ];

        for (const input of inputs) {
          const result = balancer.balanceCodeRegions(input);
          const inputCounts = balancer.countCodeRegions(input);
          const resultCounts = balancer.countCodeRegions(result);
          expect(resultCounts.unbalanced).toBeLessThanOrEqual(inputCounts.unbalanced);
        }
      });

      it("should never change already-balanced content (beyond line-ending normalization)", () => {
        const balanced = [
          "```js\ncode\n```",
          "~~~md\ntext\n~~~",
          "```\na\n```\n\n```\nb\n```",
          "```cpp\ncode\n```\n\n```\noutput\n```",
          "````\ncode with ``` inside\n````",
          "```js\na\n```\n~~~py\nb\n~~~\n```\nc\n```",
          "  ```\n  indented\n  ```",
          "```js {highlight}\ncode\n```",
        ];

        for (const input of balanced) {
          expect(balancer.balanceCodeRegions(input)).toBe(input);
        }
      });
    });
  });
});
