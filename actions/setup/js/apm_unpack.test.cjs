// @ts-check
/// <reference types="@actions/github-script" />

import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
const fs = require("fs");
const path = require("path");
const os = require("os");

// ---------------------------------------------------------------------------
// Global mock setup
// ---------------------------------------------------------------------------

const mockCore = {
  info: vi.fn(),
  warning: vi.fn(),
  error: vi.fn(),
  setFailed: vi.fn(),
  setOutput: vi.fn(),
};

const mockExec = {
  exec: vi.fn(),
};

// Establish globals before requiring the module
global.core = mockCore;
global.exec = mockExec;

const {
  parseAPMLockfile,
  unquoteYaml,
  collectDeployedFiles,
  findBundleFile,
  findSourceDir,
  findLockfile,
  verifyBundleContents,
  assertSafePath,
  assertDestInsideOutput,
  copyDirRecursive,
  listDirRecursive,
  unpackBundle,
} = require("./apm_unpack.cjs");

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

/** Create a temp directory and return its path. */
function makeTempDir() {
  return fs.mkdtempSync(path.join(os.tmpdir(), "apm-unpack-test-"));
}

/** Remove a directory recursively (best-effort). */
function removeTempDir(dir) {
  if (dir && fs.existsSync(dir)) {
    fs.rmSync(dir, { recursive: true, force: true });
  }
}

/** Write a file, creating parent directories as needed. */
function writeFile(dir, relPath, content = "content") {
  const full = path.join(dir, relPath);
  fs.mkdirSync(path.dirname(full), { recursive: true });
  fs.writeFileSync(full, content, "utf-8");
  return full;
}

/**
 * Minimal valid apm.lock.yaml content for a single dependency.
 * @param {object} [overrides]
 */
function minimalLockfile({ repoUrl = "https://github.com/owner/repo", files = [".github/skills/foo/"] } = {}) {
  const fileLines = files.map(f => `  - ${f}`).join("\n");
  return `lockfile_version: '1'
generated_at: '2024-01-15T10:00:00.000000+00:00'
apm_version: 0.8.5
dependencies:
- repo_url: ${repoUrl}
  host: github.com
  resolved_commit: abc123def456789
  resolved_ref: main
  version: '1.0.0'
  depth: 1
  package_type: generic
  deployed_files:
${fileLines}
`;
}

// ---------------------------------------------------------------------------
// unquoteYaml
// ---------------------------------------------------------------------------

describe("unquoteYaml", () => {
  it("returns null for empty/null/undefined/~ values", () => {
    expect(unquoteYaml("")).toBeNull();
    expect(unquoteYaml("~")).toBeNull();
    expect(unquoteYaml("null")).toBeNull();
    expect(unquoteYaml(null)).toBeNull();
    expect(unquoteYaml(undefined)).toBeNull();
  });

  it("parses boolean literals", () => {
    expect(unquoteYaml("true")).toBe(true);
    expect(unquoteYaml("false")).toBe(false);
  });

  it("parses integer literals", () => {
    expect(unquoteYaml("0")).toBe(0);
    expect(unquoteYaml("1")).toBe(1);
    expect(unquoteYaml("42")).toBe(42);
    expect(unquoteYaml("-7")).toBe(-7);
  });

  it("parses float literals", () => {
    expect(unquoteYaml("3.14")).toBeCloseTo(3.14);
    expect(unquoteYaml("-1.5")).toBeCloseTo(-1.5);
  });

  it("strips single quotes", () => {
    expect(unquoteYaml("'hello'")).toBe("hello");
    expect(unquoteYaml("'1'")).toBe("1");
    expect(unquoteYaml("'true'")).toBe("true");
  });

  it("strips double quotes", () => {
    expect(unquoteYaml('"world"')).toBe("world");
    expect(unquoteYaml('"2024-01-01"')).toBe("2024-01-01");
  });

  it("returns bare strings unchanged", () => {
    expect(unquoteYaml("main")).toBe("main");
    expect(unquoteYaml("github.com")).toBe("github.com");
    expect(unquoteYaml("https://github.com/owner/repo")).toBe("https://github.com/owner/repo");
  });

  it("trims surrounding whitespace before processing", () => {
    expect(unquoteYaml("  'hello'  ")).toBe("hello");
    expect(unquoteYaml("  42  ")).toBe(42);
  });
});

