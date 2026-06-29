import { spawn, type SpawnOptions } from "node:child_process";
import { constants as fsConstants } from "node:fs";
import { access } from "node:fs/promises";
import { join } from "node:path";

const INSTALL_COMMAND = "gh extension install github/gh-aw";
const GH_INSTALL_URL = "https://cli.github.com";

type ExecError = Error & {
  code?: string | number;
  syscall?: string;
  path?: string;
  stderr?: string;
  stdout?: string;
  output?: string;
};

type ExecCallback = (err: ExecError | null, stdout: string, stderr: string) => void;

type ExecFileLike = (file: string, args: string[], options: ExecOptions, callback: ExecCallback) => void;

type AccessLike = typeof access;

interface ExecOptions {
  env?: NodeJS.ProcessEnv;
  cwd?: string;
  maxBuffer?: number;
}

interface RunExecOptions {
  combineIO?: boolean;
  execFileFn?: ExecFileLike;
  env?: NodeJS.ProcessEnv;
}

interface RunnerOptions {
  getWorkspacePath: () => string;
  accessFn?: AccessLike;
  execFileFn?: ExecFileLike;
  platform?: NodeJS.Platform;
  env?: NodeJS.ProcessEnv;
}

export interface GhAwStatus {
  available: boolean;
  source: "dev-binary" | "gh-extension" | "gh-not-found" | "missing" | "error";
  version: string;
  command: string;
  installCommand: string;
  installUrl?: string;
  message?: string;
}

export type GhAwRunner = ((args: string[]) => Promise<string>) & {
  getStatus: () => Promise<GhAwStatus>;
};

function combineOutput(stdout: string, stderr: string): string {
  return [stdout, stderr].filter(Boolean).join("\n").trim();
}

function spawnExecFile(file: string, args: string[], options: ExecOptions, callback: ExecCallback): void {
  const { env, cwd, maxBuffer = 10 * 1024 * 1024 } = options ?? {};
  const spawnOptions: SpawnOptions = { env, cwd, stdio: ["ignore", "pipe", "pipe"], detached: true };
  const proc = spawn(file, args, spawnOptions);
  const stdoutChunks: Buffer[] = [];
  const stderrChunks: Buffer[] = [];
  let stdoutLen = 0;
  let stderrLen = 0;
  let overflowed = false;

  proc.stdout?.on("data", (chunk: Buffer) => {
    stdoutLen += chunk.length;
    if (stdoutLen > maxBuffer) {
      overflowed = true;
      return;
    }
    stdoutChunks.push(chunk);
  });

  proc.stderr?.on("data", (chunk: Buffer) => {
    stderrLen += chunk.length;
    if (stderrLen > maxBuffer) {
      overflowed = true;
      return;
    }
    stderrChunks.push(chunk);
  });

  proc.on("error", err => callback(err as ExecError, "", ""));
  proc.on("close", code => {
    const stdout = Buffer.concat(stdoutChunks).toString("utf8");
    const stderr = Buffer.concat(stderrChunks).toString("utf8");
    if (overflowed) {
      const err: ExecError = new Error("stdout/stderr maxBuffer exceeded");
      err.code = "ERR_CHILD_PROCESS_STDIO_MAXBUFFER";
      callback(err, stdout, stderr);
    } else if (code !== 0) {
      const err: ExecError = new Error(`Command failed with exit code ${code}`);
      err.code = code ?? 1;
      callback(err, stdout, stderr);
    } else {
      callback(null, stdout, stderr);
    }
  });
}

function execp(bin: string, args: string[], cwd: string, { combineIO = false, execFileFn = spawnExecFile, env = process.env }: RunExecOptions = {}): Promise<string> {
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
        if (err) {
          reject(Object.assign(err, { stderr: stderr ?? "", stdout: stdout ?? "", output }));
          return;
        }
        resolve(combineIO ? output : stdout);
      }
    );
  });
}

function parseVersionFromOutput(output: string): string {
  const trimmed = String(output ?? "").trim();
  if (!trimmed) return "";
  const match = trimmed.match(/gh(?:-aw| aw) version ([^\r\n]+)/i);
  return match?.[1]?.trim() ?? "";
}

function isMissingGh(error: unknown): boolean {
  const e = error as ExecError | undefined;
  return e?.code === "ENOENT" && e?.syscall === "spawn" && e?.path === "gh";
}

function isMissingGhAwExtension(error: unknown): boolean {
  const e = error as ExecError | undefined;
  const output = String(e?.output ?? e?.stderr ?? e?.message ?? "");
  return /extension not found:\s*aw/i.test(output) || /unknown command ["']aw["'] for ["']gh["']/i.test(output);
}

async function findDevBinary(cwd: string, accessFn: AccessLike = access, platform: NodeJS.Platform = process.platform): Promise<string | null> {
  const devBin = join(cwd, platform === "win32" ? "gh-aw.exe" : "gh-aw");
  try {
    await accessFn(devBin, fsConstants.X_OK);
    return devBin;
  } catch {
    return null;
  }
}

export function createGhAwRunner({ getWorkspacePath, accessFn = access, execFileFn = spawnExecFile, platform = process.platform, env = process.env }: RunnerOptions): (args: string[]) => Promise<string> {
  async function runExec(bin: string, args: string[], cwd: string, options?: RunExecOptions): Promise<string> {
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

export function createGhAwRunnerWithStatus(options: RunnerOptions): GhAwRunner {
  const runGhAw = createGhAwRunner(options) as GhAwRunner;
  const getStatus = async (): Promise<GhAwStatus> => {
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

      const e = error as ExecError | undefined;
      return {
        available: false,
        source: "error",
        version: "",
        command: "gh aw version",
        installCommand: INSTALL_COMMAND,
        message: String(e?.output ?? e?.stderr ?? e?.message ?? "Failed to detect gh aw."),
      };
    }
  };

  runGhAw.getStatus = getStatus;
  return runGhAw;
}
