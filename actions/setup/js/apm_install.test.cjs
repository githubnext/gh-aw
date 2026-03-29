// @ts-check
/// <reference types="@actions/github-script" />

import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
const fs = require("fs");
const path = require("path");
const os = require("os");

// ---------------------------------------------------------------------------
// Global mock setup — must be done before requiring apm_install.cjs
// ---------------------------------------------------------------------------

const mockCore = {
  info: vi.fn(),
  warning: vi.fn(),
  error: vi.fn(),
  setFailed: vi.fn(),
  setOutput: vi.fn(),
};

global.core = mockCore;

// apm_install.cjs calls require('@actions/github') only inside createOctokit(),
// which is only reached when octokitOverride is null. All unit tests inject
// octokitOverride, so we never actually load @actions/github here.

const { parsePackageSlug, selectDeployableFiles, writeWorkspaceLockfile, writeWorkspaceApmYml, parsePackagesFromEnv, scalarToYaml, main } = require("./apm_install.cjs");

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

/** Create a temp directory and return its path. */
function makeTempDir() {
  return fs.mkdtempSync(path.join(os.tmpdir(), "apm-install-test-"));
}

/** Remove a temp directory (best-effort). */
function removeTempDir(dir) {
  if (dir && fs.existsSync(dir)) {
    fs.rmSync(dir, { recursive: true, force: true });
  }
}

/** Build a minimal mock Octokit that returns the given tree and content map. */
function makeMockOctokit({ defaultBranch = "main", commitSha = "aabbcc00", treeSha = "tree00", tree = [], contentMap = {} } = {}) {
  return {
    rest: {
      repos: {
        get: vi.fn().mockResolvedValue({ data: { default_branch: defaultBranch } }),
        getCommit: vi.fn().mockResolvedValue({
          data: { sha: commitSha, commit: { tree: { sha: treeSha } } },
        }),
        getContent: vi.fn().mockImplementation(({ path: filePath }) => {
          const text = contentMap[filePath];
          if (text === undefined) {
            return Promise.reject(Object.assign(new Error("Not Found"), { status: 404 }));
          }
          return Promise.resolve({
            data: {
              type: "file",
              encoding: "base64",
              content: Buffer.from(text).toString("base64"),
            },
          });
        }),
      },
      git: {
        getTree: vi.fn().mockResolvedValue({ data: { tree } }),
      },
    },
  };
}

// ---------------------------------------------------------------------------
// parsePackageSlug
// ---------------------------------------------------------------------------

describe("parsePackageSlug", () => {
  it("parses owner/repo", () => {
    expect(parsePackageSlug("microsoft/apm-sample-package")).toEqual({
      owner: "microsoft",
      repo: "apm-sample-package",
      subpath: null,
      ref: null,
    });
  });

  it("parses owner/repo#ref", () => {
    expect(parsePackageSlug("microsoft/apm-sample-package#v2.0")).toEqual({
      owner: "microsoft",
      repo: "apm-sample-package",
      subpath: null,
      ref: "v2.0",
    });
  });

  it("parses owner/repo/subpath", () => {
    expect(parsePackageSlug("github/awesome-copilot/skills/review-and-refactor")).toEqual({
      owner: "github",
      repo: "awesome-copilot",
      subpath: "skills/review-and-refactor",
      ref: null,
    });
  });

  it("parses owner/repo/subpath#ref", () => {
    expect(parsePackageSlug("org/repo/skills/foo#main")).toEqual({
      owner: "org",
      repo: "repo",
      subpath: "skills/foo",
      ref: "main",
    });
  });

  it("handles a commit SHA as ref", () => {
    const { ref } = parsePackageSlug("org/repo#abc123def456");
    expect(ref).toBe("abc123def456");
  });

  it("throws for slug without slash", () => {
    expect(() => parsePackageSlug("just-one-part")).toThrow(/Invalid package slug/);
  });

  it("throws for empty string", () => {
    expect(() => parsePackageSlug("")).toThrow(/Invalid package slug/);
  });

  it("throws for missing repo component", () => {
    expect(() => parsePackageSlug("owner/")).toThrow(/Invalid package slug/);
  });
});

