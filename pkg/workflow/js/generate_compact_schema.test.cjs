import { describe, it, expect } from "vitest";

describe("generateCompactSchema", () => {
  it("should handle empty arrays", async () => {
    const { generateCompactSchema } = await import("./generate_compact_schema.cjs");

    expect(generateCompactSchema("[]")).toBe("[]");
  });

  it("should describe array of objects", async () => {
    const { generateCompactSchema } = await import("./generate_compact_schema.cjs");

    const json = JSON.stringify([
      { id: 1, name: "test", value: 10 },
      { id: 2, name: "test2", value: 20 },
    ]);
    expect(generateCompactSchema(json)).toBe("[{id, name, value}] (2 items)");
  });

  it("should describe array of primitives", async () => {
    const { generateCompactSchema } = await import("./generate_compact_schema.cjs");

    expect(generateCompactSchema("[1, 2, 3]")).toBe("[number] (3 items)");
    expect(generateCompactSchema('["a", "b", "c"]')).toBe("[string] (3 items)");
    expect(generateCompactSchema("[true, false]")).toBe("[boolean] (2 items)");
  });

  it("should describe objects with few keys", async () => {
    const { generateCompactSchema } = await import("./generate_compact_schema.cjs");

    const json = JSON.stringify({ id: 1, name: "test", value: 10 });
    expect(generateCompactSchema(json)).toBe("{id, name, value}");
  });

  it("should describe objects with many keys", async () => {
    const { generateCompactSchema } = await import("./generate_compact_schema.cjs");

    const obj = {};
    for (let i = 0; i < 15; i++) {
      obj[`key${i}`] = i;
    }
    const json = JSON.stringify(obj);
    const schema = generateCompactSchema(json);
    expect(schema).toMatch(/^\{key0, key1, .+\.\.\.\} \(15 keys\)$/);
  });

  it("should handle primitives", async () => {
    const { generateCompactSchema } = await import("./generate_compact_schema.cjs");

    expect(generateCompactSchema("123")).toBe("number");
    expect(generateCompactSchema('"hello"')).toBe("string");
    expect(generateCompactSchema("true")).toBe("boolean");
    expect(generateCompactSchema("null")).toBe("object"); // JSON.parse(null) is object
  });

  it("should handle invalid JSON", async () => {
    const { generateCompactSchema } = await import("./generate_compact_schema.cjs");

    expect(generateCompactSchema("not valid json")).toBe("text content");
    expect(generateCompactSchema("")).toBe("text content");
    expect(generateCompactSchema("{invalid}")).toBe("text content");
  });

  it("should handle nested structures", async () => {
    const { generateCompactSchema } = await import("./generate_compact_schema.cjs");

    const json = JSON.stringify({
      users: [{ id: 1, name: "test" }],
      meta: { total: 1 },
    });
    expect(generateCompactSchema(json)).toBe("{users, meta}");
  });

  it("should handle single item arrays", async () => {
    const { generateCompactSchema } = await import("./generate_compact_schema.cjs");

    expect(generateCompactSchema("[1]")).toBe("[number] (1 items)");
    expect(generateCompactSchema('[{"id": 1}]')).toBe("[{id}] (1 items)");
  });

  it("should handle empty objects", async () => {
    const { generateCompactSchema } = await import("./generate_compact_schema.cjs");

    expect(generateCompactSchema("{}")).toBe("{}");
  });
});
