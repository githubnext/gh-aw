import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import fs from "fs";
import os from "os";
import path from "path";
import { main, parseConfig, renderFiles, resolveRelativePath } from "./create_files.cjs";

const core = { info: vi.fn(), setFailed: vi.fn() };

describe("create_files", () => {
  let tempDir;
  let originalEnv;

  beforeEach(() => {
    tempDir = fs.mkdtempSync(path.join(os.tmpdir(), "gh-aw-create-files-"));
    originalEnv = { ...process.env };
    core.info.mockReset();
    core.setFailed.mockReset();
  });

  afterEach(() => {
    process.env = originalEnv;
    fs.rmSync(tempDir, { recursive: true, force: true });
  });

  it("writes environment content byte for byte without invoking a shell", async () => {
    const root = path.join(tempDir, "gh-aw");
    const canaryPath = path.join(tempDir, "must-not-exist");
    const content = `$(touch ${canaryPath})\n\`false\`\n\${PATH}\nUnicode: 雪man ☃️ café\n`;
    process.env = {
      ...originalEnv,
      RUNNER_TEMP: tempDir,
      GH_AW_FILE_ROOT: root,
      GH_AW_FILE_CONFIG: JSON.stringify({
        directories: ["safeoutputs/logs"],
        files: [{ path: "safeoutputs/config.json", content_env: "CONFIG_CONTENT" }],
      }),
      CONFIG_CONTENT: content,
    };

    await main(core);

    expect(core.setFailed).not.toHaveBeenCalled();
    expect(fs.readFileSync(path.join(root, "safeoutputs", "config.json"), "utf8")).toBe(content);
    expect(fs.existsSync(path.join(root, "safeoutputs", "logs"))).toBe(true);
    expect(fs.existsSync(canaryPath)).toBe(false);
  });

  it("restricts permissions when replacing an existing file", () => {
    if (process.platform === "win32") return;

    const root = path.join(tempDir, "gh-aw");
    const filePath = path.join(root, "config.json");
    fs.mkdirSync(root);
    fs.writeFileSync(filePath, "old", { mode: 0o666 });
    fs.chmodSync(filePath, 0o666);

    renderFiles({ files: [{ path: "config.json", content_env: "CONTENT" }] }, { CONTENT: "new" }, root);

    expect(fs.statSync(filePath).mode & 0o777).toBe(0o600);
    expect(fs.readFileSync(filePath, "utf8")).toBe("new");
  });

  it("rejects malformed configuration, traversal, and missing content", () => {
    expect(() => parseConfig("{}")).toThrow("files array");
    expect(() => parseConfig('{"files":[],"directories":"tmp"}')).toThrow("directories must be an array");
    expect(() => resolveRelativePath(tempDir, "/etc/passwd")).toThrow("must be relative");
    expect(() => resolveRelativePath(tempDir, "../secret")).toThrow("configured root");
    expect(() => renderFiles({ files: [{ path: "file", content_env: "MISSING" }] }, {}, tempDir)).toThrow("environment variable is missing");
  });

  it("rejects a configured root outside runner temp", async () => {
    process.env = {
      ...originalEnv,
      RUNNER_TEMP: path.join(tempDir, "runner"),
      GH_AW_FILE_ROOT: tempDir,
      GH_AW_FILE_CONFIG: '{"files":[]}',
    };

    await main(core);

    expect(core.setFailed).toHaveBeenCalledWith(expect.stringContaining("configured root"));
  });
});
