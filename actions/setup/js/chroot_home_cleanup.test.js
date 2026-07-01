import { afterEach, describe, expect, it } from "vitest";
import fs from "fs";
import os from "os";
import path from "path";
import { spawnSync } from "child_process";
import { fileURLToPath } from "url";

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const POST_SCRIPT_PATH = path.join(__dirname, "..", "post.js");
const CLEAN_SCRIPT_PATH = path.join(__dirname, "..", "clean.sh");
const INSTALL_COPILOT_CLI_SCRIPT_PATH = path.join(__dirname, "..", "sh", "install_copilot_cli.sh");

const tempDirs = [];

function createTempDir(prefix) {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), prefix));
  tempDirs.push(dir);
  return dir;
}

function createFakeSudoEnvironment() {
  const root = createTempDir("chroot-home-cleanup-");
  const fakeBin = path.join(root, "fake-bin");
  fs.mkdirSync(fakeBin, { recursive: true });

  const logPath = path.join(root, "sudo.log");
  const fakeSudoPath = path.join(fakeBin, "sudo");
  fs.writeFileSync(
    fakeSudoPath,
    `#!/usr/bin/env bash
set -euo pipefail
echo "$*" >> "$FAKE_SUDO_LOG"
if [ "$1" = "find" ]; then
  if printf '%s\\n' "$*" | grep -q -- '-print'; then
    printf '%b' "\${FAKE_FIND_PRINT_OUTPUT:-}"
    exit "\${FAKE_FIND_PRINT_STATUS:-0}"
  fi
  if printf '%s\\n' "$*" | grep -q -- '-exec'; then
    exit "\${FAKE_FIND_EXEC_STATUS:-0}"
  fi
fi
exit 0
`,
    { mode: 0o755 }
  );

  return {
    fakeBin,
    logPath,
    root,
  };
}

function runPostScript(env) {
  return spawnSync(process.execPath, [POST_SCRIPT_PATH], {
    encoding: "utf8",
    env: { ...process.env, ...env },
  });
}

function runCleanScript(env) {
  return spawnSync("bash", [CLEAN_SCRIPT_PATH], {
    encoding: "utf8",
    env: { ...process.env, ...env },
  });
}

afterEach(() => {
  while (tempDirs.length > 0) {
    const dir = tempDirs.pop();
    if (dir && fs.existsSync(dir)) {
      fs.rmSync(dir, { recursive: true, force: true });
    }
  }
});

describe("post.js chroot-home cleanup", () => {
  it("logs that no directories were found when find output is empty", () => {
    const { fakeBin, logPath } = createFakeSudoEnvironment();
    const result = runPostScript({
      PATH: `${fakeBin}:${process.env.PATH}`,
      FAKE_SUDO_LOG: logPath,
      FAKE_FIND_PRINT_OUTPUT: "",
    });

    expect(result.status).toBe(0);
    expect(result.stdout).toContain("No /tmp/awf-*-chroot-home directories found");
    expect(fs.readFileSync(logPath, "utf8")).not.toContain("-exec rm -rf -- {} +");
  });

  it("logs count of cleaned chroot-home directories", () => {
    const { fakeBin, logPath } = createFakeSudoEnvironment();
    const result = runPostScript({
      PATH: `${fakeBin}:${process.env.PATH}`,
      FAKE_SUDO_LOG: logPath,
      FAKE_FIND_PRINT_OUTPUT: "/tmp/awf-1-chroot-home\n/tmp/awf-2-chroot-home\n",
    });

    expect(result.status).toBe(0);
    expect(result.stdout).toContain("Cleaned up 2 /tmp/awf-*-chroot-home directories");
    expect(fs.readFileSync(logPath, "utf8")).toContain("-exec rm -rf -- {} +");
  });
});

describe("clean.sh chroot-home cleanup", () => {
  it("logs when no chroot-home directories are found", () => {
    const { fakeBin, logPath, root } = createFakeSudoEnvironment();
    const destination = path.join(root, "destination");
    fs.mkdirSync(destination, { recursive: true });

    const result = runCleanScript({
      PATH: `${fakeBin}:${process.env.PATH}`,
      FAKE_SUDO_LOG: logPath,
      FAKE_FIND_PRINT_OUTPUT: "",
      INPUT_DESTINATION: destination,
    });

    expect(result.status).toBe(0);
    expect(result.stdout).toContain("No /tmp/awf-*-chroot-home directories found");
    expect(fs.readFileSync(logPath, "utf8")).not.toContain("-exec rm -rf -- {} +");
  });

  it("logs successful cleanup when chroot-home directories are found", () => {
    const { fakeBin, logPath, root } = createFakeSudoEnvironment();
    const destination = path.join(root, "destination");
    fs.mkdirSync(destination, { recursive: true });

    const result = runCleanScript({
      PATH: `${fakeBin}:${process.env.PATH}`,
      FAKE_SUDO_LOG: logPath,
      FAKE_FIND_PRINT_OUTPUT: "/tmp/awf-1-chroot-home\n",
      INPUT_DESTINATION: destination,
    });

    expect(result.status).toBe(0);
    expect(result.stdout).toContain("Cleaned up /tmp/awf-*-chroot-home directories (sudo)");
    expect(fs.readFileSync(logPath, "utf8")).toContain("-exec rm -rf -- {} +");
  });
});

describe("install_copilot_cli.sh chroot-home cleanup", () => {
  it("cleans stale chroot-home directories before starting Copilot CLI installation", () => {
    const script = fs.readFileSync(INSTALL_COPILOT_CLI_SCRIPT_PATH, "utf8");

    const ownershipFixIndex = script.indexOf('sudo chown -R "$(id -u):$(id -g)" "$COPILOT_DIR"');
    const cleanupBannerIndex = script.indexOf('echo "Cleaning up stale AWF chroot home directories..."');
    const cleanupCommandIndex = script.indexOf(
      "sudo find /tmp -maxdepth 1 -name 'awf-*-chroot-home' -type d -exec rm -rf -- {} + 2>/dev/null || true"
    );

    expect(ownershipFixIndex).toBeGreaterThanOrEqual(0);
    expect(cleanupBannerIndex).toBeGreaterThan(ownershipFixIndex);
    expect(cleanupCommandIndex).toBeGreaterThan(cleanupBannerIndex);
  });
});