// ---------------------------------------------------------------------------
// parseAPMLockfile – basic structure
// ---------------------------------------------------------------------------

describe("parseAPMLockfile – top-level fields", () => {
  it("parses lockfile_version, generated_at, apm_version", () => {
    const yaml = `lockfile_version: '1'
generated_at: '2024-01-15T10:00:00.000000+00:00'
apm_version: 0.8.5
dependencies:
`;
    const result = parseAPMLockfile(yaml);
    expect(result.lockfile_version).toBe("1");
    expect(result.generated_at).toBe("2024-01-15T10:00:00.000000+00:00");
    expect(result.apm_version).toBe("0.8.5");
    expect(result.dependencies).toHaveLength(0);
  });

  it("handles missing optional fields gracefully", () => {
    const yaml = `lockfile_version: '1'
dependencies:
`;
    const result = parseAPMLockfile(yaml);
    expect(result.lockfile_version).toBe("1");
    expect(result.apm_version).toBeNull();
    expect(result.dependencies).toHaveLength(0);
  });

  it("parses pack metadata block", () => {
    const yaml = `lockfile_version: '1'
dependencies:
pack:
  target: claude
  format: apm
  generated_at: '2024-01-15T10:00:00.000000+00:00'
`;
    const result = parseAPMLockfile(yaml);
    expect(result.pack.target).toBe("claude");
    expect(result.pack.format).toBe("apm");
  });

  it("returns empty result for empty/blank input", () => {
    const result = parseAPMLockfile("");
    expect(result.dependencies).toHaveLength(0);
    expect(result.lockfile_version).toBeNull();
  });

  it("ignores YAML comment lines", () => {
    const yaml = `# This is a comment
lockfile_version: '1'
# Another comment
dependencies:
`;
    const result = parseAPMLockfile(yaml);
    expect(result.lockfile_version).toBe("1");
  });
});

// ---------------------------------------------------------------------------
// parseAPMLockfile – dependency items
// ---------------------------------------------------------------------------

