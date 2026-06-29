import { execFile } from "node:child_process";
import { constants as fsConstants } from "node:fs";
import { access } from "node:fs/promises";
import { join } from "node:path";

const INSTALL_COMMAND = "gh extension install github/gh-aw";

function combineOutput(stdout, stderr) {
  return [stdout, stderr].filter(Boolean).join("\n").trim();
}

function execp(bin, args, cwd, { combineIO = false, execFileFn = execFile, env = process.env } = {}) {
  return new Promise((resolve, reject) => {
    execFileFn(
      bin,
      args,
      {
        cwd,
        env: { ...env, CI: "1", NO_COLOR: "1", GH_NO_UPDATE_NOTIFIER: "1" },
        maxBuffer: 10 * 1024 * 1024,
      },
      (err, stdout, stderr) => {
        const output = combineOutput(stdout ?? "", stderr ?? "");
        if (err) reject(Object.assign(err, { stderr: stderr ?? "", stdout: stdout ?? "", output }));
        else resolve(combineIO ? output : stdout);
      }
    );
  });
}

function parseVersion(output) {
  const trimmed = String(output ?? "").trim();
  if (!trimmed) return "";
  const match = trimmed.match(/gh(?:-aw| aw) version (\S+)/i);
  if (match?.[1]) return match[1];
  return trimmed.split(/\s+/).at(-1) ?? "";
}

function isMissingGhAwExtension(error) {
  const output = String(error?.output ?? error?.stderr ?? error?.message ?? "");
  return /extension not found:\s*aw/i.test(output) || /unknown command ["']aw["'] for ["']gh["']/i.test(output);
}

export function createGhAwRunner({ getWorkspacePath, accessFn = access, execFileFn = execFile, platform = process.platform, env = process.env }) {
  async function findDevBinary(cwd) {
    const devBin = join(cwd, platform === "win32" ? "gh-aw.exe" : "gh-aw");
    try {
      await accessFn(devBin, fsConstants.X_OK);
      return devBin;
    } catch {
      return null;
    }
  }

  async function runExec(bin, args, cwd, options) {
    return execp(bin, args, cwd, { ...options, execFileFn, env });
  }

  return async function runGhAw(args) {
    const cwd = getWorkspacePath();
    const devBin = await findDevBinary(cwd);
    if (devBin) {
      return runExec(devBin, args, cwd);
    }

    return runExec("gh", ["aw", ...args], cwd);
  };
}

export function createGhAwRunnerWithStatus(options) {
  const runGhAw = createGhAwRunner(options);
  const getStatus = async () => {
    const cwd = options.getWorkspacePath();
    const devBin = await (async () => {
      const candidate = join(cwd, options.platform === "win32" ? "gh-aw.exe" : "gh-aw");
      try {
        await (options.accessFn ?? access)(candidate, fsConstants.X_OK);
        return candidate;
      } catch {
        return null;
      }
    })();

    if (devBin) {
      const output = await execp(devBin, ["version"], cwd, {
        combineIO: true,
        execFileFn: options.execFileFn ?? execFile,
        env: options.env ?? process.env,
      });
      return {
        available: true,
        source: "dev-binary",
        version: parseVersion(output),
        command: options.platform === "win32" ? "gh-aw.exe version" : "./gh-aw version",
        installCommand: INSTALL_COMMAND,
      };
    }

    try {
      const output = await execp("gh", ["aw", "version"], cwd, {
        combineIO: true,
        execFileFn: options.execFileFn ?? execFile,
        env: options.env ?? process.env,
      });
      return {
        available: true,
        source: "gh-extension",
        version: parseVersion(output),
        command: "gh aw version",
        installCommand: INSTALL_COMMAND,
      };
    } catch (error) {
      if (isMissingGhAwExtension(error)) {
        return {
          available: false,
          source: "missing",
          version: "",
          command: "gh aw version",
          installCommand: INSTALL_COMMAND,
          message: "gh aw is not installed. Install the GitHub CLI extension to use the dashboard outside a local dev build.",
        };
      }

      return {
        available: false,
        source: "error",
        version: "",
        command: "gh aw version",
        installCommand: INSTALL_COMMAND,
        message: String(error?.output ?? error?.stderr ?? error?.message ?? "Failed to detect gh aw."),
      };
    }
  };

  runGhAw.getStatus = getStatus;
  return runGhAw;
}
