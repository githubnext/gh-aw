import { describe, it, expect } from "vitest";
const { removeDuplicateTitleFromDescription } = require("./remove_duplicate_title.cjs");
describe("remove_duplicate_title.cjs", () => {
  describe("removeDuplicateTitleFromDescription", () => {
    (describe("basic functionality", () => {
      (it("should remove H1 header matching title", () => {
        expect(removeDuplicateTitleFromDescription("Bug Report", "# Bug Report\n\nThis is the body of the report.")).toBe("This is the body of the report.");
      }),
        it("should remove H2 header matching title", () => {
          expect(removeDuplicateTitleFromDescription("Feature Request", "## Feature Request\n\nThis is the feature description.")).toBe("This is the feature description.");
        }),
        it("should remove H3 header matching title", () => {
          expect(removeDuplicateTitleFromDescription("Documentation Update", "### Documentation Update\n\nThis is the documentation.")).toBe("This is the documentation.");
        }),
        it("should remove H4 header matching title", () => {
          expect(removeDuplicateTitleFromDescription("Refactoring", "#### Refactoring\n\nThis is the refactoring plan.")).toBe("This is the refactoring plan.");
        }),
        it("should remove H5 header matching title", () => {
          expect(removeDuplicateTitleFromDescription("Test", "##### Test\n\nThis is the test description.")).toBe("This is the test description.");
        }),
        it("should remove H6 header matching title", () => {
          expect(removeDuplicateTitleFromDescription("Note", "###### Note\n\nThis is the note content.")).toBe("This is the note content.");
        }));
    }),
      describe("case insensitivity", () => {
        (it("should match title case-insensitively", () => {
          expect(removeDuplicateTitleFromDescription("Bug Report", "# bug report\n\nBody content.")).toBe("Body content.");
        }),
          it("should match title with different casing", () => {
            expect(removeDuplicateTitleFromDescription("BUG REPORT", "# Bug Report\n\nBody content.")).toBe("Body content.");
          }),
          it("should match lowercase title", () => {
            expect(removeDuplicateTitleFromDescription("feature request", "# Feature Request\n\nBody content.")).toBe("Body content.");
          }));
      }),
      describe("whitespace handling", () => {
        (it("should handle extra spaces after hash", () => {
          expect(removeDuplicateTitleFromDescription("Title", "#    Title\n\nBody content.")).toBe("Body content.");
        }),
          it("should handle trailing spaces after title", () => {
            expect(removeDuplicateTitleFromDescription("Title", "# Title   \n\nBody content.")).toBe("Body content.");
          }),
          it("should handle multiple newlines after header", () => {
            expect(removeDuplicateTitleFromDescription("Title", "# Title\n\n\n\nBody content.")).toBe("Body content.");
          }),
          it("should handle CRLF line endings", () => {
            expect(removeDuplicateTitleFromDescription("Title", "# Title\r\n\r\nBody content.")).toBe("Body content.");
          }),
          it("should trim leading/trailing whitespace from inputs", () => {
            expect(removeDuplicateTitleFromDescription("  Title  ", "  # Title\n\nBody content.  ")).toBe("Body content.");
          }));
      }),
      describe("non-matching cases", () => {
        (it("should not remove header when title doesn't match", () => {
          expect(removeDuplicateTitleFromDescription("Bug Report", "# Feature Request\n\nBody content.")).toBe("# Feature Request\n\nBody content.");
        }),
          it("should not remove header when it's not at the start", () => {
            expect(removeDuplicateTitleFromDescription("Title", "Some text\n\n# Title\n\nBody content.")).toBe("Some text\n\n# Title\n\nBody content.");
          }),
          it("should not remove title if it's not in a header", () => {
            expect(removeDuplicateTitleFromDescription("Title", "Title\n\nBody content.")).toBe("Title\n\nBody content.");
          }),
          it("should preserve description when no header matches", () => {
            const description = "This is just body content without headers.";
            expect(removeDuplicateTitleFromDescription("Title", description)).toBe(description);
          }));
      }),
      describe("special characters in title", () => {
        (it("should handle title with parentheses", () => {
          expect(removeDuplicateTitleFromDescription("Bug Report (Important)", "# Bug Report (Important)\n\nBody content.")).toBe("Body content.");
        }),
          it("should handle title with brackets", () => {
            expect(removeDuplicateTitleFromDescription("Feature [v2.0]", "# Feature [v2.0]\n\nBody content.")).toBe("Body content.");
          }),
          it("should handle title with dots and asterisks", () => {
            expect(removeDuplicateTitleFromDescription("Fix *.txt files", "# Fix *.txt files\n\nBody content.")).toBe("Body content.");
          }),
          it("should handle title with plus and question marks", () => {
            expect(removeDuplicateTitleFromDescription("C++ Update?", "# C++ Update?\n\nBody content.")).toBe("Body content.");
          }),
          it("should handle title with dollar signs", () => {
            expect(removeDuplicateTitleFromDescription("Fix $VAR usage", "# Fix $VAR usage\n\nBody content.")).toBe("Body content.");
          }),
          it("should handle title with carets and pipes", () => {
            expect(removeDuplicateTitleFromDescription("Test ^pattern|filter", "# Test ^pattern|filter\n\nBody content.")).toBe("Body content.");
          }),
          it("should handle title with curly braces", () => {
            expect(removeDuplicateTitleFromDescription("Fix {key: value}", "# Fix {key: value}\n\nBody content.")).toBe("Body content.");
          }),
          it("should handle title with backslashes", () => {
            expect(removeDuplicateTitleFromDescription("Path\\to\\file", "# Path\\to\\file\n\nBody content.")).toBe("Body content.");
          }));
      }),
      describe("edge cases", () => {
        (it("should return empty string when both inputs are empty", () => {
          expect(removeDuplicateTitleFromDescription("", "")).toBe("");
        }),
          it("should return empty string when description is empty", () => {
            expect(removeDuplicateTitleFromDescription("Title", "")).toBe("");
          }),
          it("should return description when title is empty", () => {
            expect(removeDuplicateTitleFromDescription("", "# Some Header\n\nBody content.")).toBe("# Some Header\n\nBody content.");
          }),
          it("should handle null title", () => {
            expect(removeDuplicateTitleFromDescription(null, "# Title\n\nBody content.")).toBe("# Title\n\nBody content.");
          }),
          it("should handle undefined title", () => {
            expect(removeDuplicateTitleFromDescription(void 0, "# Title\n\nBody content.")).toBe("# Title\n\nBody content.");
          }),
          it("should handle null description", () => {
            expect(removeDuplicateTitleFromDescription("Title", null)).toBe("");
          }),
          it("should handle undefined description", () => {
            expect(removeDuplicateTitleFromDescription("Title", void 0)).toBe("");
          }),
          it("should handle non-string title", () => {
            expect(removeDuplicateTitleFromDescription(123, "# 123\n\nBody content.")).toBe("# 123\n\nBody content.");
          }),
          it("should handle non-string description", () => {
            expect(removeDuplicateTitleFromDescription("Title", 123)).toBe("");
          }));
      }),
      describe("complex scenarios", () => {
        (it("should handle description with only the header", () => {
          expect(removeDuplicateTitleFromDescription("Title", "# Title")).toBe("");
        }),
          it("should handle description with header and no content", () => {
            expect(removeDuplicateTitleFromDescription("Title", "# Title\n\n")).toBe("");
          }),
          it("should preserve other headers in the description", () => {
            expect(removeDuplicateTitleFromDescription("Main Title", "# Main Title\n\n## Section 1\n\nContent here.\n\n## Section 2\n\nMore content.")).toBe("## Section 1\n\nContent here.\n\n## Section 2\n\nMore content.");
          }),
          it("should handle description with multiple paragraphs", () => {
            expect(removeDuplicateTitleFromDescription("Bug Report", "# Bug Report\n\nFirst paragraph.\n\nSecond paragraph.\n\nThird paragraph.")).toBe("First paragraph.\n\nSecond paragraph.\n\nThird paragraph.");
          }),
          it("should handle description with code blocks", () => {
            expect(removeDuplicateTitleFromDescription("Code Fix", "# Code Fix\n\n```js\nconst x = 1;\n```\n\nExplanation.")).toBe("```js\nconst x = 1;\n```\n\nExplanation.");
          }),
          it("should handle description with lists", () => {
            expect(removeDuplicateTitleFromDescription("Tasks", "# Tasks\n\n- Task 1\n- Task 2\n- Task 3")).toBe("- Task 1\n- Task 2\n- Task 3");
          }),
          it("should handle title with numbers", () => {
            expect(removeDuplicateTitleFromDescription("Version 2.0 Release", "# Version 2.0 Release\n\nRelease notes here.")).toBe("Release notes here.");
          }),
          it("should handle very long titles", () => {
            const title = "This is a very long title that contains many words and should still be matched correctly";
            expect(removeDuplicateTitleFromDescription(title, `# ${title}\n\nBody content.`)).toBe("Body content.");
          }),
          it("should handle emoji in title", () => {
            expect(removeDuplicateTitleFromDescription("🐛 Bug Report", "# 🐛 Bug Report\n\nBody content.")).toBe("Body content.");
          }),
          it("should handle unicode characters in title", () => {
            expect(removeDuplicateTitleFromDescription("Прошу исправить ошибку", "# Прошу исправить ошибку\n\nОписание проблемы.")).toBe("Описание проблемы.");
          }));
      }),
      describe("real-world examples", () => {
        (it("should handle GitHub issue format", () => {
          expect(
            removeDuplicateTitleFromDescription("Feature: Add dark mode", "# Feature: Add dark mode\n\n## Description\n\nWe need dark mode support.\n\n## Acceptance Criteria\n\n- [ ] Dark mode toggle\n- [ ] Persistent preference")
          ).toBe("## Description\n\nWe need dark mode support.\n\n## Acceptance Criteria\n\n- [ ] Dark mode toggle\n- [ ] Persistent preference");
        }),
          it("should handle pull request format", () => {
            expect(removeDuplicateTitleFromDescription("Fix authentication bug", "# Fix authentication bug\n\n## Changes\n\n- Updated auth flow\n- Added tests\n\n## Testing\n\nManually tested all scenarios.")).toBe(
              "## Changes\n\n- Updated auth flow\n- Added tests\n\n## Testing\n\nManually tested all scenarios."
            );
          }),
          it("should handle discussion format", () => {
            expect(removeDuplicateTitleFromDescription("How to configure X?", "# How to configure X?\n\nI'm trying to configure X but can't find the documentation.\n\nCan someone help?")).toBe(
              "I'm trying to configure X but can't find the documentation.\n\nCan someone help?"
            );
          }));
      }),
      describe("performance", () => {
        (it("should handle large descriptions efficiently", () => {
          const largeBody = "Body content.\n".repeat(1e3),
            result = removeDuplicateTitleFromDescription("Title", `# Title\n\n${largeBody}`);
          expect(result).toBe(largeBody.trim());
        }),
          it("should handle multiple consecutive calls", () => {
            for (let i = 0; i < 100; i++) expect(removeDuplicateTitleFromDescription("Title", "# Title\n\nBody content.")).toBe("Body content.");
          }));
      }));
  });
});
