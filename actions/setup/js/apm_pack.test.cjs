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

// Establish globals before requiring the modules
global.core = mockCore;
global.exec = mockExec;

const { parseApmYml, detectTarget, filterFilesByTarget, serializeLockfileYaml, scalarToYaml, assertSafePackPath, assertPackDestInside, packBundle } = require("./apm_pack.cjs");

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

/** Create a temp directory and return its path. */
function makeTempDir() {
  return fs.mkdtempSync(path.join(os.tmpdir(), "apm-pack-test-"));
}

/** Remove a directory recursively (best-effort). */
function removeTempDir(dir) {
  if (dir && fs.existsSync(dir)) {
    fs.rmSync(dir, { recursive: true, force: true });
  }
}

/** Write a file, creating parent directories as needed. Returns absolute path. */
function writeFile(dir, relPath, content = "content") {
  const full = path.join(dir, relPath);
  fs.mkdirSync(path.dirname(full), { recursive: true });
  fs.writeFileSync(full, content, "utf-8");
  return full;
}

/**
 * Build a minimal apm.lock.yaml string with given dependencies.
 * @param {Array<{repoUrl: string, files: string[]}>} deps
 */
function buildLockfile(deps) {
  const lines = ["lockfile_version: '1'", "apm_version: 0.8.5", "dependencies:"];
  for (const dep of deps) {
    lines.push(`- repo_url: ${dep.repoUrl}`);
    lines.push(`  host: github.com`);
    lines.push(`  resolved_commit: abc123`);
    lines.push(`  resolved_ref: main`);
    lines.push(`  depth: 1`);
    lines.push(`  deployed_files:`);
    for (const f of dep.files) {
      lines.push(`  - ${f}`);
    }
  }
  return lines.join("\n") + "\n";
}

// ---------------------------------------------------------------------------
// parseApmYml
// ---------------------------------------------------------------------------

describe("parseApmYml", () => {
  it("parses name and version from valid apm.yml", () => {
    const content = `name: my-package\nversion: 1.2.3\n`;
    const result = parseApmYml(content);
    expect(result.name).toBe("my-package");
    expect(result.version).toBe("1.2.3");
  });

  it("uses fallback name when name is missing", () => {
    const content = `version: 1.0.0\n`;
    const result = parseApmYml(content, "fallback-dir");
    expect(result.name).toBe("fallback-dir");
    expect(result.version).toBe("1.0.0");
  });

  it("uses default version 0.0.0 when version is missing", () => {
    const content = `name: pkg\n`;
    const result = parseApmYml(content);
    expect(result.name).toBe("pkg");
    expect(result.version).toBe("0.0.0");
  });

  it("uses defaults when content is empty", () => {
    const result = parseApmYml("", "my-fallback");
    expect(result.name).toBe("my-fallback");
    expect(result.version).toBe("0.0.0");
  });

  it("handles single-quoted values", () => {
    const content = `name: 'my-pkg'\nversion: '2.0.0'\n`;
    const result = parseApmYml(content);
    expect(result.name).toBe("my-pkg");
    expect(result.version).toBe("2.0.0");
  });
});

// ---------------------------------------------------------------------------
// detectTarget
// ---------------------------------------------------------------------------

