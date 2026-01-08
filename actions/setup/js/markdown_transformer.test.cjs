import { describe, it, expect } from "vitest";

describe("markdown_transformer.cjs", () => {
  let markdownTransformer;

  beforeEach(async () => {
    markdownTransformer = await import("./markdown_transformer.cjs");
  });

  describe("increaseHeaderLevel", () => {
    it("should transform h1 to h2", () => {
      const input = "# Header 1";
      const expected = "## Header 1";
      expect(markdownTransformer.increaseHeaderLevel(input)).toBe(expected);
    });

    it("should transform h2 to h3", () => {
      const input = "## Header 2";
      const expected = "### Header 2";
      expect(markdownTransformer.increaseHeaderLevel(input)).toBe(expected);
    });

    it("should transform h3 to h4", () => {
      const input = "### Header 3";
      const expected = "#### Header 3";
      expect(markdownTransformer.increaseHeaderLevel(input)).toBe(expected);
    });

    it("should transform h4 to h5", () => {
      const input = "#### Header 4";
      const expected = "##### Header 4";
      expect(markdownTransformer.increaseHeaderLevel(input)).toBe(expected);
    });

    it("should transform h5 to h6", () => {
      const input = "##### Header 5";
      const expected = "###### Header 5";
      expect(markdownTransformer.increaseHeaderLevel(input)).toBe(expected);
    });

    it("should not transform h6 (max level)", () => {
      const input = "###### Header 6";
      const expected = "###### Header 6";
      expect(markdownTransformer.increaseHeaderLevel(input)).toBe(expected);
    });

    it("should transform multiple headers in sequence", () => {
      const input = `# Title
## Section
### Subsection
#### Detail`;
      const expected = `## Title
### Section
#### Subsection
##### Detail`;
      expect(markdownTransformer.increaseHeaderLevel(input)).toBe(expected);
    });

    it("should preserve non-header lines", () => {
      const input = `# Header

This is a paragraph.

## Another Header

More text here.`;
      const expected = `## Header

This is a paragraph.

### Another Header

More text here.`;
      expect(markdownTransformer.increaseHeaderLevel(input)).toBe(expected);
    });

    it("should not transform headers in fenced code blocks (backticks)", () => {
      const input = `# Real Header

\`\`\`
# Fake Header
## Another Fake
\`\`\`

## Real Header 2`;
      const expected = `## Real Header

\`\`\`
# Fake Header
## Another Fake
\`\`\`

### Real Header 2`;
      expect(markdownTransformer.increaseHeaderLevel(input)).toBe(expected);
    });

    it("should not transform headers in fenced code blocks (tildes)", () => {
      const input = `# Real Header

~~~
# Fake Header
## Another Fake
~~~

## Real Header 2`;
      const expected = `## Real Header

~~~
# Fake Header
## Another Fake
~~~

### Real Header 2`;
      expect(markdownTransformer.increaseHeaderLevel(input)).toBe(expected);
    });

    it("should not transform headers in code blocks with language specifier", () => {
      const input = `# Real Header

\`\`\`markdown
# Fake Header in Markdown
## Another Fake
\`\`\`

## Real Header 2`;
      const expected = `## Real Header

\`\`\`markdown
# Fake Header in Markdown
## Another Fake
\`\`\`

### Real Header 2`;
      expect(markdownTransformer.increaseHeaderLevel(input)).toBe(expected);
    });

    it("should handle nested code blocks correctly", () => {
      const input = `# Header

\`\`\`
Outer code block
# Not a header
\`\`\`

## Section

\`\`\`
Another code block
### Still not a header
\`\`\``;
      const expected = `## Header

\`\`\`
Outer code block
# Not a header
\`\`\`

### Section

\`\`\`
Another code block
### Still not a header
\`\`\``;
      expect(markdownTransformer.increaseHeaderLevel(input)).toBe(expected);
    });

    it("should preserve indentation in headers", () => {
      const input = `# Header
  ## Indented Header
    ### More Indented`;
      const expected = `## Header
  ### Indented Header
    #### More Indented`;
      expect(markdownTransformer.increaseHeaderLevel(input)).toBe(expected);
    });

    it("should handle headers with special characters", () => {
      const input = `# Header with **bold**
## Header with *italic*
### Header with \`code\`
#### Header with [link](url)`;
      const expected = `## Header with **bold**
### Header with *italic*
#### Header with \`code\`
##### Header with [link](url)`;
      expect(markdownTransformer.increaseHeaderLevel(input)).toBe(expected);
    });

    it("should handle headers with trailing spaces", () => {
      const input = "# Header   \n## Section  ";
      const expected = "## Header   \n### Section  ";
      expect(markdownTransformer.increaseHeaderLevel(input)).toBe(expected);
    });

    it("should handle headers with multiple spaces after #", () => {
      const input = "#  Header\n##   Section";
      const expected = "##  Header\n###   Section";
      expect(markdownTransformer.increaseHeaderLevel(input)).toBe(expected);
    });

    it("should not transform lines without space after #", () => {
      const input = "#No space\n## Valid Header";
      const expected = "#No space\n### Valid Header";
      expect(markdownTransformer.increaseHeaderLevel(input)).toBe(expected);
    });

    it("should not transform # in middle of line", () => {
      const input = "This is # not a header\n# This is a header";
      const expected = "This is # not a header\n## This is a header";
      expect(markdownTransformer.increaseHeaderLevel(input)).toBe(expected);
    });

    it("should handle empty string", () => {
      expect(markdownTransformer.increaseHeaderLevel("")).toBe("");
    });

    it("should handle null input", () => {
      expect(markdownTransformer.increaseHeaderLevel(null)).toBe("");
    });

    it("should handle undefined input", () => {
      expect(markdownTransformer.increaseHeaderLevel(undefined)).toBe("");
    });

    it("should handle markdown with no headers", () => {
      const input = "Just some text\nWith multiple lines\nBut no headers";
      expect(markdownTransformer.increaseHeaderLevel(input)).toBe(input);
    });

    it("should handle mixed ATX headers and content", () => {
      const input = `# Main Title

Paragraph with some content.

## Section 1

Some text here.

### Subsection 1.1

More content.

## Section 2

Final section.`;
      const expected = `## Main Title

Paragraph with some content.

### Section 1

Some text here.

#### Subsection 1.1

More content.

### Section 2

Final section.`;
      expect(markdownTransformer.increaseHeaderLevel(input)).toBe(expected);
    });

    it("should handle headers with emojis", () => {
      const input = "# 🚀 Header\n## 📝 Section";
      const expected = "## 🚀 Header\n### 📝 Section";
      expect(markdownTransformer.increaseHeaderLevel(input)).toBe(expected);
    });

    it("should handle headers with numbers", () => {
      const input = "# 1. First Header\n## 2.1 Second Header";
      const expected = "## 1. First Header\n### 2.1 Second Header";
      expect(markdownTransformer.increaseHeaderLevel(input)).toBe(expected);
    });

    it("should handle complex real-world example", () => {
      const input = `# Conversation Summary

## Initialization

Model: gpt-4

## Turn 1

### User

Hello, world!

### Assistant

Hi there!

## Turn 2

### User

\`\`\`javascript
# This is not a header
const x = 1;
\`\`\`

### Assistant

\`\`\`
# Also not a header
## Nope
\`\`\`

## Final Summary

All done!`;

      const expected = `## Conversation Summary

### Initialization

Model: gpt-4

### Turn 1

#### User

Hello, world!

#### Assistant

Hi there!

### Turn 2

#### User

\`\`\`javascript
# This is not a header
const x = 1;
\`\`\`

#### Assistant

\`\`\`
# Also not a header
## Nope
\`\`\`

### Final Summary

All done!`;
      expect(markdownTransformer.increaseHeaderLevel(input)).toBe(expected);
    });

    it("should handle code blocks with longer fences", () => {
      const input = `# Header

\`\`\`\`
# Not a header
\`\`\`\`

## Section`;
      const expected = `## Header

\`\`\`\`
# Not a header
\`\`\`\`

### Section`;
      expect(markdownTransformer.increaseHeaderLevel(input)).toBe(expected);
    });

    it("should handle indented code blocks", () => {
      const input = `# Header

  \`\`\`
  # Not a header
  \`\`\`

## Section`;
      const expected = `## Header

  \`\`\`
  # Not a header
  \`\`\`

### Section`;
      expect(markdownTransformer.increaseHeaderLevel(input)).toBe(expected);
    });

    it("should handle Windows line endings", () => {
      const input = "# Header\r\n## Section\r\n";
      const result = markdownTransformer.increaseHeaderLevel(input);
      // Line endings are normalized to \n
      expect(result).toBe("## Header\n### Section\n");
    });

    it("should handle headers at different nesting levels", () => {
      const input = `# H1
## H2
### H3
#### H4
##### H5
###### H6
##### H5 again
#### H4 again
### H3 again
## H2 again
# H1 again`;
      const expected = `## H1
### H2
#### H3
##### H4
###### H5
###### H6
###### H5 again
##### H4 again
#### H3 again
### H2 again
## H1 again`;
      expect(markdownTransformer.increaseHeaderLevel(input)).toBe(expected);
    });

    it("should handle headers with HTML entities", () => {
      const input = "# Header &amp; Content\n## Section &lt;tag&gt;";
      const expected = "## Header &amp; Content\n### Section &lt;tag&gt;";
      expect(markdownTransformer.increaseHeaderLevel(input)).toBe(expected);
    });

    it("should not transform setext-style headers", () => {
      const input = `Header 1
========

Header 2
--------

# ATX Header`;
      // Setext headers are not transformed, only ATX
      const expected = `Header 1
========

Header 2
--------

## ATX Header`;
      expect(markdownTransformer.increaseHeaderLevel(input)).toBe(expected);
    });

    it("should handle consecutive code blocks", () => {
      const input = `# Header

\`\`\`
Block 1
# Not a header
\`\`\`

\`\`\`
Block 2
## Also not a header
\`\`\`

## Section`;
      const expected = `## Header

\`\`\`
Block 1
# Not a header
\`\`\`

\`\`\`
Block 2
## Also not a header
\`\`\`

### Section`;
      expect(markdownTransformer.increaseHeaderLevel(input)).toBe(expected);
    });

    it("should handle empty headers", () => {
      const input = "# \n## Section";
      const expected = "## \n### Section";
      expect(markdownTransformer.increaseHeaderLevel(input)).toBe(expected);
    });

    it("should handle headers with only whitespace after #", () => {
      const input = "#    \n## Section";
      const expected = "##    \n### Section";
      expect(markdownTransformer.increaseHeaderLevel(input)).toBe(expected);
    });
  });
});
