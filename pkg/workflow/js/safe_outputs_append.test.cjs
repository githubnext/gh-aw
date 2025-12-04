// @ts-check
import { describe, it, expect, beforeEach, afterEach } from "vitest";
import fs from "fs";
import path from "path";
import { createAppendFunction } from "./safe_outputs_append.cjs";

describe("safe_outputs_append", () => {
  let testOutputFile;
  let appendFn;

  beforeEach(() => {
    // Use unique path for each test
    const testId = Math.random().toString(36).substring(7);
    testOutputFile = `/tmp/test-safe-outputs-append-${testId}/outputs.jsonl`;

    const outputDir = path.dirname(testOutputFile);
    fs.mkdirSync(outputDir, { recursive: true });

    appendFn = createAppendFunction(testOutputFile);
  });

  afterEach(() => {
    // Clean up test files
    try {
      if (fs.existsSync(testOutputFile)) {
        fs.unlinkSync(testOutputFile);
      }
      const testDir = path.dirname(testOutputFile);
      if (fs.existsSync(testDir)) {
        fs.rmdirSync(testDir, { recursive: true });
      }
    } catch (error) {
      // Ignore cleanup errors
    }
  });

  describe("createAppendFunction", () => {
    it("should append entry to output file", () => {
      const entry = {
        type: "test-type",
        data: "test-data",
      };

      appendFn(entry);

      const content = fs.readFileSync(testOutputFile, "utf8");
      const lines = content.trim().split("\n");
      expect(lines).toHaveLength(1);

      const parsedEntry = JSON.parse(lines[0]);
      expect(parsedEntry).toEqual({
        type: "test_type",
        data: "test-data",
      });
    });

    it("should normalize dashes to underscores in type", () => {
      const entry = {
        type: "create-pull-request",
        title: "Test PR",
      };

      appendFn(entry);

      const content = fs.readFileSync(testOutputFile, "utf8");
      const parsedEntry = JSON.parse(content.trim());
      expect(parsedEntry.type).toBe("create_pull_request");
    });

    it("should append multiple entries", () => {
      const entries = [
        { type: "type-1", data: "data-1" },
        { type: "type-2", data: "data-2" },
        { type: "type-3", data: "data-3" },
      ];

      entries.forEach(entry => appendFn(entry));

      const content = fs.readFileSync(testOutputFile, "utf8");
      const lines = content.trim().split("\n");
      expect(lines).toHaveLength(3);

      const parsedEntries = lines.map(line => JSON.parse(line));
      expect(parsedEntries[0].type).toBe("type_1");
      expect(parsedEntries[1].type).toBe("type_2");
      expect(parsedEntries[2].type).toBe("type_3");
    });

    it("should preserve entry fields", () => {
      const entry = {
        type: "upload-asset",
        path: "/test/file.png",
        sha: "abc123",
        size: 1024,
        url: "https://example.com/file.png",
      };

      appendFn(entry);

      const content = fs.readFileSync(testOutputFile, "utf8");
      const parsedEntry = JSON.parse(content.trim());
      expect(parsedEntry).toEqual({
        type: "upload_asset",
        path: "/test/file.png",
        sha: "abc123",
        size: 1024,
        url: "https://example.com/file.png",
      });
    });

    it("should throw error when output file is not configured", () => {
      const badAppendFn = createAppendFunction("");
      const entry = { type: "test", data: "test" };

      expect(() => badAppendFn(entry)).toThrow("No output file configured");
    });

    it("should throw error when write fails", () => {
      // Create a directory where the file should be (will cause write to fail)
      fs.mkdirSync(testOutputFile, { recursive: true });

      const entry = { type: "test", data: "test" };

      expect(() => appendFn(entry)).toThrow("Failed to write to output file");

      // Clean up the directory
      fs.rmdirSync(testOutputFile, { recursive: true });
    });

    it("should handle entries with nested objects", () => {
      const entry = {
        type: "complex-type",
        nested: {
          field1: "value1",
          field2: { subfield: "value2" },
        },
      };

      appendFn(entry);

      const content = fs.readFileSync(testOutputFile, "utf8");
      const parsedEntry = JSON.parse(content.trim());
      expect(parsedEntry.nested).toEqual({
        field1: "value1",
        field2: { subfield: "value2" },
      });
    });

    it("should handle entries with arrays", () => {
      const entry = {
        type: "array-type",
        items: ["item1", "item2", "item3"],
      };

      appendFn(entry);

      const content = fs.readFileSync(testOutputFile, "utf8");
      const parsedEntry = JSON.parse(content.trim());
      expect(parsedEntry.items).toEqual(["item1", "item2", "item3"]);
    });

    it("should create file if it doesn't exist", () => {
      expect(fs.existsSync(testOutputFile)).toBe(false);

      const entry = { type: "test", data: "test" };
      appendFn(entry);

      expect(fs.existsSync(testOutputFile)).toBe(true);
    });

    it("should append to existing file", () => {
      // Write initial entry
      fs.writeFileSync(testOutputFile, JSON.stringify({ type: "initial", data: "initial" }) + "\n");

      // Append new entry
      const entry = { type: "new", data: "new" };
      appendFn(entry);

      const content = fs.readFileSync(testOutputFile, "utf8");
      const lines = content.trim().split("\n");
      expect(lines).toHaveLength(2);
    });
  });
});
