import crypto from "crypto";
import fs from "fs";
import os from "os";
import path from "path";
import { afterEach, beforeEach, describe, expect, it } from "vitest";

let helpers;
let runnerTemp;

describe("body_file_helpers", () => {
  beforeEach(async () => {
    runnerTemp = fs.mkdtempSync(path.join(os.tmpdir(), "gh-aw-body-file-"));
    process.env.RUNNER_TEMP = runnerTemp;
    helpers = await import("./body_file_helpers.cjs");
    fs.mkdirSync(helpers.getSafeBodyFileRoot(), { recursive: true });
  });

  afterEach(() => {
    fs.rmSync(runnerTemp, { recursive: true, force: true });
    delete process.env.RUNNER_TEMP;
  });

  it("reads an allowlisted UTF-8 file once and returns audit metadata", () => {
    const filePath = path.join(helpers.getSafeBodyFileRoot(), "body.md");
    fs.writeFileSync(filePath, "hello body\n");
    const digest = crypto.createHash("sha256").update(fs.readFileSync(filePath)).digest("hex");

    const result = helpers.readBodyFileSnapshot("gh-aw-safe/body.md", digest);

    expect(result).toEqual({
      content: "hello body\n",
      metadata: {
        path: "gh-aw-safe/body.md",
        sha256: digest,
        bytes: 11,
      },
    });
  });

  it("rejects paths outside the allowlisted directory", () => {
    const outsidePath = path.join(runnerTemp, "outside.md");
    fs.writeFileSync(outsidePath, "nope");
    const digest = crypto.createHash("sha256").update(fs.readFileSync(outsidePath)).digest("hex");

    expect(() => helpers.readBodyFileSnapshot(outsidePath, digest)).toThrow(/path must stay under gh-aw-safe\//i);
  });

  it("rejects SHA-256 mismatches", () => {
    const filePath = path.join(helpers.getSafeBodyFileRoot(), "body.md");
    fs.writeFileSync(filePath, "hello");

    expect(() => helpers.readBodyFileSnapshot("gh-aw-safe/body.md", "a".repeat(64))).toThrow(/body_sha256 mismatch/i);
  });

  it("rejects binary files", () => {
    const filePath = path.join(helpers.getSafeBodyFileRoot(), "body.bin");
    fs.writeFileSync(filePath, Buffer.from([0x68, 0x69, 0x00, 0xff]));
    const digest = crypto.createHash("sha256").update(fs.readFileSync(filePath)).digest("hex");

    expect(() => helpers.readBodyFileSnapshot("gh-aw-safe/body.bin", digest)).toThrow(/UTF-8 text/i);
  });

  it("rejects oversized files", () => {
    const filePath = path.join(helpers.getSafeBodyFileRoot(), "body.md");
    fs.writeFileSync(filePath, "a".repeat(helpers.MAX_BODY_FILE_BYTES + 1));
    const digest = crypto.createHash("sha256").update(fs.readFileSync(filePath)).digest("hex");

    expect(() => helpers.readBodyFileSnapshot("gh-aw-safe/body.md", digest)).toThrow(/file is too large/i);
  });

  it("rejects symlink escapes", () => {
    const targetPath = path.join(runnerTemp, "outside.md");
    fs.writeFileSync(targetPath, "secret");
    fs.symlinkSync(targetPath, path.join(helpers.getSafeBodyFileRoot(), "body.md"));
    const digest = crypto.createHash("sha256").update(fs.readFileSync(targetPath)).digest("hex");

    expect(() => helpers.readBodyFileSnapshot("gh-aw-safe/body.md", digest)).toThrow(/symbolic links are not allowed/i);
  });
});
