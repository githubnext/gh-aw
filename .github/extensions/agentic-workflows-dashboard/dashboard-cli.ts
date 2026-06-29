import { spawn } from "node:child_process";
import { constants as fsConstants } from "node:fs";
import { access } from "node:fs/promises";
import { join } from "node:path";

import type { CLIStatus } from "./src/models.js";

const INSTALL_COMMAND = "gh extension install github/gh-aw";
const GH_INSTALL_URL = "https://cli.github.com";

export interface CommandError extends Error {
  code?: number | string;
  output?: string;
  path?: string;
  stderr?: string;
  stdout?: string;
  syscall?: string;
}

interface SpawnExecFileOptions {
  cwd?: string;
  env?: NodeJS.ProcessEnv;
  maxBuffer?: number;
}

type ExecFileCallback = (error: CommandError | null, stdout: string, stderr: string) => void;
export type ExecFileFunction = (file: string, args: string[], options: SpawnExecFileOptions, callback: ExecFileCallback) => void;

export interface GhAwRunnerOptions {
  accessFn?: typeof access;
  env?: NodeJS.ProcessEnv;
  execFileFn?: ExecFileFunction;
  getWorkspacePath: () => string;
  platform?: NodeJS.Platform;
}

export interface GhAwRunner {
  (args: string[]): Promise<string>;
}

export interface GhAwRunnerWithStatus extends GhAwRunner {
  getStatus: () => Promise<CLIStatus>;
}

function combineOutput(stdout: string, stderr: string): string {
  return [stdout, stderr].filter(Boolean).join("\n").trim();
}

function buildCommandError(message: string, details: Partial<CommandError> = {}): CommandError {
  return Object.assign(new Error(message), details);
}

/**
 * Wraps spawn() with the same callback signature as execFile(), but uses
 * stdio: ['ignore', 'pipe', 'pipe'] so the child process never blocks waiting
 * for stdin. This is important in environments where the parent process holds
 * a special stdin handle (e.g. Copilot CLI) that causes the child to hang.
 */
function spawnExecFile(file: string, args: string[], options: SpawnExecFileOptions = {}, callback: ExecFileCallback): void {
  const { env, cwd, maxBuffer = 10 * 1024 * 1024 } = options;
  // detached: true prevents the child from inheriting the parent's special
  // handles (e.g. Copilot CLI named pipes) that would otherwise cause gh-aw
  // to block indefinitely waiting on an inherited pipe it never owns.
  const proc = spawn(file, args, { env, cwd, stdio: ["ignore", "pipe", "pipe"], detached: true });
  const stdoutChunks: Buffer[] = [];
  const stderrChunks: Buffer[] = [];
  let stdoutLen = 0;
  let stderrLen = 0;
  let overflowed = false;

  proc.stdout?.on("data", chunk => {
    const buffer = Buffer.isBuffer(chunk) ? chunk : Buffer.from(String(chunk));
    stdoutLen += buffer.length;
    if (stdoutLen > maxBuffer) {
      overflowed = true;
      return;
    }
    stdoutChunks.push(buffer);
  });

  proc.stderr?.on("data", chunk => {
    const buffer = Buffer.isBuffer(chunk) ? chunk : Buffer.from(String(chunk));
    stderrLen += buffer.length;
    if (stderrLen > maxBuffer) {
      overflowed = true;
      return;
    }
    stderrChunks.push(buffer);
  });

  proc.on("error", error => callback(error as CommandError, "", ""));
  proc.on("close", code => {
    const stdout = Buffer.concat(stdoutChunks).toString("utf8");
    const stderr = Buffer.concat(stderrChunks).toString("utf8");
    if (overflowed) {
      callback(buildCommandError("stdout/stderr maxBuffer exceeded", { code: "ERR_CHILD_PROCESS_STDIO_MAXBUFFER" }), stdout, stderr);
      return;
    }

    if (code !== 0) {
      callback(buildCommandError(`Command failed with exit code ${code ?? "unknown"}`, { code: code ?? "unknown" }), stdout, stderr);
      return;
    }

    callback(null, stdout, stderr);
  });
}

