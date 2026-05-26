// @ts-check
/// <reference types="@actions/github-script" />

import { describe, it, expect, beforeEach, afterEach } from "vitest";
const fs = require("fs");
const path = require("path");
const os = require("os");

// Provide a minimal core mock so the module loads correctly.
global.core = {
  info: () => {},
  warning: () => {},
  error: () => {},
  setFailed: () => {},
};

const { extractInlineSkills, writeInlineSkills, filterInlineSkillFrontmatter } = require("./extract_inline_skills.cjs");

// Helper: returns a ## skill: `name` start marker line.
const skillMarker = name => `## skill: \`${name}\``;

// ─────────────────────────────────────────────────────────────────────────────
// extractInlineSkills — unit tests
// ─────────────────────────────────────────────────────────────────────────────

describe("extractInlineSkills", () => {
  it("returns original content unchanged when no markers present", () => {
    const content = "# Hello\n\nThis is a workflow.";
    const { mainContent, skills } = extractInlineSkills(content);
    expect(mainContent).toBe(content);
    expect(skills).toHaveLength(0);
  });

  it("returns empty main content and no skills for empty string", () => {
    const { mainContent, skills } = extractInlineSkills("");
    expect(mainContent).toBe("");
    expect(skills).toHaveLength(0);
  });

  it("extracts a single skill block", () => {
    const content = ["# Main workflow", "", "Handle the issue.", "", skillMarker("planner"), "---", "engine: copilot", "---", "You are a planning assistant."].join("\n");

    const { mainContent, skills } = extractInlineSkills(content);

    expect(mainContent).toBe("# Main workflow\n\nHandle the issue.");
    expect(skills).toHaveLength(1);
    expect(skills[0].name).toBe("planner");
    expect(skills[0].content).toContain("You are a planning assistant.");
    expect(skills[0].content).toContain("engine: copilot");
  });

  it("extracts multiple skill blocks", () => {
    const content = ["Main prompt.", "", skillMarker("planner"), "Planner prompt.", "", skillMarker("executor"), "Executor prompt."].join("\n");

    const { mainContent, skills } = extractInlineSkills(content);

    expect(mainContent).toBe("Main prompt.");
    expect(skills).toHaveLength(2);
    expect(skills[0].name).toBe("planner");
    expect(skills[0].content).toBe("Planner prompt.");
    expect(skills[1].name).toBe("executor");
    expect(skills[1].content).toBe("Executor prompt.");
  });

  it("skill block ends at next H2 heading", () => {
    const content = ["Main prompt.", "", skillMarker("planner"), "Planner content.", "", "## Summary", "This content is outside the skill block."].join("\n");

    const { mainContent, skills } = extractInlineSkills(content);

    expect(mainContent).toBe("Main prompt.");
    expect(skills).toHaveLength(1);
    expect(skills[0].name).toBe("planner");
    expect(skills[0].content).toBe("Planner content.");
    expect(skills[0].content).not.toContain("Summary");
    expect(skills[0].content).not.toContain("outside the skill block");
  });

  it("next skill marker (H2) ends the previous skill block", () => {
    const content = ["Main.", "", skillMarker("planner"), "Planner.", "", skillMarker("executor"), "Executor."].join("\n");

    const { skills } = extractInlineSkills(content);

    expect(skills).toHaveLength(2);
    expect(skills[0].content).toBe("Planner.");
    expect(skills[1].content).toBe("Executor.");
  });

  it("skill at start of file produces empty main content", () => {
    const content = skillMarker("only") + "\nSkill content.";
    const { mainContent, skills } = extractInlineSkills(content);
    expect(mainContent).toBe("");
    expect(skills).toHaveLength(1);
    expect(skills[0].name).toBe("only");
  });

  it("skill content is trimmed", () => {
    const content = "Main.\n\n" + skillMarker("a") + "\n\n\n  Trimmed.  \n\n";
    const { skills } = extractInlineSkills(content);
    expect(skills[0].content).toBe("Trimmed.");
  });

  it("trailing newlines are stripped from main content", () => {
    const content = "Line 1.\nLine 2.\n\n\n" + skillMarker("a") + "\nContent.";
    const { mainContent } = extractInlineSkills(content);
    expect(mainContent).toBe("Line 1.\nLine 2.");
  });

  it("accepts valid lowercase name variants", () => {
    const cases = [{ name: "my-skill" }, { name: "my_skill" }, { name: "skill1" }, { name: "a" }, { name: "planner-v2" }];
    for (const { name } of cases) {
      const { skills } = extractInlineSkills("Main.\n\n" + skillMarker(name) + "\nContent.");
      expect(skills).toHaveLength(1);
      expect(skills[0].name).toBe(name);
    }
  });

  it("does not recognize invalid separator forms", () => {
    const invalids = ["## skill: `1skill`", "## skill: `my skill`", "## skill: `my/skill`", "## skill:", "## skill: myskill", "## skill: `MySkill`", "# skill: `myskill`", "### skill: `myskill`"];
    for (const sep of invalids) {
      const content = `Main.\n\n${sep}\nContent.`;
      const { mainContent, skills } = extractInlineSkills(content);
      expect(mainContent).toBe(content);
      expect(skills).toHaveLength(0);
    }
  });
});

