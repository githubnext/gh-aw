import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import fs from "fs";
import path from "path";

const TMP_ROOT = "/tmp/gh-aw";
const THREAT_DIR = path.join(TMP_ROOT, "threat-detection");
const TEMPLATE_DIR = "/tmp/gh-aw-test-prompts";

describe("setup_threat_detection", () => {
  beforeEach(() => {
    vi.resetModules();
    fs.rmSync(TMP_ROOT, { recursive: true, force: true });
    fs.rmSync(TEMPLATE_DIR, { recursive: true, force: true });
    fs.mkdirSync(TEMPLATE_DIR, { recursive: true });

    fs.writeFileSync(
      path.join(TEMPLATE_DIR, "threat_detection.md"),
      "name={WORKFLOW_NAME}\ndescription={WORKFLOW_DESCRIPTION}\nprompt={WORKFLOW_PROMPT_FILE}\noutput={AGENT_OUTPUT_FILE}\ncomment={COMMENT_MEMORY_FILES}\npatch={AGENT_PATCH_FILE}\n"
    );

    fs.mkdirSync(THREAT_DIR, { recursive: true });
    fs.writeFileSync(path.join(THREAT_DIR, "agent_output.json"), '{"ok":true}');

    process.env.GH_AW_PROMPTS_DIR = TEMPLATE_DIR;
    process.env.WORKFLOW_NAME = "Test Workflow";
    process.env.WORKFLOW_DESCRIPTION = "Test Description";
  });

  afterEach(() => {
    fs.rmSync(TMP_ROOT, { recursive: true, force: true });
    fs.rmSync(TEMPLATE_DIR, { recursive: true, force: true });
    delete process.env.GH_AW_PROMPTS_DIR;
    delete process.env.WORKFLOW_NAME;
    delete process.env.WORKFLOW_DESCRIPTION;
    delete process.env.GH_AW_DETECTION_CONTINUE_ON_ERROR;
    delete process.env.HAS_PATCH;
    delete process.env.GH_AW_DETECTION_SKIP_PROMPT_SUMMARY;
  });

  function setupCoreMocks() {
    const summary = {
      addRaw: vi.fn().mockReturnThis(),
      write: vi.fn().mockResolvedValue(undefined),
    };
    global.core = {
      info: vi.fn(),
      warning: vi.fn(),
      error: vi.fn(),
      setFailed: vi.fn(),
      exportVariable: vi.fn(),
      summary,
    };
  }

  it("continues with fallback workflow context when prompt artifact is missing", async () => {
    setupCoreMocks();

    const module = await import("./setup_threat_detection.cjs");
    await module.main();

    expect(global.core.setFailed).not.toHaveBeenCalled();
    expect(global.core.warning).toHaveBeenCalledWith(expect.stringContaining("Missing workflow prompt context"));
    expect(global.core.exportVariable).toHaveBeenCalledWith("GH_AW_PROMPT", "/tmp/gh-aw/aw-prompts/prompt.txt");

    const generatedPromptPath = "/tmp/gh-aw/aw-prompts/prompt.txt";
    expect(fs.existsSync(generatedPromptPath)).toBe(true);
    const generatedPrompt = fs.readFileSync(generatedPromptPath, "utf8");
    expect(generatedPrompt).toContain("name=Test Workflow");
    expect(generatedPrompt).toContain("description=Test Description");
    expect(generatedPrompt).toContain("prompt=/tmp/gh-aw/threat-detection/aw-prompts/prompt.txt (unavailable)");
  });

  it("warns but continues when prompt artifact is empty", async () => {
    setupCoreMocks();
    const promptDir = path.join(THREAT_DIR, "aw-prompts");
    fs.mkdirSync(promptDir, { recursive: true });
    fs.writeFileSync(path.join(promptDir, "prompt.txt"), "");

    const module = await import("./setup_threat_detection.cjs");
    await module.main();

    expect(global.core.setFailed).not.toHaveBeenCalled();
    expect(global.core.warning).toHaveBeenCalledWith(expect.stringContaining("is empty"));
    expect(global.core.exportVariable).toHaveBeenCalledWith("GH_AW_PROMPT", "/tmp/gh-aw/aw-prompts/prompt.txt");
  });

  it("does not fail when agent output is missing and continue-on-error is enabled", async () => {
    setupCoreMocks();
    fs.rmSync(path.join(THREAT_DIR, "agent_output.json"), { force: true });
    process.env.GH_AW_DETECTION_CONTINUE_ON_ERROR = "TRUE";

    const module = await import("./setup_threat_detection.cjs");
    await module.main();

    expect(global.core.setFailed).not.toHaveBeenCalled();
    expect(global.core.warning).toHaveBeenCalledWith(expect.stringContaining("Continuing because GH_AW_DETECTION_CONTINUE_ON_ERROR=true"));
  });

  it("fails when agent output is missing and continue-on-error is disabled", async () => {
    setupCoreMocks();
    fs.rmSync(path.join(THREAT_DIR, "agent_output.json"), { force: true });
    process.env.GH_AW_DETECTION_CONTINUE_ON_ERROR = "false";

    const module = await import("./setup_threat_detection.cjs");
    await module.main();

    expect(global.core.setFailed).toHaveBeenCalledWith(expect.stringContaining("Agent output file not found"));
  });

  it("warns but continues when patch is missing and continue-on-error is enabled", async () => {
    setupCoreMocks();
    process.env.HAS_PATCH = "true";
    process.env.GH_AW_DETECTION_CONTINUE_ON_ERROR = "true";

    const module = await import("./setup_threat_detection.cjs");
    await module.main();

    expect(global.core.setFailed).not.toHaveBeenCalled();
    expect(global.core.warning).toHaveBeenCalledWith(expect.stringContaining("Continuing because GH_AW_DETECTION_CONTINUE_ON_ERROR=true"));
  });

  it("writes the rendered prompt to the step summary by default", async () => {
    setupCoreMocks();

    const module = await import("./setup_threat_detection.cjs");
    await module.main();

    expect(global.core.summary.addRaw).toHaveBeenCalledWith(expect.stringContaining("Threat Detection Prompt"));
    expect(global.core.summary.write).toHaveBeenCalled();
  });

  it("skips the step summary write when GH_AW_DETECTION_SKIP_PROMPT_SUMMARY is true", async () => {
    setupCoreMocks();
    process.env.GH_AW_DETECTION_SKIP_PROMPT_SUMMARY = "true";

    const module = await import("./setup_threat_detection.cjs");
    await module.main();

    expect(global.core.setFailed).not.toHaveBeenCalled();
    expect(global.core.summary.addRaw).not.toHaveBeenCalled();
    expect(global.core.summary.write).not.toHaveBeenCalled();
    expect(global.core.exportVariable).toHaveBeenCalledWith("GH_AW_PROMPT", "/tmp/gh-aw/aw-prompts/prompt.txt");
  });

  it("fails when patch is missing and continue-on-error is disabled", async () => {
    setupCoreMocks();
    process.env.HAS_PATCH = "true";
    process.env.GH_AW_DETECTION_CONTINUE_ON_ERROR = "false";

    const module = await import("./setup_threat_detection.cjs");
    await module.main();

    expect(global.core.setFailed).toHaveBeenCalledWith(expect.stringContaining("Patch/bundle file(s) expected but not found"));
  });
});
