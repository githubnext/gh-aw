// @ts-check
/// <reference types="@actions/github-script" />

import { afterEach, beforeEach, describe, expect, it } from "vitest";
const fs = require("fs");
const os = require("os");
const path = require("path");
const { lstatGuard } = require("./symlink_guard.cjs");

describe("lstatGuard", () => {
  let tmpDir;

  beforeEach(() => {
    tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), "symlink-guard-test-"));
  });

  afterEach(() => {
    fs.rmSync(tmpDir, { recursive: true, force: true });
  });

  it("returns Stats for a regular file", () => {
    const filePath = path.join(tmpDir, "file.txt");
    fs.writeFileSync(filePath, "hello");

    const stat = lstatGuard(filePath);

    expect(stat).not.toBeNull();
    expect(stat?.isFile()).toBe(true);
  });

  it("returns Stats for a directory", () => {
    const dirPath = path.join(tmpDir, "subdir");
    fs.mkdirSync(dirPath);

    const stat = lstatGuard(dirPath);

    expect(stat).not.toBeNull();
    expect(stat?.isDirectory()).toBe(true);
  });

  it("returns null for a symbolic link", () => {
    const target = path.join(tmpDir, "target.txt");
    fs.writeFileSync(target, "content");
    const link = path.join(tmpDir, "link.txt");
    fs.symlinkSync(target, link);

    const stat = lstatGuard(link);

    expect(stat).toBeNull();
  });

  it("returns null for a dangling symbolic link", () => {
    const link = path.join(tmpDir, "dangling.txt");
    fs.symlinkSync("/nonexistent/path", link);

    const stat = lstatGuard(link);

    expect(stat).toBeNull();
  });

  it("throws when the path does not exist", () => {
    const missing = path.join(tmpDir, "missing.txt");

    expect(() => lstatGuard(missing)).toThrow();
  });
});
