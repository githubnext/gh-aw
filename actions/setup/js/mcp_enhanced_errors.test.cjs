import { describe, it, expect } from "vitest";
import { generateEnhancedErrorMessage, generateExample, getExampleValue } from "./mcp_enhanced_errors.cjs";

describe("mcp_enhanced_errors.cjs", () => {
  describe("getExampleValue", () => {
    it("should return array for array type", () => {
      const schema = { type: "array" };
      const result = getExampleValue("generic", schema);
      expect(result).toEqual(["example"]);
    });

    it("should return labels array for labels field", () => {
      const schema = { type: "array" };
      const result = getExampleValue("labels", schema);
      expect(result).toEqual(["bug", "enhancement"]);
    });

    it("should return reviewers array for reviewers field", () => {
      const schema = { type: "array" };
      const result = getExampleValue("reviewers", schema);
      expect(result).toEqual(["octocat"]);
    });

    it("should return number for number type", () => {
      const schema = { type: "number" };
      const result = getExampleValue("count", schema);
      expect(result).toBe(42);
    });

    it("should return 123 for fields with 'number' in name", () => {
      const schema = { type: "number" };
      const result = getExampleValue("item_number", schema);
      expect(result).toBe(123);
    });

    it("should return enum value if present", () => {
      const schema = { type: "string", enum: ["RESOLVED", "DUPLICATE"] };
      const result = getExampleValue("reason", schema);
      expect(result).toBe("RESOLVED");
    });

    it("should return specific value for title field", () => {
      const schema = { type: "string" };
      const result = getExampleValue("title", schema);
      expect(result).toBe("Issue title");
    });

    it("should return specific value for body field", () => {
      const schema = { type: "string" };
      const result = getExampleValue("body", schema);
      expect(result).toBe("Your comment or description text");
    });

    it("should return default string for unknown field", () => {
      const schema = { type: "string" };
      const result = getExampleValue("unknown_field", schema);
      expect(result).toBe("example value");
    });
  });

  describe("generateExample", () => {
    it("should generate example with required fields", () => {
      const schema = {
        type: "object",
        required: ["title", "body"],
        properties: {
          title: { type: "string", description: "The title" },
          body: { type: "string", description: "The body" },
        },
      };
      const result = generateExample("create_issue", schema);
      const parsed = JSON.parse(result);
      expect(parsed).toHaveProperty("title");
      expect(parsed).toHaveProperty("body");
      expect(parsed.title).toBe("Issue title");
      expect(parsed.body).toBe("Your comment or description text");
    });

    it("should include optional field in example", () => {
      const schema = {
        type: "object",
        required: ["body"],
        properties: {
          body: { type: "string", description: "The body" },
          labels: { type: "array", description: "Labels" },
        },
      };
      const result = generateExample("create_issue", schema);
      const parsed = JSON.parse(result);
      expect(parsed).toHaveProperty("body");
      expect(parsed).toHaveProperty("labels");
    });

    it("should handle schema with no properties", () => {
      const schema = { type: "object" };
      const result = generateExample("noop", schema);
      expect(result).toBe("{}");
    });

    it("should handle null schema", () => {
      const result = generateExample("test", null);
      expect(result).toBe("{}");
    });
  });

  describe("generateEnhancedErrorMessage", () => {
    it("should generate enhanced error for single missing field", () => {
      const schema = {
        type: "object",
        required: ["item_number", "body"],
        properties: {
          item_number: {
            type: "number",
            description: "The issue, pull request, or discussion number to comment on.",
          },
          body: { type: "string", description: "Comment content in Markdown." },
        },
      };
      const result = generateEnhancedErrorMessage(["item_number"], "add_comment", schema);

      expect(result).toContain("Invalid arguments: missing or empty 'item_number'");
      expect(result).toContain("Required parameter 'item_number': The issue, pull request, or discussion number to comment on.");
      expect(result).toContain("Example:");
      expect(result).toContain('"item_number": 123');
      expect(result).toContain('"body"');
    });

    it("should generate enhanced error for multiple missing fields", () => {
      const schema = {
        type: "object",
        required: ["title", "body"],
        properties: {
          title: { type: "string", description: "Issue title." },
          body: { type: "string", description: "Issue body." },
        },
      };
      const result = generateEnhancedErrorMessage(["title", "body"], "create_issue", schema);

      expect(result).toContain("Invalid arguments: missing or empty 'title', 'body'");
      expect(result).toContain("Required parameter 'title': Issue title.");
      expect(result).toContain("Required parameter 'body': Issue body.");
      expect(result).toContain("Example:");
    });

    it("should handle missing fields without descriptions", () => {
      const schema = {
        type: "object",
        required: ["field1"],
        properties: {
          field1: { type: "string" },
        },
      };
      const result = generateEnhancedErrorMessage(["field1"], "test_tool", schema);

      expect(result).toContain("Invalid arguments: missing or empty 'field1'");
      expect(result).toContain("Example:");
    });

    it("should handle empty missing fields array", () => {
      const result = generateEnhancedErrorMessage([], "test_tool", {});
      expect(result).toBe("Invalid arguments");
    });

    it("should handle null missing fields", () => {
      const result = generateEnhancedErrorMessage(null, "test_tool", {});
      expect(result).toBe("Invalid arguments");
    });

    it("should handle schema without properties", () => {
      const schema = { type: "object", required: ["field1"] };
      const result = generateEnhancedErrorMessage(["field1"], "test_tool", schema);

      expect(result).toContain("Invalid arguments: missing or empty 'field1'");
      expect(result).toContain("Example:");
    });
  });
});
