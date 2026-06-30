// @ts-check
/// <reference types="@actions/github-script" />

const fs = require("fs");
const path = require("path");

/**
 * @param {string} rawSkills
 * @returns {string[]}
 */
function parseSkillSpecs(rawSkills) {
  return (rawSkills || "")
    .split(/\r?\n/)
    .map(skill => skill.trim())
    .filter(Boolean);
}

/**
 * @typedef {{args: string[]; displaySpec: string}} SkillInstallCommand
 */

/**
 * @param {string} skillSpec
 * @param {string} skillsDst
 * @returns {SkillInstallCommand}
 */
function buildSkillInstallCommand(skillSpec, skillsDst) {
  const atIndex = skillSpec.lastIndexOf("@");
  const hasPin = atIndex >= 0;
  const skillBase = hasPin ? skillSpec.slice(0, atIndex) : skillSpec;
  const skillRef = hasPin ? skillSpec.slice(atIndex + 1) : "";
  const parts = skillBase.split("/");
  const pinArgs = skillRef ? ["--pin", skillRef] : [];

  if (parts.length >= 3) {
    return {
      displaySpec: skillSpec,
      args: ["skill", "install", `${parts[0]}/${parts[1]}`, parts.slice(2).join("/"), ...pinArgs, "--dir", skillsDst, "--force"],
    };
  }

  if (parts.length === 2) {
    return {
      displaySpec: skillSpec,
      args: ["skill", "install", skillBase, "--all", ...pinArgs, "--dir", skillsDst, "--force"],
    };
  }

  return {
    displaySpec: skillSpec,
    args: ["skill", "install", skillSpec, "--dir", skillsDst, "--force"],
  };
}

/**
 * @param {string} skillsDst
 * @returns {number}
 */
function countInstalledSkillFiles(skillsDst) {
  if (!fs.existsSync(skillsDst)) {
    return 0;
  }

  let count = 0;
  const stack = [skillsDst];
  while (stack.length > 0) {
    const currentDir = stack.pop();
    if (!currentDir) {
      continue;
    }
    for (const entry of fs.readdirSync(currentDir, { withFileTypes: true })) {
      const entryPath = path.join(currentDir, entry.name);
      if (entry.isDirectory()) {
        stack.push(entryPath);
        continue;
      }
      if (entry.isFile() && entry.name === "SKILL.md") {
        count++;
      }
    }
  }

  return count;
}

/**
 * @param {string} skillDir
 * @param {string[]} skills
 * @param {number} installedSkillCount
 * @returns {Promise<void>}
 */
async function writeSkillSummary(skillDir, skills, installedSkillCount) {
  core.summary
    .addRaw("### Frontmatter skills installed\n\n")
    .addRaw(`- Engine skill directory: \`${skillDir}\`\n`)
    .addRaw(`- Requested references: \`${JSON.stringify(skills)}\`\n`)
    .addRaw(`- Installed SKILL.md files: ${installedSkillCount}\n`);
  await core.summary.write();
}

async function main() {
  const skillDir = process.env.GH_AW_SKILL_DIR || "";
  const skills = parseSkillSpecs(process.env.GH_AW_FRONTMATTER_SKILLS || "");
  const skillsDst = path.join("/tmp/gh-aw", skillDir);

  fs.mkdirSync(skillsDst, { recursive: true });

  core.info(`Installing frontmatter skills to ${skillsDst}`);
  core.info("Existing skills at destination may be replaced (--force) to ensure pinned refs are up to date");

  for (const skillSpec of skills) {
    core.info(`Installing skill reference: ${skillSpec}`);
    const command = buildSkillInstallCommand(skillSpec, skillsDst);
    await exec.exec("gh", command.args);
  }

  const installedSkillCount = countInstalledSkillFiles(skillsDst);
  core.info(`Installed ${installedSkillCount} skill file(s)`);
  await writeSkillSummary(skillDir, skills, installedSkillCount);
}

module.exports = { main, parseSkillSpecs, buildSkillInstallCommand, countInstalledSkillFiles };
