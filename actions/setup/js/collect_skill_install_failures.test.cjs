import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import fs from "fs";
import { createRequire } from "module";

const require = createRequire(import.meta.url);
const script = require("./collect_skill_install_failures.cjs");

describe("collect_skill_install_failures", () => {
  let originalCore;

  beforeEach(() => {
    originalCore = global.core;
    global.core = {
      info: vi.fn(),
      warning: vi.fn(),
      setOutput: vi.fn(),
    };
  });

  afterEach(() => {
    global.core = originalCore;
    fs.rmSync("/tmp/gh-aw/skill_install_failures.json", { force: true });
  });

  it("returns empty array when failures file is missing", () => {
    expect(script.readSkillInstallFailures()).toEqual([]);
  });

  it("filters malformed entries and emits sanitized outputs", async () => {
    fs.mkdirSync("/tmp/gh-aw", { recursive: true });
    fs.writeFileSync("/tmp/gh-aw/skill_install_failures.json", JSON.stringify([{ skill: "owner/repo", error: "line1\nline2" }, { skill: "missing-error" }]), "utf8");

    await script.main();

    expect(global.core.setOutput).toHaveBeenCalledWith("failure_count", "1");
    expect(global.core.setOutput).toHaveBeenCalledWith("errors", "owner/repo\tline1 line2");
  });

  it("warns and returns empty array on unreadable failures file", () => {
    fs.mkdirSync("/tmp/gh-aw", { recursive: true });
    fs.writeFileSync("/tmp/gh-aw/skill_install_failures.json", "{invalid", "utf8");

    expect(script.readSkillInstallFailures()).toEqual([]);
    expect(global.core.warning).toHaveBeenCalledWith(expect.stringContaining("Could not read skill install failures file"));
  });
});
