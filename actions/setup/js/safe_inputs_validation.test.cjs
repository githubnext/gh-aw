import { describe, it, expect } from "vitest";

describe("safe_inputs_validation.cjs", () => {
  describe("validateRequiredFields", () => {
    it("should return empty array when no required fields", async () => {
      const { validateRequiredFields } = await import("./safe_inputs_validation.cjs");

      const args = { foo: "bar" };
      const schema = { type: "object", properties: { foo: { type: "string" } } };

      const missing = validateRequiredFields(args, schema);

      expect(missing).toEqual([]);
    });

    it("should return empty array when all required fields are present", async () => {
      const { validateRequiredFields } = await import("./safe_inputs_validation.cjs");

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
      const { validateRequiredFields } = await import("./safe_inputs_validation.cjs");

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
      const { validateRequiredFields } = await import("./safe_inputs_validation.cjs");

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
      const { validateRequiredFields } = await import("./safe_inputs_validation.cjs");

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
      const { validateRequiredFields } = await import("./safe_inputs_validation.cjs");

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
      const { validateRequiredFields } = await import("./safe_inputs_validation.cjs");

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
      const { validateRequiredFields } = await import("./safe_inputs_validation.cjs");

      const args = { name: "test" };
      const schema = {
        type: "object",
        properties: { name: { type: "string" } },
      };

      const missing = validateRequiredFields(args, schema);

      expect(missing).toEqual([]);
    });

    it("should handle null schema", async () => {
      const { validateRequiredFields } = await import("./safe_inputs_validation.cjs");

      const args = { name: "test" };
      const missing = validateRequiredFields(args, null);

      expect(missing).toEqual([]);
    });

    it("should handle undefined schema", async () => {
      const { validateRequiredFields } = await import("./safe_inputs_validation.cjs");

      const args = { name: "test" };
      const missing = validateRequiredFields(args, undefined);

      expect(missing).toEqual([]);
    });
  });
});
