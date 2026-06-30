import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import fs from "fs";
import os from "os";
import path from "path";
import { createRequire } from "module";

const require = createRequire(import.meta.url);
const script = require("./install_frontmatter_skills.cjs");

describe("install_frontmatter_skills", () => {
  let originalEnv;
  let originalCore;
  let originalExec;
  let tempRoot;

  beforeEach(() => {
    originalEnv = {
      GH_AW_SKILL_DIR: process.env.GH_AW_SKILL_DIR,
      GH_AW_FRONTMATTER_SKILLS: process.env.GH_AW_FRONTMATTER_SKILLS,
    };
    originalCore = global.core;
    originalExec = global.exec;
    tempRoot = fs.mkdtempSync(path.join(os.tmpdir(), "gh-aw-frontmatter-skills-"));

    global.core = {
      info: vi.fn(),
      summary: {
        addRaw: vi.fn().mockReturnThis(),
        write: vi.fn().mockResolvedValue(undefined),
      },
    };
    global.exec = {
      exec: vi.fn().mockResolvedValue(0),
    };
  });

  afterEach(() => {
    if (originalEnv.GH_AW_SKILL_DIR === undefined) {
      delete process.env.GH_AW_SKILL_DIR;
    } else {
      process.env.GH_AW_SKILL_DIR = originalEnv.GH_AW_SKILL_DIR;
    }
    if (originalEnv.GH_AW_FRONTMATTER_SKILLS === undefined) {
      delete process.env.GH_AW_FRONTMATTER_SKILLS;
    } else {
      process.env.GH_AW_FRONTMATTER_SKILLS = originalEnv.GH_AW_FRONTMATTER_SKILLS;
    }
    global.core = originalCore;
    global.exec = originalExec;
    fs.rmSync(tempRoot, { recursive: true, force: true });
    fs.rmSync("/tmp/gh-aw/.claude", { recursive: true, force: true });
  });

  it("splits repo-level and path-level skill specs into gh skill install arguments", () => {
    expect(script.buildSkillInstallCommand("githubnext/skills@abc123", "/tmp/gh-aw/.claude/skills").args).toEqual(["skill", "install", "githubnext/skills", "--all", "--pin", "abc123", "--dir", "/tmp/gh-aw/.claude/skills", "--force"]);
    expect(script.buildSkillInstallCommand("githubnext/skills/review/security@abc123", "/tmp/gh-aw/.claude/skills").args).toEqual([
      "skill",
      "install",
      "githubnext/skills",
      "review/security",
      "--pin",
      "abc123",
      "--dir",
      "/tmp/gh-aw/.claude/skills",
      "--force",
    ]);
  });

  it("omits --pin when the resolved skill spec is unpinned", () => {
    expect(script.buildSkillInstallCommand("githubnext/skills/review/security", "/tmp/gh-aw/.claude/skills").args).toEqual(["skill", "install", "githubnext/skills", "review/security", "--dir", "/tmp/gh-aw/.claude/skills", "--force"]);
  });

  it("reads skill specs from the env var and installs them at runtime", async () => {
    process.env.GH_AW_SKILL_DIR = ".claude/skills";
    process.env.GH_AW_FRONTMATTER_SKILLS = ["githubnext/skills@abc123", "githubnext/skills/review/security@def456", "${{ inputs.skill_ref }}"].join("\n");
    fs.mkdirSync("/tmp/gh-aw/.claude/skills/example", { recursive: true });
    fs.writeFileSync("/tmp/gh-aw/.claude/skills/example/SKILL.md", "# test\n", "utf8");

    await script.main();

    expect(global.exec.exec).toHaveBeenNthCalledWith(1, "gh", ["skill", "install", "githubnext/skills", "--all", "--pin", "abc123", "--dir", "/tmp/gh-aw/.claude/skills", "--force"]);
    expect(global.exec.exec).toHaveBeenNthCalledWith(2, "gh", ["skill", "install", "githubnext/skills", "review/security", "--pin", "def456", "--dir", "/tmp/gh-aw/.claude/skills", "--force"]);
    expect(global.exec.exec).toHaveBeenNthCalledWith(3, "gh", ["skill", "install", "${{ inputs.skill_ref }}", "--dir", "/tmp/gh-aw/.claude/skills", "--force"]);
    expect(global.core.summary.addRaw).toHaveBeenCalledWith(expect.stringContaining("### Frontmatter skills installed"));
    expect(global.core.summary.addRaw).toHaveBeenCalledWith(expect.stringContaining('["githubnext/skills@abc123","githubnext/skills/review/security@def456","${{ inputs.skill_ref }}"]'));
  });
});
