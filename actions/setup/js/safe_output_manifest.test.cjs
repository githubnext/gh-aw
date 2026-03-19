// @ts-check
import { describe, it, expect, beforeEach, afterEach } from "vitest";
import fs from "fs";
import path from "path";
import { MANIFEST_FILE_PATH, CREATE_ITEM_TYPES, NOT_LOGGED_TYPES, createManifestLogger, ensureManifestExists, extractCreatedItemFromResult } from "./safe_output_manifest.cjs";

describe("safe_output_manifest", () => {
  let testManifestFile;

  beforeEach(() => {
    const testId = Math.random().toString(36).substring(7);
    testManifestFile = `/tmp/test-safe-output-manifest-${testId}/items.jsonl`;
    fs.mkdirSync(path.dirname(testManifestFile), { recursive: true });
  });

  afterEach(() => {
    try {
      const testDir = path.dirname(testManifestFile);
      if (fs.existsSync(testDir)) {
        fs.rmSync(testDir, { recursive: true, force: true });
      }
    } catch (_err) {
      // Ignore cleanup errors
    }
  });

  describe("MANIFEST_FILE_PATH", () => {
    it("should be the expected default path", () => {
      expect(MANIFEST_FILE_PATH).toBe("/tmp/gh-aw/safe-output-items.jsonl");
    });
  });

  describe("CREATE_ITEM_TYPES", () => {
    it("should include expected create types", () => {
      expect(CREATE_ITEM_TYPES.has("create_issue")).toBe(true);
      expect(CREATE_ITEM_TYPES.has("add_comment")).toBe(true);
      expect(CREATE_ITEM_TYPES.has("create_discussion")).toBe(true);
      expect(CREATE_ITEM_TYPES.has("create_pull_request")).toBe(true);
      expect(CREATE_ITEM_TYPES.has("create_project")).toBe(true);
    });

    it("should not include update/delete types", () => {
      expect(CREATE_ITEM_TYPES.has("update_issue")).toBe(false);
      expect(CREATE_ITEM_TYPES.has("close_issue")).toBe(false);
      expect(CREATE_ITEM_TYPES.has("add_labels")).toBe(false);
      expect(CREATE_ITEM_TYPES.has("noop")).toBe(false);
    });
  });

  describe("NOT_LOGGED_TYPES", () => {
    it("should contain only noop and internal meta types", () => {
      expect(NOT_LOGGED_TYPES.has("noop")).toBe(true);
      expect(NOT_LOGGED_TYPES.has("missing_tool")).toBe(true);
      expect(NOT_LOGGED_TYPES.has("missing_data")).toBe(true);
    });

    it("should not contain any handler or modification types (all are logged by default)", () => {
      expect(NOT_LOGGED_TYPES.has("create_issue")).toBe(false);
      expect(NOT_LOGGED_TYPES.has("add_labels")).toBe(false);
      expect(NOT_LOGGED_TYPES.has("close_issue")).toBe(false);
      expect(NOT_LOGGED_TYPES.has("update_issue")).toBe(false);
    });

    it("should not contain CREATE_ITEM_TYPES (they are logged)", () => {
      for (const type of CREATE_ITEM_TYPES) {
        expect(NOT_LOGGED_TYPES.has(type)).toBe(false);
      }
    });

    it("should allow custom safe job types to be logged automatically (not in exclusion list)", () => {
      // Custom safe job types are never added to NOT_LOGGED_TYPES so they are always logged
      expect(NOT_LOGGED_TYPES.has("my_custom_job_type")).toBe(false);
      expect(NOT_LOGGED_TYPES.has("deploy_to_staging")).toBe(false);
    });
  });

  describe("createManifestLogger", () => {
    it("should append a JSONL entry when called with a url", () => {
      const log = createManifestLogger(testManifestFile);
      log({ type: "create_issue", url: "https://github.com/owner/repo/issues/1", number: 1, repo: "owner/repo", temporaryId: "aw_abc123" });

      const content = fs.readFileSync(testManifestFile, "utf8");
      const lines = content.trim().split("\n");
      expect(lines).toHaveLength(1);

      const entry = JSON.parse(lines[0]);
      expect(entry.type).toBe("create_issue");
      expect(entry.url).toBe("https://github.com/owner/repo/issues/1");
      expect(entry.number).toBe(1);
      expect(entry.repo).toBe("owner/repo");
      expect(entry.temporaryId).toBe("aw_abc123");
      expect(entry.timestamp).toBeDefined();
      // timestamp should be a valid ISO 8601 string (Date.parse returns NaN for invalid dates)
      expect(Date.parse(entry.timestamp)).not.toBeNaN();
    });

    it("should append a JSONL entry for an item without a url (modification type)", () => {
      const log = createManifestLogger(testManifestFile);
      log({ type: "add_labels", number: 20875 });

      const content = fs.readFileSync(testManifestFile, "utf8");
      const lines = content.trim().split("\n");
      expect(lines).toHaveLength(1);

      const entry = JSON.parse(lines[0]);
      expect(entry.type).toBe("add_labels");
      expect(entry.url).toBeUndefined();
      expect(entry.number).toBe(20875);
      expect(entry.timestamp).toBeDefined();
    });

    it("should skip null/undefined items", () => {
      const log = createManifestLogger(testManifestFile);
      log(null);
      log(undefined);

      // File is created by createManifestLogger() immediately, but should be empty
      expect(fs.existsSync(testManifestFile)).toBe(true);
      expect(fs.readFileSync(testManifestFile, "utf8")).toBe("");
    });

    it("should omit optional fields when not provided", () => {
      const log = createManifestLogger(testManifestFile);
      log({ type: "create_discussion", url: "https://github.com/owner/repo/discussions/5" });

      const content = fs.readFileSync(testManifestFile, "utf8");
      const entry = JSON.parse(content.trim());
      expect(entry.type).toBe("create_discussion");
      expect(entry.url).toBe("https://github.com/owner/repo/discussions/5");
      expect(entry.number).toBeUndefined();
      expect(entry.repo).toBeUndefined();
      expect(entry.temporaryId).toBeUndefined();
    });

    it("should append multiple entries in JSONL format (one per line)", () => {
      const log = createManifestLogger(testManifestFile);
      log({ type: "create_issue", url: "https://github.com/owner/repo/issues/1", number: 1, repo: "owner/repo" });
      log({ type: "create_issue", url: "https://github.com/owner/repo/issues/2", number: 2, repo: "owner/repo" });
      log({ type: "add_comment", url: "https://github.com/owner/repo/issues/1#issuecomment-123", repo: "owner/repo" });

      const content = fs.readFileSync(testManifestFile, "utf8");
      const lines = content.trim().split("\n");
      expect(lines).toHaveLength(3);

      const entries = lines.map(l => JSON.parse(l));
      expect(entries[0].type).toBe("create_issue");
      expect(entries[0].number).toBe(1);
      expect(entries[1].number).toBe(2);
      expect(entries[2].type).toBe("add_comment");
    });

    it("should write single-line JSON (no formatting) per entry", () => {
      const log = createManifestLogger(testManifestFile);
      log({ type: "create_issue", url: "https://github.com/owner/repo/issues/1", number: 1, repo: "owner/repo" });

      const content = fs.readFileSync(testManifestFile, "utf8");
      const lines = content.split("\n");
      // Two lines: the JSON entry + trailing newline
      expect(lines).toHaveLength(2);
      expect(lines[1]).toBe("");
      // First line should be parseable as a single JSON object
      expect(() => JSON.parse(lines[0])).not.toThrow();
    });

    it("should throw when the manifest file cannot be written", () => {
      // Create a directory where the file should be to force a write error
      fs.mkdirSync(testManifestFile, { recursive: true });

      const log = createManifestLogger(testManifestFile);
      expect(() => log({ type: "create_issue", url: "https://github.com/owner/repo/issues/1" })).toThrow("Failed to write to manifest file");

      // Clean up
      fs.rmSync(testManifestFile, { recursive: true, force: true });
    });
  });

  describe("ensureManifestExists", () => {
    it("should create an empty file if the manifest does not exist", () => {
      expect(fs.existsSync(testManifestFile)).toBe(false);
      ensureManifestExists(testManifestFile);
      expect(fs.existsSync(testManifestFile)).toBe(true);
      expect(fs.readFileSync(testManifestFile, "utf8")).toBe("");
    });

    it("should not overwrite an existing file", () => {
      const content = '{"type":"create_issue","url":"https://github.com/o/r/issues/1","timestamp":"2024-01-01T00:00:00.000Z"}\n';
      fs.writeFileSync(testManifestFile, content);

      ensureManifestExists(testManifestFile);

      expect(fs.readFileSync(testManifestFile, "utf8")).toBe(content);
    });

    it("should throw when the file cannot be created", () => {
      // Use a path under a non-existent directory without creating it
      const badFile = "/tmp/nonexistent-dir-xyz/items.jsonl";
      expect(() => ensureManifestExists(badFile)).toThrow("Failed to create manifest file");
    });
  });

  describe("extractCreatedItemFromResult", () => {
    it("should extract item from create_issue result", () => {
      const result = { success: true, repo: "owner/repo", number: 42, url: "https://github.com/owner/repo/issues/42", temporaryId: "aw_def456" };
      const item = extractCreatedItemFromResult("create_issue", result);
      expect(item).toEqual({
        type: "create_issue",
        url: "https://github.com/owner/repo/issues/42",
        number: 42,
        repo: "owner/repo",
        temporaryId: "aw_def456",
      });
    });

    it("should extract item from create_project result using projectUrl", () => {
      const result = { success: true, projectUrl: "https://github.com/orgs/owner/projects/5", temporaryId: "aw_proj01" };
      const item = extractCreatedItemFromResult("create_project", result);
      expect(item).not.toBeNull();
      expect(item.url).toBe("https://github.com/orgs/owner/projects/5");
      expect(item.type).toBe("create_project");
    });

    it("should extract item from add_comment result", () => {
      const result = { success: true, commentId: 999, url: "https://github.com/owner/repo/issues/1#issuecomment-999", repo: "owner/repo", itemNumber: 1 };
      const item = extractCreatedItemFromResult("add_comment", result);
      expect(item).not.toBeNull();
      expect(item.url).toBe("https://github.com/owner/repo/issues/1#issuecomment-999");
      expect(item.type).toBe("add_comment");
    });

    it("should return null for excluded types (noop and internal meta types)", () => {
      const result = { success: true, url: "https://github.com/owner/repo/issues/1" };
      expect(extractCreatedItemFromResult("noop", result)).toBeNull();
      expect(extractCreatedItemFromResult("missing_tool", result)).toBeNull();
      expect(extractCreatedItemFromResult("missing_data", result)).toBeNull();
    });

    it("should extract item from custom safe job type (generic: any type not excluded is logged)", () => {
      const result = { success: true, number: 42 };
      const item = extractCreatedItemFromResult("my_custom_job_type", result);
      expect(item).not.toBeNull();
      expect(item.type).toBe("my_custom_job_type");
      expect(item.number).toBe(42);
    });

    it("should extract item from add_labels result (modification type without url)", () => {
      const result = { success: true, number: 20875, labelsAdded: ["bug", "cli"], contextType: "issue" };
      const item = extractCreatedItemFromResult("add_labels", result);
      expect(item).not.toBeNull();
      expect(item.type).toBe("add_labels");
      expect(item.url).toBeUndefined();
      expect(item.number).toBe(20875);
    });

    it("should extract item from close_issue result (modification type with url)", () => {
      const result = { success: true, number: 123, url: "https://github.com/owner/repo/issues/123", title: "Test" };
      const item = extractCreatedItemFromResult("close_issue", result);
      expect(item).not.toBeNull();
      expect(item.type).toBe("close_issue");
      expect(item.url).toBe("https://github.com/owner/repo/issues/123");
      expect(item.number).toBe(123);
    });

    it("should return null for staged results (no item actually modified)", () => {
      // Staged results have staged: true — nothing was really changed
      const stagedResult = { success: true, staged: true, previewInfo: { repo: "owner/repo", title: "Test" } };
      expect(extractCreatedItemFromResult("create_issue", stagedResult)).toBeNull();
      expect(extractCreatedItemFromResult("add_comment", stagedResult)).toBeNull();
      expect(extractCreatedItemFromResult("add_labels", stagedResult)).toBeNull();
    });

    it("should return null for staged results even if url is somehow present", () => {
      // Defensive: staged flag takes precedence over any URL
      const stagedResultWithUrl = { success: true, staged: true, url: "https://github.com/owner/repo/issues/1" };
      expect(extractCreatedItemFromResult("create_issue", stagedResultWithUrl)).toBeNull();
    });

    it("should return item without url when result has no URL (for logged types)", () => {
      const result = { success: true, repo: "owner/repo", number: 1 };
      const item = extractCreatedItemFromResult("create_issue", result);
      expect(item).not.toBeNull();
      expect(item.url).toBeUndefined();
      expect(item.number).toBe(1);
    });

    it("should return null for null/undefined result", () => {
      expect(extractCreatedItemFromResult("create_issue", null)).toBeNull();
      expect(extractCreatedItemFromResult("create_issue", undefined)).toBeNull();
    });

    it("should omit optional fields when not present in result", () => {
      const result = { success: true, url: "https://github.com/owner/repo/issues/1" };
      const item = extractCreatedItemFromResult("create_issue", result);
      expect(item).not.toBeNull();
      expect(item.number).toBeUndefined();
      expect(item.repo).toBeUndefined();
      expect(item.temporaryId).toBeUndefined();
    });
  });
});