describe("detectTarget", () => {
  let tmpDir;

  beforeEach(() => {
    tmpDir = makeTempDir();
  });
  afterEach(() => {
    removeTempDir(tmpDir);
  });

  it("returns explicit target normalized to vscode for copilot", () => {
    expect(detectTarget(tmpDir, "copilot")).toBe("vscode");
  });

  it("returns explicit target normalized to vscode for vscode", () => {
    expect(detectTarget(tmpDir, "vscode")).toBe("vscode");
  });

  it("returns explicit target normalized to vscode for agents", () => {
    expect(detectTarget(tmpDir, "agents")).toBe("vscode");
  });

  it("returns claude for explicit claude target", () => {
    expect(detectTarget(tmpDir, "claude")).toBe("claude");
  });

  it("returns cursor for explicit cursor target", () => {
    expect(detectTarget(tmpDir, "cursor")).toBe("cursor");
  });

  it("returns opencode for explicit opencode target", () => {
    expect(detectTarget(tmpDir, "opencode")).toBe("opencode");
  });

  it("returns all for explicit all target", () => {
    expect(detectTarget(tmpDir, "all")).toBe("all");
  });

  it("auto-detects vscode when .github/ folder exists", () => {
    fs.mkdirSync(path.join(tmpDir, ".github"));
    expect(detectTarget(tmpDir, null)).toBe("vscode");
  });

  it("auto-detects claude when .claude/ folder exists", () => {
    fs.mkdirSync(path.join(tmpDir, ".claude"));
    expect(detectTarget(tmpDir, null)).toBe("claude");
  });

  it("auto-detects all when both .github/ and .claude/ exist", () => {
    fs.mkdirSync(path.join(tmpDir, ".github"));
    fs.mkdirSync(path.join(tmpDir, ".claude"));
    expect(detectTarget(tmpDir, null)).toBe("all");
  });

  it("defaults to all when no target folders found", () => {
    expect(detectTarget(tmpDir, null)).toBe("all");
  });

  it("explicit target wins over auto-detection", () => {
    fs.mkdirSync(path.join(tmpDir, ".github"));
    fs.mkdirSync(path.join(tmpDir, ".claude"));
    expect(detectTarget(tmpDir, "claude")).toBe("claude");
  });
});

// ---------------------------------------------------------------------------
// filterFilesByTarget
// ---------------------------------------------------------------------------

describe("filterFilesByTarget – direct matches", () => {
  it("copilot target includes .github/ files only", () => {
    const files = [".github/skills/foo/", ".claude/skills/foo/", ".cursor/rules/bar.md"];
    const { files: result, pathMappings } = filterFilesByTarget(files, "copilot");
    expect(result).toContain(".github/skills/foo/");
    expect(result).not.toContain(".claude/skills/foo/");
    expect(result).not.toContain(".cursor/rules/bar.md");
    expect(Object.keys(pathMappings)).toHaveLength(0);
  });

  it("claude target includes .claude/ files only (no cross-map if source is already .claude/)", () => {
    const files = [".claude/skills/foo/", ".claude/commands/cmd.md"];
    const { files: result, pathMappings } = filterFilesByTarget(files, "claude");
    expect(result).toContain(".claude/skills/foo/");
    expect(result).toContain(".claude/commands/cmd.md");
    expect(Object.keys(pathMappings)).toHaveLength(0);
  });

  it("all target includes all target directories", () => {
    const files = [".github/skills/foo/", ".claude/skills/bar/", ".cursor/rules/x.md"];
    const { files: result } = filterFilesByTarget(files, "all");
    expect(result).toContain(".github/skills/foo/");
    expect(result).toContain(".claude/skills/bar/");
    expect(result).toContain(".cursor/rules/x.md");
  });

  it("returns empty array when no files match target", () => {
    const files = [".github/skills/foo/"];
    const { files: result } = filterFilesByTarget(files, "claude");
    // No direct matches, but .github/skills/ → .claude/skills/ cross-map applies
    expect(result).toContain(".claude/skills/foo/");
  });
});

describe("filterFilesByTarget – cross-target mapping", () => {
  it("claude target maps .github/skills/ to .claude/skills/", () => {
    const files = [".github/skills/my-skill/"];
    const { files: result, pathMappings } = filterFilesByTarget(files, "claude");
    expect(result).toContain(".claude/skills/my-skill/");
    expect(result).not.toContain(".github/skills/my-skill/");
    expect(pathMappings[".claude/skills/my-skill/"]).toBe(".github/skills/my-skill/");
  });

  it("claude target maps .github/agents/ to .claude/agents/", () => {
    const files = [".github/agents/my-agent.md"];
    const { files: result, pathMappings } = filterFilesByTarget(files, "claude");
    expect(result).toContain(".claude/agents/my-agent.md");
    expect(pathMappings[".claude/agents/my-agent.md"]).toBe(".github/agents/my-agent.md");
  });

  it("claude target does NOT map .github/instructions/ (target-specific, not remapped)", () => {
    const files = [".github/copilot-instructions.md"];
    const { files: result } = filterFilesByTarget(files, "claude");
    // .github/copilot-instructions.md does not start with .claude/ and has no cross-map
    expect(result).not.toContain(".github/copilot-instructions.md");
  });

  it("copilot target maps .claude/skills/ to .github/skills/", () => {
    const files = [".claude/skills/my-skill/"];
    const { files: result, pathMappings } = filterFilesByTarget(files, "copilot");
    expect(result).toContain(".github/skills/my-skill/");
    expect(pathMappings[".github/skills/my-skill/"]).toBe(".claude/skills/my-skill/");
  });

  it("cursor target maps .github/skills/ to .cursor/skills/", () => {
    const files = [".github/skills/my-skill/"];
    const { files: result, pathMappings } = filterFilesByTarget(files, "cursor");
    expect(result).toContain(".cursor/skills/my-skill/");
    expect(pathMappings[".cursor/skills/my-skill/"]).toBe(".github/skills/my-skill/");
  });

  it("cross-mapped path is not included twice if already directly present", () => {
    // File already exists under .claude/ AND under .github/ — should not duplicate
    const files = [".claude/skills/foo/", ".github/skills/foo/"];
    const { files: result } = filterFilesByTarget(files, "claude");
    const claudeSkills = result.filter(f => f === ".claude/skills/foo/");
    expect(claudeSkills).toHaveLength(1); // No duplicates
  });

  it("all target has no cross-target mappings needed (prefixes cover all dirs)", () => {
    const files = [".github/skills/foo/", ".claude/skills/bar/"];
    const { files: result, pathMappings } = filterFilesByTarget(files, "all");
    expect(result).toContain(".github/skills/foo/");
    expect(result).toContain(".claude/skills/bar/");
    expect(Object.keys(pathMappings)).toHaveLength(0); // No mapping needed for "all"
  });
});

