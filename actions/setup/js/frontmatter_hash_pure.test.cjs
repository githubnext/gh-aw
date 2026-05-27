// @ts-check
import { describe, it, expect } from "vitest";
const path = require("path");
const fs = require("fs");
const {
  computeFrontmatterHash,
  computeBodyHash,
  extractFrontmatterAndBody,
  extractImportsFromText,
  extractRelevantTemplateExpressions,
  marshalCanonicalJSON,
  marshalSorted,
  extractHashFromLockFile,
  extractBodyHashFromLockFile,
  normalizeFrontmatterText,
  parseBoolFromFrontmatter,
  defaultFileReader,
  createGitHubFileReader,
} = require("./frontmatter_hash_pure.cjs");

describe("frontmatter_hash_pure (text-based)", () => {
  describe("extractFrontmatterAndBody", () => {
    it("should extract frontmatter text and body", () => {
      const content = `---
engine: copilot
description: Test workflow
---

# Workflow Body

Test content here`;

      const result = extractFrontmatterAndBody(content);
      expect(result.frontmatterText).toContain("engine: copilot");
      expect(result.frontmatterText).toContain("description: Test workflow");
      expect(result.markdown).toContain("# Workflow Body");
    });

    it("should handle empty frontmatter", () => {
      const content = `# No frontmatter here`;
      const result = extractFrontmatterAndBody(content);
      expect(result.frontmatterText).toBe("");
      expect(result.markdown).toBe(content);
    });

    it("should handle frontmatter with imports", () => {
      const content = `---
engine: copilot
imports:
  - shared/test.md
  - shared/common.md
---

# Body`;

      const result = extractFrontmatterAndBody(content);
      expect(result.frontmatterText).toContain("imports:");
      expect(result.frontmatterText).toContain("- shared/test.md");
    });
  });

  describe("extractImportsFromText", () => {
    it("should extract imports from frontmatter text", () => {
      const frontmatterText = `engine: copilot
imports:
  - shared/test.md
  - shared/common.md
description: Test`;

      const result = extractImportsFromText(frontmatterText);
      expect(result).toEqual(["shared/test.md", "shared/common.md"]);
    });

    it("should handle no imports", () => {
      const frontmatterText = `engine: copilot
description: Test`;

      const result = extractImportsFromText(frontmatterText);
      expect(result).toEqual([]);
    });

    it("should handle imports with quotes", () => {
      const frontmatterText = `imports:
  - "shared/test.md"
  - 'shared/common.md'`;

      const result = extractImportsFromText(frontmatterText);
      expect(result).toEqual(["shared/test.md", "shared/common.md"]);
    });

    it("should stop at next top-level key", () => {
      const frontmatterText = `imports:
  - shared/test.md
engine: copilot`;

      const result = extractImportsFromText(frontmatterText);
      expect(result).toEqual(["shared/test.md"]);
    });
  });

  describe("extractRelevantTemplateExpressions", () => {
    it("should extract env expressions", () => {
      const markdown = "Use $" + "{{ env.MY_VAR }} here\nAnd also $" + "{{ env.OTHER }}";

      const result = extractRelevantTemplateExpressions(markdown);
      expect(result).toEqual(["$" + "{{ env.MY_VAR }}", "$" + "{{ env.OTHER }}"]);
    });

    it("should extract vars expressions", () => {
      const markdown = "Use $" + "{{ vars.CONFIG }} here";

      const result = extractRelevantTemplateExpressions(markdown);
      expect(result).toEqual(["$" + "{{ vars.CONFIG }}"]);
    });

    it("should ignore non-env/vars expressions", () => {
      const markdown = "Use $" + "{{ github.repository }} here\nBut include $" + "{{ env.TEST }}";

      const result = extractRelevantTemplateExpressions(markdown);
      expect(result).toEqual(["$" + "{{ env.TEST }}"]);
    });

    it("should deduplicate and sort expressions", () => {
      const markdown = "$" + "{{ env.B }} and $" + "{{ env.A }} and $" + "{{ env.B }}";

      const result = extractRelevantTemplateExpressions(markdown);
      expect(result).toEqual(["$" + "{{ env.A }}", "$" + "{{ env.B }}"]);
    });
  });

  describe("marshalCanonicalJSON", () => {
    it("should serialize with sorted keys", () => {
      const data = { c: 3, a: 1, b: 2 };
      const result = marshalCanonicalJSON(data);
      expect(result).toBe('{"a":1,"b":2,"c":3}');
    });

    it("should handle nested objects", () => {
      const data = { outer: { z: 26, a: 1 } };
      const result = marshalCanonicalJSON(data);
      expect(result).toBe('{"outer":{"a":1,"z":26}}');
    });

    it("should handle arrays", () => {
      const data = { items: [3, 1, 2] };
      const result = marshalCanonicalJSON(data);
      expect(result).toBe('{"items":[3,1,2]}');
    });

    it("should handle mixed types", () => {
      const data = {
        str: "value",
        num: 42,
        bool: true,
        nil: null,
        arr: [1, 2],
        obj: { x: 1 },
      };
      const result = marshalCanonicalJSON(data);
      expect(result).toBe('{"arr":[1,2],"bool":true,"nil":null,"num":42,"obj":{"x":1},"str":"value"}');
    });
  });

  describe("marshalSorted", () => {
    it("should handle primitives", () => {
      expect(marshalSorted("test")).toBe('"test"');
      expect(marshalSorted(42)).toBe("42");
      expect(marshalSorted(true)).toBe("true");
      expect(marshalSorted(null)).toBe("null");
    });

    it("should handle empty collections", () => {
      expect(marshalSorted([])).toBe("[]");
      expect(marshalSorted({})).toBe("{}");
    });
  });

  describe("extractHashFromLockFile", () => {
    it("should extract hash from old format lock file", () => {
      const content = `# frontmatter-hash: abc123def456

name: "Test Workflow"
on:
  push:`;

      const result = extractHashFromLockFile(content);
      expect(result).toBe("abc123def456");
    });

    it("should extract hash from new JSON metadata format", () => {
      const content = `# gh-aw-metadata: {"schema_version":"v1","frontmatter_hash":"abc123def456789"}

name: "Test Workflow"
on:
  push:`;

      const result = extractHashFromLockFile(content);
      expect(result).toBe("abc123def456789");
    });

    it("should extract hash from new JSON metadata format with additional fields", () => {
      const content = `# gh-aw-metadata: {"schema_version":"v1","frontmatter_hash":"xyz789abc123","stop_time":"2025-01-01T00:00:00Z","compiler_version":"0.1.0"}

name: "Test Workflow"
on:
  push:`;

      const result = extractHashFromLockFile(content);
      expect(result).toBe("xyz789abc123");
    });

    it("should handle new format with whitespace variations", () => {
      const content = `#  gh-aw-metadata:  {"schema_version":"v1","frontmatter_hash":"whitespace123"}

name: "Test Workflow"`;

      const result = extractHashFromLockFile(content);
      expect(result).toBe("whitespace123");
    });

    it("should fall back to old format if JSON parsing fails", () => {
      const content = `# gh-aw-metadata: {invalid json}
# frontmatter-hash: fallback123

name: "Test Workflow"`;

      const result = extractHashFromLockFile(content);
      expect(result).toBe("fallback123");
    });

    it("should prefer new format over old format when both present", () => {
      const content = `# gh-aw-metadata: {"schema_version":"v1","frontmatter_hash":"new123"}
# frontmatter-hash: old123

name: "Test Workflow"`;

      const result = extractHashFromLockFile(content);
      expect(result).toBe("new123");
    });

    it("should return empty string if no hash found", () => {
      const content = `name: "Test Workflow"
on:
  push:`;

      const result = extractHashFromLockFile(content);
      expect(result).toBe("");
    });

    it("should return empty string if metadata has no frontmatter_hash field", () => {
      const content = `# gh-aw-metadata: {"schema_version":"v1"}

name: "Test Workflow"`;

      const result = extractHashFromLockFile(content);
      expect(result).toBe("");
    });
  });

  describe("normalizeFrontmatterText", () => {
    it("should trim whitespace", () => {
      const text = `  engine: copilot  
  description: test  `;

      const result = normalizeFrontmatterText(text);
      expect(result).toBe("engine: copilot  \n  description: test");
    });

    it("should normalize line endings", () => {
      const text = "engine: copilot\r\ndescription: test\r\n";

      const result = normalizeFrontmatterText(text);
      expect(result).toBe("engine: copilot\ndescription: test");
    });
  });

  describe("parseBoolFromFrontmatter", () => {
    it("should return true when key is present with value true", () => {
      const frontmatter = "engine: copilot\ninlined-imports: true\ndescription: test";
      expect(parseBoolFromFrontmatter(frontmatter, "inlined-imports")).toBe(true);
    });

    it("should return false when key is present with value false", () => {
      const frontmatter = "engine: copilot\ninlined-imports: false\ndescription: test";
      expect(parseBoolFromFrontmatter(frontmatter, "inlined-imports")).toBe(false);
    });

    it("should return false when key is absent", () => {
      const frontmatter = "engine: copilot\ndescription: test";
      expect(parseBoolFromFrontmatter(frontmatter, "inlined-imports")).toBe(false);
    });

    it("should return false for empty frontmatter", () => {
      expect(parseBoolFromFrontmatter("", "inlined-imports")).toBe(false);
    });
  });

  describe("computeFrontmatterHash", () => {
    it.each([
      {
        id: "FH-TV-001",
        content: "---\n---\n\n# Empty Workflow\n",
        expectedHash: "4c8309afbcf816cd80c0824dce2b50047834b29e14b34b96953e88ae81048c46",
      },
      {
        id: "FH-TV-002",
        content: "---\nengine: copilot\ndescription: Test workflow\non:\n  schedule: daily\n---\n\n# Test Workflow\n",
        expectedHash: "b9def9907e3328e2e03e8c47c315723df39788f251627313b1a984bb61b9cbce",
      },
      {
        id: "FH-TV-003",
        content:
          "---\nengine: claude\ndescription: Complex workflow\ntracker-id: complex-test\ntimeout-minutes: 30\non:\n  schedule: daily\n  workflow_dispatch: true\npermissions:\n  contents: read\n  actions: read\ntools:\n  playwright:\n    version: v1.41.0\nlabels:\n  - test\n  - complex\nbots:\n  - copilot\n---\n\n# Complex Workflow\n",
        expectedHash: "8c63a05ef42cbfaff9be87a06257282cb4dcb952f71481d9d65ec3037003dbe8",
      },
    ])("should match Appendix A vector $id", async ({ content, expectedHash }) => {
      const testFile = path.join(__dirname, `test-workflow-${expectedHash}.md`);
      fs.writeFileSync(testFile, content, "utf8");
      try {
        const hash = await computeFrontmatterHash(testFile);
        expect(hash).toBe(expectedHash);
      } finally {
        if (fs.existsSync(testFile)) {
          fs.unlinkSync(testFile);
        }
      }
    });

    it("should compute hash for simple frontmatter", async () => {
      // Create a temporary test file
      const testFile = path.join(__dirname, "test-workflow-hash-simple.md");
      const content = "---\nengine: copilot\ndescription: Test workflow\n---\n\nUse $" + "{{ env.TEST }} here";

      fs.writeFileSync(testFile, content, "utf8");

      try {
        const hash = await computeFrontmatterHash(testFile);

        // Hash should be a 64-character hex string
        expect(hash).toMatch(/^[a-f0-9]{64}$/);

        // Computing again should produce the same hash (deterministic)
        const hash2 = await computeFrontmatterHash(testFile);
        expect(hash2).toBe(hash);
      } finally {
        if (fs.existsSync(testFile)) {
          fs.unlinkSync(testFile);
        }
      }
    });

    it("should include template expressions in hash", async () => {
      const testFile1 = path.join(__dirname, "test-workflow-hash-expr1.md");
      const testFile2 = path.join(__dirname, "test-workflow-hash-expr2.md");

      const content1 = "---\nengine: copilot\n---\n\nUse $" + "{{ env.VAR1 }}";
      const content2 = "---\nengine: copilot\n---\n\nUse $" + "{{ env.VAR2 }}";

      fs.writeFileSync(testFile1, content1, "utf8");
      fs.writeFileSync(testFile2, content2, "utf8");

      try {
        const hash1 = await computeFrontmatterHash(testFile1);
        const hash2 = await computeFrontmatterHash(testFile2);

        // Different expressions should produce different hashes
        expect(hash1).not.toBe(hash2);
      } finally {
        if (fs.existsSync(testFile1)) fs.unlinkSync(testFile1);
        if (fs.existsSync(testFile2)) fs.unlinkSync(testFile2);
      }
    });

    it("should work with custom file reader", async () => {
      const tmpDir = fs.mkdtempSync(path.join(require("os").tmpdir(), "frontmatter-hash-test-"));
      const testFile = path.join(tmpDir, "test.md");
      const content = "---\nengine: copilot\ndescription: Test\n---\n\nBody";

      // Create an in-memory file system mock
      const mockFileSystem = {
        [testFile]: content,
      };

      const customFileReader = async filePath => {
        if (mockFileSystem[filePath]) {
          return mockFileSystem[filePath];
        }
        throw new Error(`File not found: ${filePath}`);
      };

      try {
        const hash = await computeFrontmatterHash(testFile, { fileReader: customFileReader });
        expect(hash).toHaveLength(64); // SHA-256 is 64 hex chars
        expect(hash).toMatch(/^[0-9a-f]{64}$/);
      } finally {
        fs.rmSync(tmpDir, { recursive: true, force: true });
      }
    });

    it("should handle imports with custom file reader", async () => {
      const tmpDir = fs.mkdtempSync(path.join(require("os").tmpdir(), "frontmatter-hash-test-"));
      const mainFile = path.join(tmpDir, "main.md");
      const sharedDir = path.join(tmpDir, "shared");
      const importedFile = path.join(sharedDir, "imported.md");

      // Create an in-memory file system mock
      const mockFileSystem = {
        [mainFile]: "---\nengine: copilot\nimports:\n  - shared/imported.md\n---\n\nMain body",
        [importedFile]: "---\ntools:\n  bash: true\n---\n\nImported content",
      };

      const customFileReader = async filePath => {
        if (mockFileSystem[filePath]) {
          return mockFileSystem[filePath];
        }
        throw new Error(`File not found: ${filePath}`);
      };

      try {
        const hash = await computeFrontmatterHash(mainFile, { fileReader: customFileReader });
        expect(hash).toHaveLength(64);
        expect(hash).toMatch(/^[0-9a-f]{64}$/);
      } finally {
        fs.rmSync(tmpDir, { recursive: true, force: true });
      }
    });

    it("should include body-text in hash when inlined-imports is true", async () => {
      const tmpDir = fs.mkdtempSync(path.join(require("os").tmpdir(), "frontmatter-hash-test-"));
      const testFile = path.join(tmpDir, "test.md");

      const withBody = "---\nengine: copilot\ninlined-imports: true\n---\n\nBody content here";
      const withDifferentBody = "---\nengine: copilot\ninlined-imports: true\n---\n\nDifferent body content";
      const withoutFlag = "---\nengine: copilot\n---\n\nBody content here";

      const fileSystem = {};
      const makeReader = content => async () => content;

      try {
        const hashWithBody = await computeFrontmatterHash(testFile, { fileReader: makeReader(withBody) });
        const hashDifferentBody = await computeFrontmatterHash(testFile, { fileReader: makeReader(withDifferentBody) });
        const hashWithoutFlag = await computeFrontmatterHash(testFile, { fileReader: makeReader(withoutFlag) });

        // Different body content → different hash when inlined-imports: true
        expect(hashWithBody).not.toBe(hashDifferentBody);
        // Same body but without inlined-imports flag → different canonical data → different hash
        expect(hashWithBody).not.toBe(hashWithoutFlag);
      } finally {
        fs.rmSync(tmpDir, { recursive: true, force: true });
      }
    });

    it("should not include body-text in hash when inlined-imports is false", async () => {
      const tmpDir = fs.mkdtempSync(path.join(require("os").tmpdir(), "frontmatter-hash-test-"));
      const testFile = path.join(tmpDir, "test.md");

      const withBodyA = "---\nengine: copilot\n---\n\nBody content A";
      const withBodyB = "---\nengine: copilot\n---\n\nBody content B";

      const makeReader = content => async () => content;

      try {
        const hashA = await computeFrontmatterHash(testFile, { fileReader: makeReader(withBodyA) });
        const hashB = await computeFrontmatterHash(testFile, { fileReader: makeReader(withBodyB) });

        // Body changes should not affect hash when inlined-imports is not set
        expect(hashA).toBe(hashB);
      } finally {
        fs.rmSync(tmpDir, { recursive: true, force: true });
      }
    });

    it("should reject oversized normalized frontmatter input (FH-TV-NEG-001)", async () => {
      const testFile = path.join(__dirname, "test-workflow-hash-oversized.md");
      const oversizedValue = "a".repeat(1_048_577);
      const content = `---\ndescription: ${oversizedValue}\n---\n\n# Oversized Workflow`;

      fs.writeFileSync(testFile, content, "utf8");

      try {
        await expect(computeFrontmatterHash(testFile)).rejects.toThrow("frontmatter hash input exceeds 1048576 bytes after normalization");
      } finally {
        if (fs.existsSync(testFile)) {
          fs.unlinkSync(testFile);
        }
      }
    });
  });

  describe("extractBodyHashFromLockFile", () => {
    it("should return empty string when no body hash is present", () => {
      const content = `# gh-aw-metadata: {"schema_version":"v3","frontmatter_hash":"abc123"}
name: "Test Workflow"`;
      expect(extractBodyHashFromLockFile(content)).toBe("");
    });

    it("should extract body hash from JSON metadata format", () => {
      const content = `# gh-aw-metadata: {"schema_version":"v4","frontmatter_hash":"abc123","body_hash":"def456"}
name: "Test Workflow"`;
      expect(extractBodyHashFromLockFile(content)).toBe("def456");
    });

    it("should return empty string when no gh-aw-metadata comment is present", () => {
      const content = `# frontmatter-hash: abc123
name: "Test Workflow"`;
      expect(extractBodyHashFromLockFile(content)).toBe("");
    });

    it("should return empty string when metadata JSON is invalid", () => {
      const content = `# gh-aw-metadata: {invalid}
name: "Test Workflow"`;
      expect(extractBodyHashFromLockFile(content)).toBe("");
    });
  });

  describe("computeBodyHash", () => {
    it("should compute a 64-char hex SHA-256 hash", async () => {
      const tmpDir = fs.mkdtempSync(path.join(require("os").tmpdir(), "body-hash-test-"));
      const testFile = path.join(tmpDir, "test.md");
      const content = "---\nengine: copilot\n---\n\n# My Workflow\n\nDo some stuff.";
      const makeReader = () => async () => content;

      try {
        const hash = await computeBodyHash(testFile, { fileReader: makeReader() });
        expect(hash).toMatch(/^[a-f0-9]{64}$/);
      } finally {
        fs.rmSync(tmpDir, { recursive: true, force: true });
      }
    });

    it("should produce the same hash for identical body content", async () => {
      const tmpDir = fs.mkdtempSync(path.join(require("os").tmpdir(), "body-hash-test-"));
      const testFile = path.join(tmpDir, "test.md");
      const content = "---\nengine: copilot\n---\n\n# My Workflow\n\nDo some stuff.";
      const makeReader = () => async () => content;

      try {
        const hash1 = await computeBodyHash(testFile, { fileReader: makeReader() });
        const hash2 = await computeBodyHash(testFile, { fileReader: makeReader() });
        expect(hash1).toBe(hash2);
      } finally {
        fs.rmSync(tmpDir, { recursive: true, force: true });
      }
    });

    it("should produce different hashes when body content differs", async () => {
      const tmpDir = fs.mkdtempSync(path.join(require("os").tmpdir(), "body-hash-test-"));
      const testFile = path.join(tmpDir, "test.md");
      const contentA = "---\nengine: copilot\n---\n\n# Body A";
      const contentB = "---\nengine: copilot\n---\n\n# Body B";

      try {
        const hashA = await computeBodyHash(testFile, { fileReader: async () => contentA });
        const hashB = await computeBodyHash(testFile, { fileReader: async () => contentB });
        expect(hashA).not.toBe(hashB);
      } finally {
        fs.rmSync(tmpDir, { recursive: true, force: true });
      }
    });

    it("should produce the same hash when only frontmatter changes", async () => {
      const tmpDir = fs.mkdtempSync(path.join(require("os").tmpdir(), "body-hash-test-"));
      const testFile = path.join(tmpDir, "test.md");
      const contentA = "---\nengine: copilot\ndescription: version 1\n---\n\nSame body";
      const contentB = "---\nengine: copilot\ndescription: version 2\n---\n\nSame body";

      try {
        const hashA = await computeBodyHash(testFile, { fileReader: async () => contentA });
        const hashB = await computeBodyHash(testFile, { fileReader: async () => contentB });
        expect(hashA).toBe(hashB);
      } finally {
        fs.rmSync(tmpDir, { recursive: true, force: true });
      }
    });

    it("should include imported file bodies in the hash", async () => {
      const tmpDir = fs.mkdtempSync(path.join(require("os").tmpdir(), "body-hash-test-"));
      const mainFile = path.join(tmpDir, "main.md");
      const importedFile = path.join(tmpDir, "shared", "imported.md");

      const fileSystemBase = {
        [mainFile]: "---\nengine: copilot\nimports:\n  - shared/imported.md\n---\n\nMain body",
        [importedFile]: "---\ntools:\n  bash: true\n---\n\nImported body v1",
      };
      const fileSystemChanged = {
        [mainFile]: "---\nengine: copilot\nimports:\n  - shared/imported.md\n---\n\nMain body",
        [importedFile]: "---\ntools:\n  bash: true\n---\n\nImported body v2 (changed)",
      };

      const makeReader = fs_map => async filePath => {
        if (fs_map[filePath]) return fs_map[filePath];
        throw new Error(`File not found: ${filePath}`);
      };

      try {
        const hashBase = await computeBodyHash(mainFile, { fileReader: makeReader(fileSystemBase) });
        const hashChanged = await computeBodyHash(mainFile, { fileReader: makeReader(fileSystemChanged) });
        expect(hashBase).not.toBe(hashChanged);
      } finally {
        fs.rmSync(tmpDir, { recursive: true, force: true });
      }
    });

    it("should not be affected by changes to imported file frontmatter only", async () => {
      const tmpDir = fs.mkdtempSync(path.join(require("os").tmpdir(), "body-hash-test-"));
      const mainFile = path.join(tmpDir, "main.md");
      const importedFile = path.join(tmpDir, "shared", "imported.md");

      const fileSystemBase = {
        [mainFile]: "---\nengine: copilot\nimports:\n  - shared/imported.md\n---\n\nMain body",
        [importedFile]: "---\ntools:\n  bash: true\n---\n\nImported body",
      };
      const fileSystemFrontmatterChanged = {
        [mainFile]: "---\nengine: copilot\nimports:\n  - shared/imported.md\n---\n\nMain body",
        [importedFile]: "---\ntools:\n  bash: true\ndescription: changed frontmatter\n---\n\nImported body",
      };

      const makeReader = fs_map => async filePath => {
        if (fs_map[filePath]) return fs_map[filePath];
        throw new Error(`File not found: ${filePath}`);
      };

      try {
        const hashBase = await computeBodyHash(mainFile, { fileReader: makeReader(fileSystemBase) });
        const hashFrontmatterChanged = await computeBodyHash(mainFile, { fileReader: makeReader(fileSystemFrontmatterChanged) });
        // Only imported frontmatter changed, body is the same → hashes should match
        expect(hashBase).toBe(hashFrontmatterChanged);
      } finally {
        fs.rmSync(tmpDir, { recursive: true, force: true });
      }
    });
  });
});