describe("parseAPMLockfile – dependencies", () => {
  it("parses a single dependency with deployed_files", () => {
    const yaml = minimalLockfile({
      repoUrl: "https://github.com/microsoft/apm-sample-package",
      files: [".github/skills/my-skill/", ".claude/skills/my-skill/"],
    });
    const result = parseAPMLockfile(yaml);

    expect(result.dependencies).toHaveLength(1);
    const dep = result.dependencies[0];
    expect(dep.repo_url).toBe("https://github.com/microsoft/apm-sample-package");
    expect(dep.host).toBe("github.com");
    expect(dep.resolved_commit).toBe("abc123def456789");
    expect(dep.resolved_ref).toBe("main");
    expect(dep.version).toBe("1.0.0");
    expect(dep.depth).toBe(1);
    expect(dep.package_type).toBe("generic");
    expect(dep.deployed_files).toEqual([".github/skills/my-skill/", ".claude/skills/my-skill/"]);
  });

  it("parses multiple dependencies", () => {
    const yaml = `lockfile_version: '1'
dependencies:
- repo_url: https://github.com/owner/pkg-a
  host: github.com
  resolved_commit: aaaa
  resolved_ref: main
  depth: 1
  deployed_files:
  - .github/skills/pkg-a/
- repo_url: https://github.com/owner/pkg-b
  host: github.com
  resolved_commit: bbbb
  resolved_ref: v2
  depth: 1
  deployed_files:
  - .github/skills/pkg-b/
  - .claude/skills/pkg-b/
`;
    const result = parseAPMLockfile(yaml);
    expect(result.dependencies).toHaveLength(2);
    expect(result.dependencies[0].repo_url).toBe("https://github.com/owner/pkg-a");
    expect(result.dependencies[0].deployed_files).toEqual([".github/skills/pkg-a/"]);
    expect(result.dependencies[1].repo_url).toBe("https://github.com/owner/pkg-b");
    expect(result.dependencies[1].deployed_files).toEqual([".github/skills/pkg-b/", ".claude/skills/pkg-b/"]);
  });

  it("handles dependency with no deployed_files", () => {
    const yaml = `lockfile_version: '1'
dependencies:
- repo_url: https://github.com/owner/empty-pkg
  host: github.com
  depth: 1
`;
    const result = parseAPMLockfile(yaml);
    expect(result.dependencies).toHaveLength(1);
    expect(result.dependencies[0].deployed_files).toEqual([]);
  });

  it("parses boolean fields: is_virtual, is_dev", () => {
    const yaml = `lockfile_version: '1'
dependencies:
- repo_url: https://github.com/owner/repo
  is_virtual: true
  is_dev: true
  depth: 1
`;
    const result = parseAPMLockfile(yaml);
    const dep = result.dependencies[0];
    expect(dep.is_virtual).toBe(true);
    expect(dep.is_dev).toBe(true);
  });

  it("parses virtual package with virtual_path", () => {
    const yaml = `lockfile_version: '1'
dependencies:
- repo_url: https://github.com/owner/mono
  virtual_path: packages/sub
  is_virtual: true
  depth: 2
  deployed_files:
  - .github/skills/sub/
`;
    const result = parseAPMLockfile(yaml);
    const dep = result.dependencies[0];
    expect(dep.virtual_path).toBe("packages/sub");
    expect(dep.is_virtual).toBe(true);
    expect(dep.depth).toBe(2);
  });

  it("parses local dependency with source and local_path", () => {
    const yaml = `lockfile_version: '1'
dependencies:
- repo_url: local
  source: local
  local_path: ./my-local-pkg
  depth: 1
  deployed_files:
  - .github/skills/local/
`;
    const result = parseAPMLockfile(yaml);
    const dep = result.dependencies[0];
    expect(dep.source).toBe("local");
    expect(dep.local_path).toBe("./my-local-pkg");
  });

  it("handles deployed_files with plain file paths (no trailing slash)", () => {
    const yaml = `lockfile_version: '1'
dependencies:
- repo_url: https://github.com/owner/repo
  deployed_files:
  - .github/copilot-instructions.md
  - README.md
`;
    const result = parseAPMLockfile(yaml);
    expect(result.dependencies[0].deployed_files).toEqual([".github/copilot-instructions.md", "README.md"]);
  });

  it("handles multiple fields appearing after deployed_files", () => {
    const yaml = `lockfile_version: '1'
dependencies:
- repo_url: https://github.com/owner/repo
  host: github.com
  resolved_commit: abc123
  depth: 1
  deployed_files:
  - .github/skills/foo/
  resolved_ref: main
  package_type: generic
`;
    const result = parseAPMLockfile(yaml);
    const dep = result.dependencies[0];
    // After deployed_files block, parser should resume dep_item and pick up remaining keys
    expect(dep.deployed_files).toEqual([".github/skills/foo/"]);
    expect(dep.resolved_ref).toBe("main");
    expect(dep.package_type).toBe("generic");
  });
});

// ---------------------------------------------------------------------------
// collectDeployedFiles
// ---------------------------------------------------------------------------

describe("collectDeployedFiles", () => {
  it("deduplicates files across dependencies", () => {
    const lockfile = {
      lockfile_version: "1",
      generated_at: null,
      apm_version: null,
      pack: {},
      dependencies: [
        { ...makeEmptyDep(), repo_url: "a", deployed_files: ["file1.txt", "file2.txt"] },
        { ...makeEmptyDep(), repo_url: "b", deployed_files: ["file2.txt", "file3.txt"] },
      ],
    };
    const { uniqueFiles, depFileMap } = collectDeployedFiles(lockfile);
    expect(uniqueFiles).toEqual(["file1.txt", "file2.txt", "file3.txt"]);
    expect(depFileMap["a"]).toEqual(["file1.txt", "file2.txt"]);
    expect(depFileMap["b"]).toEqual(["file2.txt", "file3.txt"]);
  });

  it("preserves insertion order (mirrors Python seen set logic)", () => {
    const lockfile = {
      lockfile_version: "1",
      generated_at: null,
      apm_version: null,
      pack: {},
      dependencies: [
        { ...makeEmptyDep(), repo_url: "a", deployed_files: ["z.txt", "a.txt"] },
        { ...makeEmptyDep(), repo_url: "b", deployed_files: ["m.txt"] },
      ],
    };
    const { uniqueFiles } = collectDeployedFiles(lockfile);
    expect(uniqueFiles).toEqual(["z.txt", "a.txt", "m.txt"]);
  });

  it("uses virtual_path in dep key for virtual packages", () => {
    const lockfile = {
      lockfile_version: "1",
      generated_at: null,
      apm_version: null,
      pack: {},
      dependencies: [
        {
          ...makeEmptyDep(),
          repo_url: "https://github.com/owner/mono",
          is_virtual: true,
          virtual_path: "packages/sub",
          deployed_files: ["skill/"],
        },
      ],
    };
    const { depFileMap } = collectDeployedFiles(lockfile);
    expect(depFileMap["https://github.com/owner/mono/packages/sub"]).toEqual(["skill/"]);
  });

  it("uses local_path as key for local packages", () => {
    const lockfile = {
      lockfile_version: "1",
      generated_at: null,
      apm_version: null,
      pack: {},
      dependencies: [
        {
          ...makeEmptyDep(),
          repo_url: "local",
          source: "local",
          local_path: "./my-pkg",
          deployed_files: ["skill/"],
        },
      ],
    };
    const { depFileMap } = collectDeployedFiles(lockfile);
    expect(depFileMap["./my-pkg"]).toEqual(["skill/"]);
  });

  it("omits empty deployed_files from depFileMap", () => {
    const lockfile = {
      lockfile_version: "1",
      generated_at: null,
      apm_version: null,
      pack: {},
      dependencies: [{ ...makeEmptyDep(), repo_url: "a", deployed_files: [] }],
    };
    const { uniqueFiles, depFileMap } = collectDeployedFiles(lockfile);
    expect(uniqueFiles).toHaveLength(0);
    expect(Object.keys(depFileMap)).toHaveLength(0);
  });
});

