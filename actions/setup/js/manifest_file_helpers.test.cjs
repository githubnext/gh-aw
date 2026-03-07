// @ts-check
import { describe, it, expect } from "vitest";
import { createRequire } from "module";

const require = createRequire(import.meta.url);
const { extractFilenamesFromPatch, checkForManifestFiles } = require("./manifest_file_helpers.cjs");

describe("manifest_file_helpers", () => {
  describe("extractFilenamesFromPatch", () => {
    it("should return empty array for empty patch", () => {
      expect(extractFilenamesFromPatch("")).toEqual([]);
      expect(extractFilenamesFromPatch(null)).toEqual([]);
      expect(extractFilenamesFromPatch(undefined)).toEqual([]);
    });

    it("should extract a single filename", () => {
      const patch = `diff --git a/src/index.js b/src/index.js
index abc..def 100644
--- a/src/index.js
+++ b/src/index.js
@@ -1 +1 @@
-old
+new
`;
      expect(extractFilenamesFromPatch(patch)).toEqual(["index.js"]);
    });

    it("should extract basename only (no directory path)", () => {
      const patch = `diff --git a/path/to/deep/package.json b/path/to/deep/package.json
index abc..def 100644
--- a/path/to/deep/package.json
+++ b/path/to/deep/package.json
`;
      expect(extractFilenamesFromPatch(patch)).toEqual(["package.json"]);
    });

    it("should extract multiple filenames", () => {
      const patch = `diff --git a/src/index.js b/src/index.js
index abc..def 100644
diff --git a/package.json b/package.json
index abc..def 100644
diff --git a/README.md b/README.md
index abc..def 100644
`;
      const result = extractFilenamesFromPatch(patch);
      expect(result).toContain("index.js");
      expect(result).toContain("package.json");
      expect(result).toContain("README.md");
      expect(result).toHaveLength(3);
    });

    it("should deduplicate filenames", () => {
      const patch = `diff --git a/src/index.js b/src/index.js
index abc..def 100644
diff --git a/lib/index.js b/lib/index.js
index abc..def 100644
`;
      const result = extractFilenamesFromPatch(patch);
      expect(result).toEqual(["index.js"]);
    });

    it("should handle files at root (no directory)", () => {
      const patch = `diff --git a/package.json b/package.json
index abc..def 100644
`;
      expect(extractFilenamesFromPatch(patch)).toEqual(["package.json"]);
    });

    it("should capture both sides of a rename header", () => {
      // When package.json is renamed, the a/ side is the original manifest filename.
      // Both sides must be captured so the manifest check catches the rename.
      const patch = `diff --git a/package.json b/package.json.bak
similarity index 100%
rename from package.json
rename to package.json.bak
`;
      const result = extractFilenamesFromPatch(patch);
      expect(result).toContain("package.json");
      expect(result).toContain("package.json.bak");
    });

    it("should ignore dev/null sentinel in new-file diffs", () => {
      const patch = `diff --git a/dev/null b/src/new-file.js
new file mode 100644
index 0000000..abc1234
--- /dev/null
+++ b/src/new-file.js
@@ -0,0 +1 @@
+hello
`;
      const result = extractFilenamesFromPatch(patch);
      expect(result).toEqual(["new-file.js"]);
      expect(result).not.toContain("null");
    });

    it("should ignore dev/null sentinel in deleted-file diffs", () => {
      const patch = `diff --git a/src/old-file.js b/dev/null
deleted file mode 100644
index abc1234..0000000
--- a/src/old-file.js
+++ /dev/null
@@ -1 +0,0 @@
-hello
`;
      const result = extractFilenamesFromPatch(patch);
      expect(result).toEqual(["old-file.js"]);
      expect(result).not.toContain("null");
    });
  });

  describe("checkForManifestFiles", () => {
    it("should return false for empty patch", () => {
      const result = checkForManifestFiles("", ["package.json"]);
      expect(result.hasManifestFiles).toBe(false);
      expect(result.manifestFilesFound).toEqual([]);
    });

    it("should return false for empty manifest files list", () => {
      const patch = `diff --git a/package.json b/package.json\n`;
      const result = checkForManifestFiles(patch, []);
      expect(result.hasManifestFiles).toBe(false);
      expect(result.manifestFilesFound).toEqual([]);
    });

    it("should return false for null manifest files list", () => {
      const patch = `diff --git a/package.json b/package.json\n`;
      const result = checkForManifestFiles(patch, null);
      expect(result.hasManifestFiles).toBe(false);
      expect(result.manifestFilesFound).toEqual([]);
    });

    it("should detect package.json as a manifest file", () => {
      const patch = `diff --git a/package.json b/package.json
index abc..def 100644
--- a/package.json
+++ b/package.json
@@ -1 +1 @@
-{"name": "old"}
+{"name": "new"}
`;
      const result = checkForManifestFiles(patch, ["package.json", "go.mod"]);
      expect(result.hasManifestFiles).toBe(true);
      expect(result.manifestFilesFound).toContain("package.json");
    });

    it("should detect manifest files in nested directories", () => {
      const patch = `diff --git a/nested/path/go.mod b/nested/path/go.mod
index abc..def 100644
`;
      const result = checkForManifestFiles(patch, ["go.mod", "go.sum"]);
      expect(result.hasManifestFiles).toBe(true);
      expect(result.manifestFilesFound).toContain("go.mod");
    });

    it("should not detect non-manifest files", () => {
      const patch = `diff --git a/src/index.js b/src/index.js
index abc..def 100644
diff --git a/README.md b/README.md
index abc..def 100644
`;
      const result = checkForManifestFiles(patch, ["package.json", "go.mod", "requirements.txt"]);
      expect(result.hasManifestFiles).toBe(false);
      expect(result.manifestFilesFound).toEqual([]);
    });

    it("should return all manifest files found", () => {
      const patch = `diff --git a/package.json b/package.json
index abc..def 100644
diff --git a/package-lock.json b/package-lock.json
index abc..def 100644
diff --git a/src/index.js b/src/index.js
index abc..def 100644
`;
      const result = checkForManifestFiles(patch, ["package.json", "package-lock.json", "yarn.lock"]);
      expect(result.hasManifestFiles).toBe(true);
      expect(result.manifestFilesFound).toContain("package.json");
      expect(result.manifestFilesFound).toContain("package-lock.json");
      expect(result.manifestFilesFound).toHaveLength(2);
    });

    it("should match by filename only, not partial name", () => {
      const patch = `diff --git a/src/my-package.json b/src/my-package.json
index abc..def 100644
`;
      const result = checkForManifestFiles(patch, ["package.json"]);
      expect(result.hasManifestFiles).toBe(false);
    });

    it("should detect manifest file via the a/ side of a rename header", () => {
      // package.json is renamed to package.json.bak - the original name must be flagged
      const patch = `diff --git a/package.json b/package.json.bak
similarity index 100%
rename from package.json
rename to package.json.bak
`;
      const result = checkForManifestFiles(patch, ["package.json", "package-lock.json"]);
      expect(result.hasManifestFiles).toBe(true);
      expect(result.manifestFilesFound).toContain("package.json");
    });
  });

  describe("extractPathsFromPatch", () => {
    const { extractPathsFromPatch } = require("./manifest_file_helpers.cjs");

    it("should return empty array for empty patch", () => {
      expect(extractPathsFromPatch("")).toEqual([]);
      expect(extractPathsFromPatch(null)).toEqual([]);
    });

    it("should return full paths not just basenames", () => {
      const patch = `diff --git a/.github/workflows/ci.yml b/.github/workflows/ci.yml
index abc..def 100644
`;
      const result = extractPathsFromPatch(patch);
      expect(result).toContain(".github/workflows/ci.yml");
      expect(result).not.toContain("ci.yml"); // basenames not returned
    });

    it("should include both a/ and b/ paths for renames", () => {
      const patch = `diff --git a/.github/old.yml b/.github/new.yml
similarity index 100%
rename from .github/old.yml
rename to .github/new.yml
`;
      const result = extractPathsFromPatch(patch);
      expect(result).toContain(".github/old.yml");
      expect(result).toContain(".github/new.yml");
    });

    it("should skip dev/null sentinel", () => {
      const patch = `diff --git a/dev/null b/.github/workflows/new.yml
new file mode 100644
index 0000000..abc
`;
      const result = extractPathsFromPatch(patch);
      expect(result).toContain(".github/workflows/new.yml");
      expect(result).not.toContain("dev/null");
    });
  });

  describe("checkForProtectedPaths", () => {
    const { checkForProtectedPaths } = require("./manifest_file_helpers.cjs");

    it("should return false for empty patch", () => {
      const result = checkForProtectedPaths("", [".github/"]);
      expect(result.hasProtectedPaths).toBe(false);
      expect(result.protectedPathsFound).toEqual([]);
    });

    it("should return false for empty prefixes list", () => {
      const patch = `diff --git a/.github/workflows/ci.yml b/.github/workflows/ci.yml\n`;
      const result = checkForProtectedPaths(patch, []);
      expect(result.hasProtectedPaths).toBe(false);
    });

    it("should detect .github/ files", () => {
      const patch = `diff --git a/.github/workflows/ci.yml b/.github/workflows/ci.yml
index abc..def 100644
`;
      const result = checkForProtectedPaths(patch, [".github/"]);
      expect(result.hasProtectedPaths).toBe(true);
      expect(result.protectedPathsFound).toContain(".github/workflows/ci.yml");
    });

    it("should not flag files outside protected path prefix", () => {
      const patch = `diff --git a/src/ci.yml b/src/ci.yml
index abc..def 100644
`;
      const result = checkForProtectedPaths(patch, [".github/"]);
      expect(result.hasProtectedPaths).toBe(false);
    });

    it("should detect AGENTS.md via basename check (not path prefix)", () => {
      // AGENTS.md is checked via checkForManifestFiles (basename), not path prefix
      const patch = `diff --git a/AGENTS.md b/AGENTS.md
index abc..def 100644
`;
      const basenameResult = checkForManifestFiles(patch, ["AGENTS.md"]);
      expect(basenameResult.hasManifestFiles).toBe(true);
    });
  });
});