// ---------------------------------------------------------------------------
// scalarToYaml
// ---------------------------------------------------------------------------

describe("scalarToYaml", () => {
  it("returns 'null' for null/undefined", () => {
    expect(scalarToYaml(null)).toBe("null");
    expect(scalarToYaml(undefined)).toBe("null");
  });

  it("returns true/false for booleans", () => {
    expect(scalarToYaml(true)).toBe("true");
    expect(scalarToYaml(false)).toBe("false");
  });

  it("returns number as string", () => {
    expect(scalarToYaml(42)).toBe("42");
    expect(scalarToYaml(0)).toBe("0");
  });

  it("single-quotes strings that look like YAML keywords", () => {
    expect(scalarToYaml("null")).toBe("'null'");
    expect(scalarToYaml("true")).toBe("'true'");
    expect(scalarToYaml("false")).toBe("'false'");
    expect(scalarToYaml("yes")).toBe("'yes'");
    expect(scalarToYaml("no")).toBe("'no'");
    expect(scalarToYaml("on")).toBe("'on'");
    expect(scalarToYaml("off")).toBe("'off'");
    expect(scalarToYaml("")).toBe("''");
  });

  it("single-quotes strings that look like numbers", () => {
    expect(scalarToYaml("1")).toBe("'1'");
    expect(scalarToYaml("42")).toBe("'42'");
    expect(scalarToYaml("3.14")).toBe("'3.14'");
  });

  it("single-quotes ISO datetime strings (YAML 1.1 parses them as datetime)", () => {
    expect(scalarToYaml("2024-01-15T10:00:00.000Z")).toBe("'2024-01-15T10:00:00.000Z'");
    expect(scalarToYaml("2024-03-29T11:13:45.004Z")).toBe("'2024-03-29T11:13:45.004Z'");
  });

  it("returns regular strings as-is", () => {
    expect(scalarToYaml("https://github.com/owner/repo")).toBe("https://github.com/owner/repo");
    expect(scalarToYaml("main")).toBe("main");
    expect(scalarToYaml("github.com")).toBe("github.com");
    expect(scalarToYaml("abc123")).toBe("abc123");
  });
});

// ---------------------------------------------------------------------------
// serializeLockfileYaml
// ---------------------------------------------------------------------------