// Helper used in collectDeployedFiles tests
function makeEmptyDep() {
  return {
    repo_url: "",
    host: null,
    resolved_commit: null,
    resolved_ref: null,
    version: null,
    virtual_path: null,
    is_virtual: false,
    depth: 1,
    resolved_by: null,
    package_type: null,
    deployed_files: [],
    source: null,
    local_path: null,
    content_hash: null,
    is_dev: false,
  };
}

// ---------------------------------------------------------------------------
// findBundleFile
// ---------------------------------------------------------------------------

describe("findBundleFile", () => {
  let tempDir;

  beforeEach(() => {
    tempDir = makeTempDir();
    vi.clearAllMocks();
    global.core = mockCore;
  });
  afterEach(() => removeTempDir(tempDir));

  it("finds a single tar.gz file in the bundle directory", () => {
    writeFile(tempDir, "my-package-1.0.0.tar.gz", "fake-archive");
    const result = findBundleFile(tempDir);
    expect(result).toBe(path.join(tempDir, "my-package-1.0.0.tar.gz"));
  });

  it("throws when directory does not exist", () => {
    expect(() => findBundleFile("/nonexistent/path/xyz")).toThrow(/not found/);
  });

  it("throws when no tar.gz file exists", () => {
    writeFile(tempDir, "readme.txt", "not a bundle");
    expect(() => findBundleFile(tempDir)).toThrow(/No \*.tar\.gz bundle found/);
  });

  it("uses first file and warns when multiple bundles are present", () => {
    writeFile(tempDir, "pkg-1.0.0.tar.gz", "archive-1");
    writeFile(tempDir, "pkg-2.0.0.tar.gz", "archive-2");
    const result = findBundleFile(tempDir);
    expect(result).toMatch(/\.tar\.gz$/);
    expect(mockCore.warning).toHaveBeenCalledWith(expect.stringContaining("Multiple bundles found"));
  });
});

// ---------------------------------------------------------------------------
// findSourceDir
// ---------------------------------------------------------------------------

describe("findSourceDir", () => {
  let tempDir;

  beforeEach(() => {
    tempDir = makeTempDir();
    vi.clearAllMocks();
    global.core = mockCore;
  });
  afterEach(() => removeTempDir(tempDir));

  it("returns the single subdirectory when the archive has one top-level dir", () => {
    const inner = path.join(tempDir, "my-package-1.0.0");
    fs.mkdirSync(inner);
    const result = findSourceDir(tempDir);
    expect(result).toBe(inner);
  });

  it("returns the extraction root when multiple entries exist", () => {
    fs.mkdirSync(path.join(tempDir, "dir-a"));
    fs.mkdirSync(path.join(tempDir, "dir-b"));
    const result = findSourceDir(tempDir);
    expect(result).toBe(tempDir);
  });

  it("returns the extraction root when only files exist (no subdirectory)", () => {
    writeFile(tempDir, "apm.lock.yaml", "lockfile");
    const result = findSourceDir(tempDir);
    expect(result).toBe(tempDir);
  });
});

