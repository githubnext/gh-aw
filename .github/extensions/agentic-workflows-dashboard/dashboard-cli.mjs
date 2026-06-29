import { spawn } from "node:child_process";
import { constants as fsConstants } from "node:fs";
import { access } from "node:fs/promises";
import { join } from "node:path";

const INSTALL_COMMAND = "gh extension install github/gh-aw";

function combineOutput(stdout, stderr) {
  return [stdout, stderr].filter(Boolean).join("\n").trim();
}

/**
 * Wraps spawn() with the same callback signature as execFile(), but uses
 * stdio: ['ignore', 'pipe', 'pipe'] so the child process never blocks waiting
 * for stdin. This is important in environments where the parent process holds
 * a special stdin handle (e.g. Copilot CLI) that causes the child to hang.
 */
function spawnExecFile(file, args, options, callback) {
  const { env, cwd, maxBuffer = 10 * 1024 * 1024 } = options ?? {};
  // detached: true prevents the child from inheriting the parent's special
  // handles (e.g. Copilot CLI named pipes) that would otherwise cause gh-aw
  // to block indefinitely waiting on an inherited pipe it never owns.
  const proc = spawn(file, args, { env, cwd, stdio: ["ignore", "pipe", "pipe"], detached: true });
  const stdoutChunks = [];
  const stderrChunks = [];
  let stdoutLen = 0;
  let stderrLen = 0;
  let overflowed = false;

  proc.stdout.on("data", chunk => {
    stdoutLen += chunk.length;
    if (stdoutLen > maxBuffer) { overflowed = true; return; }
    stdoutChunks.push(chunk);
  });
  proc.stderr.on("data", chunk => {
    stderrLen += chunk.length;
    if (stderrLen > maxBuffer) { overflowed = true; return; }
    stderrChunks.push(chunk);
  });

  proc.on("error", err => callback(err, "", ""));
  proc.on("close", code => {
    const stdout = Buffer.concat(stdoutChunks).toString("utf8");
    const stderr = Buffer.concat(stderrChunks).toString("utf8");
    if (overflowed) {
      const err = new Error("stdout/stderr maxBuffer exceeded");
      err.code = "ERR_CHILD_PROCESS_STDIO_MAXBUFFER";
      callback(err, stdout, stderr);
    } else if (code !== 0) {
      const err = new Error(`Command failed with exit code ${code}`);
      err.code = code;
      callback(err, stdout, stderr);
    } else {
      callback(null, stdout, stderr);
    }
  });
}

function execp(bin, args, cwd, { combineIO = false, execFileFn = spawnExecFile, env = process.env } = {}) {
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

function parseVersionFromOutput(output) {
  const trimmed = String(output ?? "").trim();
  if (!trimmed) return "";
  const match = trimmed.match(/gh(?:-aw| aw) version ([^\r\n]+)/i);
  return match?.[1]?.trim() ?? "";
}

function isMissingGhAwExtension(error) {
  const output = String(error?.output ?? error?.stderr ?? error?.message ?? "");
  return /extension not found:\s*aw/i.test(output) || /unknown command ["']aw["'] for ["']gh["']/i.test(output);
}

async function findDevBinary(cwd, accessFn = access, platform = process.platform) {
  const devBin = join(cwd, platform === "win32" ? "gh-aw.exe" : "gh-aw");
  try {
    await accessFn(devBin, fsConstants.X_OK);
    return devBin;
  } catch {
    return null;
  }
}

export function createGhAwRunner({ getWorkspacePath, accessFn = access, execFileFn = spawnExecFile, platform = process.platform, env = process.env }) {
  async function runExec(bin, args, cwd, options) {
    return execp(bin, args, cwd, { ...options, execFileFn, env });
  }

  return async function runGhAw(args) {
    const cwd = getWorkspacePath();
    const devBin = await findDevBinary(cwd, accessFn, platform);
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
    const devBin = await findDevBinary(cwd, options.accessFn ?? access, options.platform ?? process.platform);

    if (devBin) {
      const output = await execp(devBin, ["version"], cwd, {
        combineIO: true,
        execFileFn: options.execFileFn ?? spawnExecFile,
        env: options.env ?? process.env,
      });
      return {
        available: true,
        source: "dev-binary",
        version: parseVersionFromOutput(output) || "unknown",
        command: `${devBin} version`,
        installCommand: INSTALL_COMMAND,
      };
    }

    try {
      const output = await execp("gh", ["aw", "version"], cwd, {
        combineIO: true,
        execFileFn: options.execFileFn ?? spawnExecFile,
        env: options.env ?? process.env,
      });
      return {
        available: true,
        source: "gh-extension",
        version: parseVersionFromOutput(output) || "unknown",
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