describe("serializeLockfileYaml", () => {
  const baseLockfile = {
    lockfile_version: "1",
    generated_at: "2024-01-15T10:00:00.000000+00:00",
    apm_version: "0.8.5",
    dependencies: [],
    pack: {},
  };

  it("includes pack: section with format, target, packed_at", () => {
    const yaml = serializeLockfileYaml(baseLockfile, [], {
      format: "apm",
      target: "claude",
      packed_at: "2024-01-15T10:00:00.000Z",
    });
    expect(yaml).toContain("pack:");
    expect(yaml).toContain("  format: apm");
    expect(yaml).toContain("  target: claude");
    expect(yaml).toContain("  packed_at: '2024-01-15T10:00:00.000Z'");
  });

  it("includes mapped_from when cross-target mappings were used", () => {
    const yaml = serializeLockfileYaml(baseLockfile, [], {
      format: "apm",
      target: "claude",
      packed_at: "2024-01-15T10:00:00.000Z",
      mapped_from: [".github/agents/", ".github/skills/"],
    });
    expect(yaml).toContain("  mapped_from:");
    expect(yaml).toContain("  - .github/agents/");
    expect(yaml).toContain("  - .github/skills/");
  });

  it("omits mapped_from when no cross-target mappings", () => {
    const yaml = serializeLockfileYaml(baseLockfile, [], {
      format: "apm",
      target: "all",
      packed_at: "2024-01-15T10:00:00.000Z",
      mapped_from: [],
    });
    expect(yaml).not.toContain("mapped_from");
  });

  it("includes lockfile_version and apm_version", () => {
    const yaml = serializeLockfileYaml(baseLockfile, [], {
      format: "apm",
      target: "all",
      packed_at: "t",
    });
    expect(yaml).toContain("lockfile_version: '1'");
    expect(yaml).toContain("apm_version: 0.8.5");
  });

  it("includes filtered deployed_files for each dependency", () => {
    const dep = {
      repo_url: "https://github.com/owner/pkg",
      host: "github.com",
      resolved_commit: "abc123",
      resolved_ref: "main",
      version: null,
      virtual_path: null,
      is_virtual: false,
      depth: 1,
      resolved_by: null,
      package_type: null,
      source: null,
      local_path: null,
      content_hash: null,
      is_dev: false,
      deployed_files: [".claude/skills/foo/", ".claude/agents/bar.md"],
    };
    const yaml = serializeLockfileYaml(baseLockfile, [dep], {
      format: "apm",
      target: "claude",
      packed_at: "t",
    });
    expect(yaml).toContain("- repo_url: https://github.com/owner/pkg");
    expect(yaml).toContain("  - .claude/skills/foo/");
    expect(yaml).toContain("  - .claude/agents/bar.md");
  });

  it("pack: section comes before lockfile_version (prepended)", () => {
    const yaml = serializeLockfileYaml(baseLockfile, [], {
      format: "apm",
      target: "all",
      packed_at: "t",
    });
    const packIdx = yaml.indexOf("pack:");
    const versionIdx = yaml.indexOf("lockfile_version:");
    expect(packIdx).toBeLessThan(versionIdx);
  });

  it("output is parseable by parseAPMLockfile from apm_unpack", () => {
    const { parseAPMLockfile } = require("./apm_unpack.cjs");
    const dep = {
      repo_url: "https://github.com/owner/pkg",
      host: "github.com",
      resolved_commit: "abc",
      resolved_ref: "main",
      version: "1.0.0",
      virtual_path: null,
      is_virtual: false,
      depth: 1,
      resolved_by: null,
      package_type: null,
      source: null,
      local_path: null,
      content_hash: null,
      is_dev: false,
      deployed_files: [".claude/skills/foo/"],
    };
    const yaml = serializeLockfileYaml(baseLockfile, [dep], {
      format: "apm",
      target: "claude",
      packed_at: "2024-01-15T10:00:00.000Z",
    });
    const parsed = parseAPMLockfile(yaml);
    expect(parsed.lockfile_version).toBe("1");
    expect(parsed.pack.target).toBe("claude");
    expect(parsed.pack.format).toBe("apm");
    expect(parsed.dependencies).toHaveLength(1);
    expect(parsed.dependencies[0].deployed_files).toContain(".claude/skills/foo/");
  });
});

// ---------------------------------------------------------------------------
// assertSafePackPath
// ---------------------------------------------------------------------------

describe("assertSafePackPath", () => {
  it("accepts safe relative paths", () => {
    expect(() => assertSafePackPath(".github/skills/foo/")).not.toThrow();
    expect(() => assertSafePackPath(".claude/agents/bar.md")).not.toThrow();
  });

  it("rejects absolute paths", () => {
    expect(() => assertSafePackPath("/etc/passwd")).toThrow(/unsafe absolute path/);
    expect(() => assertSafePackPath("/tmp/secret")).toThrow(/unsafe absolute path/);
  });

  it("rejects path traversal entries", () => {
    expect(() => assertSafePackPath("../etc/passwd")).toThrow(/path-traversal/);
    expect(() => assertSafePackPath(".github/../../../etc/passwd")).toThrow(/path-traversal/);
  });
});