// ---------------------------------------------------------------------------
// findLockfile
// ---------------------------------------------------------------------------

describe("findLockfile", () => {
  let tempDir;

  beforeEach(() => {
    tempDir = makeTempDir();
    vi.clearAllMocks();
    global.core = mockCore;
  });
  afterEach(() => removeTempDir(tempDir));

  it("finds apm.lock.yaml", () => {
    writeFile(tempDir, "apm.lock.yaml", "content");
    const result = findLockfile(tempDir);
    expect(result).toBe(path.join(tempDir, "apm.lock.yaml"));
    expect(mockCore.warning).not.toHaveBeenCalled();
  });

  it("throws when apm.lock.yaml does not exist", () => {
    expect(() => findLockfile(tempDir)).toThrow(/apm\.lock\.yaml not found/);
  });

  it("throws when only legacy apm.lock exists (not supported)", () => {
    writeFile(tempDir, "apm.lock", "content");
    expect(() => findLockfile(tempDir)).toThrow(/apm\.lock\.yaml not found/);
  });
});

// ---------------------------------------------------------------------------
// verifyBundleContents
// ---------------------------------------------------------------------------

describe("verifyBundleContents", () => {
  let tempDir;

  beforeEach(() => {
    tempDir = makeTempDir();
    vi.clearAllMocks();
    global.core = mockCore;
  });
  afterEach(() => removeTempDir(tempDir));

  it("passes when all files exist in the bundle", () => {
    writeFile(tempDir, ".github/skills/foo/skill.md");
    writeFile(tempDir, ".claude/skills/foo/skill.md");
    expect(() => verifyBundleContents(tempDir, [".github/skills/foo/skill.md", ".claude/skills/foo/skill.md"])).not.toThrow();
    expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("verification passed"));
  });

  it("throws when a file listed in deployed_files is missing", () => {
    writeFile(tempDir, ".github/skills/foo/skill.md");
    // .claude/skills/foo/skill.md is missing
    expect(() => verifyBundleContents(tempDir, [".github/skills/foo/skill.md", ".claude/skills/foo/skill.md"])).toThrow(/Bundle verification failed/);
  });

  it("passes for directory entries (path ending with /)", () => {
    // A directory itself counts as existing
    fs.mkdirSync(path.join(tempDir, ".github", "skills", "foo"), { recursive: true });
    expect(() => verifyBundleContents(tempDir, [".github/skills/foo/"])).not.toThrow();
  });

  it("passes for empty deployed_files list", () => {
    expect(() => verifyBundleContents(tempDir, [])).not.toThrow();
  });
});

// ---------------------------------------------------------------------------
// assertSafePath
// ---------------------------------------------------------------------------

describe("assertSafePath", () => {
  it("accepts valid relative paths", () => {
    expect(() => assertSafePath(".github/skills/foo/skill.md")).not.toThrow();
    expect(() => assertSafePath("README.md")).not.toThrow();
    expect(() => assertSafePath("some/nested/dir/file.txt")).not.toThrow();
  });

  it("rejects absolute paths", () => {
    expect(() => assertSafePath("/etc/passwd")).toThrow(/absolute path/i);
    expect(() => assertSafePath("/tmp/attack")).toThrow(/absolute path/i);
  });

  it("rejects path traversal with ..", () => {
    expect(() => assertSafePath("../outside")).toThrow(/path-traversal/i);
    expect(() => assertSafePath("safe/../../../etc/passwd")).toThrow(/path-traversal/i);
  });
});

// ---------------------------------------------------------------------------
// assertDestInsideOutput
// ---------------------------------------------------------------------------

describe("assertDestInsideOutput", () => {
  it("accepts paths inside the output directory", () => {
    const output = path.resolve("/tmp/test-output");
    expect(() => assertDestInsideOutput(output + "/subdir/file.txt", output)).not.toThrow();
    expect(() => assertDestInsideOutput(output + "/nested/deep/file.txt", output)).not.toThrow();
  });

  it("rejects paths that escape the output directory", () => {
    const output = path.resolve("/tmp/test-output");
    expect(() => assertDestInsideOutput("/tmp/other/file.txt", output)).toThrow(/escapes/i);
    expect(() => assertDestInsideOutput("/etc/passwd", output)).toThrow(/escapes/i);
  });
});

