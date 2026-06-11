import { describe, it, expect, beforeEach, afterEach } from "vitest";
import fs from "fs";
import path from "path";

describe("writeLargeContentToFile", () => {
  const testDir = "/tmp/gh-aw/safeoutputs";

  beforeEach(() => {
    // Clean up test directory before each test
    if (fs.existsSync(testDir)) {
      fs.rmSync(testDir, { recursive: true });
    }
  });

  afterEach(() => {
    // Clean up test directory after each test
    if (fs.existsSync(testDir)) {
      fs.rmSync(testDir, { recursive: true });
    }
  });

  it("should create directory if it doesn't exist", async () => {
    const { writeLargeContentToFile } = await import("./write_large_content_to_file.cjs");

    expect(fs.existsSync(testDir)).toBe(false);

    const content = JSON.stringify({ test: "data" });
    writeLargeContentToFile(content);

    expect(fs.existsSync(testDir)).toBe(true);
  });

  it("should write content to file with hash-based filename", async () => {
    const { writeLargeContentToFile } = await import("./write_large_content_to_file.cjs");

    const content = JSON.stringify({ test: "data" });
    const result = writeLargeContentToFile(content);

    expect(result).toHaveProperty("filename");
    expect(result.filename).toMatch(/^[a-f0-9]{64}\.json$/);

    const filepath = path.join(testDir, result.filename);
    expect(fs.existsSync(filepath)).toBe(true);

    const written = fs.readFileSync(filepath, "utf8");
    expect(written).toBe(content);
  });

  it("should return schema description", async () => {
    const { writeLargeContentToFile } = await import("./write_large_content_to_file.cjs");

    const content = JSON.stringify({ id: 1, name: "test", value: 10 });
    const result = writeLargeContentToFile(content);

    expect(result).toHaveProperty("description");
    expect(result.description).toBe("{id, name, value}");
  });

  it("should use .json extension", async () => {
    const { writeLargeContentToFile } = await import("./write_large_content_to_file.cjs");

    const content = JSON.stringify([1, 2, 3]);
    const result = writeLargeContentToFile(content);

    expect(result.filename).toMatch(/\.json$/);
  });

  it("should generate consistent hash for same content", async () => {
    const { writeLargeContentToFile } = await import("./write_large_content_to_file.cjs");

    const content = JSON.stringify({ test: "data" });
    const result1 = writeLargeContentToFile(content);
    const result2 = writeLargeContentToFile(content);

    expect(result1.filename).toBe(result2.filename);
  });

  it("should generate different hash for different content", async () => {
    const { writeLargeContentToFile } = await import("./write_large_content_to_file.cjs");

    const content1 = JSON.stringify({ test: "data1" });
    const content2 = JSON.stringify({ test: "data2" });

    const result1 = writeLargeContentToFile(content1);
    const result2 = writeLargeContentToFile(content2);

    expect(result1.filename).not.toBe(result2.filename);
  });

  it("should handle arrays", async () => {
    const { writeLargeContentToFile } = await import("./write_large_content_to_file.cjs");

    const content = JSON.stringify([{ id: 1 }, { id: 2 }]);
    const result = writeLargeContentToFile(content);

    expect(result.description).toBe("[{id}] (2 items)");
  });

  it("should handle large content", async () => {
    const { writeLargeContentToFile } = await import("./write_large_content_to_file.cjs");

    const largeObj = {};
    for (let i = 0; i < 1000; i++) {
      largeObj[`key${i}`] = `value${i}`;
    }
    const content = JSON.stringify(largeObj);

    const result = writeLargeContentToFile(content);

    expect(result).toHaveProperty("filename");
    expect(result).toHaveProperty("description");

    const filepath = path.join(testDir, result.filename);
    const written = fs.readFileSync(filepath, "utf8");
    expect(written).toBe(content);
  });

  it("should handle non-JSON content gracefully", async () => {
    const { writeLargeContentToFile } = await import("./write_large_content_to_file.cjs");

    const content = "this is plain text, not JSON";
    const result = writeLargeContentToFile(content);

    expect(result.description).toBe("text content");
    const filepath = path.join(testDir, result.filename);
    expect(fs.readFileSync(filepath, "utf8")).toBe(content);
  });

  it("should handle empty object", async () => {
    const { writeLargeContentToFile } = await import("./write_large_content_to_file.cjs");

    const content = JSON.stringify({});
    const result = writeLargeContentToFile(content);

    expect(result.description).toBe("{}");
  });

  it("should handle empty array", async () => {
    const { writeLargeContentToFile } = await import("./write_large_content_to_file.cjs");

    const content = JSON.stringify([]);
    const result = writeLargeContentToFile(content);

    expect(result.description).toBe("[]");
  });

  it("should handle nested object (only top-level keys listed)", async () => {
    const { writeLargeContentToFile } = await import("./write_large_content_to_file.cjs");

    const content = JSON.stringify({ a: { b: 1 }, c: [1, 2] });
    const result = writeLargeContentToFile(content);

    expect(result.description).toBe("{a, c}");
  });

  it("should work when directory already exists", async () => {
    const { writeLargeContentToFile } = await import("./write_large_content_to_file.cjs");

    fs.mkdirSync(testDir, { recursive: true });
    expect(fs.existsSync(testDir)).toBe(true);

    const content = JSON.stringify({ already: "there" });
    expect(() => writeLargeContentToFile(content)).not.toThrow();
  });

  it("should handle JSON primitive (number)", async () => {
    const { writeLargeContentToFile } = await import("./write_large_content_to_file.cjs");

    const content = JSON.stringify(42);
    const result = writeLargeContentToFile(content);

    expect(result.description).toBe("number");
  });
});
