import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import fs from "fs";
import os from "os";
import path from "path";

const core = { info: vi.fn(), setFailed: vi.fn() };
global.core = core;

const { main, parseConfig, renderPrompt, resolvePromptFile } = require("./create_prompt.cjs");

describe("create_prompt", () => {
  let tempDir;
  let originalEnv;

  beforeEach(() => {
    tempDir = fs.mkdtempSync(path.join(os.tmpdir(), "gh-aw-create-prompt-"));
    originalEnv = { ...process.env };
    core.info.mockReset();
    core.setFailed.mockReset();
  });

  afterEach(() => {
    process.env = originalEnv;
    fs.rmSync(tempDir, { recursive: true, force: true });
  });

  it("renders shell and JavaScript injection payloads byte for byte as data", () => {
    const payloads = [
      "$(touch /tmp/gh-aw-command-substitution)",
      "`touch /tmp/gh-aw-backtick`",
      "${PATH}; rm -rf /",
      "${{ github.event.issue.title }}",
      '"; process.exit(1); //',
      "'; require('child_process').execSync('false'); //",
      "line one\nGH_AW_PROMPT_deadbeefdeadbeef_EOF\ntouch /tmp/gh-aw-heredoc\nline four",
      "pipe | ampersand & semicolon ; redirect > file",
      "carriage\r\nreturn\r\ncontent",
      "Unicode: 雪man ☃️ café",
    ];
    const env = {};
    const items = payloads.map((payload, index) => {
      const key = `GH_AW_PROMPT_CONTENT_${index}`;
      env[key] = payload;
      return { content_env: key };
    });

    expect(renderPrompt({ items }, env, tempDir)).toBe(payloads.join(""));
  });

  it("preserves ordering across inline content and prompt files", () => {
    fs.writeFileSync(path.join(tempDir, "system.md"), "file content\n", "utf8");
    const config = {
      items: [{ content_env: "FIRST" }, { file: "system.md" }, { content_env: "LAST" }],
    };
    const env = { FIRST: "<system>\n", LAST: "</system>\nuser prompt\n" };

    expect(renderPrompt(config, env, tempDir)).toBe("<system>\nfile content\n</system>\nuser prompt\n");
  });

  it("only includes conditional content for the exact true value", () => {
    const config = {
      items: [{ content_env: "ALWAYS" }, { content_env: "CONDITIONAL", condition_env: "INCLUDE" }],
    };

    expect(renderPrompt(config, { ALWAYS: "a", CONDITIONAL: "b", INCLUDE: "true" }, tempDir)).toBe("ab");
    for (const value of ["false", "TRUE", "1", "", "true; throw new Error()"]) {
      expect(renderPrompt(config, { ALWAYS: "a", CONDITIONAL: "b", INCLUDE: value }, tempDir)).toBe("a");
    }
  });

  it("rejects malformed configuration and missing content", () => {
    expect(() => parseConfig("{}")).toThrow("items array");
    expect(() => renderPrompt({ items: [null] }, {}, tempDir)).toThrow("must be an object");
    expect(() => renderPrompt({ items: [{}] }, {}, tempDir)).toThrow("exactly one");
    expect(() => renderPrompt({ items: [{ content_env: "TEXT", file: "system.md" }] }, { TEXT: "x" }, tempDir)).toThrow("exactly one");
    expect(() => renderPrompt({ items: [{ content_env: "MISSING" }] }, {}, tempDir)).toThrow("environment variable is missing");
  });

  it("rejects absolute paths and traversal outside the prompts directory", () => {
    expect(() => resolvePromptFile(tempDir, "/etc/passwd")).toThrow("relative path");
    expect(() => resolvePromptFile(tempDir, "../secret")).toThrow("within the prompt directory");
    expect(resolvePromptFile(tempDir, "nested/prompt.md")).toBe(path.join(tempDir, "nested", "prompt.md"));
  });

  it("writes the rendered prompt without invoking a shell", async () => {
    const promptPath = path.join(tempDir, "output", "prompt.txt");
    const canaryPath = path.join(tempDir, "must-not-exist");
    process.env = {
      ...originalEnv,
      RUNNER_TEMP: tempDir,
      GH_AW_PROMPT: promptPath,
      GH_AW_PROMPT_CONFIG: JSON.stringify({ items: [{ content_env: "PAYLOAD" }] }),
      PAYLOAD: `$(touch ${canaryPath})\n\`false\`\n`,
    };

    await main();

    expect(core.setFailed).not.toHaveBeenCalled();
    expect(fs.readFileSync(promptPath, "utf8")).toBe(process.env.PAYLOAD);
    expect(fs.existsSync(canaryPath)).toBe(false);
  });

  it("fails closed when required runtime inputs are absent", async () => {
    process.env = { ...originalEnv };
    delete process.env.GH_AW_PROMPT;
    delete process.env.GH_AW_PROMPT_CONFIG;
    delete process.env.RUNNER_TEMP;

    await main();

    expect(core.setFailed).toHaveBeenCalledWith(expect.stringContaining("GH_AW_PROMPT environment variable is not set"));
  });
});