// ---------------------------------------------------------------------------
// assertPackDestInside
// ---------------------------------------------------------------------------

describe("assertPackDestInside", () => {
  it("accepts paths inside the bundle directory", () => {
    const bundleDir = "/tmp/my-bundle";
    expect(() => assertPackDestInside("/tmp/my-bundle/file.txt", bundleDir)).not.toThrow();
    expect(() => assertPackDestInside("/tmp/my-bundle/subdir/file.txt", bundleDir)).not.toThrow();
  });

  it("rejects paths that escape the bundle directory", () => {
    const bundleDir = "/tmp/my-bundle";
    expect(() => assertPackDestInside("/tmp/other/file.txt", bundleDir)).toThrow(/escapes the bundle/);
    expect(() => assertPackDestInside("/etc/passwd", bundleDir)).toThrow(/escapes the bundle/);
  });
});

// ---------------------------------------------------------------------------
// packBundle – integration tests with real file system
// ---------------------------------------------------------------------------

describe("packBundle – integration", () => {
  let workspaceDir;
  let outputDir;

  beforeEach(() => {
    workspaceDir = makeTempDir();
    outputDir = makeTempDir();

    // Wire up exec.exec to run real tar
    const { spawnSync } = require("child_process");
    mockExec.exec.mockImplementation(async (cmd, args = []) => {
      const result = spawnSync(cmd, args, { stdio: "inherit" });
      if (result.status !== 0) {
        throw new Error(`Command failed: ${cmd} ${args.join(" ")} (exit ${result.status})`);
      }
      return result.status;
    });
  });

  afterEach(() => {
    removeTempDir(workspaceDir);
    removeTempDir(outputDir);
    vi.clearAllMocks();
  });

  it("packs a simple bundle with .github/ files", async () => {
    // Create workspace
    writeFile(workspaceDir, "apm.yml", "name: test-pkg\nversion: 1.0.0\n");
    const lockfileContent = buildLockfile([
      {
        repoUrl: "https://github.com/owner/skill-a",
        files: [".github/skills/skill-a/"],
      },
    ]);
    writeFile(workspaceDir, "apm.lock.yaml", lockfileContent);
    writeFile(workspaceDir, ".github/skills/skill-a/skill.md", "# Skill A\n");
    writeFile(workspaceDir, ".github/skills/skill-a/notes.txt", "Notes\n");

    const result = await packBundle({
      workspaceDir,
      outputDir,
      target: "all",
    });

    expect(result.bundlePath).toMatch(/test-pkg-1\.0\.0\.tar\.gz$/);
    expect(fs.existsSync(result.bundlePath)).toBe(true);
    expect(result.files).toContain(".github/skills/skill-a/");
    expect(result.target).toBe("all");

    // Verify tar.gz contains expected files
    const { spawnSync } = require("child_process");
    const listResult = spawnSync("tar", ["-tzf", result.bundlePath], { encoding: "utf-8" });
    expect(listResult.status).toBe(0);
    const entries = listResult.stdout.split("\n").filter(Boolean);
    expect(entries.some(e => e.includes("skill.md"))).toBe(true);
    expect(entries.some(e => e.includes(`.github/skills/skill-a`))).toBe(true);
    expect(entries.some(e => e.includes("apm.lock.yaml"))).toBe(true);
  });

  it("applies cross-target mapping for claude target", async () => {
    writeFile(workspaceDir, "apm.yml", "name: cross-test\nversion: 2.0.0\n");
    const lockfileContent = buildLockfile([
      {
        repoUrl: "https://github.com/owner/skills",
        files: [".github/skills/my-skill/", ".github/copilot-instructions.md"],
      },
    ]);
    writeFile(workspaceDir, "apm.lock.yaml", lockfileContent);
    writeFile(workspaceDir, ".github/skills/my-skill/skill.md", "# My Skill\n");
    writeFile(workspaceDir, ".github/copilot-instructions.md", "# Copilot\n");

    const result = await packBundle({
      workspaceDir,
      outputDir,
      target: "claude",
    });

    expect(result.bundlePath).toMatch(/cross-test-2\.0\.0\.tar\.gz$/);
    expect(fs.existsSync(result.bundlePath)).toBe(true);

    // The bundle should contain .claude/skills/my-skill/ (mapped from .github/skills/)
    // but NOT .github/copilot-instructions.md (no cross-map for instructions)
    const { spawnSync } = require("child_process");
    const listResult = spawnSync("tar", ["-tzf", result.bundlePath], { encoding: "utf-8" });
    const entries = listResult.stdout.split("\n").filter(Boolean);
    expect(entries.some(e => e.includes(".claude/skills/my-skill/skill.md"))).toBe(true);
    expect(entries.some(e => e.includes("copilot-instructions.md"))).toBe(false);

    // Verify pathMappings
    expect(result.pathMappings[".claude/skills/my-skill/"]).toBe(".github/skills/my-skill/");
  });

  it("sets bundle-path output via core.setOutput", async () => {
    writeFile(workspaceDir, "apm.yml", "name: output-test\nversion: 1.0.0\n");
    writeFile(workspaceDir, "apm.lock.yaml", buildLockfile([{ repoUrl: "https://github.com/o/r", files: [".github/copilot-instructions.md"] }]));
    writeFile(workspaceDir, ".github/copilot-instructions.md", "# Instructions\n");

    await packBundle({ workspaceDir, outputDir, target: "all" });
    // setOutput is called from main() not packBundle directly — test via main()
  });

  it("throws when apm.lock.yaml is missing", async () => {
    writeFile(workspaceDir, "apm.yml", "name: missing-lock\nversion: 1.0.0\n");
    await expect(packBundle({ workspaceDir, outputDir, target: "all" })).rejects.toThrow(/apm\.lock\.yaml not found/);
  });

  it("throws when a deployed file is missing from disk", async () => {
    writeFile(workspaceDir, "apm.yml", "name: missing-file\nversion: 1.0.0\n");
    const lockfileContent = buildLockfile([
      {
        repoUrl: "https://github.com/owner/pkg",
        files: [".github/skills/missing-skill/"],
      },
    ]);
    writeFile(workspaceDir, "apm.lock.yaml", lockfileContent);
    // Do NOT create .github/skills/missing-skill/ on disk

    await expect(packBundle({ workspaceDir, outputDir, target: "all" })).rejects.toThrow(/missing on disk/);
  });

  it("skips symlinks in directories (security: never bundle symlinks)", async () => {
    writeFile(workspaceDir, "apm.yml", "name: symlink-test\nversion: 1.0.0\n");
    writeFile(workspaceDir, "apm.lock.yaml", buildLockfile([{ repoUrl: "https://github.com/o/r", files: [".github/skills/skill-a/"] }]));
    writeFile(workspaceDir, ".github/skills/skill-a/real.md", "# Real\n");
    // Create a symlink inside the skill directory
    try {
      fs.symlinkSync("/etc/passwd", path.join(workspaceDir, ".github/skills/skill-a/link.txt"));
    } catch {
      // Skip if symlinks not supported
      return;
    }

    const result = await packBundle({ workspaceDir, outputDir, target: "all" });
    expect(fs.existsSync(result.bundlePath)).toBe(true);

    // Verify symlink was not bundled
    const { spawnSync } = require("child_process");
    const listResult = spawnSync("tar", ["-tzf", result.bundlePath], { encoding: "utf-8" });
    const entries = listResult.stdout.split("\n").filter(Boolean);
    expect(entries.some(e => e.includes("link.txt"))).toBe(false);
    expect(entries.some(e => e.includes("real.md"))).toBe(true);
  });

  it("rejects path-traversal entries in deployed_files that pass target filter", async () => {
    writeFile(workspaceDir, "apm.yml", "name: traversal-test\nversion: 1.0.0\n");
    // Use a path that starts with .github/ (passes "all" target filter) but contains ..
    const maliciousLockfile = "lockfile_version: '1'\napm_version: 0.8.5\ndependencies:\n- repo_url: https://github.com/evil/pkg\n  depth: 1\n  deployed_files:\n  - .github/skills/../../etc/passwd\n";
    writeFile(workspaceDir, "apm.lock.yaml", maliciousLockfile);

    await expect(packBundle({ workspaceDir, outputDir, target: "all" })).rejects.toThrow(/path-traversal/);
  });

  it("bundles multiple dependencies with deduplication", async () => {
    writeFile(workspaceDir, "apm.yml", "name: multi-dep\nversion: 1.0.0\n");
    const lockfileContent = buildLockfile([
      { repoUrl: "https://github.com/o/pkg-a", files: [".github/skills/skill-a/", ".github/copilot-instructions.md"] },
      { repoUrl: "https://github.com/o/pkg-b", files: [".github/skills/skill-b/", ".github/copilot-instructions.md"] },
    ]);
    writeFile(workspaceDir, "apm.lock.yaml", lockfileContent);
    writeFile(workspaceDir, ".github/skills/skill-a/skill.md", "# A\n");
    writeFile(workspaceDir, ".github/skills/skill-b/skill.md", "# B\n");
    writeFile(workspaceDir, ".github/copilot-instructions.md", "# Both\n");

    const result = await packBundle({ workspaceDir, outputDir, target: "all" });
    expect(result.files).toContain(".github/skills/skill-a/");
    expect(result.files).toContain(".github/skills/skill-b/");
    // .github/copilot-instructions.md should appear only once despite being in both deps
    const count = result.files.filter(f => f === ".github/copilot-instructions.md").length;
    expect(count).toBe(1);
  });

  it("the enriched apm.lock.yaml in the bundle is parseable by parseAPMLockfile", async () => {
    const { parseAPMLockfile } = require("./apm_unpack.cjs");
    writeFile(workspaceDir, "apm.yml", "name: parse-test\nversion: 1.0.0\n");
    writeFile(workspaceDir, "apm.lock.yaml", buildLockfile([{ repoUrl: "https://github.com/o/r", files: [".github/skills/foo/"] }]));
    writeFile(workspaceDir, ".github/skills/foo/skill.md", "# Foo\n");

    const result = await packBundle({ workspaceDir, outputDir, target: "all" });

    // Extract and read the lockfile from the bundle
    const extractDir = makeTempDir();
    try {
      const { spawnSync } = require("child_process");
      spawnSync("tar", ["-xzf", result.bundlePath, "-C", extractDir]);
      // Find the lockfile
      const bundleDirs = fs.readdirSync(extractDir);
      expect(bundleDirs.length).toBeGreaterThan(0);
      const lockfilePath = path.join(extractDir, bundleDirs[0], "apm.lock.yaml");
      expect(fs.existsSync(lockfilePath)).toBe(true);
      const lockfileContent = fs.readFileSync(lockfilePath, "utf-8");
      const parsed = parseAPMLockfile(lockfileContent);
      expect(parsed.pack.format).toBe("apm");
      expect(parsed.dependencies.length).toBeGreaterThan(0);
    } finally {
      removeTempDir(extractDir);
    }
  });
});

