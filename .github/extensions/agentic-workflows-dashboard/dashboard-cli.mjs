import { execFile } from "node:child_process";
import { constants as fsConstants } from "node:fs";
import { access } from "node:fs/promises";
import { join } from "node:path";

function execp(bin, args, cwd) {
  return new Promise((resolve, reject) => {
    execFile(
      bin,
      args,
      {
        cwd,
        env: { ...process.env, NO_COLOR: "1", GH_NO_UPDATE_NOTIFIER: "1" },
        maxBuffer: 10 * 1024 * 1024,
      },
      (err, stdout, stderr) => {
        if (err) reject(Object.assign(err, { stderr: stderr ?? "" }));
        else resolve(stdout);
      }
    );
  });
}

export function createGhAwRunner({ getWorkspacePath }) {
  return async function runGhAw(args) {
    const cwd = getWorkspacePath();
    const isWin = process.platform === "win32";
    const devBin = join(cwd, isWin ? "gh-aw.exe" : "gh-aw");
    try {
      await access(devBin, fsConstants.X_OK);
      return await execp(devBin, args, cwd);
    } catch {
      return await execp("gh", ["aw", ...args], cwd);
    }
  };
}
