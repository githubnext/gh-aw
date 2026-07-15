import { describe, it, expect } from "vitest";

describe("mcp_scripts_validation.cjs", () => {
  describe("validateRequiredFields", () => {
    it("should return empty array when no required fields", async () => {
      const { validateRequiredFields } = await import("./mcp_scripts_validation.cjs");

      const args = { foo: "bar" };
      const schema = { type: "object", properties: { foo: { type: "string" } } };

      const missing = validateRequiredFields(args, schema);

      expect(missing).toEqual([]);
    });

    it("should return empty array when all required fields are present", async () => {
      const { validateRequiredFields } = await import("./mcp_scripts_validation.cjs");

      const args = { name: "test", age: 25 };
      const schema = {
        type: "object",
        properties: { name: { type: "string" }, age: { type: "number" } },
        required: ["name", "age"],
      };

      const missing = validateRequiredFields(args, schema);

      expect(missing).toEqual([]);
    });

    it("should return missing field names when fields are undefined", async () => {
      const { validateRequiredFields } = await import("./mcp_scripts_validation.cjs");

      const args = { name: "test" };
      const schema = {
        type: "object",
        properties: { name: { type: "string" }, age: { type: "number" } },
        required: ["name", "age"],
      };

      const missing = validateRequiredFields(args, schema);

      expect(missing).toEqual(["age"]);
    });

    it("should return missing field names when fields are null", async () => {
      const { validateRequiredFields } = await import("./mcp_scripts_validation.cjs");

      const args = { name: "test", age: null };
      const schema = {
        type: "object",
        properties: { name: { type: "string" }, age: { type: "number" } },
        required: ["name", "age"],
      };

      const missing = validateRequiredFields(args, schema);

      expect(missing).toEqual(["age"]);
    });

    it("should return missing field names when string fields are empty", async () => {
      const { validateRequiredFields } = await import("./mcp_scripts_validation.cjs");

      const args = { name: "", age: 25 };
      const schema = {
        type: "object",
        properties: { name: { type: "string" }, age: { type: "number" } },
        required: ["name", "age"],
      };

      const missing = validateRequiredFields(args, schema);

      expect(missing).toEqual(["name"]);
    });

    it("should return missing field names when string fields are whitespace only", async () => {
      const { validateRequiredFields } = await import("./mcp_scripts_validation.cjs");

      const args = { name: "   ", age: 25 };
      const schema = {
        type: "object",
        properties: { name: { type: "string" }, age: { type: "number" } },
        required: ["name", "age"],
      };

      const missing = validateRequiredFields(args, schema);

      expect(missing).toEqual(["name"]);
    });

    it("should return multiple missing field names", async () => {
      const { validateRequiredFields } = await import("./mcp_scripts_validation.cjs");

      const args = {};
      const schema = {
        type: "object",
        properties: { name: { type: "string" }, age: { type: "number" }, email: { type: "string" } },
        required: ["name", "age", "email"],
      };

      const missing = validateRequiredFields(args, schema);

      expect(missing).toEqual(["name", "age", "email"]);
    });

    it("should handle schema without required array", async () => {
      const { validateRequiredFields } = await import("./mcp_scripts_validation.cjs");

      const args = { name: "test" };
      const schema = {
        type: "object",
        properties: { name: { type: "string" } },
      };

      const missing = validateRequiredFields(args, schema);

      expect(missing).toEqual([]);
    });

    it("should handle null schema", async () => {
      const { validateRequiredFields } = await import("./mcp_scripts_validation.cjs");

      const args = { name: "test" };
      const missing = validateRequiredFields(args, null);

      expect(missing).toEqual([]);
    });

    it("should handle undefined schema", async () => {
      const { validateRequiredFields } = await import("./mcp_scripts_validation.cjs");

      const args = { name: "test" };
      const missing = validateRequiredFields(args, undefined);

      expect(missing).toEqual([]);
    });
  });

  describe("validateStringInputLengths", () => {
    it("should return empty array when no string properties exceed limit", async () => {
      const { validateStringInputLengths } = await import("./mcp_scripts_validation.cjs");

      const args = { name: "hello", count: 42 };
      const schema = {
        type: "object",
        properties: {
          name: { type: "string" },
          count: { type: "number" },
        },
      };

      const violations = validateStringInputLengths(args, schema);

      expect(violations).toEqual([]);
    });

    it("should return violation when string property exceeds 10 KB limit", async () => {
      const { validateStringInputLengths, MAX_STRING_INPUT_BYTES } = await import("./mcp_scripts_validation.cjs");

      const oversizedValue = "a".repeat(MAX_STRING_INPUT_BYTES + 1);
      const args = { message: oversizedValue };
      const schema = {
        type: "object",
        properties: { message: { type: "string" } },
      };

      const violations = validateStringInputLengths(args, schema);

      expect(violations).toHaveLength(1);
      expect(violations[0].field).toBe("message");
      expect(violations[0].actualLength).toBe(MAX_STRING_INPUT_BYTES + 1);
      expect(violations[0].unit).toBe("bytes");
    });

    it("should not flag a string at exactly the 10 KB limit", async () => {
      const { validateStringInputLengths, MAX_STRING_INPUT_BYTES } = await import("./mcp_scripts_validation.cjs");

      const exactValue = "a".repeat(MAX_STRING_INPUT_BYTES);
      const args = { message: exactValue };
      const schema = {
        type: "object",
        properties: { message: { type: "string" } },
      };

      const violations = validateStringInputLengths(args, schema);

      expect(violations).toEqual([]);
    });

    it("should return multiple violations when multiple strings exceed the limit", async () => {
      const { validateStringInputLengths, MAX_STRING_INPUT_BYTES } = await import("./mcp_scripts_validation.cjs");

      const oversizedValue = "x".repeat(MAX_STRING_INPUT_BYTES + 100);
      const args = { a: oversizedValue, b: oversizedValue };
      const schema = {
        type: "object",
        properties: {
          a: { type: "string" },
          b: { type: "string" },
        },
      };

      const violations = validateStringInputLengths(args, schema);

      expect(violations).toHaveLength(2);
      expect(violations.map(v => v.field).sort()).toEqual(["a", "b"]);
    });

    it("should not flag non-string typed properties even if they contain string values", async () => {
      const { validateStringInputLengths, MAX_STRING_INPUT_BYTES } = await import("./mcp_scripts_validation.cjs");

      const oversizedValue = "z".repeat(MAX_STRING_INPUT_BYTES + 1);
      const args = { count: oversizedValue };
      const schema = {
        type: "object",
        properties: { count: { type: "number" } },
      };

      const violations = validateStringInputLengths(args, schema);

      expect(violations).toEqual([]);
    });

    it("should handle null or undefined schema gracefully", async () => {
      const { validateStringInputLengths } = await import("./mcp_scripts_validation.cjs");

      const args = { message: "hello" };

      expect(validateStringInputLengths(args, null)).toEqual([]);
      expect(validateStringInputLengths(args, undefined)).toEqual([]);
    });

    it("should respect a custom maxBytes override", async () => {
      const { validateStringInputLengths } = await import("./mcp_scripts_validation.cjs");

      const args = { label: "abcde" }; // 5 ASCII bytes
      const schema = {
        type: "object",
        properties: { label: { type: "string" } },
      };

      // Limit of 4 bytes — should flag the 5-byte value
      const violations = validateStringInputLengths(args, schema, 4);
      expect(violations).toHaveLength(1);
      expect(violations[0].field).toBe("label");

      // Limit of 5 bytes — exactly at limit, should not flag
      const noViolations = validateStringInputLengths(args, schema, 5);
      expect(noViolations).toEqual([]);
    });

    it("should measure byte length correctly for multi-byte UTF-8 characters", async () => {
      const { validateStringInputLengths } = await import("./mcp_scripts_validation.cjs");

      // Each emoji is 4 bytes in UTF-8
      const emojiString = "🚀".repeat(10); // 40 bytes
      const args = { emoji: emojiString };
      const schema = {
        type: "object",
        properties: { emoji: { type: "string" } },
      };

      // Limit of 39 bytes — should flag (10 emojis = 40 bytes)
      const violations = validateStringInputLengths(args, schema, 39);
      expect(violations).toHaveLength(1);

      // Limit of 40 bytes — at limit, should not flag
      const noViolations = validateStringInputLengths(args, schema, 40);
      expect(noViolations).toEqual([]);
    });

    it("should not flag missing (undefined) string values", async () => {
      const { validateStringInputLengths } = await import("./mcp_scripts_validation.cjs");

      const args = {};
      const schema = {
        type: "object",
        properties: { message: { type: "string" } },
      };

      const violations = validateStringInputLengths(args, schema);

      expect(violations).toEqual([]);
    });

    it("should enforce explicit maxLength for string fields", async () => {
      const { validateStringInputLengths, MAX_STRING_INPUT_BYTES } = await import("./mcp_scripts_validation.cjs");

      // Exceeds explicit maxLength
      const valueExceedingExplicitMax = "a".repeat(MAX_STRING_INPUT_BYTES + 1);
      const args = { body: valueExceedingExplicitMax };
      const schema = {
        type: "object",
        properties: { body: { type: "string", maxLength: MAX_STRING_INPUT_BYTES } },
      };

      const violations = validateStringInputLengths(args, schema);
      expect(violations).toHaveLength(1);
      expect(violations[0].field).toBe("body");
    });

    it("should still check string fields without maxLength when other fields have explicit maxLength", async () => {
      const { validateStringInputLengths, MAX_STRING_INPUT_BYTES } = await import("./mcp_scripts_validation.cjs");

      const oversizedValue = "a".repeat(MAX_STRING_INPUT_BYTES + 1);
      const args = { body: oversizedValue, title: oversizedValue };
      const schema = {
        type: "object",
        properties: {
          // body has explicit maxLength lower than value
          body: { type: "string", maxLength: MAX_STRING_INPUT_BYTES },
          // title has no maxLength — checked against default 10KB
          title: { type: "string" },
        },
      };

      const violations = validateStringInputLengths(args, schema);
      expect(violations).toHaveLength(2);
      expect(violations.map(v => v.field).sort()).toEqual(["body", "title"]);
    });

    it("should enforce explicit maxLength using character count", async () => {
      const { validateStringInputLengths } = await import("./mcp_scripts_validation.cjs");

      const args = { body: "🚀".repeat(3) };
      const schema = {
        type: "object",
        properties: { body: { type: "string", maxLength: 3 } },
      };

      const violations = validateStringInputLengths(args, schema);
      expect(violations).toEqual([]);
    });
  });

  describe("validateStringMinLengths", () => {
    it("should return empty array when all string fields meet minLength", async () => {
      const { validateStringMinLengths } = await import("./mcp_scripts_validation.cjs");

      const args = { body: "This body is long enough to satisfy the constraint" };
      const schema = {
        type: "object",
        properties: { body: { type: "string", minLength: 20 } },
      };

      expect(validateStringMinLengths(args, schema)).toEqual([]);
    });

    it("should return violation when field is shorter than minLength", async () => {
      const { validateStringMinLengths } = await import("./mcp_scripts_validation.cjs");

      const args = { body: "." };
      const schema = {
        type: "object",
        properties: { body: { type: "string", minLength: 20 } },
      };

      const violations = validateStringMinLengths(args, schema);
      expect(violations).toHaveLength(1);
      expect(violations[0].field).toBe("body");
      expect(violations[0].minLength).toBe(20);
      expect(violations[0].actualLength).toBe(1);
    });

    it("should trim whitespace before comparing against minLength", async () => {
      const { validateStringMinLengths } = await import("./mcp_scripts_validation.cjs");

      // 25 spaces — trims to 0 chars, below minLength 20
      const args = { body: " ".repeat(25) };
      const schema = {
        type: "object",
        properties: { body: { type: "string", minLength: 20 } },
      };

      const violations = validateStringMinLengths(args, schema);
      expect(violations).toHaveLength(1);
      expect(violations[0].actualLength).toBe(0);
    });

    it("should accept a value at exactly minLength", async () => {
      const { validateStringMinLengths } = await import("./mcp_scripts_validation.cjs");

      const args = { body: "12345678901234567890" }; // exactly 20 chars
      const schema = {
        type: "object",
        properties: { body: { type: "string", minLength: 20 } },
      };

      expect(validateStringMinLengths(args, schema)).toEqual([]);
    });

    it("should skip absent fields", async () => {
      const { validateStringMinLengths } = await import("./mcp_scripts_validation.cjs");

      const args = {};
      const schema = {
        type: "object",
        properties: { body: { type: "string", minLength: 20 } },
      };

      expect(validateStringMinLengths(args, schema)).toEqual([]);
    });

    it("should skip fields with no minLength in schema", async () => {
      const { validateStringMinLengths } = await import("./mcp_scripts_validation.cjs");

      const args = { title: "short" };
      const schema = {
        type: "object",
        properties: { title: { type: "string" } },
      };

      expect(validateStringMinLengths(args, schema)).toEqual([]);
    });

    it("should skip non-string typed fields", async () => {
      const { validateStringMinLengths } = await import("./mcp_scripts_validation.cjs");

      const args = { count: 3 };
      const schema = {
        type: "object",
        properties: { count: { type: "number", minLength: 5 } },
      };

      expect(validateStringMinLengths(args, schema)).toEqual([]);
    });

    it("should return violations for multiple short fields", async () => {
      const { validateStringMinLengths } = await import("./mcp_scripts_validation.cjs");

      const args = { title: "Hi", body: "." };
      const schema = {
        type: "object",
        properties: {
          title: { type: "string", minLength: 5 },
          body: { type: "string", minLength: 20 },
        },
      };

      const violations = validateStringMinLengths(args, schema);
      expect(violations).toHaveLength(2);
      const fields = violations.map(v => v.field);
      expect(fields).toContain("title");
      expect(fields).toContain("body");
    });

    it("should handle null/undefined inputSchema gracefully", async () => {
      const { validateStringMinLengths } = await import("./mcp_scripts_validation.cjs");

      const args = { body: "short" };

      expect(validateStringMinLengths(args, null)).toEqual([]);
      expect(validateStringMinLengths(args, undefined)).toEqual([]);
    });

    it("should accept minLength of zero (always passes)", async () => {
      const { validateStringMinLengths } = await import("./mcp_scripts_validation.cjs");

      const args = { body: "" };
      const schema = {
        type: "object",
        properties: { body: { type: "string", minLength: 0 } },
      };

      expect(validateStringMinLengths(args, schema)).toEqual([]);
    });
  });
});