function execp(bin: string, args: string[], cwd: string, { combineIO = false, execFileFn = spawnExecFile, env = process.env }: { combineIO?: boolean; execFileFn?: ExecFileFunction; env?: NodeJS.ProcessEnv } = {}): Promise<string> {
  return new Promise((resolve, reject) => {
    execFileFn(
      bin,
      args,
      {
        cwd,
        env: { ...env, CI: "1", NO_COLOR: "1", GH_NO_UPDATE_NOTIFIER: "1" },
        maxBuffer: 10 * 1024 * 1024,
      },
      (error, stdout, stderr) => {
        const output = combineOutput(stdout, stderr);
        if (error) {
          reject(Object.assign(error, { stderr, stdout, output }));
          return;
        }
        resolve(combineIO ? output : stdout);
      }
    );
  });
}

function parseVersionFromOutput(output: string): string {
  const trimmed = String(output).trim();
  if (!trimmed) return "";
  const match = trimmed.match(/gh(?:-aw| aw) version ([^\r\n]+)/i);
  return match?.[1]?.trim() ?? "";
}

function isMissingGh(error: unknown): error is CommandError {
  return typeof error === "object" && error !== null && (error as CommandError).code === "ENOENT" && (error as CommandError).syscall === "spawn" && (error as CommandError).path === "gh";
}

function isMissingGhAwExtension(error: unknown): boolean {
  const output = typeof error === "object" && error !== null ? String((error as CommandError).output ?? (error as CommandError).stderr ?? (error as CommandError).message ?? "") : "";
  return /extension not found:\s*aw/i.test(output) || /unknown command ["']aw["'] for ["']gh["']/i.test(output);
}

async function findDevBinary(cwd: string, accessFn: typeof access = access, platform: NodeJS.Platform = process.platform): Promise<string | null> {
  const devBin = join(cwd, platform === "win32" ? "gh-aw.exe" : "gh-aw");
  try {
    await accessFn(devBin, fsConstants.X_OK);
    return devBin;
  } catch {
    return null;
  }
}

export function createGhAwRunner({ getWorkspacePath, accessFn = access, execFileFn = spawnExecFile, platform = process.platform, env = process.env }: GhAwRunnerOptions): GhAwRunner {
  async function runExec(bin: string, args: string[], cwd: string, options?: { combineIO?: boolean }): Promise<string> {
    return execp(bin, args, cwd, { ...options, execFileFn, env });
  }

  return async function runGhAw(args: string[]): Promise<string> {
    const cwd = getWorkspacePath();
    const devBin = await findDevBinary(cwd, accessFn, platform);
    if (devBin) {
      return runExec(devBin, args, cwd);
    }

    return runExec("gh", ["aw", ...args], cwd);
  };
}

export function createGhAwRunnerWithStatus(options: GhAwRunnerOptions): GhAwRunnerWithStatus {
  const runGhAw = createGhAwRunner(options) as GhAwRunnerWithStatus;

  runGhAw.getStatus = async (): Promise<CLIStatus> => {
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
      if (isMissingGh(error)) {
        return {
          available: false,
          source: "gh-not-found",
          version: "",
          command: "gh aw version",
          installCommand: INSTALL_COMMAND,
          installUrl: GH_INSTALL_URL,
          message: "Install the GitHub CLI to use this dashboard.",
        };
      }

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

      const output = typeof error === "object" && error !== null ? String((error as CommandError).output ?? (error as CommandError).stderr ?? (error as CommandError).message ?? "Failed to detect gh aw.") : "Failed to detect gh aw.";
      return {
        available: false,
        source: "error",
        version: "",
        command: "gh aw version",
        installCommand: INSTALL_COMMAND,
        message: output,
      };
    }
  };

  return runGhAw;
}