// ─────────────────────────────────────────────────────────────────────────────
// writeInlineSkills — integration tests (real filesystem)
// ─────────────────────────────────────────────────────────────────────────────

describe("writeInlineSkills", () => {
  let tmpDir;

  beforeEach(() => {
    tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), "inline-skills-test-"));
  });

  afterEach(() => {
    if (fs.existsSync(tmpDir)) {
      fs.rmSync(tmpDir, { recursive: true, force: true });
    }
  });

  it("returns original content unchanged when no markers present", () => {
    const content = "# Workflow\n\nNo skills here.";
    const result = writeInlineSkills(content, tmpDir);
    expect(result).toBe(content);
    const skillsDir = path.join(tmpDir, ".github", "skills");
    expect(fs.existsSync(skillsDir)).toBe(false);
  });

  it("writes a single skill file and returns main content", () => {
    const content = ["# Workflow", "", "Main prompt.", "", skillMarker("helper"), "---", "description: A helper skill", "---", "You are a helper."].join("\n");

    const result = writeInlineSkills(content, tmpDir);

    expect(result).toBe("# Workflow\n\nMain prompt.");

    const skillPath = path.join(tmpDir, ".github", "skills", "helper/SKILL.md");
    expect(fs.existsSync(skillPath)).toBe(true);
    const written = fs.readFileSync(skillPath, "utf8");
    expect(written).toContain("You are a helper.");
    expect(written).toContain("description: A helper skill");
  });

  it("writes multiple skill files", () => {
    const content = ["Main.", "", skillMarker("planner"), "Planner.", "", skillMarker("executor"), "Executor."].join("\n");

    writeInlineSkills(content, tmpDir);

    expect(fs.existsSync(path.join(tmpDir, ".github", "skills", "planner/SKILL.md"))).toBe(true);
    expect(fs.existsSync(path.join(tmpDir, ".github", "skills", "executor/SKILL.md"))).toBe(true);
  });

  it("skill file content ends with a newline", () => {
    const content = "Main.\n\n" + skillMarker("a") + "\nContent without trailing newline";
    writeInlineSkills(content, tmpDir);
    const written = fs.readFileSync(path.join(tmpDir, ".github", "skills", "a/SKILL.md"), "utf8");
    expect(written.endsWith("\n")).toBe(true);
  });

  it("creates .github/skills directory if it does not exist", () => {
    const content = "Main.\n\n" + skillMarker("new") + "\nContent.";
    const skillsDir = path.join(tmpDir, ".github", "skills");
    expect(fs.existsSync(skillsDir)).toBe(false);
    writeInlineSkills(content, tmpDir);
    expect(fs.existsSync(skillsDir)).toBe(true);
  });

  it("skill block ends at H2 — content after is not written to skill file", () => {
    const content = ["Main.", "", skillMarker("a"), "Skill body.", "", "## Notes", "Footer content that should not appear in the skill file."].join("\n");

    const result = writeInlineSkills(content, tmpDir);

    expect(result).toBe("Main.");
    const written = fs.readFileSync(path.join(tmpDir, ".github", "skills", "a/SKILL.md"), "utf8");
    expect(written).toContain("Skill body.");
    expect(written).not.toContain("Footer content");
  });

  it("strips unsupported frontmatter fields when writing skill file", () => {
    const content = ["Main.", "", skillMarker("a"), "---", "engine: copilot", "description: A helper", "tools:", "  github:", "    toolsets: [issues]", "---", "Skill prompt."].join("\n");

    writeInlineSkills(content, tmpDir);

    const written = fs.readFileSync(path.join(tmpDir, ".github", "skills", "a/SKILL.md"), "utf8");
    expect(written).toContain("description: A helper");
    expect(written).not.toContain("engine:");
    expect(written).not.toContain("tools:");
    expect(written).toContain("Skill prompt.");
  });

  it("writes only description when mixed with unsupported fields", () => {
    const content = ["Main.", "", skillMarker("a"), "---", "description: A helpful skill", "engine: openai", "---", "Prompt."].join("\n");

    writeInlineSkills(content, tmpDir);

    const written = fs.readFileSync(path.join(tmpDir, ".github", "skills", "a/SKILL.md"), "utf8");
    expect(written).toContain("description: A helpful skill");
    expect(written).not.toContain("engine:");
  });
});