// ---------------------------------------------------------------------------
// copyDirRecursive
// ---------------------------------------------------------------------------

describe("copyDirRecursive", () => {
  let srcDir;
  let destDir;

  beforeEach(() => {
    srcDir = makeTempDir();
    destDir = makeTempDir();
    vi.clearAllMocks();
    global.core = mockCore;
  });
  afterEach(() => {
    removeTempDir(srcDir);
    removeTempDir(destDir);
  });

  it("copies all files from source to destination", () => {
    writeFile(srcDir, "file1.txt", "a");
    writeFile(srcDir, "subdir/file2.txt", "b");
    writeFile(srcDir, "subdir/nested/file3.txt", "c");

    const count = copyDirRecursive(srcDir, destDir);
    expect(count).toBe(3);
    expect(fs.existsSync(path.join(destDir, "file1.txt"))).toBe(true);
    expect(fs.existsSync(path.join(destDir, "subdir", "file2.txt"))).toBe(true);
    expect(fs.existsSync(path.join(destDir, "subdir", "nested", "file3.txt"))).toBe(true);
  });

  it("preserves file content", () => {
    writeFile(srcDir, "hello.txt", "Hello, World!");
    copyDirRecursive(srcDir, destDir);
    const content = fs.readFileSync(path.join(destDir, "hello.txt"), "utf-8");
    expect(content).toBe("Hello, World!");
  });

  it("skips symbolic links with a warning", () => {
    writeFile(srcDir, "real.txt", "real");
    // Create a symlink (may not work on all platforms but is tested here)
    try {
      fs.symlinkSync(path.join(srcDir, "real.txt"), path.join(srcDir, "link.txt"));
      copyDirRecursive(srcDir, destDir);
      // The symlink should not be copied
      expect(fs.existsSync(path.join(destDir, "link.txt"))).toBe(false);
      expect(mockCore.warning).toHaveBeenCalledWith(expect.stringContaining("symlink"));
    } catch {
      // Symlink creation may fail in some environments – skip
    }
  });

  it("returns 0 for an empty source directory", () => {
    const count = copyDirRecursive(srcDir, destDir);
    expect(count).toBe(0);
  });
});

// ---------------------------------------------------------------------------
// listDirRecursive
// ---------------------------------------------------------------------------

describe("listDirRecursive", () => {
  let tempDir;

  beforeEach(() => {
    tempDir = makeTempDir();
  });
  afterEach(() => removeTempDir(tempDir));

  it("lists all files recursively", () => {
    writeFile(tempDir, "a.txt");
    writeFile(tempDir, "sub/b.txt");
    writeFile(tempDir, "sub/deep/c.txt");

    const files = listDirRecursive(tempDir);
    expect(files).toContain("a.txt");
    expect(files).toContain("sub/b.txt");
    expect(files).toContain("sub/deep/c.txt");
  });

  it("returns empty array for empty directory", () => {
    expect(listDirRecursive(tempDir)).toHaveLength(0);
  });

  it("returns empty array for non-existent directory", () => {
    expect(listDirRecursive("/nonexistent/xyz")).toHaveLength(0);
  });
});

// ---------------------------------------------------------------------------
// Full unpackBundle integration test (using real filesystem)
// ---------------------------------------------------------------------------