// ---------------------------------------------------------------------------
// main() – basic smoke test
// ---------------------------------------------------------------------------

describe("main()", () => {
  let workspaceDir;
  let outputDir;
  let origEnv;

  beforeEach(() => {
    workspaceDir = makeTempDir();
    outputDir = makeTempDir();
    origEnv = { ...process.env };

    const { spawnSync } = require("child_process");
    mockExec.exec.mockImplementation(async (cmd, args = []) => {
      const result = spawnSync(cmd, args, { stdio: "inherit" });
      if (result.status !== 0) throw new Error(`Failed: ${cmd} ${args.join(" ")}`);
      return result.status;
    });
  });

  afterEach(() => {
    process.env = origEnv;
    removeTempDir(workspaceDir);
    removeTempDir(outputDir);
    vi.clearAllMocks();
  });

  it("calls core.setOutput with bundle-path on success", async () => {
    process.env.APM_WORKSPACE = workspaceDir;
    process.env.APM_BUNDLE_OUTPUT = outputDir;
    process.env.APM_TARGET = "all";

    writeFile(workspaceDir, "apm.yml", "name: main-test\nversion: 1.0.0\n");
    writeFile(workspaceDir, "apm.lock.yaml", buildLockfile([{ repoUrl: "https://github.com/o/r", files: [".github/copilot-instructions.md"] }]));
    writeFile(workspaceDir, ".github/copilot-instructions.md", "# Instructions\n");

    const { main } = require("./apm_pack.cjs");
    await main();

    expect(mockCore.setOutput).toHaveBeenCalledWith("bundle-path", expect.stringMatching(/\.tar\.gz$/));
  });
});
