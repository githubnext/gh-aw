import { describe, it, expect } from "vitest";

// Import the function to test
const { removeDuplicateTitleFromDescription } = require("./remove_duplicate_title.cjs");

describe("remove_duplicate_title.cjs", () => {
  describe("removeDuplicateTitleFromDescription", () => {
    // Basic functionality tests
    describe("basic functionality", () => {
      it("should remove H1 header matching title", () => {
        const title = "Bug Report";
        const description = "# Bug Report\n\nThis is the body of the report.";
        const expected = "This is the body of the report.";
        expect(removeDuplicateTitleFromDescription(title, description)).toBe(expected);
      });

      it("should remove H2 header matching title", () => {
        const title = "Feature Request";
        const description = "## Feature Request\n\nThis is the feature description.";
        const expected = "This is the feature description.";
        expect(removeDuplicateTitleFromDescription(title, description)).toBe(expected);
      });

      it("should remove H3 header matching title", () => {
        const title = "Documentation Update";
        const description = "### Documentation Update\n\nThis is the documentation.";
        const expected = "This is the documentation.";
        expect(removeDuplicateTitleFromDescription(title, description)).toBe(expected);
      });

      it("should remove H4 header matching title", () => {
        const title = "Refactoring";
        const description = "#### Refactoring\n\nThis is the refactoring plan.";
        const expected = "This is the refactoring plan.";
        expect(removeDuplicateTitleFromDescription(title, description)).toBe(expected);
      });

      it("should remove H5 header matching title", () => {
        const title = "Test";
        const description = "##### Test\n\nThis is the test description.";
        const expected = "This is the test description.";
        expect(removeDuplicateTitleFromDescription(title, description)).toBe(expected);
      });

      it("should remove H6 header matching title", () => {
        const title = "Note";
        const description = "###### Note\n\nThis is the note content.";
        const expected = "This is the note content.";
        expect(removeDuplicateTitleFromDescription(title, description)).toBe(expected);
      });
    });

    // Case insensitivity tests
    describe("case insensitivity", () => {
      it("should match title case-insensitively", () => {
        const title = "Bug Report";
        const description = "# bug report\n\nBody content.";
        const expected = "Body content.";
        expect(removeDuplicateTitleFromDescription(title, description)).toBe(expected);
      });

      it("should match title with different casing", () => {
        const title = "BUG REPORT";
        const description = "# Bug Report\n\nBody content.";
        const expected = "Body content.";
        expect(removeDuplicateTitleFromDescription(title, description)).toBe(expected);
      });

      it("should match lowercase title", () => {
        const title = "feature request";
        const description = "# Feature Request\n\nBody content.";
        const expected = "Body content.";
        expect(removeDuplicateTitleFromDescription(title, description)).toBe(expected);
      });
    });

    // Whitespace handling tests
    describe("whitespace handling", () => {
      it("should handle extra spaces after hash", () => {
        const title = "Title";
        const description = "#    Title\n\nBody content.";
        const expected = "Body content.";
        expect(removeDuplicateTitleFromDescription(title, description)).toBe(expected);
      });

      it("should handle trailing spaces after title", () => {
        const title = "Title";
        const description = "# Title   \n\nBody content.";
        const expected = "Body content.";
        expect(removeDuplicateTitleFromDescription(title, description)).toBe(expected);
      });

      it("should handle multiple newlines after header", () => {
        const title = "Title";
        const description = "# Title\n\n\n\nBody content.";
        const expected = "Body content.";
        expect(removeDuplicateTitleFromDescription(title, description)).toBe(expected);
      });

      it("should handle CRLF line endings", () => {
        const title = "Title";
        const description = "# Title\r\n\r\nBody content.";
        const expected = "Body content.";
        expect(removeDuplicateTitleFromDescription(title, description)).toBe(expected);
      });

      it("should trim leading/trailing whitespace from inputs", () => {
        const title = "  Title  ";
        const description = "  # Title\n\nBody content.  ";
        const expected = "Body content.";
        expect(removeDuplicateTitleFromDescription(title, description)).toBe(expected);
      });
    });

    // Non-matching cases
    describe("non-matching cases", () => {
      it("should not remove header when title doesn't match", () => {
        const title = "Bug Report";
        const description = "# Feature Request\n\nBody content.";
        expect(removeDuplicateTitleFromDescription(title, description)).toBe(description);
      });

      it("should not remove header when it's not at the start", () => {
        const title = "Title";
        const description = "Some text\n\n# Title\n\nBody content.";
        expect(removeDuplicateTitleFromDescription(title, description)).toBe(description);
      });

      it("should not remove title if it's not in a header", () => {
        const title = "Title";
        const description = "Title\n\nBody content.";
        expect(removeDuplicateTitleFromDescription(title, description)).toBe(description);
      });

      it("should preserve description when no header matches", () => {
        const title = "Title";
        const description = "This is just body content without headers.";
        expect(removeDuplicateTitleFromDescription(title, description)).toBe(description);
      });
    });

    // Special characters in title
    describe("special characters in title", () => {
      it("should handle title with parentheses", () => {
        const title = "Bug Report (Important)";
        const description = "# Bug Report (Important)\n\nBody content.";
        const expected = "Body content.";
        expect(removeDuplicateTitleFromDescription(title, description)).toBe(expected);
      });

      it("should handle title with brackets", () => {
        const title = "Feature [v2.0]";
        const description = "# Feature [v2.0]\n\nBody content.";
        const expected = "Body content.";
        expect(removeDuplicateTitleFromDescription(title, description)).toBe(expected);
      });

      it("should handle title with dots and asterisks", () => {
        const title = "Fix *.txt files";
        const description = "# Fix *.txt files\n\nBody content.";
        const expected = "Body content.";
        expect(removeDuplicateTitleFromDescription(title, description)).toBe(expected);
      });

      it("should handle title with plus and question marks", () => {
        const title = "C++ Update?";
        const description = "# C++ Update?\n\nBody content.";
        const expected = "Body content.";
        expect(removeDuplicateTitleFromDescription(title, description)).toBe(expected);
      });

      it("should handle title with dollar signs", () => {
        const title = "Fix $VAR usage";
        const description = "# Fix $VAR usage\n\nBody content.";
        const expected = "Body content.";
        expect(removeDuplicateTitleFromDescription(title, description)).toBe(expected);
      });

      it("should handle title with carets and pipes", () => {
        const title = "Test ^pattern|filter";
        const description = "# Test ^pattern|filter\n\nBody content.";
        const expected = "Body content.";
        expect(removeDuplicateTitleFromDescription(title, description)).toBe(expected);
      });

      it("should handle title with curly braces", () => {
        const title = "Fix {key: value}";
        const description = "# Fix {key: value}\n\nBody content.";
        const expected = "Body content.";
        expect(removeDuplicateTitleFromDescription(title, description)).toBe(expected);
      });

      it("should handle title with backslashes", () => {
        const title = "Path\\to\\file";
        const description = "# Path\\to\\file\n\nBody content.";
        const expected = "Body content.";
        expect(removeDuplicateTitleFromDescription(title, description)).toBe(expected);
      });
    });

    // Edge cases
    describe("edge cases", () => {
      it("should return empty string when both inputs are empty", () => {
        expect(removeDuplicateTitleFromDescription("", "")).toBe("");
      });

      it("should return empty string when description is empty", () => {
        expect(removeDuplicateTitleFromDescription("Title", "")).toBe("");
      });

      it("should return description when title is empty", () => {
        const description = "# Some Header\n\nBody content.";
        expect(removeDuplicateTitleFromDescription("", description)).toBe(description);
      });

      it("should handle null title", () => {
        const description = "# Title\n\nBody content.";
        expect(removeDuplicateTitleFromDescription(null, description)).toBe(description);
      });

      it("should handle undefined title", () => {
        const description = "# Title\n\nBody content.";
        expect(removeDuplicateTitleFromDescription(undefined, description)).toBe(description);
      });

      it("should handle null description", () => {
        expect(removeDuplicateTitleFromDescription("Title", null)).toBe("");
      });

      it("should handle undefined description", () => {
        expect(removeDuplicateTitleFromDescription("Title", undefined)).toBe("");
      });

      it("should handle non-string title", () => {
        const description = "# 123\n\nBody content.";
        expect(removeDuplicateTitleFromDescription(123, description)).toBe(description);
      });

      it("should handle non-string description", () => {
        expect(removeDuplicateTitleFromDescription("Title", 123)).toBe("");
      });
    });

    // Complex scenarios
    describe("complex scenarios", () => {
      it("should handle description with only the header", () => {
        const title = "Title";
        const description = "# Title";
        expect(removeDuplicateTitleFromDescription(title, description)).toBe("");
      });

      it("should handle description with header and no content", () => {
        const title = "Title";
        const description = "# Title\n\n";
        expect(removeDuplicateTitleFromDescription(title, description)).toBe("");
      });

      it("should preserve other headers in the description", () => {
        const title = "Main Title";
        const description = "# Main Title\n\n## Section 1\n\nContent here.\n\n## Section 2\n\nMore content.";
        const expected = "## Section 1\n\nContent here.\n\n## Section 2\n\nMore content.";
        expect(removeDuplicateTitleFromDescription(title, description)).toBe(expected);
      });

      it("should handle description with multiple paragraphs", () => {
        const title = "Bug Report";
        const description = "# Bug Report\n\nFirst paragraph.\n\nSecond paragraph.\n\nThird paragraph.";
        const expected = "First paragraph.\n\nSecond paragraph.\n\nThird paragraph.";
        expect(removeDuplicateTitleFromDescription(title, description)).toBe(expected);
      });

      it("should handle description with code blocks", () => {
        const title = "Code Fix";
        const description = "# Code Fix\n\n```js\nconst x = 1;\n```\n\nExplanation.";
        const expected = "```js\nconst x = 1;\n```\n\nExplanation.";
        expect(removeDuplicateTitleFromDescription(title, description)).toBe(expected);
      });

      it("should handle description with lists", () => {
        const title = "Tasks";
        const description = "# Tasks\n\n- Task 1\n- Task 2\n- Task 3";
        const expected = "- Task 1\n- Task 2\n- Task 3";
        expect(removeDuplicateTitleFromDescription(title, description)).toBe(expected);
      });

      it("should handle title with numbers", () => {
        const title = "Version 2.0 Release";
        const description = "# Version 2.0 Release\n\nRelease notes here.";
        const expected = "Release notes here.";
        expect(removeDuplicateTitleFromDescription(title, description)).toBe(expected);
      });

      it("should handle very long titles", () => {
        const title = "This is a very long title that contains many words and should still be matched correctly";
        const description = `# ${title}\n\nBody content.`;
        const expected = "Body content.";
        expect(removeDuplicateTitleFromDescription(title, description)).toBe(expected);
      });

      it("should handle emoji in title", () => {
        const title = "🐛 Bug Report";
        const description = "# 🐛 Bug Report\n\nBody content.";
        const expected = "Body content.";
        expect(removeDuplicateTitleFromDescription(title, description)).toBe(expected);
      });

      it("should handle unicode characters in title", () => {
        const title = "Прошу исправить ошибку";
        const description = "# Прошу исправить ошибку\n\nОписание проблемы.";
        const expected = "Описание проблемы.";
        expect(removeDuplicateTitleFromDescription(title, description)).toBe(expected);
      });
    });

    // Real-world examples
    describe("real-world examples", () => {
      it("should handle GitHub issue format", () => {
        const title = "Feature: Add dark mode";
        const description = "# Feature: Add dark mode\n\n## Description\n\nWe need dark mode support.\n\n## Acceptance Criteria\n\n- [ ] Dark mode toggle\n- [ ] Persistent preference";
        const expected = "## Description\n\nWe need dark mode support.\n\n## Acceptance Criteria\n\n- [ ] Dark mode toggle\n- [ ] Persistent preference";
        expect(removeDuplicateTitleFromDescription(title, description)).toBe(expected);
      });

      it("should handle pull request format", () => {
        const title = "Fix authentication bug";
        const description = "# Fix authentication bug\n\n## Changes\n\n- Updated auth flow\n- Added tests\n\n## Testing\n\nManually tested all scenarios.";
        const expected = "## Changes\n\n- Updated auth flow\n- Added tests\n\n## Testing\n\nManually tested all scenarios.";
        expect(removeDuplicateTitleFromDescription(title, description)).toBe(expected);
      });

      it("should handle discussion format", () => {
        const title = "How to configure X?";
        const description = "# How to configure X?\n\nI'm trying to configure X but can't find the documentation.\n\nCan someone help?";
        const expected = "I'm trying to configure X but can't find the documentation.\n\nCan someone help?";
        expect(removeDuplicateTitleFromDescription(title, description)).toBe(expected);
      });
    });

    // Performance considerations
    describe("performance", () => {
      it("should handle large descriptions efficiently", () => {
        const title = "Title";
        const largeBody = "Body content.\n".repeat(1000);
        const description = `# Title\n\n${largeBody}`;
        const result = removeDuplicateTitleFromDescription(title, description);
        expect(result).toBe(largeBody.trim());
      });

      it("should handle multiple consecutive calls", () => {
        const title = "Title";
        const description = "# Title\n\nBody content.";
        const expected = "Body content.";

        for (let i = 0; i < 100; i++) {
          expect(removeDuplicateTitleFromDescription(title, description)).toBe(expected);
        }
      });
    });
  });
});