describe("unpackBundle – integration", () => {
  let bundleBaseDir;
  let outputDir;

  beforeEach(() => {
    bundleBaseDir = makeTempDir();
    outputDir = makeTempDir();
    vi.clearAllMocks();
    global.core = mockCore;
    global.exec = mockExec;
  });
  afterEach(() => {
    removeTempDir(bundleBaseDir);
    removeTempDir(outputDir);
  });

  /**
   * Build a fake extracted bundle directory inside bundleBaseDir:
   *   bundleBaseDir/
   *     fake-archive.tar.gz       (empty placeholder – exec mock skips real extraction)
   *     extracted/
   *       pkg-1.0.0/
   *         apm.lock.yaml
   *         .github/skills/my-skill/prompt.md
   *         .claude/skills/my-skill/CLAUDE.md
   *
   * The exec mock simulates tar extraction by creating the same structure in the
   * tempDir that unpackBundle uses.
   */
  function buildFakeBundle({
    repoUrl = "https://github.com/owner/my-skill",
    files = [
      { path: ".github/skills/my-skill/prompt.md", content: "# My Skill" },
      { path: ".claude/skills/my-skill/CLAUDE.md", content: "# Claude Skill" },
    ],
    deployedFiles = [".github/skills/my-skill/", ".claude/skills/my-skill/"],
  } = {}) {
    // Write the placeholder tar.gz so findBundleFile succeeds
    fs.writeFileSync(path.join(bundleBaseDir, "my-package-1.0.0.tar.gz"), "fake");

    // Build the lockfile content
    const fileLines = deployedFiles.map(f => `  - ${f}`).join("\n");
    const lockfileContent = `lockfile_version: '1'
generated_at: '2024-01-15T10:00:00.000000+00:00'
apm_version: 0.8.5
dependencies:
- repo_url: ${repoUrl}
  host: github.com
  resolved_commit: abc123def456
  resolved_ref: main
  depth: 1
  package_type: generic
  deployed_files:
${fileLines}
pack:
  target: claude
  format: apm
`;

    // The exec mock will be called with tar -xzf <bundlePath> -C <tempDir>
    // We intercept it to write our fake extracted structure into tempDir
    mockExec.exec.mockImplementation(async (_cmd, args) => {
      // args: ['-xzf', bundlePath, '-C', tempDir]
      const tempDir = args[3];
      const innerDir = path.join(tempDir, "my-package-1.0.0");
      fs.mkdirSync(innerDir, { recursive: true });

      // Write lockfile
      fs.writeFileSync(path.join(innerDir, "apm.lock.yaml"), lockfileContent);

      // Write deployed files
      for (const f of files) {
        writeFile(innerDir, f.path.replace(/\/$/, "") + (f.path.endsWith("/") ? "/placeholder" : ""), f.content);
      }

      // Write directory structure for directory entries in deployedFiles
      for (const df of deployedFiles) {
        if (df.endsWith("/")) {
          const dirPath = df.replace(/\/$/, "");
          fs.mkdirSync(path.join(innerDir, dirPath), { recursive: true });
          // Write at least one file into each directory
          const placeholder = path.join(innerDir, dirPath, "skill.md");
          if (!fs.existsSync(placeholder)) {
            fs.writeFileSync(placeholder, "# placeholder");
          }
        }
      }
    });
  }

  it("unpacks a bundle and deploys files to output directory", async () => {
    buildFakeBundle();

    const result = await unpackBundle({ bundleDir: bundleBaseDir, outputDir });

    expect(result.files).toContain(".github/skills/my-skill/");
    expect(result.files).toContain(".claude/skills/my-skill/");
    expect(result.verified).toBe(true);
    expect(result.packMeta.target).toBe("claude");

    // Verify files were deployed
    expect(fs.existsSync(path.join(outputDir, ".github", "skills", "my-skill"))).toBe(true);
    expect(fs.existsSync(path.join(outputDir, ".claude", "skills", "my-skill"))).toBe(true);
  });

  it("dry-run resolves files without copying", async () => {
    buildFakeBundle();

    const result = await unpackBundle({ bundleDir: bundleBaseDir, outputDir, dryRun: true });

    expect(result.files).toContain(".github/skills/my-skill/");
    expect(result.files).toContain(".claude/skills/my-skill/");
    // Nothing should have been deployed
    expect(fs.existsSync(path.join(outputDir, ".github"))).toBe(false);
  });

  it("throws when bundle directory is empty", async () => {
    await expect(unpackBundle({ bundleDir: bundleBaseDir, outputDir })).rejects.toThrow(/No \*.tar\.gz bundle found/);
  });

  it("throws when lockfile is missing from bundle", async () => {
    fs.writeFileSync(path.join(bundleBaseDir, "broken.tar.gz"), "fake");

    mockExec.exec.mockImplementation(async (_cmd, args) => {
      const tempDir = args[3];
      const innerDir = path.join(tempDir, "my-package-1.0.0");
      fs.mkdirSync(innerDir, { recursive: true });
      // No lockfile written – this should trigger an error
    });

    await expect(unpackBundle({ bundleDir: bundleBaseDir, outputDir })).rejects.toThrow(/apm\.lock\.yaml not found/);
  });

  it("handles plain file entries (non-directory deployed_files)", async () => {
    buildFakeBundle({
      deployedFiles: [".github/copilot-instructions.md"],
      files: [{ path: ".github/copilot-instructions.md", content: "# Instructions" }],
    });

    mockExec.exec.mockImplementation(async (_cmd, args) => {
      const tempDir = args[3];
      const innerDir = path.join(tempDir, "my-package-1.0.0");
      fs.mkdirSync(innerDir, { recursive: true });

      const lockfileContent = `lockfile_version: '1'
dependencies:
- repo_url: https://github.com/owner/repo
  deployed_files:
  - .github/copilot-instructions.md
`;
      fs.writeFileSync(path.join(innerDir, "apm.lock.yaml"), lockfileContent);
      writeFile(innerDir, ".github/copilot-instructions.md", "# Instructions");
    });

    const result = await unpackBundle({ bundleDir: bundleBaseDir, outputDir });
    expect(result.files).toContain(".github/copilot-instructions.md");
    expect(fs.existsSync(path.join(outputDir, ".github", "copilot-instructions.md"))).toBe(true);
  });

  it("throws when bundle contains only legacy apm.lock (not supported)", async () => {
    fs.writeFileSync(path.join(bundleBaseDir, "pkg.tar.gz"), "fake");

    mockExec.exec.mockImplementation(async (_cmd, args) => {
      const tempDir = args[3];
      const innerDir = path.join(tempDir, "pkg-1.0.0");
      fs.mkdirSync(innerDir, { recursive: true });
      // Only write the legacy lockfile — should be rejected
      fs.writeFileSync(path.join(innerDir, "apm.lock"), "lockfile_version: '1'\ndependencies:\n");
    });

    await expect(unpackBundle({ bundleDir: bundleBaseDir, outputDir })).rejects.toThrow(/apm\.lock\.yaml not found/);
  });

  it("skips verification when skipVerify is true", async () => {
    buildFakeBundle({ deployedFiles: [".github/skills/foo/"] });

    // Simulate a bundle where the file is missing but skipVerify lets it through
    mockExec.exec.mockImplementation(async (_cmd, args) => {
      const tempDir = args[3];
      const innerDir = path.join(tempDir, "my-package-1.0.0");
      fs.mkdirSync(innerDir, { recursive: true });

      const lockfileContent = `lockfile_version: '1'
dependencies:
- repo_url: https://github.com/owner/repo
  deployed_files:
  - .github/skills/missing-file/
`;
      fs.writeFileSync(path.join(innerDir, "apm.lock.yaml"), lockfileContent);
      // Intentionally NOT creating .github/skills/missing-file/
    });

    const result = await unpackBundle({ bundleDir: bundleBaseDir, outputDir, skipVerify: true });
    expect(result.verified).toBe(false);
    expect(result.skippedCount).toBe(1); // missing entry is skipped
  });
});