// ---------------------------------------------------------------------------
// selectDeployableFiles – full package (no subpath)
// ---------------------------------------------------------------------------

describe("selectDeployableFiles – full package", () => {
  /** @type {Array<{path: string, sha: string}>} */
  const tree = [
    { path: ".github/skills/foo/skill.md", sha: "a1" },
    { path: ".github/agents/bar.md", sha: "a2" },
    { path: ".claude/skills/foo/skill.md", sha: "a3" },
    { path: ".cursor/rules/style.md", sha: "a4" },
    { path: ".opencode/instructions.md", sha: "a5" },
    { path: "README.md", sha: "a6" },
    { path: "apm.yml", sha: "a7" },
  ];

  it("includes files under all target dirs", () => {
    const result = selectDeployableFiles(tree, null).map(e => e.path);
    expect(result).toContain(".github/skills/foo/skill.md");
    expect(result).toContain(".github/agents/bar.md");
    expect(result).toContain(".claude/skills/foo/skill.md");
    expect(result).toContain(".cursor/rules/style.md");
    expect(result).toContain(".opencode/instructions.md");
  });

  it("excludes non-target files", () => {
    const result = selectDeployableFiles(tree, null).map(e => e.path);
    expect(result).not.toContain("README.md");
    expect(result).not.toContain("apm.yml");
  });
});

// ---------------------------------------------------------------------------
// selectDeployableFiles – individual primitive (with subpath)
// ---------------------------------------------------------------------------

describe("selectDeployableFiles – primitive subpath", () => {
  /** @type {Array<{path: string, sha: string}>} */
  const tree = [
    { path: ".github/skills/review-and-refactor/skill.md", sha: "b1" },
    { path: ".github/skills/review-and-refactor/notes.txt", sha: "b2" },
    { path: ".claude/skills/review-and-refactor/skill.md", sha: "b3" },
    { path: ".github/skills/other-skill/skill.md", sha: "b4" },
    { path: ".github/agents/agent.md", sha: "b5" },
    { path: "README.md", sha: "b6" },
  ];

  it("selects files under the subpath in all target dirs", () => {
    const result = selectDeployableFiles(tree, "skills/review-and-refactor").map(e => e.path);
    expect(result).toContain(".github/skills/review-and-refactor/skill.md");
    expect(result).toContain(".github/skills/review-and-refactor/notes.txt");
    expect(result).toContain(".claude/skills/review-and-refactor/skill.md");
  });

  it("excludes sibling skills", () => {
    const result = selectDeployableFiles(tree, "skills/review-and-refactor").map(e => e.path);
    expect(result).not.toContain(".github/skills/other-skill/skill.md");
  });

  it("excludes files from unrelated directories", () => {
    const result = selectDeployableFiles(tree, "skills/review-and-refactor").map(e => e.path);
    expect(result).not.toContain(".github/agents/agent.md");
    expect(result).not.toContain("README.md");
  });

  it("returns empty array when no files match subpath", () => {
    expect(selectDeployableFiles(tree, "skills/nonexistent")).toHaveLength(0);
  });
});

// ---------------------------------------------------------------------------
// writeWorkspaceLockfile
// ---------------------------------------------------------------------------

