import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import fs from "node:fs";
import os from "node:os";
import path from "node:path";
import { createRequire } from "node:module";

const { inventoryDirectory, detectSkillConflicts, main } = createRequire(import.meta.url)("./skill_inventory.cjs");

describe("skill inventory", () => {
  let root;
  let oldWorkspace;
  let oldSkillDir;
  let oldCore;
  beforeEach(() => {
    root = fs.mkdtempSync(path.join(os.tmpdir(), "gh-aw-skills-"));
    oldWorkspace = process.env.GITHUB_WORKSPACE;
    oldSkillDir = process.env.GH_AW_SKILL_DIR;
    oldCore = global.core;
    global.core = { warning: vi.fn(), info: vi.fn() };
  });
  afterEach(() => {
    if (oldWorkspace === undefined) delete process.env.GITHUB_WORKSPACE;
    else process.env.GITHUB_WORKSPACE = oldWorkspace;
    if (oldSkillDir === undefined) delete process.env.GH_AW_SKILL_DIR;
    else process.env.GH_AW_SKILL_DIR = oldSkillDir;
    global.core = oldCore;
    fs.rmSync(root, { recursive: true, force: true });
    fs.rmSync("/tmp/gh-aw/skill_inventory.json", { force: true });
    fs.rmSync("/tmp/gh-aw/skill_install_history.json", { force: true });
  });

  it("records stable names, sources, scopes, digests and declared purposes without content", () => {
    const dir = path.join(root, "skills");
    fs.mkdirSync(path.join(dir, "review"), { recursive: true });
    fs.writeFileSync(path.join(dir, "review", "SKILL.md"), "---\ndescription: Review changes\n---\nSECRET_CONTENT\n");
    const skills = inventoryDirectory(dir, "repository", "skills");
    expect(skills).toEqual([{ name: "review", source: "skills/review", scope: "repository", digest: expect.stringMatching(/^sha256:[0-9a-f]{64}$/), purpose: "Review changes" }]);
    expect(JSON.stringify(skills)).not.toContain("SECRET_CONTENT");
    expect(inventoryDirectory(dir, "repository", "skills")).toEqual(skills);
  });

  it("reports exact-name and cross-scope collisions without treating similar names as conflicts", () => {
    const entries = [
      { name: "review", source: "skills/review", scope: "repository", digest: "a" },
      { name: "review", source: "staged/.claude/skills/review", scope: "engine", digest: "b" },
      { name: "review-security", source: "skills/review-security", scope: "repository", digest: "c" },
    ];
    expect(detectSkillConflicts(entries)).toEqual([{ name: "review", kind: "scope_collision", sources: ["skills/review", "staged/.claude/skills/review"] }]);
    expect(detectSkillConflicts([entries[0], { ...entries[0], source: "other/review" }])).toEqual([]);
  });

  it("writes activation evidence and warns for conflicting scopes", () => {
    process.env.GITHUB_WORKSPACE = root;
    process.env.GH_AW_SKILL_DIR = ".claude/skills";
    const repo = path.join(root, ".github/skills/review");
    const staged = "/tmp/gh-aw/.claude/skills/review";
    fs.mkdirSync(repo, { recursive: true });
    fs.mkdirSync(staged, { recursive: true });
    fs.writeFileSync(path.join(repo, "SKILL.md"), "# repo");
    fs.writeFileSync(path.join(staged, "SKILL.md"), "# staged");
    try {
      main();
      const evidence = JSON.parse(fs.readFileSync("/tmp/gh-aw/skill_inventory.json", "utf8"));
      expect(evidence.skills).toHaveLength(2);
      expect(evidence.conflicts).toMatchObject([{ name: "review", kind: "scope_collision" }]);
      expect(global.core.warning).toHaveBeenCalledTimes(1);
    } finally {
      fs.rmSync(staged, { recursive: true, force: true });
    }
  });

  it("reports overwritten installations with the same name and different digests", () => {
    process.env.GITHUB_WORKSPACE = root;
    process.env.GH_AW_SKILL_DIR = ".claude/skills";
    fs.writeFileSync(
      "/tmp/gh-aw/skill_install_history.json",
      JSON.stringify([
        { name: "review", source: "frontmatter-1", digest: "sha256:first" },
        { name: "review", source: "frontmatter-2", digest: "sha256:second" },
      ])
    );
    main();
    const evidence = JSON.parse(fs.readFileSync("/tmp/gh-aw/skill_inventory.json", "utf8"));
    expect(evidence.conflicts).toContainEqual({ name: "review", kind: "name_collision", sources: ["frontmatter-1", "frontmatter-2"] });
  });
});