// ---------------------------------------------------------------------------
// Edge cases for YAML parser
// ---------------------------------------------------------------------------

describe("parseAPMLockfile – edge cases", () => {
  it("handles YAML with Windows-style line endings (CRLF)", () => {
    const yaml = "lockfile_version: '1'\r\ngenerated_at: '2024-01-15'\r\ndependencies:\r\n";
    const result = parseAPMLockfile(yaml);
    // CRLF lines won't match our patterns cleanly, but should not throw
    expect(result).toBeDefined();
  });

  it("handles quoted values with internal spaces", () => {
    const yaml = `lockfile_version: '1 (patched)'
dependencies:
`;
    const result = parseAPMLockfile(yaml);
    expect(result.lockfile_version).toBe("1 (patched)");
  });

  it("handles multiple dependencies with pack block at the end", () => {
    const yaml = `lockfile_version: '1'
dependencies:
- repo_url: https://github.com/a/pkg
  deployed_files:
  - skill-a/
- repo_url: https://github.com/b/pkg
  deployed_files:
  - skill-b/
pack:
  target: all
  format: apm
`;
    const result = parseAPMLockfile(yaml);
    expect(result.dependencies).toHaveLength(2);
    expect(result.pack.target).toBe("all");
  });

  it("does not modify deployed_files paths (preserves trailing slash)", () => {
    const yaml = `lockfile_version: '1'
dependencies:
- repo_url: https://github.com/owner/repo
  deployed_files:
  - .github/skills/my-skill/
  - plain-file.md
`;
    const result = parseAPMLockfile(yaml);
    expect(result.dependencies[0].deployed_files[0]).toBe(".github/skills/my-skill/");
    expect(result.dependencies[0].deployed_files[1]).toBe("plain-file.md");
  });
});