// ─────────────────────────────────────────────────────────────────────────────
// filterInlineSkillFrontmatter — unit tests
// ─────────────────────────────────────────────────────────────────────────────

describe("filterInlineSkillFrontmatter", () => {
  it("returns content unchanged when no frontmatter present", () => {
    const content = "Just a plain prompt.";
    expect(filterInlineSkillFrontmatter(content, "skill")).toBe(content);
  });

  it("keeps description field", () => {
    const content = "---\ndescription: A planner\n---\nPrompt.";
    const result = filterInlineSkillFrontmatter(content, "skill");
    expect(result).toContain("description: A planner");
    expect(result).toContain("Prompt.");
  });

  it("strips unsupported fields and keeps supported ones", () => {
    const content = "---\nengine: copilot\ndescription: Helper\n---\nPrompt.";
    const result = filterInlineSkillFrontmatter(content, "skill");
    expect(result).toContain("description: Helper");
    expect(result).not.toContain("engine:");
  });

  it("omits frontmatter entirely when no supported fields remain", () => {
    const content = "---\nengine: copilot\ntools:\n  github:\n    toolsets: [issues]\n---\nPrompt.";
    const result = filterInlineSkillFrontmatter(content, "skill");
    expect(result).not.toContain("---");
    expect(result).toContain("Prompt.");
  });

  it("returns content unchanged when no closing delimiter found", () => {
    const content = "---\nengine: copilot\nPrompt without closing delimiter.";
    expect(filterInlineSkillFrontmatter(content, "skill")).toBe(content);
  });

  it("drops frontmatter when only unsupported fields are present", () => {
    const content = "---\nmodel: gpt-4o\n---\nYou are a summarizer.";
    const result = filterInlineSkillFrontmatter(content, "agent");
    expect(result).toBe("You are a summarizer.");
  });

  it("handles content with only description field", () => {
    const content = "---\ndescription: Summarizes files\n---\nYou are a summarizer.";
    const result = filterInlineSkillFrontmatter(content, "agent");
    expect(result).toBe("---\ndescription: Summarizes files\n---\nYou are a summarizer.");
  });
});
