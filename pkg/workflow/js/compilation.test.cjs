import { describe, it, expect } from "vitest";
import fs from "fs";
import path from "path";

describe("TypeScript compilation output", () => {
  it("generates JavaScript files with 2-space indentation", () => {
    const jsFiles = ["add_labels.js", "create_issue.js", "create_discussion.js"];

    jsFiles.forEach(fileName => {
      const filePath = path.join(process.cwd(), fileName);
      if (fs.existsSync(filePath)) {
        const content = fs.readFileSync(filePath, "utf8");
        const lines = content.split("\n");

        // Check that indented lines use 2-space increments
        lines.forEach((line, index) => {
          if (line.trim() !== "" && line.startsWith("  ")) {
            // Count leading spaces
            const leadingSpaces = line.match(/^(\s*)/)[1].length;
            // Should be a multiple of 2
            expect(leadingSpaces % 2, `Line ${index + 1} in ${fileName} has ${leadingSpaces} leading spaces, should be multiple of 2`).toBe(
              0
            );
          }
        });
      }
    });
  });

  it("uses consistent 2-space indentation pattern", () => {
    const jsFiles = ["add_labels.js", "create_issue.js", "create_discussion.js"];

    jsFiles.forEach(fileName => {
      const filePath = path.join(process.cwd(), fileName);
      if (fs.existsSync(filePath)) {
        const content = fs.readFileSync(filePath, "utf8");
        const lines = content.split("\n");

        // Look for function-level statements (should be 2 spaces from function start)
        let inFunction = false;
        lines.forEach((line, index) => {
          const trimmed = line.trim();

          // Track when we're inside a function
          if (trimmed.startsWith("async function") || trimmed.startsWith("function")) {
            inFunction = true;
          }

          // If we're in a function and find a basic statement, check indentation
          if (inFunction && trimmed !== "" && !trimmed.startsWith("//")) {
            const leadingSpaces = line.match(/^(\s*)/)[1].length;

            // Look for direct function body statements (should have 2 spaces minimum)
            if (
              trimmed.startsWith("const ") ||
              trimmed.startsWith("let ") ||
              trimmed.startsWith("if (") ||
              trimmed.startsWith("return ") ||
              trimmed.startsWith("core.")
            ) {
              // Should have at least 2 spaces if inside function
              expect(leadingSpaces >= 2, `Line ${index + 1} in ${fileName} should have at least 2 spaces: "${trimmed}"`).toBe(true);
            }
          }

          // Reset function tracking at end of function
          if (trimmed === "}" && inFunction) {
            inFunction = false;
          }
        });
      }
    });
  });
});