describe("writeWorkspaceLockfile", () => {
  /** @type {string} */
  let tmpDir;

  beforeEach(() => {
    tmpDir = makeTempDir();
  });
  afterEach(() => removeTempDir(tmpDir));

  it("writes lockfile_version, generated_at, dependencies", () => {
    writeWorkspaceLockfile(tmpDir, []);
    const content = fs.readFileSync(path.join(tmpDir, "apm.lock.yaml"), "utf-8");
    expect(content).toContain("lockfile_version: '1'");
    expect(content).toContain("generated_at:");
    expect(content).toContain("dependencies:");
  });

  it("quotes ISO timestamp in generated_at so YAML parsers keep string type", () => {
    writeWorkspaceLockfile(tmpDir, []);
    const content = fs.readFileSync(path.join(tmpDir, "apm.lock.yaml"), "utf-8");
    expect(content).toMatch(/generated_at: '\d{4}-\d{2}-\d{2}T/);
  });

  it("writes one dependency entry", () => {
    writeWorkspaceLockfile(tmpDir, [
      {
        repo_url: "https://github.com/owner/pkg",
        resolved_commit: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
        resolved_ref: "main",
        deployed_files: [".github/skills/skill-a/skill.md"],
      },
    ]);
    const content = fs.readFileSync(path.join(tmpDir, "apm.lock.yaml"), "utf-8");
    expect(content).toContain("- repo_url: https://github.com/owner/pkg");
    expect(content).toContain("  resolved_commit: aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa");
    expect(content).toContain("  resolved_ref: main");
    expect(content).toContain("  - .github/skills/skill-a/skill.md");
  });

  it("round-trips through parseAPMLockfile from apm_unpack", () => {
    const { parseAPMLockfile } = require("./apm_unpack.cjs");
    writeWorkspaceLockfile(tmpDir, [
      {
        repo_url: "https://github.com/o/r",
        resolved_commit: "deadbeef",
        resolved_ref: "v1.0",
        deployed_files: [".github/skills/foo/skill.md"],
      },
    ]);
    const content = fs.readFileSync(path.join(tmpDir, "apm.lock.yaml"), "utf-8");
    const parsed = parseAPMLockfile(content);
    expect(parsed.lockfile_version).toBe("1");
    expect(parsed.dependencies).toHaveLength(1);
    expect(parsed.dependencies[0].repo_url).toBe("https://github.com/o/r");
    expect(parsed.dependencies[0].resolved_commit).toBe("deadbeef");
    expect(parsed.dependencies[0].deployed_files).toContain(".github/skills/foo/skill.md");
  });
});

// ---------------------------------------------------------------------------
// writeWorkspaceApmYml
// ---------------------------------------------------------------------------

describe("writeWorkspaceApmYml", () => {
  /** @type {string} */
  let tmpDir;

  beforeEach(() => {
    tmpDir = makeTempDir();
  });
  afterEach(() => removeTempDir(tmpDir));

  it("writes apm.yml with name and version", () => {
    writeWorkspaceApmYml(tmpDir);
    const content = fs.readFileSync(path.join(tmpDir, "apm.yml"), "utf-8");
    expect(content).toContain("name: gh-aw-workspace");
    expect(content).toContain("version: 0.0.0");
  });
});

// ---------------------------------------------------------------------------
// parsePackagesFromEnv
// ---------------------------------------------------------------------------

describe("parsePackagesFromEnv", () => {
  afterEach(() => {
    delete process.env.APM_PACKAGES;
  });

  it("returns empty array when APM_PACKAGES is not set", () => {
    delete process.env.APM_PACKAGES;
    expect(parsePackagesFromEnv()).toEqual([]);
  });

  it("parses a JSON array of package slugs", () => {
    process.env.APM_PACKAGES = '["microsoft/apm-sample-package","github/awesome-copilot/skills/foo"]';
    expect(parsePackagesFromEnv()).toEqual(["microsoft/apm-sample-package", "github/awesome-copilot/skills/foo"]);
  });

  it("filters out empty strings", () => {
    process.env.APM_PACKAGES = '["pkg-a","","pkg-b"]';
    expect(parsePackagesFromEnv()).toEqual(["pkg-a", "pkg-b"]);
  });

  it("throws on non-array JSON", () => {
    process.env.APM_PACKAGES = '"single-string"';
    expect(() => parsePackagesFromEnv()).toThrow(/JSON array/);
  });

  it("throws on malformed JSON", () => {
    process.env.APM_PACKAGES = "[not valid json";
    expect(() => parsePackagesFromEnv()).toThrow(/parse APM_PACKAGES/);
  });
});

// ---------------------------------------------------------------------------
// main() – mocked Octokit integration tests
// ---------------------------------------------------------------------------

describe("main() – mocked Octokit", () => {
  /** @type {string} */
  let tmpDir;

  beforeEach(() => {
    tmpDir = makeTempDir();
    vi.clearAllMocks();
  });
  afterEach(() => {
    removeTempDir(tmpDir);
    delete process.env.APM_PACKAGES;
  });

  it("writes apm.yml and empty lockfile when packages list is empty", async () => {
    await main({ octokitOverride: {}, workspaceDir: tmpDir, packages: [] });
    expect(fs.existsSync(path.join(tmpDir, "apm.yml"))).toBe(true);
    const lockContent = fs.readFileSync(path.join(tmpDir, "apm.lock.yaml"), "utf-8");
    expect(lockContent).toContain("lockfile_version: '1'");
  });

  it("downloads files for a single package and writes lockfile", async () => {
    const octokit = makeMockOctokit({
      commitSha: "abcdef1234567890abcdef1234567890abcdef12",
      tree: [
        { type: "blob", path: ".github/skills/test-skill/skill.md", sha: "s1" },
        { type: "blob", path: ".github/copilot-instructions.md", sha: "s2" },
        { type: "blob", path: "README.md", sha: "s3" }, // non-target, should be skipped
      ],
      contentMap: {
        ".github/skills/test-skill/skill.md": "# Test Skill",
        ".github/copilot-instructions.md": "# Instructions",
      },
    });

    await main({ octokitOverride: octokit, workspaceDir: tmpDir, packages: ["test-org/test-pkg"] });

    expect(fs.existsSync(path.join(tmpDir, ".github/skills/test-skill/skill.md"))).toBe(true);
    expect(fs.existsSync(path.join(tmpDir, ".github/copilot-instructions.md"))).toBe(true);
    expect(fs.existsSync(path.join(tmpDir, "README.md"))).toBe(false);

    const lockContent = fs.readFileSync(path.join(tmpDir, "apm.lock.yaml"), "utf-8");
    expect(lockContent).toContain("https://github.com/test-org/test-pkg");
    expect(lockContent).toContain("abcdef1234567890abcdef1234567890abcdef12");
    expect(lockContent).toContain(".github/skills/test-skill/skill.md");
  });

  it("skips path-traversal entries from the GitHub tree", async () => {
    const octokit = makeMockOctokit({
      tree: [
        { type: "blob", path: ".github/../../../etc/passwd", sha: "evil" },
        { type: "blob", path: ".github/skills/safe/skill.md", sha: "safe" },
      ],
      contentMap: {
        ".github/skills/safe/skill.md": "safe content",
      },
    });

    await main({ octokitOverride: octokit, workspaceDir: tmpDir, packages: ["attacker/repo"] });

    expect(fs.existsSync(path.join(tmpDir, ".github/skills/safe/skill.md"))).toBe(true);
    expect(fs.existsSync(path.join(tmpDir, "etc/passwd"))).toBe(false);
  });

  it("lockfile is valid input for apm_pack (round-trip fixture)", async () => {
    const { parseAPMLockfile } = require("./apm_unpack.cjs");

    const octokit = makeMockOctokit({
      commitSha: "cccc0000000000000000000000000000cccccccc",
      tree: [
        { type: "blob", path: ".github/skills/my-skill/skill.md", sha: "f1" },
        { type: "blob", path: ".claude/agents/my-agent.md", sha: "f2" },
      ],
      contentMap: {
        ".github/skills/my-skill/skill.md": "# My Skill",
        ".claude/agents/my-agent.md": "# My Agent",
      },
    });

    await main({ octokitOverride: octokit, workspaceDir: tmpDir, packages: ["my-org/my-pkg"] });

    const lockContent = fs.readFileSync(path.join(tmpDir, "apm.lock.yaml"), "utf-8");
    const parsed = parseAPMLockfile(lockContent);

    expect(parsed.lockfile_version).toBe("1");
    expect(parsed.dependencies).toHaveLength(1);
    expect(parsed.dependencies[0].deployed_files).toContain(".github/skills/my-skill/skill.md");
    expect(parsed.dependencies[0].deployed_files).toContain(".claude/agents/my-agent.md");

    // Verify installed files are actually present on disk
    expect(fs.existsSync(path.join(tmpDir, ".github/skills/my-skill/skill.md"))).toBe(true);
    expect(fs.existsSync(path.join(tmpDir, ".claude/agents/my-agent.md"))).toBe(true);
  });
});
